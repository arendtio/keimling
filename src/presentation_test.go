package main

import (
	"strings"
	"testing"
)

func TestRenderExecEscapes(t *testing.T) {
	got := renderToolCall("exec", `{"command":"printf 'a\\nb'","interactive":false}`)
	if strings.Contains(got, `"command"`) {
		t.Fatal(got)
	}
	if !strings.Contains(got, `printf 'a\nb'`) {
		t.Fatal(got)
	}
}
func TestRenderPlanAndFinish(t *testing.T) {
	p := renderPlan([]PlanStep{{Step: "Inspect", Status: "done"}, {Step: "Test", Status: "in_progress"}})
	if !strings.Contains(p, "1/2 complete") || !strings.Contains(p, "✓") || !strings.Contains(p, "◉") {
		t.Fatal(p)
	}
	f := renderFinish("done", "**Verified** working")
	if !strings.Contains(f, "Task complete") || strings.Contains(f, "**") {
		t.Fatal(f)
	}
}
