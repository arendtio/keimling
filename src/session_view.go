package main

import (
	"encoding/json"
	"strings"
)

type sessionMessage struct {
	Role      string   `json:"role"` // user | assistant | tool | tool_group | system
	Name      string   `json:"name,omitempty"`
	Text      string   `json:"text,omitempty"`
	Command   string   `json:"command,omitempty"`   // tool_group: visible command / label
	Result    string   `json:"result,omitempty"`    // display / model-facing output
	Raw       string   `json:"raw,omitempty"`       // raw shell output only when filtered
	Filtered  bool     `json:"filtered,omitempty"`  // true when Raw differs from model Result
	Status    string   `json:"status,omitempty"`    // failed | stopped; empty means ran
	Collapsed bool     `json:"collapsed,omitempty"` // long system note starts closed
	Event     int      `json:"event,omitempty"`     // 1-based journal item index
	Files     []string `json:"files,omitempty"`
}

type sessionChapter struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	Memory     string `json:"memory,omitempty"` // raw | keep | distill | summarize | collapse
	Summary    string `json:"summary,omitempty"`
	Intro      string `json:"intro,omitempty"`
	Conclusion string `json:"conclusion,omitempty"`
	Start      int    `json:"start"`
	End        int    `json:"end"`
}

func sessionMessages(history []json.RawMessage, limit int) []sessionMessage {
	if limit <= 0 {
		limit = 120
	}
	var out []sessionMessage
	consumed := map[int]bool{}
	for i := 0; i < len(history); i++ {
		if consumed[i] {
			continue
		}
		ev := i + 1
		var item struct {
			Type      string `json:"type"`
			Role      string `json:"role"`
			Content   any    `json:"content"`
			Name      string `json:"name"`
			CallID    string `json:"call_id"`
			Arguments string `json:"arguments"`
			Output    string `json:"output"`
			Text      string `json:"text"`
			Peer      string `json:"peer"`
			Direction string `json:"direction"`
		}
		if json.Unmarshal(history[i], &item) != nil {
			continue
		}
		switch {
		case item.Role == "user":
			if s, ok := item.Content.(string); ok {
				body, files := splitChatFiles(s)
				s = strings.TrimSpace(body)
				names := chatFileNames(files)
				if s == "" && len(names) == 0 {
					continue
				}
				if len(names) > 0 && !strings.HasPrefix(s, "[AUTOPILOT") && !strings.HasPrefix(s, "Continue the current plan") && !strings.HasPrefix(s, "<compacted_conversation_memory>") {
					if _, _, peer := parsePeerUserText(s); !peer {
						out = append(out, sessionMessage{Role: "user", Text: s, Files: names, Event: ev})
						continue
					}
				}
				if s == "" {
					continue
				}
				if strings.HasPrefix(s, "[AUTOPILOT") || strings.HasPrefix(s, "Continue the current plan") {
					out = append(out, sessionMessage{Role: "system", Text: s, Collapsed: true, Event: ev})
					continue
				}
				if strings.HasPrefix(s, "<compacted_conversation_memory>") {
					out = append(out, sessionMessage{Role: "system", Text: "Context compacted — earlier work is in chapters / history tools.", Event: ev})
					continue
				}
				if name, body, ok := parsePeerUserText(s); ok {
					out = append(out, sessionMessage{Role: "peer", Name: "From " + name, Text: body, Event: ev})
					continue
				}
				out = append(out, sessionMessage{Role: "user", Text: s, Event: ev})
			}
		case item.Type == "message":
			if text := strings.TrimSpace(messageText(item.Content)); text != "" {
				out = append(out, sessionMessage{Role: "assistant", Text: text, Event: ev})
			}
		case item.Type == "keimling_peer":
			body, fileRefs := splitChatFiles(item.Text)
			text := strings.TrimSpace(body)
			peer := strings.TrimSpace(item.Peer)
			names := chatFileNames(fileRefs)
			if (text == "" && len(names) == 0) || peer == "" {
				continue
			}
			name := "To " + peer
			if item.Direction == "received" {
				name = "From " + peer
			}
			out = append(out, sessionMessage{Role: "peer", Name: name, Text: text, Files: names, Event: ev})
		case item.Type == "keimling_notice":
			text := strings.TrimSpace(item.Text)
			if text == "" {
				continue
			}
			out = append(out, sessionMessage{Role: "system", Text: text, Collapsed: noticeCollapsed(text), Event: ev})
		case item.Type == "function_call":
			cmd := toolCommandLabel(item.Name, item.Arguments)
			result := ""
			raw := ""
			var toolOK *bool
			endEv := ev
			if outIdx, next, ok := findCallOutput(history, i, item.CallID); ok {
				result = strings.TrimSpace(next.Output)
				raw = strings.TrimSpace(next.RawOutput)
				toolOK = next.OK
				endEv = outIdx + 1
				consumed[outIdx] = true
			}
			filtered := raw != "" && raw != result
			if !filtered {
				raw = ""
			}
			out = append(out, sessionMessage{
				Role:     "tool_group",
				Name:     item.Name,
				Command:  cmd,
				Result:   result,
				Raw:      raw,
				Filtered: filtered,
				Status:   toolCardStatus(result, raw, toolOK),
				Event:    endEv,
			})
			if item.Name == "finish_task" {
				if summary := finishTaskSummaryMessage(item.Arguments, endEv); summary != nil {
					out = append(out, *summary)
				}
			}
		case item.Type == "function_call_output":
			// Orphan output (call already consumed or missing).
			t := strings.TrimSpace(item.Output)
			if t == "" {
				continue
			}
			out = append(out, sessionMessage{Role: "tool_group", Name: "result", Command: "result", Result: t, Event: ev})
		}
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

type callOutput struct {
	Output    string
	RawOutput string
	OK        *bool
}

// findCallOutput pairs a tool call with its result. A peer line may sit between them.
func findCallOutput(history []json.RawMessage, callIndex int, callID string) (int, callOutput, bool) {
	for j := callIndex + 1; j < len(history) && j < callIndex+6; j++ {
		var next struct {
			Type      string `json:"type"`
			CallID    string `json:"call_id"`
			Output    string `json:"output"`
			RawOutput string `json:"raw_output"`
			OK        *bool  `json:"ok"`
		}
		if json.Unmarshal(history[j], &next) != nil {
			continue
		}
		if next.Type == "function_call" {
			break
		}
		if next.Type != "function_call_output" {
			continue
		}
		if callID != "" && next.CallID != "" && next.CallID != callID {
			continue
		}
		return j, callOutput{Output: next.Output, RawOutput: next.RawOutput, OK: next.OK}, true
	}
	return 0, callOutput{}, false
}

func finishTaskSummaryMessage(args string, event int) *sessionMessage {
	var f struct {
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}
	if json.Unmarshal([]byte(args), &f) != nil {
		return nil
	}
	summary := strings.TrimSpace(f.Summary)
	if summary == "" {
		return nil
	}
	title := "Task complete"
	if f.Status == "blocked" {
		title = "Action needed"
	} else if f.Status != "done" {
		title = "Task " + strings.TrimSpace(f.Status)
		if title == "Task " {
			title = "Task finished"
		}
	}
	return &sessionMessage{
		Role:  "assistant",
		Text:  "**" + title + "**\n\n" + summary,
		Event: event,
	}
}

func toolCardStatus(result, raw string, ok *bool) string {
	if strings.HasPrefix(strings.TrimSpace(result), "INTERRUPTED:") {
		return "stopped"
	}
	if ok != nil {
		if !*ok {
			return "failed"
		}
		return ""
	}
	if toolTextFailed(result) || toolTextFailed(raw) {
		return "failed"
	}
	return ""
}

func toolTextFailed(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, p := range []string{
		"DENIED", "SKIPPED", "INTERRUPTED:",
		"patch failed:", "hash mismatch", "file hash mismatch",
		"invalid line range", "invalid status", "invalid tool arguments",
		"access denied:", "plan too long",
		"activate_skill:", "read_skill_file:", "show_artifact error:",
	} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	if i := strings.Index(s, " error:"); i >= 0 && i < 48 {
		return true
	}
	if code, found := leadingExitCode(s); found && code != "0" {
		return true
	}
	return false
}

func leadingExitCode(s string) (string, bool) {
	s = strings.TrimSpace(s)
	const key = "exit_code="
	if !strings.HasPrefix(s, key) {
		return "", false
	}
	rest := s[len(key):]
	if i := strings.IndexAny(rest, " \n\r\t"); i >= 0 {
		rest = rest[:i]
	}
	if rest == "" {
		return "", false
	}
	return rest, true
}

func markToolOK(name, result, raw string) bool {
	if toolTextFailed(result) || toolTextFailed(raw) {
		return false
	}
	if name == "exec" {
		src := raw
		if strings.TrimSpace(src) == "" {
			src = result
		}
		if code, found := leadingExitCode(src); found && code != "0" {
			return false
		}
	}
	return true
}

func toolCommandLabel(name, args string) string {
	if name == "exec" {
		var a struct {
			Command string `json:"command"`
		}
		if json.Unmarshal([]byte(args), &a) == nil && strings.TrimSpace(a.Command) != "" {
			return a.Command
		}
	}
	if name == "edit_file" || name == "write_file" || name == "read_file" {
		var a struct {
			Path string `json:"path"`
		}
		if json.Unmarshal([]byte(args), &a) == nil && a.Path != "" {
			return name + " " + a.Path
		}
	}
	if name == "message_agent" {
		var a struct {
			InstanceID string `json:"instance_id"`
			Text       string `json:"text"`
		}
		if json.Unmarshal([]byte(args), &a) == nil && strings.TrimSpace(a.Text) != "" {
			return strings.TrimSpace(a.Text)
		}
	}
	if name == "list_agents" {
		return "list_agents"
	}
	if name == "data_processing" || name == "data_processing_help" {
		if label := dataProcessingLabel(name, args); label != "" {
			return label
		}
	}
	if strings.TrimSpace(args) != "" && len(args) < 160 {
		return name + " " + args
	}
	return name
}

func sessionChapters(sessionPath string) []sessionChapter {
	idx := loadHistoryIndex(sessionPath)
	if len(idx.Sections) == 0 {
		if events, err := originalEvents(sessionPath); err == nil && len(events) > 0 {
			idx = buildHistoryIndex(events, HistoryIndex{Version: historyIndexVersion, WorkingState: idx.WorkingState})
			_ = saveHistoryIndex(sessionPath, idx)
		}
	}
	if len(idx.Sections) == 0 {
		return nil
	}
	var chapters []sessionChapter
	for _, s := range idx.Sections {
		if s.Level != "task" || s.Hidden {
			continue
		}
		mem := s.Memory
		if mem == "" {
			mem = memoryRaw
		}
		chapters = append(chapters, sessionChapter{
			ID:         s.ID,
			Title:      s.Title,
			Status:     s.Status,
			Memory:     mem,
			Summary:    s.Summary,
			Intro:      s.Intro,
			Conclusion: s.Conclusion,
			Start:      s.Start,
			End:        s.End,
		})
	}
	return chapters
}

func sessionWorkingState(sessionPath string) string {
	if c, err := loadCharter(sessionPath); err == nil && c != nil && !c.Empty() {
		return c.RenderMarkdown()
	}
	return strings.TrimSpace(loadHistoryIndex(sessionPath).WorkingState)
}
