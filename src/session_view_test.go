package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
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
	if len(msgs) != 1 || msgs[0].Role != "tool_group" || msgs[0].Command != "echo hi" || msgs[0].Result != "hi" || msgs[0].Raw != "RAW\nhi" || !msgs[0].Filtered {
		t.Fatalf("%#v", msgs)
	}

	plain, _ := json.Marshal(map[string]any{
		"type": "function_call", "name": "exec", "call_id": "c2",
		"arguments": `{"command":"echo ok"}`,
	})
	plainOut, _ := json.Marshal(map[string]any{
		"type": "function_call_output", "call_id": "c2", "output": "ok\n",
	})
	msgs = sessionMessages([]json.RawMessage{plain, plainOut}, 10)
	if len(msgs) != 1 || msgs[0].Filtered || msgs[0].Raw != "" || msgs[0].Result != "ok" {
		t.Fatalf("unfiltered %#v", msgs)
	}
}

func TestMessageAgentCardShowsTheText(t *testing.T) {
	call, _ := json.Marshal(map[string]any{
		"type": "function_call", "name": "message_agent", "call_id": "c1",
		"arguments": `{"instance_id":"agent-1","text":"Tell me a joke."}`,
	})
	peer, _ := json.Marshal(map[string]any{
		"type": "keimling_peer", "direction": "sent", "peer": "Entertainer 12", "text": "Tell me a joke.",
	})
	out, _ := json.Marshal(map[string]any{
		"type": "function_call_output", "call_id": "c1", "output": "Sent to Entertainer 12: Tell me a joke.",
	})
	msgs := sessionMessages([]json.RawMessage{call, peer, out}, 10)
	if len(msgs) != 2 || msgs[0].Role != "tool_group" || msgs[0].Command != "Tell me a joke." || msgs[0].Result != "Sent to Entertainer 12: Tell me a joke." {
		t.Fatalf("tool card %#v", msgs)
	}
	if msgs[1].Role != "peer" || msgs[1].Name != "To Entertainer 12" {
		t.Fatalf("peer %#v", msgs[1])
	}
	if strings.Contains(msgs[0].Command, "{") {
		t.Fatalf("command shows JSON: %q", msgs[0].Command)
	}
}

func TestSessionMessagesFinishTaskSummaryCard(t *testing.T) {
	call, _ := json.Marshal(map[string]any{
		"type": "function_call", "name": "finish_task", "call_id": "f1",
		"arguments": `{"status":"done","summary":"All checks passed."}`,
	})
	out, _ := json.Marshal(map[string]any{
		"type": "function_call_output", "call_id": "f1", "output": "task status recorded: done",
	})
	msgs := sessionMessages([]json.RawMessage{call, out}, 10)
	if len(msgs) != 2 {
		t.Fatalf("want tool_group + summary, got %#v", msgs)
	}
	if msgs[0].Role != "tool_group" || msgs[0].Name != "finish_task" {
		t.Fatalf("tool card: %#v", msgs[0])
	}
	if msgs[1].Role != "assistant" || !strings.Contains(msgs[1].Text, "Task complete") || !strings.Contains(msgs[1].Text, "All checks passed.") {
		t.Fatalf("summary card: %#v", msgs[1])
	}
	blocked, _ := json.Marshal(map[string]any{
		"type": "function_call", "name": "finish_task", "call_id": "f2",
		"arguments": `{"status":"blocked","summary":"Need API key"}`,
	})
	bout, _ := json.Marshal(map[string]any{
		"type": "function_call_output", "call_id": "f2", "output": "task status recorded: blocked",
	})
	msgs = sessionMessages([]json.RawMessage{blocked, bout}, 10)
	if len(msgs) != 2 || !strings.Contains(msgs[1].Text, "Action needed") || !strings.Contains(msgs[1].Text, "Need API key") {
		t.Fatalf("blocked summary: %#v", msgs)
	}
}

func TestSessionMessagesNoticesAndFailures(t *testing.T) {
	notice, _ := json.Marshal(map[string]any{"type": "keimling_notice", "text": "Run stopped: empty response"})
	pilot, _ := json.Marshal(map[string]any{"role": "user", "content": "[AUTOPILOT SUPERVISOR — not a human authorization or user request] Reassess objective and continue."})
	cont, _ := json.Marshal(map[string]any{"role": "user", "content": "Continue the current plan autonomously. Inspect results, take the next concrete step."})
	call, _ := json.Marshal(map[string]any{"type": "function_call", "name": "exec", "call_id": "c1", "arguments": `{"command":"false"}`})
	out, _ := json.Marshal(map[string]any{"type": "function_call_output", "call_id": "c1", "output": "exit_code=1\nno", "ok": false})
	stopCall, _ := json.Marshal(map[string]any{"type": "function_call", "name": "exec", "call_id": "c2", "arguments": `{"command":"sleep"}`})
	stopOut, _ := json.Marshal(map[string]any{"type": "function_call_output", "call_id": "c2", "output": "INTERRUPTED: previous action's execution state is unknown."})
	msgs := sessionMessages([]json.RawMessage{notice, pilot, cont, call, out, stopCall, stopOut}, 20)
	if len(msgs) != 5 {
		t.Fatalf("len %d %#v", len(msgs), msgs)
	}
	if msgs[0].Role != "system" || msgs[0].Text != "Run stopped: empty response" || msgs[0].Collapsed {
		t.Fatalf("notice %#v", msgs[0])
	}
	if msgs[1].Role != "system" || !msgs[1].Collapsed || !strings.Contains(msgs[1].Text, "AUTOPILOT") {
		t.Fatalf("pilot %#v", msgs[1])
	}
	if msgs[2].Role != "system" || !msgs[2].Collapsed || !strings.Contains(msgs[2].Text, "Continue the current plan") {
		t.Fatalf("continue %#v", msgs[2])
	}
	if msgs[3].Role != "tool_group" || msgs[3].Status != "failed" {
		t.Fatalf("failed tool %#v", msgs[3])
	}
	if msgs[4].Status != "stopped" {
		t.Fatalf("stopped tool %#v", msgs[4])
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
