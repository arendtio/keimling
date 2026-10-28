package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSearchThreadsByTitleAndContent(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, session, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	_ = writeThreadMeta(ws.ID, ThreadMeta{ID: th.ID, Title: "Network timeout"})
	user, _ := json.Marshal(map[string]any{"role": "user", "content": "fix the pty deadlock in the shell bridge"})
	row, _ := json.Marshal(map[string]any{"type": "item", "item": json.RawMessage(user)})
	if err := os.WriteFile(session, append(row, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}

	byTitle, err := searchThreads(ws.ID, "timeout")
	if err != nil || len(byTitle) != 1 {
		t.Fatalf("title: %v %#v", err, byTitle)
	}
	byContent, err := searchThreads(ws.ID, "deadlock")
	if err != nil || len(byContent) != 1 {
		t.Fatalf("content: %v %#v", err, byContent)
	}
	miss, err := searchThreads(ws.ID, "xyzzy-no-hit")
	if err != nil || len(miss) != 0 {
		t.Fatalf("miss: %v %#v", err, miss)
	}
}
