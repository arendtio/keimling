# Technical: Process model (working decision)

## Core rule

**The agent server is the source of truth and the UI host.**  
Whatever you see in the WebView is the same HTML/CSS/JS the server would serve to a browser. The WebView does not ship a second UI copy for remote sessions.

## Recommended topology

```
┌─────────────────────────────┐
│  Agent host (server mode    │
│  or local embedded agent)   │
│  - agent loop + tools       │
│  - embed.FS UI              │
│  - HTTP + WS API (no TLS)   │
│  - auth secret              │
└─────────────▲───────────────┘
              │ same origin / same UI
     ┌────────┴────────┬──────────────────┐
     │                 │                  │
 CLI client      WebView client     Phone/browser
 (API only)   (Navigate to URL)    (Navigate to URL)
```

Optional: TLS reverse proxy in front of `--listen` for remote exposure.

### Local interactive use (auto / ui / cli without remote)

1. Start an **embedded agent server** on `127.0.0.1` with an ephemeral port and a short-lived local token.
2. **GUI path:** open WebView → `Navigate(http://127.0.0.1:<port>/…)`.
3. **CLI path:** talk to the same localhost API (no WebView).
4. When the client exits, shut down the embedded server (unless later we add “detach server”; not MVP).

### Remote attach

1. Do **not** start a local agent (tools must not run on the client machine).
2. GUI: WebView → `Navigate(http://remote/…)` (or `https://` if a proxy terminates TLS).
3. CLI: `--connect <url>` + secret; API only.
4. Browser/phone: open the same URL.

### Server-only

- Process is only the agent HTTP server.
- `--listen` required; access secret required (config or flag).

## Why not “local UI server + proxy to remote agent”?

A local reverse-proxy that serves UI while the agent lives elsewhere duplicates assets, complicates auth/cookies/WS, and breaks “GUI == browser on the server.” Direct Navigate-to-remote keeps one UI.

## Presentation / mode flags (no subcommands)

Use normal CLI flags (and the same keys in `config.json`). **No** `keimling ui|cli|server` subcommands.

Frozen names: see `requirements/technical/03-http-api.md`.

| Intent | Flag | Config | Behavior |
| --- | --- | --- | --- |
| Autodetect | *(default)* | `"presentation":"auto"` | Try native GUI; on failure run CLI. |
| Force GUI | `--ui` | `"presentation":"ui"` | Require WebView; fail if unavailable. |
| Force CLI | `--cli` | `"presentation":"cli"` | Shell client only; never WebView. |
| Server only | `--server` | `"presentation":"server"` | HTTP agent only; no interactive client. |
| Listen | `--listen` | `"listen"` | Mandatory when server. |
| Connect | `--connect` | `"connect"` | Remote agent base URL; no local agent. |
| Access secret | `--access-secret` | `"access_secret"` | Auth for API/events. |

`--ui`, `--cli`, and `--server` are mutually exclusive; flags override config.

## Autodetect

1. If display/webview available → GUI + embedded local agent (unless `--connect`).
2. Else → CLI + embedded local agent (unless `--connect`).
3. `--ui`: step 1 only; failure exits non-zero.
4. `--cli`: step 2 only.
5. `--server`: no client loop.

## GUI content MVP

- Scrollable text transcript with ANSI/color interpretation (small JS helper or minimal lib).
- Input for instructions; approval controls as needed.
- No full PTY terminal widget in MVP.

## Config parity

Every runtime option exposed on the CLI (presentation mode, listen, connect, access secret, existing agent options, …) must be representable in `~/.config/keimling/config.json` (or configured path). CLI wins over config.
