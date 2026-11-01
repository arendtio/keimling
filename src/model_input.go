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
	if a != nil && strings.TrimSpace(a.Cfg.AgentInstanceID) != "" {
		if hint := peerReplyHint(out, a.Cfg.AgentInstanceID); hint != "" {
			item, _ := json.Marshal(map[string]string{"role": "developer", "content": hint})
			out = append(out, item)
		}
	}
	return out
}

// peerReplyHint tells a weak model the message_agent arguments for this turn.
// The system prompt alone is not enough: the model writes the answer in chat
// unless the recipient id is stated on the turn.
func peerReplyHint(items []json.RawMessage, selfID string) string {
	var peers []peerTurn
	var lastCall string
	var lastCallID string
	var lastOutID string
	var sawList bool
	var listOut string
	lastKind := ""
	for _, raw := range items {
		var item map[string]any
		if json.Unmarshal(raw, &item) != nil {
			continue
		}
		switch typ, _ := item["type"].(string); typ {
		case "function_call":
			name, _ := item["name"].(string)
			lastCall = name
			lastCallID, _ = item["call_id"].(string)
			if name == "list_agents" {
				sawList = true
			}
			if name == "message_agent" {
				var args struct {
					InstanceID string `json:"instance_id"`
				}
				_ = json.Unmarshal([]byte(fmtArg(item["arguments"])), &args)
				if id := strings.TrimSpace(args.InstanceID); id != "" {
					lastOutID = id
				}
			}
			lastKind = "call"
		case "function_call_output":
			id, _ := item["call_id"].(string)
			if lastCall == "list_agents" && (id == "" || id == lastCallID) {
				listOut = fmtArg(item["output"])
			}
			lastKind = "output"
		default:
			role, _ := item["role"].(string)
			if strings.EqualFold(role, "user") {
				content := fmtArg(item["content"])
				if _, body, ok := parsePeerUserText(content); ok {
					peers = append(peers, peerTurn{id: peerIDFromLabel(content), body: body})
					lastKind = "peer"
				}
			}
		}
	}
	if lastKind == "output" && lastCall == "list_agents" && len(peers) > 0 {
		id := listedAgentMentioned(listOut, peers[0].body)
		text := passOnText(peers[0].body)
		if id != "" && id != strings.TrimSpace(selfID) && text != "" {
			return "Call message_agent.\ninstance_id: " + id + "\ntext is the request to pass on, not an answer you write. For this message, text is: " + text
		}
		return ""
	}
	if lastKind != "peer" || len(peers) == 0 {
		return ""
	}
	last := peers[len(peers)-1]
	origin := peers[0].id
	if lastOutID != "" && lastOutID == last.id && last.id != origin {
		return "The message above is the answer you asked for. Call message_agent now.\ninstance_id: " + origin + "\ntext: " + last.body
	}
	if peerAsksSomeoneElse(last.body) && !sawList {
		return ""
	}
	if last.id == "" || last.id == strings.TrimSpace(selfID) {
		return ""
	}
	return "Call message_agent. instance_id: " + last.id + "\ntext is your answer to the message above. Do not write chat text."
}

type peerTurn struct {
	id, body string
}

func fmtArg(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func peerIDFromLabel(content string) string {
	name, _, ok := parsePeerUserText(content)
	if !ok {
		return ""
	}
	for _, key := range []string{"thread_id=", "instance_id="} {
		if i := strings.Index(name, key); i >= 0 {
			rest := strings.TrimSpace(name[i+len(key):])
			if f := strings.IndexAny(rest, " \t"); f >= 0 {
				rest = rest[:f]
			}
			return strings.Trim(rest, `"`)
		}
	}
	return strings.TrimSpace(name)
}

func peerAsksSomeoneElse(body string) bool {
	return strings.Contains(strings.ToLower(body), "ask ")
}

func passOnText(body string) string {
	low := strings.ToLower(body)
	i := strings.Index(low, " for ")
	if i < 0 {
		return ""
	}
	rest := strings.TrimSpace(body[i+len(" for "):])
	if j := strings.Index(strings.ToLower(rest), " and tell"); j >= 0 {
		rest = strings.TrimSpace(rest[:j])
	}
	rest = strings.Trim(rest, ".! ")
	if rest == "" {
		return ""
	}
	return "Tell me " + rest + "."
}

func listedAgentMentioned(listOut, body string) string {
	bestName, bestID := "", ""
	lowBody := strings.ToLower(body)
	for _, line := range strings.Split(listOut, "\n") {
		id := fieldAfter(line, "instance_id=")
		name := strings.Trim(fieldAfter(line, "name="), `"`)
		if id == "" || name == "" {
			continue
		}
		if strings.Contains(lowBody, strings.ToLower(name)) && len(name) > len(bestName) {
			bestName, bestID = name, id
		}
	}
	return bestID
}

func fieldAfter(line, key string) string {
	i := strings.Index(line, key)
	if i < 0 {
		return ""
	}
	rest := strings.TrimSpace(line[i+len(key):])
	if strings.HasPrefix(rest, `"`) {
		rest = rest[1:]
		if j := strings.Index(rest, `"`); j >= 0 {
			return rest[:j]
		}
	}
	if j := strings.IndexAny(rest, " \t"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

// wireModelInput is the transcript sent to the model. Stored fields that are
// only for the UI, such as raw_output, are removed.
func (a *Agent) wireModelInput() []json.RawMessage {
	items := a.modelInput()
	if a != nil {
		t := a.callChatTarget()
		items = prepareReasoningInput(items, t.ProviderID, t.Model, a.reasoningOwnerMap())
	}
	if normalized, err := normalizeResponseInput(items); err == nil {
		return normalized
	}
	return items
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
