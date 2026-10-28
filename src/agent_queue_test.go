package main

import "testing"

func TestAgentQueueRoundTrip(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Q", "q")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Q1")
	if err != nil {
		t.Fatal(err)
	}
	if err := enqueueAgentMessage(inst.ID, "hello", "peer-1", "peer"); err != nil {
		t.Fatal(err)
	}
	if err := enqueueAgentMessage(inst.ID, "alarm text", "alarm", "alarm"); err != nil {
		t.Fatal(err)
	}
	items, err := dequeueAgentMessages(inst.ID)
	if err != nil || len(items) != 2 {
		t.Fatalf("%v %v", items, err)
	}
	if formatQueueWake(items[0]) != "[Message from agent peer-1] hello" {
		t.Fatalf("format %q", formatQueueWake(items[0]))
	}
	again, err := dequeueAgentMessages(inst.ID)
	if err != nil || len(again) != 0 {
		t.Fatalf("expected empty after drain: %v %v", again, err)
	}
}
