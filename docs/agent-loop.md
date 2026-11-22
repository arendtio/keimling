# Agent loop, terminal, and supervision

## Terminal REPL

- Keimling runs as a direct interactive REPL: readable status output, stable
  input editing, Unicode and ANSI support, multiline input, and reliable cursor
  handling.
- Tab completion for slash commands completes unique prefixes, extends the
  shared prefix of several matches, and lists suggestions.
- Model output streams over SSE. Tool calls run only after a complete, valid
  model response has arrived. Independent reads and non-interactive commands
  from that response run together. A call that changes files, takes the
  terminal, or needs a decision still runs on its own. MCP calls share a
  scheduler with a saved cap and one-at-a-time stdio; see
  [mcp-calls.md](mcp-calls.md).
- A `Working…` status line shows elapsed time and estimated tokens per second
  where measurable; the final line reports actual usage and average rate. Hidden
  reasoning tokens are never presented as measured live throughput.
- Interactive shell commands run through a PTY with exclusive terminal
  ownership: the REPL and subprocesses never read `stdin` at the same time.
  Cancellation and return to the prompt are reliable.
- Approval prompts never swallow keyboard input and default to `[Y/n]`.

## Slash commands

| Command | Purpose |
| --- | --- |
| `/help` | Show help |
| `/status` | Model, provider, review and compaction models, context usage, actual token usage, runtime information, autopilot state, session details |
| `/mode`, `/mode ask\|auto\|yolo` | Show or switch the approval mode |
| `/compact`, `/compact <text>` | Run semantic compaction now, optionally with extra guidance |
| `/stop` | Interrupt active work while keeping plan and session |
| `/newchat` | Archive the conversation and start fresh |
| `/resume`, `/continue` | Resume unfinished work |
| `/autopilot on\|off` | Enable or disable supervision |
| `/voice` | Record audio; the transcript becomes an editable draft |
| `/exit`, `/quit` | Quit |

`/resume` and `/continue` resume unfinished work **without** adding an artificial
user message to the history. After an interruption, potentially non-idempotent
tool operations are not blindly retried; an unknown execution state is recorded
explicitly.

A chat call sends an encrypted reasoning blob only to the provider and model
that produced it. Readable reasoning text still goes out. Switching back to
that model includes the blob again, unless that part of the active context was
already compacted away. When the switch leaves such a blob behind, the thread
shows that immediately. Historical Responses objects are normalised into valid
input items before resubmission — response-level metadata that the API rejects
is removed while tool-call relationships and valid reasoning content stay intact.

## Run outcomes in the chat

A run that stops, a failed tool, an autopilot decision, and a failed compaction
leave a line in the same transcript as messages and tool cards. Short lines stay
open. Long internal instructions stay collapsed. Reloading the thread shows them
again, and the next model call sees the same note. Token-rate progress stays on
the status line while the run is going. The CLI text stream is not copied into
the chat.

When the run goes idle, the Thinking rows and tool cards of that run close, so
what stays open is what the agent wrote to the user. Rows the user already had
open before the run keep their state, and a closed row opens again on a click.

A failed tool card is labeled Failed and stays collapsed. A tool interrupted by
Stop is labeled Stopped. The run itself adds the line `Stopped` or
`Run stopped: …`. In a chapter, those rows stay outside the “messages in
between” fold.

## Approval modes

| Mode | Behaviour |
| --- | --- |
| `ask` | Confirm write operations and system changes |
| `auto` | Routine development work runs directly; everything else is judged by a separate review model; critical actions stay protected |
| `yolo` | Skip approvals as configured; only for isolated environments |

In `auto` mode these run without asking, as long as they stay inside the
working directory and are not critical commands: reads, ordinary file edits
(`edit_file`, `write_file`), and formatters, linters, build, vet and
test runs such as `gofmt -w .`, `go vet ./...`, `go test ./...`, `golangci-lint run`,
`prettier --write .` or `git diff`. Redirections other than to `/dev/null`,
command substitution, pipes into shells, `cd ..`, deletions, `git commit`,
`git push`, `make install` and similar remain subject to review. The hard floor in
`forceApproval` still applies to every call before anything is auto-approved.

Before any approval question, Keimling prints what would happen: the tool line
names the affected path, and file edits show the old/new preview above the
`Approve? [Y/n]` prompt. Approval reasons are specific, for example
`modifying existing file` or `write outside working directory`.

A failed review call is retried twice with a short pause before it counts as an
outage. On an outage the call becomes a question to the user. A live realtime
call and a run with nobody to answer refuse instead, so an unattended run still
ends safely.

## Autopilot

