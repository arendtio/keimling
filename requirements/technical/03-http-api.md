# Technical: HTTP API (agent surface)

## Transport

- **HTTP/1.1** only in-process (no TLS). WebSocket-over-HTTP later only if SSE proves insufficient.
- **SSE** (`text/event-stream`) for server→client event push (transcript lines, approvals, status).
- **JSON REST** for client→server commands.
- Same origin serves **static UI** from `embed.FS` (vanilla HTML/CSS/JS).

## Auth

- Shared **access secret** (not the model API key).
- Accepted as:
  - `Authorization: Bearer <secret>`, or
  - `X-Keimling-Access-Secret: <secret>`
- Constant-time compare.
- Unauthenticated requests → `401`.
- Static UI HTML may load without secret so the login/connect form can render; **`/api/*` and `/events` always require secret**. (Alternative: protect everything except `GET /` and static assets under `/assets/`.)

## Endpoints (MVP)

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/` | no | UI shell (`index.html`) |
| `GET` | `/assets/*` | no | CSS/JS |
| `GET` | `/files/{token}` | capability token | Shared file download (see `05-file-shares.md`); no access secret |
| `GET` | `/api/health` | yes | Liveness + build/mode info |
| `GET` | `/api/session` | yes | Session view: `messages`, `chapters` (task-level history index), `working_state`, legacy `lines` |
| `POST` | `/api/message` | yes | Send user instruction `{ "text": "..." }` |
| `POST` | `/api/newchat` | yes | Archive current journal and start a fresh chat |
| `GET`/`PUT` | `/api/settings` | yes | Runtime agent knobs (model, mode, reasoning, web, autopilot, max_steps, context_limit, voice_model, `base_url`) plus write-only `api_key` / `openai_api_key`; GET exposes only `api_key_set` / `openai_api_key_set`. See `04-auth-and-settings.md`. |
| `POST` | `/api/transcribe` | yes | Multipart `audio` file → `{ "text": "..." }` (OpenAI transcription) |
| `GET` | `/events` | yes | SSE stream of agent events |

Exact JSON shapes evolve with tests; keep payloads small and stable.

### SSE event types (initial)

```
event: ready
data: {"ok":true}

event: line
data: {"stream":"out","text":"..."}

event: status
data: {"state":"idle"|"running"|"awaiting_approval"}
```

## Listen rules

| Context | Bind | Secret |
| --- | --- | --- |
| `--server` | `--listen` **required** (e.g. `0.0.0.0:7420`) | `--access-secret` **required** (or config) |
| Embedded local (auto/ui/cli) | `127.0.0.1:<ephemeral>` | process-generated launch token (same auth mechanism) |

## Flags and config (frozen names)

| CLI | Config JSON | Notes |
| --- | --- | --- |
| `--ui` | `"presentation": "ui"` | Force WebView; fail if unavailable |
| `--cli` | `"presentation": "cli"` | Force shell client |
| `--server` | `"presentation": "server"` | HTTP agent only |
| *(default)* | `"presentation": "auto"` or omit | Autodetect GUI → else CLI |
| `--listen` | `"listen"` | `host:port`; required for server |
| `--public-url` | `"public_url"` | Public base for `share_file` capability URLs |
| `--connect` | `"connect"` | Remote agent base URL; no local agent |
| `--access-secret` | `"access_secret"` | Shared client/server secret |

`--ui`, `--cli`, and `--server` are mutually exclusive. Config key `presentation` is the string form; CLI bool flags set it when passed (flags override config).

Example config fragment:

```json
{
  "presentation": "server",
  "listen": "0.0.0.0:7420",
  "access_secret": "change-me",
  "connect": ""
}
```

## Status

- Shell + auth + embed UI: done.
- `/api/session`, `/api/message`, `/api/approve`, `/events` (SSE): done; agent output tees via `Agent.Emit`.
- CLI `--connect` remote client: done.
- WebView (`--ui` / auto): done (system webview; `make build` / `make build-nogui`).
- HTTPS: out of scope (external proxy).
