# Keimling documentation

This directory describes what Keimling is meant to be, how it works, and which
rules its tools follow. It is derived from the consolidated product and
architecture concept and uses the current project name throughout; the project
was previously called *tinyagent*.

## Contents

| Document | Topic |
| --- | --- |
| [concept.md](concept.md) | Product vision, guiding principles, platform, provider integration, reliability |
| [agent-loop.md](agent-loop.md) | Terminal REPL, slash commands, approval modes, autopilot |
| [tools-and-editing.md](tools-and-editing.md) | Tool surface, file reading, `replace_lines`, `apply_patch` |
| [context-and-history.md](context-and-history.md) | JSONL journal, model switching, semantic compaction, context estimation |
| [security-and-quality.md](security-and-quality.md) | Safety rules, deferred work, acceptance criteria |

## Related documents

- [`../README.md`](../README.md) — installation, command-line options, interactive commands
- [`../src`](../src) — implementation

## Status conventions

These documents mix agreed design and verified behaviour. Statements that
describe something not implemented yet are marked as *planned* or *deferred*.
Configuration paths, flags, and commands match the current implementation:

- configuration: `~/.config/keimling/config.json`
- session data: `~/.local/state/keimling/`
- binary: `./keimling`, built from `src/` with `make`

## Language

The documentation is English to match the source code, tool descriptions, and
prompts. Conversations with the agent can happen in any language.
