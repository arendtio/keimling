package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckModelOpenRouterSuffix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"deepseek/deepseek-v4.1-flash-1017"}]}`))
	}))
	defer server.Close()
	c := NewClient("testkey")
	c.BaseURL = server.URL
	if err := c.CheckModel(context.Background(), "deepseek/deepseek-v4.1-flash-1017"); err != nil {
		t.Fatalf("exact id: %v", err)
	}
	ids := []string{"deepseek/deepseek-v4.1-flash-1017"}
	if err := validateModelInCatalog(ids, "deepseek/deepseek-v4.1-flash-1017:floor", true); err != nil {
		t.Fatalf("suffix should validate against base model: %v", err)
	}
	if err := validateModelInCatalog(ids, "deepseek/does-not-exist:floor", true); err == nil {
		t.Fatal("expected error for unknown base model")
	}
}

func TestSuffixNotStrippedForOtherProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"id":"deepseek/deepseek-v4.1-flash-1017"}]}`))
	}))
	defer server.Close()
	c := NewClient("testkey")
	c.BaseURL = server.URL
	if err := c.CheckModel(context.Background(), "deepseek/deepseek-v4.1-flash-1017:floor"); err == nil {
		t.Fatal("non-OpenRouter endpoints must match IDs exactly")
	}
	ids := []string{"deepseek/deepseek-v4.1-flash-1017"}
	if err := validateModelInCatalog(ids, "deepseek/deepseek-v4.1-flash-1017:floor", false); err == nil {
		t.Fatal("non-OpenRouter must not strip suffix")
	}
}

func TestOpenRouterBaseModel(t *testing.T) {
	if b, ok := openRouterBaseModel("a/b:floor"); !ok || b != "a/b" {
		t.Fatalf("got %q %v", b, ok)
	}
	if _, ok := openRouterBaseModel("a/b"); ok {
		t.Fatal("no suffix expected")
	}
}
