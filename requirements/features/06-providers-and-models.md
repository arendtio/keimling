# Feature: Providers, models, and budget display

## Intent

Separate **LLM providers** (endpoint + credentials + optional capabilities) from
**models** (API id, context window, reasoning default, optional pricing).
Surfaces provider budget and session spend in the chat UI without storing huge
remote catalogs.

## Provider

- Fields: `id`, `name`, `base_url`, `api_key`, `kind` (`auto` | `openai` |
  `openrouter` | `litellm` | `opencode` | `local`), `models[]`,
  `active_model_id`.
- Legacy single `model` / `reasoning` on the provider migrate into one model
  entry when `models` is empty.
- Active provider still mirrors into legacy `api.*` for CLI/workers.

## Models (per provider)

- Manually curated favorites: `id` (API model string), `name`, `context_limit`,
  `reasoning`, optional `prompt_price` / `completion_price` (USD per token),
  optional `base_url` (model-only endpoint, which also selects the wire
  format; see Wire formats).
- Prefill: Settings can request upstream `/models` (OpenRouter returns
  `context_length` + pricing; OpenAI-compatible often only ids) and fill the
  edit form — catalogs are not persisted wholesale. An OpenRouter request asks
  for every output kind (`output_modalities=all`), the same list as the site's
  "all models" view. The bare OpenRouter list is text only and leaves out
  speech, transcription, image, and the other kinds. Suggestions belong to the
  provider selected in the model form; switching provider shows that
  provider’s last fetch, or none. OpenAI’s `/models` list has no modality
  filter and no capability fields. The request uses the URL
  and key in the open provider form. See REQ-PM-5. LiteLLM also reads
  `GET {root}/model_group/info` (the model groups the key may call, with
  context, price, and vision) and `GET {root}/model/info` (deployments with
  `model_info`, which adds audio input) with that same key. See REQ-PM-8.
- The form asks for USD per million tokens and stores USD per token. See
  REQ-PM-9.
- Compaction and context use the context length stored on the model in
  effect. There is no global context-length setting. See REQ-PM-14.

`kind` labels the vendor for the UI and for vendor-specific behaviour; the
base URL still decides the wire format (see below). `opencode` is detected from
an `opencode.ai` host and skips the LiteLLM management probes.

For an `opencode` provider the prefill also reads the Go catalog
(`{root}/inference/go/v1/models`) and marks every model the Go subscription
covers with `go` and its monthly allowance (`$100/mo` on the plain Go plan).
Go-only models are added to the list with their Go base URL. The marker is a
prefill hint, not stored state, and a Go model bills through the subscription
only when the base URL under `/inference/go/` is used.

## Wire formats

- A provider profile is one base URL plus key. The base URL picks the wire
  format, so a vendor that serves several APIs needs one provider per family
  (or one model-level override, below): a URL ending in `/chat/completions`
  talks Chat Completions, one containing `/anthropic/` (or ending `/messages`)
  talks Anthropic Messages, one containing `/google/` or
  `generativelanguage` talks Gemini `streamGenerateContent`, anything else is
  the OpenAI Responses API.
- A model may carry its own `base_url`, which wins over the provider's. This is
  how one provider (for example opencode.ai, whose families live under
  `/inference/openai/v1`, `/inference/anthropic/v1`, `/inference/google/v1beta`)
  gets models from more than one family. Catalog prefill fills it from the
  catalog's `api` map.
- Requests and streams are translated to and from the Responses item shape, so
  the agent loop, journal, and transcript do not change per format. Side tasks
  (compaction, summarization, filter, transcription) still speak Responses on
  the provider's own base URL, so a side model from another family needs its
  own provider profile.

## Budget / spend

- **Session spend:** accumulate billed tokens for the process/thread; estimate
  USD from the model this chat is using. Settings is the fallback when the
  chat has not chosen its own model.
