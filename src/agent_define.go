package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func defineAgentTool(name string) bool {
	switch name {
	case "agent_instructions", "save_agent", "spawn_agent", "save_team", "spawn_team", "set_agent_limit":
		return true
	default:
		return false
	}
}

func defineAgentTools() []map[string]any {
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	defs := map[string]any{
		"type":        "array",
		"description": "Definition ids or unique definition names. Empty on update keeps the saved list.",
		"items":       map[string]any{"type": "string"},
	}
	tools := map[string]any{
		"type":        []any{"array", "null"},
		"description": "Always send this field. Tool names this definition may use (catalog names such as exec, read_file, write_file, message_agent; mcp for every MCP server). Required on create; an empty array grants no tools. On update, null keeps the saved choice.",
		"items":       map[string]any{"type": "string"},
	}
	peers := map[string]any{
		"type":        []any{"array", "null"},
		"description": "Definition ids or unique definition names this kind of agent may contact. Empty or null means no declared contacts. On update, null keeps the saved list and an empty array clears it.",
		"items":       map[string]any{"type": "string"},
	}
	spawnPeers := map[string]any{
		"type":        []any{"array", "null"},
		"description": "Existing agent instance ids or unique instance names the new instance may contact. Empty or null adds none beyond the definition’s declared contacts.",
		"items":       map[string]any{"type": "string"},
	}
	return []map[string]any{
		toolDef("agent_instructions", "Return the instructions a new agent definition starts from. Call this before save_agent when writing or revising a prompt, and start from this text. Pass unused as an empty string.", map[string]any{"unused": str("Pass empty string")}, "unused"),
		toolDef("save_agent", "Create an agent definition, or replace the instructions of an existing one. id empty creates. On create, instructions empty stores the agent_instructions text and the mode is auto, and tools is required: a definition without a tool list grants nothing. On update, instructions empty keeps the saved text; a non-empty value replaces it and leaves the mode unchanged; tools null keeps the saved choice. peers names other definitions this kind of agent may contact; instances of those definitions may message each other in a shared workspace. Send every parameter. Returns the definition id.", map[string]any{"id": str("Definition id to update, or empty to create"), "name": str("Name. Required on create. Empty on update keeps the saved name."), "instructions": str("Job text. Empty on create uses the starting instructions. Empty on update keeps the saved text."), "tools": tools, "peers": peers}, "id", "name", "instructions", "tools", "peers"),
		toolDef("spawn_agent", "Start one instance of a definition. definition is an id or a unique name. name empty uses the definition name. path is an existing directory that becomes the current workspace of the instance, for example a git worktree; empty uses the workspace this call is running in. task is the first message, or empty when the instance should wait. peers names existing agent instances (ids or unique names) the new instance may contact, in addition to the contacts its definition declares. spend_limit_usd and spend_limit_tokens cap that instance for its whole life. Empty keeps the definition ceiling. 0 means no ceiling for that unit. Send every parameter. The instance runs in the background; the view of the user does not switch to it.", map[string]any{"definition": str("Definition id or unique definition name"), "name": str("Instance name, or empty to use the definition name"), "path": str("Existing directory the instance works in (a git worktree), or empty for the workspace of this call"), "task": str("First message for the new instance, or empty."), "peers": spawnPeers, "spend_limit_usd": str("Dollar ceiling for the life of the instance, 0 for none, or empty to keep the definition ceiling."), "spend_limit_tokens": str("Token ceiling for the life of the instance, 0 for none, or empty to keep the definition ceiling.")}, "definition", "name", "path", "task", "peers", "spend_limit_usd", "spend_limit_tokens"),
		toolDef("set_agent_limit", "Change the spend ceiling of an existing agent instance. Requires the same grant as save_agent. instance_id is an id or a unique name. spend_limit_usd and spend_limit_tokens replace that unit. 0 or empty clears it. The ceiling lasts for the life of the instance. An instance cannot change its own ceiling. Send every parameter.", map[string]any{"instance_id": str("Instance id or unique name"), "spend_limit_usd": str("Dollar ceiling, or 0 to clear it"), "spend_limit_tokens": str("Token ceiling, or 0 to clear it")}, "instance_id", "spend_limit_usd", "spend_limit_tokens"),
		toolDef("save_team", "Create a team of agent definitions, or replace the definitions of an existing team. id empty creates. definitions are definition ids or unique names. On update, an empty list keeps the saved definitions. Send every parameter.", map[string]any{"id": str("Team id to update, or empty to create"), "name": str("Team name. Required on create. Empty on update keeps the saved name."), "definitions": defs}, "id", "name", "definitions"),
		toolDef("spawn_team", "Start one instance per definition of a team. team is an id or a unique name. path is an existing directory that becomes the current workspace of every instance, for example a git worktree; empty uses the workspace this call is running in. Send every parameter. Teammates may message each other. The instances run in the background; the view of the user does not switch to them.", map[string]any{"team": str("Team id or unique team name"), "path": str("Existing directory the instances work in (a git worktree), or empty for the workspace of this call")}, "team", "path"),
	}
}

