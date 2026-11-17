package sdk_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/ny4rl4th0t3p/playhold-sdk-go"
)

// These tests talk to a real server, ./testserver on the playhold version this SDK is built for, so they check that both
// sides agree on the wire. Its counter module has these name, schema and messages.

const (
	moduleName = "counter"
	code       = "room-1"
	schema     = 2

	msgAdd   uint32 = 1
	msgClose uint32 = 2
)

// serverBinary is the test server, built once for all the tests.
var serverBinary string

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	dir, err := os.MkdirTemp("", "playhold-testserver-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()
	serverBinary = filepath.Join(dir, "testserver")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // the first build downloads playhold
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", serverBinary, ".")
	build.Dir = "testserver"
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "building the test server:", err)
		return 1
	}
	return m.Run()
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// startServer starts a test server of its own for this test, with its flags, such as -max-players 2, and returns its
// URL. Each test gets a fresh one, so rooms and guests never carry over between tests.
func startServer(t *testing.T, flags ...string) string {
	t.Helper()
	server := exec.CommandContext(t.Context(), serverBinary, flags...)
	server.Stderr = os.Stderr
	stdout, err := server.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, server.Start())
	t.Cleanup(func() {
		_ = server.Process.Kill()
		_ = server.Wait()
	})
	lines := bufio.NewScanner(stdout)
	require.True(t, lines.Scan(), "the test server printed no URL")
	return lines.Text()
}

func options(url string, identity sdk.Identity) *sdk.Options {
	return &sdk.Options{URL: url, Identity: identity, SchemaVersions: map[string]uint32{moduleName: schema}}
}

func connect(t *testing.T, url string, identity sdk.Identity) *sdk.Client {
	t.Helper()
	c, err := sdk.Connect(testContext(t), options(url, identity))
	require.NoError(t, err)
	t.Cleanup(c.Close)
	return c
}

func viewOf(t *testing.T, views []sdk.ModuleView) string {
	t.Helper()
	require.Len(t, views, 1)
	assert.Equal(t, moduleName, views[0].Module)
	text := &wrapperspb.StringValue{}
	require.NoError(t, proto.Unmarshal(views[0].View, text))
	return text.GetValue()
}

// requireCode checks the error's code, and returns the number of the frame it's about.
func requireCode(t *testing.T, err error, want sdk.ErrorCode) (seq uint64) {
	t.Helper()
	var got *sdk.Error
	require.ErrorAs(t, err, &got)
	assert.Equal(t, want, got.Code)
	return got.Seq
}

func TestNewGuestGetsAnIdentityAndCanComeBack(t *testing.T) {
	url := startServer(t)

	first := connect(t, url, sdk.Identity{}).Identity()
	assert.NotEmpty(t, first.PlayerID)
	assert.NotEmpty(t, first.Secret)

	assert.Equal(t, first, connect(t, url, first).Identity(), "the secret isn't sent again, but it's kept")
}

func TestCreateRoomReturnsACodeOthersCanJoin(t *testing.T) {
	url := startServer(t)
	alice := connect(t, url, sdk.Identity{})

	made, err := alice.CreateRoom(testContext(t))

	require.NoError(t, err)
	assert.Len(t, made, 26)
	bob := connect(t, url, sdk.Identity{})
	requireCode(t, bob.Join(testContext(t), made), sdk.ErrorRoomFull) // the room has one seat, and alice has it
}

func TestQueuedPlayersAreMatchedIntoOneRoom(t *testing.T) {
	url := startServer(t, "-max-players", "2", "-matchmaking")
	alice := connect(t, url, sdk.Identity{})
	bob := connect(t, url, sdk.Identity{})

	require.NoError(t, alice.Queue(testContext(t), map[string]string{"mode": "casual"}))
	require.NoError(t, bob.Queue(testContext(t), nil))
	code, err := alice.Matched(testContext(t))
	require.NoError(t, err)
	other, err := bob.Matched(testContext(t))
	require.NoError(t, err)

	assert.Equal(t, code, other)
	require.NoError(t, alice.Join(testContext(t), code))
	require.NoError(t, bob.Join(testContext(t), code))
}

func TestServerRefusesAClient(t *testing.T) {
	url := startServer(t)

	tests := []struct {
		name string
		opts *sdk.Options
		want sdk.ErrorCode
	}{
		{
			name: "wrong secret",
			opts: options(url, sdk.Identity{PlayerID: "nobody", Secret: "wrong"}),
			want: sdk.ErrorUnauthorized,
		},
		{
			name: "too old",
			opts: &sdk.Options{URL: url, SchemaVersions: map[string]uint32{moduleName: schema - 1}},
			want: sdk.ErrorUpdateRequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sdk.Connect(testContext(t), tt.opts)

			requireCode(t, err, tt.want)
		})
	}
}

func TestConnectReportsAServerThatIsNotThere(t *testing.T) {
	_, err := sdk.Connect(testContext(t), options("ws://127.0.0.1:1/", sdk.Identity{}))

	require.Error(t, err)
}

func TestJoinSendAndReceiveViews(t *testing.T) {
	c := connect(t, startServer(t), sdk.Identity{})
	ctx := testContext(t)

	require.NoError(t, c.Join(ctx, code))
	views, err := c.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0", viewOf(t, views))

	_, err = c.Send(ctx, moduleName, msgAdd, wrapperspb.Int32(3))
	require.NoError(t, err)
	views, err = c.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "3", viewOf(t, views))
}

func TestFullRoomRefusesAJoin(t *testing.T) {
	url := startServer(t)
	require.NoError(t, connect(t, url, sdk.Identity{}).Join(testContext(t), code))

	requireCode(t, connect(t, url, sdk.Identity{}).Join(testContext(t), code), sdk.ErrorRoomFull)
}

func TestRefusedMessageIsReportedAndTheConnectionCarriesOn(t *testing.T) {
	c := connect(t, startServer(t), sdk.Identity{})
	ctx := testContext(t)
	require.NoError(t, c.Join(ctx, code))
	_, err := c.Next(ctx)
	require.NoError(t, err)

	seq, err := c.Send(ctx, moduleName, msgAdd, wrapperspb.Int32(-1))
	require.NoError(t, err)

	_, err = c.Next(ctx)
	assert.Equal(t, seq, requireCode(t, err, sdk.ErrorRejected))

	views, err := c.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0", viewOf(t, views))
}

func TestNextReportsAClosedRoom(t *testing.T) {
	c := connect(t, startServer(t), sdk.Identity{})
	ctx := testContext(t)
	require.NoError(t, c.Join(ctx, code))
	_, err := c.Next(ctx)
	require.NoError(t, err)

	_, err = c.Send(ctx, moduleName, msgClose, wrapperspb.Int32(0))
	require.NoError(t, err)

	for err == nil {
		_, err = c.Next(ctx)
	}
	require.ErrorIs(t, err, sdk.ErrRoomClosed)
}
