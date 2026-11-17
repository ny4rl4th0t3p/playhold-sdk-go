# playhold-sdk-go

Connects a Go program to a [playhold](https://github.com/ny4rl4th0t3p/playhold) server, for bots and load tests. It's
minimal on purpose: it doesn't reconnect by itself, because a load test has to see and count what goes wrong.

> **Status: early development,** like playhold itself: the API can change between 0.x versions.

```bash
go get github.com/ny4rl4th0t3p/playhold-sdk-go@v0.3.0
```

```go
import sdk "github.com/ny4rl4th0t3p/playhold-sdk-go"
```

How to use it: [Go SDK](https://ny4rl4th0t3p.github.io/playhold/reference/sdk-go.html) in playhold's documentation.

## Versions

An SDK `v0.X.y` is built for playhold `v0.X`, and works with any playhold `v0.X.*`. To use playhold 0.4.3, take the
latest 0.4.x SDK.

## Development

Needs Go and network access: the wire code is generated from playhold's proto at the version in the `Makefile`, and the
tests run against `testserver/`, a small game server on that same version, in a module of its own so this one never
requires playhold.

```bash
make check   # build, vet, test, lint, vuln, and that testserver/go.mod requires the pinned version
make gen     # generate internal/wire from playhold's proto at PLAYHOLD_VERSION
make tidy    # tidy both modules
```

To move to a new playhold version: change `PLAYHOLD_VERSION` in the `Makefile`, require the same version in
`testserver/go.mod`, then `make tidy gen check`.

## License

[Apache-2.0](LICENSE).