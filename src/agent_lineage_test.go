package main

import (
	"strings"
	"testing"
)

// lineageFixture builds a parent instance in one workspace and returns it with
// its workspace. The definition grants the tools the parent needs to control
// what it starts.
func lineageFixture(t *testing.T, name string) (AgentInstanceMeta, AgentRecipe) {
	t.Helper()
	withTempAgentHomes(t)
	t.Setenv("KEIMLING_HOST", "")
	t.Setenv("KEIMLING_SECRET", "")
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe(name, "Start and control.")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setRecipeTools(recipe.ID, []string{"list_agents", "message_agent", "stop_agent", "delete_agent", "spawn_agent"}); err != nil {
		t.Fatal(err)
	}
	parent, err := createAgentInstance(recipe, ws.ID, name+"-1")
	if err != nil {
		t.Fatal(err)
	}
	return parent, recipe
}

func TestSpawnerSeesAndControlsItsChildAnywhere(t *testing.T) {
	parent, recipe := lineageFixture(t, "Lead")
	child, err := createAgentInstanceForPath(recipe, parent.CurrentWorkspace, t.TempDir(), "Child", parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if child.Spawner != parent.ID {
		t.Fatalf("spawner = %q, want %q", child.Spawner, parent.ID)
	}
	if !sharesAWorkspace(parent, child) {
		t.Fatal("the caller workspace stays in the child's ACL")
	}
	a := &Agent{Cfg: Config{AgentInstanceID: parent.ID}}
	listed := a.listWorkspaceAgents()
	if !strings.Contains(listed, child.ID) || !strings.Contains(listed, "relation=child") {
		t.Fatalf("the spawner must see what it started:\n%s", listed)
	}
	if got := a.stopAgentTool(`{"instance_id":"` + child.ID + `"}`); !strings.Contains(got, "stopped") {
		t.Fatalf("stop: %q", got)
	}
	if got := a.deleteAgentTool(`{"instance_id":"` + child.ID + `"}`); !strings.Contains(got, "deleted") {
		t.Fatalf("delete: %q", got)
	}
	if _, err := readAgentInstanceMeta(child.ID); err == nil {
		t.Fatal("the child should be gone")
	}
}

// Control follows reach, not the spawn line: whoever may talk to an instance,
// or works beside it, may stop and delete it. The team relies on that (the
// optimizer removes developers it did not start).
func TestControlFollowsReach(t *testing.T) {
	parent, recipe := lineageFixture(t, "Lead")
	child, err := createAgentInstanceForPath(recipe, parent.CurrentWorkspace, "", "Child", parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	kid := &Agent{Cfg: Config{AgentInstanceID: child.ID}}
	listed := kid.listWorkspaceAgents()
	if !strings.Contains(listed, parent.ID) || !strings.Contains(listed, "relation=parent") {
		t.Fatalf("the child must see its spawner:\n%s", listed)
	}
	if got := kid.stopAgentTool(`{"instance_id":"` + parent.ID + `"}`); !strings.Contains(got, "stopped") {
		t.Fatalf("an instance in my workspace is reachable: %q", got)
	}

	// An instance in another workspace that is neither a contact nor in my
	// workspace cannot even be resolved.
	otherWS, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	far, err := createAgentInstance(recipe, otherWS.ID, "Far")
	if err != nil {
		t.Fatal(err)
	}
	if got := kid.stopAgentTool(`{"instance_id":"` + far.ID + `"}`); !strings.Contains(got, "unknown agent") {
		t.Fatalf("out of reach: %q", got)
	}
}

func TestLineageIsContactInBothDirections(t *testing.T) {
	parent, recipe := lineageFixture(t, "Lead")
	child, err := createAgentInstanceForPath(recipe, parent.CurrentWorkspace, "", "Child", parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	// No peer list on either side: lineage alone has to carry the contact.
	if !agentsMayContact(parent, child) || !agentsMayContact(child, parent) {
		t.Fatal("spawner and child must reach each other")
	}
	other, err := createAgentInstance(recipe, parent.CurrentWorkspace, "Stranger")
	if err != nil {
		t.Fatal(err)
	}
	if agentsMayContact(other, child) || agentsMayContact(child, other) {
		t.Fatal("two instances without a line must not reach each other")
	}
}

func TestDeletingAnInstanceDropsItsLine(t *testing.T) {
	parent, recipe := lineageFixture(t, "Lead")
	child, err := createAgentInstanceForPath(recipe, parent.CurrentWorkspace, "", "Child", parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := linkAgentInstances(parent.ID, child.ID); err != nil {
		t.Fatal(err)
	}
	if err := deleteAgentInstance(child.ID); err != nil {
		t.Fatal(err)
	}
	after, err := readAgentInstanceMeta(parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range after.Peers {
		if p == child.ID {
			t.Fatalf("peers still list the deleted instance: %v", after.Peers)
		}
	}
}
