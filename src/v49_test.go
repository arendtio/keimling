package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSafePrefixHistoricalOrphan(t *testing.T) {
	items := []json.RawMessage{
		json.RawMessage(`{"role":"user","content":"old task"}`),
		json.RawMessage(`{"type":"function_call","call_id":"orphan"}`),
		json.RawMessage(`{"role":"user","content":"later task"}`),
		json.RawMessage(`{"type":"function_call","call_id":"current"}`),
	}
	if got := safePrefixEnd(items, 1); got != 3 {
		t.Fatalf("expected boundary before current call, got %d", got)
	}
	if got := safePrefixEnd(items, 0); got != 3 {
		t.Fatalf("must not compact live call, got %d", got)
	}
}
func TestSafePrefixStillBlocksActiveTool(t *testing.T) {
	items := []json.RawMessage{json.RawMessage(`{"role":"user","content":"task"}`), json.RawMessage(`{"type":"function_call","call_id":"active"}`)}
	if got := safePrefixEnd(items, 0); got != 1 {
		t.Fatalf("expected preceding safe prefix, got %d", got)
	}
}
func TestOutlineBadJSONRejected(t *testing.T) {
	_, err := parseSemanticOutline("I cannot supply JSON", 4)
	if err == nil || !strings.Contains(err.Error(), "invalid semantic outline") {
		t.Fatalf("unexpected err %v", err)
	}
}
