# CLI / Web-UI parity audit

This document maps what the **Web UI** (`src/webui/`) and the **CLI clients** can
do for each product feature, what the **server** offers behind it, and where a
gap remains. It is an audit only: nothing here changes behaviour.

## Two CLI surfaces, one UI

Keimling has one frontend (`src/webui/`) and two terminal clients, so "CLI"
always means one of these:

| Surface | Entry | What it is |
| --- | --- | --- |
| **Local REPL** | `--cli` (`main.go:226`, presentation resolution `main.go:281`, REPL loop `main.go:659`) | The agent runs in this process. All tools are available to the model and the REPL owns the transcript. No HTTP server is started (`main.go:642-660`). |
| **Remote client** | `--connect` (`main.go:308-330` -> `runRemoteCLI`, `src/run_remote_cli.go:12`) | Thin HTTP/SSE client against a running server (`src/remote_client.go`). It sends messages and approvals and prints events. |

Tool parity is therefore **automatic**: `definitions()` (`openai.go:91-121`) plus
`chatExtraTools()` (`agent_tools.go:42-50`) build one tool list for every
surface, and a remote client can trigger any tool by asking the model. The real
differences are (a) direct user controls (slash commands, panels) and
(b) how much of the event stream a client renders.

## Method

Every cell was read from the source, not inferred from names:

- Server routes: `appserver.go:162-242` (`appServer` mux) and
  `host.go:495-507` (host/agent routes).
- Agent tools: `openai.go:91-121`, `agent_tools.go:13-81`,
  `agent_define.go:23-36`, `agent_watch.go:12`, `datalang_tool.go:197`,
  `low_context.go:69`.
- Event kinds: `appserver.go:80-125` (emit -> hub broadcast), `present.go:10-145`.
- Web UI wiring: `apiURL("/api/...")` call sites and the corresponding DOM
  handlers in `src/webui/assets/app.js`.
- CLI: the REPL command chain (`main.go:671-741`, `main.go:803`,
  `terminal.go:349-381`), `slash_completion.go:8-11`, `mcp_cli.go`,
  `skills_cli.go`, and the whole of `remote_client.go` / `run_remote_cli.go`.

Legend for the **Gap** column:

- **-** parity for the intended surface.
- **Model tool only** - no human-facing control outside chat, but the tool exists
  on both surfaces.
