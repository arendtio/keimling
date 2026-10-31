package main

import (
	"fmt"
	"os"
	"strings"
)

// agentTool is one row in the instance tool grant.
type agentTool struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

// agentToolGroup is one topic in the grant table. Read and write are separate
// choices. The names are catalog tool ids; the UI shows their labels.
type agentToolGroup struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Read  []string `json:"read"`
	Write []string `json:"write"`
}

// agentToolCatalog is the stable set a user can grant. MCP tools are one row.
func agentToolCatalog() []agentTool {
	return []agentTool{
		{Name: "exec", Label: "Run commands"},
		{Name: "read_file", Label: "Read files"},
		{Name: "edit_file", Label: "Edit files"},
		{Name: "write_file", Label: "Write files"},
		{Name: "share_file", Label: "Share a file"},
		{Name: "show_artifact", Label: "Show a preview"},
		{Name: "search_history", Label: "Search history"},
		{Name: "read_history", Label: "Read history"},
		{Name: "activate_skill", Label: "Use a skill"},
		{Name: "read_skill_file", Label: "Read a skill file"},
		{Name: "delegate", Label: "Ask another model"},
		{Name: "update_plan", Label: "Keep a plan"},
		{Name: "finish_task", Label: "Finish a task"},
		{Name: "web_search", Label: "Search the web"},
		{Name: "list_credentials", Label: "List credentials"},
		{Name: "change_workspace", Label: "Switch workspace"},
		{Name: "set_alarm", Label: "Set an alarm"},
		{Name: "list_alarms", Label: "List alarms"},
		{Name: "cancel_alarm", Label: "Cancel an alarm"},
		{Name: "list_agents", Label: "List agents"},
		{Name: "message_agent", Label: "Message an agent"},
	}
}

// agentToolGroups sorts the catalog by kind of work. A write choice covers
// every tool named on that side; edit and write are one file-writing choice.
func agentToolGroups() []agentToolGroup {
	return []agentToolGroup{
		{ID: "files", Label: "Files", Read: []string{"read_file"}, Write: []string{"edit_file", "write_file"}},
		{ID: "share", Label: "Share", Read: []string{}, Write: []string{"share_file", "show_artifact"}},
		{ID: "commands", Label: "Commands", Read: []string{}, Write: []string{"exec"}},
		{ID: "history", Label: "History", Read: []string{"search_history", "read_history"}, Write: []string{}},
		{ID: "skills", Label: "Skills", Read: []string{"read_skill_file"}, Write: []string{"activate_skill"}},
		{ID: "plan", Label: "Plan", Read: []string{}, Write: []string{"update_plan", "finish_task"}},
		{ID: "web", Label: "Web", Read: []string{"web_search"}, Write: []string{}},
		{ID: "credentials", Label: "Credentials", Read: []string{"list_credentials"}, Write: []string{}},
		{ID: "workspace", Label: "Workspace", Read: []string{}, Write: []string{"change_workspace"}},
		{ID: "alarms", Label: "Alarms", Read: []string{"list_alarms"}, Write: []string{"set_alarm", "cancel_alarm"}},
		{ID: "agents", Label: "Agents", Read: []string{"list_agents"}, Write: []string{"message_agent"}},
		{ID: "models", Label: "Models", Read: []string{}, Write: []string{"delegate"}},
	}
}

// mcpServerGrantID is the stored choice for one MCP server. It grants every
// tool that server exposes. "mcp" remains the older choice that grants every server.
func mcpServerGrantID(server string) string {
	return "mcp__" + sanitizeMCPToken(server)
}

func isMCPServerGrant(name string) bool {
	name = strings.TrimSpace(name)
	if name == "mcp" {
		return true
	}
	if !strings.HasPrefix(name, "mcp__") {
		return false
	}
	rest := strings.TrimPrefix(name, "mcp__")
	return rest != "" && !strings.Contains(rest, "__") && rest == sanitizeMCPToken(rest)
}

func knownAgentTool(name string) bool {
	name = strings.TrimSpace(name)
	if isMCPServerGrant(name) {
		return true
	}
	for _, t := range agentToolCatalog() {
		if t.Name == name {
			return true
		}
	}
	return false
}

// definitionAllowsName reports whether a saved definition permits this tool
// name or MCP-server choice. A nil definition permits the catalog and any server.
func definitionAllowsName(def *[]string, name string) bool {
	if def == nil {
		return knownAgentTool(name)
	}
	for _, g := range *def {
		if g == name {
			return true
		}
		if g == "mcp" && isMCPServerGrant(name) {
			return true
		}
	}
	return false
}

