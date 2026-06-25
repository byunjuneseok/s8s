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

## Getting started

```sh
make build            # builds ./bin/s8s
./bin/s8s configure   # add a context (API Key / Secret Key)
./bin/s8s             # launch the TUI
```

`configure` writes `~/.s8s/config.yaml` and sets the new context as current.
It also runs non-interactively:

```sh
s8s configure --context toss-main --api-key <KEY> --secret-key <SECRET>
```

## Commands

Press `:` to open the command bar, `?` for help.

| Command | What it does |
| --- | --- |
| `:holdings` | Positions and per-currency totals (the default screen) |
| `:watch` | Watchlist quotes; `:watch add <sym>` / `:watch rm <sym>` |
| `:orderbook <sym>` (`:ob`) | Bid/ask depth for a symbol |
| `:orders` | Order list; `m` to modify, `c` to cancel the selected order |
| `:order` | Open the buy/sell entry modal (blocked in read-only contexts) |
| `:ctx` | List contexts; `:ctx use <name>` switches and persists |
| `:account` (`:acct`) | List accounts and choose the active one |
| `:refresh` | Refresh the current screen |
| `:quit` / `:q` | Quit |

Market data refreshes by polling; only the visible screen polls, and the poller
backs off when the broker's rate-limit headers run low.

## Configuration

`~/.s8s/config.yaml` follows a kubeconfig-style layout: a `context` references a
`broker`, which references a `credential`. Mark a context `read-only: true` to
block all order placement, modification, and cancellation.

Secrets need not be stored in cleartext. A credential's `client-secret` (or
`client-id`) may be a reference:

- `${env:VAR}` — read from the environment variable `VAR`.
- `keychain:SERVICE` (or `keychain:SERVICE/ACCOUNT`) — read from the macOS
  Keychain via `security find-generic-password`.
- anything else — used as a plaintext value.

## Layout

```
cmd/s8s            entry point + TUI wiring
internal/domain    broker-neutral domain models
internal/broker    Broker interface (+ provider adapters in subpackages)
internal/config    ~/.s8s/config.yaml loading, validation, secret resolution
internal/session   active-context manager (pure logic)
internal/poll      polling scheduler with rate-limit backoff
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
