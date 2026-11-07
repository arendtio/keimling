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
	SeeImages    bool
	HearAudio    bool
}

func (a *Agent) beginRunChat() {
	if a == nil {
		return
	}
	a.observeAgentModelChange()
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
	if strings.TrimSpace(a.Cfg.AgentInstanceID) != "" {
		return a.agentChatTarget(a.settingsChatTarget())
	}
	return a.effectiveChatTargetFor(a.Cfg.ThreadID)
}

func (a *Agent) effectiveChatTargetFor(threadID string) chatTarget {
	settings := a.settingsChatTarget()
	if a == nil {
		return settings
	}
	if strings.TrimSpace(a.Cfg.AgentInstanceID) != "" {
		return a.agentChatTarget(settings)
	}
	pid, mid := a.savedChatIDs(threadID)
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
		a.clearThreadChatModelOn(threadID, pid, mid)
		return settings
	}
	t := chatTargetFrom(p, m, a.Cfg.ContextLimit, a.Cfg.Reasoning)
	t.Custom = true
	return t
}

// configuredAgentModel is the instance override, else the definition, else empty.
func (a *Agent) configuredAgentModel() (id, source string) {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return "", ""
	}
	inst, err := readAgentInstanceMeta(a.Cfg.AgentInstanceID)
	if err != nil {
		return "", "settings"
	}
	if id := strings.TrimSpace(inst.Model); id != "" {
		return id, "instance"
	}
	rec, err := readAgentRecipe(inst.RecipeID)
	if err != nil {
		return "", "settings"
	}
	if id := strings.TrimSpace(rec.Model); id != "" {
		return id, "definition"
	}
	return "", "settings"
}

func (a *Agent) agentChatTarget(settings chatTarget) chatTarget {
	id, _ := a.configuredAgentModel()
	if id == "" {
		settings.Custom = false
		return settings
	}
	providers, _, ok := a.liveProviders()
	if !ok {
		settings.Custom = false
		return settings
	}
	p, m := findProviderModel(providers, id, settings.ProviderID)
	if p == nil || m == nil {
		settings.Custom = false
		return settings
	}
	t := chatTargetFrom(p, m, a.Cfg.ContextLimit, a.Cfg.Reasoning)
	t.Custom = false
	return t
}

func (a *Agent) chatModelLock() (locked bool, source string, unresolved bool, saved string) {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return false, "", false, ""
	}
	id, source := a.configuredAgentModel()
	if id == "" {
		return true, "settings", false, ""
	}
	providers, _, ok := a.liveProviders()
	if !ok {
		return true, source, true, id
	}
	settings := a.settingsChatTarget()
	p, m := findProviderModel(providers, id, settings.ProviderID)
	if p == nil || m == nil {
		return true, source, true, id
	}
	return true, source, false, id
}

// findProviderModel returns the configured model. When several providers have
// the same id, preferProvider wins if it has that id.
func findProviderModel(providers []modelProvider, modelID, preferProvider string) (*modelProvider, *providerModel) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, nil
	}
	var firstP *modelProvider
	var firstM *providerModel
	for i := range providers {
		providers[i].ensureProviderModels()
		m := providerModelByID(providers[i].Models, modelID)
		if m == nil {
			continue
		}
		if providers[i].ID == strings.TrimSpace(preferProvider) {
			return &providers[i], m
		}
		if firstM == nil {
			firstP = &providers[i]
			firstM = m
		}
	}
	return firstP, firstM
}

func validateKnownModel(configPath, modelID string) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil
	}
	fc, err := readConfiguration(strings.TrimSpace(configPath))
	if err != nil || len(fc.ModelProviders) == 0 {
		return fmt.Errorf("unknown model")
	}
	fc.ensureModelProviders()
	p, m := findProviderModel(fc.ModelProviders, modelID, fc.ActiveModelProviderID)
	if p == nil || m == nil {
		return fmt.Errorf("unknown model")
	}
	return nil
}

// observeAgentModelChange tells the user when an idle agent chat changes model
// and encrypted reasoning will not be sent. A run in progress is left alone.
func (a *Agent) observeAgentModelChange() {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) == "" || a.runChat != nil {
		return
	}
	next := a.effectiveChatTarget()
	if a.modelNoted {
		a.noteEncryptedReasoningOnModelChange(a.notedChat)
	}
	a.notedChat = next
	a.modelNoted = true
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
	t.SeeImages = m.nativeImages()
	t.HearAudio = m.nativeAudio()
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

