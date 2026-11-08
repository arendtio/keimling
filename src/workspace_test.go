package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "keimling-test-data-")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_DATA_HOME", dir)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestWorkspaceIDForRootStable(t *testing.T) {
	dir := t.TempDir()
	a := workspaceIDForRoot(dir)
	b := workspaceIDForRoot(dir + string(filepath.Separator))
	if a == "" || a != b {
		t.Fatalf("ids %q %q", a, b)
	}
	home, _ := os.UserHomeDir()
	if home != "" && workspaceIDForRoot(home) != workspaceGlobalID {
		t.Fatalf("home should be global, got %q", workspaceIDForRoot(home))
	}
}

func TestResolveStartupWorkspaceCreatesThread(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	root := t.TempDir()
	ws, threadID, session, err := resolveStartupWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Root != canonicalPath(root) {
		t.Fatalf("root %q want %q", ws.Root, canonicalPath(root))
	}
	if threadID == "" || session == "" {
		t.Fatal("expected thread+session")
	}
	cwd, _ := os.Getwd()
	if canonicalPath(cwd) != ws.Root {
		t.Fatalf("cwd %q want %q", cwd, ws.Root)
	}
	if _, err := os.Stat(filepath.Dir(session)); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureGlobalWorkspace(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	m, err := ensureGlobalWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != workspaceGlobalID || !m.Builtin || m.Root == "" {
		t.Fatalf("%#v", m)
	}
	m2, err := ensureGlobalWorkspace()
	if err != nil || m2.ID != m.ID {
		t.Fatalf("%v %#v", err, m2)
	}
}

func TestCreateAndListThreads(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, session, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	if th.ID == "" || session == "" {
		t.Fatal(th, session)
	}
	list, err := listThreads(ws.ID)
	if err != nil || len(list) != 1 || !list[0].Active {
		t.Fatalf("%v %#v", err, list)
	}
}

func TestDeleteThreadDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, session, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(session, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := deleteThreadDir(ws.ID, th.ID); err != nil {
		t.Fatal(err)
	}
	list, err := listThreads(ws.ID)
	if err != nil || len(list) != 0 {
		t.Fatalf("%v %#v", err, list)
	}
	if err := deleteThreadDir(ws.ID, th.ID); err == nil {
		t.Fatal("expected missing thread error")
	}
}

func TestHideWorkspaceLeavesFiles(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	root := t.TempDir()
	marker := filepath.Join(root, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := ensureWorkspaceForRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	_, session, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(session, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	metaPath := filepath.Join(workspaceDir(ws.ID), workspaceMetaFile)
	if err := hideWorkspace(ws.ID); err != nil {
		t.Fatal(err)
	}
	list, err := listWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range list {
		if m.ID == ws.ID {
			t.Fatal("hidden workspace still listed")
		}
	}
	for _, path := range []string{marker, session, metaPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("file removed: %s: %v", path, err)
		}
	}
	if err := hideWorkspace(workspaceGlobalID); err == nil {
		t.Fatal("home workspace was removed from the menu")
	}
	again, err := ensureWorkspaceForRoot(root)
	if err != nil || again.Hidden {
		t.Fatalf("opening the folder should restore the menu entry: %#v %v", again, err)
	}
	list, err = listWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range list {
		if m.ID == ws.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("restored workspace missing from the menu")
	}
}

func TestWorkspaceThreadListSkipsAgentPlaceholder(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeThreadMeta(ws.ID, ThreadMeta{ID: "main", Title: "New thread"}); err != nil {
		t.Fatal(err)
	}
	list, err := listThreads(ws.ID)
	if err != nil || len(list) != 1 || list[0].ID != th.ID {
		t.Fatalf("list: %v %#v", err, list)
	}
	if err := writeActiveThreadID(ws.ID, "main"); err != nil {
		t.Fatal(err)
	}
	id, _, err := ensureActiveThread(ws.ID)
	if err != nil || id != th.ID {
		t.Fatalf("active %s %v", id, err)
	}
	got, err := readActiveThreadID(ws.ID)
	if err != nil || got != th.ID {
		t.Fatalf("stored %q %v", got, err)
	}
}

func TestRenameWorkspaceKeepsFolder(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	root := t.TempDir()
	marker := filepath.Join(root, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := ensureWorkspaceNamed(root, "Alpha")
	if err != nil || ws.Name != "Alpha" || ws.Root == "" {
		t.Fatalf("%#v %v", ws, err)
	}
	renamed, err := renameWorkspace(ws.ID, "  Beta  ")
	if err != nil || renamed.Name != "Beta" || renamed.Root != ws.Root {
		t.Fatalf("%#v %v", renamed, err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal(err)
	}
	if _, err := renameWorkspace(ws.ID, " "); err == nil {
		t.Fatal("empty name was stored")
	}
	again, err := ensureWorkspaceNamed(root, "")
	if err != nil || again.Name != "Beta" {
		t.Fatalf("opening again should keep the chosen name: %#v %v", again, err)
	}
}

func TestShowWorkspaceRestoresMenu(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	root := t.TempDir()
	ws, err := ensureWorkspaceNamed(root, "Kept")
	if err != nil {
		t.Fatal(err)
	}
	if err := hideWorkspace(ws.ID); err != nil {
		t.Fatal(err)
	}
	if err := showWorkspace(ws.ID); err != nil {
		t.Fatal(err)
	}
	list, err := listWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range list {
		if m.ID == ws.ID && m.Name == "Kept" {
			found = true
		}
	}
	if !found {
		t.Fatal("restored workspace missing")
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal(err)
	}
}
