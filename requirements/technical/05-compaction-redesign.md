# Compaction redesign (draft)

Status: implemented (core path) — see `src/semantic_render.go`, denser trigger in
`compactContext` / `adoptCompaction`, `--compact-parallel`, GUI intro/conclusion.

## Goals

- Improve compaction **while simplifying** the algorithm where possible.
- Keep quality; do not start by “use a cheaper model for everything”.
- Align compact output with the GUI chapter model so the same structure is usable in UI and in model context.
- Separate a **fast index/classify** phase from a **slower, optionally parallel render** phase.

## Two compaction modes

Incremental forward compaction alone is not enough: after many chapters exist, the agent still needs a way to shrink the compact view further.

| Mode | Input | Intent |
| --- | --- | --- |
| **Raw → compact** | Unstructured / raw journal slice (new work since last checkpoint) | Segment into chapters/sections, assign treatment, produce compact bodies |
| **Compact → denser** | Already compacted chapters (and their tree) | Drop obsolete chapters, **merge** several chapters into one, or re-treat existing nodes without re-reading all raw events |

Both modes must be designed explicitly. “Only compact the future” is the common path; “recompact the compact archive” is the escape hatch when chapter count or size grows again.

## Hierarchy

- Nested structure stays valuable: keep a parent chapter while collapsing or summarizing only some children.
- Practical depth: **up to ~3 levels** (10 was a max ceiling, not a target).
- Not every node needs title + intro + body + conclusion.
- Framing (title / short intro / conclusion) is mainly needed for:
  - `summarize` (especially),
  - `distill`,
  - and typically **top-level** chapters.
- Leaves that are `keep` or `collapse` can stay minimal (verbatim or archive pointer).

## Pipeline phases

### 1. Index (should be seconds)

- Decide boundaries: which event ranges (or existing chapter IDs) form which nodes.
- Assign treatment: `keep` | `distill` | `summarize` | `collapse` (and, in denser mode: drop / merge).
- Produce titles lightly if cheap; **do not** generate full intro/conclusion here.

### 2. Render (slower; optionally parallel)

- Per node that needs body work (`distill` / `summarize`, and framing where required): generate body and, when applicable, intro/conclusion.
- Parallelism is a **config toggle** (on for remote API, often off for local single-model hosts).
- Framing and summaries run in this phase because they cost time and parallelize well.

### 3. Working state

- One concise global working state after render (objective, constraints, open issues, links across chapters).

## Context for leaf summaries (critical)

When summarizing or distilling chapter N, the model must **not** see only chapter N’s raw events.

Required surrounding context at least includes:

- Original / current overall objective (and durable user constraints)
- High-level arc: where the story came from and where it is going (e.g. sibling/parent titles, working state, neighboring chapter titles or short notes)
- Enough global state that intro/conclusion can situate the chapter in the whole task

Isolated per-chapter summarization without arc context is considered incorrect.

## Denser-mode triggers (automatic)

Denser compaction is primarily **automatic**, not only manual. Indications include:

- Many chapters are losing relevance and need a cross-cutting recompact (model/system notices archive bloat or low relevance).
- After a compaction pass, **little headroom was freed** and usage remains high (e.g. still around **~70%** of the context budget) — treat that as a signal to run denser mode on the already-compacted view, not only to wait for more raw events.

Exact thresholds and who decides (heuristic vs. model judgment) remain implementation detail; the requirement is that denser mode can fire from these situations without the user inventing a special command first. Manual `/compact` may still force either mode.

## Merge visibility in the GUI

When chapters are merged:

- The result appears as a **new chapter** in the UI.
- The **old chapters are no longer shown** in the chat/chapter UI (they are gone from the visible compact narrative).

Raw journal retention for recovery/search is unchanged at the storage layer; this decision is about **what the product shows** as the living chapter list.

## Chat UI vs. model context (near-term)

Products like Cursor often keep a full chat transcript visible while the model only receives a reduced context.

Keimling already separates immutable journal from a compact model-facing view under the hood. For the **near term**, the product choice is:

- The in-app chat/chapter view should show the **compact context** (what the agent is effectively working from), not a parallel “full transcript UI” that diverges from context.
- A later option to show full journal history beside/under compact context is **deferred** — interesting, but not required for the first redesign.

## Relationship to GUI chapters

Compact chapters should be the same conceptual objects the Web UI folds/shows (stable identity, title, treatment, optional intro/body/conclusion), not a parallel undocumented memory format.

## Explicitly deferred / open

- Exact schema for intro/conclusion fields
- Numeric thresholds for “still ~70%” / chapter-count triggers
- Dual UI: full chat history vs. reduced context (Cursor-style split) — later
- Whether index may use a smaller model later (quality-first; evaluate only after the content algorithm is right)

## Non-goals for this draft

- Replacing semantic treatment modes with a single flat summary
- Removing nested children entirely
- Optimizing primarily via cheaper models before the algorithm is clear
- Shipping a separate full-history chat UI in the first redesign
