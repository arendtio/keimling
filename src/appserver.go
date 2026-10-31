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

	mu                    sync.Mutex
	busy                  bool
	cancel                context.CancelFunc
	stopRequested         bool
	skillCandidates       []SkillCandidate
	skillPreviews         *skillPreviewHub
	pendingProviders      []modelProvider
	pendingActiveProvider string
	budgetCache           providerBudgetCache
}

func newAppServer(opts appServerOpts) *appServer {
	s := &appServer{
		secret:        opts.AccessSecret,
		agent:         opts.Agent,
		hub:           newEventHub(),
		mux:           http.NewServeMux(),
		skillPreviews: newSkillPreviewHub(),
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
			case "message", "tool":
				var payload map[string]any
				if json.Unmarshal([]byte(text), &payload) == nil {
					s.hub.Broadcast(kind, payload)
					return
				}
			case "approval":
				var payload map[string]any
				if json.Unmarshal([]byte(text), &payload) == nil {
					s.hub.Broadcast("approval", payload)
					return
				}
				s.hub.Broadcast("line", map[string]string{"text": text})
			case "artifact":
				var payload map[string]any
				if json.Unmarshal([]byte(text), &payload) == nil {
					s.hub.Broadcast("artifact", payload)
					return
				}
				s.hub.Broadcast("line", map[string]string{"text": text})
			case "compaction":
				var payload map[string]any
				if json.Unmarshal([]byte(text), &payload) == nil {
					s.hub.Broadcast("compaction", payload)
					return
				}
				s.hub.Broadcast("line", map[string]string{"text": text})
			case "thread":
				var payload map[string]any
				if json.Unmarshal([]byte(text), &payload) == nil {
					s.hub.Broadcast("thread", payload)
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
		if strings.TrimSpace(s.agent.Cfg.AgentInstanceID) != "" {
			go s.agentWakeLoop()
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
	s.mux.Handle("/api/workspaces", s.requireSecret(http.HandlerFunc(s.handleWorkspaces)))
	s.mux.Handle("/api/workspaces/open", s.requireSecret(http.HandlerFunc(s.handleWorkspaceOpen)))
	s.mux.Handle("/api/workspaces/switch", s.requireSecret(http.HandlerFunc(s.handleWorkspaceSwitch)))
	s.mux.Handle("/api/threads", s.requireSecret(http.HandlerFunc(s.handleThreads)))
	s.mux.Handle("/api/threads/switch", s.requireSecret(http.HandlerFunc(s.handleThreadSwitch)))
	s.mux.Handle("/api/threads/delete", s.requireSecret(http.HandlerFunc(s.handleThreadDelete)))
	s.mux.Handle("/api/threads/autopilot", s.requireSecret(http.HandlerFunc(s.handleThreadAutopilot)))
	s.mux.Handle("/api/context", s.requireSecret(http.HandlerFunc(s.handleContext)))
	s.mux.Handle("/api/compact", s.requireSecret(http.HandlerFunc(s.handleCompact)))
	s.mux.Handle("/api/skills", s.requireSecret(http.HandlerFunc(s.handleSkills)))
	s.mux.Handle("/api/skills/add", s.requireSecret(http.HandlerFunc(s.handleSkillsAdd)))
	s.mux.Handle("/api/skills/search", s.requireSecret(http.HandlerFunc(s.handleSkillsSearch)))
	s.mux.Handle("/api/skills/preview", s.requireSecret(http.HandlerFunc(s.handleSkillsPreview)))
	s.mux.Handle("/api/skills/preview/file", s.requireSecret(http.HandlerFunc(s.handleSkillsPreviewFile)))
	s.mux.Handle("/api/skills/review", s.requireSecret(http.HandlerFunc(s.handleSkillsReview)))
	s.mux.Handle("/api/skills/install", s.requireSecret(http.HandlerFunc(s.handleSkillsInstall)))
	s.mux.Handle("/api/skills/remove", s.requireSecret(http.HandlerFunc(s.handleSkillsRemove)))
	s.mux.Handle("/api/skills/check-update", s.requireSecret(http.HandlerFunc(s.handleSkillsCheckUpdate)))
	s.mux.Handle("/api/skills/update", s.requireSecret(http.HandlerFunc(s.handleSkillsUpdate)))
	s.mux.Handle("/api/mcp", s.requireSecret(http.HandlerFunc(s.handleMCP)))
	s.mux.Handle("/api/mcp/add", s.requireSecret(http.HandlerFunc(s.handleMCPAdd)))
	s.mux.Handle("/api/mcp/remove", s.requireSecret(http.HandlerFunc(s.handleMCPRemove)))
	s.mux.Handle("/api/mcp/reload", s.requireSecret(http.HandlerFunc(s.handleMCPReload)))
	s.mux.Handle("/api/artifacts", s.requireSecret(http.HandlerFunc(s.handleArtifacts)))
	s.mux.Handle("/api/message", s.requireSecret(http.HandlerFunc(s.handleMessage)))
	s.mux.Handle("/api/stop", s.requireSecret(http.HandlerFunc(s.handleStop)))
	s.mux.Handle("/api/newchat", s.requireSecret(http.HandlerFunc(s.handleNewChat)))
	s.mux.Handle("/api/settings", s.requireSecret(http.HandlerFunc(s.handleSettings)))
	s.mux.Handle("/api/settings/model-catalog", s.requireSecret(http.HandlerFunc(s.handleModelCatalog)))
	s.mux.Handle("/api/settings/budget", s.requireSecret(http.HandlerFunc(s.handleProviderBudget)))
	s.mux.Handle("/api/gui-prefs", s.requireSecret(http.HandlerFunc(s.handleGuiPrefs)))
	s.mux.Handle("/api/client-connections", s.requireSecret(http.HandlerFunc(s.handleClientConnections)))
	s.mux.Handle("/api/credentials", s.requireSecret(http.HandlerFunc(s.handleCredentials)))
	s.mux.Handle("/api/credentials/unlock", s.requireSecret(http.HandlerFunc(s.handleCredentialsUnlock)))
	s.mux.Handle("/api/credentials/", s.requireSecret(http.HandlerFunc(s.handleCredentialsItem)))
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
		"model":             s.agent.Cfg.Model,
		"mode":              s.agent.Cfg.Mode,
		"reasoning":         s.agent.Cfg.Reasoning,
		"web":               s.agent.Cfg.Web,
		"autopilot":         s.agent.Cfg.Autopilot,
		"max_steps":         s.agent.Cfg.MaxSteps,
		"context_limit":     s.agent.Cfg.ContextLimit,
		"voice_model":       s.agent.Cfg.VoiceModel,
		"workspace_id":      s.agent.Cfg.WorkspaceID,
		"workspace_root":    s.agent.Cfg.WorkspaceRoot,
		"thread_id":         s.agent.Cfg.ThreadID,
		"agent_instance_id": s.agent.Cfg.AgentInstanceID,
		"lines":             sessionTranscript(uiHistory, 200),
		"messages":          sessionMessages(uiHistory, 120),
		"chapters":          sessionChapters(path),
		"working_state":     sessionWorkingState(path),
		"busy":              s.isBusy(),
		"voice":             s.voiceAvailable(),
		"state":             s.agentState(),
		"budget":            s.budgetSnapshot(r.Context(), false),
	})
}

func (s *appServer) agentState() string {
	if s.isBusy() {
		return "running"
	}
	return "idle"
}

func (s *appServer) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := listWorkspaces()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cur := ""
	if s.agent != nil {
		cur = s.agent.Cfg.WorkspaceID
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"current":    cur,
		"workspaces": list,
	})
}

