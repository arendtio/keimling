package main

import (
	"strings"
	"testing"
)

func TestRecordingStopKeys(t *testing.T) {
	if !submitRecordingKey('\r') || !submitRecordingKey('\n') || submitRecordingKey(' ') {
		t.Fatal("recording stop key behavior")
	}
}
func TestHereDocDisplay(t *testing.T) {
	s := highlightCommand("python3 - <<'PY'\ndef f():\n    return 1\nPY")
	if !strings.Contains(s, "\033[35mdef\033[0m") || !strings.Contains(s, "\033[35mreturn\033[0m") {
		t.Fatal(s)
	}
}
