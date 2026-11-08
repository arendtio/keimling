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
	"time"
)

func TestCallTitleSeedUsesSpokenLines(t *testing.T) {
	user, _ := json.Marshal(map[string]any{"role": "user", "content": "the login token refresh is broken"})
	asst, _ := json.Marshal(map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": "I can look at the refresh path"}}})
	call, _ := json.Marshal(map[string]any{"type": "function_call", "name": "read_file", "arguments": "{}"})
	got := callTitleSeed([]json.RawMessage{user, asst, call})
	if !strings.Contains(got, "User: the login token refresh is broken") || !strings.Contains(got, "Assistant: I can look") || strings.Contains(got, "read_file") {
		t.Fatal(got)
	}
	if callTitleSeed(nil) != "" {
		t.Fatal("empty history")
	}
}

func TestCallNamesThreadAfterAWhile(t *testing.T) {
	callTitleDelay = 20 * time.Millisecond
	callTitleRetry = time.Millisecond
	t.Cleanup(func() {
		callTitleDelay = 20 * time.Second
		callTitleRetry = 5 * time.Second
	})
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "completed",
			"output": []map[string]any{{
				"type":    "message",
				"content": []map[string]any{{"type": "output_text", "text": "Login token refresh"}},
			}},
		})
	}))
	defer srv.Close()
	user, _ := json.Marshal(map[string]any{"role": "user", "content": "the login token refresh is broken"})
	a := &Agent{
		Cfg:     Config{WorkspaceID: ws.ID, ThreadID: th.ID, DelegateFast: "fast-model"},
		Client:  &Client{BaseURL: srv.URL + "/v1", HTTP: srv.Client()},
		History: []json.RawMessage{user},
	}
	s := &appServer{agent: a}
	s.noteCallThreadTitle()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		meta, err := readThreadMeta(ws.ID, th.ID)
		if err == nil && meta.Title == "Login token refresh" {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	meta, _ := readThreadMeta(ws.ID, th.ID)
	t.Fatalf("title %q", meta.Title)
}

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
	hold := make(chan struct{})
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(hold)
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "completed",
			"output": []map[string]any{{
				"type":    "message",
				"content": []map[string]any{{"type": "output_text", "text": "Renamed Thread"}},
			}},
		})
	}))
	defer slow.Close()
	fresh, session2, err := createThread(ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	_ = session2
	a2 := &Agent{
		Cfg:    Config{WorkspaceID: ws.ID, ThreadID: fresh.ID, DelegateFast: "fast-model"},
		Client: &Client{BaseURL: slow.URL + "/v1", HTTP: slow.Client()},
	}
	done := make(chan struct{})
	go func() {
		a2.maybeGenerateThreadTitle(context.Background(), "please rename this thread from the first line")
		close(done)
	}()
	<-hold
	if err := writeThreadMeta(ws.ID, ThreadMeta{ID: fresh.ID, Title: defaultThreadTitle, ModelProviderID: "prov", ModelID: "kept-model"}); err != nil {
		t.Fatal(err)
	}
	close(release)
	<-done
	kept, err := readThreadMeta(ws.ID, fresh.ID)
	if err != nil || kept.ModelID != "kept-model" || kept.ModelProviderID != "prov" || kept.Title != "Renamed Thread" {
		t.Fatalf("title rewrite dropped the model: %+v %v", kept, err)
	}

	// Second call should not overwrite.
	a.Emit = func(kind, text string) { t.Fatal("should not emit again") }
	a.maybeGenerateThreadTitle(context.Background(), "another message")
	_ = os.WriteFile(filepath.Join(threadDir(ws.ID, th.ID), "keep"), []byte("x"), 0o600)
	if !strings.Contains(meta.Title, "Auth") {
		t.Fatal(meta.Title)
	}
}
