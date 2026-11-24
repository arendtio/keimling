package main

import (
	"encoding/json"
	"strings"
)

// mcpToolNamesInHistory returns the MCP tools a thread called before, newest
// first and without duplicates, at most limit. After a restart the lazy list is
// empty; a model that continues a plan would then call a tool that is no longer
// offered and end up poking around.
func mcpToolNamesInHistory(history []json.RawMessage, limit int) []string {
	if limit <= 0 {
		limit = 8
	}
	seen := map[string]bool{}
	out := make([]string, 0, limit)
	for i := len(history) - 1; i >= 0; i-- {
		var item struct {
			Type string `json:"type"`
			Name string `json:"name"`
		}
		if json.Unmarshal(history[i], &item) != nil || item.Type != "function_call" {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if !strings.HasPrefix(name, "mcp__") || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// preloadMCPTools loads the MCP tools this thread used before, so a resumed
// conversation keeps its hands. A tool that cannot be loaded is skipped: the
// prompt tells the model to call find_tools in that case.
func (a *Agent) preloadMCPTools() {
	if a == nil || a.MCP == nil {
		return
	}
	for _, name := range mcpToolNamesInHistory(a.History, 8) {
		server, tool, ok := parseMCPToolName(name)
		if !ok {
			continue
		}
		_ = a.loadMCPTools(server, tool)
	}
}