func (a *Agent) savedChatIDs(threadID string) (string, string) {
	if a == nil {
		return "", ""
	}
	threadID = strings.TrimSpace(threadID)
	var meta ThreadMeta
	var err error
	if threadID == "" || threadID == strings.TrimSpace(a.Cfg.ThreadID) {
		meta, err = a.readOwnThreadMeta()
	} else {
		meta, err = readThreadMeta(a.Cfg.WorkspaceID, threadID)
	}
	if err != nil && strings.TrimSpace(meta.ModelID) == "" && strings.TrimSpace(meta.ModelProviderID) == "" {
		return "", ""
	}
	return strings.TrimSpace(meta.ModelProviderID), strings.TrimSpace(meta.ModelID)
}

// clearThreadChatModelOn drops a saved pair that is no longer in Settings.
// A newer pair written in the meantime is left alone.
func (a *Agent) clearThreadChatModelOn(threadID, providerID, modelID string) {
	if a == nil {
		return
	}
	_ = updateWorkspaceThreadMeta(a.Cfg.WorkspaceID, threadID, func(meta *ThreadMeta) error {
		if strings.TrimSpace(meta.ModelProviderID) != providerID || strings.TrimSpace(meta.ModelID) != modelID {
			return errThreadMetaSkip
		}
		meta.ModelProviderID = ""
		meta.ModelID = ""
		return nil
	})
}

// saveThreadChatModel stores a configured pair on the worker's current thread.
// The current Settings pair clears the thread choice.
func (a *Agent) saveThreadChatModel(providerID, modelID string) error {
	if a == nil {
		return fmt.Errorf("no thread")
	}
	return a.saveThreadChatModelOn(a.Cfg.ThreadID, providerID, modelID)
}

func (a *Agent) saveThreadChatModelOn(threadID, providerID, modelID string) error {
	if a == nil || a.Cfg.WorkspaceID == "" || strings.TrimSpace(threadID) == "" {
		return fmt.Errorf("no thread")
	}
	if strings.TrimSpace(a.Cfg.AgentInstanceID) != "" {
		return fmt.Errorf("an agent chat uses its definition's model")
	}
	threadID = strings.TrimSpace(threadID)
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
	clear := p.ID == settingsP.ID && m.ID == settingsModel
	var prev chatTarget
	if threadID == a.Cfg.ThreadID {
		prev = a.effectiveChatTarget()
	}
	if err := updateWorkspaceThreadMeta(a.Cfg.WorkspaceID, threadID, func(meta *ThreadMeta) error {
		if clear {
			meta.ModelProviderID = ""
			meta.ModelID = ""
			return nil
		}
		meta.ModelProviderID = p.ID
		meta.ModelID = m.ID
		return nil
	}); err != nil {
		return err
	}
	if threadID == a.Cfg.ThreadID {
		a.noteEncryptedReasoningOnModelChange(prev)
	}
	return nil
}

func (a *Agent) threadModelView() map[string]any {
	if a == nil {
		return map[string]any{}
	}
	return a.threadModelViewFor(a.Cfg.ThreadID)
}

func (a *Agent) threadModelViewFor(threadID string) map[string]any {
	locked, source, unresolved, saved := a.chatModelLock()
	next := a.effectiveChatTarget()
	if strings.TrimSpace(a.Cfg.AgentInstanceID) == "" && strings.TrimSpace(threadID) != "" && threadID != a.Cfg.ThreadID {
		next = a.effectiveChatTargetFor(threadID)
	}
	t := next
	if locked || (a.runChat != nil && threadID == a.Cfg.ThreadID) {
		t = a.callChatTarget()
	}
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
	nextName := next.Label
	if nextName == "" {
		nextName = next.Model
	}
	chosen := t
	if !locked {
		chosen = next
	}
	return map[string]any{
		"provider_id":          chosen.ProviderID,
		"model_id":             chosen.Model,
		"label":                label,
		"custom":               chosen.Custom,
		"reasoning":            t.Reasoning,
		"context_limit":        t.ContextLimit,
		"settings_provider_id": settings.ProviderID,
		"settings_model_id":    settings.Model,
		"providers":            pubs,
		"locked":               locked,
		"source":               source,
		"unresolved":           unresolved,
		"saved_model_id":       saved,
		"next_label":           nextName,
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
	a.syncInstanceWorkspace()
	t := a.callChatTarget()
	input = prepareReasoningInput(input, t.ProviderID, t.Model, a.reasoningOwnerMap())
	var resp Response
	var err error
	input = expandChatFiles(input, t.SeeImages, t.HearAudio)
	a.withChatEndpoint(func(model, reasoning string) {
		resp, err = a.Client.Respond(ctx, model, reasoning, input, a.Cfg.Web, a.modelInstructions(), extra, a.instanceToolGrant())
	})
	return resp, err
}