func (s *appServer) handleWorkspaceOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	ws, err := ensureWorkspaceForRoot(body.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.respondWorkspaceSwitch(w, ws.ID)
}

func (s *appServer) handleWorkspaceSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if _, err := readWorkspaceMeta(strings.TrimSpace(body.ID)); err != nil {
		http.Error(w, "unknown workspace", http.StatusNotFound)
		return
	}
	s.respondWorkspaceSwitch(w, strings.TrimSpace(body.ID))
}

func (s *appServer) respondWorkspaceSwitch(w http.ResponseWriter, id string) {
	cfgPath := ""
	if s.agent != nil {
		cfgPath = s.agent.Cfg.ConfigPath
		if id == s.agent.Cfg.WorkspaceID {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": id, "relaunch": false})
			return
		}
	}
	_ = mergeGuiPrefsWorkspace(cfgPath, id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"id":       id,
		"relaunch": true,
	})
}

func (s *appServer) handleThreads(w http.ResponseWriter, r *http.Request) {
	if s.agent == nil || s.agent.Cfg.WorkspaceID == "" {
		http.Error(w, "no workspace", http.StatusServiceUnavailable)
		return
	}
	if aid := strings.TrimSpace(s.agent.Cfg.AgentInstanceID); aid != "" {
		s.handleAgentInstanceThreads(w, r, aid)
		return
	}
	switch r.Method {
	case http.MethodGet:
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		var (
			list []ThreadMeta
			err  error
		)
		if q != "" {
			list, err = searchThreads(s.agent.Cfg.WorkspaceID, q)
		} else {
			list, err = listThreads(s.agent.Cfg.WorkspaceID)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Annotate active thread with live agent state.
		state := "idle"
		s.mu.Lock()
		if s.busy {
			state = "running"
		}
		s.mu.Unlock()
		for i := range list {
			if list[i].ID == s.agent.Cfg.ThreadID {
				list[i].Active = true
			} else {
				list[i].Active = false
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspace_id": s.agent.Cfg.WorkspaceID,
			"current":      s.agent.Cfg.ThreadID,
			"state":        state,
			"query":        q,
			"threads":      list,
		})
	case http.MethodPost:
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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":        true,
			"thread_id": s.agent.Cfg.ThreadID,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *appServer) handleAgentInstanceThreads(w http.ResponseWriter, r *http.Request, instanceID string) {
	switch r.Method {
	case http.MethodGet:
		state := "idle"
		s.mu.Lock()
		if s.busy {
			state = "running"
		}
		s.mu.Unlock()
		title := "Main"
		if m, err := readAgentInstanceMeta(instanceID); err == nil && strings.TrimSpace(m.Name) != "" {
			title = m.Name
		}
		list := []ThreadMeta{{
			ID:     "main",
			Title:  title,
			Active: true,
		}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspace_id":      s.agent.Cfg.WorkspaceID,
			"agent_instance_id": instanceID,
			"current":           "main",
			"state":             state,
			"query":             "",
			"threads":           list,
		})
	case http.MethodPost:
		http.Error(w, "agent instances use a fixed main thread", http.StatusBadRequest)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *appServer) handleThreadSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	if strings.TrimSpace(s.agent.Cfg.AgentInstanceID) != "" {
		http.Error(w, "agent instances use a fixed main thread", http.StatusBadRequest)
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		http.Error(w, "agent busy", http.StatusConflict)
		return
	}
	s.mu.Unlock()
	if err := s.agent.SwitchThread(strings.TrimSpace(body.ID)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "thread_id": s.agent.Cfg.ThreadID})
}

