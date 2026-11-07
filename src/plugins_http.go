package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func (s *appServer) applyPluginChange(r *http.Request) {
	if s == nil || s.agent == nil {
		return
	}
	s.agent.Skills = LoadSkillRegistry("")
	s.agent.reloadMCP(r.Context())
}

func (s *appServer) handlePlugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := listPluginStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"plugins": list})
}

func (s *appServer) handlePluginAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Location string `json:"location"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || strings.TrimSpace(body.Location) == "" {
		http.Error(w, "location required", http.StatusBadRequest)
		return
	}
	st, err := addPluginSource(body.Location)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.applyPluginChange(r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

func (s *appServer) handlePluginEnabled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID      string `json:"id"`
		Enabled *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" || body.Enabled == nil {
		http.Error(w, "id and enabled required", http.StatusBadRequest)
		return
	}
	st, err := setPluginEnabled(body.ID, *body.Enabled)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.applyPluginChange(r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

func (s *appServer) handlePluginRefresh(w http.ResponseWriter, r *http.Request) {
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
	st, err := refreshPlugin(body.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.applyPluginChange(r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

func (s *appServer) handlePluginRemove(w http.ResponseWriter, r *http.Request) {
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
	if err := removePlugin(body.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.applyPluginChange(r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": body.ID})
}

func (s *appServer) handlePluginCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"commands": listPluginCommands()})
}

func (s *appServer) handlePluginCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID     string   `json:"id"`
		Grant  []string `json:"grant"`
		Revoke []string `json:"revoke"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	var st pluginStatus
	var err error
	switch {
	case len(body.Grant) > 0:
		st, err = grantPluginCode(body.ID, body.Grant)
	case len(body.Revoke) > 0:
		st, err = revokePluginCode(body.ID, body.Revoke)
	default:
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.applyPluginChange(r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

func (s *appServer) handlePluginAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "plugin and agent name required", http.StatusBadRequest)
		return
	}
	inst, err := startPluginAgent(body.ID, body.Name, body.WorkspaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"instance": inst})
}
