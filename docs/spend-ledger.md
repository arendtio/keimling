# Spend ledger: cost and balance history

Keimling records what each model call cost and how a provider balance moved
over time. The records are for the operator: they make the cost of a long
thread visible without putting anything into the chat or the workspace.

## Where the data lives

Everything sits in the Keimling config directory, beside `config.json`:

- Linux and macOS: `~/.config/keimling/spend/` (or `$XDG_CONFIG_HOME/keimling/spend/`)
- Windows: `%AppData%\keimling\spend\`

There is one directory per process config: `--config /path/settings.json` still
writes to `~/.config/keimling/spend/`. Nothing is written into the Git
repository or the workspace, so a running chat cannot dirty `git status`.

## Files and format

Two prefixes, one JSONL file per month, one JSON object per line:

| File | Written when | Content |
| --- | --- | --- |
| `spend-YYYY-MM.jsonl` | A model response arrives | `time`, `provider`, `model`, `call`, `input_tokens`, `output_tokens`, `usd`, `priced` |
| `budget-YYYY-MM.jsonl` | A provider budget fetch really happens | `time`, `provider`, `kind`, `remaining_usd`, `limit_usd`, `usage_usd`, `account_remaining_usd` |

Example spend line:

```json
{"time":"2026-11-21T18:04:12Z","provider":"openrouter","model":"vendor/model","call":"chat","input_tokens":18342,"output_tokens":911,"usd":0.0412,"priced":true}
```

Rules:

- `time` is UTC in RFC3339. `call` is `chat`, `compaction`, or `filter`.
- `usd` comes from the model prices saved under Settings → Models
  (`prompt_price` / `completion_price`, USD per token). A model without saved
  prices still gets a record, with `"priced": false` and no `usd` field — there
  are no estimated numbers.
- Budget lines are written only on a real provider fetch: OpenRouter
  `/api/v1/key` and `/api/v1/credits`, LiteLLM `/key/info`. A cache hit within
  the 90 s TTL writes nothing, and a provider without a budget endpoint
  (OpenAI) writes nothing.
- Appends are one line under a process-wide mutex, so parallel writers do not
  interleave. A write error is logged and never fails the chat.
- The ledger is append-only. Compaction, delegate, autopilot, and background
  jobs use the same model clients, so their calls are recorded too.

## Viewing it

Settings → **Spend** shows three things for the selected window (30 days daily,
3 days hourly):

1. a bar chart of priced cost per hour or day,
2. a table per model with provider, calls, tokens, and cost,
3. the balance history from the budget snapshots, when there are any.

The view reads `GET /api/spend?granularity=day|hour&days=N`, which returns
hourly or daily buckets plus per-model rows. A single call is never shown; it
stays in the raw JSONL file.

## Deleting the history

Stop Keimling first if you want a clean cut, then delete the files:

```sh
rm ~/.config/keimling/spend/spend-*.jsonl
rm ~/.config/keimling/spend/budget-*.jsonl
```

Deleting the whole `spend/` directory resets cost and balance history. The
directory is recreated on the next model call.
