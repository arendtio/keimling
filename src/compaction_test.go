package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestSoftCompactionAndJournalResume(t *testing.T) {
	p := filepath.Join(t.TempDir(), "session.jsonl")
	j, e := OpenJournal(p, true)
	if e != nil {
		t.Fatal(e)
	}
	c, _ := json.Marshal(map[string]any{"type": "function_call", "call_id": "123", "name": "exec", "arguments": "{}"})
	o, _ := json.Marshal(map[string]any{"type": "function_call_output", "call_id": "123", "output": strings.Repeat("long logs ", 200)})
	for _, item := range []json.RawMessage{c, o} {
		if e := j.AppendItem(item); e != nil {
			t.Fatal(e)
		}
	}
	compact, n := pruneOutputs(j.History, 0)
	if n != 1 {
		t.Fatalf("compressed %d", n)
	}
	if e := j.AppendCheckpoint(compact, "soft"); e != nil {
		t.Fatal(e)
	}
	j.Close()
	j, e = OpenJournal(p, true)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	if len(j.History) != 2 {
		t.Fatal(len(j.History))
	}
	if len(j.History[1]) >= len(o)/2 {
		t.Fatal("still uncompressed")
	}
}
func TestSafePrefixPreservesPair(t *testing.T) {
	items := []json.RawMessage{
		json.RawMessage(`{"role":"user","content":"hello"}`),
		json.RawMessage(`{"type":"function_call","call_id":"a"}`),
		json.RawMessage(`{"type":"function_call_output","call_id":"a","output":"ok"}`),
		json.RawMessage(`{"role":"user","content":"next"}`),
	}
	if got := safePrefixEnd(items, 2); got != 1 {
		t.Fatalf("boundary=%d", got)
	}
}
