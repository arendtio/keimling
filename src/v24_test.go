package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func captureFilterStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	w.Close()
	var b bytes.Buffer
	io.Copy(&b, r)
	r.Close()
	return b.String()
}
func TestRawOutputBypassIsVisible(t *testing.T) {
	a := &Agent{}
	original := "exit_code=0\n" + strings.Repeat("CPU details\n", 30)
	msg := captureFilterStdout(t, func() {
		got := a.filterExecResult(context.Background(), "cpu", `{"command":"lscpu","raw_output":true}`, original)
		if got != original {
			t.Fatal("must preserve raw output")
		}
	})
	if !strings.Contains(msg, "raw_output=true") {
		t.Fatalf("missing reason: %q", msg)
	}
}
func TestInteractiveBypassIsVisible(t *testing.T) {
	a := &Agent{}
	original := "exit_code=0\n" + strings.Repeat("line\n", 30)
	msg := captureFilterStdout(t, func() {
		got := a.filterExecResult(context.Background(), "ssh", `{"command":"ssh host","interactive":true}`, original)
		if got != original {
			t.Fatal("must preserve interactive output")
		}
	})
	if !strings.Contains(msg, "interactive=true") {
		t.Fatalf("missing reason: %q", msg)
	}
}
