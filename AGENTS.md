# AGENTS.md — Keimling

Orientation for coding agents. Prefer this over guessing; deeper design lives under `docs/` and `requirements/`.

## What this is

Keimling loads this file (and nested `AGENTS.md` / `AGENTS.override.md` from the
Git root down to the process CWD, plus an optional global file beside
`config.json`) into model instructions each turn — [agents.md](https://agents.md/)
convention. Deeper files override broader ones; safety/approvals and explicit
user instructions still win.

Keimling is a Linux-first interactive coding agent in **Go** (OpenAI **Responses** API). One binary can run as:

- **CLI** REPL (`--cli`)
- **Native WebView** GUI (`--ui`)
- **HTTP server** (`--server --listen …`) serving the same embedded Web UI
- **Host** that proxies the UI to **workspace / agent-instance workers**

There is **one UI** (`src/webui/`, embedded via `//go:embed`). Remote and local clients use that UI; the WebView is not a separate frontend stack.

## Layout

| Path | Role |
| --- | --- |
| `src/` | Go module `keimling` (`package main`); all implementation |
| `src/webui/` | Embedded HTML/CSS/JS (`index.html`, `assets/app.js`, `app.css`, …) |
| `commands.md` | **Approved** build/test/run commands — prefer these |
| `Makefile` | Wraps `commands.md` targets; binary lands at `./keimling` |
| `docs/` | Product/architecture docs (`concept.md`, `agent-loop.md`, …) |
| `requirements/` | Specs: `features/`, `technical/`, `user-stories/` |
| `Dockerfile` | Static `nogui` server image + exec toolchain |

## Commands (use these)

See [`commands.md`](commands.md). Day-to-day:

```sh
make check          # gofmt + go vet -tags nogui + go test -tags nogui
make test           # tests only (nogui)
make build          # GUI if WebKit available
make build-nogui    # CGO_ENABLED=0 -tags nogui
make run ARGS='…'
```

Default tests/CI use **nogui** (no WebKit). Do not invent new lifecycle commands without updating `commands.md`.

## Architecture map (code)

| Area | Key files |
| --- | --- |
| Entry / flags / config | `main.go` (`fileConfig`, `~/.config/keimling/config.json`) |
| Agent loop | `agent.go`, `control.go`, `openai.go`, `policy.go` |
| Tools | `tools.go`, `edit_file.go`, `agent_tools.go`, `pty_*.go` |
| HTTP API + SSE | `appserver.go`, `eventhub.go` |
| Host / workers | `host.go`, `host_agents.go`, `workspace.go`, `agent_instance.go` |
| Session / UI transcript | `session_view.go`, journal JSONL; GUI consumes SSE `message` / `tool` via `present.go` |
| Presentation | `present.go` (structured GUI events + CLI stdout), `presentation.go` (ANSI) |
| Compaction / charter | `compaction*.go`, `charter*.go`, `history_index.go` |
| Credentials | `credentials*.go` (vault, MCP inject, exec env, SSH agent) |
| GUI shell | `gui_webview.go` (`//go:build cgo && !nogui`), `gui_stub.go` |
| Web client | `webui/assets/app.js` (composer, SSE, settings), `app.css` |

**Mental model:** Agent logic emits **semantic events** (`message`, `tool`, `status`, …). CLI renders ANSI; GUI updates a `messages` transcript and re-renders. On idle, `GET /api/session` reloads the journal-backed view (`session_view.go`). Do not revive a parallel “live ANSI dump” for chat.

## Config & state

- Config: `~/.config/keimling/config.json` (also model providers, keys, `context_limit`, `max_steps`, …)
- GUI prefs / connections: gui-prefs + `connections.json` (client book vs server bookmarks)
- Sessions / workspaces: under `~/.local/state/keimling/` (or `/data` in Docker)
- Auth for HTTP: `--access-secret` / `KEIMLING_SECRET`

Settings changes that must survive restart go through `persistAgentConfig` → `fileConfig` (not memory-only).

## Conventions

- **Binary size:** keep `./keimling` as small as practical. Prefer small deps (or none), subset/minified embedded assets, avoid shipping unused languages/themes/fonts, and don’t pull large frameworks into `webui/` or Go. Measure impact before adding embed payloads (`make build` / `make build-nogui`); strip/ldflags already used in release builds.
- **Language:** code, prompts, docs, and this file are English; user chat may be any language.
- **Go:** single `main` package in `src/`; keep changes minimal and local; pair behaviour with `_test.go` when practical.
- **Web UI:** vanilla JS/CSS (no bundler). Assets are embedded — rebuild the binary to ship UI changes; every KB in `webui/` lands in the binary.
- **Approval modes:** `ask` / `auto` / `yolo` — see `docs/agent-loop.md` and `policy.go`.
- **Busy input:** while the agent runs, GUI may **queue/steer** text (and voice→transcript); Interrupt stops the run.
- **Exec tool UI:** collapsed by default; expanded = Command / Raw / Filtered (filtered only when actually filtered).

## Requirements workflow

Product intent and acceptance criteria live in `requirements/`. Prefer updating those when behaviour changes meaningfully; `docs/` explains the system for humans.

## Don’t

- Don’t add a second UI framework or duplicate the agent loop for “remote vs local”.
- Don’t grow the binary casually (full highlight.js language packs, icon fonts, heavy WASM, fat Go modules) without a clear need and size check.
- Don’t use ad-hoc build/test commands outside `commands.md` once tooling is established.
- Don’t commit secrets (`.env`, key files, vault material).
- Don’t force-push `main` or skip hooks unless explicitly asked.

## Quick reads

1. [`README.md`](README.md) — run/flags  
2. [`docs/concept.md`](docs/concept.md) — vision  
3. [`docs/agent-loop.md`](docs/agent-loop.md) — REPL / modes  
4. [`requirements/technical/02-process-model.md`](requirements/technical/02-process-model.md) — host/workers  
5. [`commands.md`](commands.md) — how to build and test  
