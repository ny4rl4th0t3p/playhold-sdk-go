# playhold-sdk-go

Connects a Go program to a [playhold](https://github.com/ny4rl4th0t3p/playhold) server, for bots and load tests. It's
minimal on purpose: it doesn't reconnect by itself, because a load test has to see and count what goes wrong.

> **Status: early development,** like playhold itself: the API can change between 0.x versions.

```bash
go get github.com/ny4rl4th0t3p/playhold-sdk-go@v0.4.1
```

```go
import sdk "github.com/ny4rl4th0t3p/playhold-sdk-go"
```

How to use it: [Go SDK](https://ny4rl4th0t3p.github.io/playhold/reference/sdk-go.html) in playhold's documentation.

## Versions

An SDK `v0.X.y` is built for playhold `v0.X`, and works with any playhold `v0.X.*`. To use playhold 0.4.3, take the
latest 0.4.x SDK.

The frames come from playhold's `wire` module, `github.com/ny4rl4th0t3p/playhold/wire`, which needs only protobuf, so a
program can use both playhold and this SDK: Go's protobuf registry is global to a program, and both share the one
package. `v0.3.0` generated its own copy of the frames instead, so a program with it and playhold `v0.3.0` panics at
start; use `v0.4.0` or later with playhold in one program.

## Development

Needs Go and network access: the tests run against `testserver/`, a small game server in a module of its own, so this
one never requires playhold itself, only its `wire` module.

```bash
make check   # build, vet, test, lint, vuln, and that both go.mod files require the versions in the Makefile
make tidy    # tidy both modules
```

To move to a new playhold version: change `PLAYHOLD_VERSION` in the `Makefile`, require playhold's `wire` module at it
in `go.mod` and playhold at it in `testserver/go.mod`, then `make tidy check`.

## License

[Apache-2.0](LICENSE).