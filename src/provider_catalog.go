package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
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
	Transcribes     bool   `json:"transcribes,omitempty"`
	Speaks          bool   `json:"speaks,omitempty"`
	Realtime        bool   `json:"realtime,omitempty"`
	// CapabilitiesKnown is false when the catalog has no capability fields,
	// as with OpenAI's model list. Prefill then leaves the checkboxes alone.
	CapabilitiesKnown bool `json:"capabilities_known,omitempty"`
	// Outputs is the catalog's output kind when it is not plain text, so the
	// prefill list can show speech, transcription, image, and the rest.
	Outputs string `json:"outputs,omitempty"`
	// BaseURL is the endpoint the catalog names for this model. Its path picks
	// the wire format, which is how one provider serves several API families.
	BaseURL string `json:"base_url,omitempty"`
	// Go and GoMonthly mark a model the opencode Go subscription pays for, and
	// its monthly allowance on the plain Go plan.
	Go        bool   `json:"go,omitempty"`
	GoMonthly string `json:"go_monthly,omitempty"`
}

func fetchProviderModelCatalog(ctx context.Context, baseURL, apiKey, kind string) ([]catalogModel, error) {
	models, err := fetchOpenAIModelList(ctx, baseURL, apiKey, openAIModelsPath(baseURL, kind))
	if detectProviderKind(baseURL, kind) == "opencode" {
		// The Go subscription has its own catalog: same fields plus the plan
		// allowance per model. Marking costs one extra request and no state.
		if goModels, goErr := fetchOpenAIModelList(ctx, openCodeGoURL(baseURL, "/v1"), apiKey, "models"); goErr == nil {
			models = markGoModels(models, goModels)
		}
	}
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
// OpenRouter and opencode are excluded so their catalog stays the /v1/models
// response alone.
func liteLLMCatalogRoot(baseURL, kind string) (string, bool) {
	switch detectProviderKind(baseURL, kind) {
	case "openrouter", "opencode":
		return "", false
	}
	root := providerRootURL(baseURL)
	if root == "" {
		return "", false
	}
	return root, true
}

// openAIModelsPath is the /models request. OpenRouter's bare list is text
// output only; speech, transcription, image, and the other kinds are returned
// when output_modalities=all, which is the site's "all models" list.
func openAIModelsPath(baseURL, kind string) string {
	if detectProviderKind(baseURL, kind) == "openrouter" {
		return "models?output_modalities=all"
	}
	return "models"
}

func fetchOpenAIModelList(ctx context.Context, baseURL, apiKey, path string) ([]catalogModel, error) {
	c := NewClient(apiKey)
	c.BaseURL = baseURL
	c.HTTP = &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(path), nil)
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
			// models.dev shape, as opencode.ai/inference/v1/models serves it.
			Limit *struct {
				Context float64 `json:"context"`
			} `json:"limit"`
			Cost *struct {
				Text *struct {
					Input struct {
						Price json.RawMessage `json:"price"`
					} `json:"input"`
					Output struct {
						Price json.RawMessage `json:"price"`
					} `json:"output"`
				} `json:"text"`
			} `json:"cost"`
			Type       string `json:"type"`
			Modalities *struct {
				Input  []string `json:"input"`
				Output []string `json:"output"`
			} `json:"modalities"`
			Capabilities *struct {
				Reasoning struct {
					Supported bool `json:"supported"`
				} `json:"reasoning"`
			} `json:"capabilities"`
			API map[string]catalogAPIEntry `json:"api"`
			Go  *struct {
				MonthlyUsage map[string]float64 `json:"monthly_usage"`
			} `json:"go"`
			Architecture *struct {
				Modality         string   `json:"modality"`
				InputModalities  []string `json:"input_modalities"`
				OutputModalities []string `json:"output_modalities"`
			} `json:"architecture"`
			InputModalities []string `json:"input_modalities"`
			Endpoints       []string `json:"endpoints"`
		}
		if err := json.Unmarshal(raw, &row); err != nil {
			continue
		}
		id := strings.TrimSpace(row.ID)
		if id == "" || strings.EqualFold(row.Type, "embedding") {
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
		case row.Limit != nil && row.Limit.Context > 0:
			m.ContextLimit = int(row.Limit.Context)
		case row.MaxTokens > 0:
			m.ContextLimit = int(row.MaxTokens)
		}
		if row.Pricing != nil {
			m.PromptPrice = catalogPrice(row.Pricing.Prompt)
			m.CompletionPrice = catalogPrice(row.Pricing.Completion)
		} else if row.Cost != nil && row.Cost.Text != nil {
			// models.dev quotes USD per million tokens; the form stores per token.
			m.PromptPrice = catalogPricePerToken(row.Cost.Text.Input.Price)
			m.CompletionPrice = catalogPricePerToken(row.Cost.Text.Output.Price)
		}
		inputs := row.InputModalities
		modality := ""
		if row.Architecture != nil {
			inputs = append(inputs, row.Architecture.InputModalities...)
			modality = row.Architecture.Modality
		}
		outputs := []string(nil)
		if row.Architecture != nil {
			outputs = row.Architecture.OutputModalities
			m.Outputs = catalogOutputs(outputs)
		}
		endpoints := row.Endpoints
		if row.Modalities != nil {
			inputs = append(inputs, row.Modalities.Input...)
			outputs = append(outputs, row.Modalities.Output...)
			m.Outputs = catalogOutputs(outputs)
		}
		if base := catalogAPIBase(row.API); base != "" {
			m.BaseURL = base
		}
		if row.Go != nil {
			m.Go = true
			if v := row.Go.MonthlyUsage["go"]; v > 0 {
				m.GoMonthly = "$" + strconv.FormatFloat(v, 'g', -1, 64) + "/mo"
			}
		}
		m.CapabilitiesKnown = row.Architecture != nil || len(endpoints) > 0 || row.Modalities != nil
		m.SeesImages, m.HearsAudio, m.Transcribes, m.Speaks, m.Realtime = catalogCapabilities(inputs, modality, outputs, endpoints)
		out = append(out, m)
	}
	return out, nil
}

