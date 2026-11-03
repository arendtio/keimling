package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// AgentRecipe is a reusable agent definition under ~/.config/keimling/agents/.
type AgentRecipe struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Instructions string `json:"instructions"`
	Model        string `json:"model,omitempty"`
	// Tools is nil until the definition saves a tool choice. Nil grants every tool.
	Tools      *[]string `json:"tools,omitempty"`
	Skills     []string  `json:"skills,omitempty"`
	MCPServers []string  `json:"mcp_servers,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func configHome() string {
	if runtime.GOOS == "windows" {
		if d, err := os.UserConfigDir(); err == nil && strings.TrimSpace(d) != "" {
			return d
		}
	}
	if x := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); x != "" {
		return x
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

func keimlingConfigDir() string {
	return filepath.Join(configHome(), "keimling")
}

func agentRecipesDir() string {
	return filepath.Join(configHome(), "keimling", "agents")
}

func agentRecipePath(id string) string {
	return filepath.Join(agentRecipesDir(), id+".json")
}

func newRecipeID() string {
	return newStampedID("recipe")
}

func ensureAgentRecipesDir() error {
	return os.MkdirAll(agentRecipesDir(), 0o700)
}

func writeAgentRecipe(r AgentRecipe) error {
	if err := ensureAgentRecipesDir(); err != nil {
		return err
	}
	r.ID = strings.TrimSpace(r.ID)
	if r.ID == "" {
		return fmt.Errorf("recipe id required")
	}
	if strings.TrimSpace(r.Name) == "" {
		r.Name = r.ID
	}
	r.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := agentRecipePath(r.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readAgentRecipe(id string) (AgentRecipe, error) {
	var r AgentRecipe
	b, err := os.ReadFile(agentRecipePath(strings.TrimSpace(id)))
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return r, err
	}
	return r, nil
}

func listAgentRecipes() ([]AgentRecipe, error) {
	if err := ensureAgentRecipesDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(agentRecipesDir())
	if err != nil {
		return nil, err
	}
	out := make([]AgentRecipe, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		r, err := readAgentRecipe(id)
		if err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func createAgentRecipe(name, instructions string) (AgentRecipe, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Agent"
	}
	instr := strings.TrimSpace(instructions)
	if instr == "" {
		instr = "You are a focused Keimling agent. Work carefully in the workspaces you can access. Use change_workspace before touching files outside the current directory."
	}
	r := AgentRecipe{
		ID:           newRecipeID(),
		Name:         name,
		Instructions: instr,
	}
	if err := writeAgentRecipe(r); err != nil {
		return AgentRecipe{}, err
	}
	return r, nil
}

func updateAgentRecipe(id, name, instructions string, updateInstructions bool) (AgentRecipe, error) {
	r, err := readAgentRecipe(id)
	if err != nil {
		return AgentRecipe{}, err
	}
	if n := strings.TrimSpace(name); n != "" {
		r.Name = n
	}
	if updateInstructions {
		r.Instructions = strings.TrimSpace(instructions)
	}
	if err := writeAgentRecipe(r); err != nil {
		return AgentRecipe{}, err
	}
	return r, nil
}

func setRecipeModel(id, model string) (AgentRecipe, error) {
	r, err := readAgentRecipe(id)
	if err != nil {
		return AgentRecipe{}, err
	}
	r.Model = strings.TrimSpace(model)
	if err := writeAgentRecipe(r); err != nil {
		return AgentRecipe{}, err
	}
	return r, nil
}

func setRecipeTools(id string, tools []string) (AgentRecipe, error) {
	r, err := readAgentRecipe(id)
	if err != nil {
		return AgentRecipe{}, err
	}
	next, err := cleanToolNames(tools)
	if err != nil {
		return AgentRecipe{}, err
	}
	r.Tools = &next
	if err := writeAgentRecipe(r); err != nil {
		return AgentRecipe{}, err
	}
	return r, nil
}

func deleteAgentRecipe(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("definition id required")
	}
	used, err := instancesUsingRecipe(id)
	if err != nil {
		return err
	}
	if len(used) > 0 {
		return fmt.Errorf("definition still used by %d instance(s); delete them first", len(used))
	}
	path := agentRecipePath(id)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("unknown definition %q", id)
		}
		return err
	}
	return os.Remove(path)
}
