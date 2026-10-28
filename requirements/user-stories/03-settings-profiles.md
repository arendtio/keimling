# User stories: Settings profiles

## US-20 — Hybrid Keimling connections

**As** a user (GUI or CLI), **I want** a machine-local connection book plus optional server-side bookmarks, **so that** remote Settings do not pretend the server’s “local” profile is my desktop.

**Acceptance**

- Settings → Connection shows **Currently connected**, **My connections (this machine)**, and **On this server**.
- Client book persists in `~/.config/keimling/connections.json` (CLI can resolve `--connect` by name/id).
- Server book remains editable on the process; connecting to a server entry adopts/updates the client book.
- Continue still navigates or resumes (reload), as today.
- Wording never calls the current remote session “Local desktop agent”.

## US-21 — Save model providers

**As** an operator, **I want** multiple model providers (base URL + key + default model), **so that** I can keep OpenAI and OpenRouter (etc.) side by side.

**Acceptance**

- Settings → Models lists providers; one is active.
- Saving updates the running agent and config; active provider mirrors legacy api fields.
- Behaviour and OpenAI services remain below the provider list in the same tab.
