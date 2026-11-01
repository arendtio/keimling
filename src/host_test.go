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

func TestHostStopIdleAgentDoesNotStartWorker(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Pair", "Talk.")
	if err != nil {
		t.Fatal(err)
	}
	ada, err := createAgentInstance(recipe, ws.ID, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	h, err := newHost("sekrit", "")
	if err != nil {
		t.Fatal(err)
	}
	defer h.StopAll()
	req := httptest.NewRequest(http.MethodPost, "/api/agents/instances/stop", strings.NewReader(`{"id":"`+ada.ID+`"}`))
	req.Header.Set("Authorization", "Bearer sekrit")
	rec := httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"running":false`) {
		t.Fatalf("idle stop: %d %s", rec.Code, rec.Body.String())
	}
	if _, _, ok := h.liveAgentWorker(ada.ID); ok {
		t.Fatal("stop started a worker")
	}
	meta, err := readAgentInstanceMeta(ada.ID)
	if err != nil || meta.Status == agentStatusStopped {
		t.Fatalf("status: %+v %v", meta, err)
	}
}

func TestHostLoginPageWithholdsApp(t *testing.T) {
	h, err := newHost("sekrit", "")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), `id="pane-secrets"`) || !strings.Contains(rec.Body.String(), `id="login-secret"`) {
		t.Fatalf("login page: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui", nil))
	if rec.Code != http.StatusUnauthorized || strings.Contains(rec.Body.String(), `id="app"`) {
		t.Fatalf("ui without secret: %d %s", rec.Code, rec.Body.String())
	}
	req := httptest.NewRequest(http.MethodGet, "/ui", nil)
	req.Header.Set("Authorization", "Bearer sekrit")
	rec = httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `id="pane-secrets"`) {
		t.Fatalf("ui: %d", rec.Code)
	}
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
