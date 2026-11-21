package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFetchOpenRouterKeyBudget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/key" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"limit":           100.0,
				"limit_remaining": 42.5,
				"usage":           57.5,
			},
		})
	}))
	defer srv.Close()
	rem, lim, usage, err := fetchOpenRouterKeyBudget(context.Background(), srv.URL+"/api/v1", "k")
	if err != nil {
		t.Fatal(err)
	}
	if rem == nil || *rem != 42.5 || lim == nil || *lim != 100 || usage == nil || *usage != 57.5 {
		t.Fatalf("%v %v %v", rem, lim, usage)
	}
}

func TestFetchOpenRouterAccountAndKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/key":
			if r.Header.Get("Authorization") != "Bearer chat" {
				t.Fatalf("key auth %s", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"limit": 10.0, "limit_remaining": 4.0, "usage": 6.0},
			})
		case "/api/v1/credits":
			if r.Header.Get("Authorization") != "Bearer mgmt" {
				t.Fatalf("credits auth %s", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"total_credits": 100.5, "total_usage": 25.75},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	info := fetchProviderBudget(context.Background(), "openrouter", srv.URL+"/api/v1", "chat", "mgmt")
	if info.Error != "" || info.RemainingUSD == nil || *info.RemainingUSD != 4 {
		t.Fatalf("%+v", info)
	}
	if info.AccountRemainingUSD == nil || *info.AccountRemainingUSD != 74.75 {
		t.Fatalf("account %+v", info)
	}
}

func TestOpenRouterAccountLoadsWhenTheKeyCallFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/key":
			http.Error(w, "no", http.StatusForbidden)
		case "/api/v1/credits":
			if r.Header.Get("Authorization") != "Bearer mgmt" {
				t.Fatalf("credits auth %s", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"total_credits": 20.0, "total_usage": 5.0},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	info := fetchProviderBudget(context.Background(), "openrouter", srv.URL+"/api/v1", "chat", "mgmt")
	if info.Error == "" || info.AccountRemainingUSD == nil || *info.AccountRemainingUSD != 15 {
		t.Fatalf("%+v", info)
	}
}

func TestBudgetFollowsTheChatModel(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// A real fetch also appends one spend-ledger budget snapshot; keep it out
	// of the developer's config directory.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/key":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"limit": nil, "limit_remaining": nil, "usage": 1.0},
			})
		case "/api/v1/credits":
			if r.Header.Get("Authorization") != "Bearer mgmt" {
				t.Fatalf("credits auth %s", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"total_credits": 20.0, "total_usage": 5.0},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer up.Close()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	writeModelConfig(t, cfgPath, fileConfig{
		ActiveModelProviderID: "oa",
		ModelProviders: []modelProvider{
			{
				ID: "oa", Name: "OpenAI", BaseURL: "https://api.openai.com/v1", Kind: "openai",
				APIKey: "oa", ActiveModelID: "gpt",
				Models: []providerModel{{ID: "gpt", Name: "GPT"}},
			},
			{
				ID: "or", Name: "OpenRouter", BaseURL: up.URL + "/api/v1", Kind: "openrouter",
				APIKey: "chat", ManagementKey: "mgmt", ActiveModelID: "other",
				Models: []providerModel{
					{ID: "other", Name: "Other"},
					{ID: "deepseek", Name: "DeepSeek", PromptPrice: "0.000002", CompletionPrice: "0.000008"},
				},
			},
		},
	})
	const wsID, thID = "ws-budget", "th-budget"
	if err := writeThreadMeta(wsID, ThreadMeta{ID: thID, ModelProviderID: "or", ModelID: "deepseek"}); err != nil {
		t.Fatal(err)
	}
	s := &appServer{agent: &Agent{
		Cfg:    Config{ConfigPath: cfgPath, WorkspaceID: wsID, ThreadID: thID},
		Client: &Client{InputTokens: 1000},
	}}
	snap := s.budgetSnapshot(context.Background(), true)
	spend := snap["session"].(sessionSpendInfo)
	if !spend.Priced || spend.EstimateUSD == nil || *spend.EstimateUSD != 0.002 {
		t.Fatalf("session %+v", spend)
	}
	info := snap["provider"].(providerBudgetInfo)
	if !info.Supported || info.AccountRemainingUSD == nil || *info.AccountRemainingUSD != 15 {
		t.Fatalf("provider %+v", info)
	}
}

