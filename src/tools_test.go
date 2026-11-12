package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFileTools(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	path := "test.txt"
	if err := os.WriteFile(path, []byte("alpha\nbeta\ngamma\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got := executeTool(context.Background(), "read_file", `{"path":"test.txt","start_line":2,"end_line":2}`)
	hash := hashBytes([]byte("beta\n"))
	if !strings.Contains(got, hash) || !strings.HasSuffix(got, "beta\n") {
		t.Fatal(got)
	}
	args := `{"path":"test.txt","old_string":"beta\n","new_string":"delta\n","replace_all":false}`
	if got = executeTool(context.Background(), "edit_file", args); got != "edited successfully" {
		t.Fatal(got)
	}
	if got = executeTool(context.Background(), "edit_file", args); !strings.Contains(got, "old_string not found") {
		t.Fatal(got)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "alpha\ndelta\ngamma\n" {
		t.Fatal(string(b))
	}
}
func TestReadFileLargeWholeFileIsAPreview(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	body := strings.Repeat("line\n", 2000) + "MARKER-TAIL\n"
	if err := os.WriteFile("big.txt", []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got := executeTool(context.Background(), "read_file", `{"path":"big.txt","start_line":0,"end_line":0}`)
	if strings.Contains(got, "MARKER-TAIL") || !strings.Contains(got, "path: big.txt") || !strings.Contains(got, "preview:") {
		t.Fatalf("whole file: %s", got)
	}
	got = executeTool(context.Background(), "read_file", `{"path":"big.txt","start_line":2001,"end_line":2001}`)
	if !strings.Contains(got, "MARKER-TAIL") || !strings.Contains(got, "hash:") {
		t.Fatalf("range: %s", got)
	}
	if err := os.WriteFile("small.txt", []byte("exact\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got = executeTool(context.Background(), "read_file", `{"path":"small.txt","start_line":0,"end_line":0}`)
	if !strings.Contains(got, "exact\n") || !strings.Contains(got, "hash:") {
		t.Fatalf("small: %s", got)
	}
}

func TestExec(t *testing.T) {
	got := executeTool(context.Background(), "exec", `{"command":"printf ok","timeout_seconds":5}`)
	if !strings.Contains(got, "exit_code=0") || !strings.Contains(got, "ok") {
		t.Fatal(got)
	}
}
func TestJournal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "j.jsonl")
	j, e := OpenJournal(path, false)
	if e != nil {
		t.Fatal(e)
	}
	if e := j.AppendItem([]byte(`{"role":"user","content":"hi"}`)); e != nil {
		t.Fatal(e)
	}
	j.Close()
	j, e = OpenJournal(path, true)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	if len(j.History) != 1 {
		t.Fatalf("got %d", len(j.History))
	}
}

func TestShellAndExit(t *testing.T) {
	if selectedShell() == "/bin/bash" {
		got := executeTool(context.Background(), "exec", `{"command":"set -o pipefail; false | true","timeout_seconds":5}`)
		if !strings.Contains(got, "exit_code=1") {
			t.Fatalf("pipefail: %s", got)
		}
	}
	got := executeTool(context.Background(), "exec", `{"command":"printf 'out\\n'; printf 'err\\n' >&2; exit 7","timeout_seconds":5}`)
	if !strings.Contains(got, "exit_code=7") || !strings.Contains(got, "out") || !strings.Contains(got, "err") {
		t.Fatal(got)
	}
	cmd := shellCommand(selectedShell(), "printf ok")
	if runtime.GOOS == "windows" {
		if len(cmd.Args) < 5 || cmd.Args[len(cmd.Args)-2] != "-Command" || cmd.Args[len(cmd.Args)-1] != "printf ok" {
			t.Fatalf("windows shell args: %q", cmd.Args)
		}
		return
	}
	if len(cmd.Args) != 3 || cmd.Args[1] != "-c" || cmd.Args[2] != "printf ok" {
		t.Fatalf("shell args: %q", cmd.Args)
	}
}
