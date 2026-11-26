package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func gatewayPair(t *testing.T) (from, to AgentInstanceMeta) {
	t.Helper()
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Pair", "Talk.")
	if err != nil {
		t.Fatal(err)
	}
	from, err = createAgentInstance(recipe, ws.ID, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	to, err = createAgentInstance(recipe, ws.ID, "Bea")
	if err != nil {
		t.Fatal(err)
	}
	if err := addAgentInstancePeer(from.ID, to.ID); err != nil {
		t.Fatal(err)
	}
	return from, to
}

func TestGatewayInspectionHoldsUntilAllowed(t *testing.T) {
	from, to := gatewayPair(t)
	if err := setGatewayInspect(from.ID, to.ID, true); err != nil {
		t.Fatal(err)
	}
	if !gatewayInspect(from.ID, to.ID) {
		t.Fatal("rule not stored")
	}
	if gatewayInspect(to.ID, from.ID) {
		t.Fatal("the other direction must stay open")
	}
	sender := &Agent{Cfg: Config{AgentInstanceID: from.ID}}
	done := make(chan string, 1)
	go func() {
		done <- sender.messageAgent(`{"instance_id":"` + to.ID + `","text":"the joke"}`)
	}()
	var held []heldMessage
	for i := 0; i < 50; i++ {
		held = listHeldMessages()
		if len(held) == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(held) != 1 {
		t.Fatalf("held = %+v", held)
	}
	if held[0].From != from.ID || held[0].To != to.ID || held[0].Text != "the joke" {
		t.Fatalf("held entry = %+v", held[0])
	}
	if items, _ := dequeueAgentMessages(to.ID); len(items) != 0 {
		t.Fatalf("a held message must not reach the recipient: %+v", items)
	}
	if err := decideHeldMessage(held[0].ID, true); err != nil {
		t.Fatal(err)
	}
	select {
	case out := <-done:
		if !strings.Contains(out, "Sent to") {
			t.Fatalf("sender result: %s", out)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the sender did not continue after the allow")
	}
	items, err := dequeueAgentMessages(to.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Text != "the joke" || items[0].Kind != "peer" || items[0].From != from.ID {
		t.Fatalf("delivered = %+v", items)
	}
	if left := listHeldMessages(); len(left) != 0 {
		t.Fatalf("the decided message must leave the list: %+v", left)
	}
}

func TestGatewayBlockStopsTheMessage(t *testing.T) {
	from, to := gatewayPair(t)
	if err := setGatewayInspect(from.ID, to.ID, true); err != nil {
		t.Fatal(err)
	}
	id, ch, err := holdGatewayMessage(from.ID, "Ada", to.ID, "Bea", "block me")
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan string, 1)
	go func() {
		status, _ := waitGatewayDecision(context.Background(), id, ch)
		result <- status
	}()
	if err := decideHeldMessage(id, false); err != nil {
		t.Fatal(err)
	}
	select {
	case status := <-result:
		if status != "blocked" {
			t.Fatalf("status = %q", status)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the waiter was not released")
	}
	if items, _ := dequeueAgentMessages(to.ID); len(items) != 0 {
		t.Fatalf("a blocked message must not be delivered: %+v", items)
	}
	if left := listHeldMessages(); len(left) != 0 {
		t.Fatalf("the blocked message must leave the list: %+v", left)
	}
}

func TestGatewayOffDeliversAtOnce(t *testing.T) {
	from, to := gatewayPair(t)
	sender := &Agent{Cfg: Config{AgentInstanceID: from.ID}}
	out := sender.messageAgent(`{"instance_id":"` + to.ID + `","text":"straight through"}`)
	if !strings.Contains(out, "Sent to") {
		t.Fatalf("result: %s", out)
	}
	if left := listHeldMessages(); len(left) != 0 {
		t.Fatalf("nothing should be held: %+v", left)
	}
	items, err := dequeueAgentMessages(to.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Text != "straight through" {
		t.Fatalf("delivered = %+v", items)
	}
}

func TestGatewayEdgesListBothDirections(t *testing.T) {
	from, to := gatewayPair(t)
	edges := gatewayEdges()
	var forward, back bool
	for _, e := range edges {
		if e.From == from.ID && e.To == to.ID {
			forward = true
		}
		if e.From == to.ID && e.To == from.ID {
			back = true
		}
	}
	if !forward || !back {
		t.Fatalf("edges = %+v", edges)
	}
	if err := setGatewayInspect(from.ID, to.ID, true); err != nil {
		t.Fatal(err)
	}
	for _, e := range gatewayEdges() {
		if e.From == from.ID && e.To == to.ID && !e.Inspect {
			t.Fatal("edge did not report the rule")
		}
	}
	if err := setGatewayInspect(from.ID, to.ID, false); err != nil {
		t.Fatal(err)
	}
	for _, e := range gatewayEdges() {
		if e.From == from.ID && e.To == to.ID && e.Inspect {
			t.Fatal("edge kept a cleared rule")
		}
	}
}

func TestWaitingApprovalIsListedWithoutTheChat(t *testing.T) {
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: &Agent{ApprovalCh: make(chan bool, 1), Cfg: Config{ThreadID: "main"}}})
	srv.noteApproval("main", map[string]any{"name": "exec", "reason": "run it", "command": "ls"})
	list := srv.approvalList()
	if len(list) != 1 {
		t.Fatalf("list = %+v", list)
	}
	if list[0]["name"] != "exec" || list[0]["thread_id"] != "main" {
		t.Fatalf("entry = %+v", list[0])
	}
	srv.noteApproval("main", map[string]any{"clear": true})
	if len(srv.approvalList()) != 0 {
		t.Fatal("a clear must remove the entry")
	}
}

func TestHostInboxDecidesForAnUnopenedWorker(t *testing.T) {
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: &Agent{ApprovalCh: make(chan bool, 1), Cfg: Config{ThreadID: "main"}}})
	h := &Host{secret: "tok", workers: map[string]*hostWorker{
		agentWorkerKey("agent-1"): {Key: agentWorkerKey("agent-1"), Kind: "agent", Ref: "agent-1", Secret: "tok", Handler: srv.Handler()},
	}}
	srv.noteApproval("main", map[string]any{"name": "exec", "reason": "run it"})
	items := h.listApprovals(context.Background())
	if len(items) != 1 || items[0].Worker != agentWorkerKey("agent-1") || items[0].Tool != "exec" {
		t.Fatalf("items = %+v", items)
	}
	rec := httptest.NewRecorder()
	body := `{"worker":"` + items[0].Worker + `","thread_id":"` + items[0].ThreadID + `","allow":true}`
	h.handleApprovalDecide(rec, httptest.NewRequest(http.MethodPost, "/api/approvals/decide", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("decide: %d %s", rec.Code, rec.Body.String())
	}
	if len(srv.approvalList()) != 0 {
		t.Fatal("the decision must clear the entry")
	}
}
