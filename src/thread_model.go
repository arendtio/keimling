package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// chatTarget is the main chat model for one call.
// Compaction, autopilot, delegates, and review stay on Settings.
type chatTarget struct {
	ProviderID   string
	Model        string
	Label        string
	Reasoning    string
	ContextLimit int
	BaseURL      string
	APIKey       string
	Custom       bool
}

func (a *Agent) beginRunChat() {
	if a == nil {
		return
	}
	t := a.effectiveChatTarget()
	a.runChat = &t
}

func (a *Agent) endRunChat() {
	if a == nil {
		return
	}
	a.runChat = nil
}

// callChatTarget is the model for the chat call in progress.
// A run keeps the target it started with. Idle calls follow the thread live.
func (a *Agent) callChatTarget() chatTarget {
	if a != nil && a.runChat != nil {
		return *a.runChat
	}
	return a.effectiveChatTarget()
}

func (a *Agent) activeChatContextLimit() int {
	if a == nil {
		return 0
	}
	if t := a.callChatTarget(); t.ContextLimit > 0 {
		return t.ContextLimit
	}
	return a.Cfg.ContextLimit
}

func (a *Agent) effectiveChatTarget() chatTarget {
	if a == nil {
		return chatTarget{}
	}
	settings := a.settingsChatTarget()
	pid, mid := a.threadChatIDs()
	if pid == "" || mid == "" {
		return settings
	}
	providers, _, ok := a.liveProviders()
	p := (*modelProvider)(nil)
	if ok {
		p = modelProviderByID(providers, pid)
	}
	var m *providerModel
	if p != nil {
		p.ensureProviderModels()
		m = providerModelByID(p.Models, mid)
	}
	if p == nil || m == nil {
		a.clearThreadChatModel(pid, mid)
		return settings
	}
	t := chatTargetFrom(p, m, a.Cfg.ContextLimit, a.Cfg.Reasoning)
	t.Custom = true
	return t
}

func (a *Agent) settingsChatTarget() chatTarget {
	if a == nil {
		return chatTarget{}
	}
	fallback := chatTarget{
		Model:        a.Cfg.Model,
		Label:        a.Cfg.Model,
		Reasoning:    a.Cfg.Reasoning,
		ContextLimit: a.Cfg.ContextLimit,
		BaseURL:      a.Cfg.BaseURL,
		APIKey:       a.Cfg.Key,
	}
	if a == nil {
		return fallback
	}
	providers, activeID, ok := a.liveProviders()
	if !ok {
		return fallback
	}
	p := modelProviderByID(providers, activeID)
	if p == nil {
		p = &providers[0]
	}
	p.ensureProviderModels()
	return chatTargetFrom(p, p.activeModel(), a.Cfg.ContextLimit, a.Cfg.Reasoning)
}

func chatTargetFrom(p *modelProvider, m *providerModel, globalLimit int, globalReasoning string) chatTarget {
	t := chatTarget{
		Reasoning:    globalReasoning,
		ContextLimit: globalLimit,
	}
	if p == nil {
		return t
	}
	t.ProviderID = p.ID
	t.BaseURL = strings.TrimSpace(p.BaseURL)
	t.APIKey = strings.TrimSpace(p.APIKey)
	if m == nil {
		t.Model = strings.TrimSpace(p.Model)
		t.Label = t.Model
		return t
	}
	t.Model = m.ID
	t.Label = m.displayName()
	if r := strings.TrimSpace(m.Reasoning); r != "" {
		t.Reasoning = r
	}
	if m.ContextLimit >= 1000 {
		t.ContextLimit = m.ContextLimit
	}
	return t
}

func (a *Agent) liveProviders() ([]modelProvider, string, bool) {
	if a == nil {
		return nil, "", false
	}
	path := strings.TrimSpace(a.Cfg.ConfigPath)
	if path == "" {
		return nil, "", false
	}
	fc, err := readConfiguration(path)
	if err != nil || len(fc.ModelProviders) == 0 {
		return nil, "", false
	}
	fc.ensureModelProviders()
	return fc.ModelProviders, fc.ActiveModelProviderID, true
}

func (a *Agent) threadChatIDs() (string, string) {
	if a == nil || a.Cfg.WorkspaceID == "" || a.Cfg.ThreadID == "" {
		return "", ""
	}
	meta, err := readThreadMeta(a.Cfg.WorkspaceID, a.Cfg.ThreadID)
	if err != nil {
		return "", ""
	}
	return strings.TrimSpace(meta.ModelProviderID), strings.TrimSpace(meta.ModelID)
}