func (a *Agent) runDefineAgentTool(name, args string) string {
	switch name {
	case "agent_instructions":
		return defaultAgentInstructions
	case "save_agent":
		return a.saveAgentTool(args)
	case "spawn_agent":
		return a.spawnAgentTool(args)
	case "set_agent_limit":
		return a.setAgentLimitTool(args)
	case "save_team":
		return a.saveTeamTool(args)
	case "spawn_team":
		return a.spawnTeamTool(args)
	default:
		return name + " error: unknown tool"
	}
}

func (a *Agent) saveAgentTool(args string) string {
	var x struct {
		ID           string    `json:"id"`
		Name         string    `json:"name"`
		Instructions string    `json:"instructions"`
		Tools        *[]string `json:"tools"`
		Peers        *[]string `json:"peers"`
	}
	if err := decodeArgs(args, &x); err != nil {
		return "save_agent error: " + err.Error()
	}
	id := strings.TrimSpace(x.ID)
	if id == "" {
		if strings.TrimSpace(x.Name) == "" {
			return "save_agent error: name required"
		}
		if x.Tools == nil {
			return "save_agent error: tools required on create. Pass the tool names this definition may use (for example [\"exec\",\"read_file\"]); an empty list grants no tools."
		}
		rec, err := createAgentRecipe(x.Name, x.Instructions)
		if err != nil {
			return "save_agent error: " + err.Error()
		}
		if rec, err = setRecipeTools(rec.ID, *x.Tools); err != nil {
			_ = deleteAgentRecipe(rec.ID)
			return "save_agent error: " + err.Error()
		}
		if x.Peers != nil {
			if rec, err = setRecipePeers(rec.ID, *x.Peers); err != nil {
				_ = deleteAgentRecipe(rec.ID)
				return "save_agent error: " + err.Error()
			}
		}
		return formatDefinition(rec, true)
	}
	if _, err := readAgentRecipe(id); err != nil {
		return "save_agent error: " + err.Error()
	}
	replace := strings.TrimSpace(x.Instructions) != ""
	rec, err := updateAgentRecipe(id, x.Name, x.Instructions, replace)
	if err != nil {
		return "save_agent error: " + err.Error()
	}
	if x.Tools != nil {
		if rec, err = setRecipeTools(rec.ID, *x.Tools); err != nil {
			return "save_agent error: " + err.Error()
		}
	}
	if x.Peers != nil {
		if rec, err = setRecipePeers(rec.ID, *x.Peers); err != nil {
			return "save_agent error: " + err.Error()
		}
	}
	return formatDefinition(rec, false)
}

func formatDefinition(rec AgentRecipe, created bool) string {
	verb := "updated"
	if created {
		verb = "created"
	}
	return fmt.Sprintf("definition %s %q %s mode %s", rec.ID, rec.Name, verb, rec.Mode)
}

func (a *Agent) setAgentLimitTool(args string) string {
	if a == nil || !a.instanceToolGranted("save_agent") {
		return "set_agent_limit error: editing an agent is not granted"
	}
	var x struct {
		InstanceID       string `json:"instance_id"`
		SpendLimitUSD    string `json:"spend_limit_usd"`
		SpendLimitTokens string `json:"spend_limit_tokens"`
	}
	if err := decodeArgs(args, &x); err != nil {
		return "set_agent_limit error: " + err.Error()
	}
	meta, err := a.resolveAgentRef(x.InstanceID)
	if err != nil {
		return "set_agent_limit error: " + err.Error()
	}
	if meta.ID == a.spawnerInstanceID() {
		return "set_agent_limit error: an instance cannot change its own ceiling"
	}
	updated, err := setAgentInstanceSpend(meta.ID, x.SpendLimitUSD, x.SpendLimitTokens)
	if err != nil {
		return "set_agent_limit error: " + err.Error()
	}
	usd, tokens := "none", "none"
	if updated.SpendLimitUSD != nil && *updated.SpendLimitUSD > 0 {
		usd = fmt.Sprintf("%.4f", *updated.SpendLimitUSD)
	}
	if updated.SpendLimitTokens != nil && *updated.SpendLimitTokens > 0 {
		tokens = fmt.Sprintf("%d", *updated.SpendLimitTokens)
	}
	return fmt.Sprintf("ceiling for %s (%s): $%s, %s tokens", updated.Name, updated.ID, usd, tokens)
}

