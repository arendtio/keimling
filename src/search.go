package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const braveSearchURL = "https://api.search.brave.com/res/v1/web/search"

// searchConfig is the web-search service. It is not a model provider.
// Kind is "", "brave", "url", or "openai".
type searchConfig struct {
	Kind   string `json:"kind,omitempty"`
	URL    string `json:"url,omitempty"`
	APIKey string `json:"api_key,omitempty"`
}

func (s searchConfig) enabled() bool {
	switch strings.TrimSpace(s.Kind) {
	case "brave", "openai":
		return strings.TrimSpace(s.APIKey) != ""
	case "url":
		return strings.TrimSpace(s.URL) != ""
	default:
		return false
	}
}

func (a *Agent) searchWeb(ctx context.Context, query string) (string, error) {
	if a == nil || a.Cfg.DisableExternal || !a.Cfg.Web || !a.Cfg.Search.enabled() {
		return "", fmt.Errorf("web search is off")
	}
	return runSearch(ctx, a.Cfg.Search, query)
}

func runSearch(ctx context.Context, s searchConfig, query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("search query is empty")
	}
	if !s.enabled() {
		return "", fmt.Errorf("web search is off")
	}
	if s.Kind == "openai" {
		c := NewClient(s.APIKey)
		if u := strings.TrimSpace(s.URL); u != "" {
			nu, err := normalizeBaseURL(u)
			if err != nil {
				return "", err
			}
			c.BaseURL = nu
		}
		return c.SearchWeb(ctx, query)
	}
	endpoint, err := searchEndpoint(s, query)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json, text/plain;q=0.5")
	switch s.Kind {
	case "brave":
		req.Header.Set("X-Subscription-Token", strings.TrimSpace(s.APIKey))
	default:
		if k := strings.TrimSpace(s.APIKey); k != "" {
			req.Header.Set("Authorization", "Bearer "+k)
		}
	}
	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("search HTTP %d", res.StatusCode)
	}
	return formatSearchBody(data), nil
}

func searchEndpoint(s searchConfig, query string) (string, error) {
	raw := strings.TrimSpace(s.URL)
	if s.Kind == "brave" && raw == "" {
		raw = braveSearchURL
	}
	if strings.Contains(raw, "{query}") {
		return strings.ReplaceAll(raw, "{query}", url.QueryEscape(query)), nil
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("search URL is invalid")
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func formatSearchBody(data []byte) string {
	var body map[string]any
	if json.Unmarshal(data, &body) == nil {
		if text := formatSearchResults(searchResultList(body)); text != "" {
			return text
		}
	}
	text := strings.TrimSpace(string(data))
	if len(text) > 4000 {
		text = text[:4000]
	}
	return text
}

func searchResultList(body map[string]any) []any {
	if web, ok := body["web"].(map[string]any); ok {
		if list, ok := web["results"].([]any); ok {
			return list
		}
	}
	for _, key := range []string{"results", "items"} {
		if list, ok := body[key].([]any); ok {
			return list
		}
	}
	return nil
}

func formatSearchResults(list []any) string {
	var b strings.Builder
	n := 0
	for _, item := range list {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := searchField(row, "title")
		link := searchField(row, "url")
		snip := searchField(row, "description", "content", "snippet")
		if title == "" && link == "" && snip == "" {
			continue
		}
		if n > 0 {
			b.WriteString("\n\n")
		}
		if title != "" {
			b.WriteString(title)
			b.WriteByte('\n')
		}
		if link != "" {
			b.WriteString(link)
			b.WriteByte('\n')
		}
		if snip != "" {
			b.WriteString(snip)
		}
		n++
		if n >= 8 {
			break
		}
	}
	return strings.TrimSpace(b.String())
}

func searchField(row map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := row[k].(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func firstOpenAIProvider(fc *fileConfig) *modelProvider {
	if fc == nil {
		return nil
	}
	for i := range fc.ModelProviders {
		p := &fc.ModelProviders[i]
		if p.Kind == "openai" || strings.Contains(p.BaseURL, "api.openai.com") {
			return p
		}
	}
	return nil
}

// migrateServiceChoices keeps an existing OpenAI web search and a saved voice
// model on the first start, then remembers that the choice was made.
func migrateServiceChoices(fc *fileConfig) bool {
	if fc == nil || fc.ServicesChosen {
		return false
	}
	fc.ServicesChosen = true
	if strings.TrimSpace(fc.VoiceProviderID) == "" && strings.TrimSpace(fc.VoiceModel) != "" {
		if p := firstOpenAIProvider(fc); p != nil {
			fc.VoiceProviderID = p.ID
		}
	}
	webOn := fc.Features.WebSearch == nil || *fc.Features.WebSearch
	if strings.TrimSpace(fc.Search.Kind) == "" && webOn {
		key := strings.TrimSpace(fc.OpenAI.Key)
		if key == "" {
			if p := firstOpenAIProvider(fc); p != nil {
				key = strings.TrimSpace(p.APIKey)
			}
		}
		if key != "" {
			fc.Search.Kind = "openai"
			fc.Search.APIKey = key
		}
	}
	return true
}

func prepareStartupServices(path string, fc *fileConfig, credKey string) error {
	if fc == nil {
		return nil
	}
	seeded := seedOpenAIProvider(fc, credKey)
	before := len(fc.ModelProviders)
	fc.ensureModelProviders()
	migrated := migrateServiceChoices(fc)
	if path == "" || (!seeded && !migrated && len(fc.ModelProviders) == before) {
		return nil
	}
	return updateConfiguration(path, func(dst *fileConfig) { *dst = *fc })
}

func (a *Agent) voiceClient() (*Client, string, error) {
	if a == nil || a.Cfg.DisableExternal {
		return nil, "", fmt.Errorf("voice transcription is off")
	}
	id := strings.TrimSpace(a.Cfg.VoiceProviderID)
	model := strings.TrimSpace(a.Cfg.VoiceModel)
	if id == "" || model == "" {
		return nil, "", fmt.Errorf("voice transcription is off")
	}
	fc, err := readConfiguration(a.Cfg.ConfigPath)
	if err != nil {
		return nil, "", err
	}
	fc.ensureModelProviders()
	p := modelProviderByID(fc.ModelProviders, id)
	if p == nil {
		return nil, "", fmt.Errorf("voice provider is not configured")
	}
	if strings.TrimSpace(p.BaseURL) == "" {
		return nil, "", fmt.Errorf("voice provider has no URL")
	}
	c := NewClient(p.APIKey)
	c.BaseURL = p.BaseURL
	return c, model, nil
}
