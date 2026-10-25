package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV38HistoryIndexAndRetrieval(t *testing.T) {
	p := filepath.Join(t.TempDir(), "session.jsonl")
	j, e := OpenJournal(p, false)
	if e != nil {
		t.Fatal(e)
	}
	entries := []any{
		map[string]any{"role": "user", "content": "Investigate PTY deadlock"},
		map[string]any{"type": "function_call", "name": "exec", "call_id": "id-1", "arguments": `{"command":"ps"}`},
		map[string]any{"type": "function_call_output", "call_id": "id-1", "output": "Found duplicate stdin reader"},
		map[string]any{"type": "function_call", "name": "finish_task", "call_id": "id-2", "arguments": `{"status":"done"}`},
		map[string]any{"type": "function_call_output", "call_id": "id-2", "output": "task status recorded"},
		map[string]any{"role": "user", "content": "Add retry handling"},
		map[string]any{"type": "function_call", "name": "exec", "call_id": "id-3", "arguments": `{"command":"go test"}`},
	}
	for _, v := range entries {
		b, _ := json.Marshal(v)
		if e := j.AppendItem(b); e != nil {
			t.Fatal(e)
		}
	}
	if e := j.Close(); e != nil {
		t.Fatal(e)
	}
	a := &Agent{Cfg: Config{SessionPath: p}}
	if e := a.updateHistoryIndex("Keep changes tested; unfinished retry handling"); e != nil {
		t.Fatal(e)
	}
	idx := loadHistoryIndex(p)
	if idx.WorkingState == "" || len(idx.Sections) < 4 {
		t.Fatalf("bad index: %#v", idx)
	}
	results := a.searchHistory(`{"query":"stdin reader"}`)
	if !strings.Contains(results, "task-1") {
		t.Fatalf("search failed: %s", results)
	}
	old := a.readHistory(`{"section_id":"task-1","start_event":0,"count":20}`)
	if !strings.Contains(old, "duplicate stdin reader") {
		t.Fatalf("read failed: %s", old)
	}
	if !strings.Contains(idx.Sections[0].Status, "completed") {
		t.Fatal("completed task not recognized")
	}
	task, ok := pendingTask(p)
	if !ok || task != "Add retry handling" {
		t.Fatalf("resume %q %v", task, ok)
	}
	// A snapshot checkpoint must not erase the original journal's addressable items.
	view := []json.RawMessage{json.RawMessage(`{"role":"user","content":"summary"}`)}
	j, e = OpenJournal(p, true)
	if e != nil {
		t.Fatal(e)
	}
	if e := j.AppendCheckpoint(view, "semantic"); e != nil {
		t.Fatal(e)
	}
	j.Close()
	again, e := originalEvents(p)
	if e != nil {
		t.Fatal(e)
	}
	if len(again) != len(entries) {
		t.Fatalf("lost event entries %d", len(again))
	}
}
func TestV38HistoryIndexRebuild(t *testing.T) {
	p := filepath.Join(t.TempDir(), "session.jsonl")
	j, e := OpenJournal(p, false)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(map[string]any{"role": "user", "content": "Do thing"})
	j.AppendItem(b)
	done, _ := json.Marshal(map[string]any{"type": "function_call", "name": "finish_task"})
	j.AppendItem(done)
	j.Close()
	_, ok := pendingTask(p)
	if ok {
		t.Fatal("should not resume finished task")
	}
	a := &Agent{Cfg: Config{SessionPath: p}}
	if err := a.updateHistoryIndex(""); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(indexPath(p)); err != nil {
		t.Fatal(err)
	}
	if got := a.searchHistory(`{"query":"thing"}`); !strings.Contains(got, "task-1") {
		t.Fatal(got)
	}
}

func TestV38ResumeRepairsCallsWithoutReplaying(t *testing.T) {
	p := filepath.Join(t.TempDir(), "session.jsonl")
	j, e := OpenJournal(p, false)
	if e != nil {
		t.Fatal(e)
	}
	raw := json.RawMessage(`{"type":"function_call","name":"exec","call_id":"pending1","arguments":"{\"command\":\"touch /tmp/file\"}"}`)
	if e := j.AppendItem(raw); e != nil {
		t.Fatal(e)
	}
	a := &Agent{Cfg: Config{SessionPath: p}, Journal: j, History: j.History}
	if e := a.repairInterruptedCalls(); e != nil {
		t.Fatal(e)
	}
	if len(a.History) != 2 || !strings.Contains(string(a.History[1]), "execution state is unknown") {
		t.Fatalf("missing interrupted output: %v", a.History)
	}
	if e := a.repairInterruptedCalls(); e != nil {
		t.Fatal(e)
	}
	if len(a.History) != 2 {
		t.Fatal("duplicate interrupted output")
	}
	j.Close()
}
