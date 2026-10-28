package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCustomEndpointNoKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"r1","status":"completed","output":[]}`))
	}))
	defer srv.Close()
	c := NewClient("")
	c.BaseURL = srv.URL + "/v1"
	if _, e := c.Respond(context.Background(), "local-model", "low", nil, false, "", nil); e != nil {
		t.Fatal(e)
	}
}
func TestNormalizeBaseURL(t *testing.T) {
	for _, v := range []string{"http://localhost:8080/v1", "https://example.org/v1/"} {
		u, e := normalizeBaseURL(v)
		if e != nil || u == "" {
			t.Fatalf("%s %v", v, e)
		}
	}
	for _, v := range []string{"file:///tmp/server", "https://user:pass@host/v1", "http://host/v1?q=1"} {
		if _, e := normalizeBaseURL(v); e == nil {
			t.Errorf("accepted %s", v)
		}
	}
}
