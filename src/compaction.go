package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// compactionSkipEvent drops non-conversation items (system/developer) from compact input/output.
func compactionSkipEvent(raw json.RawMessage) bool {
	var v struct {
		Role string `json:"role"`
	}
	if json.Unmarshal(raw, &v) != nil {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(v.Role))
	return role == "system" || role == "developer"
}

// forCompaction returns the conversational slice to feed the semantic compactor.
func forCompaction(history []json.RawMessage) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(history))
	for _, raw := range history {
		if compactionSkipEvent(raw) {
			continue
		}
		out = append(out, raw)
	}
	return out
}

// The journal is the immutable source of truth. History is a compact model-facing view.
// Conservative, configurable estimate; API-reported tokens are preferred once available.
func contextEstimate(history []json.RawMessage) int {
	n := 0
	for _, h := range history {
		n += len(h)
	}
	return n/3 + 16*len(history)
}
func itemKind(raw json.RawMessage) (typ, call string) {
	var x struct {
		Type   string `json:"type"`
		CallID string `json:"call_id"`
	}
	_ = json.Unmarshal(raw, &x)
	return x.Type, x.CallID
}
func pruneOutputs(history []json.RawMessage, keepRecent int) ([]json.RawMessage, int) {
	out := append([]json.RawMessage(nil), history...)
	changed := 0
	end := len(out) - keepRecent
	if end < 0 {
		end = 0
	}
	for i := 0; i < end; i++ {
		typ, _ := itemKind(out[i])
		if typ != "function_call_output" {
			continue
		}
		var v map[string]json.RawMessage
		if json.Unmarshal(out[i], &v) != nil {
			continue
		}
		var old string
		if json.Unmarshal(v["output"], &old) != nil || len(old) < 280 {
			continue
		}
		// Do not confuse an absent transcript with successful execution.
		head := old
		if len(head) > 160 {
			head = head[:160]
		}
		tail := old
		if len(tail) > 160 {
			tail = tail[len(tail)-160:]
		}
		short := fmt.Sprintf("[Earlier tool output archived; not complete. Excerpt start: %q; excerpt end: %q. Re-run a read-only inspection if exact details are needed.]", head, tail)
		b, _ := json.Marshal(short)
		if len(b) >= len(v["output"]) {
			continue
		}
		v["output"] = b
		b, _ = json.Marshal(v)
		out[i] = b
		changed++
	}
	return out, changed
}

// Only split the history at a boundary outside a pending function call.
func safePrefixEnd(history []json.RawMessage, recent int) int {
	target := len(history) - recent
	if target < 1 {
		return 0
	}
	pending := map[string]bool{}
	boundary := 0
	for i, raw := range history {
		typ, id := itemKind(raw)
		// A later user turn is evidence that an earlier incomplete tool turn
		// is historical, not an in-flight operation. Keep its raw event in
		// the journal; the compactor will record execution as unknown.
		var role struct {
			Role string `json:"role"`
		}
		_ = json.Unmarshal(raw, &role)
		if role.Role == "user" && len(pending) != 0 {
			pending = map[string]bool{}
		}
		if typ == "function_call" {
			pending[id] = true
		}
		if typ == "function_call_output" {
			delete(pending, id)
		}
		if i+1 <= target && len(pending) == 0 {
			boundary = i + 1
		}
	}
	return boundary
}
func (a *Agent) maybeCompact(ctx context.Context) error {
	return a.manageBackgroundCompaction(ctx)
}