func (a *Agent) spawnAgentTool(args string) string {
	var x struct {
		Definition       string   `json:"definition"`
		Name             string   `json:"name"`
		Path             string   `json:"path"`
		Task             string   `json:"task"`
		Peers            []string `json:"peers"`
		SpendLimitUSD    string   `json:"spend_limit_usd"`
		SpendLimitTokens string   `json:"spend_limit_tokens"`
	}
	if err := decodeArgs(args, &x); err != nil {
		return "spawn_agent error: " + err.Error()
	}
	ws, err := a.defineWorkspace()
	if err != nil {
		return "spawn_agent error: " + err.Error()
	}
	rec, err := resolveAgentRecipe(x.Definition)
	if err != nil {
		return "spawn_agent error: " + err.Error()
	}
	started, body, err := postHostJSON("/api/agents/instances", map[string]any{
		"recipe_id":          rec.ID,
		"name":               strings.TrimSpace(x.Name),
		"path":               strings.TrimSpace(x.Path),
		"workspace_id":       ws,
		"spawner_id":         a.spawnerInstanceID(),
		"peers":              x.Peers,
		"spend_limit_usd":    x.SpendLimitUSD,
		"spend_limit_tokens": x.SpendLimitTokens,
	})
	if err == nil {
		return formatHostSpawn("spawn_agent", started, body) + queueInstanceTask(instanceIDFromHost(body), x.Task) + spawnBackgroundNote
	}
	if err != errNoHost {
		return "spawn_agent error: " + err.Error()
	}
	inst, err := createAgentInstanceForPath(rec, ws, x.Path, x.Name, a.spawnerInstanceID())
	if err != nil {
		return "spawn_agent error: " + err.Error()
	}
	inst, err = finalizeSpawnSpend(inst, x.SpendLimitUSD, x.SpendLimitTokens)
	if err != nil {
		rollbackSpawn([]AgentInstanceMeta{inst})
		return "spawn_agent error: " + err.Error()
	}
	if err := linkSpawnedInstances(a.spawnerInstanceID(), []AgentInstanceMeta{inst}); err != nil {
		rollbackSpawn([]AgentInstanceMeta{inst})
		return "spawn_agent error: " + err.Error()
	}
	if err := applySpawnContacts(inst, x.Peers); err != nil {
		rollbackSpawn([]AgentInstanceMeta{inst})
		return "spawn_agent error: " + err.Error()
	}
	return formatInstance(inst, false) + queueInstanceTask(inst.ID, x.Task) + spawnBackgroundNote
}

// spawnerInstanceID is the agent instance that runs this call. A workspace
// thread has none: its spawns link only the instances they create.
func (a *Agent) spawnerInstanceID() string {
	if a == nil {
		return ""
	}
	return strings.TrimSpace(a.Cfg.AgentInstanceID)
}

// linkSpawnedInstances connects every instance of one spawn with the others and
// the spawning agent with each of them. The contact is written before a task is
// queued, so a new instance can answer its spawner straight away. An error here
// fails the spawn: an instance that cannot answer is not delivered silently.
func linkSpawnedInstances(spawnerID string, created []AgentInstanceMeta) error {
	for i := range created {
		for j := i + 1; j < len(created); j++ {
			if err := linkAgentInstances(created[i].ID, created[j].ID); err != nil {
				return fmt.Errorf("contact %s and %s: %w", created[i].ID, created[j].ID, err)
			}
		}
	}
	spawnerID = strings.TrimSpace(spawnerID)
	if spawnerID == "" {
		return nil
	}
	for _, inst := range created {
		if inst.ID == spawnerID {
			continue
		}
		if err := linkAgentInstances(spawnerID, inst.ID); err != nil {
			return fmt.Errorf("contact %s and %s: %w", spawnerID, inst.ID, err)
		}
	}
	return nil
}

// spawnBackgroundNote tells the calling model where the new instance ran to.
// A spawn does not attach, so the user still looks at the chat they were in.
const spawnBackgroundNote = "\nThe instance runs in the background. The user's view stays on the chat they are in; the instance appears in the sidebar and opens on a click."

func instanceIDFromHost(body []byte) string {
	var resp struct {
		Instance AgentInstanceMeta `json:"instance"`
	}
	if json.Unmarshal(body, &resp) != nil {
		return ""
	}
	return strings.TrimSpace(resp.Instance.ID)
}

