package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func (s *appServer) credentialsBroker() *CredentialBroker {
	if s == nil || s.agent == nil {
		return nil
	}
	if s.agent.Credentials == nil {
		s.agent.Credentials = NewCredentialBroker(s.agent.Cfg.ConfigPath)
	}
	return s.agent.Credentials
}

func (s *appServer) handleCredentials(w http.ResponseWriter, r *http.Request) {
	b := s.credentialsBroker()
	if b == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"vault":       b.Status(),
			"credentials": b.ListPublic(),
		})
	case http.MethodPost:
		var body struct {
			ID           string   `json:"id"`
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			Type         string   `json:"type"`
			Service      string   `json:"service"`
			AllowedTools []string `json:"allowed_tools"`
			InjectArg    string   `json:"inject_arg"`
			InjectEnv    string   `json:"inject_env"`
			Secret       string   `json:"secret"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if !b.Unlocked() {
			http.Error(w, "vault locked", http.StatusConflict)
			return
		}
		out, err := b.Upsert(Credential{
			ID: body.ID, Name: body.Name, Description: body.Description, Type: body.Type,
			Service: body.Service, AllowedTools: body.AllowedTools, InjectArg: body.InjectArg, InjectEnv: body.InjectEnv,
		}, body.Secret, true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out.public())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *appServer) handleCredentialsItem(w http.ResponseWriter, r *http.Request) {
	b := s.credentialsBroker()
	if b == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/credentials/")
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut, http.MethodPatch:
		var body struct {
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			Type         string   `json:"type"`
			Service      string   `json:"service"`
			AllowedTools []string `json:"allowed_tools"`
			InjectArg    string   `json:"inject_arg"`
			InjectEnv    string   `json:"inject_env"`
			Secret       *string  `json:"secret"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if !b.Unlocked() {
			http.Error(w, "vault locked", http.StatusConflict)
			return
		}
		replace := body.Secret != nil && strings.TrimSpace(*body.Secret) != ""
		sec := ""
		if replace {
			sec = *body.Secret
		}
		out, err := b.Upsert(Credential{
			ID: id, Name: body.Name, Description: body.Description, Type: body.Type,
			Service: body.Service, AllowedTools: body.AllowedTools, InjectArg: body.InjectArg, InjectEnv: body.InjectEnv,
		}, sec, replace)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out.public())
	case http.MethodDelete:
		if !b.Unlocked() {
			http.Error(w, "vault locked", http.StatusConflict)
			return
		}
		if err := b.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *appServer) handleCredentialsUnlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b := s.credentialsBroker()
	if b == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Password string `json:"password"`
		Create   bool   `json:"create"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	var err error
	if body.Create && !b.Exists() {
		err = b.Create(body.Password)
	} else {
		err = b.Unlock(body.Password)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":          true,
		"vault":       b.Status(),
		"credentials": b.ListPublic(),
	})
}
