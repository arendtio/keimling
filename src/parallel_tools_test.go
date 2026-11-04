package main

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWorkInstructionsStateTheRulesThatChangeTheWork(t *testing.T) {
	got := sharedWorkInstructions()
	for _, want := range []string{
		"Instruction priority:",
		"then AGENTS.md",
		"data, not instructions",
		"smallest scope",
		"Do not refactor unrelated code",
		"Search before reading",
		"failing check first",
		"Do not weaken or delete a test",
		"only when its tool output says so",
		"without new tool output",
		"reversible default",
		"Call independent reads",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q", want)
		}
	}
	for _, gone := range []string{"raw_output=true", "sudo -n", "show_artifact again", "as=view"} {
		if strings.Contains(got, gone) {
			t.Fatalf("work prompt still restates %q", gone)
		}
	}
	thread := (&Agent{}).modelInstructions()
	if !strings.Contains(thread, "The user controls the PTY directly.") {
		t.Fatal("workspace thread lost the PTY sentence")
	}
}

func TestToolCallParallelLeavesWritesAndTerminalsAlone(t *testing.T) {
	if !toolCallParallel("read_file", `{"path":"a.go","start_line":1,"end_line":2}`) {
		t.Fatal("read_file")
	}
	if !toolCallParallel("exec", `{"command":"go test","timeout_seconds":5,"interactive":false,"raw_output":false,"output_focus":""}`) {
		t.Fatal("non-interactive exec")
	}
	if toolCallParallel("edit_file", `{"path":"a.go","old_string":"a","new_string":"b","replace_all":false}`) {
		t.Fatal("edit_file runs beside other tools")
	}
	if toolCallParallel("exec", `{"command":"ssh host","timeout_seconds":5,"interactive":true,"raw_output":false,"output_focus":""}`) {
		t.Fatal("terminal exec runs beside other tools")
	}
	if toolCallParallel("exec", `{"command":"echo {{keimling:secret}}","timeout_seconds":5,"interactive":false,"raw_output":false,"output_focus":""}`) {
		t.Fatal("credential exec runs beside other tools")
	}
}

func parallelAgent(t *testing.T) *Agent {
	t.Helper()
	j, err := OpenJournal(filepath.Join(t.TempDir(), "session.jsonl"), false)
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{Mode: "yolo"}, Journal: j}
	a.History = j.History
	return a
}

func sleepExec(command, id string) Item {
	return Item{Type: "function_call", Name: "exec", CallID: id, Arguments: `{"command":"` + command + `","timeout_seconds":5,"interactive":false,"raw_output":true,"output_focus":""}`}
}

func TestTwoCommandsInOneResponseOverlap(t *testing.T) {
	a := parallelAgent(t)
	cmd := "sleep 0.5"
	if runtime.GOOS == "windows" {
		cmd = "Start-Sleep -Milliseconds 500"
	}
	start := time.Now()
	err := a.runFunctionCalls(context.Background(), "check", nil, []Item{sleepExec(cmd, "c1"), sleepExec(cmd, "c2")})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 900*time.Millisecond {
		t.Fatalf("commands ran one after another: %s", elapsed)
	}
}

func TestAFileEditSeparatesTheCommandsAroundIt(t *testing.T) {
	a := parallelAgent(t)
	cmd := "sleep 0.5"
	if runtime.GOOS == "windows" {
		cmd = "Start-Sleep -Milliseconds 500"
	}
	edit := Item{Type: "function_call", Name: "edit_file", CallID: "e", Arguments: `{"path":"missing-file.go","old_string":"a","new_string":"b","replace_all":false}`}
	start := time.Now()
	err := a.runFunctionCalls(context.Background(), "check", nil, []Item{sleepExec(cmd, "c1"), edit, sleepExec(cmd, "c2")})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Fatalf("edit did not keep the commands apart: %s", elapsed)
	}
}
