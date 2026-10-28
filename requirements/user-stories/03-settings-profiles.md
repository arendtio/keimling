# User stories: Settings profiles

## US-20 — Save Keimling connections

**As** a user, **I want** to save several local/remote Keimling connections, **so that** I can switch without retyping URL and secret.

**Acceptance**

- Settings → Connection shows a list (local + remotes).
- I can add, edit, and delete remotes; local cannot be deleted.
- Continue uses the selected profile (local resume or remote navigate).
- Profiles persist in gui-prefs across launches.

## US-21 — Save model providers

**As** an operator, **I want** multiple model providers (base URL + key + default model), **so that** I can keep OpenAI and OpenRouter (etc.) side by side.

**Acceptance**

- Settings → Models lists providers; one is active.
- Saving updates the running agent and config; active provider mirrors legacy api fields.
- Behaviour and OpenAI services remain below the provider list in the same tab.
