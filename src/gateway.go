package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// A gateway is a directed inspection rule between two agent instances.
// When inspect is on, a peer message from From to To is held until the user
// allows it through or blocks it. The rule is global: agents cross workspaces,
// so the pair is the instance ids, not a workspace.
type gatewayEdge struct {
	From     string `json:"from"`
	FromName string `json:"from_name"`
	To       string `json:"to"`
	ToName   string `json:"to_name"`
	Inspect  bool   `json:"inspect"`
}

type heldMessage struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	FromName  string    `json:"from_name"`
	To        string    `json:"to"`
	ToName    string    `json:"to_name"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type gatewayFile struct {
	Inspect []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"inspect"`
	Pending []heldMessage `json:"pending"`
}

var (
	gatewayMu      sync.Mutex
	gatewayWaiters = map[string]chan string{}
)

func gatewayPath() string {
	return filepath.Join(keimlingConfigDir(), "gateways.json")
}

func loadGateways() gatewayFile {
	var st gatewayFile
	b, err := os.ReadFile(gatewayPath())
	if err != nil {
		return st
	}
	_ = json.Unmarshal(b, &st)
	if st.Pending == nil {
		st.Pending = []heldMessage{}
	}
	return st
}

func saveGateways(st gatewayFile) error {
	if st.Pending == nil {
		st.Pending = []heldMessage{}
	}
	if st.Inspect == nil {
		st.Inspect = []struct {
			From string `json:"from"`
			To   string `json:"to"`
		}{}
	}
	if err := os.MkdirAll(filepath.Dir(gatewayPath()), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(gatewayPath(), append(b, '\n'), 0o600)
}

func gatewayKey(from, to string) string {
	return strings.TrimSpace(from) + "\n" + strings.TrimSpace(to)
}

// gatewayInspect reports whether a peer message from one instance to another
// must wait for the user. A missing rule means the message goes through.
func gatewayInspect(from, to string) bool {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if from == "" || to == "" {
		return false
	}
	gatewayMu.Lock()
	defer gatewayMu.Unlock()
	st := loadGateways()
	for _, r := range st.Inspect {
		if r.From == from && r.To == to {
			return true
		}
	}
	return false
}

func setGatewayInspect(from, to string, inspect bool) error {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if from == "" || to == "" || from == to {
		return fmt.Errorf("from and to required")
	}
	gatewayMu.Lock()
	defer gatewayMu.Unlock()
	st := loadGateways()
	next := make([]struct {
		From string `json:"from"`
		To   string `json:"to"`
	}, 0, len(st.Inspect)+1)
	for _, r := range st.Inspect {
		if r.From == from && r.To == to {
			continue
		}
		next = append(next, r)
	}
	if inspect {
		next = append(next, struct {
			From string `json:"from"`
			To   string `json:"to"`
		}{From: from, To: to})
	}
	st.Inspect = next
	return saveGateways(st)
}

// gatewayEdges lists each direction between instances that may contact each
// other. A pair that cannot talk does not appear.
func gatewayEdges() []gatewayEdge {
	all, err := listAgentInstances()
	if err != nil {
		return nil
	}
	var edges []gatewayEdge
	for i := range all {
		for j := range all {
			if i == j {
				continue
			}
			if !agentsMayContact(all[i], all[j]) {
				continue
			}
			edges = append(edges, gatewayEdge{
				From:     all[i].ID,
				FromName: agentLabel(all[i]),
				To:       all[j].ID,
				ToName:   agentLabel(all[j]),
				Inspect:  gatewayInspect(all[i].ID, all[j].ID),
			})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromName != edges[j].FromName {
			return edges[i].FromName < edges[j].FromName
		}
		if edges[i].ToName != edges[j].ToName {
			return edges[i].ToName < edges[j].ToName
		}
		return edges[i].From < edges[j].From
	})
	return edges
}

func agentLabel(m AgentInstanceMeta) string {
	if name := strings.TrimSpace(m.Name); name != "" {
		return name
	}
	return m.ID
}

func listHeldMessages() []heldMessage {
	gatewayMu.Lock()
	defer gatewayMu.Unlock()
	st := loadGateways()
	out := append([]heldMessage(nil), st.Pending...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// holdGatewayMessage records a peer message and returns its id. The caller
// waits with waitGatewayDecision. Register the waiter before this returns so a
// decision cannot land in the gap.
func holdGatewayMessage(from, fromName, to, toName, text string) (string, <-chan string, error) {
	text = strings.TrimSpace(text)
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if from == "" || to == "" || text == "" {
		return "", nil, fmt.Errorf("from, to, and text required")
	}
	msg := heldMessage{
		ID:        newStampedID("hold"),
		From:      from,
		FromName:  strings.TrimSpace(fromName),
		To:        to,
		ToName:    strings.TrimSpace(toName),
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}
	ch := make(chan string, 1)
	gatewayMu.Lock()
	defer gatewayMu.Unlock()
	gatewayWaiters[msg.ID] = ch
	st := loadGateways()
	st.Pending = append(st.Pending, msg)
	if err := saveGateways(st); err != nil {
		delete(gatewayWaiters, msg.ID)
		return "", nil, err
	}
	return msg.ID, ch, nil
}

// waitGatewayDecision blocks until the user allows or blocks the held message,
// or until ctx ends. A cancel drops the message if it is still waiting.
func waitGatewayDecision(ctx context.Context, id string, ch <-chan string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case status := <-ch:
		return status, nil
	case <-ctx.Done():
		_ = dropHeldMessage(id)
		return "", ctx.Err()
	}
}

func dropHeldMessage(id string) error {
	gatewayMu.Lock()
	defer gatewayMu.Unlock()
	if err := removeHeldLocked(id, "dropped"); err != nil {
		return err
	}
	signalHeld(id, "dropped")
	return nil
}

func removeHeldLocked(id, status string) error {
	st := loadGateways()
	next := st.Pending[:0]
	found := false
	for _, m := range st.Pending {
		if m.ID == id {
			found = true
			continue
		}
		next = append(next, m)
	}
	if !found {
		return fmt.Errorf("unknown held message %q", id)
	}
	st.Pending = next
	return saveGateways(st)
}

func signalHeld(id, status string) {
	if ch := gatewayWaiters[id]; ch != nil {
		select {
		case ch <- status:
		default:
		}
		delete(gatewayWaiters, id)
	}
}

// decideHeldMessage delivers the message when allow is true, otherwise drops
// it. A sender that is still waiting hears the result. A sender that already
// stopped still has the message delivered on allow, because the user asked
// for that.
func decideHeldMessage(id string, allow bool) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	gatewayMu.Lock()
	st := loadGateways()
	var msg *heldMessage
	for i := range st.Pending {
		if st.Pending[i].ID == id {
			msg = &st.Pending[i]
			break
		}
	}
	if msg == nil {
		gatewayMu.Unlock()
		return fmt.Errorf("unknown held message %q", id)
	}
	copyMsg := *msg
	if err := removeHeldLocked(id, ""); err != nil {
		gatewayMu.Unlock()
		return err
	}
	gatewayMu.Unlock()
	status := "blocked"
	if allow {
		if err := enqueueAgentMessage(copyMsg.To, copyMsg.Text, copyMsg.From, "peer"); err != nil {
			gatewayMu.Lock()
			signalHeld(id, "blocked")
			gatewayMu.Unlock()
			return err
		}
		status = "approved"
	}
	gatewayMu.Lock()
	signalHeld(id, status)
	gatewayMu.Unlock()
	return nil
}

func (h *Host) handleGateways(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"edges":   gatewayEdges(),
		"pending": listHeldMessages(),
	})
}

func (h *Host) handleGatewayInspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Inspect *bool  `json:"inspect"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || body.Inspect == nil {
		http.Error(w, "from, to, and inspect required", http.StatusBadRequest)
		return
	}
	if err := setGatewayInspect(body.From, body.To, *body.Inspect); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "edges": gatewayEdges(), "pending": listHeldMessages()})
}

func (h *Host) handleGatewayDecide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID    string `json:"id"`
		Allow *bool  `json:"allow"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || body.Allow == nil || strings.TrimSpace(body.ID) == "" {
		http.Error(w, "id and allow required", http.StatusBadRequest)
		return
	}
	if err := decideHeldMessage(body.ID, *body.Allow); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "pending": listHeldMessages()})
}
