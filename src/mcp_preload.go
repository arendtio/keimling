package main

import (
	"encoding/json"
	"strings"
	"time"
)

// mcpStartWait bounds how long a run waits for the connect that began with the
// process. A slow start must not hold the chat.
const mcpStartWait = 5 * time.Second

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
// conversation keeps its hands. It returns a note naming the tools that stayed
// out, because the load fails silently while the servers are still connecting.
func (a *Agent) preloadMCPTools() string {
	if a == nil || a.MCP == nil {
		return ""
	}
	names := mcpToolNamesInHistory(a.History, 8)
	if len(names) == 0 {
		return ""
	}
	// The process connects the servers in a goroutine. Right after a restart
	// the tool list and the server lines are still empty, so wait for it.
	a.MCP.waitStart(mcpStartWait)
	var missing []string
	for _, name := range names {
		server, tool, ok := parseMCPToolName(name)
		if !ok {
			continue
		}
		if note := a.loadMCPTools(server, tool); !strings.HasPrefix(note, "Loaded ") {
			missing = append(missing, name)
		} else {
			// The defs are process wide; the thread needs the name to see them.
			a.rememberMCPTool(a.MCP.exposedName(server, tool))
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return "MCP tools this thread used before are not loaded: " + strings.Join(missing, ", ") + ". Call find_tools before using one."
}
