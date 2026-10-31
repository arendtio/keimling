package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestToolGrantNilAllowsAllAndExplicitDenies(t *testing.T) {
	if !toolNameGranted(nil, "exec") || !toolNameGranted(nil, "mcp__x__y") {
		t.Fatal("nil grant allows every tool")
	}
	only := []string{"set_alarm", "mcp"}
	if !toolNameGranted(only, "set_alarm") || !toolNameGranted(only, "mcp__demo__ping") {
		t.Fatal("explicit grant should allow the named tool and mcp")
	}
	if toolNameGranted(only, "exec") || toolNameGranted([]string{}, "set_alarm") {
		t.Fatal("tools outside the saved choice are denied")
	}
	defs := []map[string]any{{"name": "exec"}, {"name": "set_alarm"}, {"name": "mcp__demo__ping"}}
	filtered := filterGrantedTools(defs, only)
	if len(filtered) != 2 || filtered[0]["name"] != "set_alarm" || filtered[1]["name"] != "mcp__demo__ping" {
		t.Fatalf("filtered: %#v", filtered)
	}
	if len(filterGrantedTools(defs, nil)) != 3 {
		t.Fatal("nil grant does not filter")
	}
	chat := &Agent{}
	if chat.instanceToolGrant() != nil || !chat.instanceToolGranted("exec") {
		t.Fatal("a workspace chat is not tool-filtered")
	}
}

func TestSetAgentInstanceAccessGrantsWorkspacesAndTools(t *testing.T) {
	withTempAgentHomes(t)
	ws1, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ws2, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Cross", "Work across workspaces.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws1.ID, "Cross-1")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Tools != nil {
		t.Fatal("a new instance has no saved tool choice")
	}
	tools := []string{"set_alarm", "message_agent"}
	got, err := setAgentInstanceAccess(inst.ID, []string{ws2.ID, ws1.ID}, &tools)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Workspaces) != 2 || got.Tools == nil || len(*got.Tools) != 2 {
		t.Fatalf("access: %+v", got)
	}
	in2, err := listAgentInstancesForWorkspace(ws2.ID)
	if err != nil || len(in2) != 1 || in2[0].ID != inst.ID {
		t.Fatalf("instance should appear in the second workspace: %+v %v", in2, err)
	}
	a := &Agent{Cfg: Config{AgentInstanceID: inst.ID}}
	if !a.instanceToolGranted("set_alarm") || a.instanceToolGranted("exec") {
		t.Fatalf("grant = %#v", a.instanceToolGrant())
	}
	if _, err := setAgentInstanceAccess(inst.ID, nil, &tools); err == nil {
		t.Fatal("expected at least one workspace")
	}
	bad := []string{"not_a_tool"}
	if _, err := setAgentInstanceAccess(inst.ID, []string{ws1.ID}, &bad); err == nil {
		t.Fatal("expected unknown tool")
	}
	if _, err := setAgentInstanceAccess(inst.ID, []string{"missing-workspace"}, &tools); err == nil {
		t.Fatal("expected unknown workspace")
	}
}

func TestInstanceToolsFollowDefinitionAndCannotWiden(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Narrow", "Only alarms.")
	if err != nil {
		t.Fatal(err)
	}
	only := []string{"set_alarm", "list_alarms"}
	if _, err := setRecipeTools(recipe.ID, only); err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Narrow-1")
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{AgentInstanceID: inst.ID}}
	if !a.instanceToolGranted("set_alarm") || a.instanceToolGranted("exec") || a.instanceToolGranted("message_agent") {
		t.Fatal("instance should follow the definition")
	}
	wider := []string{"set_alarm", "exec"}
	if _, err := setAgentInstanceAccess(inst.ID, []string{ws.ID}, &wider); err == nil {
		t.Fatal("instance cannot add a tool the definition does not grant")
	}
	same := []string{"list_alarms", "set_alarm"}
	got, err := setAgentInstanceAccess(inst.ID, []string{ws.ID}, &same)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tools != nil {
		t.Fatal("a set equal to the definition follows the definition")
	}
	narrow := []string{"set_alarm"}
	got, err = setAgentInstanceAccess(inst.ID, []string{ws.ID}, &narrow)
	if err != nil || got.Tools == nil || len(*got.Tools) != 1 {
		t.Fatalf("narrow: %+v %v", got.Tools, err)
	}
	if a.instanceToolGranted("list_alarms") {
		t.Fatal("override should drop list_alarms")
	}
	shrunk := []string{"message_agent"}
	if _, err := setRecipeTools(recipe.ID, shrunk); err != nil {
		t.Fatal(err)
	}
	if a.instanceToolGranted("set_alarm") || a.instanceToolGranted("message_agent") {
		t.Fatalf("a saved narrower set cannot keep a dropped tool or gain a new one: %#v", a.instanceToolGrant())
	}
}

