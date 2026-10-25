package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResumeAndNewChat(t *testing.T) {
	p := filepath.Join(t.TempDir(), "session.jsonl")
	j, e := OpenJournal(p, true)
	if e != nil {
		t.Fatal(e)
	}
	if e = j.AppendItem([]byte(`{"role":"user","content":"hello"}`)); e != nil {
		t.Fatal(e)
	}
	j.Close()
	j, e = OpenJournal(p, true)
	if e != nil {
		t.Fatal(e)
	}
	if len(j.History) != 1 {
		t.Fatalf("history=%d", len(j.History))
	}
	if e = j.NewChat(); e != nil {
		t.Fatal(e)
	}
	if len(j.History) != 0 {
		t.Fatal("old items remain")
	}
	a, _ := filepath.Glob(p + ".*.bak")
	if len(a) != 1 {
		t.Fatal("archive absent")
	}
	if b, e := os.ReadFile(a[0]); e != nil || len(b) == 0 {
		t.Fatal("archive empty")
	}
	j.Close()
}
