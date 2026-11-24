package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Escalation is one entry an agent raised for the human. It lives in its own
// file so a restart keeps it and the sidebar can show it without touching a
// chat.
type Escalation struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	AgentName string    `json:"agent_name"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	// Done marks an entry the human has worked off. It stays in the file so the
	// history stays complete.
	Done   bool      `json:"done,omitempty"`
	DoneAt time.Time `json:"done_at,omitempty"`
}

func escalationPath() string {
	return filepath.Join(keimlingConfigDir(), "escalations.jsonl")
}

// addEscalation appends one entry. A file per line: append-only, readable while
// an agent writes, and easy to hand to the UI newest-first.
func addEscalation(agentID, agentName, text string) (Escalation, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Escalation{}, fmt.Errorf("text required")
	}
	e := Escalation{
		ID:        newStampedID("esc"),
		AgentID:   strings.TrimSpace(agentID),
		AgentName: strings.TrimSpace(agentName),
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}
	if err := os.MkdirAll(filepath.Dir(escalationPath()), 0o700); err != nil {
		return Escalation{}, err
	}
	f, err := os.OpenFile(escalationPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Escalation{}, err
	}
	defer f.Close()
	b, err := json.Marshal(e)
	if err != nil {
		return Escalation{}, err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return Escalation{}, err
	}
	return e, nil
}

// listEscalations returns the newest entries first. A malformed line is skipped:
// one broken write must not hide the rest.
func listEscalations(limit int) []Escalation {
	out := []Escalation{}
	f, err := os.Open(escalationPath())
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e Escalation
		if json.Unmarshal([]byte(line), &e) != nil || strings.TrimSpace(e.Text) == "" {
			continue
		}
		out = append(out, e)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// markEscalationDone sets or clears the done flag of one entry. The file is
// rewritten as a whole: it is a short list, and every line stays one entry.
func markEscalationDone(id string, done bool) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	list := listEscalations(0)
	found := false
	for i := range list {
		if list[i].ID != id {
			continue
		}
		found = true
		list[i].Done = done
		if done {
			list[i].DoneAt = time.Now().UTC()
		} else {
			list[i].DoneAt = time.Time{}
		}
	}
	if !found {
		return fmt.Errorf("unknown escalation %q", id)
	}
	var b strings.Builder
	for _, e := range list {
		line, err := json.Marshal(e)
		if err != nil {
			return err
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return os.WriteFile(escalationPath(), []byte(b.String()), 0o600)
}

// handleEscalationDone takes {"id": "esc-…", "done": true|false}.
func (h *Host) handleEscalationDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID   string `json:"id"`
		Done *bool  `json:"done"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	done := true
	if body.Done != nil {
		done = *body.Done
	}
	if err := markEscalationDone(body.ID, done); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": strings.TrimSpace(body.ID), "done": done, "open": openEscalations()})
}

// openEscalations counts the entries nobody has worked off yet.
func openEscalations() int {
	n := 0
	for _, e := range listEscalations(0) {
		if !e.Done {
			n++
		}
	}
	return n
}

// handleEscalations serves the list the sidebar shows.
func (h *Host) handleEscalations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"escalations": listEscalations(50)})
}

func escalationToolDefs() []map[string]any {
	return []map[string]any{
		toolDef("escalate", "Raise a problem for the human to see in the Escalations tab. Use it when a role, a permission, a setup or a repeated failure blocks the work and you cannot fix it yourself. Write what happened, what you tried and what you need, in one short paragraph. This does not stop you from also messaging another agent. Do not use it for progress notes.", map[string]any{"text": map[string]any{"type": "string", "description": "What is blocked, what you tried, and what you need. A few sentences."}}, "text"),
	}
}

// escalateTool records one entry. It needs an agent instance: the tab names who
// raised it, and a chat is not an agent.
func (a *Agent) escalateTool(argsJSON string) string {
	selfID := ""
	if a != nil {
		selfID = strings.TrimSpace(a.Cfg.AgentInstanceID)
	}
	if selfID == "" {
		return "escalate error: only available on agent instances"
	}
	var x struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &x); err != nil {
		return "escalate error: invalid arguments"
	}
	name := selfID
	if m, err := readAgentInstanceMeta(selfID); err == nil && strings.TrimSpace(m.Name) != "" {
		name = strings.TrimSpace(m.Name)
	}
	e, err := addEscalation(selfID, name, x.Text)
	if err != nil {
		return "escalate error: " + err.Error()
	}
	return fmt.Sprintf("raised for the human as %s: %s", e.ID, e.Text)
}
