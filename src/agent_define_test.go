package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefineAgentToolsAreGrantedLikeOtherTools(t *testing.T) {
	names := map[string]bool{}
	for _, tool := range definitions(false) {
		name, _ := tool["name"].(string)
		names[name] = true
	}
	for _, name := range []string{"agent_instructions", "save_agent", "spawn_agent", "save_team", "spawn_team"} {
		if !names[name] {
			t.Fatalf("missing %s", name)
		}
		if !definitionAllowsName(nil, name) {
			t.Fatalf("all-tools grant drops %s", name)
		}
	}
	fixed := []string{"read_file"}
	kept := filterGrantedTools(definitions(false), fixed)
	for _, tool := range kept {
		if tool["name"] == "save_agent" {
			t.Fatal("fixed grant included save_agent")
		}
	}
	with := []string{"save_agent", "spawn_agent"}
	found := map[string]bool{}
	for _, tool := range filterGrantedTools(definitions(false), with) {
		name, _ := tool["name"].(string)
		found[name] = true
	}
	if !found["save_agent"] || !found["spawn_agent"] || found["save_team"] {
		t.Fatalf("grant filter: %v", found)
	}
}

func TestDefineAgentToolsFollowApproval(t *testing.T) {
	if g := gateToolCall(false, "ask", "agent_instructions", `{}`, false); !g.Allow {
		t.Fatalf("read: %+v", g)
	}
	if g := gateToolCall(false, "auto", "save_agent", `{}`, false); !g.Review {
		t.Fatalf("workspace auto: %+v", g)
	}
	if g := gateToolCall(true, "auto", "spawn_agent", `{}`, false); !g.Review {
		t.Fatalf("agent auto: %+v", g)
	}
	if g := gateToolCall(false, "ask", "save_team", `{}`, false); !g.Ask {
		t.Fatalf("ask: %+v", g)
	}
	if g := gateToolCall(false, "yolo", "spawn_team", `{}`, false); !g.Allow {
		t.Fatalf("yolo: %+v", g)
	}
	if g := gateToolCall(false, "ask", "stop_agent", `{}`, false); !g.Allow {
		t.Fatalf("stop ask: %+v", g)
	}
	if g := gateToolCall(false, "auto", "stop_agent", `{}`, false); !g.Allow {
		t.Fatalf("stop auto: %+v", g)
	}
	if g := gateToolCall(false, "ask", "delete_agent", `{}`, false); !g.Ask || g.Allow {
		t.Fatalf("delete ask: %+v", g)
	}
	if g := gateToolCall(false, "auto", "delete_agent", `{}`, false); !g.Ask || g.Allow {
		t.Fatalf("delete auto: %+v", g)
	}
	if g := gateToolCall(false, "yolo", "delete_agent", `{}`, false); !g.Allow {
		t.Fatalf("delete yolo: %+v", g)
	}
}

func TestSaveAgentStartsFromTheKnownInstructions(t *testing.T) {
	withTempAgentHomes(t)
	a := &Agent{}
	got := a.runDefineAgentTool("agent_instructions", `{"unused":""}`)
	if got != defaultAgentInstructions {
		t.Fatalf("template: %q", got)
	}
	created := a.runDefineAgentTool("save_agent", `{"id":"","name":"Reviewer","instructions":""}`)
	if strings.Contains(created, "error") {
		t.Fatal(created)
	}
	list, err := listAgentRecipes()
	if err != nil || len(list) != 1 {
		t.Fatalf("recipes: %+v %v", list, err)
	}
	if list[0].Instructions != defaultAgentInstructions || list[0].Mode != "auto" {
		t.Fatalf("created: %+v", list[0])
	}
	if !strings.Contains(created, list[0].ID) {
		t.Fatal(created)
	}
	list[0].Mode = "ask"
	if err := writeAgentRecipe(list[0]); err != nil {
		t.Fatal(err)
	}
	updated := a.runDefineAgentTool("save_agent", `{"id":"`+list[0].ID+`","name":"","instructions":"Look at the diff."}`)
	if strings.Contains(updated, "error") {
		t.Fatal(updated)
	}
	again, err := readAgentRecipe(list[0].ID)
	if err != nil || again.Instructions != "Look at the diff." || again.Mode != "ask" || again.Name != "Reviewer" {
		t.Fatalf("updated: %+v %v", again, err)
	}
	kept := a.runDefineAgentTool("save_agent", `{"id":"`+list[0].ID+`","name":"Checker","instructions":""}`)
	if strings.Contains(kept, "error") {
		t.Fatal(kept)
	}
	final, err := readAgentRecipe(list[0].ID)
	if err != nil || final.Instructions != "Look at the diff." || final.Name != "Checker" {
		t.Fatalf("kept: %+v %v", final, err)
	}
}

