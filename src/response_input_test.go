package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeResponseInputPreservesSemantics(t *testing.T) {
	items := []json.RawMessage{
		json.RawMessage(`{"type":"message","id":"msg_123","status":"completed","role":"assistant","phase":"final_answer","content":[{"type":"output_text","text":"hello","annotations":[],"logprobs":null}]}`),
		json.RawMessage(`{"type":"reasoning","id":"rs_1","status":"completed","encrypted_content":"opaque","summary":[]}`),
		json.RawMessage(`{"type":"function_call","id":"fc_1","status":"completed","name":"exec","call_id":"call_1","arguments":"{}"}`),
		json.RawMessage(`{"type":"function_call_output","call_id":"call_1","output":"{\"status\":\"completed\"}"}`),
		json.RawMessage(`{"role":"user","content":"test"}`),
	}
	normalized, err := normalizeResponseInput(items)
	if err != nil {
		t.Fatal(err)
	}
	if len(normalized) != len(items) {
		t.Fatal("lost history items")
	}
	if bytes.Contains(normalized[0], []byte(`"status"`)) || bytes.Contains(normalized[0], []byte(`"annotations"`)) || bytes.Contains(normalized[0], []byte(`"id"`)) {
		t.Fatalf("unremoved output fields: %s", normalized[0])
	}
	if !bytes.Contains(normalized[0], []byte(`"phase":"final_answer"`)) {
		t.Fatal("lost phase")
	}
	if !bytes.Contains(normalized[1], []byte(`"encrypted_content":"opaque"`)) {
		t.Fatal("lost reasoning")
	}
	if bytes.Contains(normalized[2], []byte(`"id"`)) || bytes.Contains(normalized[2], []byte(`"status"`)) {
		t.Fatalf("invalid function call: %s", normalized[2])
	}
	if !bytes.Contains(normalized[3], []byte(`\"status\":\"completed\"`)) {
		t.Fatal("modified tool output")
	}
	if !bytes.Contains(items[0], []byte(`"status"`)) {
		t.Fatal("modified original journal")
	}
}

func TestNormalizeNoticeBecomesDeveloperMessage(t *testing.T) {
	items := []json.RawMessage{
		json.RawMessage(`{"type":"keimling_notice","text":"Run stopped: empty response"}`),
	}
	normalized, err := normalizeResponseInput(items)
	if err != nil {
		t.Fatal(err)
	}
	if len(normalized) != 1 {
		t.Fatal(len(normalized))
	}
	if bytes.Contains(normalized[0], []byte(`"type"`)) || bytes.Contains(normalized[0], []byte("keimling_notice")) {
		t.Fatalf("notice type leaked: %s", normalized[0])
	}
	if !bytes.Contains(normalized[0], []byte(`"role":"developer"`)) || !bytes.Contains(normalized[0], []byte("Run stopped: empty response")) {
		t.Fatalf("%s", normalized[0])
	}
}

func TestRespondNormalizesBeforeHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Input []map[string]json.RawMessage `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if _, exists := payload.Input[1]["status"]; exists {
			t.Error("status sent to API")
		}
		if _, exists := payload.Input[1]["id"]; exists {
			t.Error("message id sent to API")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"completed","output":[]}`))
	}))
	defer srv.Close()
	c := NewClient("test")
	c.BaseURL = srv.URL
	_, err := c.Respond(context.Background(), "test-model", "low", []json.RawMessage{
		json.RawMessage(`{"role":"user","content":"hey"}`),
		json.RawMessage(`{"type":"message","role":"assistant","id":"msg_123","status":"completed","content":[{"type":"output_text","text":"hi"}]}`),
	}, false, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRawContextOmitsStoredCommandOutput(t *testing.T) {
	item, err := json.Marshal(map[string]any{
		"type":       "function_call_output",
		"call_id":    "c1",
		"output":     "exit_code=0\n[Filtered]\nshort",
		"raw_output": "exit_code=0\nSECRET-LISTING",
	})
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{History: []json.RawMessage{item}, Charter: &Charter{Version: charterVersion}}
	before, err := json.Marshal(a.rawContextView())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(before, []byte("SECRET-LISTING")) {
		t.Fatalf("unsent history included in context view: %s", before)
	}
	sent, err := json.Marshal(map[string]any{
		"instructions": "be brief",
		"tools":        []map[string]any{{"name": "exec"}},
		"input": []map[string]any{{
			"type":    "function_call_output",
			"call_id": "c1",
			"output":  "exit_code=0\n[Filtered]\nshort",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	a.noteSentChat("", sent)
	raw, err := json.Marshal(a.rawContextView())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("SECRET-LISTING")) {
		t.Fatalf("raw command output included in context view: %s", raw)
	}
	if !bytes.Contains(raw, []byte("[Filtered]")) {
		t.Fatalf("filtered output missing: %s", raw)
	}
}

func contextPart(t *testing.T, a *Agent, name string) string {
	t.Helper()
	raw, err := json.Marshal(a.rawContextView())
	if err != nil {
		t.Fatal(err)
	}
	var view struct {
		Parts []struct {
			Name string `json:"name"`
			Text string `json:"text"`
		} `json:"parts"`
	}
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatal(err)
	}
	for _, p := range view.Parts {
		if p.Name == name {
			return p.Text
		}
	}
	t.Fatalf("context view missing %s", name)
	return ""
}

func TestRawContextIsTheLastSentRequest(t *testing.T) {
	a := &Agent{Cfg: Config{Web: false}, MCP: testMCPServer("mail", "Mail.", mcpToolsNamed("list"))}
	before, err := json.Marshal(a.rawContextView())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(before, []byte("mcp__mail__list")) {
		t.Fatal("unsent MCP schema shown in the context view")
	}
	sent, err := json.Marshal(map[string]any{
		"instructions": "Mail is connected.",
		"tools":        []map[string]any{{"name": "mcp__mail__list", "description": "List mail"}},
		"input":        []map[string]any{{"role": "user", "content": "any mail?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	a.noteSentChat("", sent)
	toolsText := contextPart(t, a, "tools")
	if !strings.Contains(toolsText, "mcp__mail__list") {
		t.Fatal("sent MCP tool missing from the context view")
	}
	if strings.Contains(toolsText, "list_credentials") {
		t.Fatal("a tool that was not in the request was added")
	}
	if !strings.Contains(contextPart(t, a, "input 1"), "any mail?") {
		t.Fatal("sent transcript missing")
	}
}
