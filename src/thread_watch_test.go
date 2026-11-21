package main

import (
	"strings"
	"testing"
	"time"
)

func threadWatchAgent(t *testing.T, wsID, threadID string) *Agent {
	t.Helper()
	withTempAgentHomes(t)
	return &Agent{Cfg: Config{WorkspaceID: wsID, ThreadID: threadID, WatchesLive: true}}
}

func TestThreadOffersWatchToolsWhenAWakeLoopRuns(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	live := &Agent{Cfg: Config{WorkspaceID: ws.ID, ThreadID: th.ID, WatchesLive: true}}
	names := map[string]bool{}
	for _, tool := range live.agentExtraTools() {
		if name, ok := tool["name"].(string); ok {
			names[name] = true
		}
	}
	for _, want := range []string{"create_watch", "list_watches", "cancel_watch"} {
		if !names[want] {
			t.Fatalf("thread with a wake loop is missing %s: %v", want, names)
		}
	}
	off := &Agent{Cfg: Config{WorkspaceID: ws.ID, ThreadID: th.ID}}
	for _, tool := range off.agentExtraTools() {
		if name, _ := tool["name"].(string); strings.HasSuffix(name, "_watch") {
			t.Fatalf("thread without a wake loop offered %s", name)
		}
	}
}

func TestThreadWatchPromptMatchesTheTools(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	live := &Agent{Cfg: Config{WorkspaceID: ws.ID, ThreadID: th.ID, WatchesLive: true}}
	text := live.modelInstructions()
	if strings.Contains(text, "This chat cannot do that") {
		t.Fatalf("live thread still says it cannot watch:\n%s", text)
	}
	if !strings.Contains(text, "create_watch wakes this chat") {
		t.Fatalf("live thread prompt does not name its watch:\n%s", text)
	}
	off := &Agent{Cfg: Config{WorkspaceID: ws.ID, ThreadID: th.ID}}
	if !strings.Contains(off.modelInstructions(), "This chat cannot do that") {
		t.Fatal("thread without a wake loop lost its agent offer")
	}
}

func TestThreadWatchIsStoredInTheThread(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{WorkspaceID: ws.ID, ThreadID: th.ID, WatchesLive: true}}
	got := a.createWatch(`{"when":"in 90s","message":"check the build"}`)
	if !strings.Contains(got, "watch ") {
		t.Fatal(got)
	}
	list, err := readWatches(threadWatchesPath(ws.ID, th.ID))
	if err != nil || len(list) != 1 || list[0].Message != "check the build" {
		t.Fatalf("%+v %v", list, err)
	}
	if !strings.Contains(a.listWatchesTool(""), "check the build") {
		t.Fatal(a.listWatchesTool(""))
	}
	bad := a.createWatch(`{"when":"every 30s","message":"x","steps":[{"name":"exec","arguments":"{}"}],"source":"file","path":"/tmp"}`)
	if !strings.Contains(bad, "wakes on the clock only") {
		t.Fatal(bad)
	}
}

func TestDueThreadWatchQueuesOnce(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	w := AgentWatch{
		ID:      "watch-1",
		Kind:    "time",
		FireAt:  now.Add(-2 * time.Minute),
		Message: "the update finished",
	}
	if err := addAgentWatch(threadWatchesPath(ws.ID, th.ID), w); err != nil {
		t.Fatal(err)
	}
	fired, err := fireDueThreadWatches(ws.ID, th.ID, now)
	if err != nil || fired != 1 {
		t.Fatalf("fired %d %v", fired, err)
	}
	// A restart replays the pump; the wake must not be queued a second time.
	again, err := fireDueThreadWatches(ws.ID, th.ID, now.Add(time.Second))
	if err != nil || again != 0 {
		t.Fatalf("replay fired %d %v", again, err)
	}
	items, err := peekThreadMessages(ws.ID, th.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("%+v %v", items, err)
	}
	if items[0].WatchID != "watch-1" || items[0].Kind != "watch" || !strings.Contains(items[0].Text, "the update finished") {
		t.Fatalf("%+v", items[0])
	}
	if err := dropThreadWatch(ws.ID, th.ID, "watch-1"); err != nil {
		t.Fatal(err)
	}
	if left, _ := readWatches(threadWatchesPath(ws.ID, th.ID)); len(left) != 0 {
		t.Fatalf("watch survived: %+v", left)
	}
}

