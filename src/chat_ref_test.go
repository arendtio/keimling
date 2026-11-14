package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestChatRefReadsAnotherWorkspaceAndStaysReadOnly(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(data); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ws.Name = "Alpha"
	if err := writeWorkspaceMeta(ws); err != nil {
		t.Fatal(err)
	}
	th, session, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := readThreadMeta(ws.ID, th.ID)
	if err != nil {
		t.Fatal(err)
	}
	meta.Title = "Notes"
	if err := writeThreadMeta(ws.ID, meta); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(session, []byte("UNIQUE-OTHER-CHAT\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ws.Hidden = true
	if err := writeWorkspaceMeta(ws); err != nil {
		t.Fatal(err)
	}

	ref := "chat:Alpha/Notes"
	got := executeTool(context.Background(), "read_file", `{"path":"`+ref+`","start_line":0,"end_line":0}`)
	if !strings.Contains(got, "UNIQUE-OTHER-CHAT") || !strings.Contains(got, ref) || strings.Contains(got, session) {
		t.Fatalf("read: %s", got)
	}
	byID := "chat:" + ws.ID + "/" + th.ID
	got = executeTool(context.Background(), "read_file", `{"path":"`+byID+`","start_line":0,"end_line":0}`)
	if !strings.Contains(got, "UNIQUE-OTHER-CHAT") {
		t.Fatalf("id read: %s", got)
	}

	before, err := os.ReadFile(session)
	if err != nil {
		t.Fatal(err)
	}
	edit, _ := json.Marshal(map[string]any{"path": session, "old_string": "UNIQUE-OTHER-CHAT", "new_string": "CHANGED", "replace_all": false})
	if msg := executeTool(context.Background(), "edit_file", string(edit)); !strings.Contains(msg, "read-only") {
		t.Fatalf("edit path: %s", msg)
	}
	edit, _ = json.Marshal(map[string]any{"path": ref, "old_string": "UNIQUE-OTHER-CHAT", "new_string": "CHANGED", "replace_all": false})
	if msg := executeTool(context.Background(), "edit_file", string(edit)); !strings.Contains(msg, "read-only") {
		t.Fatalf("edit ref: %s", msg)
	}
	after, err := os.ReadFile(session)
	if err != nil || string(after) != string(before) {
		t.Fatalf("journal changed: %s", after)
	}

	a := &Agent{}
	code := `return input`
	got = a.runDataTool(`{"input_ref":"` + ref + `","inputs":[],"code":` + jsonString(code) + `,"timeout_ms":1000,"output_mode":"inline"}`)
	if !strings.Contains(got, "UNIQUE-OTHER-CHAT") {
		t.Fatalf("data: %s", got)
	}

	second, session2, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	meta2, err := readThreadMeta(ws.ID, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	meta2.Title = "Notes"
	if err := writeThreadMeta(ws.ID, meta2); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(session2, []byte("other\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = resolveChatJournal(ref)
	if err == nil || !strings.Contains(err.Error(), th.ID) || !strings.Contains(err.Error(), second.ID) {
		t.Fatalf("ambiguous: %v", err)
	}
}