// openCodeGoURL is a path under opencode's Go prefix, for any base URL form:
// a family base, the plain catalog, or the Go prefix itself.
func openCodeGoURL(baseURL, suffix string) string {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Host == "" {
		return ""
	}
	path := strings.TrimRight(u.Path, "/")
	if i := strings.Index(path, "/inference"); i >= 0 {
		path = path[:i]
	} else {
		path = ""
	}
	return u.Scheme + "://" + u.Host + path + "/inference/go" + suffix
}

// markGoModels copies Go membership onto the plain catalog. Models only in the
// Go list are appended, because a Go key can call them and nothing else.
func markGoModels(models, goModels []catalogModel) []catalogModel {
	byID := map[string]*catalogModel{}
	for i := range models {
		byID[models[i].ID] = &models[i]
	}
	for _, gm := range goModels {
		if m, ok := byID[gm.ID]; ok {
			m.Go = true
			if m.GoMonthly == "" {
				m.GoMonthly = gm.GoMonthly
			}
			continue
		}
		gm.Go = true
		models = append(models, gm)
	}
	return models
}

// catalogAPIEntry is one wire format a catalog model is served on.
type catalogAPIEntry struct {
	BaseURL string `json:"base_url"`
}

// catalogAPIBase picks the endpoint to call, preferring the family Keimling
// speaks natively. Chat Completions shares its base with the Responses API, so
// that one is stored as the full endpoint to make the format detectable.
func catalogAPIBase(api map[string]catalogAPIEntry) string {
	for _, family := range []string{"responses", "chat-completions", "messages", "generate-content"} {
		entry, ok := api[family]
		base := strings.TrimRight(strings.TrimSpace(entry.BaseURL), "/")
		if !ok || base == "" {
			continue
		}
		if family == "chat-completions" && !strings.HasSuffix(base, "/chat/completions") {
			base += "/chat/completions"
		}
		return base
	}
	return ""
}

// catalogCapabilities reads what the catalog actually reports. A model whose
// only output is a transcript takes audio, but that is transcription, not a
// chat model that hears. OpenAI's list has none of these fields, so every
// flag stays off and the form leaves the choice to the user.
func catalogCapabilities(inputs []string, modality string, outputs, endpoints []string) (images, audio, transcribes, speaks, realtime bool) {
	images, audio = catalogModalities(inputs, modality)
	note := func(s string) {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "transcription":
			transcribes = true
		case "speech", "audio":
			speaks = true
		case "realtime":
			realtime = true
		}
	}
	for _, o := range outputs {
		note(o)
	}
	for _, e := range endpoints {
		e = strings.ToLower(e)
		switch {
		case strings.Contains(e, "transcription"):
			transcribes = true
		case strings.Contains(e, "realtime"):
			realtime = true
		case strings.Contains(e, "/audio/speech") || strings.HasSuffix(e, "/speech"):
			speaks = true
		}
	}
	textOut := len(outputs) == 0
	for _, o := range outputs {
		if strings.EqualFold(strings.TrimSpace(o), "text") {
			textOut = true
		}
	}
	if transcribes && !textOut {
		audio = false
	}
	return images, audio, transcribes, speaks, realtime
}

func catalogOutputs(outs []string) string {
	clean := make([]string, 0, len(outs))
	for _, o := range outs {
		o = strings.TrimSpace(o)
		if o != "" {
			clean = append(clean, o)
		}
	}
	if len(clean) == 0 || (len(clean) == 1 && clean[0] == "text") {
		return ""
	}
	return strings.Join(clean, ", ")
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

// catalogPricePerToken turns a per-million price into the per-token string the
// model form stores. 0 stays "0" so a free model lists as free.
func catalogPricePerToken(raw json.RawMessage) string {
	v, ok := parseUSDPerToken(catalogPrice(raw))
	if !ok {
		return ""
	}
	return strconv.FormatFloat(v/1e6, 'g', -1, 64)
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
		if e.Transcribes {
			cur.Transcribes = true
		}
		if e.Speaks {
			cur.Speaks = true
		}
		if e.Realtime {
			cur.Realtime = true
		}
		if e.CapabilitiesKnown {
			cur.CapabilitiesKnown = true
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