func TestMissedThreadWatchOlderThanADayIsDropped(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	old := AgentWatch{ID: "watch-old", Kind: "time", FireAt: now.Add(-48 * time.Hour), Message: "stale"}
	if err := addAgentWatch(threadWatchesPath(ws.ID, th.ID), old); err != nil {
		t.Fatal(err)
	}
	fired, err := fireDueThreadWatches(ws.ID, th.ID, now)
	if err != nil || fired != 0 {
		t.Fatalf("stale watch fired %d %v", fired, err)
	}
	if items, _ := peekThreadMessages(ws.ID, th.ID); len(items) != 0 {
		t.Fatalf("stale wake queued: %+v", items)
	}
	if left, _ := readWatches(threadWatchesPath(ws.ID, th.ID)); len(left) != 0 {
		t.Fatalf("stale watch survived: %+v", left)
	}
}

func TestRecurringThreadWatchMovesOnAndQueuesOnce(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	w := AgentWatch{ID: "watch-every", Kind: "time", FireAt: now.Add(-time.Minute), Every: 5 * time.Minute, Message: "poll"}
	if err := addAgentWatch(threadWatchesPath(ws.ID, th.ID), w); err != nil {
		t.Fatal(err)
	}
	if fired, err := fireDueThreadWatches(ws.ID, th.ID, now); err != nil || fired != 1 {
		t.Fatalf("fired %d %v", fired, err)
	}
	if fired, err := fireDueThreadWatches(ws.ID, th.ID, now); err != nil || fired != 0 {
		t.Fatalf("second tick fired %d %v", fired, err)
	}
	list, err := readWatches(threadWatchesPath(ws.ID, th.ID))
	if err != nil || len(list) != 1 {
		t.Fatalf("%+v %v", list, err)
	}
	if !list[0].FireAt.After(now) || !list[0].FiredAt.IsZero() {
		t.Fatalf("recurring watch not advanced: %+v", list[0])
	}
}

func TestThreadWakeIsRequeuedWhenNoRunStarts(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	other, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := enqueueThreadWake(ws.ID, th.ID, "wake up", "watch-9", 0); err != nil {
		t.Fatal(err)
	}
	// Built without a workspace so the wake loop does not race this test.
	s := newAppServer(appServerOpts{Agent: &Agent{Cfg: Config{ThreadID: other.ID, WatchesLive: true}}})
	s.agent.Cfg.WorkspaceID = ws.ID
	// The shell agent sits on another thread and has no model client, so the
	// forked run cannot start and the wake has to stay queued.
	if s.drainThreadQueueFor(ws.ID, th.ID, time.Now()) {
		t.Fatal("run started without a model client")
	}
	items, err := peekThreadMessages(ws.ID, th.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("%+v %v", items, err)
	}
	if items[0].Attempts != 1 || items[0].WatchID != "watch-9" {
		t.Fatalf("%+v", items[0])
	}
}

func TestThreadWakeGivesUpAfterRepeatedFailures(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	w := AgentWatch{ID: "watch-give-up", Kind: "time", FireAt: now.Add(-time.Minute), Message: "try me"}
	if err := addAgentWatch(threadWatchesPath(ws.ID, th.ID), w); err != nil {
		t.Fatal(err)
	}
	if _, err := fireDueThreadWatches(ws.ID, th.ID, now); err != nil {
		t.Fatal(err)
	}
	s := &appServer{}
	item, ok, err := dequeueThreadMessage(ws.ID, th.ID)
	if err != nil || !ok {
		t.Fatalf("%+v %v", item, err)
	}
	s.requeueThreadItem(ws.ID, th.ID, item)
	item, ok, err = dequeueThreadMessage(ws.ID, th.ID)
	if err != nil || !ok || item.Attempts != 1 {
		t.Fatalf("first retry: %+v %v %v", item, ok, err)
	}
	// One more failure reaches wakeThreadRetryLimit: the wake and its watch go.
	s.requeueThreadItem(ws.ID, th.ID, item)
	if items, _ := peekThreadMessages(ws.ID, th.ID); len(items) != 0 {
		t.Fatalf("wake kept retrying: %+v", items)
	}
	if left, _ := readWatches(threadWatchesPath(ws.ID, th.ID)); len(left) != 0 {
		t.Fatalf("watch kept retrying: %+v", left)
	}
}
