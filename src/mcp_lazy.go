package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	mcpPurposeLimit = 200
	mcpLoadLimit    = 5
	mcpNameHint     = 12
)

func mcpClip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n < 1 {
		return ""
	}
	return string(r[:n-1]) + "…"
}

var mcpPurposeSkip = map[string]bool{
	"get": true, "list": true, "set": true, "create": true, "update": true,
	"delete": true, "remove": true, "add": true, "search": true, "find": true,
	"read": true, "write": true, "the": true, "and": true, "for": true,
	"mcp": true, "tool": true, "tools": true,
}

func mcpServerPurpose(instructions string, tools []mcpToolName) string {
	if s := mcpClip(instructions, mcpPurposeLimit); s != "" {
		return s
	}
	seen := map[string]bool{}
	var words []string
	for _, t := range tools {
		for _, part := range strings.FieldsFunc(strings.ToLower(t.name), func(r rune) bool {
			return r == '_' || r == '-' || r == ' ' || r == '.'
		}) {
			if utf8.RuneCountInString(part) < 3 || mcpPurposeSkip[part] || seen[part] {
				continue
			}
			seen[part] = true
			words = append(words, part)
			if len(words) == 6 {
				return mcpClip("Works with "+strings.Join(words, ", ")+".", mcpPurposeLimit)
			}
		}
	}
	if len(words) == 0 {
		return "Connected tools."
	}
	return mcpClip("Works with "+strings.Join(words, ", ")+".", mcpPurposeLimit)
}

type mcpToolName struct {
	name string
}

func (m *MCPManager) serverCatalog() string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := m.connectedNamesLocked()
	if len(names) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Connected MCP servers. Tool definitions are not loaded. Call mcp_tools with the server name and a short query for the job, then call a returned tool by its exact name. Loaded tools stay until the context is compacted.\n")
	for _, name := range names {
		s := m.sessions[name]
		tools := make([]mcpToolName, 0, len(s.tools))
		for _, t := range s.tools {
			tools = append(tools, mcpToolName{name: t.Name})
		}
		fmt.Fprintf(&b, "- %s: %s\n", name, mcpServerPurpose(s.instructions, tools))
	}
	return strings.TrimSpace(b.String())
}

func (m *MCPManager) connectedNamesLocked() []string {
	names := make([]string, 0, len(m.sessions))
	for name, s := range m.sessions {
		if s != nil && s.status == "connected" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (m *MCPManager) visibleDefs() []map[string]any {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.connectedNamesLocked()) == 0 {
		return nil
	}
	out := []map[string]any{mcpToolsDef()}
	if len(m.loaded) == 0 {
		return out
	}
	names := make([]string, 0, len(m.loaded))
	for name := range m.loaded {
		names = append(names, name)
	}
	sort.Strings(names)
	byName := map[string]map[string]any{}
	for _, d := range m.defs {
		name, _ := d["name"].(string)
		byName[name] = d
	}
	for _, name := range names {
		if d, ok := byName[name]; ok {
			out = append(out, d)
		}
	}
	return out
}

func mcpToolsDef() map[string]any {
	return toolDef("mcp_tools", "Load up to 5 tool definitions from one connected MCP server. server is the server name from the MCP servers list. query says what the job needs. Then call a returned tool by its exact name. Loaded tools stay until the context is compacted.", map[string]any{
		"server": map[string]any{"type": "string", "description": "Server name"},
		"query":  map[string]any{"type": "string", "description": "What the job needs, or empty when the server has only a few tools"},
	}, "server", "query")
}

func (m *MCPManager) clearLoaded() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.loaded = nil
	m.mu.Unlock()
}

