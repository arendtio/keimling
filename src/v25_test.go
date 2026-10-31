package main

import (
	"context"
	"strings"
	"testing"
)

func TestInteractiveTrueAlwaysNeedsTerminal(t *testing.T) {
	for _, c := range []string{"lscpu", "uname -a", "date", "whoami", "pwd", "ls", "pacman -Qi inetutils"} {
		args := `{"command":"` + c + `","interactive":true}`
		if !toolNeedsTerminal("exec", args) {
			t.Fatalf("interactive=true must take the terminal path: %q", c)
		}
		if !strings.Contains(renderToolCall("exec", args), "interactive") {
			t.Fatalf("render must show interactive: %q", c)
		}
	}
	if toolNeedsTerminal("exec", `{"command":"pwd","interactive":false}`) {
		t.Fatal("interactive=false must not take the terminal path")
	}
	if isTerminal(0) {
		return
	}
	got := executeTool(context.Background(), "exec", `{"command":"uname -s","interactive":true}`)
	if !strings.Contains(got, "interactive execution unavailable") || strings.Contains(got, "Linux") {
		t.Fatalf("interactive flag was masked: %q", got)
	}
}
