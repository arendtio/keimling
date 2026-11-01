package main

import (
	"encoding/json"
	"os"
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
	if !ok || !strings.Contains(name, "Ada") || !strings.Contains(name, from.ID) || body != "the joke" {
		t.Fatalf("receiver wake: %q", woke)
	}
	rawUser := []byte(`{"role":"user","content":"` + woke + `"}`)
	recv := sessionMessages([]json.RawMessage{rawUser}, 10)
	if len(recv) != 1 || recv[0].Role != "peer" || !strings.Contains(recv[0].Name, "From Ada") || !strings.Contains(recv[0].Name, from.ID) || recv[0].Text != "the joke" {
		t.Fatalf("receiver channel: %+v", recv)
	}
}

func TestAgentPromptNamesItself(t *testing.T) {
	withTempAgentHomes(t)
	recipe, err := createAgentRecipe("Pair", "Watch the alarm.")
	if err != nil {
		t.Fatal(err)
	}
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ada, err := createAgentInstance(recipe, ws.ID, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{AgentInstanceID: ada.ID, AgentInstructions: "Watch the alarm."}}
	block := a.modelInstructions()
	for _, want := range []string{"You are an agent instance", ada.Name, ada.ID, "Your task:\nWatch the alarm.", "not the workspace assistant", "message_agent", "own context", "set_alarm wakes only you", "[Alarm]", "via message_agent", "It is not the human.", "text is a joke you invent", "text is not the request sentence.", "call list_agents"} {
		if !strings.Contains(block, want) {
			t.Fatalf("prompt missing %q\n%s", want, block)
		}
	}
	for _, banned := range []string{"You are keimling, a careful command-line assistant", "EXTREMELY SHORT", "German in this conversation", "Linux CLI", "Linux shell"} {
		if strings.Contains(block, banned) {
			t.Fatalf("agent prompt still has the thread voice %q", banned)
		}
	}
	if !strings.Contains(block, "Operating system: "+hostOSName()) || !strings.Contains(block, "Shell for exec: "+selectedShell()) {
		t.Fatal("agent prompt does not name the host system")
	}
	if strings.Index(block, "You are an agent instance") > strings.Index(block, "Your task:") || strings.Index(block, "Your task:") > strings.Index(block, "How to use tools:") {
		t.Fatal("agent prompt is out of order")
	}
	thread := (&Agent{}).modelInstructions()
	if !strings.Contains(thread, "You are keimling, a careful command-line assistant") || !strings.Contains(thread, "EXTREMELY SHORT") || !strings.Contains(thread, "Operating system: "+hostOSName()) || !strings.Contains(thread, "Shell for exec: "+selectedShell()) {
		t.Fatal("workspace thread lost its own prompt")
	}
	if strings.Contains(block, "[From ") {
		t.Fatal("agent prompt contains a sender line the model can copy")
	}
	if strings.Contains(thread, "You are an agent instance") || strings.Contains(thread, "text is a joke you invent") {
		t.Fatal("workspace thread was given an agent identity")
	}
}

func TestWorkspacePromptIncludesWorkingDirectory(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	dir := t.TempDir()
	ws, err := ensureWorkspaceForRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(ws.Root); err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{WorkspaceID: ws.ID, WorkspaceRoot: ws.Root}}
	block := a.workspacePrompt()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{ws.ID, "Working directory: " + cwd, "Commands start in this directory."} {
		if !strings.Contains(block, want) {
			t.Fatalf("prompt missing %q\n%s", want, block)
		}
	}
	moved := t.TempDir()
	if err := os.Chdir(moved); err != nil {
		t.Fatal(err)
	}
	again := a.workspacePrompt()
	now, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(again, "Working directory: "+now) || strings.Contains(again, "Working directory: "+cwd) {
		t.Fatalf("prompt kept the old directory:\n%s", again)
	}
}

func TestMessageAgentRepliesIntoThread(t *testing.T) {
	withTempAgentHomes(t)
	t.Setenv("KEIMLING_HOST", "")
	t.Setenv("KEIMLING_SECRET", "")
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
	th, session, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	j, err := OpenJournal(agentInstanceSessionPath(ada.ID), false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	sender := &Agent{Cfg: Config{AgentInstanceID: ada.ID}, Journal: j, History: j.History}
	out := sender.messageAgent(`{"instance_id":"` + th.ID + `","text":"twice"}`)
	if !strings.Contains(out, "twice") {
		t.Fatalf("tool result: %s", out)
	}
	events, err := originalEvents(session)
	if err != nil || len(events) != 1 || !strings.Contains(string(events[0]), "twice") || !strings.Contains(string(events[0]), "Ada") {
		t.Fatalf("thread: %v %v", events, err)
	}
	woke := formatQueueWake(AgentQueueItem{Text: "how often", From: th.ID, FromName: th.Title, Workspace: ws.ID, Kind: "thread"})
	if woke != "[From thread_id="+th.ID+" via message_agent] how often" {
		t.Fatalf("thread wake: %q", woke)
	}
	prefixed := sender.messageAgent(`{"instance_id":"agent-` + th.ID + `","text":"again"}`)
	if !strings.Contains(prefixed, "again") {
		t.Fatalf("prefixed thread id: %s", prefixed)
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