func TestPeerSendIsVisibleToBothChannels(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Pair", "Talk.")
	if err != nil {
		t.Fatal(err)
	}
	from, err := createAgentInstance(recipe, ws.ID, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	to, err := createAgentInstance(recipe, ws.ID, "Bea")
	if err != nil {
		t.Fatal(err)
	}
	if err := addAgentInstancePeer(from.ID, to.ID); err != nil {
		t.Fatal(err)
	}
	j, err := OpenJournal(agentInstanceSessionPath(from.ID), false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	sender := &Agent{Cfg: Config{AgentInstanceID: from.ID}, Journal: j, History: j.History}
	out := sender.messageAgent(`{"instance_id":"` + to.ID + `","text":"the joke"}`)
	if !strings.Contains(out, "Bea") || !strings.Contains(out, "the joke") {
		t.Fatalf("tool result: %s", out)
	}
	view := sessionMessages(sender.History, 20)
	var sent bool
	for _, m := range view {
		if m.Role == "peer" && m.Name == "To Bea" && m.Text == "the joke" {
			sent = true
		}
	}
	if !sent {
		t.Fatalf("sender channel: %+v", view)
	}
	woke := formatQueueWake(AgentQueueItem{Text: "the joke", From: from.ID, Kind: "peer"})
	name, body, ok := parsePeerUserText(woke)
	if !ok || name != "Ada" || body != "the joke" {
		t.Fatalf("receiver wake: %q", woke)
	}
	rawUser := []byte(`{"role":"user","content":"` + woke + `"}`)
	recv := sessionMessages([]json.RawMessage{rawUser}, 10)
	if len(recv) != 1 || recv[0].Role != "peer" || recv[0].Name != "From Ada" || recv[0].Text != "the joke" {
		t.Fatalf("receiver channel: %+v", recv)
	}
}

func TestAgentToolGroupsCoverCatalogOnce(t *testing.T) {
	seen := map[string]int{}
	var filesWrite, alarmRead, alarmWrite []string
	for _, g := range agentToolGroups() {
		if g.ID == "files" {
			filesWrite = append([]string(nil), g.Write...)
		}
		if g.ID == "alarms" {
			alarmRead = append([]string(nil), g.Read...)
			alarmWrite = append([]string(nil), g.Write...)
		}
		for _, name := range append(append([]string{}, g.Read...), g.Write...) {
			seen[name]++
		}
	}
	if strings.Join(filesWrite, ",") != "edit_file,write_file" {
		t.Fatalf("file writing: %v", filesWrite)
	}
	if strings.Join(alarmRead, ",") != "list_alarms" || strings.Join(alarmWrite, ",") != "set_alarm,cancel_alarm" {
		t.Fatalf("alarms read %v write %v", alarmRead, alarmWrite)
	}
	for _, tool := range agentToolCatalog() {
		if seen[tool.Name] != 1 {
			t.Fatalf("tool %s is in %d groups", tool.Name, seen[tool.Name])
		}
		delete(seen, tool.Name)
	}
	if len(seen) != 0 {
		t.Fatalf("groups name tools outside the catalog: %v", seen)
	}
	for _, g := range agentToolGroups() {
		if g.ID == "mcp" {
			t.Fatal("mcp is not a read/write topic")
		}
	}
}

func TestMCPServerGrantIsOneServer(t *testing.T) {
	if !toolNameGranted([]string{"mcp__alpha"}, "mcp__alpha__echo") {
		t.Fatal("a server choice allows that server's tools")
	}
	if toolNameGranted([]string{"mcp__alpha"}, "mcp__beta__echo") || toolNameGranted([]string{"mcp__alpha"}, "exec") {
		t.Fatal("a server choice does not allow another server or a builtin tool")
	}
	if !toolNameGranted([]string{"mcp"}, "mcp__beta__echo") {
		t.Fatal("the legacy mcp choice still allows every server")
	}
	if _, err := cleanToolNames([]string{"mcp__alpha__echo"}); err == nil {
		t.Fatal("an individual mcp tool is not a server choice")
	}
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("MCP", "One server.")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setRecipeTools(recipe.ID, []string{"set_alarm", "mcp__alpha"}); err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "MCP-1")
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{AgentInstanceID: inst.ID}}
	if !a.instanceToolGranted("mcp__alpha__echo") || a.instanceToolGranted("mcp__beta__echo") {
		t.Fatal("instance should follow the definition's server")
	}
	other := []string{"set_alarm", "mcp__beta"}
	if _, err := setAgentInstanceAccess(inst.ID, []string{ws.ID}, &other); err == nil {
		t.Fatal("instance cannot add a server the definition does not grant")
	}
	narrow := []string{"set_alarm"}
	if _, err := setAgentInstanceAccess(inst.ID, []string{ws.ID}, &narrow); err != nil {
		t.Fatal(err)
	}
	if a.instanceToolGranted("mcp__alpha__echo") {
		t.Fatal("narrowed instance should not keep the server")
	}
}

