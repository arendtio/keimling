package main

import (
	"context"
	"strings"
	"testing"
)

func TestAccidentalInteractiveDiagnostic(t *testing.T) {
	for _, c := range []string{"lscpu", "uname -a", "date", "whoami"} {
		if !safeDiagnostic(c) {
			t.Fatalf("diagnostic unexpectedly interactive: %q", c)
		}
		if toolNeedsTerminal("exec", `{"command":"`+c+`","interactive":true}`) {
			t.Fatalf("terminal should not be requested: %q", c)
		}
	}
	for _, c := range []string{"ssh host", "scp a b", "lscpu; ssh host", "lscpu $(touch /tmp/file)"} {
		if safeDiagnostic(c) {
			t.Fatalf("unsafe command marked diagnostic: %q", c)
		}
	}
	got := executeTool(context.Background(), "exec", `{"command":"uname -s","interactive":true}`)
	if !strings.Contains(got, "exit_code=0") || !strings.Contains(got, "Linux") {
		t.Fatalf("not executed non-interactively: %q", got)
	}
}
