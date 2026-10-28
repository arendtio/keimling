package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
