# Security, integrity, and quality standards

## Safety rules

- Shell commands and file mutations, as well as external services, remain subject
  to the configured approval rules. Neither autopilot nor skills may bypass them.
- Non-idempotent operations with uncertain execution state are not restarted.
- Static and model-generated input is structurally validated before writes. On
  match failures or history conflicts, rejection wins over silent corruption.
- The original JSONL journal is preserved; checkpoints are written only after
  complete, successful validation.
- Terminal errors look like errors. A failed edit never shows a green checkmark.
- `edit_file` / `write_file` confine targets to the working directory, reject
  protected paths, and write atomically (temp file + rename), keeping permissions.

## Observability

- `/status` reports model and provider, review and compaction models, context
  usage, actual token usage, runtime information, autopilot status, and session
  details.
- Compaction ends with a useful report (before/after estimates, savings, node
  count and depth, treatment modes, section titles) rather than a bare "success".
- Output filtering states when it was skipped and why.

## Deliberately deferred or undecided

- No built-in Tree-sitter, ctags, LSP, or custom language-parser stack. Optional
  external MCP integration may come later.
- No bundled `systematic-debugging` skill; a general problem-solving methodology
  was discussed but not finalised.
- No fixed summary-token budget and no hard reduction threshold that discards
  useful compaction results.
- No automatic replay of possibly executed shell commands when resuming.
- No mandatory vector index or embedding service for history search.
- A global, durable section tree with stable IDs and fully semantic incremental
  restructuring remains future work.
- Complete, provider-independent discovery of context-window sizes remains
  unresolved.
- Nested AGENTS.md progressive disclosure beyond the 32 KiB instruction budget
  (on-demand load when editing a deep path) remains optional future work; the
  root→CWD chain is already injected each turn.
- A general Chat Completions adapter and an automatic IPv4 fallback are not
  mandatory core features.

## Acceptance criteria for further development

A change counts as sufficiently validated only when:

1. The Linux-amd64 binary builds without new mandatory runtime services, and its
   size is deliberately checked.
2. `go test ./...`, `go vet ./...`, and the relevant regression tests pass.
3. The feature is exercised against realistic, large persisted sessions —
   especially across model changes, tool-call boundaries, and compaction.
4. Interruptions, network errors, and partial API responses neither corrupt
   sessions nor trigger uncontrolled tool retries.
5. Context reduction is judged together with semantic fidelity, never by
   percentage alone.
6. Status and error messages clearly state what succeeded and what failed.

## Summary

Keimling stays a robust, small, model-agnostic terminal agent. The main
investments are precise text-based file editing, lightweight extensible skills,
reliable agent and autopilot orchestration, and semantically structured,
reversible context management. Powerful but heavyweight specialist capabilities
belong in optional external integrations rather than in the core.
