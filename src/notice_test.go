package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoteSurvivesReloadAndReachesModel(t *testing.T) {
	j, err := OpenJournal(filepath.Join(t.TempDir(), "session.jsonl"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = j.Close() })
	var saw string
	a := &Agent{Journal: j, History: j.History, Emit: func(kind, text string) {
		if kind == "message" {
			saw = text
		}
	}}
	call := map[string]any{"type": "function_call", "name": "exec", "call_id": "c1", "arguments": `{"command":"sleep 1"}`}
	if err := a.add(call); err != nil {
		t.Fatal(err)
	}
	err = context.Canceled
	a.finishRun(&err)
	if !strings.Contains(saw, "Stopped") {
		t.Fatalf("emit %s", saw)
	}
	msgs := sessionMessages(a.History, 20)
	if len(msgs) != 2 || msgs[0].Status != "stopped" || msgs[1].Text != "Stopped" {
		t.Fatalf("%#v", msgs)
	}
	norm, err := normalizeResponseInput(a.History)
	if err != nil {
		t.Fatal(err)
	}
	joined := string(norm[len(norm)-1])
	if !strings.Contains(joined, `"role":"developer"`) || !strings.Contains(joined, "Stopped") {
		t.Fatalf("model input %s", joined)
	}
}

func TestMarkToolOK(t *testing.T) {
	if markToolOK("exec", "exit_code=0\nok", "") {
		// success
	} else {
		t.Fatal("exit 0")
	}
	if markToolOK("exec", "filtered", "exit_code=2\nno") {
		t.Fatal("exit 2 should fail")
	}
	if markToolOK("read_file", "DENIED by user", "") {
		t.Fatal("denied")
	}
	if !markToolOK("read_file", "hello", "") {
		t.Fatal("plain read")
	}
}
