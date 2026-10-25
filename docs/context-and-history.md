# History, model switching, and context compaction

## The journal

- The full conversation, all tool calls, and all results are persisted as a
  JSONL journal under `~/.local/state/keimling/`; existing sessions resume
  automatically (`--resume=true` by default).
- Compaction never overwrites or deletes original session data.
- Section indexes, summaries, working state, and checkpoints live in separate,
  reconstructible sidecar files.
- Historical content stays reachable through `search_history` (section names,
  titles, summaries) and `read_history` (targeted retrieval of original events).

## Switching models and providers

- On a model or provider change, visible messages and valid tool outputs are
  transferred, while incompatible encrypted thinking blocks are excluded from
  the new active context. The encrypted originals stay in the journal.
- `/continue-new-model` does exactly this and does not require the original model
  to be reachable.
- An optional semantic model handoff (turning the previous model's knowledge into
  portable text) was discussed but is neither required nor able to reconstruct
  hidden reasoning completely.

## Semantic compaction

### Goal

Reduce context while keeping the agent able to act and retaining essential
information. The rule is **not** to rewrite everything more briefly: first
segment by meaning and assess relevance, then distill or hide selected content.

### Section tree

- The model identifies semantic section and subsection boundaries using event
  IDs, not line numbers or fixed-size blocks.
- The tree is recursive with up to ten levels; actual depth follows the content.
- Parent and child nodes are evaluated independently, so an important overall
  topic can stay active while completed subtopics are distilled or collapsed.
- Original events are assigned to leaf ranges completely and without overlap;
  parents aggregate their children without duplicating events.
- The section index is updated incrementally during long sessions. Stable node
  IDs and a globally linked tree remain future work.

### Treatment modes

| Mode | Meaning |
| --- | --- |
| `keep` / `verbatim` | The exact original wording is genuinely necessary |
| `distill` | Essential findings, precise technical facts, and open tasks in highly condensed form |
| `summarize` | Briefly retain enduringly relevant results and connections |
| `collapse` | Keep only a minimal archive reference in the active context |

A good distillation prioritises decisions, confirmed results, technical names,
paths, versions, root causes, open problems, and rejected hypotheses with their
rationale. Repetitive logs and process narratives may go if the consequential
findings survive. Facts, hypotheses, and unverified claims are never conflated.

### Procedure

1. Semantically analyse the original history, or the part not yet structured.
2. Produce and structurally validate the section tree, relevance decisions, and
   fidelity requirements.
3. Process `distill`/`summarize` sections, replace `collapse` sections with a
   short archive reference, and reserve `keep` for indispensable exact content.
4. Review the whole result for coherence, contradictions, duplicated facts,
   current constraints, and outstanding tasks.
5. Estimate net savings and atomically commit the best qualitatively valid
   candidate as the active context checkpoint, without touching the journal.
6. Report before/after token estimates, percentage saved, node count and depth,
   treatment modes, and informative section titles.

### Adaptive acceptance

- The quality target is **at least 30 % reduction** (at most 70 % of the original
  context remaining).
- This target is not a hard rejection threshold. A valid result between 10 % and
  30 % reduction gets up to two additional distillation passes, and the best valid
  result wins.
- 30 % or more is accepted immediately; below 10 % is rejected only after those
  additional attempts.
- Further passes work on the already condensed candidate, not on the original
  history again.
- Fixed output limits such as 8192 tokens for every compaction request are
  avoided in favour of dynamic budgets informed by input size and provider
  constraints; `incomplete_details.reason` is inspected and `max_output_tokens`
  retried appropriately.
- Empty or invalid semantic JSON is diagnosed, corrected when possible, and
  validated again. Structured outputs are preferred where the provider supports
  them.
- Valid intermediate candidates are preserved so a later failed pass does not
  discard earlier progress.

## Background compaction

- Automatic compaction starts before the limit is reached (around 70 % usage)
  and guarantees safe continuation at high usage (around 90 %).
- Background compaction has a session-scoped context: it does not fail with
  `context canceled` merely because an individual agent task finished or was
  stopped, and it is cancelled cleanly when switching sessions or exiting.
- Snapshots are taken only at complete tool-call/result boundaries. Orphaned tool
  calls are diagnosed without rerunning commands of uncertain execution state;
  calls still in flight are protected.
- If no background snapshot is available at the critical threshold, a controlled
  foreground recovery attempt runs.
- Failed background compaction backs off instead of retrying immediately.
- Manual and automatic compaction use the same semantic criteria; only snapshot
  timing and execution flow differ.
- On rejection or fallback, concrete token estimates, achieved savings, and the
  reason are shown.

## Context estimation

- A raw JSON-byte heuristic is too inaccurate and can overestimate real usage
  several times over.
- Actual `usage.input_tokens` from the Responses API calibrates the estimate;
  history growth since the measurement is added, plus a safety margin (currently
  15 %).
- Calibration is invalidated or re-evaluated when switching model or provider,
  starting a new chat, or changing input parameters.
- `/status` distinguishes measured API values, `usage-calibrated` estimates, and
  uncalibrated `heuristic` estimates, and reports `cached_tokens` and
  `reasoning_tokens` separately. Cached input tokens still occupy context even
  when billing is cheaper.
