# Technical: Process model (working decision)

## Core rule

**The agent server is the source of truth and the UI host.**  
Whatever you see in the WebView is the same HTML/CSS/JS the server would serve to a browser. The WebView does not ship a second UI copy for remote sessions.

## Workspaces and agents

Traces to REQ-WS-6, REQ-WS-7, REQ-WS-8, REQ-WS-11, REQ-AG-11.

One process serves the UI and runs every workspace and every agent instance. A workspace is not an operating-system process. Switching workspace is the same kind of change as switching thread. A run keeps going in that process while the user looks at another thread. Each workspace still has its own root: a session starts there, and a tool path is judged inside or outside that root. Each thread then keeps its own working directory, so a directory change in one thread does not move another (REQ-WS-11). An agent instance keeps its own queue and its own directory when it changes workspace among the folders it may use. Those are in-process, too.

Copy, paste, and the command-line fallback in REQ-WR-8 stay. The window is still created before any second copy of this binary would have been started; opening a workspace no longer starts one.

## Recommended topology

```
┌──────────────────────────────────┐
│  One process (UI + workspaces +  │
│  threads + agent instances)      │
└─────────────▲────────────────────┘
              │ same origin / same UI
     ┌────────┴────────┬──────────────────┐
     │                 │                  │
 CLI client      WebView client     Phone/browser
```

Optional: TLS reverse proxy in front of `--listen` for remote exposure.

### Local interactive use (auto / ui / cli without remote)

1. Start one process on `127.0.0.1` with an ephemeral port and a short-lived local token. The launch workspace is in that process, with its root as the session directory.
2. **GUI path:** open WebView → that URL. A workspace switch loads the other workspace in the same process.
3. **CLI path:** the same in-process agent for the launch workspace.
4. When that process exits, its runs stop.

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