func TestBlankManagementKeyKeepsStored(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := updateConfiguration(cfgPath, func(fc *fileConfig) {
		fc.ModelProviders = []modelProvider{{
			ID: "p", Name: "OR", BaseURL: "https://openrouter.ai/api/v1",
			APIKey: "chat", ManagementKey: "mgmt", Kind: "openrouter",
			Models: []providerModel{{ID: "m"}}, ActiveModelID: "m",
		}}
		fc.ActiveModelProviderID = "p"
	}); err != nil {
		t.Fatal(err)
	}
	s := &appServer{agent: &Agent{Cfg: Config{ConfigPath: cfgPath}}}
	out, _, err := s.resolveModelProviders(agentSettings{
		ModelProviders: []modelProvider{{
			ID: "p", Name: "OR", BaseURL: "https://openrouter.ai/api/v1",
			Kind: "openrouter", Models: []providerModel{{ID: "m"}}, ActiveModelID: "m",
		}},
		ActiveModelProviderID: "p",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out[0].ManagementKey != "mgmt" || out[0].APIKey != "chat" {
		t.Fatalf("%+v", out[0])
	}
	pub, _ := json.Marshal(publicModelProviders(out))
	if strings.Contains(string(pub), "mgmt") || strings.Contains(string(pub), "chat") {
		t.Fatalf("leaked %s", pub)
	}
	if !strings.Contains(string(pub), `"management_key_set":true`) {
		t.Fatalf("%s", pub)
	}
}

func TestFetchLiteLLMKeyBudget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key/info" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Fatalf("key was put in the URL: %s", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Fatalf("auth %s", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"info": map[string]any{"max_budget": 50.0, "spend": 12.5},
		})
	}))
	defer srv.Close()
	rem, lim, usage, err := fetchLiteLLMKeyBudget(context.Background(), srv.URL+"/v1", "k")
	if err != nil {
		t.Fatal(err)
	}
	if rem == nil || *rem != 37.5 || lim == nil || *lim != 50 || usage == nil || *usage != 12.5 {
		t.Fatalf("%v %v %v", rem, lim, usage)
	}
}

func TestProviderBudgetCacheTTL(t *testing.T) {
	var c providerBudgetCache
	info := providerBudgetInfo{Supported: true, Kind: "openrouter"}
	c.put("k", info)
	got, ok := c.get("k")
	if !ok || !got.CacheHit {
		t.Fatal("expected cache hit")
	}
	c.mu.Lock()
	c.fetchedAt = time.Now().Add(-2 * providerBudgetTTL)
	c.mu.Unlock()
	if _, ok := c.get("k"); ok {
		t.Fatal("expected miss after TTL")
	}
}

func TestHandleProviderBudgetUnsupported(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := updateConfiguration(cfgPath, func(fc *fileConfig) {
		fc.API.BaseURL = "https://api.openai.com/v1"
		fc.API.Key = "k"
		fc.API.Model = "gpt"
		fc.ModelProviders = []modelProvider{{
			ID: "p1", Name: "OpenAI", BaseURL: "https://api.openai.com/v1", APIKey: "k",
			Kind: "openai", Model: "gpt", ActiveModelID: "gpt",
			Models: []providerModel{{ID: "gpt", PromptPrice: "0.001", CompletionPrice: "0.002"}},
		}}
		fc.ActiveModelProviderID = "p1"
	}); err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Cfg:    Config{ConfigPath: cfgPath, Model: "gpt", Key: "k", BaseURL: "https://api.openai.com/v1"},
		Client: &Client{InputTokens: 1000, OutputTokens: 100},
	}
	srv := newAppServer(appServerOpts{AccessSecret: "s", Agent: a})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/settings/budget", nil)
	req.Header.Set("Authorization", "Bearer s")
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	sess := body["session"].(map[string]any)
	if sess["priced"] != true {
		t.Fatalf("session=%#v", sess)
	}
	prov := body["provider"].(map[string]any)
	if prov["supported"] != false {
		t.Fatalf("provider=%#v", prov)
	}
}
