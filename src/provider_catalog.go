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
	root, probe := liteLLMCatalogRoot(baseURL, kind)
	if !probe {
		return models, err
	}
	// A LiteLLM proxy publishes per-model metadata on two management routes:
	// GET /model_group/info lists the model groups this key may call, and
	// GET /model/info lists the deployments with their model_info. Both are
	// optional: a plain OpenAI-compatible server answers 404 and the catalog
	// stays the /v1/models list.
	groups, _ := fetchLiteLLMModelGroupInfo(ctx, root, apiKey)
	info, _ := fetchLiteLLMModelInfo(ctx, root, apiKey)
	extra := mergeCatalogModels(groups, info)
	if err != nil {
		if len(extra) > 0 {
			return extra, nil
		}
		return nil, err
	}
	return mergeCatalogModels(models, extra), nil
}

// liteLLMCatalogRoot reports the proxy root to ask for LiteLLM model metadata.
// OpenRouter is excluded so its catalog stays the /v1/models response alone.
func liteLLMCatalogRoot(baseURL, kind string) (string, bool) {
	if detectProviderKind(baseURL, kind) == "openrouter" {
		return "", false
	}
	root := providerRootURL(baseURL)
	if root == "" {
		return "", false
	}
	return root, true
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
			ID             string  `json:"id"`
			Name           string  `json:"name"`
			ContextLength  float64 `json:"context_length"`
			ContextWindow  float64 `json:"context_window"`
			MaxInputTokens float64 `json:"max_input_tokens"`
			MaxTokens      float64 `json:"max_model_len"`
			Pricing        *struct {
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
		case row.MaxInputTokens > 0:
			// LiteLLM /v1/models reports the group's input limit here.
			m.ContextLimit = int(row.MaxInputTokens)
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

func fetchLiteLLMModelInfo(ctx context.Context, root, apiKey string) ([]catalogModel, error) {
	b, err := getLiteLLMJSON(ctx, root, "/model/info", apiKey)
	if err != nil || len(b) == 0 {
		return nil, err
	}
	return parseLiteLLMModelInfo(b)
}

func fetchLiteLLMModelGroupInfo(ctx context.Context, root, apiKey string) ([]catalogModel, error) {
	b, err := getLiteLLMJSON(ctx, root, "/model_group/info", apiKey)
	if err != nil || len(b) == 0 {
		return nil, err
	}
	return parseLiteLLMModelGroupInfo(b)
}

// getLiteLLMJSON reads one LiteLLM management route. An empty result means the
// route does not exist here or this key may not read it; that is not an error,
// because most OpenAI-compatible servers have neither route.
func getLiteLLMJSON(ctx context.Context, root, path, apiKey string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, root+path, nil)
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
		return nil, fmt.Errorf("litellm %s HTTP %d", path, res.StatusCode)
	}
	return b, nil
}

// parseLiteLLMModelGroupInfo reads GET /model_group/info: one row per model
// group with the fields the proxy serves on /chat/completions.
func parseLiteLLMModelGroupInfo(b []byte) ([]catalogModel, error) {
	var doc struct {
		Data []struct {
			ModelGroup     string   `json:"model_group"`
			Description    string   `json:"description"`
			MaxInputTokens float64  `json:"max_input_tokens"`
			InputCost      *float64 `json:"input_cost_per_token"`
			OutputCost     *float64 `json:"output_cost_per_token"`
			SupportsVision bool     `json:"supports_vision"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	out := make([]catalogModel, 0, len(doc.Data))
	for _, row := range doc.Data {
		id := strings.TrimSpace(row.ModelGroup)
		if id == "" {
			continue
		}
		m := catalogModel{ID: id, SeesImages: row.SupportsVision}
		if name := strings.TrimSpace(row.Description); name != "" && len(name) <= 120 {
			m.Name = name
		}
		if row.MaxInputTokens > 0 {
			m.ContextLimit = int(row.MaxInputTokens)
		}
		// 0.0 is the proxy's placeholder for "no price known"; leaving it empty
		// lets /model/info supply the real rate.
		if row.InputCost != nil && *row.InputCost > 0 {
			m.PromptPrice = strconv.FormatFloat(*row.InputCost, 'g', -1, 64)
		}
		if row.OutputCost != nil && *row.OutputCost > 0 {
			m.CompletionPrice = strconv.FormatFloat(*row.OutputCost, 'g', -1, 64)
		}
		out = append(out, m)
	}
	return out, nil
}

// parseLiteLLMModelInfo reads GET /model/info: one row per deployment with
// model_name and model_info. model_info carries the per-token prices, the token
// limits, and the capability flags, including audio input which /model_group/info
// does not report.
func parseLiteLLMModelInfo(b []byte) ([]catalogModel, error) {
	var doc struct {
		Data []struct {
			ModelName string `json:"model_name"`
			ModelInfo struct {
				MaxInputTokens float64  `json:"max_input_tokens"`
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
			m.ContextLimit = int(row.ModelInfo.MaxInputTokens)
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
