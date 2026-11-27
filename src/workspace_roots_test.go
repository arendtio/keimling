package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The temp tree and this workspace's artifact copies count as part of the
// workspace; everything else outside the root does not.
// The temp tree and this workspace's artifact copies count as part of the
// workspace; everything else outside the root does not.
func TestTempAndArtifactsCountAsWorkspace(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "work", "demo")
	// The test data home lives under the real temp tree, so move the temp
	// allowance out of the way; otherwise every artifact path would count as
	// inside for the wrong reason.
	t.Setenv("TMPDIR", filepath.Join(string(filepath.Separator), "work", "scratch"))
	if !inWorkspace(root, root, filepath.Join(os.TempDir(), "scratch.txt")) {
		t.Fatal("a temp file was treated as outside the workspace")
	}
	if !inWorkspace(root, root, filepath.Join(os.TempDir(), "no", "such", "dir", "x.txt")) {
		t.Fatal("a missing temp path was treated as outside the workspace")
	}
	id := workspaceIDForRoot(root)
	artifact := filepath.Join(workspaceArtifactsDir(id), "chart.html")
	if !inWorkspace(root, root, artifact) {
		t.Fatalf("own artifact %q was treated as outside the workspace", artifact)
	}
	if inWorkspace(root, root, filepath.Join(workspaceArtifactsDir("someoneelse"), "chart.html")) {
		t.Fatal("another workspace's artifacts were treated as inside")
	}
	if inWorkspace(root, root, filepath.Join(workspaceDir(id), workspaceMetaFile)) {
		t.Fatal("workspace metadata was treated as inside")
	}
	if inWorkspace(root, root, filepath.Join(os.TempDir(), "..", "etc", "passwd")) {
		t.Fatal("a path that climbs out of the temp tree was treated as inside")
	}
}

// outsideTemp moves the temp allowance to a fresh directory, so a test can keep
// using the real temp tree as "outside the workspace"; the temp tree itself
// counts as part of the workspace.
func outsideTemp(t *testing.T) {
	t.Helper()
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "tmp"))
}

// outsideWorkspaceDir returns a directory outside root and outside the temp
// allowance. The temp tree counts as part of the workspace, so the test points
// TMPDIR into root and keeps the target beside it; a real temp dir would
// otherwise count as inside and hide an escape.
func outsideWorkspaceDir(t *testing.T, root string) string {
	t.Helper()
	t.Setenv("TMPDIR", filepath.Join(root, "tmp"))
	out := filepath.Join(filepath.Dir(root), "outside-"+filepath.Base(root))
	if err := os.MkdirAll(out, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(out) })
	return out
}

// A symlink cannot carry a path past the boundary, and a path below a symlinked
// directory is judged by its target.
func TestSymlinksCannotEscapeTheWorkspace(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if !inWorkspace(root, root, filepath.Join(root, "link.txt")) {
		t.Fatal("a symlink to a workspace file was treated as outside")
	}
	outside := outsideWorkspaceDir(t, root)
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if inWorkspace(root, root, filepath.Join(root, "escape")) {
		t.Fatal("a symlink out of the workspace was treated as inside")
	}
	if inWorkspace(root, root, filepath.Join(root, "escape", "new.txt")) {
		t.Fatal("a path below a symlink out of the workspace was treated as inside")
	}
}

// The write tools follow the same boundary: temp is allowed, a symlink escape is
// not.
func TestWriteFileFollowsTheWidenedBoundary(t *testing.T) {
	root := t.TempDir()
	a := &Agent{Cfg: Config{WorkspaceRoot: root}}
	ctx := a.bindDirs(context.Background())
	scratch := filepath.Join(os.TempDir(), "scratch.txt")
	t.Cleanup(func() { _ = os.Remove(scratch) })
	if got := executeTool(ctx, "write_file", `{"path":`+jsonQuote(scratch)+`,"content":"scratch\n"}`); !strings.Contains(got, "wrote") {
		t.Fatalf("temp write: %s", got)
	}
	if b, err := os.ReadFile(scratch); err != nil || string(b) != "scratch\n" {
		t.Fatalf("temp file: %q %v", b, err)
	}
	outside := outsideWorkspaceDir(t, root)
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	leak := filepath.Join(root, "escape", "leak.txt")
	if got := executeTool(ctx, "write_file", `{"path":`+jsonQuote(leak)+`,"content":"x\n"}`); !strings.Contains(got, "outside working directory") {
		t.Fatalf("write through a symlink out of the workspace: %s", got)
	}
	if _, err := os.Stat(filepath.Join(outside, "leak.txt")); !os.IsNotExist(err) {
		t.Fatalf("a file outside the workspace exists: %v", err)
	}
}