func (m *MCPManager) loadTools(server, query string, allow func(string) bool) string {
	if m == nil {
		return "No MCP server is connected."
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if allow == nil {
		allow = func(string) bool { return true }
	}
	sess, name := m.findServerLocked(server)
	if sess == nil {
		return "No MCP server named " + strings.TrimSpace(server) + ". Connected: " + strings.Join(m.connectedNamesLocked(), ", ") + "."
	}
	if sess.status != "connected" {
		return "MCP server " + name + " is not connected."
	}
	var hits []mcpHit
	for _, tool := range sess.tools {
		exposed := m.exposedLocked(name, tool.Name)
		if exposed == "" || !allow(exposed) {
			continue
		}
		hits = append(hits, mcpHit{exposed: exposed, score: mcpSearchScore(tool.Name, tool.Description, query), def: m.defLocked(exposed)})
	}
	if len(hits) == 0 {
		return "MCP server " + name + " has no tools this agent can use."
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].exposed < hits[j].exposed
	})
	q := strings.TrimSpace(query)
	if q == "" && len(hits) > mcpLoadLimit {
		return "Say what you need from " + name + ". Names: " + mcpHitNames(hits) + "."
	}
	picked := hits
	if q != "" {
		picked = nil
		for _, h := range hits {
			if h.score > 0 {
				picked = append(picked, h)
			}
		}
		if len(picked) == 0 {
			return "No tool on " + name + " matched. Names: " + mcpHitNames(hits) + "."
		}
	}
	if len(picked) > mcpLoadLimit {
		picked = picked[:mcpLoadLimit]
	}
	if m.loaded == nil {
		m.loaded = map[string]struct{}{}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Loaded %d tool", len(picked))
	if len(picked) != 1 {
		b.WriteString("s")
	}
	b.WriteString(" from " + name + ". Call one by its exact name.\n")
	for _, h := range picked {
		m.loaded[h.exposed] = struct{}{}
		desc, _ := h.def["description"].(string)
		fmt.Fprintf(&b, "- %s: %s\n", h.exposed, mcpClip(desc, mcpPurposeLimit))
	}
	return strings.TrimSpace(b.String())
}

type mcpHit struct {
	exposed string
	score   int
	def     map[string]any
}

func mcpHitNames(hits []mcpHit) string {
	n := len(hits)
	if n > mcpNameHint {
		n = mcpNameHint
	}
	names := make([]string, 0, n)
	for i := 0; i < n; i++ {
		names = append(names, hits[i].exposed)
	}
	return strings.Join(names, ", ")
}

func mcpSearchScore(name, description, query string) int {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return 0
	}
	hay := strings.ToLower(name + " " + description)
	score := 0
	for _, w := range words {
		if strings.Contains(hay, w) {
			score += 2
		}
	}
	return score
}

func (m *MCPManager) findServerLocked(server string) (*mcpSession, string) {
	server = strings.TrimSpace(server)
	if s := m.sessions[server]; s != nil {
		return s, server
	}
	for name, s := range m.sessions {
		if s != nil && strings.EqualFold(name, server) {
			return s, name
		}
	}
	return nil, ""
}

func (m *MCPManager) exposedLocked(server, tool string) string {
	want := mcpExposedName(server, tool)
	if _, ok := m.routes[want]; ok {
		return want
	}
	for exposed, route := range m.routes {
		if route.Server == server && route.Tool == tool {
			return exposed
		}
	}
	return ""
}

func (m *MCPManager) defLocked(name string) map[string]any {
	for _, d := range m.defs {
		if n, _ := d["name"].(string); n == name {
			return d
		}
	}
	return map[string]any{"name": name, "description": name}
}

func (a *Agent) loadMCPTools(server, query string) string {
	if a == nil || a.MCP == nil {
		return "No MCP server is connected."
	}
	return a.MCP.loadTools(server, query, func(name string) bool {
		return a.instanceToolGranted(name)
	})
}

func (a *Agent) withMCPCatalog(base string) string {
	if a == nil || a.MCP == nil {
		return base
	}
	cat := a.MCP.serverCatalog()
	if cat == "" {
		return base
	}
	return strings.TrimSpace(base) + "\n\n" + cat
}