// clearThreadChatModel drops a saved pair that is no longer in Settings.
// A newer pair written in the meantime is left alone.
func (a *Agent) clearThreadChatModel(providerID, modelID string) {
	if a == nil || a.Cfg.WorkspaceID == "" || a.Cfg.ThreadID == "" {
		return
	}
	meta, _ := readThreadMeta(a.Cfg.WorkspaceID, a.Cfg.ThreadID)
	if strings.TrimSpace(meta.ModelProviderID) != providerID || strings.TrimSpace(meta.ModelID) != modelID {
		return
	}
	meta.ID = a.Cfg.ThreadID
	meta.ModelProviderID = ""
	meta.ModelID = ""
	_ = writeThreadMeta(a.Cfg.WorkspaceID, meta)
}

// saveThreadChatModel stores a configured pair. The current Settings pair clears the thread choice.
func (a *Agent) saveThreadChatModel(providerID, modelID string) error {
	if a == nil || a.Cfg.WorkspaceID == "" || a.Cfg.ThreadID == "" {
		return fmt.Errorf("no thread")
	}
	providers, activeID, ok := a.liveProviders()
	if !ok {
		return fmt.Errorf("no configured providers")
	}
	p := modelProviderByID(providers, providerID)
	if p == nil {
		return fmt.Errorf("unknown provider")
	}
	p.ensureProviderModels()
	m := providerModelByID(p.Models, modelID)
	if m == nil {
		return fmt.Errorf("unknown model")
	}
	settingsP := modelProviderByID(providers, activeID)
	if settingsP == nil {
		settingsP = &providers[0]
	}
	settingsP.ensureProviderModels()
	settingsModel := ""
	if sm := settingsP.activeModel(); sm != nil {
		settingsModel = sm.ID
	}
	meta, _ := readThreadMeta(a.Cfg.WorkspaceID, a.Cfg.ThreadID)
	meta.ID = a.Cfg.ThreadID
	if p.ID == settingsP.ID && m.ID == settingsModel {
		meta.ModelProviderID = ""
		meta.ModelID = ""
	} else {
		meta.ModelProviderID = p.ID
		meta.ModelID = m.ID
	}
	return writeThreadMeta(a.Cfg.WorkspaceID, meta)
}

func (a *Agent) threadModelView() map[string]any {
	t := a.effectiveChatTarget()
	settings := a.settingsChatTarget()
	providers, _, ok := a.liveProviders()
	pubs := []map[string]any{}
	if ok {
		pubs = publicModelProviders(providers)
	}
	label := t.Label
	if label == "" {
		label = t.Model
	}
	return map[string]any{
		"provider_id":          t.ProviderID,
		"model_id":             t.Model,
		"label":                label,
		"custom":               t.Custom,
		"reasoning":            t.Reasoning,
		"context_limit":        t.ContextLimit,
		"settings_provider_id": settings.ProviderID,
		"settings_model_id":    settings.Model,
		"providers":            pubs,
	}
}

// withChatEndpoint applies the chat provider only for fn.
// The process client stays on the Settings endpoint for compaction and other roles.
func (a *Agent) withChatEndpoint(fn func(model, reasoning string)) {
	if a == nil || fn == nil {
		return
	}
	t := a.callChatTarget()
	model := t.Model
	if model == "" {
		model = a.Cfg.Model
	}
	reasoning := t.Reasoning
	if reasoning == "" {
		reasoning = a.Cfg.Reasoning
	}
	if a.Client == nil {
		fn(model, reasoning)
		return
	}
	prevURL, prevKey := a.Client.BaseURL, a.Client.APIKey
	if u := strings.TrimSpace(t.BaseURL); u != "" {
		a.Client.BaseURL = u
	}
	if t.ProviderID != "" {
		a.Client.APIKey = t.APIKey
	}
	defer func() {
		a.Client.BaseURL = prevURL
		a.Client.APIKey = prevKey
	}()
	fn(model, reasoning)
}

func (a *Agent) respondMain(ctx context.Context, input []json.RawMessage, extra []map[string]any) (Response, error) {
	if a == nil || a.Client == nil {
		return Response{}, fmt.Errorf("no API client")
	}
	var resp Response
	var err error
	a.withChatEndpoint(func(model, reasoning string) {
		resp, err = a.Client.Respond(ctx, model, reasoning, input, a.Cfg.Web, a.skillsCatalogBlock(), extra)
	})
	return resp, err
}
