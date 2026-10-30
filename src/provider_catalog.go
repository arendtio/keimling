package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// catalogModel is a non-persisted upstream model entry for Settings prefill.
type catalogModel struct {
	ID              string `json:"id"`
	Name            string `json:"name,omitempty"`
	ContextLimit    int    `json:"context_limit,omitempty"`
	PromptPrice     string `json:"prompt_price,omitempty"`
	CompletionPrice string `json:"completion_price,omitempty"`
}

func fetchProviderModelCatalog(ctx context.Context, baseURL, apiKey string) ([]catalogModel, error) {
	c := NewClient(apiKey)
	c.BaseURL = baseURL
	c.HTTP = &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("models"), nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model catalog: %w", err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model catalog HTTP %d: %s", res.StatusCode, truncate(string(b), 240))
	}
	return parseModelCatalog(b)
}

func parseModelCatalog(b []byte) ([]catalogModel, error) {
	var doc struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("invalid model catalog: %w", err)
	}
	if doc.Data == nil {
		return nil, fmt.Errorf("model catalog missing data array")
	}
	out := make([]catalogModel, 0, len(doc.Data))
	for _, raw := range doc.Data {
		var row struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			ContextLength int    `json:"context_length"`
			ContextWindow int    `json:"context_window"`
			MaxTokens     int    `json:"max_model_len"`
			Pricing       *struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
		}
		if err := json.Unmarshal(raw, &row); err != nil {
			continue
		}
		id := strings.TrimSpace(row.ID)
		if id == "" {
			continue
		}
		m := catalogModel{
			ID:   id,
			Name: strings.TrimSpace(row.Name),
		}
		switch {
		case row.ContextLength > 0:
			m.ContextLimit = row.ContextLength
		case row.ContextWindow > 0:
			m.ContextLimit = row.ContextWindow
		case row.MaxTokens > 0:
			m.ContextLimit = row.MaxTokens
		}
		if row.Pricing != nil {
			m.PromptPrice = strings.TrimSpace(row.Pricing.Prompt)
			m.CompletionPrice = strings.TrimSpace(row.Pricing.Completion)
		}
		out = append(out, m)
	}
	return out, nil
}

func catalogModelByID(list []catalogModel, id string) *catalogModel {
	id = strings.TrimSpace(id)
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

func parseUSDPerToken(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 {
		return 0, false
	}
	return v, true
}
