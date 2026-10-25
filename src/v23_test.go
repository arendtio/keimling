package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLongOutputAlwaysUsesSuccessfulFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "completed", "output": []any{map[string]any{"type": "message", "content": []any{map[string]any{"type": "output_text", "text": "CPU: Intel i7-4770S 3.10GHz"}}}}})
	}))
	defer server.Close()
	c := NewClient("")
	c.BaseURL = server.URL + "/v1"
	a := &Agent{Client: c}
	original := "exit_code=0\n" + strings.Repeat("CPU: Intel i7-4770S 3.10GHz\n", 11)
	result := a.filterExecResult(context.Background(), "Identify CPU", `{"command":"lscpu"}`, original)
	if result == original {
		t.Fatal("long output was not filtered")
	}
	if result != "exit_code=0\n[Filtered]\nCPU: Intel i7-4770S 3.10GHz" {
		t.Fatalf("unexpected result: %q", result)
	}
}
