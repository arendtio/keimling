package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const charterTailKeep = 12

// refreshCharter folds recent model-facing history into the structured charter
// and shrinks History to a short raw tail (no semantic summary blob).
func (a *Agent) refreshCharter(ctx context.Context, instruction string) error {
	if a == nil || a.Client == nil {
		return fmt.Errorf("no client")
	}
	end := safePrefixEnd(a.History, charterTailKeep)
	if end == 0 && len(a.History) > 2 {
		end = safePrefixEnd(a.History, 2)
	}
	if end == 0 {
		return fmt.Errorf("nothing safe to fold into charter yet")
	}
	material := forCompaction(a.History[:end])
	if len(material) == 0 {
		return fmt.Errorf("nothing conversational left to fold")
	}
	c := a.ensureCharter()
	beforeEst := a.contextOccupancyEstimate()

	var body strings.Builder
	body.WriteString("Current charter JSON:\n")
	cur, _ := json.MarshalIndent(c, "", "  ")
	body.Write(cur)
	body.WriteString("\n\nNew material to fold (truncated JSONL events):\n")
	for i, raw := range material {
		if i >= 80 {
			body.WriteString("…(truncated)\n")
			break
		}
		body.WriteString(truncate(string(raw), 1500))
		body.WriteByte('\n')
	}
	if extra := strings.TrimSpace(instruction); extra != "" {
		body.WriteString("\nADDITIONAL USER INSTRUCTION: ")
		body.WriteString(extra)
		body.WriteByte('\n')
	}

	instr := `Update the project charter JSON. Return ONLY valid JSON with keys:
goal (string), plan (array of {step,status:pending|in_progress|done}), position (1-based int),
findings (string array), open_questions (string array), constraints (string array), recent (string array of short notes not yet folded).
Merge durable facts into findings/constraints/goal/plan; clear or shrink recent after folding.
Do not invent facts. Keep findings terse. English. No markdown fences.`

	a.reportCompaction("prepare", fmt.Sprintf("folding %d events into charter", len(material)), time.Now(), false, nil)

	var text string
	err := a.withCompactionProgress("model", "refreshing charter with model", func() error {
		out, e := a.Client.compactionAsk(ctx, a.compactionModel(), instr, body.String(), 4096)
		text = out
		return e
	})
	if err != nil {
		return fmt.Errorf("charter refresh: %w", err)
	}
	applyStarted := time.Now()
	a.reportCompaction("apply", "writing charter and shrinking tail", applyStarted, false, nil)
	next, err := parseCharterJSON(text)
	if err != nil {
		a.reportCompaction("apply", "parse charter", applyStarted, true, err)
		return fmt.Errorf("charter refresh parse: %w", err)
	}
	a.Charter = next
	if err := a.persistCharter(); err != nil {
		a.reportCompaction("apply", "persist charter", applyStarted, true, err)
		return err
	}
	tail := a.History[end:]
	if err := a.checkpoint(tail, "charter"); err != nil {
		a.reportCompaction("apply", "checkpoint", applyStarted, true, err)
		return err
	}
	_ = a.updateHistoryIndex("") // UI chapter bookmarks only
	afterEst := a.contextOccupancyEstimate()
	a.reportCompaction("done", fmt.Sprintf("charter refresh ~%d → ~%d tokens", beforeEst, afterEst), applyStarted, true, nil)
	fmt.Println(a.Charter.RenderMarkdown())
	return nil
}

func parseCharterJSON(text string) (*Charter, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty charter")
	}
	// Strip optional markdown fence.
	if i := strings.Index(text, "{"); i >= 0 {
		if j := strings.LastIndex(text, "}"); j > i {
			text = text[i : j+1]
		}
	}
	var c Charter
	if err := json.Unmarshal([]byte(text), &c); err != nil {
		return nil, err
	}
	c.Version = charterVersion
	c.clamp()
	return &c, nil
}