- **CLI missing** - the server/UI can do it; the CLI client cannot reach it.
- **Rendering** - the CLI lacks the presentation, not the capability.
- **Backend work** - the CLI would need a new transport/backend, not just wiring.
- **Unclear** - the code is ambiguous; see [Open questions](#open-questions).

## Agents and instances

Routes live on the host mux: `host.go:495-507`. UI: instance list
`app.js:3963-4090`, teams `app.js:4817-5083`, recipes `app.js:4934-5000`.

| Feature | Web UI | CLI | Backend / route | Gap |
| --- | --- | --- | --- | --- |
| Agent definition create/update | Editor, `POST /api/agents/recipes`, `/api/agents/recipes/update` (`app.js:4934`, `app.js:4976`) | Model tool `save_agent` (`agent_define.go:32`); no slash command | `agent_define.go:23-35`, `host.go:495-497` | CLI missing (no direct control; chat only) |
| Starting instructions | `agent_instructions` tool only | same tool (both surfaces) | `agent_define.go:31` | Model tool only |
| Spawn agent instance | Spawn form `app.js:5030` (`POST /api/agents/instances`) | Model tool `spawn_agent`; falls back to in-process `createAgentInstance` without a host (`agent_define.go:95-130`) | `agent_define.go:95-130`, `host.go:498` | CLI missing (no direct control) |
| Teams create/delete | `POST /api/agents/teams`, `/delete` (`app.js:5062`, `app.js:4676`) | Model tool `save_team` (`agent_define.go:34`) | `host.go:505-507`, `agent_team.go` | CLI missing |
| Spawn team | `POST /api/agents/teams/spawn` (`app.js:5083`) | Model tool `spawn_team` (`agent_define.go:209`) | `host.go:506` | CLI missing |
| List instances | Sidebar poll `app.js:4036-4074` | `list_agents` tool (`agent_tools.go:74`, impl `agent_tools.go:115`) | `host.go:498`, `agent_instance.go` | CLI missing |
| Message an instance | Instance chat (`app.js:4090` `attachAgent`) | `message_agent` tool (`agent_tools.go:75`, host path `agent_tools.go:288`) | `host.go:503`, `agent_queue.go` | Model tool only |
| Stop an instance | `POST /api/agents/instances/stop` (`app.js:4286`) | `stop_agent` tool (`agent_tools.go:76`) | `host.go:500` | Model tool only |
| Delete an instance | Confirm + `POST /api/agents/instances/delete` (`app.js:4333`) | `delete_agent` tool (`agent_tools.go:77`) | `host.go:502` | Model tool only |
| Reset an instance context | `POST /api/agents/instances/reset` (`app.js:4309`) | - | `host.go:501` | CLI missing |
| Contacts / access (allow peer) | Access editor `app.js:4179-4252` (`GET/POST /api/agents/instances/access`) | `allow_peer` tool (`agent_access.go`), used to gate `list_agents`/`message_agent` | `host.go:504`, `agent_access.go` | CLI missing |
| Switch workspace (instance) | Workspace switcher (`app.js:3599`) | `change_workspace` tool (`agent_tools.go:55`, impl `agent_tools.go:84-110`) | `agent_instance.go`, `workspace.go` | Model tool only |

## Threads, history, chat state

| Feature | Web UI | CLI | Backend / route | Gap |
| --- | --- | --- | --- | --- |
| New thread | `+` button -> `POST /api/threads` (`app.js:9709-9722`) | `/newchat` (`main.go:721`, `agent.go:573-600` creates a thread when a workspace is set) | `appserver.go:170`, `appserver.go:221` (`/api/newchat`) | - (local CLI). Remote CLI: CLI missing |
| List / search threads | Thread list + search `app.js:3840-3893` (`GET /api/threads?q=`, `appserver.go:639-659`) | - | `appserver.go:170`, `workspace.go:342`, `workspace.go:483` | CLI missing (remote and local) |
| Switch thread | `POST /api/threads/switch` (`app.js:3898`) | - (one thread per process, `main.go:535`) | `appserver.go:171` | CLI missing |
| Thread title | Set by the server's `thread` event (`app.js:6462`, `thread_title.go`) | - (same event, not rendered) | `appserver.go:170`, `thread_title.go` | Rendering |
| Delete thread | Confirm + `POST /api/threads/delete` (`app.js:3938`) | - | `appserver.go:172` | CLI missing |
| Search inside the session journal | Model tools `search_history` / `read_history` | same tools (`openai.go:98-99`) | `history_index.go` | - |
| Thread model override | `GET/PUT /api/threads/model` (`app.js:4850`, `app.js:6302`, handler `appserver.go:933-948`) | - (only `--model` at start, `main.go:204`) | `appserver.go:174`, `thread_model.go` | CLI missing |
| Low-context / deliver | `GET /api/threads/low-context`, `/deliver` | - | `appserver.go:175-176`, `low_context.go` | CLI missing |
| Chat branches (edit + reselect) | Branch nav `app.js:5452-5577` (`POST /api/branch`, `/branch/select`) | - | `appserver.go:216-217`, `requirements/technical/08-chat-branches.md` | CLI missing |
| Stop the run | `POST /api/stop` (`app.js:10351`, handler `appserver.go:2822`) | `/stop` while running (`main.go:803`); remote CLI: no stop | `appserver.go:220` | Remote CLI missing |
| Compact now | `POST /api/compact` (`app.js:7228`) + progress (`app.js:6461`) | `/compact [text]` (`main.go:691-704`) | `appserver.go:179`, `compaction.go` | - (local). Remote: CLI missing |
| Plan display | Side plan from the charter (`app.js:6937-6996`) | Plan printed as text lines by the tool (`parallel_tools.go:279-291` -> `renderPlan`) | `update_plan` (`openai.go:104`), `charter.go` | Rendering (no persistent side plan in the CLI) |
| Task finish | `presentFinish` bubble (`present.go:114-142`) | same text in the terminal | `finish_task` (`openai.go:105`) | - |

## Watches, autopilot, delegation

| Feature | Web UI | CLI | Backend / route | Gap |
| --- | --- | --- | --- | --- |
| Create/list/cancel watches | Rendered as watch messages only (`app.js:5386-5387`); no management UI | Tools `create_watch`, `list_watches`, `cancel_watch` (`agent_watch.go:12`) - but a *thread* watch needs a wake loop (`agent_watch.go:57-66`) | `agent_alarms.go:63-92`, loop started at `appserver.go:131-136` | CLI missing (thread watches only exist under `--server`; instance watches need the host) |
| Watch wake notices | Banner in chat (`appserver.go:3386`) | Local CLI: printed when the loop runs; remote CLI ignores `message` events | `agent_watch_run.go` | Rendering (remote) |
| Autopilot toggle | Settings switch (`app.js:226-253`, `POST /api/threads/autopilot`) | `/autopilot on\|off` (`main.go:705-707`, `main.go:915-928`) | `appserver.go:173`, `autopilot.go` | - (local). Remote: CLI missing |
| Autopilot instructions | Editor + save (`app.js:2283-2350`) | - (config file only, `features.autopilot_instructions`) | `appserver.go:222` (`/api/settings`), `autopilot_instructions.go` | CLI missing |
| Delegate | Tool card in the chat | same tool (`openai.go:102`) | `delegate.go`, `requirements/technical/07-vision-delegate.md` | - |

## Artifacts, files, attachments

| Feature | Web UI | CLI | Backend / route | Gap |
| --- | --- | --- | --- | --- |
| `show_artifact` | Artifact panel + sandboxed iframe (`app.js:64-82`, event `app.js:6457`) | Prints title + URL (`artifact.go:122-127`); banner via `presentToolStart` (`present.go:53-68`) | Tool `openai.go:96`, `artifact.go:75-127` | Rendering (no panel/history) |
| Artifact history | `GET /api/artifacts` (`app.js:8668`, handler `appserver.go:1299`) | - | `appserver.go:214` | CLI missing |
| `share_file` | Link opened in the UI; proxied via `/api/artifact-proxy` (`appserver.go:193`) | Prints capability URL (`parallel_tools.go:325-327`) | `file_share.go`, `/files/` (`appserver.go:242`) | - |
| Attach files / images | Upload + drag & drop (`app.js:10126-10155`, `POST /api/attachments`, handler `appserver.go:3085`) | `@path` refs resolved before the call (`main.go:749`, `chat_files.go:127`) | `appserver.go:218`; ids in the message body `appserver.go:2985-3000` | CLI missing (no upload; only paths that already exist) |
| Workspace file picker | `GET /api/files?q=` (`appserver.go:219`) | `@path` completion only | `appserver.go:219` | UI-only wiring (backend exists) |

## Voice, realtime, screen

| Feature | Web UI | CLI | Backend / route | Gap |
| --- | --- | --- | --- | --- |
| Voice recording | Record button (`app.js:11553-11612`) -> `POST /api/transcribe` (`app.js:11498`) | F1/Space/Enter or `/voice` (`terminal.go:349-381`), transcribed in-process | `appserver.go:231` (`handleTranscribe` `appserver.go:3741`), `openai.go:394` | - (different transport, same model call) |
| Realtime call (voice loop) | WebRTC/WS (`app.js:10421-10586`), routes `appserver.go:234-240` | - | `realtime.go`, `requirements/features/19-realtime-call.md` | Backend work (browser transport: WebRTC + mic) |
| Screen sharing | `GET /api/screen/targets`, `POST /api/screen/share` (`app.js:11330-11376`, handler `screen_frame.go:106`) | - | `appserver.go:232-233`, `screen_portal_linux.go` | Backend work |
| Screen input tools (`screen_click` ...) | Work while a screen is shared | Error without a shared screen (`screen_input.go:110-115`) | `screen_input.go:75-105` | Backend work (input capture lives in the UI session) |

## Context, credentials, skills, MCP, data

| Feature | Web UI | CLI | Backend / route | Gap |
| --- | --- | --- | --- | --- |
| Context/budget display | Budget line (`app.js:6109-6144`) via `GET /api/context` (`app.js:7011`) | `/status` prints the same budget (`main.go:881-913`) | `appserver.go:177`, `context_usage.go`, `budget.go` | - (local). Remote: `/status` absent |
| Provider budgets | Settings tab (`/api/settings/budget`, `appserver.go:224`) | - | `provider_budget.go` | CLI missing |
| Credential vault CRUD | List/editor (`app.js:1530-1692`) via `/api/credentials*` | Unlock only at startup (`main.go:600-605`) or `KEIMLING_VAULT_PASSWORD`; no list/add/rotate | `appserver.go:227-229`, `credentials_api.go` | CLI missing |
| Credentials for tools | Model tool `list_credentials` (`agent_tools.go:13-27`) | same tool (both surfaces) | `credentials.go` | - |
| Skills list/search/preview/review/install/check/update/remove/add | Skills settings (`app.js:7509`, `/api/skills/*`) | `/skills ...` (`skills_cli.go:11-236`), same operations | `appserver.go:180-191`, `skills*.go` | - |
| Skills enable/disable + policy | `POST /api/skills/enabled`, `/api/skills/policy` (`appserver.go:181-182`) | - | `skills_prefs.go` | CLI missing |
| MCP list/add/remove/reload | MCP settings (`app.js:4792`, `app.js:8269`, `app.js:8305`) | `/mcp [list\|reload\|add\|remove]` (`mcp_cli.go:10-70`) | `appserver.go:192-198`, `mcp*.go` | - |
| MCP reload-one / log / allow / OAuth sign-in | `/api/mcp/reload-one`, `/log`, `/allow`, `/sign-in` (`appserver.go:199-202`) | - | `mcp_oauth.go`, `mcp_lazy.go` | CLI missing |
| MCP tool discovery | `find_tools` (`low_context.go:69`) | same tool (`agent_tools.go:48`) | `low_context.go`, `mcp.go:469` | - |
| Data processing | `data_processing` tool (`datalang_tool.go:197`) | same tool | `datalang*.go`, `requirements/technical/06-datalang.md` | - |
| Plugins | Plugin settings (`app.js:1271-1473`, `/api/plugins*`) | - | `appserver.go:203-211`, `plugins.go`, `plugins_http.go` | CLI missing |
| Settings / providers / model catalog | Settings tabs (`app.js:1485-1535`), `/api/settings`, `/api/settings/model-catalog` | Flags only (`main.go:197-234`); no runtime change | `appserver.go:222-223` | CLI missing |
| Client connections / GUI prefs | Address book UI (`app.js:1055-1125`) | `--connect` name/id from `connections.json` (`client_connections.go:205-235`) | `appserver.go:225-226` | - (read path only; no add/remove from the CLI) |
| Workspaces list/switch/hide/rename/restore | `app.js:3431-3638` | `--workspace` flag (`main.go:221`); `change_workspace` tool for instances | `appserver.go:164-169` | CLI missing (no runtime switch) |

## Open questions

Places where the code allows more than one reading. They are marked instead of
guessed:

1. **Remote CLI and assistant text.** `readSSE` handles only `ready`, `line`, and
   `status` (`remote_client.go:163-195`). Assistant output is a `message` event
   (`present.go:32-36`, broadcast at `appserver.go:87-91`), and `cliWrite`
   explicitly does not mirror to `line` (`present.go:9-15`). Reading of the code:
   the remote CLI shows progress/plan/share lines and the startup history
   (`/api/session`, `appserver.go:291-320`), but not the assistant's answer text
   or tool cards. **Not verified by a manual remote run**; treat as a finding to
   confirm before planning work.
2. **`/transcribe` in the REPL.** `main.go:730-733` answers "Use F1 or /voice to
   record audio." - it is a stub, while `/voice` (`terminal.go:360`) records. The
   intended command name is unclear.
3. **`/stop` in the local REPL.** It is handled only inside the running-task loop
   (`main.go:803`), not in the outer command chain (`main.go:671-741`); the help
   text (`main.go:868`) and `slash_completion.go:9` still advertise it.
4. **Watches in `--cli`.** A workspace thread exposes the watch tools only when
   `WatchesLive` is set (`agent_tools.go:59-61`), which only the app server does
   (`appserver.go:131-136`). Whether a standalone `--cli --agent-instance` is
   expected to fire watches is not stated anywhere; the tools are offered for
   instances, but no wake loop runs outside the server.
5. **Doc drift.** `docs/agent-loop.md` lists the slash commands without `/skills`
   and `/mcp`, while `main.go:842-877` documents both.

## Prioritised gaps

### A. CLI missing, cheap to add (client/REPL wiring)

1. **Remote CLI renders only `line`/`status`.** No `message`, `tool`, `artifact`
   events (`remote_client.go:163-195`). Answers and tool cards stay invisible
   (see open question 1).
2. **Remote CLI has almost no commands.** Only `/exit`, `/quit`, `/approve`,
   `/deny` (`run_remote_cli.go:52-85`); missing `/stop`, `/status`, `/newchat`,
   `/compact`, `/mode`, `/autopilot`, `/skills`, `/mcp`, `/voice`.
3. **No thread control in the CLI.** List/search/switch/delete exist as routes
   (`appserver.go:170-172`, `appserver.go:639-659`) but have no CLI path; the
   local REPL only creates threads (`/newchat`).
4. **No runtime model/provider switch.** Flags at start (`main.go:197-234`) and
   `/status` (`main.go:881`) only; the UI uses `/api/settings` +
   `/api/threads/model` (`appserver.go:174`, `appserver.go:222-223`).
5. **No credential management.** Only the startup unlock (`main.go:600-605`) and
   the model-side `list_credentials` tool; `/api/credentials*`
   (`appserver.go:227-229`) is unreachable.
6. **Skills enable/disable and policy** (`appserver.go:181-182`) have no CLI
   equivalent, although the rest of the skills surface does
   (`skills_cli.go:11-236`).
7. **MCP extras** - reload-one, log, allow-list, OAuth sign-in, start, before
   (`appserver.go:195-202`) are UI-only; `/mcp` covers list/add/remove/reload
   (`mcp_cli.go:10-70`).
8. **Artifact history / panel** (`appserver.go:214`) and **attachments**
   (`appserver.go:218`) are UI-only; the CLI prints URLs and reads `@path`.
9. **Workspace switching at runtime** (`appserver.go:164-169`) is UI-only; the
   CLI has `--workspace` and the instance-only `change_workspace` tool.

### B. UI wiring only (backend already there)

- Threads, workspaces, context/budget, compaction, settings, credentials,
  plugins, chat branches all have routes and server logic; a CLI command would
  only need a client call.
- Artifact panel and attachment upload: the server side is generic
  (`/api/artifacts`, `/api/attachments`); the CLI needs a rendering choice, not
  new backend behaviour.

### C. Backend work required

- **Realtime call** (`appserver.go:234-240`, `realtime.go`): browser WebRTC/WS
  transport; a CLI client would need a different audio path.
- **Screen sharing and screen input** (`appserver.go:232-233`,
  `screen_input.go:110-115`): capture and input live in the browser session;
  without one, the tools refuse to run.
- **Watches outside `--server`**: a wake loop only exists in the app server
  (`appserver.go:131-136`); a standalone CLI would need its own loop.
