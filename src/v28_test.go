package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestInteractiveFlagTakesTerminal(t *testing.T) {
	for _, command := range []string{"pacman -Qi inetutils 2>/dev/null | head -20", "ssh host", "sudo pacman -S inetutils", "passwd", "pwd"} {
		a, _ := json.Marshal(map[string]any{"command": command, "interactive": true, "timeout_seconds": 10})
		if !toolNeedsTerminal("exec", string(a)) {
			t.Fatalf("interactive=true must take the terminal path: %q", command)
		}
		if !strings.Contains(renderToolCall("exec", string(a)), "interactive") {
			t.Fatalf("render must show interactive: %q", command)
		}
	}
}

func TestInteractiveFlagDoesNotReturnCommandOutput(t *testing.T) {
	if isTerminal(0) {
		t.Skip("requires a non-terminal stdin")
	}
	got := executeTool(context.Background(), "exec", `{"command":"uname -a | head -1","interactive":true,"timeout_seconds":10}`)
	if !strings.Contains(got, "interactive execution unavailable") || strings.Contains(got, "Linux") {
		t.Fatalf("interactive flag was masked: %q", got)
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
