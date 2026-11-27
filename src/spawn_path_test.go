package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Feature 013: a spawn may name the directory the instance works in, so a
// developer agent gets its own git worktree instead of the workspace of the
// caller. Without the argument nothing changes.

func TestSpawnAgentWithPathStartsInThatWorkspace(t *testing.T) {
	withTempAgentHomes(t)
	t.Setenv("KEIMLING_HOST", "")
	t.Setenv("KEIMLING_SECRET", "")
	caller, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	worktree := t.TempDir()
	thread := &Agent{Cfg: Config{WorkspaceID: caller.ID}}
	saved := thread.runDefineAgentTool("save_agent", `{"id":"","name":"Developer","instructions":"Build it.","tools":["exec","read_file","write_file","edit_file","list_agents","message_agent","spawn_agent","save_agent","finish_task","update_plan"]}`)
	if strings.Contains(saved, "error") {
		t.Fatal(saved)
	}
	started := thread.runDefineAgentTool("spawn_agent", `{"definition":"Developer","name":"","path":"`+worktree+`"}`)
	if strings.Contains(started, "error") {
		t.Fatal(started)
	}
	inst := onlyInstance(t)
	want, err := ensureWorkspaceForRoot(worktree)
	if err != nil {
		t.Fatal(err)
	}
	if inst.CurrentWorkspace != want.ID || !agentInstanceAllowsWorkspace(inst, want.ID) {
		t.Fatalf("instance workspace: %+v", inst)
	}
	if !agentInstanceAllowsWorkspace(inst, caller.ID) {
		t.Fatalf("caller workspace left the ACL: %+v", inst)
	}
	if root := strings.TrimSpace(want.Root); root != canonicalPath(worktree) {
		t.Fatalf("workspace root %q", root)
	}
	a := &Agent{Cfg: Config{
		AgentInstanceID: inst.ID,
		WorkspaceID:     inst.CurrentWorkspace,
		WorkspaceRoot:   want.Root,
	}}
	ctx := a.bindDirs(context.Background())
	inside := filepath.Join(want.Root, "notes.txt")
	if got := executeTool(ctx, "write_file", `{"path":"`+inside+`","content":"in the worktree\n"}`); !strings.Contains(got, "wrote") {
		t.Fatalf("write inside: %s", got)
	}
	if b, err := os.ReadFile(inside); err != nil || string(b) != "in the worktree\n" {
		t.Fatalf("worktree file: %q %v", b, err)
	}
	// Both trees live in the temp directory, which counts as part of every
	// workspace. Move the temp allowance into the instance's own root so the
	// caller's tree is genuinely outside it.
	t.Setenv("TMPDIR", filepath.Join(want.Root, "tmp"))
	outside := filepath.Join(caller.Root, "leak.txt")
	if got := executeTool(ctx, "write_file", `{"path":"`+outside+`","content":"outside\n"}`); !strings.Contains(got, "outside working directory") {
		t.Fatalf("write outside: %s", got)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("wrote outside the workspace: %v", err)
	}
}

func TestSpawnAgentWithoutPathUsesTheCallerWorkspaceOnly(t *testing.T) {
	withTempAgentHomes(t)
	t.Setenv("KEIMLING_HOST", "")
	t.Setenv("KEIMLING_SECRET", "")
	caller, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	thread := &Agent{Cfg: Config{WorkspaceID: caller.ID}}
	if saved := thread.runDefineAgentTool("save_agent", `{"id":"","name":"Runner","instructions":"Run.","tools":["exec","read_file","write_file","edit_file","list_agents","message_agent","spawn_agent","save_agent","finish_task","update_plan"]}`); strings.Contains(saved, "error") {
		t.Fatal(saved)
	}
	if started := thread.runDefineAgentTool("spawn_agent", `{"definition":"Runner","name":""}`); strings.Contains(started, "error") {
		t.Fatal(started)
	}
	inst := onlyInstance(t)
	if inst.CurrentWorkspace != caller.ID || len(inst.Workspaces) != 1 || inst.Workspaces[0] != caller.ID {
		t.Fatalf("instance without path: %+v", inst)
	}
}

