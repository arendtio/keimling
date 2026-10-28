# Feature: Settings profiles (connections & model providers)

## Intent

Settings keep two separate catalogs:

1. **Keimling connections** — how this UI reaches a Keimling process (local window vs remote server).
2. **Model providers** — how that process calls LLM APIs (OpenAI, OpenRouter, local OpenAI-compatible, …).

Only **one** connection and **one** model provider are active at a time. Switching Keimling connection always goes through Settings → Continue (reconnect / navigate); there is no seamless multi-server UI.

Per-chat model picking is out of scope for this step.

## Connection profiles

Stored in `gui-prefs.json` (UI/client):

- Always includes a non-deletable **local** profile (“This window”).
- Zero or more **remote** profiles: name, URL, access secret.
- `active_connection_id` selects which Continue uses.
- Legacy `connect` / `access_secret` are kept in sync with the active remote (or cleared for local) for CLI/`--connect` migration.

## Model providers

Stored in process `config.json`:

- List of providers: id, name, base_url, api_key (optional), model, reasoning.
- `active_model_provider_id` selects the live provider.
- Active provider is **mirrored** into legacy `api.base_url` / `api.key` / model / reasoning so workers and CLI keep working.
- On load, if the list is empty, synthesize one provider from legacy `api` fields.

## Settings UI

| Tab | Content |
| --- | --- |
| Connection | Profile list + edit; Appearance |
| Models | Provider list + edit; OpenAI services; Behaviour |
| Skills / MCP | Unchanged |

Save on Models applies providers + behaviour to the attached agent process.
