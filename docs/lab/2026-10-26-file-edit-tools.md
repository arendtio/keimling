# Lab: file edit tools across agents (2026-10-26)

## Observation

Most coding agents converge on exact string replacement, not unified diffs:

| Agent | Edit | Create / overwrite |
| --- | --- | --- |
| Claude Code | `Edit`: `file_path`, `old_string`, `new_string`, `replace_all?` | `Write`: `file_path`, `content` |
| Cursor | `StrReplace`: `path`, `old_string`/`new_string`, `replace_all?` | `Write`: `path`, `contents` |
| Anthropic text_editor | `str_replace`: `path`, `old_str`, `new_str` | `create` / `insert` |
| OpenAI Codex | still `apply_patch` (`*** Begin Patch` …) | same tool |

Keimling previously followed the Codex-style `apply_patch` plus hash-gated `replace_lines`. Models trained on Claude/Cursor tooling emit `old_string`/`new_string` more reliably than hunks.

## Decision

Replace `apply_patch` and `replace_lines` with:

- `edit_file` — exact unique match (or `replace_all`)
- `write_file` — create or overwrite whole file

Deletes stay via `exec` (no dedicated delete tool for now). Path safety (cwd + protected paths) unchanged.
