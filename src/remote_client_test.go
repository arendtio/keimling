package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewRemoteClientValidation(t *testing.T) {
	if _, err := newRemoteClient("not-a-url", "s"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := newRemoteClient("http://127.0.0.1:9", ""); err == nil {
		t.Fatal("expected secret error")
	}
	c, err := newRemoteClient("http://127.0.0.1:9/", "s")
	if err != nil || c.base != "http://127.0.0.1:9" {
		t.Fatalf("%v %#v", err, c)
	}
}

func TestRemoteClientAgainstAppServer(t *testing.T) {
	j, err := OpenJournal(filepath.Join(t.TempDir(), "s.jsonl"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &Agent{Cfg: Config{Model: "m", Mode: "ask"}, Journal: j}
	_ = a.add(map[string]any{"role": "user", "content": "hi"})
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	c, err := newRemoteClient(ts.URL, "tok")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	h, err := c.Health(ctx)
	if err != nil || h["ok"] != true {
		t.Fatalf("%v %#v", err, h)
	}
	lines, model, _, _, err := c.Session(ctx)
	if err != nil || model != "m" || len(lines) != 1 {
		t.Fatalf("%v %q %#v", err, model, lines)
	}

	var gotLine atomic.Bool
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- c.StreamEvents(ctx2, sseHandler{
			OnLine: func(text string) {
				if strings.Contains(text, "from-hub") {
					gotLine.Store(true)
					cancel()
				}
			},
		})
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		srv.hub.Broadcast("line", map[string]string{"text": "from-hub"})
		if gotLine.Load() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !gotLine.Load() {
		cancel()
		t.Fatal("no sse line")
	}
	<-done
}

func TestReadSSE(t *testing.T) {
	var ready bool
	var line, status string
	err := readSSE(strings.NewReader("event: ready\ndata: {\"ok\":true}\n\nevent: line\ndata: {\"text\":\"x\"}\n\nevent: status\ndata: {\"state\":\"running\"}\n\n"), sseHandler{
		OnReady:  func() { ready = true },
		OnLine:   func(text string) { line = text },
		OnStatus: func(state string) { status = state },
	})
	if err != nil || !ready || line != "x" || status != "running" {
		t.Fatalf("err=%v ready=%v line=%q status=%q", err, ready, line, status)
	}
}

func TestRemoteClientMessageAccepted(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "no", 401)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	c, _ := newRemoteClient(ts.URL, "tok")
	if err := c.Message(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
}
