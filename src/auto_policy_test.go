package main

import (
	"os"
	"testing"
)

func TestRoutineCommandAcceptsDevelopmentWork(t *testing.T) {
	ok := []string{
		"gofmt -l .",
		"gofmt -w main.go",
		"go vet ./...",
		"go test ./...",
		"go build -o /tmp/x .",
		"cd src && gofmt -w .",
		"golangci-lint run",
		"staticcheck ./...",
		"prettier --write .",
		"git diff --stat",
		"rg -n TODO .",
		"ls -la",
		"grep -rn foo .",
		"go test ./... | tail -20",
	}
	for _, cmd := range ok {
		if !routineCommand(cmd) {
			t.Errorf("expected routine: %q", cmd)
		}
	}
	bad := []string{
		"gofmt -w main.go > out.txt",
		"go test ./... && rm -rf build",
		"sed -i 's/a/b/' file.go",
		"find . -name '*.go' -delete",
		"git commit -m x",
		"git push",
		"go run ./cmd/deploy",
		"make install",
		"cd .. && gofmt -l .",
		"cd /etc && ls",
		"curl https://example.com",
		"cat file | sh",
		"echo $(whoami)",
		"",
	}
	for _, cmd := range bad {
		if routineCommand(cmd) {
			t.Errorf("expected non-routine: %q", cmd)
		}
	}
}

func TestAutoApprovedFileEdits(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.WriteFile("a.txt", []byte("alpha\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, args string
		want       bool
	}{
		{"read_file", `{"path":"a.txt"}`, true},
		{"read_file", `{"path":"/etc/passwd"}`, false},
		{"replace_lines", `{"path":"a.txt"}`, true},
		{"replace_lines", `{"path":"missing.txt"}`, false},
		{"apply_patch", `{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ ~1\n-alpha\n+beta\n"}`, true},
		{"apply_patch", `{"patch":"--- /dev/null\n+++ b/new.txt\n@@ -0,0 +1,1 @@\n+x\n"}`, true},
		{"apply_patch", `{"patch":"--- a/a.txt\n+++ /dev/null\n@@ -1,1 +0,0 @@\n-alpha\n"}`, false},
		{"apply_patch", `{"patch":"--- /dev/null\n+++ b/../escape.txt\n@@ -0,0 +1,1 @@\n+x\n"}`, false},
		{"apply_patch", `{"patch":"broken"}`, false},
		{"exec", `{"command":"gofmt -w a.txt"}`, true},
		{"exec", `{"command":"rm a.txt"}`, false},
		{"exec", `{"command":"gofmt -w a.txt","interactive":true}`, false},
	}
	for _, c := range cases {
		if got := autoApproved(c.name, c.args); got != c.want {
			t.Errorf("%s %s: got %v want %v", c.name, c.args, got, c.want)
		}
	}
}
