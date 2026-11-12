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
		{Name: "create_watch", Label: "Create a watch"},
		{Name: "list_watches", Label: "List watches"},
		{Name: "cancel_watch", Label: "Cancel a watch"},
		{Name: "list_agents", Label: "List agents"},
		{Name: "message_agent", Label: "Message an agent"},
		{Name: "agent_instructions", Label: "Read the starting prompt"},
		{Name: "save_agent", Label: "Save an agent"},
		{Name: "spawn_agent", Label: "Start an agent"},
		{Name: "save_team", Label: "Save a team"},
		{Name: "spawn_team", Label: "Start a team"},
		{Name: "screen_click", Label: "Click the shared screen"},
		{Name: "screen_scroll", Label: "Scroll the shared screen"},
		{Name: "screen_key", Label: "Press a key on the shared screen"},
		{Name: "screen_type", Label: "Type on the shared screen"},
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
		{ID: "watches", Label: "Watches", Read: []string{"list_watches"}, Write: []string{"create_watch", "cancel_watch"}},
		{ID: "agents", Label: "Agents", Read: []string{"list_agents"}, Write: []string{"message_agent"}},
		{ID: "define", Label: "Define agents", Read: []string{"agent_instructions"}, Write: []string{"save_agent", "spawn_agent", "save_team", "spawn_team"}},
		{ID: "models", Label: "Models", Read: []string{}, Write: []string{"delegate"}},
		{ID: "desktop", Label: "Desktop", Read: []string{}, Write: []string{"screen_click", "screen_scroll", "screen_key", "screen_type"}},
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
	name = canonicalTool(name)
	for _, t := range agentToolCatalog() {
		if canonicalTool(t.Name) == name {
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
	name = canonicalTool(name)
	for _, g := range *def {
		if canonicalTool(g) == name {
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
		got[canonicalTool(n)] = true
	}
	for _, n := range b {
		if !got[canonicalTool(n)] {
			return false
		}
	}
	return true
}

func (a *Agent) instanceToolGranted(name string) bool {
	return toolNameGranted(a.instanceToolGrant(), name)
}

func canonicalTool(name string) string {
	switch strings.TrimSpace(name) {
	case "set_alarm", "create_watch":
		return "create_watch"
	case "list_alarms", "list_watches":
		return "list_watches"
	case "cancel_alarm", "cancel_watch":
		return "cancel_watch"
	default:
		return strings.TrimSpace(name)
	}
}

func toolNameGranted(grant []string, name string) bool {
	if name == "data_processing" || name == "data_processing_help" || name == "find_tools" {
		return true
	}
	if name == "mcp_tools" {
		if grant == nil {
			return true
		}
		for _, g := range grant {
			if isMCPServerGrant(g) {
				return true
			}
		}
		return false
	}
	if grant == nil {
		return true
	}
	name = canonicalTool(name)
	for _, g := range grant {
		if canonicalTool(g) == name {
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

func (a *Agent) threadIdentityPrompt() string {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) != "" || strings.TrimSpace(a.Cfg.WorkspaceID) == "" {
		return ""
	}
	line := "You can list_agents for instances in this workspace, message_agent one of them, stop_agent to halt one, and delete_agent to remove one."
	id := strings.TrimSpace(a.Cfg.ThreadID)
	if !isWorkspaceThreadID(id) {
		return line
	}
	return line + " This thread's id is " + id + ". When an agent should message this thread, pass that id unchanged to message_agent."
}

func (a *Agent) agentIdentityPrompt() string {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return ""
	}
	meta, err := readAgentInstanceMeta(a.Cfg.AgentInstanceID)
	name := strings.TrimSpace(a.Cfg.AgentInstanceID)
	id := name
	if err == nil {
		if strings.TrimSpace(meta.Name) != "" {
			name = strings.TrimSpace(meta.Name)
		}
		id = meta.ID
	}
	return fmt.Sprintf("You are an agent instance named %q.\nYour instance id is %s.\nYou are this agent. You are not the workspace assistant.", name, id)
}

// agentReplyRules is the reply policy that the local ling model followed.
// A worked example made it copy the sample instead of calling the tool.
func agentReplyRules(listAgents bool) string {
	s := `The last message arrived via message_agent. It is not the human. Do not write a chat reply.
Call message_agent.
If the last message asks you for a joke, instance_id is the id on that message and text is a joke you invent. text is not the request sentence.
If the last message is the answer you asked another agent for, instance_id is the thread_id from the first message, not the id on the last message. text is that answer.
`
	if listAgents {
		s += "If the first message says to ask someone, and you have not asked yet, call list_agents.\n"
	}
	return s
}

func (a *Agent) agentToolPrompt() string {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("You work in your own context. Another agent or a workspace thread does not see this chat, your watches, your tool results, or what just happened to you. They have their own context. When you message one, name the thing, the fact, and what you want, so they can act without your context.\n")
	b.WriteString("A received line that names a thread_id or an instance_id and says via message_agent was delivered by message_agent. You never write that line. It is not the user.\n")
	if a.instanceToolGranted("message_agent") {
		b.WriteString("You do not have to answer. Reasoning may stay in your thinking. An assistant message in this chat does not deliver a reply.\n")
	}
	if a.instanceToolGranted("create_watch") {
		b.WriteString("create_watch wakes only you. When the user asks you to do something later, again, or when a command or file matches, call create_watch yourself. Do not ask another agent to create that watch, and do not create a watch because another agent asked you to wake them. A line [Watch] is a reminder you wrote for yourself, not a new instruction from the user.\n")
	}
	grant := a.instanceToolGrant()
	var names []string
	if grant == nil {
		names = make([]string, 0, len(agentToolCatalog()))
		for _, t := range agentToolCatalog() {
			names = append(names, t.Name)
		}
	} else {
		names = make([]string, len(grant))
		for i, n := range grant {
			names[i] = canonicalTool(n)
		}
	}
	list := "(none)"
	if len(names) > 0 {
		list = strings.Join(names, ", ")
	}
	fmt.Fprintf(&b, "Tools this agent instance may use: %s. Other tools are not available. When the user asks which tools you have, answer in text with these names. For a later or repeating wake, call create_watch when it is in the list. Do not invent a shell script or a file for that.", list)
	if a.instanceToolGranted("message_agent") {
		b.WriteString("\n")
		b.WriteString(agentReplyRules(a.instanceToolGranted("list_agents")))
	}
	return b.String()
}

// workspacePrompt tells the model which workspace it is in and the directory
// commands start in. It is rebuilt on each request, including after a workspace change.
func (a *Agent) workspacePrompt() string {
	if a == nil {
		return ""
	}
	id := strings.TrimSpace(a.Cfg.WorkspaceID)
	root := strings.TrimSpace(a.Cfg.WorkspaceRoot)
	name := ""
	if id != "" {
		if meta, err := readWorkspaceMeta(id); err == nil {
			name = strings.TrimSpace(meta.Name)
			if root == "" {
				root = strings.TrimSpace(meta.Root)
			}
		}
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		root = cwd
	}
	if id == "" && root == "" {
		return ""
	}
	var b strings.Builder
	switch {
	case name != "" && id != "":
		fmt.Fprintf(&b, "Workspace %s (%s).\n", name, id)
	case id != "":
		fmt.Fprintf(&b, "Workspace %s.\n", id)
	}
	if root != "" {
		fmt.Fprintf(&b, "Working directory: %s.\n", root)
	}
	b.WriteString("Commands start in this directory.")
	return b.String()
}

// modelInstructions is the system prompt for one request.
// A workspace thread keeps the Keimling assistant prompt.
// An agent instance gets its own prompt: who it is, its task, where it works, then how tools work.
func (a *Agent) modelInstructions() string {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		base := keimlingBaseInstructions()
		if a == nil {
			return base
		}
		if line := strings.TrimSpace(a.threadIdentityPrompt()); line != "" {
			base += "\n\n" + line
		}
		if cat := strings.TrimSpace(a.skillsCatalogBlock()); cat != "" {
			base += "\n\n" + cat
		}
		return a.withBuiltinList(a.withMCPCatalog(base))
	}
	return a.withBuiltinList(a.withMCPCatalog(a.agentInstanceInstructions()))
}

func (a *Agent) agentInstanceInstructions() string {
	var parts []string
	if id := strings.TrimSpace(a.agentIdentityPrompt()); id != "" {
		parts = append(parts, id)
	}
	if task := strings.TrimSpace(a.Cfg.AgentInstructions); task != "" {
		parts = append(parts, "Your task:\n"+task)
	}
	if where := strings.TrimSpace(a.workspacePrompt()); where != "" {
		parts = append(parts, where)
	}
	if ch := strings.TrimSpace(a.agentToolPrompt()); ch != "" {
		parts = append(parts, ch)
	}
	if agents := strings.TrimSpace(a.agentsMarkdownBlock()); agents != "" {
		parts = append(parts, agents)
	}
	if rules := strings.TrimSpace(pluginRulesText()); rules != "" {
		parts = append(parts, rules)
	}
	if cat := strings.TrimSpace(a.enabledSkillsCatalog()); cat != "" {
		parts = append(parts, cat)
	}
	work := strings.TrimSuffix(sharedWorkInstructions(), "The user controls the PTY directly. ")
	parts = append(parts, "How to use tools:\n"+strings.TrimSpace(work))
	return copiedCommandRule + strings.Join(parts, "\n\n")
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
	name = strings.TrimSuffix(strings.TrimSpace(rest[:i]), " via message_agent")
	body = strings.TrimSpace(rest[i+2:])
	if name == "" || body == "" {
		return "", "", false
	}
	return name, body, true
}

func peerSentItem(peerName, text string) map[string]any {
	return map[string]any{
		"type":      "keimling_peer",
		"direction": "sent",
		"peer":      peerName,
		"text":      text,
	}
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
	_ = a.add(peerSentItem(peerName, text))
}