func TestSpawnAgentWithUnknownPathSpawnsNothing(t *testing.T) {
	withTempAgentHomes(t)
	t.Setenv("KEIMLING_HOST", "")
	t.Setenv("KEIMLING_SECRET", "")
	caller, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	thread := &Agent{Cfg: Config{WorkspaceID: caller.ID}}
	if saved := thread.runDefineAgentTool("save_agent", `{"id":"","name":"Runner","instructions":"Run.","tools":["exec","read_file","write_file","edit_file","list_agents","message_agent","spawn_agent","save_agent","finish_task","update_plan"]}`); strings.Contains(saved, "error") {
		t.Fatal(saved)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	started := thread.runDefineAgentTool("spawn_agent", `{"definition":"Runner","name":"","path":"`+missing+`"}`)
	if !strings.Contains(started, "error") || !strings.Contains(started, "existing directory") {
		t.Fatal(started)
	}
	list, err := listAgentInstances()
	if err != nil || len(list) != 0 {
		t.Fatalf("instances: %+v %v", list, err)
	}
}

func TestSpawnTeamWithPathStartsEveryInstanceThere(t *testing.T) {
	withTempAgentHomes(t)
	t.Setenv("KEIMLING_HOST", "")
	t.Setenv("KEIMLING_SECRET", "")
	caller, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	worktree := t.TempDir()
	a := &Agent{Cfg: Config{WorkspaceID: caller.ID}}
	for _, name := range []string{"Reader", "Writer"} {
		args, err := json.Marshal(map[string]any{"id": "", "name": name, "instructions": "Do " + name, "tools": []string{"message_agent", "list_agents"}})
		if err != nil {
			t.Fatal(err)
		}
		if saved := a.runDefineAgentTool("save_agent", string(args)); strings.Contains(saved, "error") {
			t.Fatal(saved)
		}
	}
	if made := a.runDefineAgentTool("save_team", `{"id":"","name":"Crew","definitions":["Reader","Writer"]}`); strings.Contains(made, "error") {
		t.Fatal(made)
	}
	if spawned := a.runDefineAgentTool("spawn_team", `{"team":"Crew","path":"`+worktree+`"}`); strings.Contains(spawned, "error") {
		t.Fatal(spawned)
	}
	want, err := ensureWorkspaceForRoot(worktree)
	if err != nil {
		t.Fatal(err)
	}
	list, err := listAgentInstances()
	if err != nil || len(list) != 2 {
		t.Fatalf("instances: %+v %v", list, err)
	}
	for _, inst := range list {
		if inst.CurrentWorkspace != want.ID || !agentInstanceAllowsWorkspace(inst, want.ID) {
			t.Fatalf("team instance workspace: %+v", inst)
		}
	}
}

func TestHostSpawnAgentWithPathUsesThatWorkspace(t *testing.T) {
	withTempAgentHomes(t)
	caller, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	worktree := t.TempDir()
	recipe, err := createAgentRecipe("Host Developer", "Build.")
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]string{"recipe_id": recipe.ID, "workspace_id": caller.ID, "path": worktree})
	if err != nil {
		t.Fatal(err)
	}
	h := &Host{}
	req := httptest.NewRequest(http.MethodPost, "/api/agents/instances", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.handleAgentInstances(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	want, err := ensureWorkspaceForRoot(worktree)
	if err != nil {
		t.Fatal(err)
	}
	inst := onlyInstance(t)
	if inst.CurrentWorkspace != want.ID || !agentInstanceAllowsWorkspace(inst, want.ID) {
		t.Fatalf("host instance workspace: %+v", inst)
	}
	if !agentInstanceAllowsWorkspace(inst, caller.ID) {
		t.Fatalf("caller workspace left the ACL: %+v", inst)
	}
}

func onlyInstance(t *testing.T) AgentInstanceMeta {
	t.Helper()
	list, err := listAgentInstances()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one instance, got %+v", list)
	}
	return list[0]
}
