# Technical: Process model (working decision)

## Core rule

**The agent server is the source of truth and the UI host.**  
Whatever you see in the WebView is the same HTML/CSS/JS the server would serve to a browser. The WebView does not ship a second UI copy for remote sessions.

## Workspaces

A **host** process serves the UI and proxies to **worker** OS processes (one per workspace, CWD = root). Switching workspace **reattaches** the UI; other workers keep running until the host exits. See `requirements/features/02-workspaces-and-threads.md`.

## Recommended topology

```
┌──────────────────────────────────┐
│  Host (UI + workspace API +      │
│  reverse-proxy to attached       │
│  worker)                         │
│    ├── Worker A (workspace A)    │
│    └── Worker B (workspace B)    │
└─────────────▲────────────────────┘
              │ same origin / same UI
     ┌────────┴────────┬──────────────────┐
     │                 │                  │
 CLI client*     WebView client     Phone/browser
```

\*CLI without host still runs a single in-process agent for the launch CWD.

Optional: TLS reverse proxy in front of `--listen` for remote exposure.

### Local interactive use (auto / ui / cli without remote)

1. Start a **host** on `127.0.0.1` with an ephemeral port and a short-lived local token; spawn a worker for the initial workspace.
2. **GUI path:** open WebView → host URL; workspace switch reattaches proxy, no relaunch.
3. **CLI path:** in-process agent for the launch workspace (no multi-worker host).
4. When the host exits, all workers are stopped.

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
