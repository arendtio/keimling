package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// openRuntimeAgent builds the agent for one workspace or one agent instance
// inside this process. It does not change the process working directory.
// MCP servers connect in the background so opening the workspace does not
// wait on them. cancel stops that work.
func openRuntimeAgent(configPath, workspaceID, agentInstanceID string) (*Agent, context.CancelFunc, error) {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return nil, nil, fmt.Errorf("config path required")
	}
	fc, err := readConfiguration(configPath)
	if err != nil {
		return nil, nil, err
	}
	credKey, credErr := apiKey()
	if errors.Is(credErr, errAPIKeyMissing) {
		credKey = ""
	} else if credErr != nil {
		return nil, nil, credErr
	}
	if err := prepareStartupServices(configPath, &fc, credKey); err != nil {
		return nil, nil, err
	}
	base := strings.TrimSpace(fc.API.BaseURL)
	if base == "" {
		base = defaultBaseURL
	}
	resolvedURL, err := normalizeBaseURL(base)
	if err != nil {
		return nil, nil, err
	}
	model := strings.TrimSpace(fc.API.Model)
	if model == "" {
		model = "gpt-6-luna"
	}
	key := strings.TrimSpace(fc.API.Key)
	if key == "" && strings.TrimSpace(fc.API.KeyFile) != "" {
		key, err = readKeyFile(fc.API.KeyFile)
		if err != nil {
			return nil, nil, err
		}
	}
	custom := resolvedURL != defaultBaseURL
	if key == "" && !custom {
		key, err = apiKey()
		if errors.Is(err, errAPIKeyMissing) {
			key = ""
			err = nil
		}
		if err != nil {
			return nil, nil, err
		}
	}
	mode := strings.TrimSpace(fc.API.Mode)
	if mode != "ask" && mode != "auto" && mode != "yolo" {
		mode = "auto"
	}
	web := true
	if fc.Features.WebSearch != nil {
		web = *fc.Features.WebSearch
	}
	disableExternal := fc.Features.DisableExternalServices
	if disableExternal || !fc.Search.enabled() {
		web = false
	}
	steps := fc.MaxSteps
	if steps < 1 {
		steps = 200
	}
	ctxLimit := fc.ContextLimit
	if ctxLimit < 1000 {
		ctxLimit = 128000
	}
	reasoning := strings.TrimSpace(fc.Reasoning)
	if reasoning == "" {
		reasoning = "high"
	}
	voiceModel := strings.TrimSpace(fc.VoiceModel)
	if voiceModel == "" {
		voiceModel = "gpt-transcribe"
	}
	var external *Client
	voiceProviderID := strings.TrimSpace(fc.VoiceProviderID)
	realtimeProviderID := strings.TrimSpace(fc.RealtimeProviderID)
	if !disableExternal && fc.Search.Kind == "openai" {
		external = NewClient("")
		external.APIKey = fc.Search.APIKey
		if u := strings.TrimSpace(fc.Search.URL); u != "" {
			if nu, e := normalizeBaseURL(u); e == nil {
				external.BaseURL = nu
			}
		}
	}
	if disableExternal {
		external = nil
		voiceProviderID = ""
		realtimeProviderID = ""
		web = false
	}
	policyText := strings.TrimSpace(fc.Features.ApprovalPolicy)
	policyReplace := policyText != ""
	pilot := fc.Features.Autopilot != nil && *fc.Features.Autopilot

	var (
		ws                WorkspaceMeta
		threadID          string
		sessPath          string
		agentInstructions string
		allowed           []string
	)
	if id := strings.TrimSpace(agentInstanceID); id != "" {
		inst, e := readAgentInstanceMeta(id)
		if e != nil {
			return nil, nil, e
		}
		recipe, e := readAgentRecipe(inst.RecipeID)
		if e != nil {
			return nil, nil, e
		}
		m, e := readWorkspaceMeta(inst.CurrentWorkspace)
		if e != nil {
			return nil, nil, e
		}
		ws = m
		threadID = "main"
		sessPath = agentInstanceSessionPath(inst.ID)
		agentInstanceID = inst.ID
		agentInstructions = recipe.Instructions
		allowed = append([]string{}, inst.Workspaces...)
		if recipe.Model != "" {
			model = recipe.Model
		}
		inst.Status = agentStatusIdle
		_ = writeAgentInstanceMeta(inst)
	} else {
		m, e := readWorkspaceMeta(workspaceID)
		if e != nil {
			return nil, nil, e
		}
		tid, sp, e := ensureActiveThread(m.ID)
		if e != nil {
			return nil, nil, e
		}
		ws, threadID, sessPath = m, tid, sp
		_ = writeWorkspaceMeta(m)
	}

	journal, err := OpenJournal(sessPath, true)
	if err != nil {
		return nil, nil, err
	}
	cfg := Config{
		Model: model, CompactionModel: "", Mode: mode, Key: key, SessionPath: sessPath,
		VoiceModel: voiceModel, ApprovalPolicy: policyText, Reasoning: reasoning,
		BaseURL: resolvedURL, ApprovalPolicyReplace: policyReplace,
		DelegateBalanced: "gpt-6.1-sol", DelegateDeep: "gpt-6.1-sol",
		PilotReasoning: "high",
		// A worker has no flags: review, filter, autopilot, fast delegate, and
		// web search take auxiliary_model, else this worker's chat model.
		AuxiliaryModel: strings.TrimSpace(fc.AuxiliaryModel),
		ConfigPath:     configPath, PublicURL: strings.TrimRight(strings.TrimSpace(fc.PublicURL), "/"),
		WorkspaceID: ws.ID, WorkspaceRoot: ws.Root, ThreadID: threadID,
		AgentInstanceID: strings.TrimSpace(agentInstanceID), AgentInstructions: agentInstructions,
		AllowedWorkspaces: allowed, Autopilot: pilot,
		AutopilotInstructions: strings.TrimSpace(fc.Features.AutopilotInstructions),
		ToolAsk:               fc.Features.ToolAsk, ArtifactProxy: fc.Features.ArtifactProxy,
		ExternalClient: external, VoiceProviderID: voiceProviderID,
		RealtimeModel: strings.TrimSpace(fc.RealtimeModel), RealtimeProviderID: realtimeProviderID,
		RealtimeVoice:    strings.TrimSpace(fc.RealtimeVoice),
		VisionModel:      strings.TrimSpace(fc.VisionModel),
		VisionProviderID: strings.TrimSpace(fc.VisionProviderID),
		SpeechModel:      strings.TrimSpace(fc.SpeechModel),
		SpeechProviderID: strings.TrimSpace(fc.SpeechProviderID),
		IdleSound:        fc.Features.IdleSound,
		IdleNotify:       fc.Features.IdleNotify,
		HasModelProvider: len(fc.ModelProviders) > 0, DisableExternal: disableExternal,
		Search: fc.Search, Web: web, CompactParallel: true,
		MaxSteps: steps, ContextLimit: ctxLimit, MCPParallel: mcpParallelLimit(fc.MCPParallel),
	}
	a := &Agent{
		Cfg: cfg, Client: NewClient(key), Journal: journal, History: journal.History,
		Shares: newFileShareHub(), Skills: LoadSkillRegistry(""), MCP: NewMCPManager(),
		Credentials: NewCredentialBroker(configPath),
	}
	a.Client.Spend = agentSpendHook(a)
	a.Cfg.ConversationMode = a.Cfg.Mode
	a.applyDefinitionApproval()
	if ch, chErr := loadCharter(sessPath); chErr == nil {
		a.Charter = ch
	} else {
		a.Charter = &Charter{Version: charterVersion}
	}
	a.restoreActiveBranch()
	a.Client.BaseURL = resolvedURL
	if cfg.PublicURL != "" {
		a.Shares.SetPublicBase(cfg.PublicURL)
	}
	sessionCtx, cancel := context.WithCancel(context.Background())
	a.SessionCtx = sessionCtx
	if err := a.Credentials.TryUnlockFromEnv(); err != nil {
		cancel()
		_ = journal.Close()
		return nil, nil, err
	}
	a.LastTaskMessage, a.ResumeAvailable = pendingTask(sessPath)
	go a.startMCP(sessionCtx)
	return a, cancel, nil
}
