package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Steering that arrives while a call runs must not throw the call away: its
// result lands, and the next model round sees the new instruction.
func TestSteeringDoesNotDiscardRunningCalls(t *testing.T) {
	a := parallelAgent(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.Control = newTaskControl(cancel)

	done := make(chan error, 1)
	go func() { done <- a.runFunctionCalls(ctx, "check", nil, []Item{sleepExec("sleep 0.3", "c1")}) }()

	time.Sleep(50 * time.Millisecond)
	a.Control.events <- controlEvent{kind: "steer", text: "please also look at the settings"}
	if err := a.handleControls(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	cancel()

	var history strings.Builder
	for _, raw := range a.History {
		history.Write(raw)
	}
	if strings.Contains(history.String(), "SKIPPED") {
		t.Fatal("steering discarded a running call")
	}
	if !strings.Contains(history.String(), "please also look at the settings") {
		t.Fatal("steering text is missing from the history")
	}
	if out := toolOutputFor(a.History, "c1"); strings.TrimSpace(out) == "" || strings.HasPrefix(out, "SKIPPED") {
		t.Fatalf("the running call produced no result: %q", out)
	}
}
