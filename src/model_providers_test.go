package main

import (
	"path/filepath"
	"testing"
)

func TestEnsureModelProvidersFromLegacyAPI(t *testing.T) {
	fc := fileConfig{}
	fc.API.BaseURL = "https://openrouter.ai/api/v1"
	fc.API.Key = "sk-test"
	fc.API.Model = "x"
	fc.ensureModelProviders()
	if len(fc.ModelProviders) != 1 {
		t.Fatalf("want 1 provider, got %#v", fc.ModelProviders)
	}
	p := fc.ModelProviders[0]
	if p.BaseURL != "https://openrouter.ai/api/v1" || p.APIKey != "sk-test" || p.Model != "x" {
		t.Fatalf("provider fields: %#v", p)
	}
	if fc.ActiveModelProviderID != p.ID {
		t.Fatalf("active=%q want %q", fc.ActiveModelProviderID, p.ID)
	}
	if p.Name != "OpenRouter" {
		t.Fatalf("name=%q", p.Name)
	}
	if len(p.Models) != 1 || p.Models[0].ID != "x" || p.ActiveModelID != "x" {
		t.Fatalf("migrated models: %#v active=%q", p.Models, p.ActiveModelID)
	}
	if p.Kind != "auto" {
		t.Fatalf("kind=%q", p.Kind)
	}
}

func TestMirrorActiveModelProvider(t *testing.T) {
	fc := fileConfig{
		ModelProviders: []modelProvider{
			{ID: "a", Name: "A", BaseURL: "https://a.example/v1", APIKey: "ka", Model: "ma"},
			{ID: "b", Name: "B", BaseURL: "https://b.example/v1", APIKey: "kb", Model: "mb"},
		},
		ActiveModelProviderID: "b",
	}
	fc.API.BaseURL = "https://old"
	fc.API.Key = "old"
	fc.API.Model = "old"
	fc.ensureModelProviders()
	if fc.API.BaseURL != "https://b.example/v1" || fc.API.Key != "kb" || fc.API.Model != "mb" {
		t.Fatalf("mirror: %#v", fc.API)
	}
}

func TestPublicModelProvidersHidesKeys(t *testing.T) {
	out := publicModelProviders([]modelProvider{
		{ID: "1", Name: "N", BaseURL: "https://x", APIKey: "secret", Model: "m", Reasoning: "high"},
	})
	if len(out) != 1 {
		t.Fatal(out)
	}
	if _, ok := out[0]["api_key"]; ok {
		t.Fatalf("api_key leaked: %#v", out[0])
	}
	if out[0]["api_key_set"] != true {
		t.Fatalf("api_key_set: %#v", out[0])
	}
}

func TestPersistModelProvidersRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := updateConfiguration(cfgPath, func(fc *fileConfig) {
		fc.API.BaseURL = "https://api.openai.com/v1"
		fc.API.Key = "k1"
		fc.API.Model = "gpt"
	}); err != nil {
		t.Fatal(err)
	}
	fc, err := readConfiguration(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	fc.ensureModelProviders()
	id := fc.ActiveModelProviderID
	fc.ModelProviders = append(fc.ModelProviders, modelProvider{
		ID: "p2", Name: "Local", BaseURL: "http://127.0.0.1:8080/v1", Model: "local", Reasoning: "low",
	})
	fc.ActiveModelProviderID = "p2"
	fc.ensureModelProviders()
	if err := updateConfiguration(cfgPath, func(dst *fileConfig) {
		*dst = fc
	}); err != nil {
		t.Fatal(err)
	}
	got, err := readConfiguration(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	got.ensureModelProviders()
	if got.ActiveModelProviderID != "p2" {
		t.Fatalf("active=%q (seed was %q)", got.ActiveModelProviderID, id)
	}
	if got.API.BaseURL != "http://127.0.0.1:8080/v1" || got.API.Model != "local" {
		t.Fatalf("legacy mirror: %#v", got.API)
	}
}
