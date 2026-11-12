package main

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func testMCPServer(name, instructions string, tools []mcp.Tool) *MCPManager {
	m := NewMCPManager()
	m.sessions[name] = &mcpSession{
		name:         name,
		status:       "connected",
		instructions: instructions,
		tools:        tools,
	}
	m.rebuildIndex()
	return m
}

func mcpToolsNamed(names ...string) []mcp.Tool {
	out := make([]mcp.Tool, 0, len(names))
	for _, name := range names {
		out = append(out, mcp.Tool{Name: name, Description: name})
	}
	return out
}

func TestMCPPurposePrefersServerInstructions(t *testing.T) {
	long := strings.Repeat("docs ", 80)
	got := mcpServerPurpose(long, []mcpToolName{{name: "list_issues"}})
	if strings.Contains(got, "list_issues") {
		t.Fatalf("purpose listed a tool: %q", got)
	}
	if len([]rune(got)) > mcpPurposeLimit {
		t.Fatalf("purpose %d runes", len([]rune(got)))
	}
	if !strings.HasPrefix(got, "docs") || !strings.HasSuffix(got, "…") {
		t.Fatalf("purpose %q", got)
	}
}

func TestMCPPurposeFromNamesIsAbstract(t *testing.T) {
	got := mcpServerPurpose("", []mcpToolName{{name: "list_issues"}, {name: "create_issue"}, {name: "get_file"}})
	if strings.Contains(got, "list_issues") || strings.Contains(got, "create_issue") {
		t.Fatalf("purpose listed tools: %q", got)
	}
	if !strings.Contains(got, "issues") || !strings.Contains(got, "file") {
		t.Fatalf("purpose %q", got)
	}
}

func TestMCPLoadKeepsAHandfulUntilCompaction(t *testing.T) {
	tools := mcpToolsNamed("list_issues", "create_issue", "close_issue", "list_files", "read_file", "write_note", "search_code", "ping")
	m := testMCPServer("github", "Issues and repository files.", tools)
	visible := m.visibleDefs()
	if len(visible) != 1 || visible[0]["name"] != "mcp_tools" {
		t.Fatalf("before load: %+v", visible)
	}
	cat := m.serverCatalog()
	if !strings.Contains(cat, "github: Issues and repository files.") || !strings.Contains(cat, "mcp_tools") {
		t.Fatalf("catalog %q", cat)
	}
	if strings.Contains(cat, "list_issues") {
		t.Fatalf("catalog listed a tool: %q", cat)
	}
	got := m.loadTools("github", "issue", nil)
	if !strings.Contains(got, "mcp__github__list_issues") || !strings.Contains(got, "mcp__github__create_issue") || strings.Contains(got, "ping") {
		t.Fatalf("load %q", got)
	}
	n := 0
	for _, d := range m.visibleDefs() {
		if name, _ := d["name"].(string); strings.HasPrefix(name, "mcp__") {
			n++
		}
	}
	if n == 0 || n > mcpLoadLimit {
		t.Fatalf("loaded %d", n)
	}
	m.clearLoaded()
	visible = m.visibleDefs()
	if len(visible) != 1 || visible[0]["name"] != "mcp_tools" {
		t.Fatalf("after compact: %+v", visible)
	}
}

func TestMCPSchemasStayOutUntilLoaded(t *testing.T) {
	m := testMCPServer("docs", "Documentation.", mcpToolsNamed("lookup"))
	cat := m.serverCatalog()
	if !strings.Contains(cat, "docs: Documentation.") {
		t.Fatalf("catalog %q", cat)
	}
	if strings.Contains(cat, "lookup") {
		t.Fatalf("catalog listed a tool: %q", cat)
	}
	visible := m.visibleDefs()
	if len(visible) != 1 || visible[0]["name"] != "mcp_tools" {
		t.Fatalf("visible %+v", visible)
	}
}

func TestMCPToolsFollowTheServerGrant(t *testing.T) {
	if !toolNameGranted([]string{"mcp__alpha"}, "mcp_tools") {
		t.Fatal("a server grant did not allow loading tools")
	}
	if toolNameGranted([]string{"exec"}, "mcp_tools") {
		t.Fatal("a command grant allowed loading MCP tools")
	}
	forced, _ := forceApproval("mcp_tools", `{"server":"alpha","query":"issue"}`)
	if forced {
		t.Fatal("loading tools asked for approval")
	}
	if !gateToolCall(false, "ask", "mcp_tools", `{}`, false).Allow {
		t.Fatal("ask mode stopped a tool load")
	}
}