func TestSpawnAgentQueuesTheFirstTask(t *testing.T) {
	withTempAgentHomes(t)
	t.Setenv("KEIMLING_HOST", "")
	t.Setenv("KEIMLING_SECRET", "")
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	thread := &Agent{Cfg: Config{WorkspaceID: ws.ID}}
	saved := thread.runDefineAgentTool("save_agent", `{"id":"","name":"Watcher","instructions":"Watch the server."}`)
	if strings.Contains(saved, "error") {
		t.Fatal(saved)
	}
	started := thread.runDefineAgentTool("spawn_agent", `{"definition":"Watcher","name":"","task":"Ping 192.168.0.101 every 10s."}`)
	if strings.Contains(started, "error") || !strings.Contains(started, "task queued") {
		t.Fatal(started)
	}
	list, err := listAgentInstances()
	if err != nil || len(list) != 1 {
		t.Fatal(err, len(list))
	}
	items, err := dequeueAgentMessages(list[0].ID)
	if err != nil || len(items) != 1 || items[0].Text != "Ping 192.168.0.101 every 10s." {
		t.Fatalf("%+v %v", items, err)
	}
}

func TestSpawnAgentUsesTheCallerWorkspace(t *testing.T) {
	withTempAgentHomes(t)
	t.Setenv("KEIMLING_HOST", "")
	t.Setenv("KEIMLING_SECRET", "")
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	other, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	thread := &Agent{Cfg: Config{WorkspaceID: ws.ID}}
	saved := thread.runDefineAgentTool("save_agent", `{"id":"","name":"Runner","instructions":"Run."}`)
	if strings.Contains(saved, "error") {
		t.Fatal(saved)
	}
	started := thread.runDefineAgentTool("spawn_agent", `{"definition":"Runner","name":""}`)
	if strings.Contains(started, "error") || !strings.Contains(started, "saved") {
		t.Fatal(started)
	}
	list, err := listAgentInstances()
	if err != nil || len(list) != 1 || list[0].CurrentWorkspace != ws.ID || list[0].Name != "Runner" {
		t.Fatalf("instance: %+v %v", list, err)
	}
	recipe, err := createAgentRecipe("Keeper", "Keep.")
	if err != nil {
		t.Fatal(err)
	}
	self, err := createAgentInstance(recipe, other.ID, "Keeper")
	if err != nil {
		t.Fatal(err)
	}
	agent := &Agent{Cfg: Config{AgentInstanceID: self.ID, WorkspaceID: ws.ID}}
	side := agent.runDefineAgentTool("spawn_agent", `{"definition":"Runner","name":"Side"}`)
	if strings.Contains(side, "error") || !strings.Contains(side, other.ID) {
		t.Fatal(side)
	}
	self.Workspaces = []string{}
	self.CurrentWorkspace = ws.ID
	if err := writeAgentInstanceMeta(self); err != nil {
		t.Fatal(err)
	}
	blocked := agent.runDefineAgentTool("spawn_agent", `{"definition":"Runner","name":"Nope"}`)
	if !strings.Contains(blocked, "ACL") {
		t.Fatal(blocked)
	}
	after, err := listAgentInstances()
	if err != nil {
		t.Fatal(err)
	}
	for _, inst := range after {
		if inst.Name == "Nope" {
			t.Fatalf("spawned outside ACL: %+v", inst)
		}
	}
}

