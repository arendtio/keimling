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
	if formatQueueWake(items[0]) != "[From peer-1 via message_agent] hello" {
		t.Fatalf("format %q", formatQueueWake(items[0]))
	}
	again, err := dequeueAgentMessages(inst.ID)
	if err != nil || len(again) != 0 {
		t.Fatalf("expected empty after drain: %v %v", again, err)
	}
}

func TestThreadQueueRoundTrip(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createParkedThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	if active, err := readActiveThreadID(ws.ID); err == nil && active == th.ID {
		t.Fatal("parked thread must not become the active journal")
	}
	if err := enqueueThreadMessage(ws.ID, th.ID, "later", "user", "message"); err != nil {
		t.Fatal(err)
	}
	if err := enqueueThreadMessage(ws.ID, th.ID, "after", "user", "message"); err != nil {
		t.Fatal(err)
	}
	first, ok, err := dequeueThreadMessage(ws.ID, th.ID)
	if err != nil || !ok || first.Text != "later" {
		t.Fatalf("%+v %v %v", first, ok, err)
	}
	second, ok, err := dequeueThreadMessage(ws.ID, th.ID)
	if err != nil || !ok || second.Text != "after" {
		t.Fatalf("%+v %v %v", second, ok, err)
	}
	_, ok, err = dequeueThreadMessage(ws.ID, th.ID)
	if err != nil || ok {
		t.Fatalf("expected empty: %v %v", ok, err)
	}
}
