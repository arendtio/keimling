package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	agentStatusIdle    = "idle"
	agentStatusRunning = "running"
	agentStatusStopped = "stopped"
)

// AgentInstanceMeta is persisted under ~/.local/share/keimling/agent-instances/<id>/.
type AgentInstanceMeta struct {
	ID               string    `json:"id"`
	RecipeID         string    `json:"recipe_id"`
	Name             string    `json:"name"`
	Workspaces       []string  `json:"workspaces"`      // ACL of workspace ids
	Peers            []string  `json:"peers,omitempty"` // instance ids this agent may message
	CurrentWorkspace string    `json:"current_workspace"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func agentInstancesRoot() string {
	return filepath.Join(dataHome(), "keimling", "agent-instances")
}

func agentInstanceDir(id string) string {
	return filepath.Join(agentInstancesRoot(), id)
}

func agentInstanceMetaPath(id string) string {
	return filepath.Join(agentInstanceDir(id), "meta.json")
}

func agentInstanceThreadDir(id string) string {
	return filepath.Join(agentInstanceDir(id), "thread")
}

func agentInstanceSessionPath(id string) string {
	return filepath.Join(agentInstanceThreadDir(id), "session.jsonl")
}

func newAgentInstanceID() string {
	return fmt.Sprintf("agent-%s", time.Now().UTC().Format("20060102-150405.000000000"))
}

func writeAgentInstanceMeta(m AgentInstanceMeta) error {
	if err := os.MkdirAll(agentInstanceDir(m.ID), 0o700); err != nil {
		return err
	}
	m.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := agentInstanceMetaPath(m.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readAgentInstanceMeta(id string) (AgentInstanceMeta, error) {
	var m AgentInstanceMeta
	b, err := os.ReadFile(agentInstanceMetaPath(strings.TrimSpace(id)))
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, err
	}
	return m, nil
}

func listAgentInstances() ([]AgentInstanceMeta, error) {
	entries, err := os.ReadDir(agentInstancesRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]AgentInstanceMeta, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m, err := readAgentInstanceMeta(e.Name())
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func listAgentInstancesForWorkspace(workspaceID string) ([]AgentInstanceMeta, error) {
	all, err := listAgentInstances()
	if err != nil {
		return nil, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return all, nil
	}
	out := make([]AgentInstanceMeta, 0, len(all))
	for _, m := range all {
		if agentInstanceAllowsWorkspace(m, workspaceID) {
			out = append(out, m)
		}
	}
	return out, nil
}

func agentInstanceAllowsWorkspace(m AgentInstanceMeta, workspaceID string) bool {
	for _, id := range m.Workspaces {
		if id == workspaceID {
			return true
		}
	}
	return false
}

func createAgentInstance(recipe AgentRecipe, workspaceID, name string) (AgentInstanceMeta, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return AgentInstanceMeta{}, fmt.Errorf("workspace_id required")
	}
	if _, err := readWorkspaceMeta(workspaceID); err != nil {
		return AgentInstanceMeta{}, fmt.Errorf("workspace: %w", err)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = recipe.Name
	}
	id := newAgentInstanceID()
	if err := os.MkdirAll(agentInstanceThreadDir(id), 0o700); err != nil {
		return AgentInstanceMeta{}, err
	}
	_ = writeThreadMetaForPath(agentInstanceThreadDir(id), ThreadMeta{
		ID:    "main",
		Title: name,
	})
	m := AgentInstanceMeta{
		ID:               id,
		RecipeID:         recipe.ID,
		Name:             name,
		Workspaces:       []string{workspaceID},
		CurrentWorkspace: workspaceID,
		Status:           agentStatusIdle,
		CreatedAt:        time.Now().UTC(),
	}
	if err := writeAgentInstanceMeta(m); err != nil {
		_ = os.RemoveAll(agentInstanceDir(id))
		return AgentInstanceMeta{}, err
	}
	return m, nil
}

func writeThreadMetaForPath(threadDir string, m ThreadMeta) error {
	if err := os.MkdirAll(threadDir, 0o700); err != nil {
		return err
	}
	m.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := filepath.Join(threadDir, threadMetaFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func deleteAgentInstance(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("instance id required")
	}
	dir := agentInstanceDir(id)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("unknown agent instance %q", id)
		}
		return err
	}
	return os.RemoveAll(dir)
}

func setAgentInstanceWorkspace(id, workspaceID string) error {
	m, err := readAgentInstanceMeta(id)
	if err != nil {
		return err
	}
	if !agentInstanceAllowsWorkspace(m, workspaceID) {
		return fmt.Errorf("workspace %q not allowed for this agent", workspaceID)
	}
	if _, err := readWorkspaceMeta(workspaceID); err != nil {
		return err
	}
	m.CurrentWorkspace = workspaceID
	return writeAgentInstanceMeta(m)
}

func agentInstanceAllowsPeer(m AgentInstanceMeta, peerID string) bool {
	peerID = strings.TrimSpace(peerID)
	for _, id := range m.Peers {
		if id == peerID {
			return true
		}
	}
	return false
}

func addAgentInstancePeer(id, peerID string) error {
	id = strings.TrimSpace(id)
	peerID = strings.TrimSpace(peerID)
	if id == "" || peerID == "" {
		return fmt.Errorf("instance and peer id required")
	}
	if id == peerID {
		return fmt.Errorf("cannot add self as peer")
	}
	if _, err := readAgentInstanceMeta(peerID); err != nil {
		return fmt.Errorf("peer: %w", err)
	}
	m, err := readAgentInstanceMeta(id)
	if err != nil {
		return err
	}
	if agentInstanceAllowsPeer(m, peerID) {
		return nil
	}
	m.Peers = append(m.Peers, peerID)
	return writeAgentInstanceMeta(m)
}

func instancesUsingRecipe(recipeID string) ([]AgentInstanceMeta, error) {
	recipeID = strings.TrimSpace(recipeID)
	all, err := listAgentInstances()
	if err != nil {
		return nil, err
	}
	out := make([]AgentInstanceMeta, 0)
	for _, m := range all {
		if m.RecipeID == recipeID {
			out = append(out, m)
		}
	}
	return out, nil
}
