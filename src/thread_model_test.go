package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThreadChatModel(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	const wsID, thID = "ws-model", "th-model"
	if err := writeThreadMeta(wsID, ThreadMeta{ID: thID, Title: "Model", AutopilotInstructions: "KEEP"}); err != nil {
		t.Fatal(err)
	}
	fc := fileConfig{}
	fc.ContextLimit = 8000
	fc.Reasoning = "medium"
	fc.ActiveModelProviderID = "p2"
	fc.ModelProviders = []modelProvider{
		{
			ID: "p1", Name: "One", BaseURL: "https://one.example/v1", APIKey: "k1", Reasoning: "xhigh",
			ActiveModelID: "m1",
			Models: []providerModel{
				{ID: "m1", Name: "Model One", ContextLimit: 32000, Reasoning: "high"},
				{ID: "m1b", Name: "Model One B", Reasoning: "medium"},
				{ID: "m1c", Name: "Model One C"},
			},
		},
		{
			ID: "p2", Name: "Two", BaseURL: "https://two.example/v1", APIKey: "k2",
			ActiveModelID: "m2",
			Models: []providerModel{
				{ID: "m2", Name: "Model Two", ContextLimit: 16000, Reasoning: "low"},
			},
		},
	}
	writeModelConfig(t, cfgPath, fc)
	a := &Agent{
		Cfg: Config{
			Model: "from-flags", Reasoning: "medium", ContextLimit: 8000,
			BaseURL: "https://flags.example/v1", Key: "flag-key",
			CompactionModel: "compact-model", ReviewModel: "review-model",
			ConfigPath: cfgPath, WorkspaceID: wsID, ThreadID: thID,
		},
		Client: &Client{BaseURL: "https://flags.example/v1", APIKey: "flag-key"},
	}

	got := a.effectiveChatTarget()
	if got.Custom || got.Model != "m2" || got.Label != "Model Two" || got.Reasoning != "low" || got.ContextLimit != 16000 {
		t.Fatalf("settings default: %+v", got)
	}
	if got.BaseURL != "https://two.example/v1" || got.APIKey != "k2" {
		t.Fatalf("settings endpoint: %+v", got)
	}
	if a.compactionModel() != "compact-model" || a.Cfg.Model != "from-flags" {
		t.Fatalf("settings roles changed: model %q compaction %q", a.Cfg.Model, a.compactionModel())
	}

	fc.ActiveModelProviderID = "p1"
	fc.ModelProviders[0].ActiveModelID = "m1b"
	writeModelConfig(t, cfgPath, fc)
	got = a.effectiveChatTarget()
	if got.Custom || got.Model != "m1b" || got.Reasoning != "medium" || got.ContextLimit != 8000 {
		t.Fatalf("follow settings, global context fallback: %+v", got)
	}

	if err := a.saveThreadChatModel("p1", "nope"); err == nil {
		t.Fatal("unknown model was accepted")
	}
	if err := a.saveThreadChatModel("p1", "m1"); err != nil {
		t.Fatal(err)
	}
	got = a.effectiveChatTarget()
	if !got.Custom || got.ProviderID != "p1" || got.Model != "m1" || got.Reasoning != "high" || got.ContextLimit != 32000 {
		t.Fatalf("saved choice: %+v", got)
	}
	if a.contextBudget().Limit != 32000 {
		t.Fatalf("context budget limit %d", a.contextBudget().Limit)
	}
	meta, err := readThreadMeta(wsID, thID)
	if err != nil || meta.AutopilotInstructions != "KEEP" || meta.ModelID != "m1" {
		t.Fatalf("meta: %+v %v", meta, err)
	}

	fc.ActiveModelProviderID = "p2"
	fc.ModelProviders[0].ActiveModelID = "m1"
	fc.ModelProviders[1].ActiveModelID = "m2"
	writeModelConfig(t, cfgPath, fc)
	got = a.effectiveChatTarget()
	if !got.Custom || got.Model != "m1" {
		t.Fatalf("kept after settings change: %+v", got)
	}

	if err := a.saveThreadChatModel("p2", "m2"); err != nil {
		t.Fatal(err)
	}
	meta, _ = readThreadMeta(wsID, thID)
	if meta.ModelProviderID != "" || meta.ModelID != "" || meta.AutopilotInstructions != "KEEP" {
		t.Fatalf("selecting settings did not clear: %+v", meta)
	}
	fc.ActiveModelProviderID = "p1"
	fc.ModelProviders[0].ActiveModelID = "m1b"
	writeModelConfig(t, cfgPath, fc)
	got = a.effectiveChatTarget()
	if got.Custom || got.Model != "m1b" {
		t.Fatalf("follows settings after clear: %+v", got)
	}

	if err := a.saveThreadChatModel("p1", "m1"); err != nil {
		t.Fatal(err)
	}
	fc.ModelProviders[0].ActiveModelID = "m1"
	writeModelConfig(t, cfgPath, fc)
	got = a.effectiveChatTarget()
	if !got.Custom || got.Model != "m1" {
		t.Fatalf("coincidence still saved: %+v", got)
	}
	fc.ActiveModelProviderID = "p2"
	fc.ModelProviders[1].ActiveModelID = "m2"
	writeModelConfig(t, cfgPath, fc)
	got = a.effectiveChatTarget()
	if !got.Custom || got.Model != "m1" {
		t.Fatalf("stays after settings leave the coincidence: %+v", got)
	}

	if err := a.saveThreadChatModel("p1", "m1c"); err != nil {
		t.Fatal(err)
	}
	got = a.effectiveChatTarget()
	if got.Reasoning != "medium" || got.ContextLimit != 8000 {
		t.Fatalf("empty model reasoning/context: %+v", got)
	}
	if err := a.saveThreadChatModel("p1", "m1"); err != nil {
		t.Fatal(err)
	}

	a.beginRunChat()
	if err := a.saveThreadChatModel("p2", "m2"); err != nil {
		t.Fatal(err)
	}
	var model, reasoning, base, key string
	a.withChatEndpoint(func(m, r string) {
		model, reasoning = m, r
		base, key = a.Client.BaseURL, a.Client.APIKey
	})
	if model != "m1" || reasoning != "high" || base != "https://one.example/v1" || key != "k1" {
		t.Fatalf("in-flight call: %s %s %s %s", model, reasoning, base, key)
	}
	if a.Client.BaseURL != "https://flags.example/v1" || a.Client.APIKey != "flag-key" {
		t.Fatalf("endpoint not restored: %s %s", a.Client.BaseURL, a.Client.APIKey)
	}
	if a.compactionModel() != "compact-model" || a.Cfg.Model != "from-flags" || a.Cfg.ReviewModel != "review-model" {
		t.Fatal("run retargeted process settings")
	}
	a.endRunChat()
	got = a.effectiveChatTarget()
	if got.Custom || got.Model != "m2" {
		t.Fatalf("next send: %+v", got)
	}

	if err := a.saveThreadChatModel("p1", "m1"); err != nil {
		t.Fatal(err)
	}
	fc.ModelProviders[0].Models = []providerModel{{ID: "m1b", Name: "Model One B", Reasoning: "medium"}}
	fc.ModelProviders[0].ActiveModelID = "m1b"
	writeModelConfig(t, cfgPath, fc)
	got = a.effectiveChatTarget()
	if got.Custom || got.Model != "m2" {
		t.Fatalf("deleted model falls back to settings: %+v", got)
	}
	meta, _ = readThreadMeta(wsID, thID)
	if meta.ModelID != "" || meta.AutopilotInstructions != "KEEP" {
		t.Fatalf("stale choice: %+v", meta)
	}

	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/threads/model", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "k1") || strings.Contains(rec.Body.String(), "k2") {
		t.Fatalf("GET model: %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPut, "/api/threads/model", strings.NewReader(`{"provider_id":"p1","model_id":"missing"}`))
	req.Header.Set("Authorization", "Bearer tok")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT unknown: %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPut, "/api/threads/model", strings.NewReader(`{"provider_id":"p1","model_id":"m1b"}`))
	req.Header.Set("Authorization", "Bearer tok")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"custom":true`) {
		t.Fatalf("PUT choice: %d %s", rec.Code, rec.Body.String())
	}
}

func writeModelConfig(t *testing.T, path string, fc fileConfig) {
	t.Helper()
	b, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
