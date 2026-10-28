package main

import (
	"bufio"
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed webui/*
var webuiFS embed.FS

type appServerOpts struct {
	AccessSecret string
	Agent        *Agent
}

type appServer struct {
	secret string
	agent  *Agent
	hub    *eventHub
	mux    *http.ServeMux

	mu              sync.Mutex
	busy            bool
	cancel          context.CancelFunc
	skillCandidates []SkillCandidate
}

func newAppServer(opts appServerOpts) *appServer {
	s := &appServer{
		secret: opts.AccessSecret,
		agent:  opts.Agent,
		hub:    newEventHub(),
		mux:    http.NewServeMux(),
	}
	if s.agent != nil {
		if s.agent.Shares == nil {
			s.agent.Shares = newFileShareHub()
		}
		if base := strings.TrimSpace(s.agent.Cfg.PublicURL); base != "" {
			s.agent.Shares.SetPublicBase(base)
		}
		s.agent.Emit = func(kind, text string) {
			switch kind {
			case "status":
				s.hub.Broadcast("status", map[string]string{"state": text})
			case "progress":
				var payload map[string]any
				if json.Unmarshal([]byte(text), &payload) == nil {
					s.hub.Broadcast("progress", payload)
					return
				}
			case "approval":
				var payload map[string]any
				if json.Unmarshal([]byte(text), &payload) == nil {
					s.hub.Broadcast("approval", payload)
					return
				}
				s.hub.Broadcast("line", map[string]string{"text": text})
			default:
				s.hub.Broadcast("line", map[string]string{"text": text})
			}
		}
		if s.agent.ApprovalCh == nil {
			s.agent.ApprovalCh = make(chan bool, 1)
		}
	}
	ui, err := fs.Sub(webuiFS, "webui")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(ui))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
	s.mux.Handle("/api/health", s.requireSecret(http.HandlerFunc(s.handleHealth)))
	s.mux.Handle("/api/session", s.requireSecret(http.HandlerFunc(s.handleSession)))
	s.mux.Handle("/api/context", s.requireSecret(http.HandlerFunc(s.handleContext)))
	s.mux.Handle("/api/skills", s.requireSecret(http.HandlerFunc(s.handleSkills)))
	s.mux.Handle("/api/skills/add", s.requireSecret(http.HandlerFunc(s.handleSkillsAdd)))
	s.mux.Handle("/api/skills/search", s.requireSecret(http.HandlerFunc(s.handleSkillsSearch)))
	s.mux.Handle("/api/skills/install", s.requireSecret(http.HandlerFunc(s.handleSkillsInstall)))
	s.mux.Handle("/api/mcp", s.requireSecret(http.HandlerFunc(s.handleMCP)))
	s.mux.Handle("/api/mcp/add", s.requireSecret(http.HandlerFunc(s.handleMCPAdd)))
	s.mux.Handle("/api/mcp/remove", s.requireSecret(http.HandlerFunc(s.handleMCPRemove)))
	s.mux.Handle("/api/mcp/reload", s.requireSecret(http.HandlerFunc(s.handleMCPReload)))
	s.mux.Handle("/api/message", s.requireSecret(http.HandlerFunc(s.handleMessage)))
	s.mux.Handle("/api/newchat", s.requireSecret(http.HandlerFunc(s.handleNewChat)))
	s.mux.Handle("/api/settings", s.requireSecret(http.HandlerFunc(s.handleSettings)))
	s.mux.Handle("/api/gui-prefs", s.requireSecret(http.HandlerFunc(s.handleGuiPrefs)))
	s.mux.Handle("/api/approve", s.requireSecret(http.HandlerFunc(s.handleApprove)))
	s.mux.Handle("/api/transcribe", s.requireSecret(http.HandlerFunc(s.handleTranscribe)))
	s.mux.Handle("/events", s.requireSecret(http.HandlerFunc(s.handleEvents)))
	s.mux.HandleFunc("/files/", s.handleFileShare)
	return s
}

func (s *appServer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.agent != nil && s.agent.Shares != nil && strings.TrimSpace(s.agent.Cfg.PublicURL) == "" {
			if base := publicBaseFromRequest(r); base != "" {
				s.agent.Shares.SetPublicBase(base)
			}
		}
		s.mux.ServeHTTP(w, r)
	})
}

func (s *appServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	state := "idle"
	s.mu.Lock()
	if s.busy {
		state = "running"
	}
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":    true,
		"state": state,
		"voice": s.voiceAvailable(),
	})
}

func (s *appServer) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	path := s.agent.Cfg.SessionPath
	uiHistory := s.agent.History
	if events, err := originalEvents(path); err == nil && len(events) > 0 {
		uiHistory = events
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"model":         s.agent.Cfg.Model,
		"mode":          s.agent.Cfg.Mode,
		"reasoning":     s.agent.Cfg.Reasoning,
		"web":           s.agent.Cfg.Web,
		"autopilot":     s.agent.Cfg.Autopilot,
		"max_steps":     s.agent.Cfg.MaxSteps,
		"context_limit": s.agent.Cfg.ContextLimit,
		"voice_model":   s.agent.Cfg.VoiceModel,
		"lines":         sessionTranscript(uiHistory, 200),
		"messages":      sessionMessages(uiHistory, 120),
		"chapters":      sessionChapters(path),
		"working_state": sessionWorkingState(path),
		"busy":          s.isBusy(),
		"voice":         s.voiceAvailable(),
	})
}

func (s *appServer) handleContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	c := s.agent.ensureCharter()
	b := s.agent.contextBudget()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"charter":  c,
		"markdown": c.RenderMarkdown(),
		"budget":   b,
	})
}

func (s *appServer) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	s.agent.refreshSkills()
	list := s.agent.ensureSkills().List()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"skills": list})
}

func (s *appServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"servers": s.agent.ensureMCP().Status(),
		"path":    defaultUserMCPPath(),
	})
}

func (s *appServer) handleMCPAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Name    string            `json:"name"`
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Type    string            `json:"type"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	cfg := MCPServerConfig{
		Command: body.Command,
		Args:    body.Args,
		Env:     body.Env,
		URL:     body.URL,
		Headers: body.Headers,
		Type:    body.Type,
	}
	if err := UpsertUserMCPServer(body.Name, cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.agent.reloadMCP(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "servers": s.agent.ensureMCP().Status()})
}

func (s *appServer) handleMCPRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := RemoveUserMCPServer(body.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.agent.reloadMCP(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "servers": s.agent.ensureMCP().Status()})
}

func (s *appServer) handleMCPReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	s.agent.reloadMCP(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "servers": s.agent.ensureMCP().Status()})
}

func (s *appServer) handleSkillsAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Path  string `json:"path"`
		Scope string `json:"scope"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	scope := SkillScopeUser
	if body.Scope == "project" {
		scope = SkillScopeProject
	}
	cwd, _ := os.Getwd()
	sk, err := InstallSkillFromPath(body.Path, scope, cwd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.agent.refreshSkills()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "skill": sk})
}

func (s *appServer) handleSkillsSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	client := s.agent.Cfg.ExternalClient
	if client == nil {
		http.Error(w, "web search unavailable", http.StatusServiceUnavailable)
		return
	}
	cands, raw, err := SearchSkillsOnline(r.Context(), client, body.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	s.mu.Lock()
	s.skillCandidates = cands
	s.mu.Unlock()
	lastSkillSearch = cands
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"candidates": cands, "raw": raw})
}

func (s *appServer) handleSkillsInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		URL         string `json:"url"`
		CandidateID string `json:"candidate_id"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	scope := SkillScopeUser
	if body.Scope == "project" {
		scope = SkillScopeProject
	}
	installURL := strings.TrimSpace(body.URL)
	if installURL == "" && body.CandidateID != "" {
		s.mu.Lock()
		cands := s.skillCandidates
		s.mu.Unlock()
		for _, c := range cands {
			if c.ID == body.CandidateID {
				installURL = c.SourceURL
				break
			}
		}
	}
	if installURL == "" {
		http.Error(w, "url or candidate_id required", http.StatusBadRequest)
		return
	}
	cwd, _ := os.Getwd()
	sk, err := InstallSkillFromURL(installURL, scope, cwd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.agent.refreshSkills()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "skill": sk})
}

func (s *appServer) handleNewChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		http.Error(w, "agent busy", http.StatusConflict)
		return
	}
	s.mu.Unlock()
	if err := s.agent.NewChat(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.hub.Broadcast("status", map[string]string{"state": "idle"})
	s.hub.Broadcast("line", map[string]string{"text": "\n[session] New chat started.\n"})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

type agentSettings struct {
	BaseURL      string  `json:"base_url,omitempty"`
	Model        string  `json:"model"`
	Mode         string  `json:"mode"`
	Reasoning    string  `json:"reasoning"`
	Web          *bool   `json:"web,omitempty"`
	Autopilot    *bool   `json:"autopilot,omitempty"`
	MaxSteps     *int    `json:"max_steps,omitempty"`
	ContextLimit *int    `json:"context_limit,omitempty"`
	VoiceModel   string  `json:"voice_model,omitempty"`
	APIKey       *string `json:"api_key,omitempty"`        // write-only
	OpenAIAPIKey *string `json:"openai_api_key,omitempty"` // write-only
	APIKeySet    bool    `json:"api_key_set"`              // GET only
	OpenAIKeySet bool    `json:"openai_api_key_set"`       // GET only
}

func (s *appServer) settingsSnapshot() agentSettings {
	web, auto := s.agent.Cfg.Web, s.agent.Cfg.Autopilot
	steps, limit := s.agent.Cfg.MaxSteps, s.agent.Cfg.ContextLimit
	base := s.agent.Cfg.BaseURL
	if s.agent.Client != nil && strings.TrimSpace(s.agent.Client.BaseURL) != "" {
		base = s.agent.Client.BaseURL
	}
	providerKey := strings.TrimSpace(s.agent.Cfg.Key)
	if s.agent.Client != nil && strings.TrimSpace(s.agent.Client.APIKey) != "" {
		providerKey = strings.TrimSpace(s.agent.Client.APIKey)
	}
	openaiSet := false
	if s.agent.Cfg.ExternalClient != nil {
		openaiSet = strings.TrimSpace(s.agent.Cfg.ExternalClient.APIKey) != ""
	}
	return agentSettings{
		BaseURL:      base,
		Model:        s.agent.Cfg.Model,
		Mode:         s.agent.Cfg.Mode,
		Reasoning:    s.agent.Cfg.Reasoning,
		Web:          &web,
		Autopilot:    &auto,
		MaxSteps:     &steps,
		ContextLimit: &limit,
		VoiceModel:   s.agent.Cfg.VoiceModel,
		APIKeySet:    providerKey != "",
		OpenAIKeySet: openaiSet,
	}
}

func (s *appServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.settingsSnapshot())
	case http.MethodPut, http.MethodPatch:
		s.mu.Lock()
		busy := s.busy
		s.mu.Unlock()
		if busy {
			http.Error(w, "agent busy", http.StatusConflict)
			return
		}
		var body agentSettings
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := s.applySettings(body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.settingsSnapshot())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *appServer) applySettings(body agentSettings) error {
	cfg := &s.agent.Cfg
	if raw := strings.TrimSpace(body.BaseURL); raw != "" {
		resolved, err := normalizeBaseURL(raw)
		if err != nil {
			return err
		}
		cfg.BaseURL = resolved
		if s.agent.Client != nil {
			s.agent.Client.BaseURL = resolved
		}
	}
	if m := strings.TrimSpace(body.Mode); m != "" {
		if m != "ask" && m != "auto" && m != "yolo" {
			return fmt.Errorf("mode must be ask, auto, or yolo")
		}
		cfg.Mode = m
	}
	if model := strings.TrimSpace(body.Model); model != "" {
		cfg.Model = model
	}
	if r := strings.TrimSpace(body.Reasoning); r != "" {
		switch r {
		case "low", "medium", "high", "xhigh", "max":
			cfg.Reasoning = r
		default:
			return fmt.Errorf("reasoning must be low, medium, high, xhigh, or max")
		}
	}
	if body.Web != nil {
		cfg.Web = *body.Web
	}
	if body.Autopilot != nil {
		cfg.Autopilot = *body.Autopilot
	}
	if body.MaxSteps != nil {
		if *body.MaxSteps < 1 {
			return fmt.Errorf("max_steps must be positive")
		}
		cfg.MaxSteps = *body.MaxSteps
	}
	if body.ContextLimit != nil {
		if *body.ContextLimit < 1000 {
			return fmt.Errorf("context_limit too small")
		}
		cfg.ContextLimit = *body.ContextLimit
	}
	if v := strings.TrimSpace(body.VoiceModel); v != "" {
		cfg.VoiceModel = v
	}
	writeProviderKey := body.APIKey != nil
	writeOpenAIKey := body.OpenAIAPIKey != nil
	if writeProviderKey {
		k := strings.TrimSpace(*body.APIKey)
		cfg.Key = k
		if s.agent.Client == nil {
			s.agent.Client = NewClient(k)
			s.agent.Client.BaseURL = cfg.BaseURL
		} else {
			s.agent.Client.APIKey = k
		}
	}
	if writeOpenAIKey {
		k := strings.TrimSpace(*body.OpenAIAPIKey)
		if k == "" {
			cfg.ExternalClient = nil
		} else if cfg.ExternalClient == nil {
			cfg.ExternalClient = NewClient(k)
		} else {
			cfg.ExternalClient.APIKey = k
		}
	}
	if err := s.persistAgentConfig(writeProviderKey, writeOpenAIKey); err != nil {
		return fmt.Errorf("persist config: %w", err)
	}
	return nil
}

func (s *appServer) handleGuiPrefs(w http.ResponseWriter, r *http.Request) {
	path := ""
	if s.agent != nil {
		path = s.agent.Cfg.ConfigPath
	}
	switch r.Method {
	case http.MethodGet:
		p, err := readGuiPrefs(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p)
	case http.MethodPut, http.MethodPatch:
		var body guiPrefs
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := mergeGuiPrefsConnect(path, body.Connect, body.AccessSecret); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		p, err := readGuiPrefs(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *appServer) persistAgentConfig(writeProviderKey, writeOpenAIKey bool) error {
	path := strings.TrimSpace(s.agent.Cfg.ConfigPath)
	if path == "" {
		return nil
	}
	cfg := &s.agent.Cfg
	return updateConfiguration(path, func(fc *fileConfig) {
		fc.API.BaseURL = cfg.BaseURL
		fc.API.Model = cfg.Model
		fc.API.Mode = cfg.Mode
		web := cfg.Web
		fc.Features.WebSearch = &web
		if writeProviderKey {
			fc.API.Key = cfg.Key
			fc.API.KeyFile = ""
		}
		if writeOpenAIKey {
			k := ""
			if cfg.ExternalClient != nil {
				k = strings.TrimSpace(cfg.ExternalClient.APIKey)
			}
			fc.OpenAI.Key = k
			fc.OpenAI.KeyFile = ""
		}
	})
}

func (s *appServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(body.Text)
	if text == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		http.Error(w, "agent busy", http.StatusConflict)
		return
	}
	s.busy = true
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.mu.Unlock()

	s.hub.Broadcast("line", map[string]string{"text": "\n› " + text + "\n"})
	s.hub.Broadcast("status", map[string]string{"state": "running"})

	go func() {
		defer func() {
			s.mu.Lock()
			s.busy = false
			s.cancel = nil
			s.mu.Unlock()
			s.hub.Broadcast("status", map[string]string{"state": "idle"})
		}()
		err := s.agent.Run(ctx, text, bufio.NewReader(strings.NewReader("")))
		if err != nil && ctx.Err() == nil {
			s.hub.Broadcast("line", map[string]string{"text": "\nError: " + err.Error() + "\n"})
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true})
}

func (s *appServer) voiceAvailable() bool {
	if s.agent == nil {
		return false
	}
	c := s.agent.Cfg.ExternalClient
	if c == nil {
		c = s.agent.Client
	}
	return c != nil && strings.TrimSpace(c.APIKey) != ""
}

func (s *appServer) handleTranscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	if !s.voiceAvailable() {
		http.Error(w, "transcription unavailable (no OpenAI credentials)", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseMultipartForm(28 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	file, hdr, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "audio file required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	ext := filepath.Ext(hdr.Filename)
	if ext == "" {
		ext = ".webm"
	}
	tmp, err := os.CreateTemp("", "keimling-ui-*"+ext)
	if err != nil {
		http.Error(w, "temp file", http.StatusInternalServerError)
		return
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := io.Copy(tmp, io.LimitReader(file, 25<<20)); err != nil {
		tmp.Close()
		http.Error(w, "read audio", http.StatusBadRequest)
		return
	}
	if err := tmp.Close(); err != nil {
		http.Error(w, "save audio", http.StatusInternalServerError)
		return
	}
	client := s.agent.Cfg.ExternalClient
	if client == nil {
		client = s.agent.Client
	}
	model := s.agent.Cfg.VoiceModel
	if model == "" {
		model = "gpt-transcribe"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	text, err := client.Transcribe(ctx, path, model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"text": text})
}

func (s *appServer) handleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil || s.agent.ApprovalCh == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Allow bool `json:"allow"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	select {
	case s.agent.ApprovalCh <- body.Allow:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	default:
		http.Error(w, "no pending approval", http.StatusConflict)
	}
}

func (s *appServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch := s.hub.Subscribe(64)
	defer s.hub.Unsubscribe(ch)

	fmt.Fprintf(w, "event: ready\ndata: {\"ok\":true}\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, marshalEventData(ev.Data))
			flusher.Flush()
		}
	}
}

func (s *appServer) isBusy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.busy
}

func (s *appServer) requireSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := accessSecretFromRequest(r)
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.secret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func accessSecretFromRequest(r *http.Request) string {
	if h := r.Header.Get("X-Keimling-Access-Secret"); h != "" {
		return h
	}
	auth := r.Header.Get("Authorization")
	if len(auth) >= 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func sessionTranscript(history []json.RawMessage, limit int) []string {
	if limit <= 0 {
		limit = 100
	}
	var lines []string
	for _, raw := range history {
		var item struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content any    `json:"content"`
			Name    string `json:"name"`
			Output  string `json:"output"`
		}
		if json.Unmarshal(raw, &item) != nil {
			continue
		}
		switch {
		case item.Role == "user":
			if s, ok := item.Content.(string); ok && strings.TrimSpace(s) != "" {
				lines = append(lines, "› "+s)
			}
		case item.Type == "message":
			if text := messageText(item.Content); text != "" {
				lines = append(lines, text)
			}
		case item.Type == "function_call":
			lines = append(lines, "⚙ "+item.Name)
		case item.Type == "function_call_output":
			if strings.TrimSpace(item.Output) != "" {
				lines = append(lines, truncate(item.Output, 400))
			}
		}
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines
}

func messageText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var b strings.Builder
		for _, part := range v {
			m, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if m["type"] == "output_text" {
				if t, ok := m["text"].(string); ok {
					b.WriteString(t)
				}
			}
		}
		return b.String()
	default:
		return ""
	}
}
