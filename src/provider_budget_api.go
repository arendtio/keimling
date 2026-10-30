package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (s *appServer) handleModelCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	providerID := strings.TrimSpace(r.URL.Query().Get("provider_id"))
	draftURL := strings.TrimSpace(r.URL.Query().Get("base_url"))
	providers, activeID := s.loadModelProvidersForSettings()
	var p *modelProvider
	if providerID != "" {
		p = modelProviderByID(providers, providerID)
	}
	baseURL := ""
	kind := "auto"
	key := ""
	id := providerID
	if p != nil {
		baseURL = p.BaseURL
		kind = p.Kind
		key = strings.TrimSpace(p.APIKey)
		id = p.ID
	} else if draftURL != "" {
		resolved, err := normalizeBaseURL(draftURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		baseURL = resolved
	} else {
		p = modelProviderByID(providers, activeID)
		if p == nil && len(providers) > 0 {
			p = &providers[0]
		}
		if p == nil {
			http.Error(w, "provider not found", http.StatusNotFound)
			return
		}
		baseURL = p.BaseURL
		kind = p.Kind
		key = strings.TrimSpace(p.APIKey)
		id = p.ID
	}
	if key == "" {
		key = strings.TrimSpace(s.agent.Cfg.Key)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	models, err := fetchProviderModelCatalog(ctx, baseURL, key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"provider_id": id,
		"kind":        detectProviderKind(baseURL, kind),
		"models":      models,
		"count":       len(models),
	})
}

func (s *appServer) handleProviderBudget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	refresh := strings.EqualFold(r.URL.Query().Get("refresh"), "1") ||
		strings.EqualFold(r.URL.Query().Get("refresh"), "true")
	snap := s.budgetSnapshot(r.Context(), refresh)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}

func (s *appServer) budgetSnapshot(ctx context.Context, refresh bool) map[string]any {
	providers, activeID := s.loadModelProvidersForSettings()
	p := modelProviderByID(providers, activeID)
	spend := sessionSpendInfo{}
	var active *providerModel
	if s.agent != nil && s.agent.Client != nil {
		spend.InputTokens = s.agent.Client.InputTokens
		spend.OutputTokens = s.agent.Client.OutputTokens
	}
	if p != nil {
		active = p.activeModel()
		if usd, ok := estimateSessionUSD(spend.InputTokens, spend.OutputTokens, active); ok {
			spend.EstimateUSD = usd
			spend.Priced = true
		}
	}
	out := map[string]any{
		"session": spend,
	}
	if p == nil {
		out["provider"] = providerBudgetInfo{Supported: false}
		return out
	}
	kind := detectProviderKind(p.BaseURL, p.Kind)
	cacheKey := p.ID + "|" + kind + "|" + p.BaseURL
	if !refresh {
		if cached, ok := s.budgetCache.get(cacheKey); ok {
			out["provider"] = cached
			return out
		}
	}
	key := strings.TrimSpace(p.APIKey)
	if key == "" && s.agent != nil {
		key = strings.TrimSpace(s.agent.Cfg.Key)
	}
	info := fetchProviderBudget(ctx, kind, p.BaseURL, key)
	if info.Supported && info.Error == "" {
		s.budgetCache.put(cacheKey, info)
	}
	out["provider"] = info
	return out
}
