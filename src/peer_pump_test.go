package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// peerPumpFixture is one workspace with an attached thread whose peer wakes are
// recorded instead of run, so a test sees the wake without a model call.
type peerPumpFixture struct {
	srv      *appServer
	agent    *Agent
	wsID     string
	threadID string
	woken    []string
}

func newPeerPumpFixture(t *testing.T) *peerPumpFixture {
	t.Helper()
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, session, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	j, err := OpenJournal(session, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = j.Close() })
	a := &Agent{
		Cfg:     Config{Model: "m", Mode: "ask", WorkspaceID: ws.ID, ThreadID: th.ID, SessionPath: session},
		Journal: j,
	}
	f := &peerPumpFixture{
		srv:      newAppServer(appServerOpts{AccessSecret: "tok", Agent: a}),
		agent:    a,
		wsID:     ws.ID,
		threadID: th.ID,
	}
	f.srv.startPeerRun = func(text, wsID, threadID string, item AgentQueueItem) bool {
		f.woken = append(f.woken, wsID+"/"+threadID+": "+text)
		return true
	}
	return f
}

// deliver posts one peer message to the attached thread.
func (f *peerPumpFixture) deliver(t *testing.T, name, text string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"workspace_id": f.wsID,
		"thread_id":    f.threadID,
		"from_name":    name,
		"from_id":      "agent-" + strings.ToLower(name),
		"text":         text,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/threads/deliver", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	f.srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("deliver %q: %d %s", text, rec.Code, rec.Body.String())
	}
}

// setBusy marks the attached thread as running, as it is while a run is in
// flight.
func (f *peerPumpFixture) setBusy(t *testing.T, busy bool) {
	t.Helper()
	f.srv.mu.Lock()
	f.srv.busy = busy
	f.srv.mu.Unlock()
}

func TestBusyPeerMessageWakesTheAttachedThreadAfterTheRun(t *testing.T) {
	f := newPeerPumpFixture(t)
	// A peer message that arrives during a run is stored and starts nothing.
	f.setBusy(t, true)
	f.deliver(t, "Ada", "ping")
	if len(f.woken) != 0 {
		t.Fatalf("busy deliver woke the thread: %q", f.woken)
	}
	// The run ends: the pump starts exactly one run for the stored message.
	f.setBusy(t, false)
	f.srv.pumpAgentWakes()
	want := f.wsID + "/" + f.threadID + ": [From Ada] ping"
	if len(f.woken) != 1 || f.woken[0] != want {
		t.Fatalf("wake after the run: %q", f.woken)
	}
	if name, body, ok := parsePeerUserText("[From Ada] ping"); !ok || name != "Ada" || body != "ping" {
		t.Fatalf("wake text is not a peer message: %q %q %v", name, body, ok)
	}
	// No loop: the same idle thread starts nothing more.
	f.srv.pumpAgentWakes()
	if len(f.woken) != 1 {
		t.Fatalf("pump woke twice: %q", f.woken)
	}
	// A further peer message during the peer chain stays quiet until the human
	// writes; the human turn re-arms the wake for the next busy message.
	f.deliver(t, "Ada", "again")
	f.srv.pumpAgentWakes()
	if len(f.woken) != 1 {
		t.Fatalf("peer chain woke twice: %q", f.woken)
	}
	f.srv.noteUserTurn()
	f.setBusy(t, true)
	f.deliver(t, "Ada", "third")
	f.setBusy(t, false)
	f.srv.pumpAgentWakes()
	if len(f.woken) != 2 || !strings.HasSuffix(f.woken[1], "[From Ada] third") {
		t.Fatalf("wake after the human turn: %q", f.woken)
	}
}

func TestBusyPeerMessagesShareOneWake(t *testing.T) {
	f := newPeerPumpFixture(t)
	f.setBusy(t, true)
	f.deliver(t, "Ada", "one")
	f.deliver(t, "Bob", "two")
	f.deliver(t, "Ada", "three")
	f.setBusy(t, false)
	f.srv.pumpAgentWakes()
	want := f.wsID + "/" + f.threadID + ": [From Ada] one\n[From Bob] two\n[From Ada] three"
	if len(f.woken) != 1 || f.woken[0] != want {
		t.Fatalf("one run for three messages: %q", f.woken)
	}
	name, body, ok := parsePeerUserText("[From Ada] one\n[From Bob] two\n[From Ada] three")
	if !ok || name != "Ada" || !strings.Contains(body, "two") || !strings.Contains(body, "three") {
		t.Fatalf("wake text: %q %q %v", name, body, ok)
	}
	f.srv.pumpAgentWakes()
	if len(f.woken) != 1 {
		t.Fatalf("pump woke twice: %q", f.woken)
	}
}

func TestPeerPumpIgnoresOwnOutputAndARealtimeCall(t *testing.T) {
	f := newPeerPumpFixture(t)
	// An item this thread sent to a peer is own output, not a wake.
	f.setBusy(t, true)
	raw, err := json.Marshal(peerSentItem("Ada", "hello Ada"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.agent.Journal.AppendItemFileOnly(raw); err != nil {
		t.Fatal(err)
	}
	f.setBusy(t, false)
	f.srv.pumpAgentWakes()
	if len(f.woken) != 0 {
		t.Fatalf("own output woke the thread: %q", f.woken)
	}
	// A live realtime call keeps its own delivery; the message waits for it.
	f.setBusy(t, true)
	f.deliver(t, "Ada", "during the call")
	f.setBusy(t, false)
	f.srv.mu.Lock()
	f.srv.realtimeOn = true
	f.srv.mu.Unlock()
	f.srv.pumpAgentWakes()
	if len(f.woken) != 0 {
		t.Fatalf("realtime call was bypassed: %q", f.woken)
	}
	f.srv.mu.Lock()
	f.srv.realtimeOn = false
	f.srv.mu.Unlock()
	f.srv.pumpAgentWakes()
	if len(f.woken) != 1 || !strings.HasSuffix(f.woken[0], "[From Ada] during the call") {
		t.Fatalf("message lost after the call: %q", f.woken)
	}
}
