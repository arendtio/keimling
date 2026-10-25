package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExternalWebSearchOnlyUsesExternalClient(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1/responses" {
			t.Errorf("wrong path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer external-only-key" {
			t.Errorf("wrong key: %q", got)
		}
		var payload struct {
			Tools []struct {
				Type string `json:"type"`
			} `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if len(payload.Tools) != 1 || payload.Tools[0].Type != "web_search" {
			t.Error("missing server-side OpenAI search tool")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"Result https://example.com"}]}]}`))
	}))
	defer srv.Close()
	external := NewClient("external-only-key")
	external.BaseURL = srv.URL + "/v1"
	text, err := external.SearchWeb(context.Background(), "Linux news")
	if err != nil || !strings.Contains(text, "example.com") || calls != 1 {
		t.Fatalf("result=%q err=%v calls=%d", text, err, calls)
	}
	model := NewClient("provider-key")
	if model.APIKey != "provider-key" || external.APIKey == model.APIKey || model.BaseURL != defaultBaseURL {
		t.Fatal("provider credentials mixed")
	}
}

func TestExternalSearchDefinition(t *testing.T) {
	found := false
	for _, x := range definitions(true) {
		if x["name"] == "web_search" {
			found = true
			if x["type"] != "function" {
				t.Fatal("web search must be a locally dispatched function")
			}
		}
	}
	if !found {
		t.Fatal("web_search tool not exposed")
	}
	for _, x := range definitions(false) {
		if x["name"] == "web_search" {
			t.Fatal("disabled web search exposed")
		}
	}
}
