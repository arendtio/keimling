package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// noHostEnv makes a spawn take the local path instead of asking a host.
func noHostEnv(t *testing.T) {
	t.Helper()
	t.Setenv("KEIMLING_HOST", "")
	t.Setenv("KEIMLING_SECRET", "")
}

// failSpawnPeerLink makes every contact write of a spawn fail. It restores the
// writer before the next test runs.
func failSpawnPeerLink(t *testing.T) {
	t.Helper()
	old := writeSpawnPeerLink
	writeSpawnPeerLink = func(a, b string) error { return fmt.Errorf("peers are read-only") }
	t.Cleanup(func() { writeSpawnPeerLink = old })
}

func instanceNamed(t *testing.T, name string) AgentInstanceMeta {
	t.Helper()
	list, err := listAgentInstances()
	if err != nil {
		t.Fatal(err)
	}
	for _, inst := range list {
		if inst.Name == name {
			return inst
		}
	}
	t.Fatalf("no instance named %q in %+v", name, list)
	return AgentInstanceMeta{}
}

func instanceIDs(list []AgentInstanceMeta) []string {
	out := make([]string, 0, len(list))
	for _, inst := range list {
		out = append(out, inst.Name)
	}
	return out
}

func TestSpawnAgentLinksTheCallerAndTheChild(t *testing.T) {
	withTempAgentHomes(t)
	noHostEnv(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Pair", "Talk.")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createAgentInstance(recipe, ws.ID, "Bystander"); err != nil {
		t.Fatal(err)
	}
	caller, err := createAgentInstance(recipe, ws.ID, "Caller")
	if err != nil {
		t.Fatal(err)
	}
	callerAgent := &Agent{Cfg: Config{AgentInstanceID: caller.ID, WorkspaceID: ws.ID}}
	out := callerAgent.runDefineAgentTool("spawn_agent", `{"definition":"Pair","name":"Child"}`)
	if strings.Contains(out, "error") {
		t.Fatal(out)
	}
	child := instanceNamed(t, "Child")
	bystander := instanceNamed(t, "Bystander")
	if child.ID == caller.ID {
		t.Fatal("spawn returned the caller")
	}
	caller, err = readAgentInstanceMeta(caller.ID)
	if err != nil {
		t.Fatal(err)
	}
	child, err = readAgentInstanceMeta(child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !peersContain(caller.Peers, child.ID) || !peersContain(child.Peers, caller.ID) {
		t.Fatalf("both sides should list each other: caller %v child %v", caller.Peers, child.Peers)
	}
	if agentsMayContact(caller, bystander) || agentsMayContact(child, bystander) {
		t.Fatal("a spawn linked an instance that took no part in it")
	}
	childAgent := &Agent{Cfg: Config{AgentInstanceID: child.ID, WorkspaceID: ws.ID}}
	if listed := childAgent.listAgentsTool(""); !strings.Contains(listed, caller.ID) || !strings.Contains(listed, "peer_allowed=true") {
		t.Fatalf("child list: %s", listed)
	}
	if listed := callerAgent.listAgentsTool(""); !strings.Contains(listed, child.ID) || !strings.Contains(listed, "peer_allowed=true") {
		t.Fatalf("caller list: %s", listed)
	}
	if got := childAgent.messageAgent(`{"instance_id":"` + caller.ID + `","text":"done"}`); !strings.Contains(got, "Sent to") {
		t.Fatalf("child to caller: %s", got)
	}
	if items, err := dequeueAgentMessages(caller.ID); err != nil || len(items) != 1 || items[0].Kind != "peer" {
		t.Fatalf("caller queue: %+v %v", items, err)
	}
	if got := callerAgent.messageAgent(`{"instance_id":"` + child.ID + `","text":"thanks"}`); !strings.Contains(got, "Sent to") {
		t.Fatalf("caller to child: %s", got)
	}
	if items, err := dequeueAgentMessages(child.ID); err != nil || len(items) != 1 || items[0].Kind != "peer" {
		t.Fatalf("child queue: %+v %v", items, err)
	}
}

func TestSpawnTeamLinksEveryMember(t *testing.T) {
	withTempAgentHomes(t)
	noHostEnv(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{WorkspaceID: ws.ID}}
	if saved := a.runDefineAgentTool("save_agent", `{"id":"","name":"Reader","instructions":"Read."}`); strings.Contains(saved, "error") {
		t.Fatal(saved)
	}
	if saved := a.runDefineAgentTool("save_agent", `{"id":"","name":"Writer","instructions":"Write."}`); strings.Contains(saved, "error") {
		t.Fatal(saved)
	}
	if made := a.runDefineAgentTool("save_team", `{"id":"","name":"Crew","definitions":["Reader","Writer"]}`); strings.Contains(made, "error") {
		t.Fatal(made)
	}
	if spawned := a.runDefineAgentTool("spawn_team", `{"team":"Crew"}`); strings.Contains(spawned, "error") {
		t.Fatal(spawned)
	}
	list, err := listAgentInstances()
	if err != nil || len(list) != 2 {
		t.Fatalf("instances: %v %v", instanceIDs(list), err)
	}
	reader := instanceNamed(t, "Reader")
	writer := instanceNamed(t, "Writer")
	if !peersContain(reader.Peers, writer.ID) || !peersContain(writer.Peers, reader.ID) {
		t.Fatalf("members should list each other: reader %v writer %v", reader.Peers, writer.Peers)
	}
	if len(reader.Peers) != 1 || len(writer.Peers) != 1 {
		t.Fatalf("a team spawn links only its own members: reader %v writer %v", reader.Peers, writer.Peers)
	}
}

func TestSpawnTeamLinksTheSpawningAgentWithEveryMember(t *testing.T) {
	withTempAgentHomes(t)
	noHostEnv(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Crew", "Work.")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createAgentInstance(recipe, ws.ID, "Bystander"); err != nil {
		t.Fatal(err)
	}
	caller, err := createAgentInstance(recipe, ws.ID, "Caller")
	if err != nil {
		t.Fatal(err)
	}
	team, err := createAgentTeam("Crew", []string{recipe.ID, recipe.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(team.DefinitionIDs) != 1 {
		t.Fatalf("team should name one definition once: %+v", team.DefinitionIDs)
	}
	one, err := createAgentRecipe("One", "First.")
	if err != nil {
		t.Fatal(err)
	}
	two, err := createAgentRecipe("Two", "Second.")
	if err != nil {
		t.Fatal(err)
	}
	team, err = updateAgentTeam(team.ID, "", []string{one.ID, two.ID}, true)
	if err != nil {
		t.Fatal(err)
	}
	callerAgent := &Agent{Cfg: Config{AgentInstanceID: caller.ID, WorkspaceID: ws.ID}}
	if spawned := callerAgent.runDefineAgentTool("spawn_team", `{"team":"`+team.ID+`"}`); strings.Contains(spawned, "error") {
		t.Fatal(spawned)
	}
	caller, err = readAgentInstanceMeta(caller.ID)
	if err != nil {
		t.Fatal(err)
	}
	members := []AgentInstanceMeta{instanceNamed(t, "One"), instanceNamed(t, "Two")}
	if len(caller.Peers) != len(members) {
		t.Fatalf("caller should list every member: %v", caller.Peers)
	}
	for _, member := range members {
		if !peersContain(caller.Peers, member.ID) || !peersContain(member.Peers, caller.ID) {
			t.Fatalf("caller and %s should list each other: caller %v member %v", member.Name, caller.Peers, member.Peers)
		}
		if agentsMayContact(member, instanceNamed(t, "Bystander")) {
			t.Fatalf("%s was linked to an instance outside the spawn", member.Name)
		}
	}
	if !peersContain(members[0].Peers, members[1].ID) || !peersContain(members[1].Peers, members[0].ID) {
		t.Fatal("members of one team spawn should list each other")
	}
	if agentsMayContact(caller, instanceNamed(t, "Bystander")) {
		t.Fatal("the caller was linked to an instance outside the spawn")
	}
}

func TestSpawnAgentReportsAContactThatCannotBeWritten(t *testing.T) {
	withTempAgentHomes(t)
	noHostEnv(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Pair", "Talk.")
	if err != nil {
		t.Fatal(err)
	}
	caller, err := createAgentInstance(recipe, ws.ID, "Caller")
	if err != nil {
		t.Fatal(err)
	}
	failSpawnPeerLink(t)
	callerAgent := &Agent{Cfg: Config{AgentInstanceID: caller.ID, WorkspaceID: ws.ID}}
	out := callerAgent.runDefineAgentTool("spawn_agent", `{"definition":"Pair","name":"Child"}`)
	if !strings.Contains(out, "spawn_agent error") || !strings.Contains(out, "read-only") {
		t.Fatalf("a failed contact should fail the spawn: %s", out)
	}
	list, err := listAgentInstances()
	if err != nil || len(list) != 1 || list[0].ID != caller.ID {
		t.Fatalf("a failed spawn left instances behind: %v %v", instanceIDs(list), err)
	}
	meta, err := readAgentInstanceMeta(caller.ID)
	if err != nil || len(meta.Peers) != 0 {
		t.Fatalf("a failed spawn left a contact behind: %v %v", meta.Peers, err)
	}
}

func TestSpawnTeamReportsAContactThatCannotBeWritten(t *testing.T) {
	withTempAgentHomes(t)
	noHostEnv(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{WorkspaceID: ws.ID}}
	for _, name := range []string{"Reader", "Writer"} {
		if saved := a.runDefineAgentTool("save_agent", `{"id":"","name":"`+name+`","instructions":"Do."}`); strings.Contains(saved, "error") {
			t.Fatal(saved)
		}
	}
	if made := a.runDefineAgentTool("save_team", `{"id":"","name":"Crew","definitions":["Reader","Writer"]}`); strings.Contains(made, "error") {
		t.Fatal(made)
	}
	failSpawnPeerLink(t)
	spawned := a.runDefineAgentTool("spawn_team", `{"team":"Crew"}`)
	if !strings.Contains(spawned, "spawn_team error") || !strings.Contains(spawned, "read-only") {
		t.Fatalf("a failed contact should fail the team spawn: %s", spawned)
	}
	list, err := listAgentInstances()
	if err != nil || len(list) != 0 {
		t.Fatalf("a failed team spawn left instances behind: %v %v", instanceIDs(list), err)
	}
}

func TestHostSpawnAgentReportsAContactThatCannotBeWritten(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Pair", "Talk.")
	if err != nil {
		t.Fatal(err)
	}
	caller, err := createAgentInstance(recipe, ws.ID, "Caller")
	if err != nil {
		t.Fatal(err)
	}
	h := &Host{}
	failSpawnPeerLink(t)
	body := `{"recipe_id":"` + recipe.ID + `","name":"Child","workspace_id":"` + ws.ID + `","spawner_id":"` + caller.ID + `"}`
	rec := httptest.NewRecorder()
	h.handleAgentInstances(rec, httptest.NewRequest(http.MethodPost, "/api/agents/instances", strings.NewReader(body)))
	if rec.Code/100 == 2 || !strings.Contains(rec.Body.String(), "read-only") {
		t.Fatalf("host spawn should fail on a lost contact: %d %s", rec.Code, rec.Body.String())
	}
	list, err := listAgentInstances()
	if err != nil || len(list) != 1 || list[0].ID != caller.ID {
		t.Fatalf("host spawn left an unlinked instance: %v %v", instanceIDs(list), err)
	}
}

func TestHostTeamSpawnReportsAContactThatCannotBeWritten(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Pair", "Talk.")
	if err != nil {
		t.Fatal(err)
	}
	other, err := createAgentRecipe("Other", "Work.")
	if err != nil {
		t.Fatal(err)
	}
	team, err := createAgentTeam("Crew", []string{recipe.ID, other.ID})
	if err != nil {
		t.Fatal(err)
	}
	caller, err := createAgentInstance(recipe, ws.ID, "Caller")
	if err != nil {
		t.Fatal(err)
	}
	h := &Host{}
	failSpawnPeerLink(t)
	body := `{"id":"` + team.ID + `","workspace_id":"` + ws.ID + `","spawner_id":"` + caller.ID + `"}`
	rec := httptest.NewRecorder()
	h.handleAgentTeamSpawn(rec, httptest.NewRequest(http.MethodPost, "/api/agents/teams/spawn", strings.NewReader(body)))
	if rec.Code/100 == 2 || !strings.Contains(rec.Body.String(), "read-only") {
		t.Fatalf("host team spawn should fail on a lost contact: %d %s", rec.Code, rec.Body.String())
	}
	list, err := listAgentInstances()
	if err != nil || len(list) != 1 || list[0].ID != caller.ID {
		t.Fatalf("host team spawn left unlinked instances: %v %v", instanceIDs(list), err)
	}
}

func TestHostSpawnWithoutASpawnerLeavesContactsEmpty(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Pair", "Talk.")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createAgentInstance(recipe, ws.ID, "Caller"); err != nil {
		t.Fatal(err)
	}
	h := &Host{}
	body := `{"recipe_id":"` + recipe.ID + `","name":"Solo","workspace_id":"` + ws.ID + `"}`
	rec := httptest.NewRecorder()
	h.handleAgentInstances(rec, httptest.NewRequest(http.MethodPost, "/api/agents/instances", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("spawn: %d %s", rec.Code, rec.Body.String())
	}
	solo := instanceNamed(t, "Solo")
	if len(solo.Peers) != 0 {
		t.Fatalf("a spawn without a spawner wrote contacts: %v", solo.Peers)
	}
}
