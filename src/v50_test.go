package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestV50AcceptUsefulSavings(t *testing.T) {
	if !compactionAcceptable(720313, 525697) {
		t.Fatal("27% should be accepted")
	}
	if compactionAcceptable(1000, 950) {
		t.Fatal("5% should be rejected")
	}
	if compactionAcceptable(1000, 1001) {
		t.Fatal("growth rejected")
	}
	if !strings.Contains(compactionDecision(720313, 525697), "below 30%") {
		t.Fatal("diagnostic missing")
	}
}
func TestV50CandidateViewDoesNotMutateTail(t *testing.T) {
	tail := []json.RawMessage{json.RawMessage(`{"role":"assistant","content":"live"}`)}
	before := string(tail[0])
	v := makeCompactionView("archived work", tail)
	if len(v) != 2 || string(tail[0]) != before || !strings.Contains(string(v[0]), "archived work") {
		t.Fatal("bad checkpoint candidate")
	}
}
func TestV50NoAdditionalPassAtTarget(t *testing.T) {
	a := &Agent{}
	s, size, attempts := a.adaptiveCompaction(nilContext(), "short", 100000, nil)
	if s != "short" || attempts != 0 || size >= 100000 {
		t.Fatalf("%q %d %d", s, size, attempts)
	}
}
func nilContext() context.Context { return context.Background() }
