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
	case "agent_instructions", "save_agent", "spawn_agent", "save_team", "spawn_team":
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
	return []map[string]any{
		toolDef("agent_instructions", "Return the instructions a new agent definition starts from. Call this before save_agent when writing or revising a prompt, and start from this text. Pass unused as an empty string.", map[string]any{"unused": str("Pass empty string")}, "unused"),
		toolDef("save_agent", "Create an agent definition, or replace the instructions of an existing one. id empty creates. On create, instructions empty stores the agent_instructions text and the mode is auto. On update, instructions empty keeps the saved text; a non-empty value replaces it and leaves the mode unchanged. Returns the definition id.", map[string]any{"id": str("Definition id to update, or empty to create"), "name": str("Name. Required on create. Empty on update keeps the saved name."), "instructions": str("Job text. Empty on create uses the starting instructions. Empty on update keeps the saved text.")}, "id", "name", "instructions"),
		toolDef("spawn_agent", "Start one instance of a definition in the workspace this call is running in. definition is an id or a unique name. name empty uses the definition name. An agent can start an instance only in a workspace it can use.", map[string]any{"definition": str("Definition id or unique definition name"), "name": str("Instance name, or empty to use the definition name")}, "definition", "name"),
		toolDef("save_team", "Create a team of agent definitions, or replace the definitions of an existing team. id empty creates. definitions are definition ids or unique names. On update, an empty list keeps the saved definitions.", map[string]any{"id": str("Team id to update, or empty to create"), "name": str("Team name. Required on create. Empty on update keeps the saved name."), "definitions": defs}, "id", "name", "definitions"),
		toolDef("spawn_team", "Start one instance per definition of a team in the workspace this call is running in. team is an id or a unique name. Teammates may message each other. An agent can start a team only in a workspace it can use.", map[string]any{"team": str("Team id or unique team name")}, "team"),
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
		ID           string `json:"id"`
		Name         string `json:"name"`
		Instructions string `json:"instructions"`
	}
	if err := decodeArgs(args, &x); err != nil {
		return "save_agent error: " + err.Error()
	}
	id := strings.TrimSpace(x.ID)
	if id == "" {
		if strings.TrimSpace(x.Name) == "" {
			return "save_agent error: name required"
		}
		rec, err := createAgentRecipe(x.Name, x.Instructions)
		if err != nil {
			return "save_agent error: " + err.Error()
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
	return formatDefinition(rec, false)
}

func formatDefinition(rec AgentRecipe, created bool) string {
	verb := "updated"
	if created {
		verb = "created"
	}
	return fmt.Sprintf("definition %s %q %s mode %s", rec.ID, rec.Name, verb, rec.Mode)
}

func (a *Agent) spawnAgentTool(args string) string {
	var x struct {
		Definition string `json:"definition"`
		Name       string `json:"name"`
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
	started, body, err := postHostJSON("/api/agents/instances", map[string]string{
		"recipe_id":    rec.ID,
		"name":         strings.TrimSpace(x.Name),
		"workspace_id": ws,
	})
	if err == nil {
		return formatHostSpawn("spawn_agent", started, body)
	}
	if err != errNoHost {
		return "spawn_agent error: " + err.Error()
	}
	inst, err := createAgentInstance(rec, ws, x.Name)
	if err != nil {
		return "spawn_agent error: " + err.Error()
	}
	return formatInstance(inst, false)
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
		"workspace_id": ws,
	})
	if err == nil {
		return formatHostSpawn("spawn_team", started, body)
	}
	if err != errNoHost {
		return "spawn_team error: " + err.Error()
	}
	return spawnTeamLocal(team, ws)
}

func spawnTeamLocal(team AgentTeam, ws string) string {
	created := make([]AgentInstanceMeta, 0, len(team.DefinitionIDs))
	for _, defID := range team.DefinitionIDs {
		rec, err := readAgentRecipe(defID)
		if err != nil {
			return "spawn_team error: " + err.Error()
		}
		inst, err := createAgentInstance(rec, ws, rec.Name)
		if err != nil {
			return "spawn_team error: " + err.Error()
		}
		created = append(created, inst)
	}
	for i := range created {
		for j := range created {
			if i == j {
				continue
			}
			if err := addAgentInstancePeer(created[i].ID, created[j].ID); err != nil {
				return "spawn_team error: " + err.Error()
			}
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
