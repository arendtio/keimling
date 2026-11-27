package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThreadDirectoryChangeStaysOnThatThread(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a")
	other := filepath.Join(root, "b")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	slotA := &threadCwd{dir: root}
	slotB := &threadCwd{dir: root}
	ctxA := context.WithValue(context.Background(), cwdSlotKey{}, slotA)
	ctxA = context.WithValue(ctxA, wsRootKey{}, root)
	ctxB := context.WithValue(context.Background(), cwdSlotKey{}, slotB)
	ctxB = context.WithValue(ctxB, wsRootKey{}, root)

	out := executeTool(ctxA, "exec", `{"command":"cd a","timeout_seconds":5}`)
	if !strings.Contains(out, "exit_code=0") {
		t.Fatal(out)
	}
	if filepath.Clean(slotA.get()) != filepath.Clean(sub) {
		t.Fatalf("thread A directory %q, want %q", slotA.get(), sub)
	}
	if filepath.Clean(slotB.get()) != filepath.Clean(root) {
		t.Fatalf("thread B moved to %q", slotB.get())
	}

	out = executeTool(ctxA, "exec", `{"command":"pwd -P","timeout_seconds":5}`)
	if !strings.Contains(out, filepath.Clean(sub)) {
		t.Fatalf("next command in A started elsewhere:\n%s", out)
	}
	out = executeTool(ctxB, "exec", `{"command":"cd b && pwd -P","timeout_seconds":5}`)
	if !strings.Contains(out, filepath.Clean(other)) {
		t.Fatalf("thread B cd:\n%s", out)
	}
	if filepath.Clean(slotA.get()) != filepath.Clean(sub) {
		t.Fatalf("thread A moved to %q after B's cd", slotA.get())
	}
	if !inWorkspace(root, sub, "note.txt") {
		t.Fatal("a path inside the workspace was treated as outside")
	}
	// The temp tree counts as part of the workspace, and this root lives in it,
	// so climbing above the root stays inside. A path outside the temp tree
	// must still be refused.
	if inWorkspace(root, sub, filepath.Join(os.TempDir(), "..", "etc", "passwd")) {
		t.Fatal("a path outside the workspace was treated as inside")
	}
}

func TestForkedAgentKeepsTheThreadDirectory(t *testing.T) {
	parent := &Agent{Cfg: Config{ThreadID: "one", WorkspaceRoot: "/work"}}
	parent.resetThreadDir("/work/one")
	fork := &Agent{Cfg: Config{ThreadID: "one", WorkspaceRoot: "/work"}, dirs: parent.dirTable()}
	if fork.threadDir() != "/work/one" {
		t.Fatalf("fork directory %q", fork.threadDir())
	}
	fork.resetThreadDir("/work/one/deeper")
	if parent.threadDir() != "/work/one/deeper" {
		t.Fatalf("parent directory %q", parent.threadDir())
	}
	other := &Agent{Cfg: Config{ThreadID: "two", WorkspaceRoot: "/work"}, dirs: parent.dirTable()}
	if other.threadDir() != "/work" {
		t.Fatalf("other thread directory %q", other.threadDir())
	}
}
