package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealtimeSeedKeepsChatText(t *testing.T) {
	user, _ := json.Marshal(map[string]any{"role": "user", "content": "hello"})
	asst, _ := json.Marshal(map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": "hi there"}}})
	call, _ := json.Marshal(map[string]any{"type": "function_call", "name": "read_file", "call_id": "c", "arguments": `{"path":"a.go"}`})
	out, _ := json.Marshal(map[string]any{"type": "function_call_output", "call_id": "c", "output": "package main"})
	budget, _ := json.Marshal(map[string]any{"role": "developer", "content": "<context_budget>\nlimit=1\n</context_budget>"})
	items := realtimeSeedFrom([]json.RawMessage{budget, user, asst, call, out})
	if len(items) != 4 {
		t.Fatalf("items: %#v", items)
	}
	if items[0]["role"] != "user" || items[0]["text"] != "hello" {
		t.Fatalf("user: %#v", items[0])
	}
	if items[1]["role"] != "assistant" || items[1]["text"] != "hi there" {
		t.Fatalf("assistant: %#v", items[1])
	}
	if items[2]["role"] != "assistant" || !strings.Contains(items[2]["text"], "read_file") {
		t.Fatalf("call: %#v", items[2])
	}
	if items[3]["role"] != "user" || !strings.Contains(items[3]["text"], "package main") {
		t.Fatalf("result: %#v", items[3])
	}
}

func TestRealtimeHTTPErrorUsesTheMessage(t *testing.T) {
	err := realtimeHTTPError("realtime secret", []byte(`{"error":{"message":"Incorrect API key provided"}}`))
	if err == nil || err.Error() != "realtime secret: Incorrect API key provided" {
		t.Fatal(err)
	}
}

func TestParseRealtimeSecret(t *testing.T) {
	for _, body := range []string{
		`{"value":"ephemeral"}`,
		`{"secret":"ephemeral"}`,
		`{"client_secret":{"value":"ephemeral"}}`,
		`{"client_secret":{"secret":"ephemeral"}}`,
	} {
		got, err := parseRealtimeSecret([]byte(body))
		if err != nil || got != "ephemeral" {
			t.Fatalf("%s: %q %v", body, got, err)
		}
	}
}

func TestRealtimeCallUsesEphemeralSecretAndStaysIdle(t *testing.T) {
	var sawOffer, sawOperator bool
	var mintBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realtime/client_secrets":
			if r.Header.Get("Authorization") == "Bearer operator-key" {
				sawOperator = true
			}
			mintBody, _ = io.ReadAll(r.Body)
			_ = json.NewEncoder(w).Encode(map[string]string{"value": "ephemeral-token"})
		case "/realtime/calls":
			if r.Header.Get("Authorization") != "Bearer ephemeral-token" {
				t.Errorf("calls auth: %s", r.Header.Get("Authorization"))
			}
			if r.Header.Get("Content-Type") != "application/sdp" {
				t.Errorf("content-type: %s", r.Header.Get("Content-Type"))
			}
			b, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(b), "v=0") {
				t.Errorf("offer: %q", b)
			}
			sawOffer = true
			_, _ = io.WriteString(w, "v=0\r\nanswer\r\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	raw, _ := json.Marshal(fileConfig{ModelProviders: []modelProvider{{
		ID: "oa", Name: "OpenAI", BaseURL: upstream.URL, APIKey: "operator-key", Kind: "openai",
	}}})
	if err := os.WriteFile(cfgPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	j, err := OpenJournal(filepath.Join(dir, "s.jsonl"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &Agent{
		Cfg: Config{
			ConfigPath:         cfgPath,
			Mode:               "ask",
			RealtimeProviderID: "oa", RealtimeModel: "gpt-realtime-2.1", RealtimeVoice: "marin",
			HasModelProvider: true,
		},
		Journal: j,
	}
	if err := a.add(map[string]any{"role": "user", "content": "hello"}); err != nil {
		t.Fatal(err)
	}
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})

	req := httptest.NewRequest(http.MethodPost, "/api/realtime/session", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("session: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "operator-key") || strings.Contains(rec.Body.String(), "ephemeral-token") {
		t.Fatalf("secret leaked")
	}
	var setup struct {
		History            []map[string]string `json:"history"`
		Voice              string              `json:"voice"`
		TranscriptionModel string              `json:"transcription_model"`
		Instructions       string              `json:"instructions"`
		Tools              []map[string]any    `json:"tools"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &setup); err != nil {
		t.Fatal(err)
	}
	if setup.Voice != "marin" || setup.TranscriptionModel != realtimeTranscriptionModel {
		t.Fatalf("voice %q transcription %q", setup.Voice, setup.TranscriptionModel)
	}
	if !strings.Contains(setup.Instructions, "mcp__") {
		t.Fatal("instructions do not tell the call that mcp__ is a function")
	}
	if !strings.Contains(setup.Instructions, "screen or window") {
		t.Fatal("instructions do not tell the call that a shared picture is the screen")
	}
	if len(setup.History) == 0 || setup.History[len(setup.History)-1]["text"] != "hello" {
		t.Fatalf("history: %#v", setup.History)
	}
	sawExec := false
	for _, tool := range setup.Tools {
		if tool["name"] == "exec" {
			sawExec = true
		}
	}
	if !sawExec {
		t.Fatal("exec missing")
	}

	sdpReq := httptest.NewRequest(http.MethodPost, "/api/realtime/sdp", strings.NewReader(`{"sdp":"v=0\r\noffer"}`))
	sdpReq.Header.Set("Authorization", "Bearer tok")
	sdpRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(sdpRec, sdpReq)
	if sdpRec.Code != http.StatusOK {
		t.Fatalf("sdp: %d %s", sdpRec.Code, sdpRec.Body.String())
	}
	var sdpResp struct {
		SDP string `json:"sdp"`
	}
	if err := json.Unmarshal(sdpRec.Body.Bytes(), &sdpResp); err != nil {
		t.Fatal(err)
	}
	if sdpResp.SDP != "v=0\r\nanswer\r\n" {
		t.Fatalf("answer sdp: %q", sdpResp.SDP)
	}
	if !sawOperator || !sawOffer {
		t.Fatalf("operator=%v offer=%v", sawOperator, sawOffer)
	}
	if !strings.Contains(string(mintBody), realtimeTranscriptionModel) {
		t.Fatalf("mint body: %s", mintBody)
	}
	if !srv.realtimeCallOpen() || srv.busy {
		t.Fatalf("call=%v busy=%v", srv.realtimeCallOpen(), srv.busy)
	}

	msg := httptest.NewRequest(http.MethodPost, "/api/message", strings.NewReader(`{"text":"while we talk"}`))
	msg.Header.Set("Authorization", "Bearer tok")
	msgRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(msgRec, msg)
	if msgRec.Code != http.StatusAccepted || !strings.Contains(msgRec.Body.String(), `"realtime":true`) {
		t.Fatalf("message: %d %s", msgRec.Code, msgRec.Body.String())
	}
	if srv.busy {
		t.Fatal("text model started during the call")
	}

	line := httptest.NewRequest(http.MethodPost, "/api/realtime/line", strings.NewReader(`{"role":"assistant","text":"I heard you"}`))
	line.Header.Set("Authorization", "Bearer tok")
	lineRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(lineRec, line)
	if lineRec.Code != http.StatusOK {
		t.Fatalf("line: %d %s", lineRec.Code, lineRec.Body.String())
	}
	view := sessionMessages(a.History, 20)
	var sawUser, sawAssistant bool
	for _, m := range view {
		if m.Role == "user" && m.Text == "while we talk" {
			sawUser = true
		}
		if m.Role == "assistant" && m.Text == "I heard you" {
			sawAssistant = true
		}
	}
	if !sawUser || !sawAssistant {
		t.Fatalf("transcript: %#v", view)
	}

	tool := httptest.NewRequest(http.MethodPost, "/api/realtime/tool", strings.NewReader(`{"name":"update_plan","call_id":"c1","arguments":{"steps":[{"step":"look","status":"pending"}]}}`))
	tool.Header.Set("Authorization", "Bearer tok")
	toolRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(toolRec, tool)
	if toolRec.Code != http.StatusOK || !strings.Contains(toolRec.Body.String(), "plan updated") {
		t.Fatalf("tool: %d %s", toolRec.Code, toolRec.Body.String())
	}
	if srv.busy {
		t.Fatal("tool started a text run")
	}
}

func TestRealtimeSafeToolKeepsMCPAndFillsArrayItems(t *testing.T) {
	tools := realtimeSafeTools([]map[string]any{
		{"type": "function", "name": "exec", "description": "run", "parameters": map[string]any{"type": "object", "properties": map[string]any{}}},
		{"type": "function", "name": "bad name", "description": "no", "parameters": map[string]any{"type": "object"}},
		{
			"type": "function", "name": "mcp__Operations__list_projects", "description": "List projects (MCP)",
			"parameters": map[string]any{
				"type":    "object",
				"$schema": "http://json-schema.org/draft-07/schema#",
				"properties": map[string]any{
					"tags": map[string]any{"type": "array"},
					"note": map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "null"}}},
				},
			},
		},
	})
	if len(tools) != 2 {
		t.Fatalf("tools: %#v", tools)
	}
	if tools[0]["name"] != "mcp__Operations__list_projects" || tools[1]["name"] != "exec" {
		t.Fatalf("order: %#v", tools)
	}
	params, _ := tools[0]["parameters"].(map[string]any)
	if params["additionalProperties"] != false {
		t.Fatalf("additionalProperties: %#v", params["additionalProperties"])
	}
	props, _ := params["properties"].(map[string]any)
	tags, _ := props["tags"].(map[string]any)
	if _, ok := tags["items"]; !ok {
		t.Fatalf("tags: %#v", tags)
	}
	if _, ok := params["$schema"]; ok {
		t.Fatal("schema keyword kept")
	}
}

func TestRealtimeOpenRouterIsRefused(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	raw, _ := json.Marshal(fileConfig{ModelProviders: []modelProvider{{
		ID: "or", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", APIKey: "k", Kind: "openrouter",
	}}})
	if err := os.WriteFile(cfgPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{ConfigPath: cfgPath, RealtimeProviderID: "or", RealtimeModel: "openai/gpt-realtime"}}
	_, _, err := a.realtimeClient()
	if err == nil || !strings.Contains(err.Error(), "OpenRouter") {
		t.Fatal(err)
	}
}

func TestRealtimeWakeDoesNotStartTheTextModel(t *testing.T) {
	j, err := OpenJournal(filepath.Join(t.TempDir(), "s.jsonl"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &Agent{Cfg: Config{Mode: "ask", ThreadID: "t1"}, Journal: j}
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})
	srv.realtimeOn = true
	if !srv.startAgentRun("[Watch]\nping") {
		t.Fatal("wake was not accepted")
	}
	if srv.busy {
		t.Fatal("wake started a text run")
	}
	if len(a.History) != 1 {
		t.Fatalf("history: %d", len(a.History))
	}
}
