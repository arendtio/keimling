# File edit tools

## Goal

Match the Claude Code / Cursor edit surface so models can use familiar exact
string replacement instead of unified diffs.

## Tools

| Tool | Purpose | Arguments |
| --- | --- | --- |
| `edit_file` | Targeted edit of an existing file | `path`, `old_string`, `new_string`, `replace_all` |
| `write_file` | Create or overwrite a whole file | `path`, `content` |
| `read_file` | Unchanged | `path`, `start_line`, `end_line` |

## Acceptance

- [x] `edit_file` replaces exactly one occurrence when `replace_all` is false; fails if zero or multiple matches.
- [x] `edit_file` with `replace_all` true replaces every occurrence (at least one required).
- [x] `old_string` must differ from `new_string`.
- [x] `write_file` writes atomically; creates parent directories as needed.
- [x] Paths outside the working directory or protected locations are refused.
- [x] `apply_patch` and `replace_lines` are no longer exposed to the model.
- [x] Auto mode may approve ordinary `edit_file` / `write_file` inside the cwd (same spirit as prior file edits).
