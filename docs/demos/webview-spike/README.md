# WebView spike

Minimal **Go HTTP server + system WebView** (`Navigate` to localhost).

## Dependency

[`github.com/samcharles93/webview_go`](https://github.com/samcharles93/webview_go) — maintained fork of `webview/webview_go` with Linux build tags:

| Tag | pkg-config |
| --- | --- |
| `webkitgtk_41` | `gtk+-3.0` + `webkit2gtk-4.1` |
| `webkitgtk_60` | `gtk4` + `webkitgtk-6.0` |
| `webkitgtk_40` | `gtk+-3.0` + `webkit2gtk-4.0` (not in upstream fork yet; see below) |

macOS → WKWebView, Windows → WebView2: **no build tags**, no extra browser package in our binary.

## Build

```sh
make detect
make build    # auto-picks 4.1 → 4.0 → 6.0 on Linux
make size
make baseline-size
make run      # opens a window (needs DISPLAY / GUI session)
```

## WebKitGTK 4.0

This machine and many current distros only ship 4.1+. The fork’s 8-line pattern for 4.1 applies equally to 4.0: add `webkitgtk_40_unix.go` with `pkg-config: gtk+-3.0 webkit2gtk-4.0` and adjust build constraints so the tags are mutually exclusive. Until that lands upstream (or in a Keimling `replace`), `make build` prints a note when only 4.0 is detected.

## Size (this machine, 2026-10-25)

With `-ldflags='-s -w'` (same as product `Makefile`):

| Binary | Bytes |
| --- | --- |
| HTTP-only baseline | ~5.8 MiB |
| + WebView (`webkitgtk_41`) | ~6.7 MiB |
| Delta | **~0.9 MiB** |
| Current `keimling` | ~7.3 MiB |

Details: `docs/lab/2026-10-25-client-server-discovery.md`.
