# Command registry

Approved commands for build, test, and run. Prefer these over ad-hoc variants.

## Build

| Command | Purpose |
| --- | --- |
| `make detect` | Print OS, CGO, and WebKit build tags |
| `make build` | Release binary `./keimling` (GUI when WebKit available) |
| `make build-nogui` | CLI/server-only binary (`-tags nogui`, `CGO_ENABLED=0`) |
| `make clean` | Remove `./keimling` and Go test cache |

Equivalent without make (nogui):

```sh
cd src && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -tags nogui -o ../keimling .
```

GUI on Linux (WebKitGTK 4.1):

```sh
cd src && CGO_ENABLED=1 go build -trimpath -ldflags='-s -w' -tags webkitgtk_41 -o ../keimling .
```

## Quality

| Command | Purpose |
| --- | --- |
| `make fmt` | `gofmt -l -w` on `./src` |
| `make vet` | `go vet -tags nogui` |
| `make test` | `go test -tags nogui ./...` |
| `make check` | `fmt` + `vet` + `test` |

## Run

| Command | Purpose |
| --- | --- |
| `make run ARGS='…'` | `make build` then `./keimling …` |
| `./keimling --help` | Flag list |
| `./keimling --server --listen HOST:PORT --access-secret SECRET` | HTTP agent + UI |
| `./keimling --connect URL --access-secret SECRET` | CLI attach (optional `--cli`) |
| `./keimling --ui` | Force native WebView (local embedded agent) |
| `./keimling --ui --connect URL --access-secret SECRET` | WebView against remote |
| `./keimling --cli` | Force shell REPL (local agent) |

## Spike / demos

| Command | Purpose |
| --- | --- |
| `make -C docs/demos/webview-spike detect` | WebKit tag for spike |
| `make -C docs/demos/webview-spike build` | Build spike binary |
| `make -C docs/demos/webview-spike size` | Compare sizes |

## Notes

- Default `make check` / `make test` use **nogui** so CI and headless hosts need no WebKit.
- Desktop GUI builds need system WebView (WebKitGTK / WKWebView / WebView2).
- Do not add new lifecycle commands without updating this file.
