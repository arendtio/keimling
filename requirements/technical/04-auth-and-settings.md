# Auth, trust model, and agent settings

## Trust model (current)

Possession of the **access secret** (`--access-secret` / `KEIMLING_SECRET` / config `access_secret`) is full operator access to that agent process.

The authenticated client may:

- Send messages, approve tools, read session/history
- Change runtime agent settings (model, mode, base URL, behaviour knobs)
- Set or replace **model-provider** and **OpenAI external** API keys

There is no finer-grained role split yet (read-only viewer, settings-locked server, etc.). Treat the access secret like a root password for the agent.

## Dual credentials

Keimling keeps two credential scopes on purpose:

| Scope | Used for | Settings / config |
| --- | --- | --- |
| **Model provider** | Chat, tools, compaction, delegates, autopilot against `api.base_url` | `api_key` / `api.key` (+ optional `api.key_file`) |
| **OpenAI external** | Web search + voice transcription only | `openai_api_key` / `openai.key` (+ optional `openai.key_file`) |

The model provider never receives the OpenAI external key. When the model base URL is the default OpenAI endpoint and no separate OpenAI key is configured, the provider key may be reused for external services at startup.

## Settings API

`GET`/`PUT` `/api/settings` (access secret required):

- Returns current knobs plus `api_key_set` / `openai_api_key_set` (booleans only — never raw secrets)
- Also returns `model_providers` (each with `api_key_set`, never raw keys) and `active_model_provider_id`
- `PUT` may include write-only `api_key` and/or `openai_api_key` (omit or leave blank in UI = unchanged)
- `PUT` may replace `model_providers` and set `active_model_provider_id`; provider entries may include write-only `api_key`
- Successful `PUT` applies in-process and **persists** mapped fields into the process config file (`--config`), including inline keys when provided; the active provider is mirrored into legacy `api` / `model` fields
- Config file mode should be `0600` when written

## GUI prefs (connections)

`GET`/`PUT` `/api/gui-prefs` also carries `connections[]` and `active_connection_id` (see `features/04-settings-profiles.md`). Legacy `connect` / `access_secret` stay mirrored for compatibility.

## Startup precedence (model provider key)

1. CLI `--api-key-file` (explicit flag)
2. Config inline `api.key`
3. Config `api.key_file`
4. `OPENAI_API_KEY` / `OPENAI_API_KEY_FILE` / `~/.credentials_openai` (default OpenAI base only; custom base may run without a key)

OpenAI external key: CLI `--openai-api-key-file` → `openai.key` → `openai.key_file` → reuse provider key if same OpenAI default → else env/credentials (non-fatal if missing).
