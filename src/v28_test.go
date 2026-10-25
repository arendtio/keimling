package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestReadOnlyPipelineDoesNotTakePTY(t *testing.T) {
	command := `pacman -Qi inetutils 2>/dev/null | head -20`
	args, _ := json.Marshal(map[string]any{"command": command, "interactive": true, "timeout_seconds": 10})
	if toolNeedsTerminal("exec", string(args)) {
		t.Fatal("read-only pacman query must not take over terminal")
	}
	if strings.Contains(renderToolCall("exec", string(args)), "interactive") {
		t.Fatal("render must show noninteractive execution")
	}
	for _, command := range []string{"ssh host", "sudo pacman -S inetutils", "passwd", "pacman -Syu"} {
		a, _ := json.Marshal(map[string]any{"command": command, "interactive": true, "timeout_seconds": 10})
		if !toolNeedsTerminal("exec", string(a)) {
			t.Fatalf("genuine interactive operation incorrectly downgraded: %q", command)
		}
	}
}

func TestCommandWithBadInteractiveFlagRunsNormally(t *testing.T) {
	got := executeTool(context.Background(), "exec", `{"command":"uname -a | head -1","interactive":true,"timeout_seconds":10}`)
	if !strings.Contains(got, "exit_code=0") || !strings.Contains(got, "Linux") {
		t.Fatalf("unexpected output %q", got)
	}
}

// Run with: script -q -c 'go test -run TestPTYSmoke -v' /dev/null
// This exercises an actual controlling terminal, unlike mocked unit tests.
func TestPTYSmoke(t *testing.T) {
	if !isTerminal(0) {
		t.Skip("requires real TTY")
	}
	got := interactiveExec(context.Background(), selectedShell(), "printf 'pty-ok\\n'", 4*time.Second)
	if !strings.Contains(got, "exit_code=0") {
		t.Fatalf("PTY process did not terminate cleanly: %q", got)
	}
}
