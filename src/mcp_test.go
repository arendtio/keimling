package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mark3labs/mcp-go/server/servertest"
)

func TestMCPCallTimeoutCoversALongJob(t *testing.T) {
	if mcpCallTimeout < 60*time.Minute {
		t.Fatalf("mcp call timeout %s cuts off a long server job", mcpCallTimeout)
	}
}

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

func TestMCPBearerHeader(t *testing.T) {
	if got := mcpAuthorizationHeader("secret-token"); got != "Bearer secret-token" {
		t.Fatalf("got %q", got)
	}
	if got := mcpAuthorizationHeader("Bearer secret-token"); got != "Bearer secret-token" {
		t.Fatalf("double prefix: %q", got)
	}
	if got := mcpAuthorizationHeader("  "); got != "" {
		t.Fatalf("blank: %q", got)
	}
	cfg := MCPServerConfig{}
	cfg.setBearer("secret-token")
	if cfg.Headers["Authorization"] != "Bearer secret-token" {
		t.Fatalf("%v", cfg.Headers)
	}
}

func TestRemoteURLDetectsSSE(t *testing.T) {
	mcpSrv := server.NewMCPServer("sse-test", "1.0.0", server.WithToolCapabilities(true))
	ts := servertest.NewTestServer(mcpSrv)
	defer ts.Close()
	m := NewMCPManager()
	defer m.Close()
	m.ConnectAll(context.Background(), map[string]MCPServerConfig{
		"legacy": {URL: ts.URL + "/sse"},
	})
	st := m.Status()
	if len(st) != 1 || st[0].Status != "connected" || st[0].Transport != "sse" {
		t.Fatalf("%+v", st)
	}
}

func TestRemoteURLUsesStreamableHTTP(t *testing.T) {
	mcpSrv := server.NewMCPServer("http-test", "1.0.0", server.WithToolCapabilities(true))
	ts := servertest.NewTestStreamableHTTPServer(mcpSrv)
	defer ts.Close()
	m := NewMCPManager()
	defer m.Close()
	m.ConnectAll(context.Background(), map[string]MCPServerConfig{
		"modern": {URL: ts.URL},
	})
	st := m.Status()
	if len(st) != 1 || st[0].Status != "connected" || st[0].Transport != "streamableHttp" {
		t.Fatalf("%+v", st)
	}
}

func TestRemoteURLAuthorizationRequiredDoesNotProbeSSE(t *testing.T) {
	var gets int
	var auth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			gets++
		}
		if auth == "" {
			auth = r.Header.Get("Authorization")
		}
		http.Error(w, "no", http.StatusUnauthorized)
	}))
	defer ts.Close()
	cfg := MCPServerConfig{URL: ts.URL}
	cfg.setBearer("secret-token")
	m := NewMCPManager()
	defer m.Close()
	m.ConnectAll(context.Background(), map[string]MCPServerConfig{"locked": cfg})
	st := m.Status()
	if len(st) != 1 || st[0].Status != "error" || !strings.Contains(st[0].Error, "authorization required") {
		t.Fatalf("%+v", st)
	}
	if auth != "Bearer secret-token" {
		t.Fatalf("auth %q", auth)
	}
	if gets != 0 {
		t.Fatalf("401 should stay on HTTP, gets=%d", gets)
	}
}

func TestRemoveMCPServerFromEveryConfigFile(t *testing.T) {
	withTempAgentHomes(t)
	dir := t.TempDir()
	t.Chdir(dir)
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"listen":":9","mcpServers":{"fromConfig":{"command":"cfg"},"shadow":{"command":"old"}}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpsertUserMCPServer("useronly", MCPServerConfig{Command: "user"}); err != nil {
		t.Fatal(err)
	}
	if err := UpsertUserMCPServer("shadow", MCPServerConfig{Command: "user-shadow"}); err != nil {
		t.Fatal(err)
	}
	cursor := filepath.Join(dir, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(cursor), 0o700); err != nil {
		t.Fatal(err)
	}
	raw := `{"mcpServers":{"blender":{"command":"blender"},"keep":{"command":"stay","extraFlag":true},"shadow":{"command":"cursor-wins"}}}`
	if err := os.WriteFile(cursor, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	_, sources, _ := loadMCPServers(cfgPath)
	if sources["fromConfig"] != "config" || sources["useronly"] != "user" || sources["blender"] != "cursor" || sources["shadow"] != "cursor" {
		t.Fatalf("sources: %#v", sources)
	}
	for _, name := range []string{"fromConfig", "blender", "shadow", "useronly"} {
		if err := RemoveMCPServer(name, cfgPath); err != nil {
			t.Fatal(err)
		}
	}
	if err := RemoveMCPServer("missing", cfgPath); err == nil {
		t.Fatal("expected missing server to fail")
	}
	got := LoadMCPServerConfigs(cfgPath)
	for _, name := range []string{"fromConfig", "blender", "shadow", "useronly"} {
		if _, ok := got[name]; ok {
			t.Fatalf("%s still loaded", name)
		}
	}
	if got["keep"].Command != "stay" {
		t.Fatalf("keep: %+v", got["keep"])
	}
	b, err := os.ReadFile(cursor)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "extraFlag") || strings.Contains(string(b), "blender") {
		t.Fatalf("cursor file: %s", b)
	}
	cfgb, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgb), `":9"`) && !strings.Contains(string(cfgb), ":9") {
		t.Fatalf("config.json: %s", cfgb)
	}
	if strings.Contains(string(cfgb), "fromConfig") || strings.Contains(string(cfgb), "shadow") {
		t.Fatalf("config.json still lists a removed server: %s", cfgb)
	}
}

func TestRemoveMCPServerLeavesPluginSource(t *testing.T) {
	withTempAgentHomes(t)
	t.Chdir(t.TempDir())
	root := t.TempDir()
	writePluginFixture(t, root)
	if _, err := addPluginSource(root); err != nil {
		t.Fatal(err)
	}
	if err := RemoveMCPServer("review", ""); err == nil {
		t.Fatal("expected plugin server to stay")
	}
	if _, ok := LoadMCPServerConfigs("")["review"]; !ok {
		t.Fatal("plugin server was removed")
	}
	_, sources, names := loadMCPServers("")
	if sources["review"] != "plugin" || names["review"] != "review-kit" {
		t.Fatalf("source %q name %q", sources["review"], names["review"])
	}
}

func TestMCPToolAllowUpdatesWinningFile(t *testing.T) {
	withTempAgentHomes(t)
	dir := t.TempDir()
	t.Chdir(dir)
	cursor := filepath.Join(dir, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(cursor), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cursor, []byte(`{"mcpServers":{"blender":{"command":"blender"},"keep":{"command":"stay","extraFlag":true}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := setMCPToolAllow("blender", "mcp__blender__list", true, ""); err != nil {
		t.Fatal(err)
	}
	if !mcpToolAlwaysAllowed("", "mcp__blender__list") {
		t.Fatal("mark was not visible")
	}
	b, err := os.ReadFile(cursor)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "allow_tools") || !strings.Contains(string(b), "extraFlag") {
		t.Fatalf("cursor file: %s", b)
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
