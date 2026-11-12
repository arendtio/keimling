package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLowContextKeepsBuiltinSchemasOutUntilLoaded(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, session, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{WorkspaceID: ws.ID, ThreadID: th.ID, SessionPath: session}}
	if a.lowContextOn() {
		t.Fatal("low context starts on")
	}
	if err := a.setLowContext(th.ID, true); err != nil {
		t.Fatal(err)
	}
	if !a.lowContextOn() {
		t.Fatal("low context was not saved")
	}
	full := filterGrantedTools(definitions(false, []map[string]any{{"name": "mcp__mail__list", "description": "List mail"}}), nil)
	narrow := a.narrowBuiltinTools(full)
	raw, err := json.Marshal(narrow)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("Run a shell command")) {
		t.Fatal("builtin schema sent before load")
	}
	if !bytes.Contains(raw, []byte(`"name":"find_tools"`)) && !bytes.Contains(raw, []byte(`"name": "find_tools"`)) {
		t.Fatal("search missing")
	}
	if bytes.Contains(raw, []byte("builtin_tools")) || bytes.Contains(raw, []byte("mcp_tools")) {
		t.Fatal("a second search was sent")
	}
	if !bytes.Contains(raw, []byte("mcp__mail__list")) {
		t.Fatal("mcp tool dropped")
	}
	list := builtinNameList(full)
	if !strings.Contains(list, "exec") || strings.Contains(list, "Run a shell") || strings.Contains(list, "mcp__mail__list") {
		t.Fatalf("name list %q", list)
	}
	prompt := a.withBuiltinList("base")
	if !strings.Contains(prompt, "exec") || strings.Contains(prompt, "Run a shell command") {
		t.Fatal("prompt carried a schema")
	}
	if got := a.findTools(""); strings.Contains(got, "Found") {
		t.Fatalf("empty query loaded tools: %s", got)
	}
	a.MCP = testMCPServer("mail", "Mail.", mcpToolsNamed("list"))
	got := a.findTools("mail")
	if !strings.Contains(got, "mcp__mail__list") {
		t.Fatalf("search missed mcp: %s", got)
	}
	defs := a.mcpToolDefs()
	if len(defs) != 1 || defs[0]["name"] != "mcp__mail__list" {
		t.Fatalf("mcp schema was not loaded: %+v", defs)
	}
	a.Cfg.ThreadID = "other"
	if len(a.mcpToolDefs()) != 0 {
		t.Fatal("another chat inherited the schema")
	}
	a.Cfg.ThreadID = th.ID
	got = a.findTools("read_file")
	if !strings.Contains(got, "read_file") {
		t.Fatalf("search missed builtin: %s", got)
	}
	raw, _ = json.Marshal(a.narrowBuiltinTools(full))
	if !bytes.Contains(raw, []byte("Read a UTF-8 file")) {
		t.Fatal("loaded schema missing")
	}
	a.clearBuiltinLoaded()
	raw, _ = json.Marshal(a.narrowBuiltinTools(full))
	if bytes.Contains(raw, []byte("Read a UTF-8 file")) {
		t.Fatal("schema stayed after compaction")
	}
}

func TestBuiltinToolsDoNotAsk(t *testing.T) {
	if !gateToolCall(false, "ask", "find_tools", `{"query":"exec"}`, false).Allow {
		t.Fatal("ask mode stopped a tool search")
	}
}
