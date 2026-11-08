package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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
		_ = json.NewEncoder(w).Encode(map[string]any{"recipes": list, "tools": agentToolCatalog(), "groups": agentToolGroups(), "review_prompt_factory": defaultPolicy})
	case http.MethodPost:
		var body struct {
			Name              string    `json:"name"`
			Instructions      string    `json:"instructions"`
			Model             *string   `json:"model"`
			Tools             *[]string `json:"tools"`
			Mode              string    `json:"mode"`
			ReviewPrompt      *string   `json:"review_prompt"`
			ReviewPromptReset *bool     `json:"review_prompt_reset"`
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
		if body.Model != nil {
			rec, err = setRecipeModel(rec.ID, *body.Model)
			if err != nil {
				_ = deleteAgentRecipe(rec.ID)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if body.Tools != nil {
			rec, err = setRecipeTools(rec.ID, *body.Tools)
			if err != nil {
				_ = deleteAgentRecipe(rec.ID)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		reset := body.ReviewPromptReset != nil && *body.ReviewPromptReset
		if body.Mode != "" || body.ReviewPrompt != nil || reset {
			rec, err = setRecipeApproval(rec.ID, body.Mode, body.ReviewPrompt, reset)
			if err != nil {
				_ = deleteAgentRecipe(rec.ID)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
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
		var list []AgentInstanceMeta
		var err error
		if r.URL.Query().Get("all") == "1" {
			list, err = listAgentInstances()
		} else {
			ws := strings.TrimSpace(r.URL.Query().Get("workspace"))
			if ws == "" {
				ws = h.focusWorkspaceID()
			}
			list, err = listAgentInstancesForWorkspace(ws)
		}
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
				"tools":             m.Tools,
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
			writeAttachErr(w, err)
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
		writeAttachErr(w, err)
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
	_ = setAgentWakesHeld(id, true)
	addr, secret, ok := h.liveAgentWorker(id)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": id, "running": false})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+addr+"/api/stop", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16))
	running := res.StatusCode/100 == 2
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": id, "running": running})
}

func (h *Host) handleAgentReset(w http.ResponseWriter, r *http.Request) {
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
	if _, err := readAgentInstanceMeta(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if addr, secret, ok := h.liveAgentWorker(id); ok {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+addr+"/api/instance/reset", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("Authorization", "Bearer "+secret)
		res, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer res.Body.Close()
		payload, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
		if res.StatusCode/100 != 2 {
			http.Error(w, strings.TrimSpace(string(payload)), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(res.StatusCode)
		_, _ = w.Write(payload)
		return
	}
	if err := resetAgentInstanceFiles(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": id})
}

func resetAgentInstanceFiles(id string) error {
	id = strings.TrimSpace(id)
	session := agentInstanceSessionPath(id)
	if st, err := os.Stat(session); err == nil && st.Size() > 0 {
		archive := fmt.Sprintf("%s.%s.bak", session, time.Now().Format("20060102-150405.000000000"))
		if err := os.Rename(session, archive); err != nil {
			return err
		}
	}
	_ = os.Remove(charterPath(session))
	_ = os.Remove(indexPath(session))
	_ = os.Remove(agentInstanceQueuePath(id))
	return setAgentWakesHeld(id, false)
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
		ID                string    `json:"id"`
		Name              string    `json:"name"`
		Instructions      string    `json:"instructions"`
		Model             *string   `json:"model"`
		Tools             *[]string `json:"tools"`
		Mode              string    `json:"mode"`
		ReviewPrompt      *string   `json:"review_prompt"`
		ReviewPromptReset *bool     `json:"review_prompt_reset"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	reset := body.ReviewPromptReset != nil && *body.ReviewPromptReset
	if reset && strings.TrimSpace(body.Name) == "" && body.Instructions == "" && body.Tools == nil && body.Model == nil {
		rec, err := setRecipeApproval(body.ID, body.Mode, nil, true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rec)
		return
	}
	rec, err := updateAgentRecipe(body.ID, body.Name, body.Instructions, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Model != nil {
		rec, err = setRecipeModel(body.ID, *body.Model)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if body.Tools != nil {
		rec, err = setRecipeTools(rec.ID, *body.Tools)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if body.Mode != "" || body.ReviewPrompt != nil || reset {
		rec, err = setRecipeApproval(rec.ID, body.Mode, body.ReviewPrompt, reset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
		ID        string `json:"id"`
		Text      string `json:"text"`
		From      string `json:"from"`
		FromName  string `json:"from_name"`
		Workspace string `json:"workspace"`
		Kind      string `json:"kind"`
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
	if kind == "message" || kind == "thread" {
		_ = setAgentWakesHeld(id, false)
	}
	if err := enqueueAgentQueueItem(id, AgentQueueItem{
		Text:      text,
		From:      body.From,
		FromName:  body.FromName,
		Workspace: body.Workspace,
		Kind:      kind,
	}); err != nil {
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

func (h *Host) handleAgentInstanceAccess(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		m, err := readAgentInstanceMeta(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ws, err := listWorkspaces()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		grantedWS := map[string]bool{}
		for _, id := range m.Workspaces {
			grantedWS[id] = true
		}
		workspaces := make([]map[string]any, 0, len(ws))
		for _, one := range ws {
			workspaces = append(workspaces, map[string]any{
				"id": one.ID, "name": one.Name, "root": one.Root, "granted": grantedWS[one.ID],
			})
		}
		rec, _ := readAgentRecipe(m.RecipeID)
		defAll := rec.ID == "" || rec.Tools == nil
		defAllow := map[string]bool{}
		if rec.Tools != nil {
			for _, name := range *rec.Tools {
				defAllow[name] = true
			}
		}
		follows := m.Tools == nil
		instAllow := map[string]bool{}
		if m.Tools != nil {
			for _, name := range *m.Tools {
				instAllow[name] = true
			}
		}
		tools := make([]map[string]any, 0, len(agentToolCatalog()))
		for _, t := range agentToolCatalog() {
			allowed := defAll || defAllow[t.Name]
			granted := allowed && (follows || instAllow[t.Name])
			tools = append(tools, map[string]any{
				"name": t.Name, "label": t.Label, "granted": granted, "allowed": allowed,
			})
		}
		var defTools *[]string
		if rec.ID != "" {
			defTools = rec.Tools
		}
		defAllMCP, defIDs := mcpGrantIDs(defTools)
		grantedAll, grantedIDs := defAllMCP, append([]string(nil), defIDs...)
		if !follows {
			grantedAll, grantedIDs = mcpGrantIDs(m.Tools)
			if grantedAll && !defAllMCP {
				grantedAll = false
				grantedIDs = append([]string(nil), defIDs...)
			}
			if !defAllMCP {
				allow := map[string]bool{}
				for _, id := range defIDs {
					allow[id] = true
				}
				kept := make([]string, 0, len(grantedIDs))
				for _, id := range grantedIDs {
					if allow[id] {
						kept = append(kept, id)
					}
				}
				grantedIDs = kept
			}
		}
		others, err := listAgentInstances()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		peers := make([]map[string]any, 0, len(others))
		for _, other := range others {
			if other.ID == m.ID {
				continue
			}
			peers = append(peers, map[string]any{
				"id": other.ID, "name": other.Name, "granted": agentsMayContact(m, other),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": m.ID, "name": m.Name, "follows": follows, "workspaces": workspaces, "tools": tools, "groups": agentToolGroups(),
			"peers":            peers,
			"model":            strings.TrimSpace(m.Model),
			"definition_model": strings.TrimSpace(rec.Model),
			"mcp": map[string]any{
				"allowed_all": defAllMCP,
				"allowed":     defIDs,
				"granted_all": grantedAll,
				"granted":     grantedIDs,
			},
		})
	case http.MethodPost:
		var body struct {
			ID         string    `json:"id"`
			Workspaces []string  `json:"workspaces"`
			Tools      *[]string `json:"tools"`
			Peers      *[]string `json:"peers"`
			Model      *string   `json:"model"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		if body.Model != nil {
			if err := validateKnownModel(h.config, *body.Model); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		m, err := setAgentInstanceAccess(body.ID, body.Workspaces, body.Tools)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.Peers != nil {
			m, err = setAgentInstancePeers(body.ID, *body.Peers)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if body.Model != nil {
			m, err = setAgentInstanceModel(body.ID, *body.Model)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "instance": m})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Host) wakeQueuedAgentsLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			h.wakeQueuedAgentsOnce()
		}
	}
}

func (h *Host) wakeQueuedAgentsOnce() {
	list, err := listAgentInstances()
	if err != nil {
		return
	}
	var ids []string
	h.mu.Lock()
	if h.waking == nil {
		h.waking = map[string]bool{}
	}
	for _, m := range list {
		if h.waking[m.ID] || m.WakesHeld || (!agentQueuePending(m.ID) && !agentAlarmDue(m.ID, time.Now()) && !agentWatchNeedsProcess(m.ID)) {
			continue
		}
		h.waking[m.ID] = true
		ids = append(ids, m.ID)
	}
	h.mu.Unlock()
	for _, id := range ids {
		go func(id string) {
			defer func() {
				h.mu.Lock()
				delete(h.waking, id)
				h.mu.Unlock()
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			_, _ = h.EnsureAgentWorker(ctx, id)
		}(id)
	}
}

func agentQueuePending(id string) bool {
	st, err := os.Stat(agentInstanceQueuePath(id))
	return err == nil && st.Size() > 0
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
	attached := false
	if len(created) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if err := h.AttachAgent(ctx, created[0].ID); err != nil {
			if !errors.Is(err, errHangUpFirst) {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
		} else {
			attached = true
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "instances": created, "attached": attached})
}