func TestUserSetsBidirectionalContacts(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Pair", "Talk.")
	if err != nil {
		t.Fatal(err)
	}
	ada, err := createAgentInstance(recipe, ws.ID, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	bea, err := createAgentInstance(recipe, ws.ID, "Bea")
	if err != nil {
		t.Fatal(err)
	}
	cleo, err := createAgentInstance(recipe, ws.ID, "Cleo")
	if err != nil {
		t.Fatal(err)
	}
	if err := addAgentInstancePeer(ada.ID, bea.ID); err != nil {
		t.Fatal(err)
	}
	ada, _ = readAgentInstanceMeta(ada.ID)
	bea, _ = readAgentInstanceMeta(bea.ID)
	if !agentsMayContact(bea, ada) {
		t.Fatal("an older one-way link still lets the other side reply")
	}
	bj, err := OpenJournal(agentInstanceSessionPath(bea.ID), false)
	if err != nil {
		t.Fatal(err)
	}
	defer bj.Close()
	fromBea := &Agent{Cfg: Config{AgentInstanceID: bea.ID}, Journal: bj, History: bj.History}
	if out := fromBea.messageAgent(`{"instance_id":"` + ada.ID + `","text":"back"}`); !strings.Contains(out, "back") {
		t.Fatalf("reply: %s", out)
	}
	if _, err := setAgentInstancePeers(ada.ID, []string{bea.ID}); err != nil {
		t.Fatal(err)
	}
	ada, _ = readAgentInstanceMeta(ada.ID)
	bea, _ = readAgentInstanceMeta(bea.ID)
	cleo, _ = readAgentInstanceMeta(cleo.ID)
	if !peersContain(ada.Peers, bea.ID) || !peersContain(bea.Peers, ada.ID) {
		t.Fatalf("both sides should list each other: ada %v bea %v", ada.Peers, bea.Peers)
	}
	if agentsMayContact(ada, cleo) {
		t.Fatal("Cleo was not included")
	}
	aj, err := OpenJournal(agentInstanceSessionPath(ada.ID), false)
	if err != nil {
		t.Fatal(err)
	}
	defer aj.Close()
	fromAda := &Agent{Cfg: Config{AgentInstanceID: ada.ID}, Journal: aj, History: aj.History}
	if out := fromAda.messageAgent(`{"instance_id":"` + cleo.ID + `","text":"no"}`); !strings.Contains(out, "not allowed") {
		t.Fatalf("third agent: %s", out)
	}
	if out := fromAda.allowPeer(`{"instance_id":"` + cleo.ID + `"}`); !strings.Contains(out, "Manage agents") {
		t.Fatalf("allow_peer: %s", out)
	}
	ada, _ = readAgentInstanceMeta(ada.ID)
	if agentsMayContact(ada, cleo) {
		t.Fatal("the model must not add a contact")
	}
	if _, err := setAgentInstancePeers(ada.ID, nil); err != nil {
		t.Fatal(err)
	}
	ada, _ = readAgentInstanceMeta(ada.ID)
	bea, _ = readAgentInstanceMeta(bea.ID)
	if agentsMayContact(ada, bea) || len(bea.Peers) != 0 {
		t.Fatalf("uncheck should stop both directions: ada %v bea %v", ada.Peers, bea.Peers)
	}
	if _, err := setAgentInstancePeers(ada.ID, []string{"not-an-agent"}); err == nil {
		t.Fatal("unknown agent should be rejected")
	}
}

func TestMessageAgentOwnIDIsNotAContactDenial(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Pair", "Talk.")
	if err != nil {
		t.Fatal(err)
	}
	ada, err := createAgentInstance(recipe, ws.ID, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	bea, err := createAgentInstance(recipe, ws.ID, "Bea")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setAgentInstancePeers(ada.ID, []string{bea.ID}); err != nil {
		t.Fatal(err)
	}
	j, err := OpenJournal(agentInstanceSessionPath(ada.ID), false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	fromAda := &Agent{Cfg: Config{AgentInstanceID: ada.ID}, Journal: j, History: j.History}
	out := fromAda.messageAgent(`{"instance_id":"` + ada.ID + `","text":"hi"}`)
	if strings.Contains(out, "has not allowed") || !strings.Contains(out, "this agent") {
		t.Fatalf("own id: %s", out)
	}
	out = fromAda.messageAgent(`{"instance_id":"` + bea.ID + `","text":"over"}`)
	if !strings.Contains(out, "Sent to") {
		t.Fatalf("peer: %s", out)
	}
}
