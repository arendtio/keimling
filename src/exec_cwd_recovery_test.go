package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cwdContext builds the tool context of one instance: a thread directory and
// the workspace root it falls back to.
func cwdContext(root, dir string) (context.Context, *threadCwd) {
	slot := &threadCwd{dir: dir}
	ctx := context.WithValue(context.Background(), cwdSlotKey{}, slot)
	ctx = context.WithValue(ctx, wsRootKey{}, root)
	return ctx, slot
}

func resolvedTempDir(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// A removed worktree leaves the instance without a working directory. The
// command runs in the workspace root instead of failing.
func TestExecFallsBackWhenWorkingDirectoryIsGone(t *testing.T) {
	root := resolvedTempDir(t)
	gone := filepath.Join(root, "worktree")
	if err := os.Mkdir(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, slot := cwdContext(root, gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	out := executeTool(ctx, "exec", `{"command":"pwd -P","timeout_seconds":5}`)
	if !strings.HasPrefix(out, "exit_code=0") {
		t.Fatalf("command failed instead of falling back:\n%s", out)
	}
	if !strings.Contains(out, "working directory fallback") || !strings.Contains(out, gone) {
		t.Fatalf("reply does not name the fallback:\n%s", out)
	}
	if !strings.Contains(out, filepath.Clean(root)) {
		t.Fatalf("reply does not name the workspace root:\n%s", out)
	}
	if filepath.Clean(slot.get()) != filepath.Clean(root) {
		t.Fatalf("thread directory %q, want the workspace root %q", slot.get(), root)
	}
	if _, err := os.Stat(gone); err == nil {
		t.Fatal("the missing directory was created")
	}
}

// The fallback is written back, so the next command is quiet and succeeds.
func TestExecAfterFallbackStaysQuiet(t *testing.T) {
	root := resolvedTempDir(t)
	gone := filepath.Join(root, "nested", "worktree")
	if err := os.MkdirAll(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, slot := cwdContext(root, gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	first := executeTool(ctx, "exec", `{"command":"echo one","timeout_seconds":5}`)
	if !strings.Contains(first, "working directory fallback") {
		t.Fatalf("first reply has no fallback notice:\n%s", first)
	}
	second := executeTool(ctx, "exec", `{"command":"pwd -P","timeout_seconds":5}`)
	if !strings.HasPrefix(second, "exit_code=0") {
		t.Fatalf("second reply is not a plain success:\n%s", second)
	}
	if strings.Contains(second, "fallback") || strings.Contains(second, "no longer exists") {
		t.Fatalf("second reply repeats the notice:\n%s", second)
	}
	if !strings.Contains(second, filepath.Clean(root)) {
		t.Fatalf("second command did not start in the workspace root:\n%s", second)
	}
	if filepath.Clean(slot.get()) != filepath.Clean(root) {
		t.Fatalf("thread directory %q after the second command", slot.get())
	}
}

// A failing command after the fallback keeps its own exit status, so the
// notice cannot turn a failure into a success.
func TestFailingCommandAfterFallbackKeepsItsStatus(t *testing.T) {
	root := resolvedTempDir(t)
	gone := filepath.Join(root, "worktree")
	if err := os.Mkdir(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, _ := cwdContext(root, gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	out := executeTool(ctx, "exec", `{"command":"exit 3","timeout_seconds":5}`)
	if !strings.HasPrefix(out, "exit_code=3") {
		t.Fatalf("the notice replaced the exit status:\n%s", out)
	}
	if !strings.Contains(out, "working directory fallback") {
		t.Fatalf("reply has no fallback notice:\n%s", out)
	}
	if markToolOK("exec", out, "") {
		t.Fatal("a failed command after the fallback was shown as a success")
	}
}

// A shell that cannot be started names the shell, not a directory.
func TestStartErrorNamesTheShell(t *testing.T) {
	root := resolvedTempDir(t)
	shell := filepath.Join(root, "noshell")
	cmd := shellCommand(shell, "true")
	cmd.Dir = root
	err := cmd.Run()
	if err == nil {
		t.Fatal("starting a missing shell succeeded")
	}
	msg := startErrorText(shell, root, err)
	if !strings.Contains(msg, shell) || !strings.Contains(msg, "shell") {
		t.Fatalf("message does not name the shell: %q", msg)
	}
	if strings.Contains(msg, "working directory") {
		t.Fatalf("message blames the directory: %q", msg)
	}
}

// A missing working directory names the directory, not the shell.
func TestStartErrorNamesTheMissingDirectory(t *testing.T) {
	root := resolvedTempDir(t)
	gone := filepath.Join(root, "gone")
	shell := selectedShell()
	cmd := shellCommand(shell, "true")
	cmd.Dir = gone
	err := cmd.Run()
	if err == nil {
		t.Fatal("starting in a missing directory succeeded")
	}
	msg := startErrorText(shell, gone, err)
	if !strings.Contains(msg, "working directory") || !strings.Contains(msg, gone) {
		t.Fatalf("message does not name the directory: %q", msg)
	}
	if strings.Contains(msg, "could not be started") {
		t.Fatalf("message blames the shell: %q", msg)
	}
}

// A path the command itself names stays the caller's error, never a fallback.
func TestCommandCdIntoMissingPathStaysAnError(t *testing.T) {
	root := resolvedTempDir(t)
	gone := filepath.Join(root, "not-there")
	ctx, slot := cwdContext(root, root)

	out := executeTool(ctx, "exec", `{"command":"cd `+gone+` && pwd -P","timeout_seconds":5}`)
	if !strings.Contains(out, "exit_code=1") {
		t.Fatalf("expected the caller's own failure:\n%s", out)
	}
	if strings.Contains(out, "working directory fallback") {
		t.Fatalf("a named path fell back:\n%s", out)
	}
	if filepath.Clean(slot.get()) != filepath.Clean(root) {
		t.Fatalf("thread directory moved to %q", slot.get())
	}
}

// An existing directory keeps the reply shape it had before.
func TestExecInExistingDirectoryIsUnchanged(t *testing.T) {
	root := resolvedTempDir(t)
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, slot := cwdContext(root, sub)

	out := executeTool(ctx, "exec", `{"command":"echo hi","timeout_seconds":5}`)
	if out != "exit_code=0\nhi\n" {
		t.Fatalf("unexpected reply: %q", out)
	}
	if filepath.Clean(slot.get()) != filepath.Clean(sub) {
		t.Fatalf("thread directory %q", slot.get())
	}
}