func queueInstanceTask(instanceID, task string) string {
	instanceID = strings.TrimSpace(instanceID)
	task = strings.TrimSpace(task)
	if instanceID == "" || task == "" {
		return ""
	}
	if err := enqueueAgentMessage(instanceID, task, "", "message"); err != nil {
		return "\ntask not queued: " + err.Error()
	}
	return "\ntask queued"
}

func (a *Agent) saveTeamTool(args string) string {
	var x struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Definitions []string `json:"definitions"`
	}
	if err := decodeArgs(args, &x); err != nil {
		return "save_team error: " + err.Error()
	}
	ids, err := resolveRecipeRefs(x.Definitions)
	if err != nil {
		return "save_team error: " + err.Error()
	}
	id := strings.TrimSpace(x.ID)
	if id == "" {
		if strings.TrimSpace(x.Name) == "" {
			return "save_team error: name required"
		}
		if len(ids) == 0 {
			return "save_team error: at least one definition required"
		}
		team, err := createAgentTeam(x.Name, ids)
		if err != nil {
			return "save_team error: " + err.Error()
		}
		return formatTeam(team, true)
	}
	team, err := updateAgentTeam(id, x.Name, ids, len(ids) > 0)
	if err != nil {
		return "save_team error: " + err.Error()
	}
	return formatTeam(team, false)
}

func formatTeam(t AgentTeam, created bool) string {
	verb := "updated"
	if created {
		verb = "created"
	}
	return fmt.Sprintf("team %s %q %s definitions %s", t.ID, t.Name, verb, strings.Join(t.DefinitionIDs, ", "))
}

func (a *Agent) spawnTeamTool(args string) string {
	var x struct {
		Team string `json:"team"`
		Path string `json:"path"`
	}
	if err := decodeArgs(args, &x); err != nil {
		return "spawn_team error: " + err.Error()
	}
	ws, err := a.defineWorkspace()
	if err != nil {
		return "spawn_team error: " + err.Error()
	}
	team, err := resolveAgentTeam(x.Team)
	if err != nil {
		return "spawn_team error: " + err.Error()
	}
	started, body, err := postHostJSON("/api/agents/teams/spawn", map[string]string{
		"id":           team.ID,
		"path":         strings.TrimSpace(x.Path),
		"workspace_id": ws,
		"spawner_id":   a.spawnerInstanceID(),
	})
	if err == nil {
		return formatHostSpawn("spawn_team", started, body) + spawnBackgroundNote
	}
	if err != errNoHost {
		return "spawn_team error: " + err.Error()
	}
	return a.spawnTeamLocal(team, ws, x.Path) + spawnBackgroundNote
}

func (a *Agent) spawnTeamLocal(team AgentTeam, ws, path string) string {
	created := make([]AgentInstanceMeta, 0, len(team.DefinitionIDs))
	for _, defID := range team.DefinitionIDs {
		rec, err := readAgentRecipe(defID)
		if err != nil {
			return "spawn_team error: " + err.Error()
		}
		inst, err := createAgentInstanceForPath(rec, ws, path, rec.Name, a.spawnerInstanceID())
		if err != nil {
			rollbackSpawn(created)
			return "spawn_team error: " + err.Error()
		}
		created = append(created, inst)
	}
	if err := linkSpawnedInstances(a.spawnerInstanceID(), created); err != nil {
		rollbackSpawn(created)
		return "spawn_team error: " + err.Error()
	}
	for _, inst := range created {
		if err := applySpawnContacts(inst, nil); err != nil {
			rollbackSpawn(created)
			return "spawn_team error: " + err.Error()
		}
	}
	var b strings.Builder
	for _, inst := range created {
		fmt.Fprintln(&b, formatInstance(inst, false))
	}
	return strings.TrimSpace(b.String())
}

func (a *Agent) defineWorkspace() (string, error) {
	if a == nil {
		return "", fmt.Errorf("no workspace")
	}
	if id := strings.TrimSpace(a.Cfg.AgentInstanceID); id != "" {
		m, err := readAgentInstanceMeta(id)
		if err != nil {
			return "", err
		}
		ws := strings.TrimSpace(m.CurrentWorkspace)
		if ws == "" {
			ws = strings.TrimSpace(a.Cfg.WorkspaceID)
		}
		if ws == "" {
			return "", fmt.Errorf("no workspace")
		}
		if !agentInstanceAllowsWorkspace(m, ws) {
			return "", fmt.Errorf("workspace not in ACL")
		}
		return ws, nil
	}
	ws := strings.TrimSpace(a.Cfg.WorkspaceID)
	if ws == "" {
		return "", fmt.Errorf("no workspace")
	}
	if _, err := readWorkspaceMeta(ws); err != nil {
		return "", fmt.Errorf("workspace: %w", err)
	}
	return ws, nil
}

