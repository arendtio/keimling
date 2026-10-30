package main

import (
	"testing"
)

func TestEnsureProviderModelsMigratesLegacy(t *testing.T) {
	p := modelProvider{Model: "acme/x", Reasoning: "high"}
	p.ensureProviderModels()
	if len(p.Models) != 1 || p.Models[0].ID != "acme/x" {
		t.Fatalf("models=%#v", p.Models)
	}
	if p.ActiveModelID != "acme/x" {
		t.Fatalf("active=%q", p.ActiveModelID)
	}
	if p.Model != "acme/x" || p.Reasoning != "high" {
		t.Fatalf("legacy sync: %#v", p)
	}
}

func TestEnsureProviderModelsActiveWins(t *testing.T) {
	p := modelProvider{
		Models: []providerModel{
			{ID: "a", ContextLimit: 8000},
			{ID: "b", ContextLimit: 128000, Reasoning: "low"},
		},
		ActiveModelID: "b",
		Model:         "a",
	}
	p.ensureProviderModels()
	if p.Model != "b" || p.Reasoning != "low" {
		t.Fatalf("sync: model=%q reasoning=%q", p.Model, p.Reasoning)
	}
	m := p.activeModel()
	if m == nil || m.ContextLimit != 128000 {
		t.Fatalf("active model: %#v", m)
	}
}

func TestDetectProviderKind(t *testing.T) {
	cases := []struct {
		url, kind, want string
	}{
		{"https://openrouter.ai/api/v1", "auto", "openrouter"},
		{"https://api.openai.com/v1", "auto", "openai"},
		{"http://127.0.0.1:4000/v1", "auto", "local"},
		{"https://proxy.example/v1", "litellm", "litellm"},
		{"https://openrouter.ai/api/v1", "openai", "openai"},
	}
	for _, tc := range cases {
		if got := detectProviderKind(tc.url, tc.kind); got != tc.want {
			t.Fatalf("%s/%s: got %q want %q", tc.url, tc.kind, got, tc.want)
		}
	}
}

func TestEstimateSessionUSD(t *testing.T) {
	m := &providerModel{PromptPrice: "0.001", CompletionPrice: "0.002"}
	usd, ok := estimateSessionUSD(1000, 500, m)
	if !ok || usd == nil {
		t.Fatal("expected estimate")
	}
	want := 1000*0.001 + 500*0.002
	if *usd != want {
		t.Fatalf("got %v want %v", *usd, want)
	}
	if _, ok := estimateSessionUSD(10, 10, &providerModel{}); ok {
		t.Fatal("unpriced should be false")
	}
}

func TestPublicModelProvidersIncludesModels(t *testing.T) {
	out := publicModelProviders([]modelProvider{{
		ID: "1", Name: "N", BaseURL: "https://openrouter.ai/api/v1", APIKey: "secret",
		Kind: "auto", ActiveModelID: "m1",
		Models: []providerModel{{ID: "m1", ContextLimit: 64000, PromptPrice: "0.1"}},
	}})
	if len(out) != 1 {
		t.Fatal(out)
	}
	if out[0]["api_key_set"] != true {
		t.Fatal(out[0])
	}
	models, ok := out[0]["models"].([]map[string]any)
	if !ok || len(models) != 1 || models[0]["id"] != "m1" {
		t.Fatalf("models=%#v", out[0]["models"])
	}
	if out[0]["resolved_kind"] != "openrouter" {
		t.Fatalf("resolved_kind=%v", out[0]["resolved_kind"])
	}
}
