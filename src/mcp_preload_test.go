package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMCPToolNamesInHistoryNewestFirstWithoutDuplicates(t *testing.T) {
	history := []json.RawMessage{
		json.RawMessage(`{"type":"message","role":"user","content":"hi"}`),
		json.RawMessage(`{"type":"function_call","name":"exec"}`),
		json.RawMessage(`{"type":"function_call","name":"mcp__Operations__project_status"}`),
		json.RawMessage(`{"type":"function_call","name":"mcp__Operations__query_logs"}`),
		json.RawMessage(`{"type":"function_call","name":"mcp__Operations__project_status"}`),
	}
	got := mcpToolNamesInHistory(history, 5)
	want := []string{"mcp__Operations__project_status", "mcp__Operations__query_logs"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMCPToolNamesInHistoryHonoursTheLimit(t *testing.T) {
	history := []json.RawMessage{
		json.RawMessage(`{"type":"function_call","name":"mcp__a__one"}`),
		json.RawMessage(`{"type":"function_call","name":"mcp__a__two"}`),
		json.RawMessage(`{"type":"function_call","name":"mcp__a__three"}`),
	}
	if got := mcpToolNamesInHistory(history, 2); len(got) != 2 || got[0] != "mcp__a__three" {
		t.Fatalf("limit: %v", got)
	}
	if got := mcpToolNamesInHistory(nil, 3); len(got) != 0 {
		t.Fatalf("empty: %v", got)
	}
}

func TestPreloadMCPToolsNeedsAManager(t *testing.T) {
	a := &Agent{}
	a.preloadMCPTools() // must not panic without a manager
}