- Autopilot monitors task progress, makes intermediate decisions, and can restart
  unfinished work without ever bypassing safety approvals.
- `--max-steps` (default 200) limits API rounds **per agent block**, not per
  whole autopilot task. After the limit, autopilot may start another block of the
  same size after supervision.
- Up to 200 supervision decisions are allowed in total across automatically
  started blocks; `/stop` stays effective at all times.
- Repeated identical results and repeated identical failures stop supervision
  and hand control back to the human.
- The supervisor is an AI, not the user. Its instructions can steer strategy but
  never grant approvals or authorize sensitive actions. The approval mode
  (`ask` / `auto` / `yolo`) is separate from the supervisor prompt.
- Settings can replace the whole supervisor prompt. A thread can replace that
  again. An empty text cannot be saved. Reset in Settings returns to the
  built-in prompt. Reset on the thread (Sidebar → Instructions) drops
  the thread text so later global edits apply again. A thread with no saved
  text follows the global prompt live.

## Thread chat model

The composer model is the main chat model for the open thread. A thread with no
saved choice uses the Settings default (active provider and its active model),
including after that default changes. Choosing any other saved model keeps that
pair when Settings change. Choosing the pair Settings currently use clears the
thread choice. Reasoning and context limit come from the model in effect; a
model with no context limit still uses the global fallback. A run already
started keeps the model it started with. The next sent message uses the new
choice. Compaction and the balanced and deep delegates stay on their Settings
models. Approval review, the output filter, autopilot supervision, the fast
delegate, and the external web search follow the auxiliary model or this chat
model (next section). A delegate call that looks at an attached image uses the
chat model that can see, and that call does not include the conversation.

## Auxiliary model for side tasks

Approval review, output filtering, autopilot supervision, the fast delegate, and
the external web search share one model: the *auxiliary model*. Settings → Side
tasks offers a saved provider's models plus the empty choice "Use the chat
model"; the value is stored as `auxiliary_model` in `config.json` and can also be
set with `--auxiliary-model`.

- Empty means each of those tasks uses the chat model of its run: the model the
  open thread chatted with when the run started, otherwise the Settings model.
  A chat on a cheap or local model therefore keeps its side tasks on that same
  endpoint.
- A saved value applies to the next side task without a restart and to later
  runs. It does not rewrite a chat call.
- Precedence is explicit flag first (`--review-model`, `--autopilot-model`,
  `--delegate-fast-model`), then `auxiliary_model`, then the chat model. One
  field serves all five tasks: the approval review has no separate saved model,
  and the only per-task overrides are those flags. The balanced and deep
  delegates keep `--delegate-balanced-model` and `--delegate-deep-model`.
- A saved auxiliary model that no configured provider offers never stops
  startup or a run: the task falls back to the chat model and the transcript
  says so once.
- Agent instances and workspace threads read the same setting from
  `config.json`; a worker has no flags, so its side tasks use the auxiliary
  model or that worker's chat model.

## Peer messages

An agent talks to another agent or to a workspace thread with `message_agent`.
The text lands as a `keimling_peer` item in the target's journal. When the
target is the thread this process runs and no run is in flight, the stored peer
message starts a run with the text `[From <name>] <body>`, so the run and the
transcript show a peer message and not user text. A busy thread stores the
message for the run already in flight and starts nothing. A message for another
thread is stored only; the process neither switches to that thread nor starts
it. A peer message does not wake the attached thread a second time until the
user writes in it, so two agents cannot answer each other forever; the user's
next message re-arms the wake. A wake that cannot start leaves the stored
message in place and does not report an error to the sender, so the sender does
not retry.

## Delegation

`delegate` solves one bounded, self-contained reasoning task with an independent
model that has no tools and no access to the main conversation. It is meant for
isolated reviews, extraction, synthesis, and second opinions. Results are
unverified evidence, not approvals, and no secrets are passed to it. An attached
image stays as a path in the conversation. Interpreting it is a delegate call
with that path and the user's question in the user's words, plus earlier
conversation that question depends on. The picture is sent only on that call,
and the conversation keeps the text answer. Another question uses the same path
again.

## Starting an agent in the background

`spawn_agent` and `spawn_team` start instances without taking the view. The host
creates the instance and leaves its attached target as it was, so the user stays
in the chat they were reading. The new instance shows up in the sidebar and
opens on a click. The tool result says so as well, because the calling model
would otherwise describe a switch that did not happen. A task passed to the
start is queued on the new instance, and the host wake loop starts the worker
for it, so nothing waits for a click. Messaging the instance works the same
way: `message_agent` starts its worker when a line is waiting.
