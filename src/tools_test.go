package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileTools(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(path, []byte("alpha\nbeta\ngamma\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got := executeTool(context.Background(), "read_file", `{"path":"`+path+`","start_line":2,"end_line":2}`)
	hash := hashBytes([]byte("beta\n"))
	if !strings.Contains(got, hash) || !strings.HasSuffix(got, "beta\n") {
		t.Fatal(got)
	}
	args := fmt.Sprintf(`{"path":%q,"start_line":2,"end_line":2,"expected_hash":%q,"content":"delta\n"}`, path, hash)
	if got = executeTool(context.Background(), "replace_lines", args); got != "replaced successfully" {
		t.Fatal(got)
	}
	if got = executeTool(context.Background(), "replace_lines", args); !strings.Contains(got, "hash mismatch") {
		t.Fatal(got)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "alpha\ndelta\ngamma\n" {
		t.Fatal(string(b))
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
}
func TestMarkdownLinks(t *testing.T) {
	got := formatTerminal("[Arch wiki](https://wiki.archlinux.org/)")
	if got != "Arch wiki (https://wiki.archlinux.org/)" {
		t.Fatal(got)
	}
}
