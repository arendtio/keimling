package main

import (
	"testing"
	"time"
)

func TestParseAlarmWhenRelative(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.Local)
	got, every, err := parseAlarmWhen("in 2m", now)
	if err != nil || every != 0 {
		t.Fatal(err, every)
	}
	if !got.Equal(now.Add(2 * time.Minute)) {
		t.Fatalf("got %v", got)
	}
	got, every, err = parseAlarmWhen("5m", now)
	if err != nil || every != 0 || !got.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("bare duration: %v %v %v", got, every, err)
	}
}

func TestParseAlarmWhenEvery(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.Local)
	got, every, err := parseAlarmWhen("every 30s", now)
	if err != nil {
		t.Fatal(err)
	}
	if every != 30*time.Second || !got.Equal(now.Add(30*time.Second)) {
		t.Fatalf("got %v every %v", got, every)
	}
}

func TestParseAlarmWhenAbsolute(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.Local)
	got, every, err := parseAlarmWhen("at 15:04", now)
	if err != nil || every != 0 {
		t.Fatal(err, every)
	}
	if got.Hour() != 15 || got.Minute() != 4 || got.Day() != 28 {
		t.Fatalf("got %v", got)
	}
	nowLate := time.Date(2026, 10, 28, 16, 0, 0, 0, time.Local)
	got, _, err = parseAlarmWhen("at 15:04", nowLate)
	if err != nil {
		t.Fatal(err)
	}
	if got.Day() != 29 {
		t.Fatalf("expected tomorrow, got %v", got)
	}
}

func TestAlarmPersistDueAndRepeat(t *testing.T) {
	withTempAgentHomes(t)
	wsRoot := t.TempDir()
	ws, err := ensureWorkspaceForRoot(wsRoot)
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("A", "x")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "A1")
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Minute)
	if _, err := addAgentAlarm(inst.ID, past, "too late", 0); err == nil {
		t.Fatal("expected past reject")
	}
	fire := time.Now().Add(40 * time.Millisecond)
	al, err := addAgentAlarm(inst.ID, fire, "wake up", 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	due, err := dueAgentAlarms(inst.ID, time.Now())
	if err != nil || len(due) != 1 || due[0].ID != al.ID {
		t.Fatalf("due=%v err=%v", due, err)
	}
	left, _ := readAgentAlarms(inst.ID)
	if len(left) != 1 || left[0].Every != 50*time.Millisecond {
		t.Fatalf("expected rescheduled repeat, left=%+v", left)
	}
}
