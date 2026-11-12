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
- Prefill: Settings can request upstream `/models` (OpenRouter returns
  `context_length` + pricing; OpenAI-compatible often only ids) and fill the
  edit form — catalogs are not persisted wholesale. The request uses the URL
  and key in the open provider form. See REQ-PM-5.
- Compaction/context uses the **active model’s** `context_limit` when set;
  otherwise the global settings fallback.

## Budget / spend

- **Session spend:** accumulate billed tokens for the process/thread; estimate
  USD when the active model has pricing.
- **Provider remaining:** optional capability by `kind`:
  - OpenRouter: `GET {base}/key` → `limit_remaining` (remaining **per-key
    spending cap**, not account credit balance). Account credits need
    Management-Key `GET /credits` and are out of scope for the normal API key.
  - LiteLLM: `GET {base}/key/info` with the key as the bearer. The key’s `max_budget` minus its `spend` is the remaining amount. `GET /user/info` is the user, not the key.
  - Others: omit remaining.
- Fetch sparsely: on session load, when status becomes idle, manual refresh;
  server cache TTL ~90s.
- Chat label for OpenRouter: `key left $…` so it is not confused with credits.

## UI

- Settings → Models: provider list; editor for provider + nested model list.
- Chat chrome: compact line for session estimate and provider remaining when known.

## REQ-PM-1 Startup does not require an OpenAI key

- **Bin:** explicit requirement
- **Statement:** Keimling shall start when no OpenAI API key is available. A file `~/.credentials_openai` shall be used to create an OpenAI provider only when no model provider is configured. When any model provider is already configured, that file shall not add or restore an OpenAI provider.
- **Origin:** User, 2026-11-02 (quote: "I want it to not require such an API key" and "if .credentials_openai exists, it can use that key to create an openai provider automatically, but if the file does not exist it should start normally" and "if no other provider is present it might restore the openai provider again. If other providers are configured openai should not be restored, so basically just run that method if no provider is configured").
- **Rationale:** The user chooses which services receive the data. A missing OpenAI key must not block a local or third-party provider.
- **Fit criterion:** With no key file and no providers, the process starts. With the key file and no providers, an OpenAI provider exists after start. With another provider already configured, start does not add an OpenAI provider even when the key file exists.
- **Priority:** shall
- **Status:** confirmed

## REQ-PM-2 No provider still opens the app

- **Bin:** explicit requirement
- **Statement:** When no model provider is configured, the app shall open and explain that at least one model provider is required. The user shall be able to add one or more providers, or skip. After skip, the app shall stay usable in a limited way, and the warning shall remain visible.
- **Origin:** User, 2026-11-02 (quote: "If not provider at all is configured, the app should open with a small dialog explaining that at least one model provider is required and offer the user the possibility to add one or more providers (it should also be possible to skip that step with a warning)" and "yes usable but very limited due to the skip").
- **Rationale:** First-run setup should not be a dead end, and skipping must not look like a provider exists.
- **Fit criterion:** A start with zero providers shows the explanation, an add action, and a skip action. Skip closes the dialog, leaves chat unable to call a model, and keeps the warning visible. Adding a provider clears that warning.
- **Priority:** shall
- **Status:** confirmed

## REQ-PM-3 Voice and web search choose their own service

- **Bin:** explicit requirement
- **Statement:** Voice transcription and web search shall each have their own chosen service in Settings. Compaction, the autopilot supervisor, the delegate profiles, and auto-mode approval review shall keep using the normal model provider and shall not have separate provider choices.
- **Origin:** User, 2026-11-02 (quote: "for normal model tasks like compaction, the autopilot supervisor, the three delegate profiles, and the auto-mode approval review we do not need separat options" and "For voice and websearch we need such options, even though websearch is not necessarily a model provider, but might be a provider of its own" and "The goal is to enable a completely OpenAI free usage").
- **Rationale:** Those two jobs can send data to a different service than chat. The other jobs are ordinary model calls and already follow the configured model provider.
- **Fit criterion:** Settings can set voice to a provider and model that speak transcription, and web search to Brave or a URL, or leave either unset. Unset means that feature does not call a service. A chat, compaction, autopilot, delegate, and auto-review call goes to the normal model provider. With only a non-OpenAI model provider configured, those model calls do not contact OpenAI.
- **Priority:** shall
- **Status:** confirmed

## REQ-PM-4 Web search is Brave or a URL

- **Bin:** explicit requirement
- **Statement:** Web search shall be choosable as a hosted API such as Brave, or as a URL the user supplies. It is not required to be a model provider.
- **Origin:** User, 2026-11-02 (quote: "maybe a hosted API like brave, but an URL should be possible too").
- **Rationale:** Search sends the query to whoever the user picks, including a host they run themselves.
- **Fit criterion:** Settings can select Brave with an API key, or a URL. A search then goes to that service. With search left off, a search sends nothing.
- **Priority:** shall
- **Status:** confirmed

## REQ-PM-5 Catalog prefill uses the open form

- **Bin:** explicit requirement
- **Statement:** Loading the model list for a provider form shall use the URL shown in that form and the key typed in that form. A blank key field shall keep the key already saved for that same provider. It shall not use a key saved for a different provider.
- **Origin:** User, 2026-11-03 (quote: "using the key from an existing provider is wrong. It should always use the key from the current form together with the url from the current form").
- **Rationale:** The form is adding or editing one provider. A key from another provider makes the catalog call fail against the URL just entered, and shows that other key in the error.
- **Fit criterion:** With an OpenRouter provider already saved, opening a new provider form, entering an OpenAI URL and an OpenAI key, and loading the catalog sends that URL and that key. The stored OpenRouter key is not sent. Reloading the catalog for a saved provider with the key field left blank sends that provider's saved key and the URL currently in the form.
- **Priority:** shall
- **Status:** confirmed

## REQ-PM-6 Voice model is typed, not picked from chat models

- **Bin:** explicit requirement
- **Statement:** The voice model shall be a text the user types. Choosing a voice provider shall not replace that text with the provider's chat models. The field shall say that gpt-transcribe is a sensible model for OpenAI.
- **Origin:** User, 2026-11-03 (quote: "wenn man beispielsweise OpenAI als Provider bei Voice auswählt, dann bietet er die ganz normalen Modelle an" and "das Transcribe-Modell, was wir vorher hatten, das kann man nicht mehr auswählen" and "warum wir es eigentlich als Dropdown implementieren und nicht als Freitextfeld mit 'n Hinweistext").
- **Rationale:** A provider's saved models are for chat. Transcription uses a different model id, and Keimling does not know which chat models accept audio.
- **Fit criterion:** With an OpenAI provider whose only saved model is a chat model, and voice model gpt-transcribe, Settings shows gpt-transcribe in a text field and names gpt-transcribe in the hint. Switching the voice provider leaves that text in place.
- **Priority:** shall
- **Status:** confirmed
