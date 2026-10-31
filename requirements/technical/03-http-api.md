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
| `GET` | `/api/session` | yes | Session view: `messages`, `chapters`, `working_state`, `workspace_id`, `workspace_root`, `thread_id`, legacy `lines` |
| `GET` | `/api/workspaces` | yes | Host: list workspaces + `current` (attached) + `running` |
| `POST` | `/api/workspaces/open` | yes | Host: register folder, ensure worker, attach `{ "path" }` → `{ "relaunch": false }` |
| `POST` | `/api/workspaces/switch` | yes | Host: ensure worker + attach `{ "id" }` → `{ "relaunch": false }` |
| `GET`/`POST` | `/api/threads` | yes | Host: list workspace threads (`?workspace=`); create on workspace worker (reattaches workspace if an agent was open) |
| `POST` | `/api/threads/switch` | yes | Host→workspace worker thread switch `{ "id" }` (leaves agent view) |
| `POST` | `/api/threads/delete` | yes | Host→workspace worker delete `{ "id" }` |
| `POST` | `/api/message` | yes | Send user instruction `{ "text": "..." }` |
| `POST` | `/api/stop` | yes | Cancel in-flight agent run (worker) |
| `POST` | `/api/newchat` | yes | New thread in workspace (or legacy archive when no workspace) |
| `GET`/`POST` | `/api/agents/recipes` | yes | Host: list / create definitions |
| `POST` | `/api/agents/recipes/update` | yes | Host: update definition |
| `POST` | `/api/agents/recipes/delete` | yes | Host: delete definition (no remaining instances) |
| `GET`/`POST` | `/api/agents/instances` | yes | Host: list (`?workspace=`) / create instance + attach |
| `POST` | `/api/agents/instances/attach` | yes | Host: attach UI to agent worker `{ "id" }` |
| `POST` | `/api/agents/instances/stop` | yes | Host: stop run on instance `{ "id" }` |
| `POST` | `/api/agents/instances/delete` | yes | Host: delete instance `{ "id" }` |
| `POST` | `/api/agents/instances/message` | yes | Host: enqueue wake `{ id, text }` |
| `GET`/`POST` | `/api/agents/teams` | yes | Host: list / create teams |
| `POST` | `/api/agents/teams/spawn` | yes | Host: spawn team instances |
| `POST` | `/api/agents/teams/delete` | yes | Host: delete team |
| `GET`/`PUT` | `/api/settings` | yes | Runtime knobs (mode, web, autopilot, max_steps, context_limit, voice_model) plus `model_providers` / `active_model_provider_id` (mirrored to legacy `base_url` / `model` / `api_key`); write-only keys; GET exposes `api_key_set` / `openai_api_key_set` and per-provider `api_key_set`. `autopilot_instructions` replaces the factory supervisor prompt (blank is rejected); `autopilot_instructions_reset: true` clears that override. GET returns the effective text and `autopilot_instructions_custom`. See `04-auth-and-settings.md`. |
| `GET`/`PUT` | `/api/threads/autopilot` | yes | Current thread's supervisor prompt. GET: `instructions`, `custom`, `source` (`thread` / `global` / `factory`). PUT `{instructions}` saves a full replacement (blank rejected, including a copy of the global text). PUT `{reset:true}` clears it so the thread follows the global prompt again. |
| `GET`/`PUT` | `/api/gui-prefs` | yes | Desktop UI prefs + **server** `connections[]` / `active_connection_id` (legacy `connect` / `access_secret`). |
| `GET`/`PUT` | `/api/client-connections` | yes | **Machine** connection book (`connections.json` next to config). Prefer native bridge when the UI origin is a remote agent. |
| `GET`/`POST` | `/api/credentials` | yes | List credential metadata / create (vault must be unlocked). Secrets write-only. |
| `POST` | `/api/credentials/unlock` | yes | Unlock or create vault `{ password, create? }` for process lifetime. |
| `PUT`/`DELETE` | `/api/credentials/{id}` | yes | Update (optional new secret) / delete. |
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
