package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppServerHealthRequiresSecret(t *testing.T) {
	h := newAppServer(appServerOpts{AccessSecret: "s3cret"}).Handler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated health: got %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Authorization", "Bearer s3cret")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authenticated health: got %d body %q", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["ok"] != true {
		t.Fatalf("body: %#v", body)
	}
}

func TestAppServerAcceptsHeaderSecret(t *testing.T) {
	h := newAppServer(appServerOpts{AccessSecret: "tok"}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("X-Keimling-Access-Secret", "tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestAppServerIndexIsPublic(t *testing.T) {
	h := newAppServer(appServerOpts{AccessSecret: "tok"}).Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "Keimling") {
		t.Fatalf("unexpected body: %q", body)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type: %q", ct)
	}
}

func TestAppServerRejectsWrongSecret(t *testing.T) {
	h := newAppServer(appServerOpts{AccessSecret: "right"}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestAppServerSession(t *testing.T) {
	j, err := OpenJournal(filepath.Join(t.TempDir(), "s.jsonl"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &Agent{Cfg: Config{Model: "m", Mode: "ask"}, Journal: j, History: nil}
	if err := a.add(map[string]any{"role": "user", "content": "hello"}); err != nil {
		t.Fatal(err)
	}
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})
	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Lines []string `json:"lines"`
		Model string   `json:"model"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Model != "m" || len(body.Lines) != 1 || !strings.Contains(body.Lines[0], "hello") {
		t.Fatalf("%#v", body)
	}
}

func TestAppServerEventsSSE(t *testing.T) {
	srv := newAppServer(appServerOpts{AccessSecret: "tok"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		srv.Handler().ServeHTTP(rec, req)
		close(done)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(rec.Body.String(), "event: ready") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(rec.Body.String(), "event: ready") {
		cancel()
		<-done
		t.Fatalf("body %q", rec.Body.String())
	}
	srv.hub.Broadcast("line", map[string]string{"text": "ping-line"})
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(rec.Body.String(), "ping-line") {
			cancel()
			<-done
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done
	t.Fatalf("missing broadcast in %q", rec.Body.String())
}

func TestSessionTranscript(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"role": "user", "content": "hi"})
	lines := sessionTranscript([]json.RawMessage{raw}, 10)
	if len(lines) != 1 || lines[0] != "› hi" {
		t.Fatalf("%#v", lines)
	}
}

func TestAppServerTranscribeRequiresAudio(t *testing.T) {
	a := &Agent{Cfg: Config{VoiceModel: "gpt-transcribe"}, Client: &Client{APIKey: "k"}}
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})
	req := httptest.NewRequest(http.MethodPost, "/api/transcribe", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code < 400 {
		t.Fatalf("expected client error, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestAppServerVoiceFlag(t *testing.T) {
	a := &Agent{Client: &Client{APIKey: "k"}}
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["voice"] != true {
		t.Fatalf("%#v", body)
	}
}

func TestAppServerServesAnsiJS(t *testing.T) {
	h := newAppServer(appServerOpts{AccessSecret: "tok"}).Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/ansi.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ansiToHtml") {
		t.Fatal("missing ansiToHtml")
	}
}

func TestAppServerNewChatAndSettings(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "s.jsonl")
	j, err := OpenJournal(session, false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &Agent{
		Cfg:     Config{Model: "m1", Mode: "auto", Reasoning: "high", Web: true, MaxSteps: 50, ContextLimit: 64000, SessionPath: session, VoiceModel: "gpt-transcribe"},
		Journal: j,
	}
	if err := a.add(map[string]any{"role": "user", "content": "old"}); err != nil {
		t.Fatal(err)
	}
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})
	h := srv.Handler()

	a.Client = &Client{BaseURL: "https://api.openai.com/v1"}
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"base_url":"http://127.0.0.1:8080/v1","mode":"ask","model":"m2","reasoning":"low","web":false,"max_steps":12}`))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings: %d %s", rec.Code, rec.Body.String())
	}
	if a.Cfg.Mode != "ask" || a.Cfg.Model != "m2" || a.Cfg.Reasoning != "low" || a.Cfg.Web || a.Cfg.MaxSteps != 12 {
		t.Fatalf("cfg %#v", a.Cfg)
	}
	if a.Cfg.BaseURL != "http://127.0.0.1:8080/v1" || a.Client.BaseURL != "http://127.0.0.1:8080/v1" {
		t.Fatalf("base url cfg=%q client=%q", a.Cfg.BaseURL, a.Client.BaseURL)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/newchat", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("newchat: %d %s", rec.Code, rec.Body.String())
	}
	if len(a.History) != 0 {
		t.Fatalf("history not cleared: %d", len(a.History))
	}
}

func TestAppServerSettingsAPIKeysPersist(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	session := filepath.Join(dir, "s.jsonl")
	j, err := OpenJournal(session, false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &Agent{
		Cfg: Config{
			Model: "m1", Mode: "auto", Reasoning: "high", Web: true, MaxSteps: 50, ContextLimit: 64000,
			SessionPath: session, VoiceModel: "gpt-transcribe", BaseURL: "https://api.openai.com/v1",
			ConfigPath: cfgPath,
		},
		Client:  &Client{BaseURL: "https://api.openai.com/v1"},
		Journal: j,
	}
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})
	h := srv.Handler()

	body := `{"base_url":"https://openrouter.ai/api/v1","model":"or-model","mode":"ask","reasoning":"medium","web":true,"api_key":"sk-or-test","openai_api_key":"sk-openai-test"}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "sk-or-test") || strings.Contains(rec.Body.String(), "sk-openai-test") {
		t.Fatalf("response leaked secrets: %s", rec.Body.String())
	}
	var snap map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if snap["api_key_set"] != true || snap["openai_api_key_set"] != true {
		t.Fatalf("key flags: %#v", snap)
	}
	if a.Client.APIKey != "sk-or-test" || a.Cfg.Key != "sk-or-test" {
		t.Fatalf("provider key not applied: client=%q cfg=%q", a.Client.APIKey, a.Cfg.Key)
	}
	if a.Cfg.ExternalClient == nil || a.Cfg.ExternalClient.APIKey != "sk-openai-test" {
		t.Fatalf("openai key not applied: %#v", a.Cfg.ExternalClient)
	}

	fc, err := readConfiguration(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if fc.API.Key != "sk-or-test" || fc.OpenAI.Key != "sk-openai-test" {
		t.Fatalf("config keys: api=%q openai=%q", fc.API.Key, fc.OpenAI.Key)
	}
	if fc.API.BaseURL != "https://openrouter.ai/api/v1" || fc.API.Model != "or-model" {
		t.Fatalf("config api: %#v", fc.API)
	}
	st, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0077 != 0 {
		t.Fatalf("config perms too open: %o", st.Mode().Perm())
	}

	// Omit keys → leave unchanged.
	req = httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"model":"or-model-2","mode":"ask","reasoning":"low"}`))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings2: %d %s", rec.Code, rec.Body.String())
	}
	if a.Client.APIKey != "sk-or-test" || a.Cfg.ExternalClient.APIKey != "sk-openai-test" {
		t.Fatal("keys changed when omitted")
	}
}
