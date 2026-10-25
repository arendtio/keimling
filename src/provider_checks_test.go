package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path=%s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer testkey" {
			t.Error("missing auth")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"google/gemini-test"}]}`))
	}))
	defer server.Close()
	c := NewClient("testkey")
	c.BaseURL = server.URL + "/v1"
	if err := c.CheckModel(context.Background(), "google/gemini-test"); err != nil {
		t.Fatal(err)
	}
	if err := c.CheckModel(context.Background(), "missing"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestNoAPIKeyHeaderWhenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("unexpected auth")
		}
		w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	c := NewClient("")
	c.BaseURL = server.URL + "/v1"
	if err := c.CheckModel(context.Background(), "absent"); err == nil {
		t.Fatal("expected missing model")
	}
}

func TestSafeEndpointDisplay(t *testing.T) {
	got := endpointDisplay("https://user:secret@example.com/v1/responses?access_token=secret#secret")
	if strings.Contains(got, "secret") || strings.Contains(got, "user") {
		t.Fatal(got)
	}
	if !isOpenRouter("https://openrouter.ai/api/v1") {
		t.Fatal("openrouter detection")
	}
}
