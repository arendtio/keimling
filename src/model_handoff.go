package main

import (
	"encoding/json"
	"fmt"
)

// portableHistoryView removes provider-bound reasoning data from the active
// Responses input while leaving the append-only journal's original items intact.
// It is used only when the user explicitly requests a model handoff.
func portableHistoryView(history []json.RawMessage) ([]json.RawMessage, int, error) {
	out := make([]json.RawMessage, 0, len(history))
	removed := 0
	for i, raw := range history {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, 0, fmt.Errorf("history item %d: %w", i, err)
		}
		var typ string
		_ = json.Unmarshal(obj["type"], &typ)
		if typ == "reasoning" {
			removed++
			continue
		}
		if typ == "message" || typ == "" {
			// Provider-specific thinking blocks in message content are not transferable.
			var parts []json.RawMessage
			if json.Unmarshal(obj["content"], &parts) == nil {
				filtered := make([]json.RawMessage, 0, len(parts))
				for _, part := range parts {
					var p map[string]json.RawMessage
					if json.Unmarshal(part, &p) != nil {
						filtered = append(filtered, part)
						continue
					}
					var pt string
					_ = json.Unmarshal(p["type"], &pt)
					if pt == "thinking" || pt == "redacted_thinking" || pt == "reasoning" {
						removed++
						continue
					}
					filtered = append(filtered, part)
				}
				obj["content"], _ = json.Marshal(filtered)
			}
		}
		b, err := json.Marshal(obj)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, b)
	}
	return out, removed, nil
}

func (a *Agent) prepareNewModelContinuation() (int, error) {
	// Resolve interrupted native tool calls first; never repeat side effects.
	if err := a.repairInterruptedCalls(); err != nil {
		return 0, err
	}
	view, removed, err := portableHistoryView(a.History)
	if err != nil {
		return 0, err
	}
	if err := a.Journal.AppendCheckpoint(view, "model-handoff"); err != nil {
		return 0, err
	}
	a.History = a.Journal.History
	return removed, nil
}
