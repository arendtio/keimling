package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempAgentHomes(t *testing.T) (cleanup func()) {
	t.Helper()
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	data := filepath.Join(root, "share")
	_ = os.MkdirAll(cfg, 0o700)
	_ = os.MkdirAll(data, 0o700)
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("XDG_DATA_HOME", data)
	return func() {}
}

func TestNewAgentIDsHaveNoFullStop(t *testing.T) {
	id := newAgentInstanceID()
	recipe := newRecipeID()
	thread := newThreadID()
	if strings.Contains(id, ".") || !strings.HasPrefix(id, "agent-") {
		t.Fatalf("instance id %q", id)
	}
	if strings.Contains(thread, ".") {
		t.Fatalf("thread id %q", thread)
	}
	if strings.Contains(recipe, ".") || !strings.HasPrefix(recipe, "recipe-") {
		t.Fatalf("recipe id %q", recipe)
	}
	parts := strings.Split(id, "-")
	if len(parts) < 4 || len(parts[len(parts)-1]) != 9 {
		t.Fatalf("expected prefix-date-time-fraction, got %q", id)
	}
}

func TestAgentRecipeAndInstanceRoundTrip(t *testing.T) {
	withTempAgentHomes(t)
	wsRoot := t.TempDir()
	ws, err := ensureWorkspaceForRoot(wsRoot)
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Tester", "Follow ACL.")
	if err != nil {
		t.Fatal(err)
	}
	if recipe.ID == "" || recipe.Name != "Tester" {
		t.Fatalf("recipe: %+v", recipe)
	}
	got, err := readAgentRecipe(recipe.ID)
	if err != nil || got.Instructions != "Follow ACL." {
		t.Fatalf("read recipe: %+v %v", got, err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Tester-1")
	if err != nil {
		t.Fatal(err)
	}
	if inst.CurrentWorkspace != ws.ID || !agentInstanceAllowsWorkspace(inst, ws.ID) {
		t.Fatalf("instance ACL: %+v", inst)
	}
	if _, err := os.Stat(agentInstanceSessionPath(inst.ID)); !os.IsNotExist(err) && err != nil {
		t.Fatal(err)
	}
	otherRoot := t.TempDir()
	other, err := ensureWorkspaceForRoot(otherRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := setAgentInstanceWorkspace(inst.ID, other.ID); err == nil {
		t.Fatal("expected ACL reject")
	}
	inst.Workspaces = append(inst.Workspaces, other.ID)
	if err := writeAgentInstanceMeta(inst); err != nil {
		t.Fatal(err)
	}
	if err := setAgentInstanceWorkspace(inst.ID, other.ID); err != nil {
		t.Fatal(err)
	}
	again, err := readAgentInstanceMeta(inst.ID)
	if err != nil || again.CurrentWorkspace != other.ID {
		t.Fatalf("current workspace: %+v %v", again, err)
	}
	list, err := listAgentInstancesForWorkspace(ws.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list for ws: %v %+v", err, list)
	}
	if err := deleteAgentInstance(inst.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := readAgentInstanceMeta(inst.ID); err == nil {
		t.Fatal("expected missing after delete")
	}
}

func TestChangeWorkspaceToolACL(t *testing.T) {
	withTempAgentHomes(t)
	wsRoot := t.TempDir()
	ws, err := ensureWorkspaceForRoot(wsRoot)
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("CD", "cd carefully")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "CD-1")
	if err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(ws.Root); err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{
		AgentInstanceID:   inst.ID,
		AllowedWorkspaces: []string{ws.ID},
		WorkspaceID:       ws.ID,
		WorkspaceRoot:     ws.Root,
	}}
	otherRoot := t.TempDir()
	other, err := ensureWorkspaceForRoot(otherRoot)
	if err != nil {
		t.Fatal(err)
	}
	out := a.changeWorkspace(`{"workspace_id":"` + other.ID + `"}`)
	if out == "" || out[:len("change_workspace error")] != "change_workspace error" {
		t.Fatalf("expected ACL error, got %q", out)
	}
	out = a.changeWorkspace(`{"workspace_id":"` + ws.ID + `"}`)
	if out[:len("changed workspace")] != "changed workspace" {
		t.Fatalf("expected success, got %q", out)
	}
}

func TestResolveWorkspaceAgentMentions(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	other, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Pair", "Talk.")
	if err != nil {
		t.Fatal(err)
	}
	ada, err := createAgentInstance(recipe, ws.ID, "Entertainer 4")
	if err != nil {
		t.Fatal(err)
	}
	ada2, err := createAgentInstance(recipe, ws.ID, "Entertainer 4")
	if err != nil {
		t.Fatal(err)
	}
	long, err := createAgentInstance(recipe, ws.ID, "Entertainer 40")
	if err != nil {
		t.Fatal(err)
	}
	outsider, err := createAgentInstance(recipe, other.ID, "Outsider")
	if err != nil {
		t.Fatal(err)
	}
	got, ambiguous, err := resolveWorkspaceAgentMentions(ws.ID, "please @Entertainer 40 look", nil)
	if err != nil || ambiguous || len(got) != 1 || got[0].ID != long.ID {
		t.Fatalf("longer name: %+v %v %v", got, ambiguous, err)
	}
	got, ambiguous, err = resolveWorkspaceAgentMentions(ws.ID, "@Entertainer 4 hello", nil)
	if err != nil || !ambiguous || len(got) != 0 {
		t.Fatalf("shared name needs a choice: %+v %v %v", got, ambiguous, err)
	}
	got, ambiguous, err = resolveWorkspaceAgentMentions(ws.ID, "@Entertainer 4 hello", []agentMentionPick{{ID: ada2.ID, Name: "Entertainer 4"}})
	if err != nil || ambiguous || len(got) != 1 || got[0].ID != ada2.ID {
		t.Fatalf("picked id: %+v %v %v", got, ambiguous, err)
	}
	got, ambiguous, err = resolveWorkspaceAgentMentions(ws.ID, "mail me at a@Entertainer 4", []agentMentionPick{{ID: ada.ID}})
	if err != nil || ambiguous || len(got) != 0 {
		t.Fatalf("@ inside a word is not an address: %+v %v %v", got, ambiguous, err)
	}
	got, ambiguous, err = resolveWorkspaceAgentMentions(ws.ID, "@Outsider hi", nil)
	if err != nil || ambiguous || len(got) != 0 || outsider.ID == "" {
		t.Fatalf("other workspace is not offered: %+v %v %v", got, ambiguous, err)
	}
}
