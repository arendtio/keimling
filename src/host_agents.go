package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

func (h *Host) handleAgentRecipes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := listAgentRecipes()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"recipes": list})
	case http.MethodPost:
		var body struct {
			Name         string `json:"name"`
			Instructions string `json:"instructions"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil && err != io.EOF {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		rec, err := createAgentRecipe(body.Name, body.Instructions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rec)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Host) handleAgentInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ws := strings.TrimSpace(r.URL.Query().Get("workspace"))
		if ws == "" {
			ws = h.focusWorkspaceID()
		}
		list, err := listAgentInstancesForWorkspace(ws)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.mu.Lock()
		workers := map[string]*hostWorker{}
		attached := ""
		if strings.HasPrefix(h.attached, "agent:") {
			attached = strings.TrimPrefix(h.attached, "agent:")
		}
		for key, w := range h.workers {
			if strings.HasPrefix(key, "agent:") {
				workers[strings.TrimPrefix(key, "agent:")] = w
			}
		}
		h.mu.Unlock()
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		busyBy := h.workersBusy(ctx, workers)
		out := make([]map[string]any, 0, len(list))
		for _, m := range list {
			busy := busyBy[m.ID]
			out = append(out, map[string]any{
				"id":                m.ID,
				"recipe_id":         m.RecipeID,
				"name":              m.Name,
				"workspaces":        m.Workspaces,
				"current_workspace": m.CurrentWorkspace,
				"status":            m.Status,
				"busy":              busy,
				"running":           busy, // alias: green only while executing
				"attached":          m.ID == attached,
				"updated_at":        m.UpdatedAt,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"instances": out, "attached": attached})
	case http.MethodPost:
		var body struct {
			RecipeID    string `json:"recipe_id"`
			Name        string `json:"name"`
			WorkspaceID string `json:"workspace_id"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		wsID := strings.TrimSpace(body.WorkspaceID)
		if wsID == "" {
			wsID = h.focusWorkspaceID()
		}
		if wsID == "" {
			http.Error(w, "workspace_id required", http.StatusBadRequest)
			return
		}
		recipeID := strings.TrimSpace(body.RecipeID)
		if recipeID == "" {
			http.Error(w, "recipe_id (definition) required", http.StatusBadRequest)
			return
		}
		recipe, err := readAgentRecipe(recipeID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		inst, err := createAgentInstance(recipe, wsID, body.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.mu.Lock()
		h.focusWorkspace = wsID
		h.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if err := h.AttachAgent(ctx, inst.ID); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "instance": inst, "attached": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Host) handleAgentAttach(w http.ResponseWriter, r *http.Request) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := h.AttachAgent(ctx, strings.TrimSpace(body.ID)); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": body.ID, "attached": true})
}

func (h *Host) handleAgentStop(w http.ResponseWriter, r *http.Request) {
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
	id := strings.TrimSpace(body.ID)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	worker, err := h.EnsureAgentWorker(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+worker.Addr+"/api/stop", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+worker.Secret)
	res, err := h.client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16))
	if m, e := readAgentInstanceMeta(id); e == nil {
		m.Status = agentStatusStopped
		_ = writeAgentInstanceMeta(m)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": id})
}

func (h *Host) handleAgentDelete(w http.ResponseWriter, r *http.Request) {
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
	id := strings.TrimSpace(body.ID)
	h.StopWorker(agentWorkerKey(id))
	if err := deleteAgentInstance(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": id})
}

func (h *Host) handleAgentRecipeUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Instructions string `json:"instructions"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	rec, err := updateAgentRecipe(body.ID, body.Name, body.Instructions, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rec)
}

func (h *Host) handleAgentRecipeDelete(w http.ResponseWriter, r *http.Request) {
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
	if err := deleteAgentRecipe(body.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": body.ID})
}

func (h *Host) handleAgentInstanceMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID   string `json:"id"`
		Text string `json:"text"`
		From string `json:"from"`
		Kind string `json:"kind"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(body.ID)
	text := strings.TrimSpace(body.Text)
	if id == "" || text == "" {
		http.Error(w, "id and text required", http.StatusBadRequest)
		return
	}
	kind := strings.TrimSpace(body.Kind)
	if kind == "" {
		kind = "message"
	}
	if err := enqueueAgentMessage(id, text, body.From, kind); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if _, err := h.EnsureAgentWorker(ctx, id); err != nil {
		// Queued on disk; worker may pick up later.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "queued": true, "worker": false})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "queued": true, "worker": true})
}

func (h *Host) handleAgentTeams(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := listAgentTeams()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"teams": list})
	case http.MethodPost:
		var body struct {
			Name          string   `json:"name"`
			DefinitionIDs []string `json:"definition_ids"`
			DefinitionIds []string `json:"definitionIds"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		ids := body.DefinitionIDs
		if len(ids) == 0 {
			ids = body.DefinitionIds
		}
		team, err := createAgentTeam(body.Name, ids)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(team)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Host) handleAgentTeamDelete(w http.ResponseWriter, r *http.Request) {
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
	if err := deleteAgentTeam(body.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": body.ID})
}

func (h *Host) handleAgentTeamSpawn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID          string `json:"id"`
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	wsID := strings.TrimSpace(body.WorkspaceID)
	if wsID == "" {
		wsID = h.focusWorkspaceID()
	}
	if wsID == "" {
		http.Error(w, "workspace_id required", http.StatusBadRequest)
		return
	}
	team, err := readAgentTeam(body.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created := make([]AgentInstanceMeta, 0, len(team.DefinitionIDs))
	for _, defID := range team.DefinitionIDs {
		recipe, err := readAgentRecipe(defID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		inst, err := createAgentInstance(recipe, wsID, recipe.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		created = append(created, inst)
	}
	// Allow teammates to message each other.
	for i := range created {
		for j := range created {
			if i == j {
				continue
			}
			_ = addAgentInstancePeer(created[i].ID, created[j].ID)
		}
	}
	h.mu.Lock()
	h.focusWorkspace = wsID
	h.mu.Unlock()
	if len(created) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		_ = h.AttachAgent(ctx, created[0].ID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "instances": created, "attached": len(created) > 0})
}
