package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// shortenReviewBackoff keeps the retry tests fast and restores the real pauses.
func shortenReviewBackoff(t *testing.T, pauses ...time.Duration) func() {
	t.Helper()
	if len(pauses) == 0 {
		pauses = []time.Duration{time.Millisecond, time.Millisecond}
	}
	old := reviewBackoff
	reviewBackoff = pauses
	return func() { reviewBackoff = old }
}

func writeReviewVerdict(w http.ResponseWriter, decision, reason string) {
	text, _ := json.Marshal(fmt.Sprintf(`{"decision":%q,"reason":%q}`, decision, reason))
	fmt.Fprintf(w, `{"output":[{"type":"message","content":[{"type":"output_text","text":%s}]}]}`, text)
}

func reviewTestServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(srv.Close)
	c := NewClient("")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	return c
}

func askReview(c *Client, ctx context.Context) (Verdict, error) {
	return c.Review(ctx, "review-model", "", "exec", `{"command":"ls"}`, "/work", "", nil, false)
}

// A short reviewer outage is retried, so a hiccup no longer ends the plan.
func TestReviewRetriesATransientOutage(t *testing.T) {
	defer shortenReviewBackoff(t)()
	calls := 0
	c := reviewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		writeReviewVerdict(w, "approve", "routine work")
	})
	verdict, err := askReview(c, context.Background())
	if err != nil || verdict.Decision != "approve" || calls != 3 {
		t.Fatalf("decision=%q calls=%d err=%v", verdict.Decision, calls, err)
	}
}

// Two retries, then the outage stands and the caller decides.
func TestReviewGivesUpAfterTwoRetries(t *testing.T) {
	defer shortenReviewBackoff(t)()
	calls := 0
	c := reviewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "server error", http.StatusInternalServerError)
	})
	verdict, err := askReview(c, context.Background())
	if err == nil || !strings.Contains(err.Error(), "review HTTP 500") || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	human := settleReview(true, true, verdict.Decision, verdict.Reason, err)
	if !human.Ask || human.Deny || !strings.Contains(human.Reason, "review unavailable") {
		t.Fatalf("attended outage: %+v", human)
	}
	alone := settleReview(true, false, verdict.Decision, verdict.Reason, err)
	if !alone.Deny || alone.Ask || !strings.Contains(alone.Reason, "review unavailable") {
		t.Fatalf("unattended outage: %+v", alone)
	}
}

// A canceled run does not wait out the backoff.
func TestReviewBackoffStopsWhenTheContextIsCanceled(t *testing.T) {
	defer shortenReviewBackoff(t, 5*time.Second)()
	calls := 0
	c := reviewTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "server error", http.StatusInternalServerError)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	_, err := askReview(c, ctx)
	if err == nil || calls != 1 || time.Since(start) > 2*time.Second {
		t.Fatalf("calls=%d elapsed=%s err=%v", calls, time.Since(start), err)
	}
}

// The distinction the outage needs: a human decides when one can be asked, and
// a run with nobody there (a live realtime call, a headless run) refuses.
func TestReviewOutageAsksTheHumanUnlessNobodyCanAnswer(t *testing.T) {
	human := settleReview(true, true, "", "", errReviewDown)
	if !human.Ask || human.Deny || !strings.Contains(human.Reason, "review unavailable") {
		t.Fatalf("attended: %+v", human)
	}
	alone := settleReview(true, false, "", "", errReviewDown)
	if !alone.Deny || alone.Ask || !strings.Contains(alone.Reason, "review unavailable") {
		t.Fatalf("unattended: %+v", alone)
	}
	workspace := settleReview(false, true, "", "", errReviewDown)
	if !workspace.Ask || workspace.Deny {
		t.Fatalf("workspace: %+v", workspace)
	}
	if !(&Agent{ApprovalCh: make(chan bool, 1)}).canAskHuman(context.Background(), nil) {
		t.Fatal("the app shows the question, so the agent can ask")
	}
	stdin := bufio.NewReader(strings.NewReader("y\n"))
	if !(&Agent{}).canAskHuman(context.Background(), stdin) {
		t.Fatal("the CLI reads the answer from stdin")
	}
	if (&Agent{}).canAskHuman(context.Background(), nil) {
		t.Fatal("no channel, no reader, no terminal: nobody to ask")
	}
	if !inRealtimeCall(withRealtimeCall(context.Background())) || inRealtimeCall(context.Background()) {
		t.Fatal("the realtime marker is not carried by the context")
	}
	live := withRealtimeCall(context.Background())
	if (&Agent{ApprovalCh: make(chan bool, 1)}).canAskHuman(live, stdin) {
		t.Fatal("a live realtime call has no approval prompt")
	}
	if g := gateToolCall(true, "yolo", "exec", `{"command":"make check"}`, false); !g.Allow || g.Review {
		t.Fatalf("yolo never reviews, so an outage cannot refuse it: %+v", g)
	}
}
