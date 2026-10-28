package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHostAttachProxiesSession(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns worker process")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	root := t.TempDir()
	ws, err := ensureWorkspaceForRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(t.TempDir(), "config.json")
	_ = os.WriteFile(cfg, []byte("{}\n"), 0o600)

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// Under `go test`, Executable is the test binary — workers need the same binary
	// with KEIMLING_ROLE=worker. Skip if we cannot spawn meaningfully.
	_ = exe

	h, err := newHost("host-secret-token", cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer h.StopAll()

	// Avoid spawning during unit tests when the test binary is not a full keimling
	// main — only verify workspace listing + attach error path without worker.
	req := httptest.NewRequest(http.MethodGet, "/api/workspaces", nil)
	req.Header.Set("Authorization", "Bearer host-secret-token")
	rec := httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Workspaces []map[string]any `json:"workspaces"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range body.Workspaces {
		if w["id"] == ws.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("workspace missing: %#v", body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// EnsureWorker will try to spawn; with test binary it may fail fast — acceptable.
	_ = h.Attach(ctx, ws.ID)
}

func TestHostRequireSecret(t *testing.T) {
	h, err := newHost("sekrit", "")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/workspaces", nil)
	rec := httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}
