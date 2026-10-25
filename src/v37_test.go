package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestStrictPatch(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.WriteFile("a.txt", []byte("one\ntwo\nthree\n"), 0600); err != nil {
		t.Fatal(err)
	}
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ -1,3 +1,3 @@\n one\n-two\n+changed\n three\n"
	if err := applyStrictPatch(patch); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "one\nchanged\nthree\n" {
		t.Fatal(string(b))
	}
	if err := applyStrictPatch(patch); err == nil {
		t.Fatal("must reject stale patch")
	}
	if err := applyStrictPatch("--- a/a.txt\n+++ b/a.txt\n@@ -1,1 +1,1 @@\n-nope\n+oops\n"); err == nil {
		t.Fatal("must reject incorrect context")
	}
	b, _ = os.ReadFile("a.txt")
	if string(b) != "one\nchanged\nthree\n" {
		t.Fatal("partial write")
	}
	if err := applyStrictPatch("--- /dev/null\n+++ b/new.txt\n@@ -0,0 +1,1 @@\n+created\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "new.txt")); err != nil {
		t.Fatal(err)
	}
}
func TestCompactMaterialSplitsLongHistory(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var request map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		var input []struct {
			Content string `json:"content"`
		}
		_ = json.Unmarshal(request["input"], &input)
		if len(input) == 0 || len(input[0].Content) > 140000 {
			t.Errorf("large input %d", len(input[0].Content))
		}
		i := calls.Add(1)
		fmt.Fprintf(w, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"condensed part %d"}]}],"usage":{"input_tokens":12,"output_tokens":5}}`, i)
	}))
	defer server.Close()
	c := NewClient("dummy")
	c.BaseURL = server.URL
	history := []json.RawMessage{json.RawMessage(`{"role":"user","content":"` + strings.Repeat("x", 300000) + `"}`)}
	result, err := compactMaterial(context.Background(), c, "mock", history, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "condensed") {
		t.Fatal(result)
	}
	if calls.Load() < 4 {
		t.Fatalf("expected multiple passes: %d", calls.Load())
	}
}