func resolveAgentRecipe(ref string) (AgentRecipe, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return AgentRecipe{}, fmt.Errorf("definition required")
	}
	if rec, err := readAgentRecipe(ref); err == nil {
		return rec, nil
	}
	list, err := listAgentRecipes()
	if err != nil {
		return AgentRecipe{}, err
	}
	var hits []AgentRecipe
	for _, rec := range list {
		if rec.Name == ref {
			hits = append(hits, rec)
		}
	}
	if len(hits) == 1 {
		return hits[0], nil
	}
	if len(hits) == 0 {
		return AgentRecipe{}, fmt.Errorf("unknown definition %q", ref)
	}
	return AgentRecipe{}, fmt.Errorf("definition %q matches %s", ref, joinIDs(hitsToIDs(hits)))
}

func hitsToIDs(recs []AgentRecipe) []string {
	out := make([]string, len(recs))
	for i, rec := range recs {
		out[i] = rec.ID
	}
	return out
}

func resolveRecipeRefs(refs []string) ([]string, error) {
	var ids []string
	for _, ref := range refs {
		if strings.TrimSpace(ref) == "" {
			continue
		}
		rec, err := resolveAgentRecipe(ref)
		if err != nil {
			return nil, err
		}
		ids = append(ids, rec.ID)
	}
	return uniqueNonEmpty(ids), nil
}

func resolveAgentTeam(ref string) (AgentTeam, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return AgentTeam{}, fmt.Errorf("team required")
	}
	if team, err := readAgentTeam(ref); err == nil {
		return team, nil
	}
	list, err := listAgentTeams()
	if err != nil {
		return AgentTeam{}, err
	}
	var hits []AgentTeam
	for _, team := range list {
		if team.Name == ref {
			hits = append(hits, team)
		}
	}
	if len(hits) == 1 {
		return hits[0], nil
	}
	if len(hits) == 0 {
		return AgentTeam{}, fmt.Errorf("unknown team %q", ref)
	}
	ids := make([]string, len(hits))
	for i, team := range hits {
		ids[i] = team.ID
	}
	return AgentTeam{}, fmt.Errorf("team %q matches %s", ref, joinIDs(ids))
}

func joinIDs(ids []string) string {
	return strings.Join(ids, " and ")
}

func formatInstance(inst AgentInstanceMeta, started bool) string {
	state := "saved"
	if started {
		state = "started"
	}
	return fmt.Sprintf("instance %s %q %s in workspace %s from definition %s", inst.ID, inst.Name, state, inst.CurrentWorkspace, inst.RecipeID)
}

var errNoHost = fmt.Errorf("no host")

var hostAPIClient = &http.Client{Timeout: 60 * time.Second}

// postHostJSON asks the host to create and start an instance. errNoHost means
// this process has no host to call, so the caller saves the instance itself.
func postHostJSON(path string, payload any) (bool, []byte, error) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("KEIMLING_HOST")), "/")
	secret := strings.TrimSpace(os.Getenv("KEIMLING_SECRET"))
	if base == "" || secret == "" {
		return false, nil, errNoHost
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, nil, err
	}
	req, err := http.NewRequest(http.MethodPost, base+path, bytes.NewReader(body))
	if err != nil {
		return false, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Content-Type", "application/json")
	res, err := hostAPIClient.Do(req)
	if err != nil {
		return false, nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return false, nil, err
	}
	if res.StatusCode/100 != 2 {
		msg := strings.TrimSpace(string(data))
		if msg == "" {
			msg = res.Status
		}
		return false, nil, fmt.Errorf("%s", msg)
	}
	return true, data, nil
}

func formatHostSpawn(tool string, _ bool, body []byte) string {
	var resp struct {
		Instance  AgentInstanceMeta   `json:"instance"`
		Instances []AgentInstanceMeta `json:"instances"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return tool + " error: " + err.Error()
	}
	if resp.Instance.ID != "" {
		return formatInstance(resp.Instance, true)
	}
	if len(resp.Instances) == 0 {
		return tool + " error: host returned no instance"
	}
	var b strings.Builder
	for _, inst := range resp.Instances {
		fmt.Fprintln(&b, formatInstance(inst, true))
	}
	return strings.TrimSpace(b.String())
}
