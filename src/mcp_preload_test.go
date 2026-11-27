package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
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
	if note := a.preloadMCPTools(); note != "" { // must not panic without a manager
		t.Fatalf("note %q", note)
	}
}

func mcpHistoryWith(names ...string) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(names))
	for _, name := range names {
		out = append(out, json.RawMessage(`{"type":"function_call","name":"`+name+`"}`))
	}
	return out
}

func TestPreloadMCPToolsNamesWhatStayedOut(t *testing.T) {
	m := NewMCPManager()
	m.sessions["Operations"] = &mcpSession{name: "Operations", status: "starting"}
	a := &Agent{MCP: m, History: mcpHistoryWith("mcp__Operations__project_update_status")}
	note := a.preloadMCPTools()
	if !strings.Contains(note, "mcp__Operations__project_update_status") || !strings.Contains(note, "find_tools") {
		t.Fatalf("note %q", note)
	}
}

func TestPreloadMCPToolsLoadsAToolTheThreadUsed(t *testing.T) {
	m := testMCPServer("Operations", "Ops.", mcpToolsNamed("project_update_status"))
	a := &Agent{MCP: m, History: mcpHistoryWith("mcp__Operations__project_update_status")}
	if note := a.preloadMCPTools(); note != "" {
		t.Fatalf("note %q", note)
	}
	if len(a.mcpToolDefs()) != 1 {
		t.Fatalf("defs %v", a.mcpToolDefs())
	}
}

// The tools are memory only, so a run that starts while the servers are still
// connecting has to wait for the connect instead of reading an empty list.
func TestPreloadMCPToolsWaitsForTheRunningStart(t *testing.T) {
	m := testMCPServer("Operations", "Ops.", mcpToolsNamed("project_update_status"))
	a := &Agent{MCP: m, History: mcpHistoryWith("mcp__Operations__project_update_status")}
	m.beginStart()
	go func() {
		time.Sleep(50 * time.Millisecond)
		m.finishStart()
	}()
	start := time.Now()
	if note := a.preloadMCPTools(); note != "" {
		t.Fatalf("note %q", note)
	}
	if waited := time.Since(start); waited < 40*time.Millisecond {
		t.Fatalf("returned after %v, did not wait for the start", waited)
	}
}
