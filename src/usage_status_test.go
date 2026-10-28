package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLastMainUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","output":[],"usage":{"input_tokens":1200,"output_tokens":75,"input_tokens_details":{"cached_tokens":300},"output_tokens_details":{"reasoning_tokens":40}}}`))
	}))
	defer server.Close()
	client := NewClient("")
	client.BaseURL = server.URL
	history := []json.RawMessage{json.RawMessage(`{"role":"user","content":"test"}`)}
	_, err := client.Respond(context.Background(), "test-model", "low", history, false, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	u := client.LastMainUsage
	if !u.Available || u.InputTokens != 1200 || u.OutputTokens != 75 || u.CachedTokens != 300 || u.ReasoningTokens != 40 || u.EstimatedBefore != contextEstimate(history) {
		t.Fatalf("unexpected usage: %+v", u)
	}
	if client.InputTokens != 1200 || client.OutputTokens != 75 {
		t.Fatalf("totals not updated")
	}
}