func TestSaveAndSpawnTeam(t *testing.T) {
	withTempAgentHomes(t)
	t.Setenv("KEIMLING_HOST", "")
	t.Setenv("KEIMLING_SECRET", "")
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{WorkspaceID: ws.ID}}
	for _, name := range []string{"Reader", "Writer"} {
		args, _ := json.Marshal(map[string]string{"id": "", "name": name, "instructions": "Do " + name})
		if s := a.runDefineAgentTool("save_agent", string(args)); strings.Contains(s, "error") {
			t.Fatal(s)
		}
	}
	if _, err := createAgentRecipe("Reader", "other"); err != nil {
		t.Fatal(err)
	}
	ambiguous := a.runDefineAgentTool("save_team", `{"id":"","name":"Crew","definitions":["Reader"]}`)
	if !strings.Contains(ambiguous, "matches") {
		t.Fatal(ambiguous)
	}
	made := a.runDefineAgentTool("save_team", `{"id":"","name":"Crew","definitions":["Writer"]}`)
	if strings.Contains(made, "error") {
		t.Fatal(made)
	}
	teams, err := listAgentTeams()
	if err != nil || len(teams) != 1 || len(teams[0].DefinitionIDs) != 1 {
		t.Fatalf("teams: %+v %v", teams, err)
	}
	spawned := a.runDefineAgentTool("spawn_team", `{"team":"Crew"}`)
	if strings.Contains(spawned, "error") {
		t.Fatal(spawned)
	}
	instances, err := listAgentInstances()
	if err != nil || len(instances) != 1 || instances[0].CurrentWorkspace != ws.ID {
		t.Fatalf("spawned: %+v %v text %s", instances, err, spawned)
	}
}

func TestSpawnAgentAsksTheHostToStart(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rec, err := createAgentRecipe("Runner", "Run.")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/api/agents/instances" {
			http.Error(w, "path", http.StatusNotFound)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"instance": AgentInstanceMeta{
				ID: "agent-host", Name: "Runner", RecipeID: rec.ID, CurrentWorkspace: ws.ID,
			},
		})
	}))
	defer srv.Close()
	t.Setenv("KEIMLING_HOST", srv.URL)
	t.Setenv("KEIMLING_SECRET", "secret")
	a := &Agent{Cfg: Config{WorkspaceID: ws.ID}}
	out := a.runDefineAgentTool("spawn_agent", `{"definition":"`+rec.ID+`","name":"Runner"}`)
	if !strings.Contains(out, "agent-host") || !strings.Contains(out, "started") {
		t.Fatal(out)
	}
	if got["recipe_id"] != rec.ID || got["workspace_id"] != ws.ID {
		t.Fatalf("posted: %+v", got)
	}
	list, err := listAgentInstances()
	if err != nil || len(list) != 0 {
		t.Fatalf("local instances: %+v %v", list, err)
	}
}

func TestThreadReachesTheAgentItStarted(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	other, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Ping", "Ping.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Ping")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createAgentInstance(recipe, other.ID, "Elsewhere"); err != nil {
		t.Fatal(err)
	}
	thread := &Agent{Cfg: Config{WorkspaceID: ws.ID, ThreadID: "thread-1"}}
	names := map[string]bool{}
	for _, tool := range thread.agentExtraTools() {
		name, _ := tool["name"].(string)
		names[name] = true
	}
	for _, name := range []string{"list_agents", "message_agent", "stop_agent", "delete_agent"} {
		if !names[name] {
			t.Fatalf("thread missing %s", name)
		}
	}
	listed := thread.listAgentsTool("")
	if !strings.Contains(listed, inst.ID) || strings.Contains(listed, "Elsewhere") {
		t.Fatalf("list: %s", listed)
	}
	sent := thread.messageAgent(`{"instance_id":"` + inst.ID + `","text":"please stop the watch"}`)
	if strings.Contains(sent, "error") {
		t.Fatal(sent)
	}
	queued, err := dequeueAgentMessages(inst.ID)
	if err != nil || len(queued) != 1 {
		t.Fatalf("%+v %v", queued, err)
	}
	if woke := formatQueueWake(queued[0]); woke != "[From thread_id=thread-1 via message_agent] please stop the watch" {
		t.Fatal(woke)
	}
	if got := thread.stopAgentTool(`{"instance_id":"Ping"}`); got != "stopped Ping" {
		t.Fatal(got)
	}
	if !agentWakesHeld(inst.ID) {
		t.Fatal("stop did not hold wakes")
	}
	if got := thread.messageAgent(`{"instance_id":"` + inst.ID + `","text":"resume"}`); strings.Contains(got, "error") {
		t.Fatal(got)
	}
	if agentWakesHeld(inst.ID) {
		t.Fatal("a thread message left the hold")
	}
	if got := thread.deleteAgentTool(`{"instance_id":"` + inst.ID + `"}`); got != "deleted Ping" {
		t.Fatal(got)
	}
	if _, err := readAgentInstanceMeta(inst.ID); err == nil {
		t.Fatal("instance still present")
	}
	if got := thread.stopAgentTool(`{"instance_id":"Elsewhere"}`); !strings.Contains(got, "unknown agent") {
		t.Fatal(got)
	}
}
