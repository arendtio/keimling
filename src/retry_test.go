package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRespondRetriesFailedServerError(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			fmt.Fprint(w, `{"status":"failed","error":{"code":"server_error","message":"Internal Server Error"}}`)
			return
		}
		fmt.Fprint(w, `{"status":"completed","output":[],"usage":{"input_tokens":11,"output_tokens":2}}`)
	}))
	defer srv.Close()
	c := NewClient("")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	out, err := c.Respond(context.Background(), "test-model", "low", nil, false, "")
	if err != nil || out.Status != "completed" || calls != 3 {
		t.Fatalf("status=%q calls=%d err=%v", out.Status, calls, err)
	}
	if c.InputTokens != 11 {
		t.Fatalf("usage duplicated: %d", c.InputTokens)
	}
}
func TestRespondDoesNotRetryPermanentFailure(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"status":"failed","error":{"code":"invalid_request_error"}}`)
	}))
	defer srv.Close()
	c := NewClient("")
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	_, err := c.Respond(context.Background(), "model", "low", nil, false, "")
	if err == nil || !strings.Contains(err.Error(), "invalid_request_error") || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
func TestRespondCanceledBeforeRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewClient("")
	_, err := c.Respond(ctx, "model", "low", nil, false, "")
	if err != context.Canceled {
		t.Fatalf("err=%v", err)
	}
}
