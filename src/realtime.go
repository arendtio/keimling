package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const realtimeSeedBudget = 80_000

// Input transcription is off until a session asks for it. This model emits
// conversation.item.input_audio_transcription events for the user's speech.
const realtimeTranscriptionModel = "gpt-4o-mini-transcribe"

var realtimeVoices = map[string]bool{
	"alloy": true, "ash": true, "ballad": true, "coral": true, "echo": true,
	"sage": true, "shimmer": true, "verse": true, "marin": true, "cedar": true,
}

func normalizeRealtimeVoice(v string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return "marin", nil
	}
	if !realtimeVoices[v] {
		return "", fmt.Errorf("voice must be alloy, ash, ballad, coral, echo, sage, shimmer, verse, marin, or cedar")
	}
	return v, nil
}

func parseRealtimeSecret(body []byte) (string, error) {
	var m map[string]any
	if json.Unmarshal(body, &m) != nil {
		return "", fmt.Errorf("realtime secret response is not json")
	}
	if s := realtimeSecretField(m); s != "" {
		return s, nil
	}
	if cs, ok := m["client_secret"].(map[string]any); ok {
		if s := realtimeSecretField(cs); s != "" {
			return s, nil
		}
	}
	return "", fmt.Errorf("realtime secret missing")
}

