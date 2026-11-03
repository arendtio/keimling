package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *appServer) handleModelCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		BaseURL    string `json:"base_url"`
		APIKey     string `json:"api_key"`
		ProviderID string `json:"provider_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	baseURL := strings.TrimSpace(body.BaseURL)
	if baseURL == "" {
		http.Error(w, "base URL required", http.StatusBadRequest)
		return
	}
	resolved, err := normalizeBaseURL(baseURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	key := strings.TrimSpace(body.APIKey)
	providerID := strings.TrimSpace(body.ProviderID)
	kind := "auto"
	if key == "" && providerID != "" {
		providers, _ := s.loadModelProvidersForSettings()
		if p := modelProviderByID(providers, providerID); p != nil {
			key = strings.TrimSpace(p.APIKey)
			kind = p.Kind
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	models, err := fetchProviderModelCatalog(ctx, resolved, key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"provider_id": providerID,
		"kind":        detectProviderKind(resolved, kind),
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
