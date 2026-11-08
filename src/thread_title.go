package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// maybeGenerateThreadTitle names a provisional thread from the first user message.
// Uses the fast delegate model when set, otherwise the compaction/main model.
func (a *Agent) maybeGenerateThreadTitle(ctx context.Context, seed string) {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) != "" || a.Cfg.WorkspaceID == "" || !isWorkspaceThreadID(a.Cfg.ThreadID) {
		return
	}
	seed = strings.TrimSpace(seed)
	if seed == "" {
		return
	}
	meta, err := readThreadMeta(a.Cfg.WorkspaceID, a.Cfg.ThreadID)
	if err == nil && !isProvisionalThreadTitle(meta.Title) {
		return
	}
	model := strings.TrimSpace(a.Cfg.DelegateFast)
	if model == "" {
		model = a.compactionModel()
	}
	if a.Client == nil || model == "" {
		return
	}
	askCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	block := "User request:\n" + truncate(seed, 800)
	if strings.HasPrefix(seed, "User: ") {
		block = "Conversation:\n" + truncate(seed, 1200)
	}
	raw, err := a.Client.compactionAsk(askCtx, model,
		`Write a short thread title for this chat (3–8 words). Do NOT copy or lightly paraphrase the user message. Prefer a topic label (e.g. "PTY deadlock fix"). Title only — no quotes, no punctuation fluff.`,
		block, 64)
	if err != nil {
		return
	}
	title := cleanGeneratedTitle(raw)
	if title == "" || isProvisionalThreadTitle(title) {
		return
	}
	wrote := false
	err = updateWorkspaceThreadMeta(a.Cfg.WorkspaceID, a.Cfg.ThreadID, func(meta *ThreadMeta) error {
		if !isProvisionalThreadTitle(meta.Title) {
			return errThreadMetaSkip
		}
		meta.Title = title
		wrote = true
		return nil
	})
	if err != nil || !wrote {
		return
	}
	a.emitJSON("thread", map[string]any{
		"id":           a.Cfg.ThreadID,
		"title":        title,
		"workspace_id": a.Cfg.WorkspaceID,
	})
}

// A live call has no typed first message. Wait until the conversation has
// been going, then name the thread from what was said.
var (
	callTitleDelay = 20 * time.Second
	callTitleRetry = 5 * time.Second
)

func callTitleSeed(history []json.RawMessage) string {
	var b strings.Builder
	var userText bool
	for _, raw := range history {
		var item struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content any    `json:"content"`
		}
		if json.Unmarshal(raw, &item) != nil {
			continue
		}
		if item.Type == "function_call" || item.Type == "function_call_output" || item.Type == "reasoning" {
			continue
		}
		if item.Role != "user" && item.Role != "assistant" {
			continue
		}
		text := strings.TrimSpace(messageText(item.Content))
		if text == "" {
			continue
		}
		if item.Role == "user" {
			userText = true
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		if item.Role == "user" {
			b.WriteString("User: ")
		} else {
			b.WriteString("Assistant: ")
		}
		b.WriteString(text)
	}
	if !userText {
		return ""
	}
	return truncate(b.String(), 1200)
}

func (s *appServer) noteCallThreadTitle() {
	if s == nil || s.agent == nil || strings.TrimSpace(s.agent.Cfg.AgentInstanceID) != "" || s.agent.Cfg.WorkspaceID == "" {
		return
	}
	s.mu.Lock()
	if s.callTitleDone || s.callTitleRunning {
		s.mu.Unlock()
		return
	}
	s.callTitleRunning = true
	if s.callTitleStarted.IsZero() {
		s.callTitleStarted = time.Now()
	}
	started := s.callTitleStarted
	agent := s.agent
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			s.callTitleRunning = false
			s.mu.Unlock()
		}()
		if d := callTitleDelay - time.Since(started); d > 0 {
			time.Sleep(d)
		}
		for i := 0; i < 8; i++ {
			seed := callTitleSeed(agent.History)
			if seed == "" {
				time.Sleep(callTitleRetry)
				continue
			}
			agent.maybeGenerateThreadTitle(context.Background(), seed)
			s.mu.Lock()
			s.callTitleDone = true
			s.mu.Unlock()
			return
		}
	}()
}
