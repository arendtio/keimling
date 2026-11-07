package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWatchDecideKeepsEqualsAndSkipsTheFirstChange(t *testing.T) {
	eq := AgentWatch{Condition: "equals", Value: "ok"}
	match, _, _ := watchDecide(eq, "ok")
	if !match {
		t.Fatal("equals should match")
	}
	match, _, _ = watchDecide(eq, "ok")
	if !match {
		t.Fatal("equals should match again while the value stays")
	}
	match, _, _ = watchDecide(eq, "no")
	if match {
		t.Fatal("a different value should not match equals")
	}
	changed := AgentWatch{Condition: "changed"}
	match, last, seen := watchDecide(changed, "one")
	if match || !seen || last != "one" {
		t.Fatalf("first change sample: match %v seen %v last %q", match, seen, last)
	}
	changed.Seen, changed.Last = seen, last
	match, _, _ = watchDecide(changed, "one")
	if match {
		t.Fatal("the same result is not a change")
	}
	match, last, _ = watchDecide(changed, "two")
	if !match || last != "two" {
		t.Fatalf("later difference: match %v last %q", match, last)
	}
}

func TestTimeWatchWithoutACondition(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Waker", "Wake.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Waker-1")
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{AgentInstanceID: inst.ID}}
	got := a.createWatch(`{"when":"in 2m","message":"later","condition":"","value":"","steps":[],"source":"","path":"","command":""}`)
	if !strings.Contains(got, "watch ") || !strings.Contains(got, "time") {
		t.Fatal(got)
	}
	list := a.listWatchesTool("")
	if !strings.Contains(list, "later") || !strings.Contains(list, "time") {
		t.Fatal(list)
	}
	id := strings.Fields(strings.TrimPrefix(list, "- "))[0]
	if !strings.HasPrefix(a.cancelWatch(`{"id":"`+id+`"}`), "watch cancelled") {
		t.Fatal("cancel")
	}
	if a.listWatchesTool("") != "no watches" {
		t.Fatal(a.listWatchesTool(""))
	}
}

func TestIntervalWatchRunsTheCommandAndStays(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Waker", "Wake.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Waker-1")
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{AgentInstanceID: inst.ID}}
	stepArgs, _ := json.Marshal(`{"command":"printf hi","timeout_seconds":5,"interactive":false,"raw_output":true,"output_focus":""}`)
	got := a.createWatch(`{"when":"every 1s","message":"saw","condition":"","value":"","steps":[{"name":"exec","arguments":` + string(stepArgs) + `}],"source":"","path":"","command":""}`)
	if !strings.Contains(got, "interval") {
		t.Fatal(got)
	}
	past := time.Now().Add(-time.Second)
	list, err := readAgentAlarms(inst.ID)
	if err != nil || len(list) != 1 {
		t.Fatal(err, len(list))
	}
	list[0].FireAt = past
	if err := writeAgentAlarms(inst.ID, list); err != nil {
		t.Fatal(err)
	}
	a.runDueIntervalWatches(context.Background(), time.Now())
	queued, err := dequeueAgentMessages(inst.ID)
	if err != nil || len(queued) != 1 || !strings.Contains(queued[0].Text, "hi") {
		t.Fatalf("%+v %v", queued, err)
	}
	left, err := readAgentAlarms(inst.ID)
	if err != nil || len(left) != 1 || left[0].Kind != "interval" {
		t.Fatalf("watch should stay: %+v %v", left, err)
	}
	if !agentWatchNeedsProcess(inst.ID) {
		t.Fatal("an interval watch needs the worker")
	}
}