func (s *appServer) handleThreadDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	if strings.TrimSpace(s.agent.Cfg.AgentInstanceID) != "" {
		http.Error(w, "delete the agent instance instead", http.StatusBadRequest)
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		http.Error(w, "agent busy", http.StatusConflict)
		return
	}
	s.mu.Unlock()
	if err := s.agent.DeleteThread(strings.TrimSpace(body.ID)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":        true,
		"thread_id": s.agent.Cfg.ThreadID,
	})
}

func (s *appServer) handleThreadAutopilot(w http.ResponseWriter, r *http.Request) {
	if s.agent == nil || strings.TrimSpace(s.agent.Cfg.WorkspaceID) == "" || strings.TrimSpace(s.agent.Cfg.ThreadID) == "" {
		http.Error(w, "no thread", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.writeThreadAutopilot(w)
	case http.MethodPut:
		var body struct {
			Instructions *string `json:"instructions"`
			Reset        *bool   `json:"reset"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if body.Reset != nil && *body.Reset {
			if err := s.agent.resetThreadAutopilotInstructions(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else if body.Instructions != nil {
			if err := s.agent.saveThreadAutopilotInstructions(*body.Instructions); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "instructions required", http.StatusBadRequest)
			return
		}
		s.writeThreadAutopilot(w)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *appServer) writeThreadAutopilot(w http.ResponseWriter) {
	text, custom, source := s.agent.threadAutopilotView()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"instructions": text,
		"custom":       custom,
		"source":       source,
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

func (s *appServer) handleCompact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Guidance string `json:"guidance"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body)
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

	s.hub.Broadcast("status", map[string]string{"state": "compacting"})
	go func() {
		defer func() {
			s.mu.Lock()
			s.busy = false
			s.cancel = nil
			s.mu.Unlock()
			s.hub.Broadcast("status", map[string]string{"state": "idle"})
		}()
		if err := s.agent.compactContext(ctx, "hard", body.Guidance); err != nil && ctx.Err() == nil {
			s.hub.Broadcast("compaction", map[string]any{
				"phase":   "error",
				"message": err.Error(),
				"done":    true,
				"error":   err.Error(),
			})
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true})
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

func (s *appServer) handleArtifacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"artifacts": s.agent.ensureArtifacts().List()})
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

func (s *appServer) resolveSkillCandidateURL(candidateID, fallbackURL string) string {
	installURL := strings.TrimSpace(fallbackURL)
	if installURL == "" && candidateID != "" {
		s.mu.Lock()
		cands := s.skillCandidates
		s.mu.Unlock()
		for _, c := range cands {
			if c.ID == candidateID {
				return c.SourceURL
			}
		}
	}
	return installURL
}

func (s *appServer) handleSkillsPreview(w http.ResponseWriter, r *http.Request) {
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
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	sourceURL := s.resolveSkillCandidateURL(body.CandidateID, body.URL)
	if sourceURL == "" {
		http.Error(w, "url or candidate_id required", http.StatusBadRequest)
		return
	}
	prev, err := StageSkillPreview(r.Context(), s.skillPreviews, sourceURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"preview": prev})
}

