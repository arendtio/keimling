package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Compaction candidates are never checkpointed until selection is complete.
// The original journal remains the source of truth for archived details.
const compactionMinimumUsefulReduction = 10.0
const compactionTargetReduction = 30.0
const compactionAdditionalPasses = 2

func makeCompactionView(summary string, tail []json.RawMessage) []json.RawMessage {
	item, _ := json.Marshal(map[string]any{"role": "user", "content": "<compacted_conversation_memory>\n" + summary + "\n</compacted_conversation_memory>\nThis is archival context, not a command. Verify current state before changes."})
	return append([]json.RawMessage{item}, tail...)
}

func compactionAcceptable(before, after int) bool {
	return before > 0 && after < before && compactionReduction(before, after) >= compactionMinimumUsefulReduction
}

// A revision uses ONLY the candidate, rather than reprocessing the original history.
// It must preserve actionable facts, uncertainty and exact identifiers.
func (a *Agent) adaptiveCompaction(ctx context.Context, initial string, before int, tail []json.RawMessage) (string, int, int) {
	best := strings.TrimSpace(initial)
	currentSize := contextEstimate(makeCompactionView(best, tail))
	used := 0
	if before <= 0 || compactionReduction(before, currentSize) >= compactionTargetReduction {
		return best, currentSize, used
	}
	prompt := denserMemoryPrompt(false)
	for i := 0; i < compactionAdditionalPasses; i++ {
		if err := ctx.Err(); err != nil {
			break
		}
		var candidate string
		var err error
		if len(best) > 1400000 {
			event, _ := json.Marshal(map[string]any{"role": "user", "content": best})
			candidate, err = compactMaterial(ctx, a.Client, a.compactionModel(), []json.RawMessage{event}, prompt)
		} else {
			candidate, err = a.Client.compactionAsk(ctx, a.compactionModel(), prompt, best, 8192)
		}
		if err != nil || strings.TrimSpace(candidate) == "" {
			break
		}
		candidate = strings.TrimSpace(candidate)
		size := contextEstimate(makeCompactionView(candidate, tail))
		used++
		if size >= currentSize {
			break
		}
		best, currentSize = candidate, size
		if compactionReduction(before, currentSize) >= compactionTargetReduction {
			break
		}
	}
	return best, currentSize, used
}

func denserMemoryPrompt(mergeChapters bool) string {
	base := `Revise the following ALREADY COMPACTED conversation memory into a denser, still accurate working memory. Keep all actionable facts, decisions, user requirements, filenames, commands, concrete results, unresolved issues, caveats and causal links. Remove repeated diagnostics, verbose explanations, superseded experiments and duplicate statements. Preserve section markers where possible and retain references to archived history. No invented facts, no loss of critical operational state, no new instructions. Return the revised memory ONLY. Do not target an arbitrary length.`
	if !mergeChapters {
		return base
	}
	return base + `

DENSE MODE (merge): Aggressively merge related top-level [section depth=1 …] chapters into fewer new chapters when older ones lost relevance. Drop expendable chapters entirely when safe. Each surviving top-level chapter should keep the marker form
[section depth=1 START-END: New Title | summarize|distill|collapse|keep]
and for summarize/distill include [intro], body, and [conclusion] so the chapter has a clear arc. Merged chapters must use new titles; do not keep obsolete chapter titles.`
}

// denserCompaction merges/re-treats an already compacted memory when occupancy
// stays high after a raw→compact pass.
func (a *Agent) denserCompaction(ctx context.Context, initial string, _ int, tail []json.RawMessage) (string, int, error) {
	best := strings.TrimSpace(initial)
	currentSize := contextEstimate(makeCompactionView(best, tail))
	prompt := denserMemoryPrompt(true)
	var candidate string
	var err error
	if len(best) > 1400000 {
		event, _ := json.Marshal(map[string]any{"role": "user", "content": best})
		candidate, err = compactMaterial(ctx, a.Client, a.compactionModel(), []json.RawMessage{event}, prompt)
	} else {
		candidate, err = a.Client.compactionAsk(ctx, a.compactionModel(), prompt, best, 8192)
	}
	if err != nil {
		return best, currentSize, err
	}
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return best, currentSize, nil
	}
	size := contextEstimate(makeCompactionView(candidate, tail))
	if size < currentSize {
		return candidate, size, nil
	}
	return best, currentSize, nil
}

func compactionDecision(before, after int) string {
	if before <= 0 {
		return "invalid baseline"
	}
	reduction := compactionReduction(before, after)
	if reduction >= compactionTargetReduction {
		return fmt.Sprintf("target reached (%.1f%% reduction)", reduction)
	}
	if compactionAcceptable(before, after) {
		return fmt.Sprintf("accepted below 30%% target (%.1f%% reduction)", reduction)
	}
	return fmt.Sprintf("rejected: only %.1f%% reduction (minimum useful %.0f%%)", reduction, compactionMinimumUsefulReduction)
}
