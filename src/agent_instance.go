package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	agentStatusIdle    = "idle"
	agentStatusRunning = "running"
	agentStatusStopped = "stopped"
)

// AgentInstanceMeta is persisted under ~/.local/share/keimling/agent-instances/<id>/.
type AgentInstanceMeta struct {
	ID         string   `json:"id"`
	RecipeID   string   `json:"recipe_id"`
	Name       string   `json:"name"`
	Workspaces []string `json:"workspaces"` // workspace ids this instance may use
	// Tools is nil until the user saves a tool choice. Nil grants every tool.
	// A non-nil slice, including empty, is the saved choice.
	Tools *[]string `json:"tools,omitempty"`
	Peers []string  `json:"peers,omitempty"` // instance ids this agent may message
	// Model overrides the definition. Empty follows the definition.
	Model            string    `json:"model,omitempty"`
	CurrentWorkspace string    `json:"current_workspace"`
	Status           string    `json:"status"`
	WakesHeld        bool      `json:"wakes_held,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func setAgentWakesHeld(id string, held bool) error {
	m, err := readAgentInstanceMeta(id)
	if err != nil {
		return err
	}
	if m.WakesHeld == held {
		return nil
	}
	m.WakesHeld = held
	m.UpdatedAt = time.Now().UTC()
	return writeAgentInstanceMeta(m)
}

func agentWakesHeld(id string) bool {
	m, err := readAgentInstanceMeta(id)
	return err == nil && m.WakesHeld
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

// newStampedID is prefix-date-time-nanoseconds. A dash separates the fraction so a
// model does not treat the id as ending at a full stop.
func newStampedID(prefix string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("%s-%s-%09d", prefix, now.Format("20060102-150405"), now.Nanosecond())
}

func newAgentInstanceID() string {
	return newStampedID("agent")
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

// agentMentionPick is an instance the user chose while typing a name.
type agentMentionPick struct {
	ID   string
	Name string
}

type resolvedAgentMention struct {
	ID   string
	Name string
}

// resolveWorkspaceAgentMentions finds @Name tokens in text. Name is an instance
// that may use workspaceID. A name shared by several instances is resolved only
// when picks names the id. ambiguous is true when at least one token still needs a choice.
func resolveWorkspaceAgentMentions(workspaceID, text string, picks []agentMentionPick) ([]resolvedAgentMention, bool, error) {
	agents, err := listAgentInstancesForWorkspace(workspaceID)
	if err != nil {
		return nil, false, err
	}
	byName := map[string][]AgentInstanceMeta{}
	var names []string
	seen := map[string]bool{}
	for _, m := range agents {
		name := strings.TrimSpace(m.Name)
		key := strings.ToLower(name)
		if key == "" || strings.TrimSpace(m.ID) == "" {
			continue
		}
		if !seen[key] {
			seen[key] = true
			names = append(names, name)
		}
		byName[key] = append(byName[key], m)
	}
	sort.Slice(names, func(i, j int) bool { return len([]rune(names[i])) > len([]rune(names[j])) })
	used := make([]bool, len(picks))
	var out []resolvedAgentMention
	ambiguous := false
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		if runes[i] != '@' {
			continue
		}
		if i > 0 && !unicode.IsSpace(runes[i-1]) {
			continue
		}
		rest := string(runes[i+1:])
		matched := ""
		for _, name := range names {
			if mentionNamePrefix(rest, name) {
				matched = name
				break
			}
		}
		if matched == "" {
			continue
		}
		cands := byName[strings.ToLower(matched)]
		if len(cands) == 1 {
			out = append(out, resolvedAgentMention{ID: cands[0].ID, Name: strings.TrimSpace(cands[0].Name)})
		} else {
			found := false
			for pi, p := range picks {
				if used[pi] {
					continue
				}
				id := strings.TrimSpace(p.ID)
				for _, c := range cands {
					if c.ID == id {
						out = append(out, resolvedAgentMention{ID: c.ID, Name: strings.TrimSpace(c.Name)})
						used[pi] = true
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				ambiguous = true
			}
		}
		i += len([]rune(matched))
	}
	return out, ambiguous, nil
}

// stripResolvedMentions removes the @Name tokens that selected those instances.
func stripResolvedMentions(text string, mentions []resolvedAgentMention) string {
	names := make([]string, 0, len(mentions))
	seen := map[string]bool{}
	for _, m := range mentions {
		name := strings.TrimSpace(m.Name)
		key := strings.ToLower(name)
		if name == "" || seen[key] {
			continue
		}
		seen[key] = true
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return len([]rune(names[i])) > len([]rune(names[j])) })
	runes := []rune(text)
	var b strings.Builder
	for i := 0; i < len(runes); i++ {
		if runes[i] == '@' && (i == 0 || unicode.IsSpace(runes[i-1])) {
			rest := string(runes[i+1:])
			matched := ""
			for _, name := range names {
				if mentionNamePrefix(rest, name) {
					matched = name
					break
				}
			}
			if matched != "" {
				i += len([]rune(matched))
				continue
			}
		}
		b.WriteRune(runes[i])
	}
	return strings.TrimSpace(b.String())
}

func mentionNamePrefix(rest, name string) bool {
	rr := []rune(rest)
	nn := []rune(name)
	if len(nn) == 0 || len(rr) < len(nn) {
		return false
	}
	if !strings.EqualFold(string(rr[:len(nn)]), name) {
		return false
	}
	if len(rr) == len(nn) {
		return true
	}
	return unicode.IsSpace(rr[len(nn)])
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

// ownThreadDir is where this process stores thread.json. An agent instance
// uses its own directory. A workspace thread uses the workspace. The agent
// placeholder id is never a workspace thread path.
func (a *Agent) ownThreadDir() (dir, id string, ok bool) {
	if a == nil {
		return "", "", false
	}
	if inst := strings.TrimSpace(a.Cfg.AgentInstanceID); inst != "" {
		return agentInstanceThreadDir(inst), agentThreadID, true
	}
	if a.Cfg.WorkspaceID == "" || !isWorkspaceThreadID(a.Cfg.ThreadID) {
		return "", "", false
	}
	return threadDir(a.Cfg.WorkspaceID, a.Cfg.ThreadID), a.Cfg.ThreadID, true
}

func (a *Agent) readOwnThreadMeta() (ThreadMeta, error) {
	dir, id, ok := a.ownThreadDir()
	if !ok {
		return ThreadMeta{Title: defaultThreadTitle}, fmt.Errorf("no thread")
	}
	m, err := readThreadMetaFile(filepath.Join(dir, threadMetaFile))
	m.ID = id
	if strings.TrimSpace(m.Title) == "" {
		m.Title = defaultThreadTitle
	}
	return m, err
}

func (a *Agent) writeOwnThreadMeta(m ThreadMeta) error {
	dir, id, ok := a.ownThreadDir()
	if !ok {
		return fmt.Errorf("no thread")
	}
	m.ID = id
	return writeThreadMetaForPath(dir, m)
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

func resolveAgentInstance(ref string) (AgentInstanceMeta, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return AgentInstanceMeta{}, fmt.Errorf("agent id required")
	}
	if m, err := readAgentInstanceMeta(ref); err == nil {
		return m, nil
	}
	all, err := listAgentInstances()
	if err != nil {
		return AgentInstanceMeta{}, err
	}
	var matches []AgentInstanceMeta
	for _, m := range all {
		if strings.EqualFold(m.Name, ref) {
			matches = append(matches, m)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	return AgentInstanceMeta{}, fmt.Errorf("unknown agent %q", ref)
}

func peersContain(list []string, id string) bool {
	id = strings.TrimSpace(id)
	for _, one := range list {
		if one == id {
			return true
		}
	}
	return false
}

// agentsMayContact is true when either instance lists the other. A saved link
// works in both directions so the receiver can reply.
func agentsMayContact(a, b AgentInstanceMeta) bool {
	if a.ID == "" || b.ID == "" || a.ID == b.ID {
		return false
	}
	return peersContain(a.Peers, b.ID) || peersContain(b.Peers, a.ID)
}

// setAgentInstancePeers stores the instances id may contact and writes the
// same link on each of them. Instances left out lose the link in both directions.
func setAgentInstanceModel(id, model string) (AgentInstanceMeta, error) {
	m, err := readAgentInstanceMeta(id)
	if err != nil {
		return AgentInstanceMeta{}, err
	}
	m.Model = strings.TrimSpace(model)
	if err := writeAgentInstanceMeta(m); err != nil {
		return AgentInstanceMeta{}, err
	}
	return m, nil
}

func setAgentInstancePeers(id string, peerIDs []string) (AgentInstanceMeta, error) {
	id = strings.TrimSpace(id)
	self, err := readAgentInstanceMeta(id)
	if err != nil {
		return AgentInstanceMeta{}, err
	}
	want := map[string]bool{}
	clean := make([]string, 0, len(peerIDs))
	for _, p := range peerIDs {
		p = strings.TrimSpace(p)
		if p == "" || p == id || want[p] {
			continue
		}
		if _, err := readAgentInstanceMeta(p); err != nil {
			return AgentInstanceMeta{}, fmt.Errorf("unknown agent %q", p)
		}
		want[p] = true
		clean = append(clean, p)
	}
	all, err := listAgentInstances()
	if err != nil {
		return AgentInstanceMeta{}, err
	}
	self.Peers = clean
	if err := writeAgentInstanceMeta(self); err != nil {
		return AgentInstanceMeta{}, err
	}
	for _, other := range all {
		if other.ID == id {
			continue
		}
		next := make([]string, 0, len(other.Peers)+1)
		for _, p := range other.Peers {
			if p != id && strings.TrimSpace(p) != "" {
				next = append(next, p)
			}
		}
		if want[other.ID] {
			next = append(next, id)
		}
		if sameToolSet(next, other.Peers) {
			continue
		}
		other.Peers = next
		if err := writeAgentInstanceMeta(other); err != nil {
			return AgentInstanceMeta{}, err
		}
	}
	return readAgentInstanceMeta(id)
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
