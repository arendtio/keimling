package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShortOutputNotFiltered(t *testing.T) {
	a := &Agent{}
	original := "exit_code=0\n" + strings.Repeat("line\n", 9)
	if got := a.filterExecResult(context.Background(), "diagnose", `{"command":"printf hello","raw_output":false}`, original); got != original {
		t.Fatal("short output changed")
	}
}
func TestRawOutputBypassesFilter(t *testing.T) {
	a := &Agent{}
	original := "exit_code=0\n" + strings.Repeat("line\n", 40)
	if got := a.filterExecResult(context.Background(), "diagnose", `{"command":"whoami","raw_output":true}`, original); got != original {
		t.Fatal("raw output changed")
	}
}
func TestSnapshotMergeKeepsNewEvents(t *testing.T) {
	j, e := OpenJournal(filepath.Join(t.TempDir(), "session.jsonl"), true)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	a := &Agent{Journal: j, Client: NewClient("unused")}
	for _, s := range []string{strings.Repeat("old task details ", 60), strings.Repeat("old results details ", 60), "new steering"} {
		b, _ := json.Marshal(map[string]any{"role": "user", "content": s})
		if e = j.AppendItem(b); e != nil {
			t.Fatal(e)
		}
	}
	a.History = j.History
	oldTail := string(a.History[2])
	a.Compaction = &compactionJob{end: 2, started: time.Now(), done: make(chan compactionResult, 1)}
	if e := a.adoptCompaction(a.Compaction, compactionResult{summary: "Old task completed."}); e != nil {
		t.Fatal(e)
	}
	if len(a.History) != 2 || string(a.History[1]) != oldTail {
		t.Fatal("post-snapshot steering lost")
	}
}
