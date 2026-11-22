package main

import (
	"os"
	"testing"
)

// deadWorkspace registers a workspace for a folder that is then removed, the way
// a merged worktree leaves its entry behind.
func deadWorkspace(t *testing.T) WorkspaceMeta {
	t.Helper()
	dir := t.TempDir()
	ws, err := ensureWorkspaceForRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	return ws
}

func TestPruneDropsAWorkspaceWhoseFolderIsGone(t *testing.T) {
	withTempAgentHomes(t)
	dead := deadWorkspace(t)
	removed := pruneMissingWorkspaces(nil)
	if len(removed) != 1 || removed[0].ID != dead.ID {
		t.Fatalf("removed = %+v", removed)
	}
	if _, err := readWorkspaceMeta(dead.ID); err == nil {
		t.Fatal("the entry should be gone")
	}
}

func TestPruneKeepsAWorkspaceWhoseFolderExists(t *testing.T) {
	withTempAgentHomes(t)
	alive, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if removed := pruneMissingWorkspaces(nil); len(removed) != 0 {
		t.Fatalf("removed = %+v", removed)
	}
	if _, err := readWorkspaceMeta(alive.ID); err != nil {
		t.Fatalf("a live workspace must stay: %v", err)
	}
}

func TestPruneKeepsAWorkspaceAnInstanceStillUses(t *testing.T) {
	withTempAgentHomes(t)
	dead := deadWorkspace(t)
	recipe, err := createAgentRecipe("Worker", "Work.")
	if err != nil {
		t.Fatal(err)
	}
	// The instance lists the dead workspace, so a later wake still finds it.
	inst, err := createAgentInstance(recipe, dead.ID, "Worker-1")
	if err != nil {
		t.Fatalf("instance in a registered workspace: %v", err)
	}
	if removed := pruneMissingWorkspaces(nil); len(removed) != 0 {
		t.Fatalf("removed = %+v", removed)
	}
	if _, err := readWorkspaceMeta(dead.ID); err != nil {
		t.Fatalf("a used workspace must stay: %v", err)
	}
	if err := deleteAgentInstance(inst.ID); err != nil {
		t.Fatal(err)
	}
	if removed := pruneMissingWorkspaces(nil); len(removed) != 1 || removed[0].ID != dead.ID {
		t.Fatalf("after the instance is gone the entry may go: %+v", removed)
	}
}

func TestPruneKeepsTheFocusedWorkspace(t *testing.T) {
	withTempAgentHomes(t)
	dead := deadWorkspace(t)
	keep := map[string]bool{dead.ID: true}
	if removed := pruneMissingWorkspaces(keep); len(removed) != 0 {
		t.Fatalf("removed = %+v", removed)
	}
	if _, err := readWorkspaceMeta(dead.ID); err != nil {
		t.Fatalf("the focused workspace must stay: %v", err)
	}
}

func TestPruneKeepsBuiltinAndGlobal(t *testing.T) {
	withTempAgentHomes(t)
	global, err := ensureGlobalWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	if removed := pruneMissingWorkspaces(nil); len(removed) != 0 {
		t.Fatalf("removed = %+v", removed)
	}
	if _, err := readWorkspaceMeta(global.ID); err != nil {
		t.Fatalf("the home workspace must stay: %v", err)
	}
}
