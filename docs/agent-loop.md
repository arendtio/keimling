# Agent loop, terminal, and supervision

## Terminal REPL

- Keimling runs as a direct interactive REPL: readable status output, stable
  input editing, Unicode and ANSI support, multiline input, and reliable cursor
  handling.
- Tab completion for slash commands completes unique prefixes, extends the
  shared prefix of several matches, and lists suggestions.
- Model output streams over SSE. Tool calls run only after a complete, valid
  model response has arrived.
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
| `/continue-new-model` | Resume unfinished work with provider-compatible context |
| `/autopilot on\|off` | Enable or disable supervision |
| `/voice` | Record audio; the transcript becomes an editable draft |
| `/exit`, `/quit` | Quit |

`/resume` and `/continue` resume unfinished work **without** adding an artificial
user message to the history. After an interruption, potentially non-idempotent
tool operations are not blindly retried; an unknown execution state is recorded
explicitly.

`/continue-new-model` resumes with provider-compatible context: old encrypted
thinking blocks are excluded from the new API input but kept in the original
journal. The original model does not have to be reachable. Historical Responses
objects are normalised into valid input items before resubmission — response-level
metadata that the API rejects is removed while tool-call relationships and valid
reasoning content stay intact.

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
  never grant approvals or authorize sensitive actions.

## Delegation

`delegate` solves one bounded, self-contained reasoning task with an independent
model that has no tools and no access to the main conversation. It is meant for
isolated reviews, extraction, synthesis, and second opinions. Results are
unverified evidence, not approvals, and no secrets are passed to it.
