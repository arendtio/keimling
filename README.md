# Keimling

Keimling is an experimental interactive Linux agent written in Go. It uses the OpenAI Responses API and provides local tools, interactive approval modes, persistent sessions, optional web search, audio input, and context compaction.

## Requirements

- Linux and Go 1.21 or newer to build
- An OpenAI API key, supplied through `OPENAI_API_KEY` or `~/.credentials_openai` (mode `600`)
- `arecord` (for example, from `alsa-utils`) for voice recording

## Build and test

The Go module lives in [`src/`](src). `make` builds the binary into the repository root:

```sh
make            # build ./keimling
make test       # run the test suite
make check      # gofmt, go vet, and the tests
make clean      # remove the binary
./keimling --help
```

Without make, use `cd src && go build -trimpath -o ../keimling .`.

## Run

```sh
export OPENAI_API_KEY=...
make run ARGS="--mode ask --model gpt-6-luna"
```

`make run` builds first, then starts `./keimling` with the given arguments.

The default mode is `auto`; use `--mode ask` for confirmation before tool actions. `--mode yolo` skips confirmations and should only be used in a suitably isolated environment. Configuration is read from `~/.config/keimling/config.json` by default; sessions are stored under `~/.local/state/keimling/`. Both locations can be overridden with command-line options.

## Command-line options

Run `./keimling --help` for the generated list; command-line flags override values from the configuration file.

| Option | Default | Purpose |
| --- | --- | --- |
| `--mode` | `auto` | Approval mode: `ask`, `auto`, or `yolo` |
| `--model` | `gpt-6-luna` | Model for the main conversation |
| `--reasoning` | `high` | Reasoning effort: `low`, `medium`, `high`, `xhigh`, `max` |
| `--config` | `~/.config/keimling/config.json` | JSON configuration file |
| `--api-key-file` | from `OPENAI_API_KEY` / `~/.credentials_openai` | Key file for the model provider |
| `--base-url` | OpenAI | Responses-compatible endpoint, e.g. `http://localhost:8080/v1` |
| `--api-mode` | `responses` | Only `responses` is supported |
| `--web` | `true` | Enable OpenAI web search |
| `--check-model` | `true` | Validate the model against `/models` at startup |
| `--session` | `~/.local/state/keimling/session.jsonl` | JSONL session file |
| `--resume` | `true` | Resume the last session |
| `--max-steps` | `200` | Maximum API rounds per user task |
| `--context-limit` | `128000` | Context token budget for compaction |
| `--compaction-model` | main model | Model used for context compaction |
| `--review-model` | `gpt-6-luna` | Model for automatic approval review |
| `--approval-policy` | none | Path to additional approval policy text |
| `--autopilot` | `false` | Enable the autonomous task supervisor |
| `--autopilot-model` | `gpt-6-luna` | Supervisor model |
| `--autopilot-reasoning` | `high` | Supervisor reasoning effort |
| `--delegate-fast-model` | `gpt-6-luna` | Fast delegate model |
| `--delegate-balanced-model` | `gpt-6.1-sol` | Balanced delegate model |
| `--delegate-deep-model` | `gpt-6.1-sol` | Deep delegate model |
| `--voice-model` | `gpt-transcribe` | Transcription model |
| `--disable-external-services` | `false` | Disable OpenAI web search and transcription |
| `--openai-api-key-file` | none | Separate OpenAI key file for web search and transcription |

If the model cannot be found in the provider catalog, startup exits with status 2; other check failures only warn.

## Interactive commands

| Command | Purpose |
| --- | --- |
| `/help` | Show help |
| `/status` | Show model, context estimate, tokens, and settings |
| `/mode` | Show the current approval mode |
| `/mode ask\|auto\|yolo` | Switch approval mode |
| `/compact` | Run semantic context compaction now |
| `/compact <text>` | Compaction with extra guidance |
| `/newchat` | Archive the conversation and start fresh |
| `/stop` | Cancel current work without exiting |
| `/resume`, `/continue` | Resume an unfinished task |
| `/continue-new-model` | Resume with portable history (no encrypted thinking) |
| `/autopilot on\|off` | Enable or disable task supervision |
| `/voice` | Record audio; the transcript becomes an editable draft |
| `/exit`, `/quit` | Quit |

During a task, type a message to steer the next step. Key bindings: `Ctrl+G` toggles autopilot while typing, `F1`/space starts and stops recording on an empty prompt, arrow keys move the cursor and recall history, `Home`/`End` jump within the line, and `Tab` completes slash commands.

## Configuration file

```json
{
  "api": { "base_url": "http://localhost:8080/v1", "model": "local-model", "mode": "ask", "key_file": "/path/to/key" },
  "openai": { "key_file": "/path/to/openai-key" },
  "features": { "web_search": true, "disable_external_services": false }
}
```

For custom providers, `openai.key_file` holds the separate OpenAI credentials used for web search and transcription; the model provider never receives that key.

## Patch format

`apply_patch` edits files without needing exact line numbers. A patch looks like this:

```text
--- a/src/main.go
+++ b/src/main.go
@@ ~120
 context line (leading space)
-removed line
+added line
```

- `@@ ~120` searches within 50 lines of line 120. A bare `@@` searches forward from the previous hunk. Legacy `@@ -1,2 +1,2 @@` headers are accepted; their counts only help parse empty context lines, so stale counts do not fail a patch.
- The nearest context match wins; only equally close matches are rejected as ambiguous. Exact matches win over trailing-whitespace and indentation relaxations, and the result reports which relaxation was used. Context lines are copied from the file, so indentation and line endings are preserved.
- When a context match fails, the error contains the actual file lines and tells the model to re-read them.
- `--- /dev/null` creates a file and fails if the target already exists; `+++ /dev/null` deletes one and must quote the removed lines exactly. Pasted `git diff` preambles are skipped. All files are validated before anything is written, and failures roll back already written and deleted files.

For small, targeted edits `replace_lines` with a hash is usually easier; `apply_patch` is meant for multi-line or multi-file changes.

## Safety

Keimling is experimental and is not a sandbox. In automatic modes, tools act with the permissions of the current user; shell commands can bypass restrictions applied to individual file tools. Review actions and outputs, and do not use it unattended for sensitive tasks.

See [LICENSE](LICENSE) for licensing information.
