# Tools and file editing

## Tool surface

Keimling deliberately ships few, language-agnostic tools: `exec`, `read_file`,
`edit_file`, `write_file`, `update_plan`, `finish_task`, `search_history`,
`read_history`, `delegate`, and optionally `web_search`.

`exec` runs shell commands. The agent is expected to state the specific question
a command should answer through `output_focus`; the output filter answers that
question briefly and accurately while the raw output stays visible to the user.
Filtering preserves exact error codes, numbers, paths, and other decisive facts,
never invents anything, and is skipped for short outputs, `raw_output=true`, and
interactive commands.

Failed tools are rendered as errors (red `✗`), never as a green checkmark.

## Reading files

`read_file` reads a whole file (`start_line` and `end_line` both `0`) or an
**inclusive** line range such as 50–75, and returns a SHA-256 hash of the exact,
unmodified bytes of that range. Limits: 8 MiB per file, 64 KiB per selected
range, UTF-8 only.

## Editing files

`edit_file` performs exact string replacement (Claude Code / Cursor style):

- `path` — existing file inside the working directory
- `old_string` — exact bytes to find (including whitespace)
- `new_string` — replacement (must differ from `old_string`)
- `replace_all` — if false, require exactly one match; if true, replace every match

On zero or (when not `replace_all`) multiple matches the file is left untouched
and the model should re-read with more context. Writes are atomic (temp file +
rename) and keep permissions.

## Writing files

`write_file` creates or overwrites an entire file with `content`. Parent
directories are created when needed. Prefer `edit_file` for targeted changes.
Deletes go through `exec`.

## Safety properties

- Paths must stay inside the working directory and may not touch protected
  locations such as `~/.ssh` or credentials files.
- Failed edits never partially apply.

## Plans and completion

`update_plan` keeps a short checkpoint plan with `pending`, `in_progress`, and
`done` steps. `finish_task` ends an autonomous task with `done` or `blocked`
and a concise summary; `blocked` must state the exact question a human has to
answer.

## Deliberately excluded

No `code_symbols`, `get_symbol`, or `replace_symbol` tools and no embedded
Tree-sitter parser: an experimental branch grew the binary roughly sevenfold.
Where language-aware analysis is missing, the agent uses `rg`/`grep` patterns.
Such capabilities may later live in an optional external MCP server. Unified-diff
`apply_patch` and hash-gated `replace_lines` were removed in favour of the
industry-standard edit/write surface; there is no backward-compatibility
requirement for those names.

## Skills and project instructions (planned)

A general skill loader with progressive disclosure is planned: only names and
descriptions in context, full `SKILL.md` on demand via `load_skill`, and further
references only when needed. Conventional location `~/.config/keimling/skills/`,
plus project-local `./.keimling/skills/`. Skills may be embedded with
`go:embed`, but no `systematic-debugging` skill is bundled. Skill instructions
can never override approval or safety rules. Project instructions in `AGENTS.md`
are respected, including directory and project scopes, and skills must not
consume large amounts of system-prompt context.

None of this is implemented yet in the current source tree.
