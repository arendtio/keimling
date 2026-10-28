package main

import (
	"context"
	"strings"
	"time"
)

// maybeGenerateThreadTitle names a provisional thread from the first user message.
// Uses the fast delegate model when set, otherwise the compaction/main model.
func (a *Agent) maybeGenerateThreadTitle(ctx context.Context, seed string) {
	if a == nil || a.Cfg.WorkspaceID == "" || a.Cfg.ThreadID == "" {
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
	raw, err := a.Client.compactionAsk(askCtx, model,
		`Write a short thread title for this chat (3–8 words). Do NOT copy or lightly paraphrase the user message. Prefer a topic label (e.g. "PTY deadlock fix"). Title only — no quotes, no punctuation fluff.`,
		"User request:\n"+truncate(seed, 800), 64)
	if err != nil {
		return
	}
	title := cleanGeneratedTitle(raw)
	if title == "" || isProvisionalThreadTitle(title) {
		return
	}
	meta.ID = a.Cfg.ThreadID
	meta.Title = title
	if err := writeThreadMeta(a.Cfg.WorkspaceID, meta); err != nil {
		return
	}
	a.emitJSON("thread", map[string]any{
		"id":           meta.ID,
		"title":        meta.Title,
		"workspace_id": a.Cfg.WorkspaceID,
	})
}
