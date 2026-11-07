package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAgentAlarmDueDoesNotConsume(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Waker", "Wake.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Waker-1")
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Minute)
	if err := writeAgentAlarms(inst.ID, []AgentAlarm{{ID: "a", FireAt: past, Message: "wake"}}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(agentInstanceWatchesPath(inst.ID))
	if err != nil {
		t.Fatal(err)
	}
	if !agentAlarmDue(inst.ID, time.Now()) {
		t.Fatal("a past alarm is due")
	}
	if agentAlarmDue(inst.ID, past.Add(-time.Second)) {
		t.Fatal("a future alarm is not due")
	}
	after, err := os.ReadFile(agentInstanceWatchesPath(inst.ID))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("probe rewrote alarms:\n%s", after)
	}
}

func TestHeldStopDoesNotConsumeDueAlarm(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Waker", "Wake.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Waker-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeAgentAlarms(inst.ID, []AgentAlarm{{ID: "a", FireAt: time.Now().Add(-time.Minute), Message: "wake"}}); err != nil {
		t.Fatal(err)
	}
	if err := setAgentWakesHeld(inst.ID, true); err != nil {
		t.Fatal(err)
	}
	j, err := OpenJournal(agentInstanceSessionPath(inst.ID), false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	srv := &appServer{agent: &Agent{Cfg: Config{AgentInstanceID: inst.ID}, Journal: j}}
	srv.pumpAgentWakes()
	if !agentAlarmDue(inst.ID, time.Now()) {
		t.Fatal("a held stop consumed the due alarm")
	}
	queued, err := dequeueAgentMessages(inst.ID)
	if err != nil || len(queued) != 0 {
		t.Fatalf("held stop queued a wake: %+v %v", queued, err)
	}
}

func TestResetInstanceKeepsIdentity(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Waker", "Wake.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Waker-1")
	if err != nil {
		t.Fatal(err)
	}
	other, err := createAgentInstance(recipe, ws.ID, "Other")
	if err != nil {
		t.Fatal(err)
	}
	if err := addAgentInstancePeer(inst.ID, other.ID); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentInstanceSessionPath(inst.ID), []byte("{\"type\":\"item\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := enqueueAgentMessage(inst.ID, "pending", "", "message"); err != nil {
		t.Fatal(err)
	}
	if err := resetAgentInstanceFiles(inst.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(agentInstanceSessionPath(inst.ID)); !os.IsNotExist(err) {
		t.Fatal("session file should have been archived")
	}
	matches, _ := filepath.Glob(agentInstanceSessionPath(inst.ID) + ".*.bak")
	if len(matches) != 1 {
		t.Fatalf("backup: %v", matches)
	}
	queued, err := dequeueAgentMessages(inst.ID)
	if err != nil || len(queued) != 0 {
		t.Fatalf("queue: %+v %v", queued, err)
	}
	meta, err := readAgentInstanceMeta(inst.ID)
	if err != nil || meta.Name != "Waker-1" || len(meta.Peers) != 1 || meta.WakesHeld {
		t.Fatalf("identity: %+v %v", meta, err)
	}
}
