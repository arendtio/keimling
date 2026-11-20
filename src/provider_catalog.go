package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
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
	SeesImages      bool   `json:"sees_images,omitempty"`
	HearsAudio      bool   `json:"hears_audio,omitempty"`
}

func fetchProviderModelCatalog(ctx context.Context, baseURL, apiKey, kind string) ([]catalogModel, error) {
	models, err := fetchOpenAIModelList(ctx, baseURL, apiKey)
	if detectProviderKind(baseURL, kind) != "litellm" {
		return models, err
	}
	info, infoErr := fetchLiteLLMModelInfo(ctx, baseURL, apiKey)
	if err != nil && (infoErr != nil || len(info) == 0) {
		return nil, err
	}
	if err != nil {
		return info, nil
	}
	if infoErr != nil || len(info) == 0 {
		return models, nil
	}
	return mergeCatalogModels(models, info), nil
}

func fetchOpenAIModelList(ctx context.Context, baseURL, apiKey string) ([]catalogModel, error) {
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
			ID            string  `json:"id"`
			Name          string  `json:"name"`
			ContextLength float64 `json:"context_length"`
			ContextWindow float64 `json:"context_window"`
			MaxTokens     float64 `json:"max_model_len"`
			Pricing       *struct {
				Prompt     json.RawMessage `json:"prompt"`
				Completion json.RawMessage `json:"completion"`
			} `json:"pricing"`
			Architecture *struct {
				Modality        string   `json:"modality"`
				InputModalities []string `json:"input_modalities"`
			} `json:"architecture"`
			InputModalities []string `json:"input_modalities"`
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
			m.ContextLimit = int(row.ContextLength)
		case row.ContextWindow > 0:
			m.ContextLimit = int(row.ContextWindow)
		case row.MaxTokens > 0:
			m.ContextLimit = int(row.MaxTokens)
		}
		if row.Pricing != nil {
			m.PromptPrice = catalogPrice(row.Pricing.Prompt)
			m.CompletionPrice = catalogPrice(row.Pricing.Completion)
		}
		inputs := row.InputModalities
		modality := ""
		if row.Architecture != nil {
			inputs = append(inputs, row.Architecture.InputModalities...)
			modality = row.Architecture.Modality
		}
		m.SeesImages, m.HearsAudio = catalogModalities(inputs, modality)
		out = append(out, m)
	}
	return out, nil
}

func catalogPrice(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	if raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) != nil {
			return ""
		}
		return strings.TrimSpace(s)
	}
	var n float64
	if json.Unmarshal(raw, &n) != nil || n < 0 || math.IsNaN(n) || math.IsInf(n, 0) {
		return ""
	}
	return strconv.FormatFloat(n, 'g', -1, 64)
}

func mergeCatalogModels(base, extra []catalogModel) []catalogModel {
	index := map[string]int{}
	for i, m := range base {
		index[m.ID] = i
	}
	for _, e := range extra {
		id := strings.TrimSpace(e.ID)
		if id == "" {
			continue
		}
		i, ok := index[id]
		if !ok {
			base = append(base, e)
			index[id] = len(base) - 1
			continue
		}
		cur := base[i]
		if cur.Name == "" {
			cur.Name = e.Name
		}
		if cur.ContextLimit == 0 {
			cur.ContextLimit = e.ContextLimit
		}
		if cur.PromptPrice == "" {
			cur.PromptPrice = e.PromptPrice
		}
		if cur.CompletionPrice == "" {
			cur.CompletionPrice = e.CompletionPrice
		}
		if e.SeesImages {
			cur.SeesImages = true
		}
		if e.HearsAudio {
			cur.HearsAudio = true
		}
		base[i] = cur
	}
	return base
}

func fetchLiteLLMModelInfo(ctx context.Context, baseURL, apiKey string) ([]catalogModel, error) {
	root := providerRootURL(baseURL)
	if root == "" {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, root+"/model/info", nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden || res.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("litellm model info HTTP %d", res.StatusCode)
	}
	return parseLiteLLMModelInfo(b)
}

func parseLiteLLMModelInfo(b []byte) ([]catalogModel, error) {
	var doc struct {
		Data []struct {
			ModelName string `json:"model_name"`
			ModelInfo struct {
				MaxInputTokens int      `json:"max_input_tokens"`
				InputCost      *float64 `json:"input_cost_per_token"`
				OutputCost     *float64 `json:"output_cost_per_token"`
				SupportsVision bool     `json:"supports_vision"`
				SupportsAudio  bool     `json:"supports_audio_input"`
			} `json:"model_info"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	out := make([]catalogModel, 0, len(doc.Data))
	for _, row := range doc.Data {
		id := strings.TrimSpace(row.ModelName)
		if id == "" {
			continue
		}
		m := catalogModel{ID: id, SeesImages: row.ModelInfo.SupportsVision, HearsAudio: row.ModelInfo.SupportsAudio}
		if row.ModelInfo.MaxInputTokens > 0 {
			m.ContextLimit = row.ModelInfo.MaxInputTokens
		}
		if row.ModelInfo.InputCost != nil && *row.ModelInfo.InputCost >= 0 {
			m.PromptPrice = strconv.FormatFloat(*row.ModelInfo.InputCost, 'g', -1, 64)
		}
		if row.ModelInfo.OutputCost != nil && *row.ModelInfo.OutputCost >= 0 {
			m.CompletionPrice = strconv.FormatFloat(*row.ModelInfo.OutputCost, 'g', -1, 64)
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
