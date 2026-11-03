# Feature: Settings profiles (connections & model providers)

## Intent

Settings keep separate catalogs for **how the UI reaches Keimling** and **how Keimling calls LLM APIs**.

Per-thread model picking is [REQ-TM-1](08-thread-model.md). This file only covers the Settings catalogs and the global default.

## Connection catalogs (hybrid)

Three layers in Settings → Connection:

1. **Currently connected** — this page’s Keimling (session status). Continue resumes here.
2. **My connections (this machine)** — client address book in `~/.config/keimling/connections.json` (same file for GUI + CLI). Editable; Continue navigates/reloads to that endpoint.
3. **On this server** — profiles stored on the **connected process** (`gui-prefs.json` `connections[]`). Editable; Continue may navigate to them. **Adopt into the client book only when the user actually connects** (not bulk copy).

Switching always uses the existing Continue path (navigate or local resume/relaunch). No parallel multi-server UI.

### Client book (`connections.json`)

- Path: next to `config.json` under the config directory (default `~/.config/keimling/connections.json`).
- Fields: `id`, `name`, `kind` (`remote` for saved servers; optional local desktop target via UI action), `url`, `access_secret`, plus `active_id` for last chosen remote.
- Desktop WebView reads/writes this file via native bridge even when the page origin is a remote agent.
- Browser without bridge: `localStorage` fallback (not shared with CLI).
- Local agent HTTP: `GET`/`PUT` `/api/client-connections` against the process whose config dir owns the file.
- CLI: `--connect <url|name|id>` resolves names/ids from this file; legacy `gui-prefs` `connect` still seeds/migrates.

### Server book (`gui-prefs`)

- Process-local `connections[]` + `active_connection_id`, with legacy `connect` / `access_secret` mirrored for compatibility.

## Model providers

Stored in process `config.json`:

- List of providers: id, name, base_url, api_key (optional), model, reasoning.
- `active_model_provider_id` selects the live provider.
- Active provider is **mirrored** into legacy `api.base_url` / `api.key` / model so workers and CLI keep working.
- On load, if the list is empty, synthesize one provider from legacy `api` fields.

## Settings UI

| Tab | Content |
| --- | --- |
| Connection | Current + client book + server book; Appearance |
| Models | Provider list + edit; OpenAI services; Behaviour |
| Skills / MCP | Unchanged |

Save on Models applies providers + behaviour to the attached agent process.

## REQ-SET-1 Settings can be closed from the corner, and saving leaves the dialog

- **Bin:** explicit requirement
- **Statement:** When Settings is open, a close control in the top-right of the dialog shall dismiss it without saving. A successful save of the model settings shall close the dialog. A failed save shall leave it open and show the error.
- **Origin:** User, 2026-11-04 (quote: "wenn ich die Settings öffne, ich da eigentlich nur über den Cancel-Button rauskomme" and "irgendwo in der Region oben rechts noch 'n Button zum Schließen" and "wenn man auf Speichern klickt, er den Dialog nicht schließt").
- **Rationale:** The dialog is a temporary layer over the chat. Leaving it should be visible at the corner, and a finished save should return to the chat.
- **Fit criterion:** Opening Settings shows Close at the top right. Close returns to the chat and does not write model settings. Save model settings returns to the chat after the server accepts them. A rejected save stays on Settings and shows the error.
- **Priority:** shall
- **Status:** confirmed
