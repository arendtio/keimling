package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	}, false, "")
	if err != nil {
		t.Fatal(err)
	}
}