func (s *appServer) handleSkillsPreviewFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	prev := s.skillPreviews.Get(id)
	if prev == nil {
		http.Error(w, "preview not found", http.StatusNotFound)
		return
	}
	text, err := ReadSkillPreviewFile(prev, path, skillFileCap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"path": path, "content": text})
}

func (s *appServer) handleSkillsReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		PreviewID string `json:"preview_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	prev := s.skillPreviews.Get(body.PreviewID)
	if prev == nil {
		http.Error(w, "preview not found", http.StatusNotFound)
		return
	}
	client := s.agent.Client
	if client == nil {
		http.Error(w, "model client unavailable", http.StatusServiceUnavailable)
		return
	}
	model := strings.TrimSpace(s.agent.Cfg.ReviewModel)
	if model == "" {
		model = s.agent.Cfg.Model
	}
	review, err := ReviewSkillPreview(r.Context(), client, model, prev)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"review": review, "preview": prev})
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
		PreviewID   string `json:"preview_id"`
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
	cwd, _ := os.Getwd()
	var sk *SkillMeta
	var err error
	if pid := strings.TrimSpace(body.PreviewID); pid != "" {
		prev := s.skillPreviews.Get(pid)
		if prev == nil {
			http.Error(w, "preview not found", http.StatusNotFound)
			return
		}
		sk, err = InstallSkillFromPreview(prev, scope, cwd)
	} else {
		installURL := s.resolveSkillCandidateURL(body.CandidateID, body.URL)
		if installURL == "" {
			http.Error(w, "preview_id, url, or candidate_id required", http.StatusBadRequest)
			return
		}
		sk, err = InstallSkillFromURL(installURL, scope, cwd)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.agent.refreshSkills()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "skill": sk})
}

func (s *appServer) handleSkillsRemove(w http.ResponseWriter, r *http.Request) {
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
	cwd, _ := os.Getwd()
	if err := DeleteSkill(body.Name, cwd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.agent.refreshSkills()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "skills": s.agent.ensureSkills().List()})
}

func (s *appServer) handleSkillsCheckUpdate(w http.ResponseWriter, r *http.Request) {
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
	cwd, _ := os.Getwd()
	st, err := CheckSkillUpdate(r.Context(), body.Name, cwd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"check": st})
}

func (s *appServer) handleSkillsUpdate(w http.ResponseWriter, r *http.Request) {
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
	cwd, _ := os.Getwd()
	sk, st, err := UpdateSkill(r.Context(), body.Name, cwd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.agent.refreshSkills()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "skill": sk, "check": st, "skills": s.agent.ensureSkills().List()})
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
	BaseURL                    string          `json:"base_url,omitempty"`
	Model                      string          `json:"model"`
	Mode                       string          `json:"mode"`
	Reasoning                  string          `json:"reasoning"`
	Web                        *bool           `json:"web,omitempty"`
	Autopilot                  *bool           `json:"autopilot,omitempty"`
	AutopilotInstructions      *string         `json:"autopilot_instructions,omitempty"`
	AutopilotInstructionsReset *bool           `json:"autopilot_instructions_reset,omitempty"`
	MaxSteps                   *int            `json:"max_steps,omitempty"`
	ContextLimit               *int            `json:"context_limit,omitempty"`
	VoiceModel                 string          `json:"voice_model,omitempty"`
	APIKey                     *string         `json:"api_key,omitempty"`        // write-only
	OpenAIAPIKey               *string         `json:"openai_api_key,omitempty"` // write-only
	APIKeySet                  bool            `json:"api_key_set"`              // GET only
	OpenAIKeySet               bool            `json:"openai_api_key_set"`       // GET only
	ModelProviders             []modelProvider `json:"model_providers,omitempty"`
	ActiveModelProviderID      string          `json:"active_model_provider_id,omitempty"`
}

func (s *appServer) settingsSnapshot() map[string]any {
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
	providers, activeID := s.loadModelProvidersForSettings()
	globalText := s.agent.globalAutopilotInstructions()
	effectiveGlobal, _ := resolveAutopilotInstructions(globalText, "")
	globalCustom := strings.TrimSpace(globalText) != ""
	return map[string]any{
		"base_url":                      base,
		"model":                         s.agent.Cfg.Model,
		"mode":                          s.agent.Cfg.Mode,
		"reasoning":                     s.agent.Cfg.Reasoning,
		"web":                           web,
		"autopilot":                     auto,
		"autopilot_instructions":        effectiveGlobal,
		"autopilot_instructions_custom": globalCustom,
		"max_steps":                     steps,
		"context_limit":                 limit,
		"voice_model":                   s.agent.Cfg.VoiceModel,
		"api_key_set":                   providerKey != "",
		"openai_api_key_set":            openaiSet,
		"model_providers":               publicModelProviders(providers),
		"active_model_provider_id":      activeID,
	}
}

func (s *appServer) loadModelProvidersForSettings() ([]modelProvider, string) {
	path := strings.TrimSpace(s.agent.Cfg.ConfigPath)
	fc, err := readConfiguration(path)
	if err != nil {
		fc = fileConfig{}
	}
	// Seed from live agent if config empty.
	if strings.TrimSpace(fc.API.BaseURL) == "" {
		fc.API.BaseURL = s.agent.Cfg.BaseURL
	}
	if strings.TrimSpace(fc.API.Model) == "" {
		fc.API.Model = s.agent.Cfg.Model
	}
	if strings.TrimSpace(fc.API.Key) == "" {
		fc.API.Key = s.agent.Cfg.Key
	}
	fc.ensureModelProviders()
	// Prefer live key on active provider display without writing raw key to response.
	if p := modelProviderByID(fc.ModelProviders, fc.ActiveModelProviderID); p != nil {
		if strings.TrimSpace(p.APIKey) == "" && strings.TrimSpace(s.agent.Cfg.Key) != "" {
			p.APIKey = s.agent.Cfg.Key
		}
		if strings.TrimSpace(p.Reasoning) == "" {
			p.Reasoning = s.agent.Cfg.Reasoning
		}
	}
	return fc.ModelProviders, fc.ActiveModelProviderID
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
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
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
	resetInstructions := body.AutopilotInstructionsReset != nil && *body.AutopilotInstructionsReset
	if !resetInstructions && body.AutopilotInstructions != nil {
		if _, err := normalizeAutopilotInstructions(*body.AutopilotInstructions); err != nil {
			return err
		}
	}
	cfg := &s.agent.Cfg
	writeProviders := body.ModelProviders != nil
	s.pendingProviders = nil
	s.pendingActiveProvider = ""
	if writeProviders {
		out, active, err := s.resolveModelProviders(body)
		if err != nil {
			return err
		}
		s.pendingProviders = out
		s.pendingActiveProvider = active
		applyActiveProviderToAgent(s.agent, fileConfig{
			ModelProviders:        out,
			ActiveModelProviderID: active,
		})
		cfg = &s.agent.Cfg
	}
	if raw := strings.TrimSpace(body.BaseURL); raw != "" && !writeProviders {
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
	if model := strings.TrimSpace(body.Model); model != "" && !writeProviders {
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
	if resetInstructions {
		cfg.AutopilotInstructions = ""
	} else if body.AutopilotInstructions != nil {
		text, err := normalizeAutopilotInstructions(*body.AutopilotInstructions)
		if err != nil {
			return err
		}
		cfg.AutopilotInstructions = text
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
		// Active model context wins when curated; otherwise use the global fallback.
		modelLimit := 0
		if writeProviders {
			if ap := modelProviderByID(s.pendingProviders, s.pendingActiveProvider); ap != nil {
				if m := ap.activeModel(); m != nil {
					modelLimit = m.ContextLimit
				}
			}
		}
		if modelLimit < 1000 {
			cfg.ContextLimit = *body.ContextLimit
		}
	}
	if v := strings.TrimSpace(body.VoiceModel); v != "" {
		cfg.VoiceModel = v
	}
	writeProviderKey := body.APIKey != nil && !writeProviders
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
	writeInstructions := resetInstructions || body.AutopilotInstructions != nil
	if err := s.persistAgentConfig(writeProviderKey || writeProviders, writeOpenAIKey, writeInstructions); err != nil {
		return fmt.Errorf("persist config: %w", err)
	}
	if writeProviders {
		s.budgetCache.invalidate()
	}
	return nil
}

func (s *appServer) resolveModelProviders(body agentSettings) ([]modelProvider, string, error) {
	if len(body.ModelProviders) == 0 {
		return nil, "", fmt.Errorf("at least one model provider required")
	}
	existing, _ := s.loadModelProvidersForSettings()
	byID := map[string]modelProvider{}
	for _, p := range existing {
		byID[p.ID] = p
	}
	out := make([]modelProvider, 0, len(body.ModelProviders))
	for _, in := range body.ModelProviders {
		p := modelProvider{
			ID:            strings.TrimSpace(in.ID),
			Name:          strings.TrimSpace(in.Name),
			BaseURL:       strings.TrimSpace(in.BaseURL),
			Kind:          strings.TrimSpace(in.Kind),
			Model:         strings.TrimSpace(in.Model),
			Reasoning:     strings.TrimSpace(in.Reasoning),
			ActiveModelID: strings.TrimSpace(in.ActiveModelID),
			Models:        in.Models,
		}
		if p.BaseURL == "" {
			return nil, "", fmt.Errorf("provider base_url required")
		}
		resolved, err := normalizeBaseURL(p.BaseURL)
		if err != nil {
			return nil, "", fmt.Errorf("provider %q: %w", p.Name, err)
		}
		p.BaseURL = resolved
		switch strings.ToLower(p.Kind) {
		case "", "auto", "openai", "openrouter", "litellm", "local":
			if p.Kind == "" {
				p.Kind = "auto"
			}
		default:
			return nil, "", fmt.Errorf("provider %q: invalid kind %q", p.Name, p.Kind)
		}
		if prev, ok := byID[p.ID]; ok {
			p.APIKey = prev.APIKey
		}
		if k := strings.TrimSpace(in.APIKey); k != "" {
			p.APIKey = k
		}
		if p.ID == "" {
			p.ID = newModelProviderID()
		}
		if p.Name == "" {
			p.Name = p.BaseURL
		}
		cleaned := make([]providerModel, 0, len(p.Models))
		for _, m := range p.Models {
			nm, err := normalizeProviderModel(m)
			if err != nil {
				return nil, "", fmt.Errorf("provider %q: %w", p.Name, err)
			}
			cleaned = append(cleaned, nm)
		}
		p.Models = cleaned
		if p.Reasoning != "" {
			switch p.Reasoning {
			case "low", "medium", "high", "xhigh", "max":
			default:
				return nil, "", fmt.Errorf("provider %q: invalid reasoning", p.Name)
			}
		}
		p.ensureProviderModels()
		out = append(out, p)
	}
	active := strings.TrimSpace(body.ActiveModelProviderID)
	if active == "" || modelProviderByID(out, active) == nil {
		active = out[0].ID
	}
	return out, active, nil
}

func (s *appServer) configPath() string {
	if s.agent != nil {
		return s.agent.Cfg.ConfigPath
	}
	return ""
}

func (s *appServer) handleClientConnections(w http.ResponseWriter, r *http.Request) {
	path := s.configPath()
	switch r.Method {
	case http.MethodGet:
		f, err := readClientConnections(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(f.public())
	case http.MethodPut, http.MethodPatch:
		var body struct {
			Connections *[]connectionProfile `json:"connections"`
			ActiveID    *string              `json:"active_id"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := updateClientConnections(path, func(f *clientConnectionsFile) {
			if body.Connections != nil {
				prevByID := map[string]connectionProfile{}
				for _, c := range f.Connections {
					prevByID[c.ID] = c
				}
				next := *body.Connections
				for i := range next {
					if strings.TrimSpace(next[i].AccessSecret) != "" {
						continue
					}
					if prev, ok := prevByID[strings.TrimSpace(next[i].ID)]; ok && prev.AccessSecret != "" {
						next[i].AccessSecret = prev.AccessSecret
					}
				}
				f.Connections = next
			}
			if body.ActiveID != nil {
				f.ActiveID = strings.TrimSpace(*body.ActiveID)
			}
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f, err := readClientConnections(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(f.public())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *appServer) handleGuiPrefs(w http.ResponseWriter, r *http.Request) {
	path := s.configPath()
	switch r.Method {
	case http.MethodGet:
		p, err := readGuiPrefs(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"connect":              p.Connect,
			"access_secret":        p.AccessSecret,
			"connections":          p.publicConnections(),
			"active_connection_id": p.ActiveConnectionID,
			"bg_light":             p.BgLight,
			"bg_dark":              p.BgDark,
			"workspace_id":         p.WorkspaceID,
			"window_width":         p.WindowWidth,
			"window_height":        p.WindowHeight,
		})
	case http.MethodPut, http.MethodPatch:
		var body struct {
			Connect            *string              `json:"connect"`
			AccessSecret       *string              `json:"access_secret"`
			Connections        *[]connectionProfile `json:"connections"`
			ActiveConnectionID *string              `json:"active_connection_id"`
			BgLight            *string              `json:"bg_light"`
			BgDark             *string              `json:"bg_dark"`
			WorkspaceID        *string              `json:"workspace_id"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := updateGuiPrefs(path, func(p *guiPrefs) {
			if body.Connections != nil {
				prevByID := map[string]connectionProfile{}
				for _, c := range p.Connections {
					prevByID[c.ID] = c
				}
				next := *body.Connections
				for i := range next {
					if strings.TrimSpace(next[i].AccessSecret) != "" {
						continue
					}
					if prev, ok := prevByID[strings.TrimSpace(next[i].ID)]; ok && prev.AccessSecret != "" {
						next[i].AccessSecret = prev.AccessSecret
					}
				}
				p.Connections = next
			}
			if body.ActiveConnectionID != nil {
				p.ActiveConnectionID = strings.TrimSpace(*body.ActiveConnectionID)
			}
			if body.Connect != nil {
				p.Connect = strings.TrimSpace(*body.Connect)
				if p.Connect == "" {
					p.AccessSecret = ""
				} else if body.AccessSecret != nil {
					p.AccessSecret = strings.TrimSpace(*body.AccessSecret)
				}
			} else if body.AccessSecret != nil {
				p.AccessSecret = strings.TrimSpace(*body.AccessSecret)
			}
			if body.BgLight != nil {
				if c, ok := normalizeBgColor(*body.BgLight); ok {
					p.BgLight = c
				}
			}
			if body.BgDark != nil {
				if c, ok := normalizeBgColor(*body.BgDark); ok {
					p.BgDark = c
				}
			}
			if body.WorkspaceID != nil {
				p.WorkspaceID = strings.TrimSpace(*body.WorkspaceID)
			}
			normalizeConnectionProfiles(p)
			// If only legacy connect was patched without connections array, sync profiles.
			if body.Connections == nil && body.Connect != nil {
				_ = body
				connect := strings.TrimSpace(*body.Connect)
				secret := ""
				if body.AccessSecret != nil {
					secret = strings.TrimSpace(*body.AccessSecret)
				} else {
					secret = p.AccessSecret
				}
				if connect == "" {
					p.ActiveConnectionID = localConnectionID
				} else {
					matched := false
					for i := range p.Connections {
						c := &p.Connections[i]
						if c.Kind == connectionKindRemote && c.URL == connect {
							if body.AccessSecret != nil {
								c.AccessSecret = secret
							}
							p.ActiveConnectionID = c.ID
							matched = true
							break
						}
					}
					if !matched {
						id := newConnectionID(connectionKindRemote)
						p.Connections = append(p.Connections, connectionProfile{
							ID: id, Name: "Remote", Kind: connectionKindRemote, URL: connect, AccessSecret: secret,
						})
						p.ActiveConnectionID = id
					}
				}
				normalizeConnectionProfiles(p)
			}
		}); err != nil {
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

func (s *appServer) persistAgentConfig(writeProviderKey, writeOpenAIKey, writeAutopilotInstructions bool) error {
	path := strings.TrimSpace(s.agent.Cfg.ConfigPath)
	if path == "" {
		return nil
	}
	cfg := &s.agent.Cfg
	providers := s.pendingProviders
	active := s.pendingActiveProvider
	return updateConfiguration(path, func(fc *fileConfig) {
		if providers != nil {
			fc.ModelProviders = providers
			fc.ActiveModelProviderID = active
			fc.ensureModelProviders()
		}
		fc.API.BaseURL = cfg.BaseURL
		fc.API.Model = cfg.Model
		fc.API.Mode = cfg.Mode
		web := cfg.Web
		fc.Features.WebSearch = &web
		auto := cfg.Autopilot
		fc.Features.Autopilot = &auto
		if writeAutopilotInstructions {
			fc.Features.AutopilotInstructions = cfg.AutopilotInstructions
		}
		fc.Reasoning = cfg.Reasoning
		fc.VoiceModel = cfg.VoiceModel
		fc.MaxSteps = cfg.MaxSteps
		fc.ContextLimit = cfg.ContextLimit
		if writeProviderKey {
			fc.API.Key = cfg.Key
			fc.API.KeyFile = ""
			if p := modelProviderByID(fc.ModelProviders, fc.ActiveModelProviderID); p != nil && cfg.Key != "" {
				p.APIKey = cfg.Key
			}
		}
		if writeOpenAIKey {
			k := ""
			if cfg.ExternalClient != nil {
				k = strings.TrimSpace(cfg.ExternalClient.APIKey)
			}
			fc.OpenAI.Key = k
			fc.OpenAI.KeyFile = ""
		}
		if len(fc.ModelProviders) > 0 {
			fc.mirrorActiveModelProvider()
		}
	})
}

func (s *appServer) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	s.mu.Lock()
	if !s.busy || s.cancel == nil {
		s.mu.Unlock()
		http.Error(w, "agent not running", http.StatusConflict)
		return
	}
	cancel := s.cancel
	s.stopRequested = true
	s.mu.Unlock()

	// A decision queued before/while stopping must not auto-approve the next run.
	drainApproval(s.agent)
	cancel()
	// Catch a decision that slipped in between the drain and cancel.
	drainApproval(s.agent)
	s.hub.Broadcast("line", map[string]string{"text": "\n[stop] Stopped by user.\n"})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "stopped": true})
}

func drainApproval(a *Agent) {
	if a == nil || a.ApprovalCh == nil {
		return
	}
	for {
		select {
		case <-a.ApprovalCh:
		default:
			return
		}
	}
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
	queued, err := s.acceptAgentText(text, "user", "message")
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true, "queued": queued})
}

func (s *appServer) acceptAgentText(text, from, kind string) (queued bool, err error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return false, fmt.Errorf("text required")
	}
	inst := ""
	if s.agent != nil {
		inst = strings.TrimSpace(s.agent.Cfg.AgentInstanceID)
	}
	if s.startAgentRun(text) {
		return false, nil
	}
	// Busy: steer the in-flight run (same idea as CLI typing while working).
	s.mu.Lock()
	agent := s.agent
	s.mu.Unlock()
	if agent != nil && agent.Control != nil {
		select {
		case agent.Control.events <- controlEvent{kind: "steer", text: text}:
			return true, nil
		default:
			return false, fmt.Errorf("steering queue is full")
		}
	}
	// Durable wake queue for agent instances (alarms / multi-agent).
	if inst == "" {
		return false, fmt.Errorf("agent busy")
	}
	if err := enqueueAgentMessage(inst, text, from, kind); err != nil {
		return false, err
	}
	return true, nil
}

func (s *appServer) startAgentRun(text string) bool {
	s.mu.Lock()
	if s.busy || s.agent == nil {
		s.mu.Unlock()
		return false
	}
	s.busy = true
	s.stopRequested = false
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	agent := s.agent
	control := newTaskControl(cancel)
	agent.Control = control
	s.mu.Unlock()

	s.hub.Broadcast("line", map[string]string{"text": "\n› " + text + "\n"})
	s.hub.Broadcast("status", map[string]string{"state": "running"})

	go func() {
		defer func() {
			s.mu.Lock()
			s.busy = false
			s.cancel = nil
			s.stopRequested = false
			if s.agent != nil {
				s.agent.Control = nil
			}
			s.mu.Unlock()
			s.hub.Broadcast("status", map[string]string{"state": "idle"})
			s.pumpAgentWakes()
		}()
		err := agent.Run(ctx, text, bufio.NewReader(strings.NewReader("")))
		if err != nil {
			if ctx.Err() != nil {
				// A cancelled run can leave a dangling function_call (e.g. while
				// awaiting approval) which the Responses API rejects on the next
				// message. Mark such calls interrupted before releasing busy.
				_ = agent.repairInterruptedCalls()
			} else {
				s.hub.Broadcast("line", map[string]string{"text": "\nError: " + err.Error() + "\n"})
			}
		}
	}()
	return true
}

func (s *appServer) agentWakeLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.pumpAgentWakes()
	}
}

func (s *appServer) pumpAgentWakes() {
	if s.agent == nil {
		return
	}
	inst := strings.TrimSpace(s.agent.Cfg.AgentInstanceID)
	if inst == "" {
		return
	}
	s.mu.Lock()
	busy := s.busy
	s.mu.Unlock()
	if busy {
		return
	}
	if due, err := dueAgentAlarms(inst, time.Now()); err == nil {
		for _, al := range due {
			_ = enqueueAgentMessage(inst, al.Message, "alarm", "alarm")
		}
	}
	items, err := dequeueAgentMessages(inst)
	if err != nil || len(items) == 0 {
		return
	}
	first := items[0]
	for _, it := range items[1:] {
		_ = enqueueAgentMessage(inst, it.Text, it.From, it.Kind)
	}
	if !s.startAgentRun(formatQueueWake(first)) {
		// Still busy (race) — put the head back so nothing is lost.
		_ = enqueueAgentMessage(inst, first.Text, first.From, first.Kind)
	}
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
	s.mu.Lock()
	stopping := s.stopRequested
	s.mu.Unlock()
	if stopping {
		http.Error(w, "no pending approval", http.StatusConflict)
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
