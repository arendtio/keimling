# Lab: Client–server + WebView discovery (2026-10-25)

## Context

Spike notes before implementation. Requirements live under `requirements/`.

Current `./keimling` ≈ 7.3 MB (no webview, no embedded UI yet). Budget &lt; 20 MB.

## WebView candidates

| Option | Fit | Size / weight | Notes |
| --- | --- | --- | --- |
| **`github.com/webview/webview_go`** | Strong | ~+1 MB class; CGO | Minimal API: `Navigate(url)`, `SetHtml`. Ideal if UI is 100% HTTP from agent. Upstream Linux pkg-config pinned to **webkit2gtk-4.0**; this Arch box has **4.1 only**. Need replace/fork/build-tag for 4.1 (known community issue). |
| **Wails v2/v3** | Weak for us | Still small vs Electron, heavier framework | Owns app model, bindings, often Node frontend tooling. Overkill when browser and GUI must share server-served UI and remote Navigate matters. Go ≥ newer than our current module floor may be required for v3. |
| **Lorca** | Weak | Uses installed Chrome | Bundles no Chromium but depends on Chrome/Chromium present — worse than system WebKit/WebView2 for “small Linux agent” hosts. |
| **Goleo / experimental purego** | Watch | cgo-free claims | Too early to bet the product on; revisit if CGO/cross-compile becomes painful. |

### Tentative choice

**Minimal webview binding + `Navigate` to the agent URL** (local or remote). Prefer fixing/forking `webview_go` so Linux builds against **4.0 or 4.1** (pkg-config detect / dual cgo tags) over adopting Wails.

macOS/Windows already map to WKWebView / WebView2 inside that binding — no WebKitGTK there.

Build tags idea: `//go:build gui` so `server`/`cli`-only builds can omit CGO on headless CI if needed — optional later.

## Process model (see also `requirements/technical/02-process-model.md`)

**Winner:** Agent server always serves UI + API. WebView is a window onto that URL. Remote attach = no local agent. Local use = embedded localhost agent + Navigate.

Rejected for MVP: local UI-only reverse proxy to remote agent.

## ANSI in GUI

MVP: text panel + small ANSI→HTML/CSS mapper (write ourselves or tiny dependency). Avoid xterm.js unless size stays trivial after measurement.

## Spike results (2026-10-25 evening)

Location: `docs/demos/webview-spike/`

| Binary | Flags | Size |
| --- | --- | --- |
| `keimling` (current product) | `-s -w`, `CGO_ENABLED=0` | **7 667 872** (~7.3 MiB) |
| spike HTTP-only | `-s -w`, no webview | **5 836 960** |
| spike WebView | `-s -w`, `-tags webkitgtk_41` | **6 739 640** |
| **WebView delta** | vs HTTP-only | **~0.90 MiB** |

- Build/run on Arch with **webkit2gtk-4.1** succeeded (`Navigate` to ephemeral `127.0.0.1` HTTP).
- Dependency: `github.com/samcharles93/webview_go v0.14.0` (Go 1.26+ module floor).
- Makefile auto-picks `webkitgtk_41` / `webkitgtk_60`; 4.0 needs the patch sketch in `patches/webkitgtk_40_unix.go`.
- Headroom to 20 MiB: large (product + webview + vanilla UI assets should stay well under).

## Next steps

1. ~~Minimal module + size~~ done.
2. ~~HTTP surface sketch + flag freeze~~ done.
3. ~~TDD shell + `--server`~~ done.
4. ~~SSE `/events`, `/api/message`, `/api/session`, `/api/approve` + agent Emit~~ done.
5. ~~CLI/`--connect` client~~ done.
6. ~~WebView (`--ui` / auto) + embedded localhost for local GUI~~ done (`samcharles93/webview_go`, Go 1.26, `make build` auto WebKit tag).
7. ~~ANSI→HTML in the web UI~~ done (`webui/assets/ansi.js`).
8. ~~`commands.md` registry~~ done.
9. ~~Streamlined chat UI + semantic chapters sidebar~~ done.
10. ~~Gate overlay fix, no nested scroll, tool_group cards, connection/resume flow~~ done.
11. Default `--cli` / auto-without-GUI still uses classic stdin loop — optional unify later.
12. Full config editing in UI (write `config.json`) — deferred; Connection panel is the start.
13. WebKitGTK 4.0 tag still missing upstream — Linux-only 4.0 distros need patch.

## Decisions locked (2026-10-25 evening)

- **Flags, not subcommands** for presentation modes.
- **HTTP only** in-process; TLS via external proxy; localhost plaintext OK.
- **Config parity** for all CLI options including secret/listen/connect.
- **Linux:** flexible WebKitGTK 4.0 **or** 4.1. **macOS/Windows:** system webview only.

## Still open

- Whether local ephemeral server reuses `access_secret` from config or always a launch-only random token (lean: always random for embedded local; config secret for `--server` / `--connect`).

## Flag names frozen

See `requirements/technical/03-http-api.md`: `--ui` / `--cli` / `--server` / `--listen` / `--connect` / `--access-secret`, config `presentation`, `listen`, `connect`, `access_secret`. Event push: **SSE**, not WebSocket, for zero extra deps.
