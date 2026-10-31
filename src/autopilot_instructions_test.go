package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAutopilotInstructions(t *testing.T) {
	factory, source := resolveAutopilotInstructions("", "")
	if source != "factory" || !strings.Contains(factory, "AUTOPILOT") {
		t.Fatalf("factory: %q %q", source, factory)
	}
	got, source := resolveAutopilotInstructions("GLOBAL", "")
	if got != "GLOBAL" || source != "global" {
		t.Fatalf("global: %q %q", source, got)
	}
	got, source = resolveAutopilotInstructions("GLOBAL", "THREAD")
	if got != "THREAD" || source != "thread" {
		t.Fatalf("thread: %q %q", source, got)
	}
	got, source = resolveAutopilotInstructions("", "THREAD")
	if got != "THREAD" || source != "thread" {
		t.Fatalf("thread without global: %q %q", source, got)
	}
	if _, err := normalizeAutopilotInstructions("  \n"); err == nil {
		t.Fatal("blank text was accepted")
	}
	got, err := normalizeAutopilotInstructions("  x  ")
	if err != nil || got != "x" {
		t.Fatalf("single word: %q %v", got, err)
	}
}

func TestAutopilotInstructionsSettingsAndThread(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	session := filepath.Join(t.TempDir(), "s.jsonl")
	j, err := OpenJournal(session, false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	const wsID, thID = "ws-pilot", "th-pilot"
	if err := writeThreadMeta(wsID, ThreadMeta{ID: thID, Title: "Pilot"}); err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Cfg: Config{
			Model: "m", Mode: "ask", Reasoning: "high", Web: false, MaxSteps: 20, ContextLimit: 8000,
			SessionPath: session, ConfigPath: cfgPath, WorkspaceID: wsID, ThreadID: thID,
		},
		Journal: j,
	}
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})
	h := srv.Handler()

	get := func(path string) map[string]any {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer tok")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %s", path, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body
	}
	put := func(path, jsonBody string) (int, map[string]any) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(jsonBody))
		req.Header.Set("Authorization", "Bearer tok")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		return rec.Code, body
	}

	snap := get("/api/settings")
	if snap["autopilot_instructions_custom"] != false || !strings.Contains(snap["autopilot_instructions"].(string), "AUTOPILOT") {
		t.Fatalf("settings default: %#v", snap)
	}
	if code, _ := put("/api/settings", `{"autopilot_instructions":"  "}`); code != http.StatusBadRequest {
		t.Fatalf("blank global save: %d", code)
	}
	if a.Cfg.Mode != "ask" {
		t.Fatalf("mode changed: %s", a.Cfg.Mode)
	}

	code, snap := put("/api/settings", `{"autopilot_instructions":"G1","mode":"ask"}`)
	if code != http.StatusOK || snap["autopilot_instructions"] != "G1" || snap["autopilot_instructions_custom"] != true {
		t.Fatalf("save global: %d %#v", code, snap)
	}
	if a.Cfg.Mode != "ask" {
		t.Fatalf("mode changed after save: %s", a.Cfg.Mode)
	}
	fc, err := readConfiguration(cfgPath)
	if err != nil || fc.Features.AutopilotInstructions != "G1" {
		t.Fatalf("persisted global: %q %v", fc.Features.AutopilotInstructions, err)
	}

	view := get("/api/threads/autopilot")
	if view["instructions"] != "G1" || view["custom"] != false || view["source"] != "global" {
		t.Fatalf("thread follows global: %#v", view)
	}

	code, snap = put("/api/settings", `{"autopilot_instructions":"G2"}`)
	if code != http.StatusOK {
		t.Fatalf("G2: %d %#v", code, snap)
	}
	view = get("/api/threads/autopilot")
	if view["instructions"] != "G2" || view["source"] != "global" {
		t.Fatalf("live global: %#v", view)
	}

	code, view = put("/api/threads/autopilot", `{"instructions":"G2"}`)
	if code != http.StatusOK || view["custom"] != true || view["source"] != "thread" || view["instructions"] != "G2" {
		t.Fatalf("detach with same text: %d %#v", code, view)
	}
	code, _ = put("/api/settings", `{"autopilot_instructions":"G3"}`)
	if code != http.StatusOK {
		t.Fatalf("G3: %d", code)
	}
	view = get("/api/threads/autopilot")
	if view["instructions"] != "G2" || view["source"] != "thread" {
		t.Fatalf("thread kept its text: %#v", view)
	}
	if code, _ := put("/api/threads/autopilot", `{"instructions":" \n "}`); code != http.StatusBadRequest {
		t.Fatalf("blank thread save: %d", code)
	}
	view = get("/api/threads/autopilot")
	if view["instructions"] != "G2" {
		t.Fatalf("blank save changed thread: %#v", view)
	}
	code, view = put("/api/threads/autopilot", `{"instructions":"x"}`)
	if code != http.StatusOK || view["instructions"] != "x" {
		t.Fatalf("single word: %d %#v", code, view)
	}

	code, view = put("/api/threads/autopilot", `{"reset":true}`)
	if code != http.StatusOK || view["custom"] != false || view["instructions"] != "G3" || view["source"] != "global" {
		t.Fatalf("thread reset: %d %#v", code, view)
	}
	code, _ = put("/api/settings", `{"autopilot_instructions":"G4"}`)
	if code != http.StatusOK {
		t.Fatalf("G4: %d", code)
	}
	view = get("/api/threads/autopilot")
	if view["instructions"] != "G4" || view["source"] != "global" {
		t.Fatalf("after reset, global edits apply: %#v", view)
	}

	code, snap = put("/api/settings", `{"autopilot_instructions_reset":true}`)
	if code != http.StatusOK || snap["autopilot_instructions_custom"] != false {
		t.Fatalf("settings reset: %d %#v", code, snap)
	}
	if strings.TrimSpace(a.Cfg.AutopilotInstructions) != "" {
		t.Fatalf("memory override remains: %q", a.Cfg.AutopilotInstructions)
	}
	fc, err = readConfiguration(cfgPath)
	if err != nil || strings.TrimSpace(fc.Features.AutopilotInstructions) != "" {
		t.Fatalf("file override remains: %q %v", fc.Features.AutopilotInstructions, err)
	}
	view = get("/api/threads/autopilot")
	if view["source"] != "factory" || !strings.Contains(view["instructions"].(string), "AUTOPILOT") {
		t.Fatalf("factory after settings reset: %#v", view)
	}
	if a.Cfg.Mode != "ask" {
		t.Fatalf("mode changed: %s", a.Cfg.Mode)
	}
	if prompt := a.autopilotPrompt(); !strings.Contains(prompt, "AUTOPILOT") {
		t.Fatalf("supervision prompt: %q", prompt)
	}
}
