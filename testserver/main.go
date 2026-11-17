// Command testserver is a small game server on the playhold version this SDK is built for, for the SDK's tests. It
// prints its URL, then serves until killed. It's a module of its own, so the SDK's module never requires playhold.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ny4rl4th0t3p/playhold"
)

const (
	moduleName = "counter"
	schema     = 2 // not 1, so a client below it can be tested

	msgAdd   uint32 = 1
	msgClose uint32 = 2

	matchEvery        = 10 * time.Millisecond
	readHeaderTimeout = 5 * time.Second
)

var totalKey = playhold.NewKey[int]("total")

// counter adds to a total, and its view shows it.
type counter struct{}

func (counter) Name() string { return moduleName }

func (counter) Register(r *playhold.Registry) {
	r.ClientSchema(schema, schema)
	playhold.State(r, totalKey, func() *int { return new(int) })
	playhold.Handle(r, msgAdd, func(rc *playhold.RoomContext, msg *wrapperspb.Int32Value) error {
		if msg.GetValue() < 0 {
			return errors.New("can't add less than nothing")
		}
		*totalKey.Get(rc) += int(msg.GetValue())
		return nil
	})
	playhold.Handle(r, msgClose, func(rc *playhold.RoomContext, _ *wrapperspb.Int32Value) error {
		rc.CloseRoom()
		return nil
	})
	playhold.View(r, func(rc *playhold.RoomContext) *wrapperspb.StringValue {
		return wrapperspb.String(strconv.Itoa(*totalKey.Get(rc)))
	})
}

func main() {
	maxPlayers := flag.Int("max-players", 1, "seats per room")
	matchmaking := flag.Bool("matchmaking", false, "match queued players, every 10 ms, instead of letting clients pick room codes")
	flag.Parse()
	if err := run(*maxPlayers, *matchmaking); err != nil {
		log.Fatal(err)
	}
}

func run(maxPlayers int, matchmaking bool) error {
	typ, err := playhold.NewRoomType(playhold.RoomConfig{Modules: []playhold.Module{counter{}}, MaxPlayers: maxPlayers})
	if err != nil {
		return err
	}
	quiet := slog.New(slog.DiscardHandler)
	// Development: it serves plain HTTP on localhost, for tests.
	cfg := &playhold.ServerConfig{RoomType: typ, MemoryLimit: -1, Logger: quiet, Development: true}
	if matchmaking {
		cfg.Matchmaking = &playhold.Matchmaking{Every: matchEvery}
	} else {
		cfg.AllowClientCodes = true // the tests pick their rooms' codes
	}
	server, err := playhold.NewServer(playhold.NewRuntime(playhold.Config{Logger: quiet}), cfg)
	if err != nil {
		return err
	}
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	fmt.Printf("ws://%s\n", listener.Addr())

	httpServer := &http.Server{Handler: server, ReadHeaderTimeout: readHeaderTimeout}
	return httpServer.Serve(listener)
}
