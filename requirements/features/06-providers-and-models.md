# Feature: Providers, models, and budget display

## Intent

Separate **LLM providers** (endpoint + credentials + optional capabilities) from
**models** (API id, context window, reasoning default, optional pricing).
Surfaces provider budget and session spend in the chat UI without storing huge
remote catalogs.

## Provider

- Fields: `id`, `name`, `base_url`, `api_key`, `kind` (`auto` | `openai` |
  `openrouter` | `litellm` | `local`), `models[]`, `active_model_id`.
- Legacy single `model` / `reasoning` on the provider migrate into one model
  entry when `models` is empty.
- Active provider still mirrors into legacy `api.*` for CLI/workers.

## Models (per provider)

- Manually curated favorites: `id` (API model string), `name`, `context_limit`,
  `reasoning`, optional `prompt_price` / `completion_price` (USD per token).
- Prefill: Settings can `GET` upstream `/models` (OpenRouter returns
  `context_length` + pricing; OpenAI-compatible often only ids) and fill the
  edit form — catalogs are not persisted wholesale.
- Compaction/context uses the **active model’s** `context_limit` when set;
  otherwise the global settings fallback.

## Budget / spend

- **Session spend:** accumulate billed tokens for the process/thread; estimate
  USD when the active model has pricing.
- **Provider remaining:** optional capability by `kind`:
  - OpenRouter: `GET {base}/key` → `limit_remaining` (remaining **per-key
    spending cap**, not account credit balance). Account credits need
    Management-Key `GET /credits` and are out of scope for the normal API key.
  - LiteLLM: `GET {base}/user/info` (or compatible) → budget − spend.
  - Others: omit remaining.
- Fetch sparsely: on session load, when status becomes idle, manual refresh;
  server cache TTL ~90s.
- Chat label for OpenRouter: `key left $…` so it is not confused with credits.

## UI

- Settings → Models: provider list; editor for provider + nested model list.
- Chat chrome: compact line for session estimate and provider remaining when known.
