package main

import (
	"encoding/json"
	"strings"
)

type sessionMessage struct {
	Role    string `json:"role"` // user | assistant | tool | tool_group | system
	Name    string `json:"name,omitempty"`
	Text    string `json:"text,omitempty"`
	Command string `json:"command,omitempty"` // tool_group: visible command / label
	Result  string `json:"result,omitempty"`  // model-facing / filtered journal output
	Raw     string `json:"raw,omitempty"`     // raw shell output when distinct (often empty in journal)
	Event   int    `json:"event,omitempty"`   // 1-based journal item index
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
	for i := 0; i < len(history); i++ {
		ev := i + 1
		var item struct {
			Type      string `json:"type"`
			Role      string `json:"role"`
			Content   any    `json:"content"`
			Name      string `json:"name"`
			CallID    string `json:"call_id"`
			Arguments string `json:"arguments"`
			Output    string `json:"output"`
		}
		if json.Unmarshal(history[i], &item) != nil {
			continue
		}
		switch {
		case item.Role == "user":
			if s, ok := item.Content.(string); ok {
				s = strings.TrimSpace(s)
				if s == "" || strings.HasPrefix(s, "[AUTOPILOT") || strings.HasPrefix(s, "Continue the current plan") {
					continue
				}
				if strings.HasPrefix(s, "<compacted_conversation_memory>") {
					out = append(out, sessionMessage{Role: "system", Text: "Context compacted — earlier work is in chapters / history tools.", Event: ev})
					continue
				}
				out = append(out, sessionMessage{Role: "user", Text: s, Event: ev})
			}
		case item.Type == "message":
			if text := strings.TrimSpace(messageText(item.Content)); text != "" {
				out = append(out, sessionMessage{Role: "assistant", Text: text, Event: ev})
			}
		case item.Type == "function_call":
			cmd := toolCommandLabel(item.Name, item.Arguments)
			result := ""
			raw := ""
			endEv := ev
			if i+1 < len(history) {
				var next struct {
					Type      string `json:"type"`
					CallID    string `json:"call_id"`
					Output    string `json:"output"`
					RawOutput string `json:"raw_output"`
				}
				if json.Unmarshal(history[i+1], &next) == nil && next.Type == "function_call_output" {
					if item.CallID == "" || next.CallID == "" || next.CallID == item.CallID {
						result = strings.TrimSpace(next.Output)
						raw = strings.TrimSpace(next.RawOutput)
						endEv = i + 2
						i++ // consume paired output
					}
				}
			}
			out = append(out, sessionMessage{
				Role:    "tool_group",
				Name:    item.Name,
				Command: cmd,
				Result:  result,
				Raw:     raw,
				Event:   endEv,
			})
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
	return strings.TrimSpace(loadHistoryIndex(sessionPath).WorkingState)
}
