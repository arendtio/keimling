package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestMCPExposedName(t *testing.T) {
	got := mcpExposedName("my server", "list-files!")
	if got != "mcp__my_server__list-files_" {
		t.Fatalf("got %q", got)
	}
	if !isMCPTool(got) {
		t.Fatal("expected isMCPTool")
	}
	srv, tool, ok := parseMCPToolName(got)
	if !ok || srv != "my_server" || tool != "list-files_" {
		t.Fatalf("parse: %q %q %v", srv, tool, ok)
	}
}

func TestMCPConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	servers := map[string]MCPServerConfig{
		"demo":   {Command: "echo", Args: []string{"hi"}, Env: map[string]string{"A": "1"}},
		"remote": {URL: "http://127.0.0.1:9/mcp", Type: "sse"},
	}
	if err := writeMCPFile(path, servers); err != nil {
		t.Fatal(err)
	}
	got, err := readMCPFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["demo"].Command != "echo" || len(got["demo"].Args) != 1 || got["remote"].Type != "sse" {
		t.Fatalf("round trip: %+v", got)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0077 != 0 {
		t.Fatalf("mcp.json should be 0600, got %o", st.Mode().Perm())
	}
}

func TestLoadMCPServerConfigsMerge(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"mcpServers":{"fromConfig":{"command":"true"}}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	side := filepath.Join(dir, "mcp.json")
	if err := writeMCPFile(side, map[string]MCPServerConfig{
		"fromConfig": {Command: "override"},
		"side":       {URL: "http://example/mcp"},
	}); err != nil {
		t.Fatal(err)
	}
	got := LoadMCPServerConfigs(cfgPath)
	if got["fromConfig"].Command != "override" {
		t.Fatalf("expected side mcp.json to override config.json, got %+v", got["fromConfig"])
	}
	if got["side"].URL == "" {
		t.Fatal("missing side server")
	}
}

func TestMCPServerConfigValidate(t *testing.T) {
	if err := (MCPServerConfig{}).validate(); err == nil {
		t.Fatal("expected error")
	}
	if err := (MCPServerConfig{Command: "npx"}).validate(); err != nil {
		t.Fatal(err)
	}
	if err := (MCPServerConfig{URL: "http://x"}).validate(); err != nil {
		t.Fatal(err)
	}
}

func TestMCPManagerInProcess(t *testing.T) {
	srv := server.NewMCPServer("test-server", "1.0.0", server.WithToolCapabilities(true))
	srv.AddTool(
		mcp.NewTool("echo", mcp.WithDescription("echo text"), mcp.WithString("text", mcp.Required())),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(req.GetString("text", "")), nil
		},
	)
	cli, err := mcpclient.NewInProcessClient(srv)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	ctx := context.Background()
	if err := cli.Start(ctx); err != nil {
		t.Fatal(err)
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "keimling-test", Version: "0"}
	if _, err := cli.Initialize(ctx, initReq); err != nil {
		t.Fatal(err)
	}
	listed, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil || listed == nil || len(listed.Tools) == 0 {
		t.Fatalf("list tools: %v %+v", err, listed)
	}

	m := NewMCPManager()
	m.sessions["demo"] = &mcpSession{
		name:   "demo",
		cfg:    MCPServerConfig{Command: "inprocess"},
		client: cli,
		tools:  listed.Tools,
		status: "connected",
	}
	m.rebuildIndex()

	defs := m.ToolDefs()
	if len(defs) != 1 {
		t.Fatalf("defs: %d", len(defs))
	}
	name, _ := defs[0]["name"].(string)
	if name != "mcp__demo__echo" {
		t.Fatalf("tool name %q", name)
	}
	out := m.Call(ctx, "mcp__demo__echo", `{"text":"hello"}`)
	if out != "hello" {
		t.Fatalf("call result %q", out)
	}
	st := m.Status()
	if len(st) != 1 || st[0].Status != "connected" || len(st[0].Tools) != 1 {
		t.Fatalf("status %+v", st)
	}
}

func TestForceApprovalMCP(t *testing.T) {
	forced, why := forceApproval("mcp__demo__echo", `{}`)
	if !forced || why != "MCP tool call" {
		t.Fatalf("got %v %q", forced, why)
	}
}
