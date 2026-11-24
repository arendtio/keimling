package main

import (
	"bufio"
	"context"
	"strings"
	"testing"
)

func runPlanTool(t *testing.T, a *Agent, name, args string) string {
	t.Helper()
	out, _ := a.invokeTool(context.Background(), bufio.NewReader(strings.NewReader("")), "", Item{Type: "function_call", Name: name, Arguments: args})
	return out
}

func TestFinishTaskRefusesWhileAPlanStepIsOpen(t *testing.T) {
	a := &Agent{}
	if got := runPlanTool(t, a, "update_plan", `{"steps":[{"step":"write the spec","status":"done"},{"step":"write the code","status":"pending"}]}`); got != "plan updated" {
		t.Fatalf("update_plan: %q", got)
	}
	got := runPlanTool(t, a, "finish_task", `{"status":"done","summary":"Spec and code are written, checked with the tests, nothing open."}`)
	if !strings.Contains(got, "1 plan steps are not done") || !strings.Contains(got, "write the code") {
		t.Fatalf("open step must block the end: %q", got)
	}
	if a.TaskDone {
		t.Fatal("the task must not be marked done")
	}
}

func TestFinishTaskNeedsAReport(t *testing.T) {
	a := &Agent{}
	if got := runPlanTool(t, a, "update_plan", `{"steps":[{"step":"write the spec","status":"done"}]}`); got != "plan updated" {
		t.Fatalf("update_plan: %q", got)
	}
	if got := runPlanTool(t, a, "finish_task", `{"status":"done","summary":"done"}`); !strings.Contains(got, "closing report") {
		t.Fatalf("a bare done must be refused: %q", got)
	}
	got := runPlanTool(t, a, "finish_task", `{"status":"done","summary":"Spec is at specs/020.md, tests green, nothing open."}`)
	if got != "task status recorded: done" {
		t.Fatalf("a report admits the end: %q", got)
	}
}

func TestFinishTaskBlockedNeedsNoPlan(t *testing.T) {
	a := &Agent{}
	if got := runPlanTool(t, a, "update_plan", `{"steps":[{"step":"write the spec","status":"pending"}]}`); got != "plan updated" {
		t.Fatalf("update_plan: %q", got)
	}
	if got := runPlanTool(t, a, "finish_task", `{"status":"blocked","summary":"Which tool set should the role get?"}`); got != "task status recorded: blocked" {
		t.Fatalf("blocked is allowed with open steps: %q", got)
	}
}
