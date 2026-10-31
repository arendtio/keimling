package main

import (
	"os"
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
	before, err := os.ReadFile(agentInstanceAlarmsPath(inst.ID))
	if err != nil {
		t.Fatal(err)
	}
	if !agentAlarmDue(inst.ID, time.Now()) {
		t.Fatal("a past alarm is due")
	}
	if agentAlarmDue(inst.ID, past.Add(-time.Second)) {
		t.Fatal("a future alarm is not due")
	}
	after, err := os.ReadFile(agentInstanceAlarmsPath(inst.ID))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("probe rewrote alarms:\n%s", after)
	}
}
