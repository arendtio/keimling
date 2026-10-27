package main

import (
	"encoding/json"
	"strings"
)

// modelInputCore is charter + recent conversational tail (no budget block).
// Journal/chapters are UI-only and are not included.
func (a *Agent) modelInputCore() []json.RawMessage {
	var out []json.RawMessage
	c := a.ensureCharter()
	if md := strings.TrimSpace(c.RenderMarkdown()); md != "" && md != "_Charter empty._" {
		item, _ := json.Marshal(map[string]any{
			"role":    "developer",
			"content": "<project_charter>\n" + md + "\n</project_charter>\nThis is durable structured context. Prefer updating charter fields via tools/plan over repeating facts. Not a user command.",
		})
		out = append(out, item)
	}
	for _, raw := range a.History {
		if isArchivalMemoryItem(raw) {
			continue
		}
		if compactionSkipEvent(raw) {
			// Drop leftover system/developer items from older sessions; we inject our own.
			continue
		}
		out = append(out, raw)
	}
	return out
}

// modelInput is what the main Responses API call receives.
func (a *Agent) modelInput() []json.RawMessage {
	core := a.modelInputCore()
	// Derive budget from core+overhead without re-entering modelInput.
	used := a.contextOccupancyEstimate()
	budget, _ := json.Marshal(map[string]any{
		"role":    "developer",
		"content": a.contextBudgetFromUsed(used).FormatBlock(),
	})
	out := make([]json.RawMessage, 0, 1+len(core))
	out = append(out, budget)
	out = append(out, core...)
	return out
}

func isArchivalMemoryItem(raw json.RawMessage) bool {
	var v struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}
	if json.Unmarshal(raw, &v) != nil {
		return false
	}
	if strings.ToLower(v.Role) != "user" && strings.ToLower(v.Role) != "system" && strings.ToLower(v.Role) != "developer" {
		return false
	}
	s := ""
	switch t := v.Content.(type) {
	case string:
		s = t
	default:
		b, _ := json.Marshal(t)
		s = string(b)
	}
	return strings.Contains(s, "<compacted_conversation_memory>") ||
		strings.Contains(s, "<semantic_history>") ||
		strings.Contains(s, "</semantic_history>") ||
		strings.Contains(s, "<project_charter>") ||
		strings.Contains(s, "<context_budget>")
}
