# Tools and file editing

## Tool surface

Keimling deliberately ships few, language-agnostic tools: `exec`, `read_file`,
`replace_lines`, `apply_patch`, `update_plan`, `finish_task`, `search_history`,
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
range, UTF-8 only. The hash is the `expected_hash` for `replace_lines`.

## Replacing lines

`replace_lines` replaces a line range only when the hash of the current range
matches `expected_hash`; on mismatch the file stays untouched and must be
re-read. An optional `expected_file_hash` adds whole-file concurrency protection.
The write is atomic (temporary file in the same directory, then rename) and keeps
existing permissions.

## Patching files

`apply_patch` complements `replace_lines` for models that produce diffs better
than line ranges. It is meant for multi-line and multi-file changes.

### Format

```text
--- a/src/main.go
+++ b/src/main.go
@@ ~120
 context line (leading space)
-removed line
+added line
```

- A file section is a `--- ` line immediately followed by a `+++ ` line. The old
  path uses the `a/` prefix, the new path the `b/` prefix; both may be
  `/dev/null`.
- Each hunk starts with `@@ ~N`, `@@ -N,M +N,M @@`, or a bare `@@`.
- Lines inside a hunk start with a space (context), `-` (removal), or `+`
  (addition). A `\ No newline at end of file` marker applies to the previous line.
- Several file sections and several hunks per file are allowed in one patch.

### Line hints

`@@ ~200` marks an approximate target line and searches within a radius of 50
lines. Traditional unified-diff counts are accepted; they help parse empty
context lines but are not validated strictly, because models often get them
wrong. A bare `@@` searches forward from the previous hunk and takes the first
match.

### Matching rules

- The nearest context match wins. Only equally close matches are rejected as
  ambiguous, with the number of candidates in the error.
- Exact context wins; trailing-whitespace and indentation relaxations are
  fallbacks that are reported back to the model.
- When several matches exist and the nearest one is patched, the result says so.
- Context lines are copied from the file, so indentation and line endings are
  preserved. Added lines use the file's own line ending (LF or CRLF).
- A failed match reports the actual file lines around the hint, the closest
  partial match, and how to re-read the range.

### Creating and deleting

- `--- /dev/null` with `+++ b/path` creates a file. Creation accepts only added
  lines and fails with `file already exists` when the target is present.
- `--- a/path` with `+++ /dev/null` deletes a file. The removed lines must match
  the file exactly; no relaxed matching is applied.
- Pasted `git diff` preambles (`diff --git`, `index …`, `new file mode`, rename
  and binary markers) are skipped.

### Safety properties

- Paths must stay inside the working directory and may not touch protected
  locations such as `~/.ssh` or credentials files.
- Every target is validated before anything is written; failures roll back files
  that were already written or deleted, including their permissions.
- Multi-file writes are per-file atomic. They are not a filesystem-wide
  transaction, and the implementation says so instead of pretending otherwise.

## Plans and completion

`update_plan` keeps a short checkpoint plan with `pending`, `in_progress`, and
`done` steps. `finish_task` ends an autonomous task with `done` or `blocked`
and a concise summary; `blocked` must state the exact question a human has to
answer.

## Deliberately excluded

No `code_symbols`, `get_symbol`, or `replace_symbol` tools and no embedded
Tree-sitter parser: an experimental branch grew the binary roughly sevenfold.
Where language-aware analysis is missing, the agent uses `rg`/`grep` patterns.
Such capabilities may later live in an optional external MCP server. There is no
backward-compatibility requirement for obsolete editing tools such as
`edit_file`/`write_file`.

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