- **Provider remaining:** optional capability by `kind`:
  - OpenRouter: `GET {base}/key` with the chat key → `limit_remaining`
    (that key's cap). `GET {base}/credits` with the provider's management
    key → account credits minus usage. See REQ-PM-7.
  - LiteLLM: `GET {base}/key/info` with the key as the bearer. The key’s `max_budget` minus its `spend` is the remaining amount. `GET /user/info` is the user, not the key.
  - opencode.ai: `GET {go root}/v1/usage` with the same key → one meter each
    for `rolling` (5 h), `weekly`, and `monthly`, with `status`, `percent` and
    `resetsAt`. Shown as "Go 30%" with the reset in the tooltip; a key without
    a Go subscription answers 403 and the line says so. There is no dollar
    balance endpoint (only `console/api/v1/models` and
    `console/api/v2/usage/export`, a CSV of past daily spend that needs a key
    with `Usage › Read`).
  - Others: omit remaining.
- Fetch sparsely: on session load, when status becomes idle, manual refresh;
  server cache TTL ~90s.
- Chat label for OpenRouter: `key left $…` so it is not confused with credits.

## UI

- Settings → Models: a provider list and a separate model list. A provider row and a model row each offer Edit and Remove. Remove asks by name before the provider, its key, and its models are dropped. The model list does not mark a model as the default; the main chat purpose is that choice. Prefill sits at the start of the model form, before the fields it fills.
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

The typed voice field is the transcription choice until REQ-PM-12. From 2026-11-25 a purpose points only at a saved model, and transcription stays off until such a model is saved and chosen.

## REQ-PM-7 The account total and the key limit are both visible

- **Statement:** For an OpenRouter provider, Keimling shall show the account remaining amount and, when the chat key has a limit, that key's remaining amount. The account amount shall use a management key stored for that provider. The chat key shall keep calling the models.
- **Origin:** User, 2026-11-20 (quote: "bei dem Provider Open Router, dort kann ich im Moment meine Budgets nicht richtig sehen" and "dafür 'n anderer Key-Typ notwendig" and "dass ich das Budget in Summe im Blick behalte"). Same day, the user chose both the account total and the key limit when the key has one. Later the same day (quote: "wenn ich dann wieder in die Settings geh, dann steht da auch nicht, dass der Key irgendwie gesetzt ist und ich nicht ihn editieren sollte. Es ist einfach wieder leer" and "also ich sehe nur ein DeepSeek: DeepSeek V4.1 Flash 1017 · yolo · session 11057 tok").
- **Rationale:** The chat key's own cap is often empty. The number they watch is the account total, and that read needs a different key. The line sits beside the model this chat is using, so a Settings default on another provider must not hide it.
- **Fit criterion:** Saving a management key stores it. Opening that provider again says the key is set and does not show the value. Leaving the field blank keeps the stored key. The chat line uses the provider of the model this chat is using. With a management key, that line shows account credits minus usage, including when the chat-key budget call fails. With a chat key that has a limit, the line also shows that limit's remainder. The session amount uses that model's prices. The management key is not sent on a model call. A provider that is not OpenRouter has no management key field.
- **Priority:** shall
- **Status:** confirmed

## REQ-PM-8 Prefill fills context, price, and senses when the catalog reports them

- **Statement:** When a provider catalog reports a context length, a prompt price, a completion price, or image or audio input, prefill shall put those into the model form. A field the catalog does not report shall stay as the form already has it. For a LiteLLM provider, that richer info shall be read with the same key as the model list, from `GET /model_group/info` and `GET /model/info`; this applies even when the proxy URL was not recognised as LiteLLM (for example `http://127.0.0.1:4000/v1` with kind auto), and a LiteLLM endpoint that is absent or refused shall not stop the plain `/v1/models` list from loading.
- **Origin:** User, 2026-11-20 (quote: "Modelle habe, die eigentlich Vision Capability haben … über den Light LLM Proxy mit Keimling aber anscheinend nicht erkannt" and "wenn wir Modelle abrufen … nicht die Kontextlänge und die Kosten unbedingt mit abruft"). Same day, the user said to use the normal key for LiteLLM when that key can read the info. 2026-11-25 (quote: "wenn ich auf den Prefill from API-Button drücke, dann … das Model trotzdem nicht" and "wenn ich auf der Website alle Modelle auswähle … dann findet er das" and "warum das bei unserem Prefill from API nicht auftaucht").
- **Rationale:** The model id is only half of the form. Context, price, and seeing are what they otherwise look up by hand, and other clients already get them from the proxy. `/model_group/info` is the endpoint the proxy documents for the models a key actually calls, and `/v1/models` on LiteLLM carries no price and no vision flag.
- **Fit criterion:** An OpenRouter catalog entry with context, prices, and image input fills those form fields. The OpenRouter catalog request asks for every output kind, so a speech or transcription model that the text-only list omits is in the suggestions, with that kind beside the name. A provider that is not OpenRouter is asked for `/models` with no modality filter. A LiteLLM `/model_group/info` row with a float `max_input_tokens`, token prices, a description, and `supports_vision` fills context, both prices, the display name, and seeing. A `/model/info` row with `supports_audio_input` fills hearing. When both management routes are refused, the `/v1/models` ids still load and vision stays off. A catalog entry without a price leaves the price field unchanged. An OpenRouter provider probes neither LiteLLM route.
- **Priority:** shall
- **Status:** confirmed

## REQ-PM-9 Prices are entered per million tokens

- **Statement:** The model form shall ask for USD per million tokens. A price already saved as USD per token shall stay valid and shall appear in the form as USD per million. Saving shall keep storing USD per token. A blank price shall stay blank.
- **Origin:** User, 2026-11-20 (quote: "ich die immer in Kosten per Token angeben muss. Die meisten haben aber immer Kosten per Million Token ausgewiesen" and "alles, wo ich das jetzt schon mal in irgendwelchen Config-Dateien hinterlegt hab, das auch … weiter valide bleibt" and "im Formular in Zukunft eben … Dollar per Million Token").
- **Rationale:** Vendors publish a dollar or three per million. Typing six leading zeros is where the number goes wrong.
- **Fit criterion:** A stored prompt price of 0.000002 is shown as 2. Saving 3 stores 0.000003. Opening the form and leaving it does not rewrite the config. An empty price stays empty.
- **Priority:** shall
- **Status:** confirmed

## REQ-PM-10 One auxiliary model for the side tasks

- **Statement:** Settings shall offer one "Auxiliary model" choice, saved as `auxiliary_model` in `config.json` and settable with `--auxiliary-model`. The approval review, the output filter, autopilot supervision, the fast delegate, and the external web search shall use it. An empty value shall make each of those tasks use the chat model of the run (the model the thread chatted with when the run started, else the Settings model). A set value shall apply to the next side task without a restart and to later runs. An explicit flag (`--review-model`, `--autopilot-model`, `--delegate-fast-model`) shall win over it, and the balanced and deep delegates shall keep their own flags. The empty choice shall be labelled "Use the chat model".
- **Origin:** User, 2026-11-22 (quote: "In den Model-Settings soll man ein Modell auswählen können, das für die Nebenaufgaben benutzt wird. Ist keins ausgewählt (leer), nimmt jede dieser Aufgaben das normale Modell des Chats").
- **Rationale:** A chat on a cheap or local model sent its reviews, filters, supervision, and searches to a different provider, so the cost and the data path of a run did not follow its own model.
- **Fit criterion:** With an empty value, the filter, review, autopilot, fast delegate, and web search request the chat model. After saving a model in Settings, the next side task names it in its request and the value survives a restart. `--review-model` still overrides it. A saved model that no provider offers falls back to the chat model, says so once, and never stops startup.
- **Priority:** shall
- **Status:** confirmed

REQ-PM-12 replaces this single choice. Review, the output filter, and autopilot supervision become the one side-task purpose. Delegates use the main chat model and do not use the side-task model. Web search stays the service in REQ-PM-4.

## REQ-PM-11 Each settings section saves on its own

- **Bin:** explicit requirement
- **Statement:** Settings for providers, models, and purpose choices shall each save on their own. Saving a model shall store that model. Saving a provider shall store that provider. Saving a purpose choice or another setting in that dialog shall store that choice. One of those saves shall leave the others as they were.
- **Origin:** User, 2026-11-25 (quote: "wenn ich ein Modell hinzufüge, dann muss ich ein Modell speichern. Fertig. Wenn ich einen Provider hinzufüge oder bearbeite, muss ich den Provider speichern. Fertig" and "wenn ich andere Einstellungen hab, dann muss ich eben die anderen Einstellungen irgendwie speichern" and "jeden Abschnitt einzeln speichern" and "sehr homogen … in den Benutzermustern, die da zum Speichern verwendet werden"). Same day, later (quote: "man erst einen Provider speichert und dann darin die Models anlegt, dann hat man wieder dieses Dreifach-Speichern" and "einen Speichern-Button für einen Provider" and "extra Abschnitte, in denen Models anlegen" and "wenn ich das Model anlege, auch den richtigen Provider dazu geben" and "den Speichern Button für den Abschnitt").
- **Rationale:** Today a model is only stored when the provider is saved afterward, and the purpose fields need a third save. The person cannot tell which button kept the change.
- **Fit criterion:** Adding a model and using that section's save leaves the model in place after the dialog is closed and reopened, without a second save. The model section is separate from the provider section, and saving a model includes choosing which saved provider it belongs to. Editing a provider and using that section's save leaves the provider in place the same way, without requiring the button at the bottom of Settings. Changing a purpose choice and using that section's save leaves the choice in place and does not drop a model or a provider saved earlier. Each section uses the same save control pattern.
- **Priority:** shall
- **Status:** confirmed

## REQ-PM-12 A purpose names the model that does that job

- **Bin:** explicit requirement
- **Statement:** Settings shall list purposes, and for each purpose the user shall choose a saved model. A purpose shall not accept a model id that is not saved. The choice shall offer only models that have the capability that purpose needs. Image analysis shall offer only models that can see images. A live call shall offer only models that speak the Realtime API. Transcription shall offer only models saved for transcription, and transcription shall stay off until one is chosen. Speech synthesis shall offer only models saved as able to speak. Approval review, the output filter, and autopilot supervision shall be one side-task purpose and shall share one model. A delegate, including the fast delegate, shall use the main chat model. That main chat model shall itself be a purpose. The model list shall not offer a separate control that marks a model as the default. Web search shall stay a service of its own, as REQ-PM-4 says.
- **Origin:** User, 2026-11-25 (quote: "für die verschiedenen Zwecke immer festlegen kann, welches Modell genau dafür verwendet werden soll" and "nur Modelle auswählbar sein, die auch Vision Capability haben" and "Websearch, wobei das ist eher 'n Service"). Same day, later (quote: "Review, Filter, Autopilot sind … alle so in etwa der gleiche Service" and "bei den Delegate-Varianten immer das Hauptmodell" and "das Hauptmodel könnte ein Service sein" and "Zwecke sollten eigentlich immer auf eine gespeicherte Model-ID gehen" and "für Transkription muss noch 'n Model angelegt werden, also bevor man das nutzen kann" and "ein Purpose könnte auch noch sein … Sprachsynthese"). Same day, later (quote: "das Default-Setzen komplett wegnehmen, weil das macht dann aus meiner Sicht ja jetzt ein Main-Chat-Modell" and "bei Model gibt's ja diese Remove-Button gleich auf der Übersicht. Das hätte ich gerne bei den Providern auch" and "Prefill from API … ganz oben rechts … das, was ich ganz am Anfang machen muss").
- **Rationale:** Several models can see, transcribe, or speak. One shared side-task field does not say which job it is for. A transcription id that is only a leftover key never appears in the list the user can check. A delegate is a subagent of the chat and should use the model the chat is using.
- **Fit criterion:** The purpose list shows a model choice for the main chat model, image analysis, transcription, speech synthesis, a live call, and side tasks. Each choice is one list of saved models across every provider, limited to models with that capability. With no transcription model saved, transcription sends nothing. Saving a transcription model and choosing it makes the next transcription call that model. Review, the output filter, and autopilot supervision call the side-task model. A fast delegate and a normal delegate call the main chat model. The model list has Edit and Remove and no Default control. A provider row has Edit and Remove in the same places; confirming Remove names that provider and drops it together with its models. Opening Add model or Edit model shows Prefill from API at the top of that form, before the fields. Web search is still Brave, a URL, or off. The choices survive a restart.
- **Priority:** shall
- **Status:** confirmed

Speech synthesis is only the purpose choice for now. How spoken output is produced stays open. REQ-PM-6's typed voice field remains until this list is what transcription uses.

## REQ-PM-14 No global context length

- **Bin:** explicit requirement
- **Statement:** Settings shall not offer a global context length. The context length of a call shall be the one stored on the model that call uses. A model with no stored context length shall not fall back to a global number.
- **Origin:** User, 2026-11-25 (quote: "Die globale Kontextlänge kann aus meiner Sicht weg" and "das wird ja eigentlich vom Model immer jeweils festgelegt" and "die könnte höchstens genutzt werden, um irgendwie 'ne kleinere Kontextlänge zu entforcen" and "auch das halte ich eigentlich für Schwachsinn" and "die globale Kontextlänge kann einfach weg").
- **Rationale:** The length belongs to the model. A global field, even as an optional tighter cap, is a second place to set the same thing.
- **Fit criterion:** The Models settings have no context-length field. A chat whose model stores 128000 uses 128000. Saving settings does not write a global context length. A model with no stored length does not receive one from settings.
- **Priority:** shall
- **Status:** confirmed

## REQ-PM-13 A model can carry extra instructions

- **Bin:** explicit requirement
- **Statement:** Each saved model shall be able to carry instructions that are added when that model answers. Empty instructions shall add nothing. The extra text shall not replace the instructions the agent, the autopilot, or the product already send.
- **Origin:** User, 2026-11-25 (quote: "pro Model eigene Instruktionen mitgeben" and "bei dem und dem Model, da möchte ich gern nochmal die und die Instruktionen im System Prompt mitgeben" and "manche Models zum Beispiel ziemlich verbose sind, und dann möchte ich sagen, halte ich bitte sehr kurz").
- **Rationale:** One model talks too much. The correction belongs to that model, and the other models keep the stance they already have.
- **Fit criterion:** A model saved with the sentence "Answer in one sentence" receives that sentence on its next reply, and a second model with empty instructions does not. The agent's own instructions are still sent. Clearing the field and saving leaves the next reply without that sentence.
- **Priority:** shall
- **Status:** confirmed