// compactContext supports explicit soft or semantic compaction without a usage threshold.
func (a *Agent) compactContext(ctx context.Context, level, instruction string) error {
	limit := a.Cfg.ContextLimit
	if limit <= 0 {
		return nil
	}
	estimate := contextEstimate(a.History)
	if level == "soft" {
		return fmt.Errorf("soft compaction was removed; use /compact")
	}
	if level == "auto" && estimate*100 < 70*limit {
		return nil
	}
	// Preserve the last 12 items, and do not split an active tool call.
	end := safePrefixEnd(a.History, 12)
	if end == 0 && level == "hard" && len(a.History) > 2 {
		end = safePrefixEnd(a.History, 2)
	}
	if end == 0 {
		if level == "hard" {
			fmt.Println("Context: nothing safe to compact yet")
			return nil
		}
		return fmt.Errorf("context almost full; no safe compaction boundary")
	}
	compactInput := forCompaction(a.History[:end])
	if len(compactInput) == 0 {
		return fmt.Errorf("nothing conversational left to compact (system/developer items excluded)")
	}
	// Summarize bounded chunks; never send the full history in one request.
	summary, err := func() (string, error) {
		started := time.Now()
		finished := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(1)
		fmt.Print("\r\033[2K\033[36m◌\033[0m Compacting context… 0s")
		go func() {
			defer wg.Done()
			tick := time.NewTicker(time.Second)
			defer tick.Stop()
			for {
				select {
				case <-finished:
					return
				case <-tick.C:
					fmt.Printf("\r\033[2K\033[36m◌\033[0m Compacting context… %s", time.Since(started).Truncate(time.Second))
				}
			}
		}()
		text, err := a.compactIndexed(ctx, compactInput, compactionInstruction(instruction))
		close(finished)
		wg.Wait()
		fmt.Print("\r\033[2K")
		return text, err
	}()
	if err != nil {
		return fmt.Errorf("semantic context compaction: %w", err)
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return fmt.Errorf("semantic context compaction returned empty summary")
	}
	summary, afterSize, passes := a.adaptiveCompaction(ctx, summary, estimate, a.History[end:])
	denserUsed := false
	if shouldRunDenserPass(estimate, afterSize, a.Cfg.ContextLimit) {
		if denser, denserSize, denserErr := a.denserCompaction(ctx, summary, estimate, a.History[end:]); denserErr == nil {
			summary, afterSize = denser, denserSize
			denserUsed = true
		} else {
			fmt.Println("Denser compaction skipped:", denserErr)
		}
	}
	next := makeCompactionView(summary, a.History[end:])
	if !compactionAcceptable(estimate, contextEstimate(next)) {
		return fmt.Errorf("semantic compaction did not produce a useful reduction (estimated %d → %d tokens, %.1f%%; minimum %.0f%%); original context preserved", estimate, contextEstimate(next), compactionReduction(estimate, contextEstimate(next)), compactionMinimumUsefulReduction)
	}
	fmt.Printf("Compaction: %s · additional passes: %d", compactionDecision(estimate, contextEstimate(next)), passes)
	if denserUsed {
		fmt.Print(" · denser merge")
	}
	fmt.Println()
	if err := a.checkpoint(next, "semantic"); err != nil {
		return err
	}
	_ = a.updateHistoryIndex(summary)
	fmt.Print(formatCompactionReport(summary, estimate, contextEstimate(next)))
	return nil
}
func (a *Agent) checkpoint(next []json.RawMessage, level string) error {
	if a.Compaction != nil {
		if a.Compaction.cancel != nil {
			a.Compaction.cancel()
		}
		a.Compaction = nil
	}
	if err := a.Journal.AppendCheckpoint(next, level); err != nil {
		return err
	}
	a.History = next
	if a.Client != nil {
		a.Client.LastMainUsage = MainUsage{}
	}
	return nil
}

// Both manual and background compaction use the same semantic criteria.
func compactionInstruction(extra string) string {
	base := "Preserve decisions, constraints, confirmed findings, current goals and open issues; " +
		"remove redundant diagnostics, repeated tool output, and superseded investigation details. " +
		"Choose semantic sections and compression modes according to information value, not chronology."
	if strings.TrimSpace(extra) == "" {
		return base
	}
	return base + "\nADDITIONAL USER COMPACTION INSTRUCTION: " + strings.TrimSpace(extra)
}

func compactionSufficient(before, after int) bool {
	return before > 0 && after >= 0 && after <= before*7/10
}

func compactionReduction(before, after int) float64 {
	if before <= 0 {
		return 0
	}
	return 100 * float64(before-after) / float64(before)
}
