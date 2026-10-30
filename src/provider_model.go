package main

import (
	"errors"
	"strings"
)

// providerModel is a manually curated model favorite on a provider.
type providerModel struct {
	ID              string `json:"id"`
	Name            string `json:"name,omitempty"`
	ContextLimit    int    `json:"context_limit,omitempty"`
	Reasoning       string `json:"reasoning,omitempty"`
	PromptPrice     string `json:"prompt_price,omitempty"`     // USD per input token (OpenRouter-style string)
	CompletionPrice string `json:"completion_price,omitempty"` // USD per output token
}

func (m providerModel) displayName() string {
	if n := strings.TrimSpace(m.Name); n != "" {
		return n
	}
	return strings.TrimSpace(m.ID)
}

func normalizeProviderModel(m providerModel) (providerModel, error) {
	m.ID = strings.TrimSpace(m.ID)
	m.Name = strings.TrimSpace(m.Name)
	m.Reasoning = strings.TrimSpace(m.Reasoning)
	m.PromptPrice = strings.TrimSpace(m.PromptPrice)
	m.CompletionPrice = strings.TrimSpace(m.CompletionPrice)
	if m.ID == "" {
		return m, errors.New("model id required")
	}
	if m.ContextLimit < 0 {
		m.ContextLimit = 0
	}
	if m.Reasoning != "" {
		switch m.Reasoning {
		case "low", "medium", "high", "xhigh", "max":
		default:
			return m, errors.New("invalid model reasoning")
		}
	}
	return m, nil
}

func providerModelByID(list []providerModel, id string) *providerModel {
	id = strings.TrimSpace(id)
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

// ensureProviderModels migrates legacy provider.Model into models[] when needed.
func (p *modelProvider) ensureProviderModels() {
	if p == nil {
		return
	}
	if p.Models == nil {
		p.Models = []providerModel{}
	}
	for i := range p.Models {
		if m, err := normalizeProviderModel(p.Models[i]); err == nil {
			p.Models[i] = m
		}
	}
	// Drop empty ids
	cleaned := p.Models[:0]
	for _, m := range p.Models {
		if strings.TrimSpace(m.ID) != "" {
			cleaned = append(cleaned, m)
		}
	}
	p.Models = cleaned

	legacy := strings.TrimSpace(p.Model)
	if len(p.Models) == 0 && legacy != "" {
		p.Models = []providerModel{{
			ID:        legacy,
			Reasoning: strings.TrimSpace(p.Reasoning),
		}}
	}
	active := strings.TrimSpace(p.ActiveModelID)
	if active == "" || providerModelByID(p.Models, active) == nil {
		if legacy != "" && providerModelByID(p.Models, legacy) != nil {
			p.ActiveModelID = legacy
		} else if len(p.Models) > 0 {
			p.ActiveModelID = p.Models[0].ID
		} else {
			p.ActiveModelID = ""
		}
	}
	p.syncLegacyModelFields()
}

func (p *modelProvider) syncLegacyModelFields() {
	if p == nil {
		return
	}
	m := providerModelByID(p.Models, p.ActiveModelID)
	if m == nil {
		return
	}
	p.Model = m.ID
	if r := strings.TrimSpace(m.Reasoning); r != "" {
		p.Reasoning = r
	}
}

func (p *modelProvider) activeModel() *providerModel {
	if p == nil {
		return nil
	}
	p.ensureProviderModels()
	return providerModelByID(p.Models, p.ActiveModelID)
}

func detectProviderKind(baseURL, kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	switch k {
	case "openai", "openrouter", "litellm", "local", "auto":
		if k != "auto" {
			return k
		}
	}
	u := strings.ToLower(baseURL)
	switch {
	case strings.Contains(u, "openrouter"):
		return "openrouter"
	case strings.Contains(u, "litellm"):
		return "litellm"
	case strings.Contains(u, "127.0.0.1") || strings.Contains(u, "localhost") || strings.Contains(u, "::1"):
		return "local"
	default:
		return "openai"
	}
}
