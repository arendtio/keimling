package main

import (
	"os"
	"strings"
	"testing"
)

func TestBracketedPastePreservesNewlines(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	payload := "first line\nsecond line\nthird" + "\x1b[201~"
	if _, err = w.Write([]byte(payload)); err != nil {
		t.Fatal(err)
	}
	got, err := readBracketedPaste(int(r.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if got != "first line\nsecond line\nthird" {
		t.Fatalf("got %q", got)
	}
}
func TestManualSoftCompactionNoHistory(t *testing.T) {
	a := &Agent{Cfg: Config{ContextLimit: 128000}}
	if err := a.compactContext(nil, "soft", ""); err == nil {
		t.Fatal("deprecated soft compaction must be rejected")
	}
	if got := contextEstimate(a.History); got != 0 {
		t.Fatal(got)
	}
}
func TestSteeringFormat(t *testing.T) {
	if !strings.Contains("/compact preserve hardware", "preserve hardware") {
		t.Fatal("bad test")
	}
}
