package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestSessionMessagesRoles(t *testing.T) {
	u, _ := json.Marshal(map[string]any{"role": "user", "content": "hello"})
	a, _ := json.Marshal(map[string]any{
		"type": "message",
		"content": []any{
			map[string]any{"type": "output_text", "text": "hi back"},
		},
	})
	msgs := sessionMessages([]json.RawMessage{u, a}, 10)
	if len(msgs) != 2 || msgs[0].Role != "user" || msgs[1].Role != "assistant" {
		t.Fatalf("%#v", msgs)
	}
}

func TestSessionMessagesGroupsExecResult(t *testing.T) {
	call, _ := json.Marshal(map[string]any{
		"type": "function_call", "name": "exec", "call_id": "c1",
		"arguments": `{"command":"echo hi"}`,
	})
	out, _ := json.Marshal(map[string]any{
		"type": "function_call_output", "call_id": "c1", "output": "hi\n", "raw_output": "RAW\nhi\n",
	})
	msgs := sessionMessages([]json.RawMessage{call, out}, 10)
	if len(msgs) != 1 || msgs[0].Role != "tool_group" || msgs[0].Command != "echo hi" || msgs[0].Result != "hi" || msgs[0].Raw != "RAW\nhi" {
		t.Fatalf("%#v", msgs)
	}
}

func TestSessionChaptersFromIndex(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "s.jsonl")
	idx := HistoryIndex{
		Version: historyIndexVersion,
		Sections: []HistorySection{
			{ID: "task-1", Level: "task", Title: "First chapter", Status: "completed", Memory: "raw", Start: 1, End: 10},
			{ID: "episode-1", Level: "episode", Title: "skip", Start: 1, End: 10},
			{ID: "task-2", Level: "task", Title: "Second", Status: "active", Memory: "distill", Start: 11, End: 20, Summary: "notes"},
		},
		WorkingState: "CURRENT STATE\nkeep going",
	}
	if err := saveHistoryIndex(session, idx); err != nil {
		t.Fatal(err)
	}
	ch := sessionChapters(session)
	if len(ch) != 2 || ch[0].Title != "First chapter" || ch[1].Summary != "notes" || ch[1].Memory != "distill" {
		t.Fatalf("%#v", ch)
	}
	if sessionWorkingState(session) != "CURRENT STATE\nkeep going" {
		t.Fatal(sessionWorkingState(session))
	}
}
