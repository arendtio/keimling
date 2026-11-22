package main

import (
	"os"
	"strings"
	"testing"
)

func TestEscalateRecordsAnEntryForTheHuman(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Boss", "Raise problems.")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setRecipeTools(recipe.ID, []string{"escalate"}); err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Boss-1")
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{AgentInstanceID: inst.ID}}
	got := a.escalateTool(`{"text":"The merger is blocked on a stale purpose."}`)
	if strings.Contains(got, "error") {
		t.Fatal(got)
	}
	list := listEscalations(0)
	if len(list) != 1 {
		t.Fatalf("escalations = %+v", list)
	}
	if list[0].AgentID != inst.ID || list[0].AgentName != "Boss-1" {
		t.Fatalf("entry = %+v", list[0])
	}
	if !strings.Contains(list[0].Text, "stale purpose") || list[0].CreatedAt.IsZero() {
		t.Fatalf("entry = %+v", list[0])
	}
}

func TestEscalateNeedsAnAgentInstance(t *testing.T) {
	withTempAgentHomes(t)
	a := &Agent{}
	if got := a.escalateTool(`{"text":"hello"}`); !strings.Contains(got, "only available on agent instances") {
		t.Fatalf("got %q", got)
	}
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Boss", "Raise problems.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Boss-1")
	if err != nil {
		t.Fatal(err)
	}
	onInstance := &Agent{Cfg: Config{AgentInstanceID: inst.ID}}
	if got := onInstance.escalateTool(`{"text":"   "}`); !strings.Contains(got, "text required") {
		t.Fatalf("empty text: %q", got)
	}
	if len(listEscalations(0)) != 0 {
		t.Fatal("a rejected call must not write an entry")
	}
}

func TestEscalationsAreListedNewestFirst(t *testing.T) {
	withTempAgentHomes(t)
	first, err := addEscalation("agent-1", "One", "first")
	if err != nil {
		t.Fatal(err)
	}
	second, err := addEscalation("agent-2", "Two", "second")
	if err != nil {
		t.Fatal(err)
	}
	list := listEscalations(0)
	if len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
		t.Fatalf("order = %+v", list)
	}
	if one := listEscalations(1); len(one) != 1 || one[0].ID != second.ID {
		t.Fatalf("limit = %+v", one)
	}
}

func TestEscalationsSkipABrokenLine(t *testing.T) {
	withTempAgentHomes(t)
	if _, err := addEscalation("agent-1", "One", "kept"); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(escalationPath(), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{not json\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	list := listEscalations(0)
	if len(list) != 1 || list[0].Text != "kept" {
		t.Fatalf("list = %+v", list)
	}
}
