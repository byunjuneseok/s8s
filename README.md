# s8s

A terminal UI for viewing securities accounts and trading.

Two core ideas:

1. **Resource navigation** — switch screens (holdings / orders / orderbook /
   watchlist) with `:` commands.
2. **Context switching** — store per-brokerage and per-account settings in
   `~/.s8s/config.yaml` and switch between them with `:ctx`.

The first backend is Toss Securities (OpenAPI, OAuth2 client-credentials).
The domain model is broker-neutral so other brokerages can be added behind the
same `Broker` interface.

## Status

Early scaffold. See the project board for milestones M0–M4.

## Layout

```
cmd/s8s            entry point
internal/domain    broker-neutral domain models
internal/broker    Broker interface (+ provider adapters in subpackages)
internal/config    ~/.s8s/config.yaml loading & validation
internal/tui       terminal UI shell and views
```

## Develop

```sh
make build    # build ./bin/s8s
make run      # build and run
make test     # go test ./...
make lint     # golangci-lint
```

Requires Go 1.22+.