// instanceToolGrant returns nil when this process is not an agent instance or
// the instance has never saved a tool choice. Nil means every tool. A non-nil
// slice, including empty, is the saved choice.
func (a *Agent) instanceToolGrant() []string {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return nil
	}
	m, err := readAgentInstanceMeta(a.Cfg.AgentInstanceID)
	if err != nil {
		return nil
	}
	return effectiveInstanceTools(m)
}

// effectiveInstanceTools is the definition's tool choice, narrowed by an
// instance override. Nil means every tool. A saved instance choice cannot
// add a tool the definition does not grant.
func effectiveInstanceTools(m AgentInstanceMeta) []string {
	var def *[]string
	if rec, err := readAgentRecipe(m.RecipeID); err == nil {
		def = rec.Tools
	}
	return intersectToolGrants(def, m.Tools)
}

func intersectToolGrants(def, inst *[]string) []string {
	if def == nil && inst == nil {
		return nil
	}
	if inst == nil {
		out := append([]string(nil), (*def)...)
		return out
	}
	if def == nil {
		out := append([]string(nil), (*inst)...)
		return out
	}
	out := []string{}
	for _, n := range *inst {
		if definitionAllowsName(def, n) {
			out = append(out, n)
		}
	}
	return out
}

// mcpGrantIDs is the server choices in a saved tool list.
// all is true when every server is granted (nil list, or the legacy "mcp" choice).
func mcpGrantIDs(tools *[]string) (all bool, ids []string) {
	if tools == nil {
		return true, []string{}
	}
	ids = []string{}
	for _, n := range *tools {
		if n == "mcp" {
			return true, []string{}
		}
		if isMCPServerGrant(n) {
			ids = append(ids, n)
		}
	}
	return false, ids
}

func recipeToolNames(r AgentRecipe) []string {
	if r.Tools == nil {
		names := make([]string, 0, len(agentToolCatalog()))
		for _, t := range agentToolCatalog() {
			names = append(names, t.Name)
		}
		return names
	}
	return append([]string(nil), (*r.Tools)...)
}

func cleanToolNames(tools []string) ([]string, error) {
	next := make([]string, 0, len(tools))
	got := map[string]bool{}
	for _, name := range tools {
		name = strings.TrimSpace(name)
		if name == "" || got[name] {
			continue
		}
		if !knownAgentTool(name) {
			return nil, fmt.Errorf("unknown tool %q", name)
		}
		got[name] = true
		next = append(next, name)
	}
	return next, nil
}

func sameToolSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	got := map[string]bool{}
	for _, n := range a {
		got[n] = true
	}
	for _, n := range b {
		if !got[n] {
			return false
		}
	}
	return true
}

func (a *Agent) instanceToolGranted(name string) bool {
	return toolNameGranted(a.instanceToolGrant(), name)
}

func toolNameGranted(grant []string, name string) bool {
	if grant == nil {
		return true
	}
	name = strings.TrimSpace(name)
	for _, g := range grant {
		if g == name {
			return true
		}
		if g == "mcp" && strings.HasPrefix(name, "mcp__") {
			return true
		}
		if isMCPServerGrant(g) && g != "mcp" && strings.HasPrefix(name, g+"__") {
			return true
		}
	}
	return false
}

func filterGrantedTools(tools []map[string]any, grant []string) []map[string]any {
	if grant == nil {
		return tools
	}
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		name, _ := t["name"].(string)
		if toolNameGranted(grant, name) {
			out = append(out, t)
		}
	}
	return out
}

func (a *Agent) agentToolPrompt() string {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return ""
	}
	grant := a.instanceToolGrant()
	names := grant
	if names == nil {
		names = make([]string, 0, len(agentToolCatalog()))
		for _, t := range agentToolCatalog() {
			names = append(names, t.Name)
		}
	}
	list := "(none)"
	if len(names) > 0 {
		list = strings.Join(names, ", ")
	}
	return "Tools this agent instance may use: " + list + ". Other tools are not available. When the user asks which tools you have, answer in text with these names. For a later or repeating wake, call set_alarm when it is in the list. Do not invent a shell script or a file for that."
}

