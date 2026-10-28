package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AgentTeam is a named set of definition ids under ~/.config/keimling/agent-teams/.
type AgentTeam struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	DefinitionIDs []string  `json:"definition_ids"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func agentTeamsDir() string {
	return filepath.Join(configHome(), "keimling", "agent-teams")
}

func agentTeamPath(id string) string {
	return filepath.Join(agentTeamsDir(), id+".json")
}

func ensureAgentTeamsDir() error {
	return os.MkdirAll(agentTeamsDir(), 0o700)
}

func writeAgentTeam(t AgentTeam) error {
	if err := ensureAgentTeamsDir(); err != nil {
		return err
	}
	t.ID = strings.TrimSpace(t.ID)
	if t.ID == "" {
		return fmt.Errorf("team id required")
	}
	if strings.TrimSpace(t.Name) == "" {
		t.Name = t.ID
	}
	t.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := agentTeamPath(t.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readAgentTeam(id string) (AgentTeam, error) {
	var t AgentTeam
	b, err := os.ReadFile(agentTeamPath(strings.TrimSpace(id)))
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(b, &t); err != nil {
		return t, err
	}
	return t, nil
}

func listAgentTeams() ([]AgentTeam, error) {
	if err := ensureAgentTeamsDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(agentTeamsDir())
	if err != nil {
		return nil, err
	}
	out := make([]AgentTeam, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		t, err := readAgentTeam(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func createAgentTeam(name string, definitionIDs []string) (AgentTeam, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return AgentTeam{}, fmt.Errorf("name required")
	}
	ids := uniqueNonEmpty(definitionIDs)
	if len(ids) == 0 {
		return AgentTeam{}, fmt.Errorf("at least one definition_id required")
	}
	for _, id := range ids {
		if _, err := readAgentRecipe(id); err != nil {
			return AgentTeam{}, fmt.Errorf("definition %s: %w", id, err)
		}
	}
	t := AgentTeam{
		ID:            fmt.Sprintf("team-%s", time.Now().UTC().Format("20060102-150405.000000000")),
		Name:          name,
		DefinitionIDs: ids,
	}
	if err := writeAgentTeam(t); err != nil {
		return AgentTeam{}, err
	}
	return t, nil
}

func deleteAgentTeam(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("team id required")
	}
	path := agentTeamPath(id)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("unknown team %q", id)
		}
		return err
	}
	return os.Remove(path)
}

func uniqueNonEmpty(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
