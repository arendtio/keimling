package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const providerBudgetTTL = 90 * time.Second

type providerBudgetInfo struct {
	Kind         string   `json:"kind"`
	Supported    bool     `json:"supported"`
	RemainingUSD *float64 `json:"remaining_usd,omitempty"`
	LimitUSD     *float64 `json:"limit_usd,omitempty"`
	UsageUSD     *float64 `json:"usage_usd,omitempty"`
	FetchedAt    string   `json:"fetched_at,omitempty"`
	Error        string   `json:"error,omitempty"`
	CacheHit     bool     `json:"cache_hit,omitempty"`
}

type sessionSpendInfo struct {
	InputTokens  int64    `json:"input_tokens"`
	OutputTokens int64    `json:"output_tokens"`
	EstimateUSD  *float64 `json:"estimate_usd,omitempty"`
	Priced       bool     `json:"priced"`
}

type providerBudgetCache struct {
	mu        sync.Mutex
	key       string
	info      providerBudgetInfo
	fetchedAt time.Time
}

func (c *providerBudgetCache) get(key string) (providerBudgetInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if key == "" || c.key != key || c.fetchedAt.IsZero() {
		return providerBudgetInfo{}, false
	}
	if time.Since(c.fetchedAt) > providerBudgetTTL {
		return providerBudgetInfo{}, false
	}
	out := c.info
	out.CacheHit = true
	return out, true
}

func (c *providerBudgetCache) put(key string, info providerBudgetInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.key = key
	c.info = info
	c.fetchedAt = time.Now()
}

func (c *providerBudgetCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.key = ""
	c.info = providerBudgetInfo{}
	c.fetchedAt = time.Time{}
}

func estimateSessionUSD(inputTokens, outputTokens int64, m *providerModel) (*float64, bool) {
	if m == nil {
		return nil, false
	}
	pin, okIn := parseUSDPerToken(m.PromptPrice)
	pout, okOut := parseUSDPerToken(m.CompletionPrice)
	if !okIn && !okOut {
		return nil, false
	}
	var usd float64
	if okIn {
		usd += float64(inputTokens) * pin
	}
	if okOut {
		usd += float64(outputTokens) * pout
	}
	return &usd, true
}

func fetchProviderBudget(ctx context.Context, kind, baseURL, apiKey string) providerBudgetInfo {
	info := providerBudgetInfo{Kind: kind}
	switch kind {
	case "openrouter":
		info.Supported = true
		rem, lim, usage, err := fetchOpenRouterKeyBudget(ctx, baseURL, apiKey)
		if err != nil {
			info.Error = err.Error()
			return info
		}
		info.RemainingUSD = rem
		info.LimitUSD = lim
		info.UsageUSD = usage
		info.FetchedAt = time.Now().UTC().Format(time.RFC3339)
		return info
	case "litellm":
		info.Supported = true
		rem, lim, usage, err := fetchLiteLLMUserBudget(ctx, baseURL, apiKey)
		if err != nil {
			info.Error = err.Error()
			return info
		}
		info.RemainingUSD = rem
		info.LimitUSD = lim
		info.UsageUSD = usage
		info.FetchedAt = time.Now().UTC().Format(time.RFC3339)
		return info
	default:
		return info
	}
}

func providerRootURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	base = strings.TrimSuffix(base, "/v1")
	base = strings.TrimSuffix(base, "/api")
	return strings.TrimRight(base, "/")
}

func fetchOpenRouterKeyBudget(ctx context.Context, baseURL, apiKey string) (remaining, limit, usage *float64, err error) {
	root := providerRootURL(baseURL)
	if root == "" {
		root = "https://openrouter.ai"
	}
	url := root + "/api/v1/key"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 12 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, nil, nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, nil, nil, fmt.Errorf("openrouter key HTTP %d: %s", res.StatusCode, truncate(string(b), 200))
	}
	var doc struct {
		Data struct {
			Limit          *float64 `json:"limit"`
			LimitRemaining *float64 `json:"limit_remaining"`
			Usage          *float64 `json:"usage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, nil, nil, fmt.Errorf("openrouter key parse: %w", err)
	}
	return doc.Data.LimitRemaining, doc.Data.Limit, doc.Data.Usage, nil
}

func fetchLiteLLMUserBudget(ctx context.Context, baseURL, apiKey string) (remaining, limit, usage *float64, err error) {
	root := providerRootURL(baseURL)
	if root == "" {
		return nil, nil, nil, fmt.Errorf("litellm base url required")
	}
	url := root + "/user/info"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 12 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, nil, nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, nil, nil, fmt.Errorf("litellm user/info HTTP %d: %s", res.StatusCode, truncate(string(b), 200))
	}
	var doc struct {
		UserInfo *struct {
			MaxBudget *float64 `json:"max_budget"`
			Spend     *float64 `json:"spend"`
		} `json:"user_info"`
		MaxBudget *float64 `json:"max_budget"`
		Spend     *float64 `json:"spend"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, nil, nil, fmt.Errorf("litellm user/info parse: %w", err)
	}
	var maxB, spend *float64
	if doc.UserInfo != nil {
		maxB, spend = doc.UserInfo.MaxBudget, doc.UserInfo.Spend
	} else {
		maxB, spend = doc.MaxBudget, doc.Spend
	}
	if maxB == nil {
		return nil, nil, spend, nil
	}
	s := 0.0
	if spend != nil {
		s = *spend
	}
	rem := *maxB - s
	if rem < 0 {
		rem = 0
	}
	return &rem, maxB, spend, nil
}
