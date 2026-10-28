package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsProvisionalThreadTitle(t *testing.T) {
	if !isProvisionalThreadTitle("") || !isProvisionalThreadTitle("New thread") {
		t.Fatal("expected provisional")
	}
	if !isProvisionalThreadTitle("20260102-150405.000000000") {
		t.Fatal("id-shaped title should be provisional")
	}
	if isProvisionalThreadTitle("PTY deadlock fix") {
		t.Fatal("real title")
	}
}

func TestCreateThreadDefaultTitle(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	if th.Title != defaultThreadTitle {
		t.Fatalf("title %q", th.Title)
	}
	list, err := listThreads(ws.ID)
	if err != nil || len(list) != 1 || list[0].Title != defaultThreadTitle {
		t.Fatalf("%v %#v", err, list)
	}
}

func TestMaybeGenerateThreadTitle(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, session, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "completed",
			"output": []map[string]any{{
				"type": "message",
				"content": []map[string]any{{
					"type": "output_text",
					"text": "Auth token refresh",
				}},
			}},
		})
	}))
	defer srv.Close()
	a := &Agent{
		Cfg: Config{
			WorkspaceID:  ws.ID,
			ThreadID:     th.ID,
			SessionPath:  session,
			DelegateFast: "fast-model",
			Model:        "main",
		},
		Client: &Client{BaseURL: srv.URL + "/v1", HTTP: srv.Client()},
	}
	var got map[string]any
	a.Emit = func(kind, text string) {
		if kind == "thread" {
			_ = json.Unmarshal([]byte(text), &got)
		}
	}
	a.maybeGenerateThreadTitle(context.Background(), "please fix the broken login token refresh path")
	if got == nil || got["title"] != "Auth token refresh" {
		t.Fatalf("emit %#v", got)
	}
	meta, err := readThreadMeta(ws.ID, th.ID)
	if err != nil || meta.Title != "Auth token refresh" {
		t.Fatalf("%v %#v", err, meta)
	}
	// Second call should not overwrite.
	a.Emit = func(kind, text string) { t.Fatal("should not emit again") }
	a.maybeGenerateThreadTitle(context.Background(), "another message")
	_ = os.WriteFile(filepath.Join(threadDir(ws.ID, th.ID), "keep"), []byte("x"), 0o600)
	if !strings.Contains(meta.Title, "Auth") {
		t.Fatal(meta.Title)
	}
}
