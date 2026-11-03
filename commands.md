# Command registry

Approved commands for build, test, and run. Prefer these over ad-hoc variants.

## Build

| Command | Purpose |
| --- | --- |
| `make detect` | Print OS, CGO, and WebKit build tags |
| `make build` | Release binary `./keimling` (system web view on the OS where it is built) |
| `make build-nogui` | CLI/server-only binary (`-tags nogui`, `CGO_ENABLED=0`) |
| `make clean` | Remove `./keimling` and Go test cache |

`make build` stamps the short git commit (`git rev-parse --short HEAD`). `./keimling --version` prints it. A build without that stamp reports `dev`.

Equivalent without make (nogui):

```sh
REV=$(git rev-parse --short HEAD)
cd src && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.buildVersion=$REV" -tags nogui -o ../keimling .
```

GUI on Linux (WebKitGTK 4.1):

```sh
REV=$(git rev-parse --short HEAD)
cd src && CGO_ENABLED=1 go build -trimpath -ldflags="-s -w -X main.buildVersion=$REV" -tags webkitgtk_41 -o ../keimling .
```

CLI/server binaries for other operating systems, written under `dist/` so `./keimling` stays the local build. The native window is built on that operating system with `make build` (CGO and the system web view).

```sh
REV=$(git rev-parse --short HEAD)
mkdir -p dist
cd src && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w -X main.buildVersion=$REV" -tags nogui -o ../dist/keimling-$REV-darwin-arm64 .
cd src && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w -X main.buildVersion=$REV" -tags nogui -o ../dist/keimling-$REV-darwin-amd64 .
cd src && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -X main.buildVersion=$REV" -tags nogui -o ../dist/keimling-$REV-windows-amd64.exe .
```

## GUI builds on GitHub

[`.github/workflows/gui.yml`](.github/workflows/gui.yml) runs on push to `main`, on pull requests, and via workflow dispatch. Artifacts:

| Artifact | Where it is built |
| --- | --- |
| `keimling-<commit>-linux-amd64` | `ubuntu-24.04`, CGO, `-tags webkitgtk_41` |
| `keimling-<commit>-darwin-arm64` | `macos-latest` |
| `keimling-<commit>-darwin-amd64` | `macos-15-intel` |
| `keimling-<commit>-windows-amd64.exe` | `ubuntu-24.04` cross-compile with Zig 0.17.1 (`x86_64-windows-gnu`), WebView2 headers from the webview module |

The Windows binary stays a console program, so `--cli` and `--server` still print to the terminal. Opening the window needs the WebView2 runtime, which Windows 11 already has.

Local check of that Windows GUI build, without replacing `./keimling`:

```sh
# Zig 0.17.1 extracted at /tmp/zig-x86_64-linux-0.17.1
cd src && CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  CC='/tmp/zig-x86_64-linux-0.17.1/zig cc -target x86_64-windows-gnu' \
  CXX='/tmp/zig-x86_64-linux-0.17.1/zig c++ -target x86_64-windows-gnu' \
  go build -trimpath -ldflags="-s -w -X main.buildVersion=$(git rev-parse --short HEAD)" -o /tmp/keimling-windows-gui.exe .
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
| `./keimling --compact-parallel=false` | Disable parallel distill/summarize (e.g. single local model) |

## Docker (server / nogui)

| Command | Purpose |
| --- | --- |
| `docker build -t keimling .` | Multi-stage image: static `nogui` binary on Debian slim + agent toolchain, listens on `0.0.0.0:7420` |
| `docker run --rm -p 7420:7420 -e OPENAI_API_KEY -e KEIMLING_SECRET -v keimling-data:/data keimling` | Run HTTP agent (put TLS on your reverse proxy) |

The image `CMD` is `--server --listen 0.0.0.0:7420`. State and CWD are `/data` (`HOME` + `WORKDIR`, uid `65532`). Mount the volume on `/data`. Secrets: `OPENAI_API_KEY` / `OPENAI_API_KEY_FILE`, `KEIMLING_SECRET` / `KEIMLING_SECRET_FILE` (Compose secrets). Runtime toolchain for `exec`: bash, curl, git, openssh-client, build-essential (make/gcc), Go (`golang-go`), Node (`nodejs`/`npm`), Python 3 + pip/venv, wget/jq/unzip — all from Debian apt except the Keimling binary. Go/npm/pip caches use `/tmp` (not persisted on the data volume). No WebKit/GUI.

## Lab

| Command | Purpose |
| --- | --- |
| `python3 docs/lab/agent-reply/run.py --variant rules` | Replay reply cases against llama.cpp at `http://127.0.0.1:8080/v1/responses` (Ling). The server must already be running. |

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
