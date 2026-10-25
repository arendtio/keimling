package main

import (
	"strings"
	"testing"
)

func TestHighlightDisplayOnly(t *testing.T) {
	cmd := "python3 <<'PY'\n# hello\nx = 42\nprint(\"hello\")\nPY"
	got := highlightCommand(cmd)
	if !strings.Contains(got, "\x1b[90m# hello") || !strings.Contains(got, "\x1b[33m42") || !strings.Contains(got, "\x1b[32m\"hello\"") {
		t.Fatalf("missing color: %q", got)
	}
}
func TestJSONHighlight(t *testing.T) {
	s := syntaxLine(`{"status":true,"count":42}`, "json")
	if !strings.Contains(s, "\x1b[36m\"status\"") || !strings.Contains(s, "\x1b[35mtrue") {
		t.Fatal(s)
	}
}