// syncInstanceWorkspace reloads the saved workspace grant into the running worker.
func (a *Agent) syncInstanceWorkspace() {
	if a == nil {
		return
	}
	id := strings.TrimSpace(a.Cfg.AgentInstanceID)
	if id == "" {
		return
	}
	m, err := readAgentInstanceMeta(id)
	if err != nil {
		return
	}
	a.Cfg.AllowedWorkspaces = append([]string(nil), m.Workspaces...)
	cur := strings.TrimSpace(m.CurrentWorkspace)
	if cur == "" || !agentInstanceAllowsWorkspace(m, cur) {
		if len(m.Workspaces) == 0 {
			return
		}
		cur = m.Workspaces[0]
		m.CurrentWorkspace = cur
		_ = writeAgentInstanceMeta(m)
	}
	if cur == a.Cfg.WorkspaceID {
		return
	}
	meta, err := readWorkspaceMeta(cur)
	if err != nil {
		return
	}
	if err := os.Chdir(meta.Root); err != nil {
		return
	}
	a.Cfg.WorkspaceID = cur
	a.Cfg.WorkspaceRoot = meta.Root
}

// setAgentInstanceAccess saves workspaces and an optional narrower tool set.
// A nil tools pointer means the instance follows the definition.
// A set equal to the definition is stored as follow. A tool the definition
// does not grant is rejected.
func setAgentInstanceAccess(id string, workspaces []string, tools *[]string) (AgentInstanceMeta, error) {
	m, err := readAgentInstanceMeta(id)
	if err != nil {
		return AgentInstanceMeta{}, err
	}
	clean := make([]string, 0, len(workspaces))
	seen := map[string]bool{}
	for _, w := range workspaces {
		w = strings.TrimSpace(w)
		if w == "" || seen[w] {
			continue
		}
		if _, err := readWorkspaceMeta(w); err != nil {
			return AgentInstanceMeta{}, fmt.Errorf("unknown workspace %q", w)
		}
		seen[w] = true
		clean = append(clean, w)
	}
	if len(clean) == 0 {
		return AgentInstanceMeta{}, fmt.Errorf("choose at least one workspace")
	}
	if tools != nil {
		rec, err := readAgentRecipe(m.RecipeID)
		if err != nil {
			return AgentInstanceMeta{}, err
		}
		next, err := cleanToolNames(*tools)
		if err != nil {
			return AgentInstanceMeta{}, err
		}
		allowed := recipeToolNames(rec)
		defTools := rec.Tools
		if rec.ID == "" {
			defTools = nil
		}
		for _, n := range next {
			if !definitionAllowsName(defTools, n) {
				return AgentInstanceMeta{}, fmt.Errorf("tool %q is not on the definition", n)
			}
		}
		if sameToolSet(next, allowed) {
			m.Tools = nil
		} else {
			m.Tools = &next
		}
	} else {
		m.Tools = nil
	}
	m.Workspaces = clean
	if !agentInstanceAllowsWorkspace(m, m.CurrentWorkspace) {
		m.CurrentWorkspace = clean[0]
	}
	if err := writeAgentInstanceMeta(m); err != nil {
		return AgentInstanceMeta{}, err
	}
	return m, nil
}

func agentInstanceDisplayName(id string) string {
	id = strings.TrimSpace(id)
	m, err := readAgentInstanceMeta(id)
	if err != nil || strings.TrimSpace(m.Name) == "" {
		if id == "" {
			return "agent"
		}
		return id
	}
	return strings.TrimSpace(m.Name)
}

func formatPeerUserText(name, text string) string {
	return "[From " + strings.TrimSpace(name) + "] " + strings.TrimSpace(text)
}

func parsePeerUserText(s string) (name, body string, ok bool) {
	const prefix = "[From "
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(s, prefix)
	i := strings.Index(rest, "] ")
	if i <= 0 {
		return "", "", false
	}
	name = strings.TrimSpace(rest[:i])
	body = strings.TrimSpace(rest[i+2:])
	if name == "" || body == "" {
		return "", "", false
	}
	return name, body, true
}

func (a *Agent) recordPeerSent(peerName, text string) {
	peerName = strings.TrimSpace(peerName)
	text = strings.TrimSpace(text)
	if a == nil || peerName == "" || text == "" {
		return
	}
	a.emitJSON("message", map[string]any{
		"role": "peer",
		"name": "To " + peerName,
		"text": text,
	})
	if a.Journal == nil {
		return
	}
	_ = a.add(map[string]any{
		"type":      "keimling_peer",
		"direction": "sent",
		"peer":      peerName,
		"text":      text,
	})
}
