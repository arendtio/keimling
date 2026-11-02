package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestBraveSearchUsesSubscriptionToken(t *testing.T) {
	var gotHeader, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Subscription-Token")
		gotQuery = r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"web":{"results":[{"title":"Docs","url":"https://example.com/a","description":"A page"}]}}`))
	}))
	defer srv.Close()
	text, err := runSearch(context.Background(), searchConfig{Kind: "brave", URL: srv.URL, APIKey: "brave-key"}, "keimling")
	if err != nil {
		t.Fatal(err)
	}
	if gotHeader != "brave-key" || gotQuery != "keimling" {
		t.Fatalf("header %q query %q", gotHeader, gotQuery)
	}
	if !strings.Contains(text, "https://example.com/a") || !strings.Contains(text, "Docs") {
		t.Fatalf("text %q", text)
	}
}

func TestURLSearchPlacesQuery(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.RequestURI()
		if r.Header.Get("Authorization") != "Bearer url-key" {
			t.Errorf("auth %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"results":[{"title":"Hit","url":"https://example.com/b","content":"snip"}]}`))
	}))
	defer srv.Close()
	text, err := runSearch(context.Background(), searchConfig{Kind: "url", URL: srv.URL + "/search?q={query}", APIKey: "url-key"}, "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "q=hello+world") && !strings.Contains(path, "q=hello%20world") {
		t.Fatalf("path %s", path)
	}
	if !strings.Contains(text, "https://example.com/b") {
		t.Fatalf("text %q", text)
	}
}

func TestSearchOffDoesNotCall(t *testing.T) {
	_, err := runSearch(context.Background(), searchConfig{}, "q")
	if err == nil || !strings.Contains(err.Error(), "off") {
		t.Fatalf("%v", err)
	}
}

func TestVoiceUsesChosenProvider(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := updateConfiguration(cfgPath, func(fc *fileConfig) {
		fc.ModelProviders = []modelProvider{
			{ID: "local", Name: "Local", BaseURL: "http://127.0.0.1:9931/v1", Kind: "local", Model: "gemma"},
			{ID: "oa", Name: "OpenAI", BaseURL: "https://api.openai.com/v1", APIKey: "sk", Kind: "openai", Model: "gpt"},
		}
		fc.ActiveModelProviderID = "oa"
	}); err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{ConfigPath: cfgPath, VoiceProviderID: "local", VoiceModel: "whisper-1", HasModelProvider: true}}
	c, model, err := a.voiceClient()
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL != "http://127.0.0.1:9931/v1" || c.APIKey != "" || model != "whisper-1" {
		t.Fatalf("base %q key %q model %q", c.BaseURL, c.APIKey, model)
	}
	a.Cfg.VoiceProviderID = ""
	if _, _, err := a.voiceClient(); err == nil {
		t.Fatal("voice without a provider should be off")
	}
}

func TestMigrateSearchOnce(t *testing.T) {
	fc := fileConfig{}
	fc.ModelProviders = []modelProvider{{ID: "oa", Name: "OpenAI", BaseURL: defaultBaseURL, APIKey: "sk", Kind: "openai"}}
	fc.OpenAI.Key = "sk-ext"
	fc.VoiceModel = "gpt-transcribe"
	if !migrateServiceChoices(&fc) {
		t.Fatal("expected migration")
	}
	if fc.Search.Kind != "openai" || fc.Search.APIKey != "sk-ext" || fc.VoiceProviderID != "oa" {
		t.Fatalf("%#v voice %q", fc.Search, fc.VoiceProviderID)
	}
	fc.Search.Kind = ""
	fc.Search.APIKey = ""
	fc.VoiceProviderID = ""
	if migrateServiceChoices(&fc) {
		t.Fatal("migrated again")
	}
	if fc.Search.Kind != "" || fc.VoiceProviderID != "" {
		t.Fatal("choice restored")
	}
}
