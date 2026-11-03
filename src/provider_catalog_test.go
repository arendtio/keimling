package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseModelCatalogOpenRouterShape(t *testing.T) {
	raw := []byte(`{"data":[{"id":"acme/fast","name":"Fast","context_length":200000,"pricing":{"prompt":"0.000001","completion":"0.000002"}},{"id":"plain"}]}`)
	got, err := parseModelCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%#v", got)
	}
	if got[0].ContextLimit != 200000 || got[0].PromptPrice != "0.000001" {
		t.Fatalf("%#v", got[0])
	}
	if got[1].ID != "plain" || got[1].ContextLimit != 0 {
		t.Fatalf("%#v", got[1])
	}
}

func TestFetchProviderModelCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Fatalf("auth %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"m1","context_window":8192}]}`))
	}))
	defer srv.Close()
	models, err := fetchProviderModelCatalog(context.Background(), srv.URL+"/v1", "tok")
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "m1" || models[0].ContextLimit != 8192 {
		t.Fatalf("%#v", models)
	}
}

func TestModelCatalogUsesFormKeyAndURL(t *testing.T) {
	var gotAuth, gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"gpt-form"}]}`))
	}))
	defer upstream.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := updateConfiguration(cfgPath, func(fc *fileConfig) {
		fc.ModelProviders = []modelProvider{{
			ID: "or", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", APIKey: "sk-or-stored",
		}}
		fc.ActiveModelProviderID = "or"
		fc.API.Key = "sk-or-stored"
	}); err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Cfg:    Config{ConfigPath: cfgPath, Key: "sk-or-stored"},
		Client: &Client{APIKey: "sk-or-stored"},
	}
	h := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a}).Handler()

	body := `{"base_url":"` + upstream.URL + `/v1","api_key":"sk-form-key","provider_id":"or"}`
	req := httptest.NewRequest(http.MethodPost, "/api/settings/model-catalog", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog: %d %s", rec.Code, rec.Body.String())
	}
	if gotAuth != "Bearer sk-form-key" {
		t.Fatalf("upstream auth %q", gotAuth)
	}
	if gotPath != "/v1/models" {
		t.Fatalf("upstream path %q", gotPath)
	}
	if strings.Contains(rec.Body.String(), "sk-form-key") || strings.Contains(rec.Body.String(), "sk-or-stored") {
		t.Fatalf("response leaked a key: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "gpt-form") {
		t.Fatalf("models: %s", rec.Body.String())
	}
}

func TestModelCatalogBlankKeyKeepsEditedProvider(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"gpt-saved"}]}`))
	}))
	defer upstream.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := updateConfiguration(cfgPath, func(fc *fileConfig) {
		fc.ModelProviders = []modelProvider{
			{ID: "or", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", APIKey: "sk-or-stored"},
			{ID: "oa", Name: "OpenAI", BaseURL: "https://api.openai.com/v1", APIKey: "sk-saved"},
		}
		fc.ActiveModelProviderID = "or"
		fc.API.Key = "sk-or-stored"
	}); err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Cfg:    Config{ConfigPath: cfgPath, Key: "sk-or-stored"},
		Client: &Client{APIKey: "sk-or-stored"},
	}
	h := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a}).Handler()

	body := `{"base_url":"` + upstream.URL + `/v1","provider_id":"oa"}`
	req := httptest.NewRequest(http.MethodPost, "/api/settings/model-catalog", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog: %d %s", rec.Code, rec.Body.String())
	}
	if gotAuth != "Bearer sk-saved" {
		t.Fatalf("upstream auth %q", gotAuth)
	}
}

func TestModelCatalogDoesNotBorrowAnotherKey(t *testing.T) {
	called := false
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"local"}]}`))
	}))
	defer upstream.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := updateConfiguration(cfgPath, func(fc *fileConfig) {
		fc.ModelProviders = []modelProvider{{
			ID: "or", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", APIKey: "sk-or-stored",
		}}
		fc.ActiveModelProviderID = "or"
		fc.API.Key = "sk-or-stored"
	}); err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Cfg:    Config{ConfigPath: cfgPath, Key: "sk-or-stored"},
		Client: &Client{APIKey: "sk-or-stored"},
	}
	h := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a}).Handler()

	body := `{"base_url":"` + upstream.URL + `/v1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/settings/model-catalog", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog: %d %s", rec.Code, rec.Body.String())
	}
	if !called {
		t.Fatal("upstream not called")
	}
	if gotAuth != "" || strings.Contains(gotAuth, "sk-or-stored") {
		t.Fatalf("borrowed key: %q", gotAuth)
	}
}
