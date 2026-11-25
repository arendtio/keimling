package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// approvalItem is one question waiting for the user, gathered from a live worker.
type approvalItem struct {
	Worker   string `json:"worker"`
	Kind     string `json:"kind"`
	Ref      string `json:"ref"`
	Name     string `json:"name"`
	ThreadID string `json:"thread_id"`
	Tool     string `json:"tool"`
	Reason   string `json:"reason"`
	Command  string `json:"command"`
	Preview  string `json:"preview"`
}

func (h *Host) handleApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"approvals": h.listApprovals(r.Context())})
}

func (h *Host) handleApprovalDecide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Worker   string `json:"worker"`
		ThreadID string `json:"thread_id"`
		Allow    bool   `json:"allow"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || strings.TrimSpace(body.Worker) == "" {
		http.Error(w, "worker required", http.StatusBadRequest)
		return
	}
	h.mu.Lock()
	worker := h.workers[strings.TrimSpace(body.Worker)]
	h.mu.Unlock()
	if worker == nil || worker.Handler == nil {
		http.Error(w, "worker is not running", http.StatusNotFound)
		return
	}
	payload, _ := json.Marshal(map[string]any{"allow": body.Allow, "thread_id": strings.TrimSpace(body.ThreadID)})
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	res, err := h.callWorker(ctx, worker, http.MethodPost, "/api/approve", string(payload))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(res.Body, 1<<16))
}

func (h *Host) listApprovals(ctx context.Context) []approvalItem {
	h.mu.Lock()
	workers := make([]*hostWorker, 0, len(h.workers))
	for _, w := range h.workers {
		if w != nil && w.Handler != nil {
			workers = append(workers, w)
		}
	}
	h.mu.Unlock()
	out := []approvalItem{}
	for _, w := range workers {
		out = append(out, h.workerApprovals(ctx, w)...)
	}
	return out
}

func (h *Host) workerApprovals(ctx context.Context, w *hostWorker) []approvalItem {
	if w == nil {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	res, err := h.callWorker(cctx, w, http.MethodGet, "/api/approvals", "")
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil
	}
	var body struct {
		Approvals []map[string]any `json:"approvals"`
	}
	if json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&body) != nil {
		return nil
	}
	name := w.Ref
	if w.Kind == "agent" {
		name = agentInstanceDisplayName(w.Ref)
	}
	out := make([]approvalItem, 0, len(body.Approvals))
	for _, p := range body.Approvals {
		if p == nil {
			continue
		}
		item := approvalItem{
			Worker:   w.Key,
			Kind:     w.Kind,
			Ref:      w.Ref,
			Name:     name,
			ThreadID: approvalString(p, "thread_id"),
			Tool:     approvalString(p, "name"),
			Reason:   approvalString(p, "reason"),
			Command:  approvalString(p, "command"),
			Preview:  approvalString(p, "preview"),
		}
		if item.Tool == "" {
			item.Tool = "tool"
		}
		out = append(out, item)
	}
	return out
}

func approvalString(p map[string]any, key string) string {
	s, _ := p[key].(string)
	return strings.TrimSpace(s)
}