func realtimeSecretField(m map[string]any) string {
	if m == nil {
		return ""
	}
	for _, key := range []string{"value", "secret"} {
		s, _ := m[key].(string)
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// realtimeSeedFrom is the text the live call receives at the start. It is the
// model-facing history, flattened so the session can take text items only.
func realtimeSeedFrom(items []json.RawMessage) []map[string]string {
	var all []map[string]string
	for _, raw := range items {
		row, ok := realtimeSeedRow(raw)
		if ok {
			all = append(all, row)
		}
	}
	if len(all) == 0 {
		return []map[string]string{}
	}
	used := 0
	start := len(all)
	for i := len(all) - 1; i >= 0; i-- {
		n := len(all[i]["text"])
		if used > 0 && used+n > realtimeSeedBudget {
			break
		}
		used += n
		start = i
	}
	kept := append([]map[string]string{}, all[start:]...)
	if start > 0 && strings.Contains(all[0]["text"], "<project_charter>") {
		kept = append([]map[string]string{all[0]}, kept...)
	}
	return kept
}

func realtimeSeedRow(raw json.RawMessage) (map[string]string, bool) {
	var item struct {
		Type      string `json:"type"`
		Role      string `json:"role"`
		Content   any    `json:"content"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
		Output    string `json:"output"`
		Text      string `json:"text"`
	}
	if json.Unmarshal(raw, &item) != nil {
		return nil, false
	}
	switch item.Type {
	case "reasoning":
		return nil, false
	case "function_call":
		return map[string]string{"role": "assistant", "text": truncate("Function call "+item.Name+" arguments "+item.Arguments, 500)}, true
	case "function_call_output":
		return map[string]string{"role": "user", "text": truncate("Tool result: "+item.Output, 1500)}, true
	case "keimling_notice":
		text := strings.TrimSpace(item.Text)
		if text == "" {
			return nil, false
		}
		return map[string]string{"role": "system", "text": truncate(text, 1500)}, true
	}
	role := strings.ToLower(strings.TrimSpace(item.Role))
	if item.Type == "message" && role == "" {
		role = "assistant"
	}
	text := strings.TrimSpace(realtimeContentText(item.Content))
	if text == "" || strings.Contains(text, "<context_budget>") {
		return nil, false
	}
	switch role {
	case "user", "assistant":
	case "developer", "system":
		role = "system"
	default:
		return nil, false
	}
	return map[string]string{"role": role, "text": truncate(text, 4000)}, true
}

func realtimeContentText(content any) string {
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
			t, _ := m["text"].(string)
			if t == "" {
				t, _ = m["transcript"].(string)
			}
			b.WriteString(t)
		}
		return b.String()
	default:
		return ""
	}
}

type realtimeEndpoint struct {
	Model string
	Voice string
	Kind  string
}

func (a *Agent) realtimeClient() (*Client, realtimeEndpoint, error) {
	var ep realtimeEndpoint
	if a == nil || a.Cfg.DisableExternal {
		return nil, ep, fmt.Errorf("live call is off")
	}
	id := strings.TrimSpace(a.Cfg.RealtimeProviderID)
	model := strings.TrimSpace(a.Cfg.RealtimeModel)
	if id == "" || model == "" {
		return nil, ep, fmt.Errorf("live call is off")
	}
	voice, err := normalizeRealtimeVoice(a.Cfg.RealtimeVoice)
	if err != nil {
		return nil, ep, err
	}
	fc, err := readConfiguration(a.Cfg.ConfigPath)
	if err != nil {
		return nil, ep, err
	}
	fc.ensureModelProviders()
	p := modelProviderByID(fc.ModelProviders, id)
	if p == nil {
		return nil, ep, fmt.Errorf("realtime provider is not configured")
	}
	if strings.TrimSpace(p.BaseURL) == "" {
		return nil, ep, fmt.Errorf("realtime provider has no URL")
	}
	kind := detectProviderKind(p.BaseURL, p.Kind)
	if kind == "openrouter" {
		return nil, ep, fmt.Errorf("OpenRouter does not offer a realtime call")
	}
	if strings.TrimSpace(p.APIKey) == "" {
		return nil, ep, fmt.Errorf("realtime provider has no API key")
	}
	c := NewClient(p.APIKey)
	c.BaseURL = p.BaseURL
	ep = realtimeEndpoint{Model: model, Voice: voice, Kind: kind}
	return c, ep, nil
}

func (a *Agent) realtimeInstructions() string {
	base := ""
	if a != nil {
		base = a.modelInstructions()
	}
	extra := "You are in a live voice call. The user hears your speech, and the same words are written into the chat. A typed line is the same person. Use tools the same way as in chat. A name that starts with mcp__ is a function. Call that function. Never pass that name to exec. Tool calls, especially mcp__ calls, run in the background. If the function output is the result, use it and comment on it. If the output says the call is still running, keep the conversation going and do not invent the result. When a later message says background tools finished, comment on those results before you treat the topic as done. When a user message includes an image, that image is the screen or window the user is sharing at that moment. Answer from that picture. When that message says keyboard and mouse are shared, screen_click, screen_scroll, screen_key, and screen_type act on the picture. x and y are fractions of the picture from 0 to 1. Click, then look at the next picture. screen_click, screen_scroll, screen_key, and screen_type finish before you continue."
	return strings.TrimSpace(base + "\n\n" + extra)
}

func realtimeToolList(a *Agent) []map[string]any {
	if a == nil {
		return []map[string]any{}
	}
	tools := filterGrantedTools(definitions(a.Cfg.Web, a.chatExtraTools()), a.instanceToolGrant())
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		copy := map[string]any{}
		for k, v := range tool {
			if k == "strict" {
				continue
			}
			copy[k] = v
		}
		out = append(out, copy)
	}
	return realtimeSafeTools(out)
}

func realtimeSafeTools(tools []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		safe, ok := realtimeSafeTool(tool)
		if !ok {
			continue
		}
		out = append(out, safe)
	}
	return realtimeMCPFirst(out)
}

// realtimeSafeTool makes one function definition acceptable to a realtime session.
// A schema the session would reject is dropped so the other tools still apply.
func realtimeSafeTool(tool map[string]any) (map[string]any, bool) {
	if tool == nil {
		return nil, false
	}
	raw, err := json.Marshal(tool)
	if err != nil {
		return nil, false
	}
	clone := map[string]any{}
	if json.Unmarshal(raw, &clone) != nil {
		return nil, false
	}
	tool = clone
	name, _ := tool["name"].(string)
	name = strings.TrimSpace(name)
	if !realtimeToolNameOK(name) {
		return nil, false
	}
	desc, _ := tool["description"].(string)
	desc = strings.TrimSpace(desc)
	if desc == "" {
		desc = name
	}
	params, _ := tool["parameters"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	}
	cleaned, _ := realtimeSafeSchema(params).(map[string]any)
	if cleaned == nil {
		cleaned = map[string]any{}
	}
	if _, ok := cleaned["type"]; !ok {
		cleaned["type"] = "object"
	}
	if _, ok := cleaned["properties"]; !ok {
		cleaned["properties"] = map[string]any{}
	}
	cleaned["additionalProperties"] = false
	return map[string]any{
		"type":        "function",
		"name":        name,
		"description": truncate(desc, 800),
		"parameters":  cleaned,
	}, true
}

func realtimeToolNameOK(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

func realtimeSafeSchema(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	for _, key := range []string{"$schema", "$id", "examples", "unevaluatedProperties"} {
		delete(m, key)
	}
	if types, ok := m["type"].([]any); ok {
		picked := "string"
		for _, t := range types {
			s, _ := t.(string)
			if s != "" && s != "null" {
				picked = s
				break
			}
		}
		m["type"] = picked
	}
	if _, ok := m["anyOf"]; ok {
		delete(m, "anyOf")
		if _, has := m["type"]; !has {
			m["type"] = "string"
		}
	}
	if _, ok := m["oneOf"]; ok {
		delete(m, "oneOf")
		if _, has := m["type"]; !has {
			m["type"] = "string"
		}
	}
	typ, _ := m["type"].(string)
	if typ == "array" {
		items, ok := m["items"]
		if !ok || items == nil {
			m["items"] = map[string]any{"type": "string"}
		} else {
			m["items"] = realtimeSafeSchema(items)
		}
	}
	if typ == "object" || m["properties"] != nil {
		if typ == "" {
			m["type"] = "object"
		}
		props, _ := m["properties"].(map[string]any)
		if props == nil {
			props = map[string]any{}
		}
		for k, p := range props {
			props[k] = realtimeSafeSchema(p)
		}
		m["properties"] = props
		m["additionalProperties"] = false
	}
	return m
}

func realtimeMCPFirst(tools []map[string]any) []map[string]any {
	var mcp, rest []map[string]any
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		if strings.HasPrefix(name, "mcp__") {
			mcp = append(mcp, tool)
			continue
		}
		rest = append(rest, tool)
	}
	return append(mcp, rest...)
}

func mintRealtimeSecret(ctx context.Context, c *Client, ep realtimeEndpoint) (string, error) {
	if c == nil || c.HTTP == nil {
		return "", fmt.Errorf("no realtime client")
	}
	var payload any
	if ep.Kind == "litellm" {
		payload = map[string]string{"model": ep.Model}
	} else {
		payload = map[string]any{"session": map[string]any{
			"type":  "realtime",
			"model": ep.Model,
			"audio": map[string]any{
				"input":  map[string]any{"transcription": map[string]any{"model": realtimeTranscriptionModel}},
				"output": map[string]any{"voice": ep.Voice},
			},
		}}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("realtime/client_secrets"), bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return "", realtimeHTTPError("realtime secret", body)
	}
	return parseRealtimeSecret(body)
}

func realtimeHTTPError(prefix string, body []byte) error {
	var wrap struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &wrap) == nil {
		if msg := strings.TrimSpace(wrap.Error.Message); msg != "" {
			return fmt.Errorf("%s: %s", prefix, msg)
		}
		if msg := strings.TrimSpace(wrap.Message); msg != "" {
			return fmt.Errorf("%s: %s", prefix, msg)
		}
	}
	return fmt.Errorf("%s: %s", prefix, truncate(strings.TrimSpace(string(body)), 300))
}

func exchangeRealtimeSDP(ctx context.Context, c *Client, secret, offer string) (string, error) {
	if c == nil || c.HTTP == nil {
		return "", fmt.Errorf("no realtime client")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("realtime/calls"), strings.NewReader(offer))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(secret))
	req.Header.Set("Content-Type", "application/sdp")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return "", realtimeHTTPError("realtime call", body)
	}
	answer := strings.TrimSpace(string(body))
	if answer == "" {
		return "", fmt.Errorf("realtime call returned an empty answer")
	}
	return answer, nil
}

func (s *appServer) realtimeCallOpen() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.realtimeOn
}

func (s *appServer) realtimeReady() bool {
	if s == nil || s.agent == nil {
		return false
	}
	_, _, err := s.agent.realtimeClient()
	return err == nil
}

func (s *appServer) handleRealtimeSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	if s.realtimeCallOpen() {
		http.Error(w, "a call is already open", http.StatusConflict)
		return
	}
	s.mu.Lock()
	busy := s.busy
	s.mu.Unlock()
	if busy {
		http.Error(w, "finish the current run before calling", http.StatusConflict)
		return
	}
	_, ep, err := s.agent.realtimeClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	tools := realtimeToolList(s.agent)
	history := realtimeSeedFrom(s.agent.modelInput())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"model":               ep.Model,
		"voice":               ep.Voice,
		"transcription_model": realtimeTranscriptionModel,
		"instructions":        s.agent.realtimeInstructions(),
		"tools":               tools,
		"history":             history,
	})
}

func (s *appServer) handleRealtimeSDP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		SDP string `json:"sdp"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.SDP) == "" {
		http.Error(w, "sdp required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	busy := s.busy
	s.mu.Unlock()
	if busy {
		http.Error(w, "finish the current run before calling", http.StatusConflict)
		return
	}
	c, ep, err := s.agent.realtimeClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	secret, err := mintRealtimeSecret(ctx, c, ep)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	answer, err := exchangeRealtimeSDP(ctx, c, secret, body.SDP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	s.mu.Lock()
	if s.realtimeCancel != nil {
		s.realtimeCancel()
	}
	callCtx, callCancel := context.WithCancel(context.Background())
	s.realtimeOn = true
	s.realtimeCtx = callCtx
	s.realtimeCancel = callCancel
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"sdp": answer})
}

func (s *appServer) handleRealtimeEnd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.endRealtimeCall()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *appServer) beginRealtimeCall() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.realtimeCancel != nil {
		s.realtimeCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.realtimeGen++
	s.realtimeOn = true
	s.realtimeCtx = ctx
	s.realtimeCancel = cancel
	return s.realtimeGen
}

// endRealtimeCallGen clears the call only when gen is still the active one.
func (s *appServer) endRealtimeCallGen(gen uint64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.realtimeGen != gen {
		s.mu.Unlock()
		return
	}
	cancel := s.realtimeCancel
	s.realtimeOn = false
	s.realtimeCtx = nil
	s.realtimeCancel = nil
	s.realtimeGen++
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *appServer) endRealtimeCall() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel := s.realtimeCancel
	s.realtimeOn = false
	s.realtimeCtx = nil
	s.realtimeCancel = nil
	s.realtimeGen++
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *appServer) handleRealtimeLine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Role string `json:"role"`
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	role := strings.ToLower(strings.TrimSpace(body.Role))
	text := strings.TrimSpace(body.Text)
	if text == "" || (role != "user" && role != "assistant") {
		http.Error(w, "role and text required", http.StatusBadRequest)
		return
	}
	var item any
	if role == "user" {
		item = map[string]any{"role": "user", "content": text}
	} else {
		item = map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": text}}}
	}
	if err := s.agent.add(item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.noteCallThreadTitle()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func realtimeArgs(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return "{}"
	}
	if strings.HasPrefix(s, "\"") {
		var inner string
		if json.Unmarshal(raw, &inner) == nil && strings.TrimSpace(inner) != "" {
			return inner
		}
	}
	return s
}

func toolOutputFor(history []json.RawMessage, callID string) string {
	for i := len(history) - 1; i >= 0; i-- {
		var item struct {
			Type   string `json:"type"`
			CallID string `json:"call_id"`
			Output string `json:"output"`
		}
		if json.Unmarshal(history[i], &item) != nil {
			continue
		}
		if item.Type == "function_call_output" && item.CallID == callID {
			return item.Output
		}
	}
	return ""
}

func (a *Agent) runRealtimeTool(ctx context.Context, name, callID, args string) (string, error) {
	if a == nil {
		return "", fmt.Errorf("no agent")
	}
	name = strings.TrimSpace(name)
	callID = strings.TrimSpace(callID)
	if name == "" || callID == "" {
		return "", fmt.Errorf("name and call_id required")
	}
	if strings.TrimSpace(args) == "" {
		args = "{}"
	}
	item := Item{Type: "function_call", Name: name, CallID: callID, Arguments: args}
	if err := a.add(map[string]any{"type": "function_call", "name": name, "call_id": callID, "arguments": args}); err != nil {
		return "", err
	}
	err := a.handleFunctionCall(ctx, "", nil, item)
	if out := toolOutputFor(a.History, callID); out != "" {
		return out, err
	}
	if err != nil {
		_ = a.add(map[string]any{"type": "function_call_output", "call_id": callID, "output": "INTERRUPTED: " + err.Error(), "ok": false})
		return "", err
	}
	return "", fmt.Errorf("tool returned no output")
}

func (s *appServer) handleRealtimeTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	if !s.realtimeCallOpen() {
		http.Error(w, "no live call", http.StatusConflict)
		return
	}
	var body struct {
		Name      string          `json:"name"`
		CallID    string          `json:"call_id"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	ctx := s.realtimeCtx
	s.mu.Unlock()
	if ctx == nil {
		ctx = r.Context()
	}
	out, err := s.agent.runRealtimeTool(ctx, body.Name, body.CallID, realtimeArgs(body.Arguments))
	if err != nil && out == "" {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"output": out})
}

func (s *appServer) injectRealtimeWake(text string) bool {
	if s == nil || s.agent == nil || strings.TrimSpace(text) == "" {
		return false
	}
	if err := s.agent.add(map[string]any{"role": "user", "content": text}); err != nil {
		return false
	}
	threadID := s.agent.Cfg.ThreadID
	if notice, ok := watchDisplayText(text); ok {
		s.hub.Broadcast("message", map[string]any{"role": "watch", "name": "Watch", "text": notice, "thread_id": threadID})
	} else if name, body, ok := parsePeerUserText(text); ok {
		s.hub.Broadcast("message", map[string]any{"role": "peer", "name": "From " + name, "text": body, "thread_id": threadID})
	} else {
		s.hub.Broadcast("message", map[string]any{"role": "user", "text": text, "thread_id": threadID})
	}
	s.hub.Broadcast("realtime_inject", map[string]string{"text": text, "thread_id": threadID})
	return true
}
