package main

import (
	"fmt"
	"sort"
	"strings"
)

func (a *Agent) lowContextOn() bool {
	if a == nil || strings.TrimSpace(a.Cfg.WorkspaceID) == "" || strings.TrimSpace(a.Cfg.ThreadID) == "" {
		return false
	}
	meta, err := readThreadMeta(a.Cfg.WorkspaceID, a.Cfg.ThreadID)
	if err != nil {
		return false
	}
	return meta.LowContext
}

func (a *Agent) setLowContext(threadID string, on bool) error {
	if a == nil || strings.TrimSpace(a.Cfg.WorkspaceID) == "" {
		return fmt.Errorf("no thread")
	}
	if strings.TrimSpace(threadID) == "" {
		threadID = a.Cfg.ThreadID
	}
	return updateWorkspaceThreadMeta(a.Cfg.WorkspaceID, threadID, func(meta *ThreadMeta) error {
		meta.LowContext = on
		return nil
	})
}

func (a *Agent) clearBuiltinLoaded() {
	if a == nil {
		return
	}
	a.builtinMu.Lock()
	key := a.Cfg.ThreadID
	if a.builtinByThread != nil {
		delete(a.builtinByThread, key)
	}
	if a.mcpByThread != nil {
		delete(a.mcpByThread, key)
	}
	a.builtinMu.Unlock()
}

func (a *Agent) threadToolSet(all map[string]map[string]struct{}) map[string]struct{} {
	if a == nil || all == nil {
		return nil
	}
	return all[a.Cfg.ThreadID]
}

func (a *Agent) mcpLoadedNames() []string {
	if a == nil {
		return nil
	}
	a.builtinMu.Lock()
	defer a.builtinMu.Unlock()
	set := a.threadToolSet(a.mcpByThread)
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	return names
}

func findToolsDef() map[string]any {
	return toolDef("find_tools", "Find up to 5 tools. query is words or an exact name. Searches internal tools and connected MCP servers. Then call a returned tool by its exact name. A tool whose definition was not already in the request stays loaded until the context is compacted.", map[string]any{
		"query": map[string]any{"type": "string", "description": "What the job needs, or an exact tool name"},
	}, "query")
}

func builtinToolName(tool map[string]any) string {
	name, _ := tool["name"].(string)
	return strings.TrimSpace(name)
}

func isMCPListedTool(name string) bool {
	return strings.HasPrefix(name, "mcp__")
}

func isToolSearch(name string) bool {
	return name == "find_tools" || name == "builtin_tools" || name == "mcp_tools"
}

func (a *Agent) narrowBuiltinTools(full []map[string]any) []map[string]any {
	loaded := map[string]struct{}{}
	if a != nil {
		a.builtinMu.Lock()
		for name := range a.threadToolSet(a.builtinByThread) {
			loaded[name] = struct{}{}
		}
		a.builtinMu.Unlock()
	}
	out := []map[string]any{findToolsDef()}
	for _, tool := range full {
		name := builtinToolName(tool)
		if name == "" || isToolSearch(name) {
			continue
		}
		if isMCPListedTool(name) {
			out = append(out, tool)
			continue
		}
		if _, ok := loaded[name]; ok {
			out = append(out, tool)
		}
	}
	return out
}

func builtinNameList(full []map[string]any) string {
	var names []string
	for _, tool := range full {
		name := builtinToolName(tool)
		if name == "" || isToolSearch(name) || isMCPListedTool(name) {
			continue
		}
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

func (a *Agent) withBuiltinList(base string) string {
	if a == nil || !a.lowContextOn() {
		return base
	}
	full := filterGrantedTools(definitions(a.Cfg.Web, a.chatExtraTools()), a.instanceToolGrant())
	names := builtinNameList(full)
	if names == "" {
		return base
	}
	line := "Internal tools, names only: " + names + ". Their definitions are not loaded. Call find_tools with a short query or an exact name. Then call a returned tool by its exact name. Loaded tools stay until the context is compacted."
	return strings.TrimSpace(base) + "\n\n" + line
}

type toolHit struct {
	name  string
	score int
	desc  string
	mcp   bool
}

func (a *Agent) findTools(query string) string {
	if a == nil {
		return "No tools."
	}
	full := filterGrantedTools(definitions(a.Cfg.Web, a.chatExtraTools()), a.instanceToolGrant())
	var hits []toolHit
	q := strings.TrimSpace(query)
	for _, tool := range full {
		name := builtinToolName(tool)
		if name == "" || isToolSearch(name) || isMCPListedTool(name) {
			continue
		}
		desc, _ := tool["description"].(string)
		score := mcpSearchScore(name, desc, q)
		if strings.EqualFold(name, q) {
			score += 100
		}
		hits = append(hits, toolHit{name: name, score: score, desc: desc})
	}
	if a.MCP != nil {
		for _, h := range a.MCP.search(q, func(name string) bool { return a.instanceToolGranted(name) }) {
			desc, _ := h.def["description"].(string)
			score := h.score
			if strings.EqualFold(h.exposed, q) {
				score += 100
			}
			hits = append(hits, toolHit{name: h.exposed, score: score, desc: desc, mcp: true})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].name < hits[j].name
	})
	if q == "" {
		names := make([]string, 0, mcpNameHint)
		for _, h := range hits {
			names = append(names, h.name)
			if len(names) == mcpNameHint {
				break
			}
		}
		return "Say what the job needs, or name a tool. Names: " + strings.Join(names, ", ") + "."
	}
	var picked []toolHit
	for _, h := range hits {
		if h.score > 0 {
			picked = append(picked, h)
		}
	}
	if len(picked) == 0 {
		return "No tool matched."
	}
	if len(picked) > mcpLoadLimit {
		picked = picked[:mcpLoadLimit]
	}
	reduce := a.lowContextOn()
	a.builtinMu.Lock()
	key := a.Cfg.ThreadID
	if a.builtinByThread == nil {
		a.builtinByThread = map[string]map[string]struct{}{}
	}
	if a.mcpByThread == nil {
		a.mcpByThread = map[string]map[string]struct{}{}
	}
	for _, h := range picked {
		if h.mcp {
			set := a.mcpByThread[key]
			if set == nil {
				set = map[string]struct{}{}
				a.mcpByThread[key] = set
			}
			set[h.name] = struct{}{}
			continue
		}
		if reduce {
			set := a.builtinByThread[key]
			if set == nil {
				set = map[string]struct{}{}
				a.builtinByThread[key] = set
			}
			set[h.name] = struct{}{}
		}
	}
	a.builtinMu.Unlock()
	var b strings.Builder
	fmt.Fprintf(&b, "Found %d tool", len(picked))
	if len(picked) != 1 {
		b.WriteString("s")
	}
	b.WriteString(". Call one by its exact name.\n")
	for _, h := range picked {
		fmt.Fprintf(&b, "- %s: %s\n", h.name, mcpClip(h.desc, mcpPurposeLimit))
	}
	return strings.TrimSpace(b.String())
}
