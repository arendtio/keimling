package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Execute this under script(1) to supply a real controlling terminal.
func TestPTYRepeatedProcessExit(t *testing.T) {
	if !isTerminal(0) {
		t.Skip("requires terminal")
	}
	for i := 0; i < 3; i++ {
		got := interactiveExec(context.Background(), selectedShell(), "printf 'pty-ok\\n'", 3*time.Second)
		if !strings.Contains(got, "exit_code=0") {
			t.Fatalf("iteration %d: %s", i, got)
		}
	}
}

func TestPTYCancellation(t *testing.T) {
	if !isTerminal(0) {
		t.Skip("requires terminal")
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(150 * time.Millisecond); cancel() }()
	started := time.Now()
	got := interactiveExec(ctx, selectedShell(), "sleep 15", 4*time.Second)
	if time.Since(started) > 3*time.Second {
		t.Fatalf("cancel took too long: %s", got)
	}
	if !strings.Contains(got, "interrupted") {
		t.Fatalf("did not report interruption: %s", got)
	}
}

func TestPTYTerminalRestored(t *testing.T) {
	if !isTerminal(0) {
		t.Skip("requires terminal")
	}
	before, e := getTermios(0)
	if e != nil {
		t.Fatal(e)
	}
	got := interactiveExec(context.Background(), selectedShell(), "true", 3*time.Second)
	if !strings.Contains(got, "exit_code=0") {
		t.Fatal(got)
	}
	after, e := getTermios(0)
	if e != nil {
		t.Fatal(e)
	}
	if before.Lflag != after.Lflag || before.Iflag != after.Iflag || before.Oflag != after.Oflag {
		t.Fatal("terminal flags not restored")
	}
	_ = exec.Command("true").Run()
}

func TestPTYInputForwarding(t *testing.T) {
	if !isTerminal(0) {
		t.Skip("requires controlling terminal with input")
	}
	got := interactiveExec(context.Background(), selectedShell(), "stty -echo; printf 'Password: '; IFS= read -r secret; stty echo; [ \"$secret\" = 'test-value' ]", 5*time.Second)
	if !strings.Contains(got, "exit_code=0") {
		t.Fatalf("PTY failed to forward input: %s", got)
	}
}
