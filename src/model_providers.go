package main

import (
	"fmt"
	"strings"
	"time"
)

// modelProvider is a saved LLM endpoint profile in config.json.
type modelProvider struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	BaseURL       string          `json:"base_url"`
	APIKey        string          `json:"api_key,omitempty"`
	Kind          string          `json:"kind,omitempty"` // auto|openai|openrouter|litellm|local
	Models        []providerModel `json:"models,omitempty"`
	ActiveModelID string          `json:"active_model_id,omitempty"`
	// Legacy single-model fields; mirrored from ActiveModelID for older clients.
	Model     string `json:"model,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
}

func newModelProviderID() string {
	return fmt.Sprintf("provider-%s", time.Now().UTC().Format("20060102-150405.000000000"))
}

func (fc *fileConfig) ensureModelProviders() {
	if fc.ModelProviders == nil {
		fc.ModelProviders = []modelProvider{}
	}
	if len(fc.ModelProviders) == 0 {
		base := strings.TrimSpace(fc.API.BaseURL)
		if base == "" {
			base = defaultBaseURL
		}
		name := "Default"
		if strings.Contains(base, "openrouter") {
			name = "OpenRouter"
		} else if strings.Contains(base, "openai.com") {
			name = "OpenAI"
		}
		id := newModelProviderID()
		fc.ModelProviders = []modelProvider{{
			ID:        id,
			Name:      name,
			BaseURL:   base,
			APIKey:    strings.TrimSpace(fc.API.Key),
			Kind:      "auto",
			Model:     strings.TrimSpace(fc.API.Model),
			Reasoning: "",
		}}
		fc.ActiveModelProviderID = id
	}
	seen := map[string]bool{}
	for i := range fc.ModelProviders {
		p := &fc.ModelProviders[i]
		p.ID = strings.TrimSpace(p.ID)
		p.Name = strings.TrimSpace(p.Name)
		p.BaseURL = strings.TrimSpace(p.BaseURL)
		p.APIKey = strings.TrimSpace(p.APIKey)
		p.Kind = strings.TrimSpace(p.Kind)
		p.Model = strings.TrimSpace(p.Model)
		p.Reasoning = strings.TrimSpace(p.Reasoning)
		p.ActiveModelID = strings.TrimSpace(p.ActiveModelID)
		if p.ID == "" || seen[p.ID] {
			p.ID = newModelProviderID()
		}
		seen[p.ID] = true
		if p.Name == "" {
			p.Name = p.BaseURL
			if p.Name == "" {
				p.Name = "Provider"
			}
		}
		if p.Kind == "" {
			p.Kind = "auto"
		}
		p.ensureProviderModels()
	}
	active := strings.TrimSpace(fc.ActiveModelProviderID)
	if active == "" || modelProviderByID(fc.ModelProviders, active) == nil {
		fc.ActiveModelProviderID = fc.ModelProviders[0].ID
	}
	fc.mirrorActiveModelProvider()
}

func modelProviderByID(list []modelProvider, id string) *modelProvider {
	id = strings.TrimSpace(id)
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

func (fc *fileConfig) mirrorActiveModelProvider() {
	p := modelProviderByID(fc.ModelProviders, fc.ActiveModelProviderID)
	if p == nil {
		return
	}
	p.ensureProviderModels()
	if u := strings.TrimSpace(p.BaseURL); u != "" {
		fc.API.BaseURL = u
	}
	if m := strings.TrimSpace(p.Model); m != "" {
		fc.API.Model = m
	}
	if k := strings.TrimSpace(p.APIKey); k != "" {
		fc.API.Key = k
		fc.API.KeyFile = ""
	}
}

func publicModelProviders(list []modelProvider) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, p := range list {
		p.ensureProviderModels()
		models := make([]map[string]any, 0, len(p.Models))
		for _, m := range p.Models {
			models = append(models, map[string]any{
				"id":               m.ID,
				"name":             m.Name,
				"context_limit":    m.ContextLimit,
				"reasoning":        m.Reasoning,
				"prompt_price":     m.PromptPrice,
				"completion_price": m.CompletionPrice,
			})
		}
		out = append(out, map[string]any{
			"id":              p.ID,
			"name":            p.Name,
			"base_url":        p.BaseURL,
			"kind":            p.Kind,
			"resolved_kind":   detectProviderKind(p.BaseURL, p.Kind),
			"model":           p.Model,
			"reasoning":       p.Reasoning,
			"active_model_id": p.ActiveModelID,
			"models":          models,
			"api_key_set":     strings.TrimSpace(p.APIKey) != "",
		})
	}
	return out
}

// applyActiveProviderToAgent copies active provider + model into live Config.
func applyActiveProviderToAgent(a *Agent, fc fileConfig) {
	if a == nil {
		return
	}
	p := modelProviderByID(fc.ModelProviders, fc.ActiveModelProviderID)
	if p == nil {
		return
	}
	p.ensureProviderModels()
	if u := strings.TrimSpace(p.BaseURL); u != "" {
		a.Cfg.BaseURL = u
		if a.Client != nil {
			a.Client.BaseURL = u
		}
	}
	if k := strings.TrimSpace(p.APIKey); k != "" {
		a.Cfg.Key = k
		if a.Client != nil {
			a.Client.APIKey = k
		}
	}
	if m := p.activeModel(); m != nil {
		a.Cfg.Model = m.ID
		if r := strings.TrimSpace(m.Reasoning); r != "" {
			a.Cfg.Reasoning = r
		} else if r := strings.TrimSpace(p.Reasoning); r != "" {
			a.Cfg.Reasoning = r
		}
		if m.ContextLimit >= 1000 {
			a.Cfg.ContextLimit = m.ContextLimit
		}
	} else if model := strings.TrimSpace(p.Model); model != "" {
		a.Cfg.Model = model
	}
}
