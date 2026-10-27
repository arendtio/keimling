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

	mu     sync.Mutex
	busy   bool
	cancel context.CancelFunc
}

func newAppServer(opts appServerOpts) *appServer {
	s := &appServer{
		secret: opts.AccessSecret,
		agent:  opts.Agent,
		hub:    newEventHub(),
		mux:    http.NewServeMux(),
	}
	if s.agent != nil {
		s.agent.Emit = func(kind, text string) {
			switch kind {
			case "status":
				s.hub.Broadcast("status", map[string]string{"state": text})
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
	s.mux.Handle("/api/message", s.requireSecret(http.HandlerFunc(s.handleMessage)))
	s.mux.Handle("/api/newchat", s.requireSecret(http.HandlerFunc(s.handleNewChat)))
	s.mux.Handle("/api/settings", s.requireSecret(http.HandlerFunc(s.handleSettings)))
	s.mux.Handle("/api/approve", s.requireSecret(http.HandlerFunc(s.handleApprove)))
	s.mux.Handle("/api/transcribe", s.requireSecret(http.HandlerFunc(s.handleTranscribe)))
	s.mux.Handle("/events", s.requireSecret(http.HandlerFunc(s.handleEvents)))
	return s
}

func (s *appServer) Handler() http.Handler { return s.mux }

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
		"lines":         sessionTranscript(s.agent.History, 200),
		"messages":      sessionMessages(s.agent.History, 120),
		"chapters":      sessionChapters(path),
		"working_state": sessionWorkingState(path),
		"busy":          s.isBusy(),
		"voice":         s.voiceAvailable(),
	})
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
	BaseURL      string `json:"base_url,omitempty"`
	Model        string `json:"model"`
	Mode         string `json:"mode"`
	Reasoning    string `json:"reasoning"`
	Web          *bool  `json:"web,omitempty"`
	Autopilot    *bool  `json:"autopilot,omitempty"`
	MaxSteps     *int   `json:"max_steps,omitempty"`
	ContextLimit *int   `json:"context_limit,omitempty"`
	VoiceModel   string `json:"voice_model,omitempty"`
}

func (s *appServer) settingsSnapshot() agentSettings {
	web, auto := s.agent.Cfg.Web, s.agent.Cfg.Autopilot
	steps, limit := s.agent.Cfg.MaxSteps, s.agent.Cfg.ContextLimit
	base := s.agent.Cfg.BaseURL
	if s.agent.Client != nil && strings.TrimSpace(s.agent.Client.BaseURL) != "" {
		base = s.agent.Client.BaseURL
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
	return nil
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