func TestFileWatchWakesOnAChange(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Waker", "Wake.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Waker-1")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	a := &Agent{Cfg: Config{AgentInstanceID: inst.ID}}
	got := a.createWatch(`{"when":"","message":"file","condition":"","value":"","steps":[],"source":"file","path":"` + dir + `","command":""}`)
	if !strings.Contains(got, "file") {
		t.Fatal(got)
	}
	a.ensureLiveWatches()
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	var queued []AgentQueueItem
	for time.Now().Before(deadline) {
		queued, err = dequeueAgentMessages(inst.ID)
		if err == nil && len(queued) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(queued) == 0 || !strings.Contains(queued[0].Text, "note.txt") {
		t.Fatalf("%+v %v", queued, err)
	}
	watchHubFor(inst.ID).stop(strings.Fields(strings.TrimPrefix(a.listWatchesTool(""), "- "))[0])
}

func TestWorkspaceChatHasNoWatch(t *testing.T) {
	a := &Agent{Cfg: Config{WorkspaceID: "ws", ThreadID: "th"}}
	for _, tool := range a.agentExtraTools() {
		name, _ := tool["name"].(string)
		if name == "create_watch" || name == "list_watches" || name == "cancel_watch" {
			t.Fatalf("workspace chat offered %s", name)
		}
	}
	if !strings.Contains(a.createWatch(`{}`), "only available on agent instances") {
		t.Fatal(a.createWatch(`{}`))
	}
	text := a.modelInstructions()
	for _, want := range []string{"not an agent", "create_watch", "save_agent", "spawn_agent", "This thread's id is th", "not specific enough", "message_agent recipient", "stop_agent", "delete_agent"} {
		if !strings.Contains(text, want) {
			t.Fatalf("workspace prompt missing %q\n%s", want, text)
		}
	}
}

func TestDeleteInstanceRemovesOnlyItsWatches(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Waker", "Wake.")
	if err != nil {
		t.Fatal(err)
	}
	first, err := createAgentInstance(recipe, ws.ID, "One")
	if err != nil {
		t.Fatal(err)
	}
	second, err := createAgentInstance(recipe, ws.ID, "Two")
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{AgentInstanceID: first.ID}}
	b := &Agent{Cfg: Config{AgentInstanceID: second.ID}}
	rawA, _ := json.Marshal(map[string]any{
		"when": "", "message": "only-one", "condition": "", "value": "",
		"steps": []any{}, "source": "command", "path": "", "command": "sleep 60",
	})
	if got := a.createWatch(string(rawA)); !strings.Contains(got, "command") {
		t.Fatal(got)
	}
	rawB, _ := json.Marshal(map[string]any{
		"when": "in 1h", "message": "only-two", "condition": "", "value": "",
		"steps": []any{}, "source": "", "path": "", "command": "",
	})
	if got := b.createWatch(string(rawB)); !strings.Contains(got, "time") {
		t.Fatal(got)
	}
	if got := a.listWatchesTool(""); strings.Contains(got, "only-two") || !strings.Contains(got, "only-one") {
		t.Fatalf("instance one sees: %s", got)
	}
	if got := b.listWatchesTool(""); strings.Contains(got, "only-one") || !strings.Contains(got, "only-two") {
		t.Fatalf("instance two sees: %s", got)
	}
	t.Cleanup(func() { stopInstanceWatches(first.ID) })
	a.ensureLiveWatches()
	var pid int
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, n := range readWatchPIDs(first.ID) {
			if n > 0 {
				pid = n
			}
		}
		if pid > 0 && processGroupAlive(pid) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if pid == 0 || !processGroupAlive(pid) {
		t.Fatalf("command did not start, pid %d", pid)
	}
	if err := deleteAgentInstance(first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(agentInstanceWatchesPath(first.ID)); !os.IsNotExist(err) {
		t.Fatalf("watches.json still there: %v", err)
	}
	if processGroupAlive(pid) {
		t.Fatalf("command %d still running", pid)
	}
	if got := b.listWatchesTool(""); !strings.Contains(got, "only-two") {
		t.Fatalf("other instance lost its watch: %s", got)
	}
}

func TestCommandWatchReportsTheLineThenTheExit(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Waker", "Wake.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Waker-1")
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{AgentInstanceID: inst.ID}}
	raw, _ := json.Marshal(map[string]any{
		"when": "", "message": "cmd", "condition": "", "value": "",
		"steps": []any{}, "source": "command", "path": "", "command": "printf 'ready\\n'",
	})
	if got := a.createWatch(string(raw)); !strings.Contains(got, "command") {
		t.Fatal(got)
	}
	a.ensureLiveWatches()
	deadline := time.Now().Add(2 * time.Second)
	var text string
	for time.Now().Before(deadline) {
		queued, err := dequeueAgentMessages(inst.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range queued {
			text += item.Text + "\n"
		}
		if strings.Contains(text, "ready") && strings.Contains(text, "command ended") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("command watch: %s", text)
}
