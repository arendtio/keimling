package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIFamilyFromBaseURL(t *testing.T) {
	cases := map[string]string{
		"https://api.openai.com/v1":                                apiResponses,
		"https://opencode.ai/inference/openai/v1":                  apiResponses,
		"https://opencode.ai/inference/anthropic/v1":               apiMessages,
		"https://api.anthropic.com/v1":                             apiMessages,
		"https://opencode.ai/inference/google/v1beta":              apiGemini,
		"https://generativelanguage.googleapis.com/v1beta":         apiGemini,
		"https://opencode.ai/inference/openai/v1/chat/completions": apiChatCompletions,
	}
	for base, want := range cases {
		if got := apiFamily(base); got != want {
			t.Fatalf("%s: got %s want %s", base, got, want)
		}
	}
	c := &Client{BaseURL: "https://opencode.ai/inference/google/v1beta"}
	if got := c.apiEndpoint("gemini-4-argon"); got != "https://opencode.ai/inference/google/v1beta/models/gemini-4-argon:streamGenerateContent?alt=sse" {
		t.Fatalf("gemini endpoint %s", got)
	}
	c = &Client{BaseURL: "https://opencode.ai/inference/anthropic/v1"}
	if got := c.apiEndpoint("claude-opus-5"); got != "https://opencode.ai/inference/anthropic/v1/messages" {
		t.Fatalf("messages endpoint %s", got)
	}
}

func testHistory() []json.RawMessage {
	return []json.RawMessage{
		json.RawMessage(`{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"},{"type":"input_image","image_url":"data:image/png;base64,AAAA"}]}`),
		json.RawMessage(`{"type":"function_call","name":"exec","call_id":"call_1","arguments":"{\"command\":\"ls\"}"}`),
		json.RawMessage(`{"type":"function_call_output","call_id":"call_1","output":"a.txt"}`),
	}
}

func TestClientHeaders(t *testing.T) {
	var ua, session, auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua, session, auth = r.Header.Get("User-Agent"), r.Header.Get("x-opencode-session"), r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]}}]}\n\n")
	}))
	defer srv.Close()
	c := NewClient("key")
	c.BaseURL = srv.URL + "/google/v1beta"
	c.SessionID = "thread-1"
	if _, err := c.Respond(context.Background(), "gemini-4-argon", "", testHistory(), false, "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if ua != "keimling/1.0" || auth != "Bearer key" {
		t.Fatalf("ua %q auth %q", ua, auth)
	}
	if session != "" {
		t.Fatalf("session header sent to a foreign host: %q", session)
	}
	for host, want := range map[string]bool{
		"opencode.ai":           true,
		"inference.opencode.ai": true,
		"opencode.ai.evil.test": false,
		"notopencode.ai":        false,
		"api.openai.com":        false,
	} {
		if got := openCodeSessionHost(host); got != want {
			t.Fatalf("%s: %v", host, got)
		}
	}
}

func TestHistoryTurnsKeepsCallNames(t *testing.T) {
	_, turns, err := historyTurns(testHistory())
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 3 {
		t.Fatalf("%#v", turns)
	}
	if len(turns[1].Calls) != 1 || turns[1].Calls[0].Name != "exec" {
		t.Fatalf("call %#v", turns[1])
	}
	if len(turns[2].Results) != 1 || turns[2].Results[0].Name != "exec" || turns[2].Results[0].Output != "a.txt" {
		t.Fatalf("result %#v", turns[2])
	}
}

func sseServer(t *testing.T, wantPath string, events []string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("path %s want %s", r.URL.Path, wantPath)
		}
		if r.Header.Get("Authorization") != "Bearer key" {
			t.Errorf("auth header %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, e := range events {
			_, _ = io.WriteString(w, e)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRespondChatCompletionsStream(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		if r.URL.Path != "/openai/v1/chat/completions" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_9\",\"function\":{\"name\":\"exec\",\"arguments\":\"{\\\"command\\\":\\\"pwd\\\"}\"}}]}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":11,\"completion_tokens\":7}}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	var streamed strings.Builder
	c := NewClient("key")
	c.BaseURL = srv.URL + "/openai/v1/chat/completions"
	c.StreamText = func(s string) { streamed.WriteString(s) }
	resp, err := c.Respond(context.Background(), "glm-5.3", "high", testHistory(), false, "be brief", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "completed" || resp.Usage.InputTokens != 11 || resp.Usage.OutputTokens != 7 {
		t.Fatalf("%#v", resp)
	}
	if streamed.String() != "Hello" || !resp.StreamedText {
		t.Fatalf("stream %q", streamed.String())
	}
	if len(resp.Output) != 2 {
		t.Fatalf("output %#v", resp.Output)
	}
	var call Item
	if err := json.Unmarshal(resp.Output[1], &call); err != nil || call.Type != "function_call" || call.Name != "exec" || call.Arguments != `{"command":"pwd"}` {
		t.Fatalf("call %#v %v", call, err)
	}
	var sent struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
		Tools []struct {
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Model != "glm-5.3" || len(sent.Messages) < 2 || sent.Messages[0].Role != "system" {
		t.Fatalf("sent %#v", sent)
	}
	if len(sent.Tools) == 0 || sent.Tools[0].Function.Name == "" {
		t.Fatalf("tools %#v", sent.Tools)
	}
}

func TestRespondMessagesStream(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		if r.URL.Path != "/anthropic/v1/messages" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Errorf("missing anthropic-version")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		events := []string{
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":21,\"output_tokens\":0}}}\n\n",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\"}}\n\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_1\",\"name\":\"exec\"}}\n\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"command\\\":\\\"ls\\\"}\"}}\n\n",
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":9}}\n\n",
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
		}
		for _, e := range events {
			_, _ = io.WriteString(w, e)
		}
	}))
	defer srv.Close()
	c := NewClient("key")
	c.BaseURL = srv.URL + "/anthropic/v1"
	resp, err := c.Respond(context.Background(), "claude-opus-5", "high", testHistory(), false, "be brief", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Usage.InputTokens != 21 || resp.Usage.OutputTokens != 9 || len(resp.Output) != 2 {
		t.Fatalf("%#v", resp)
	}
	var call Item
	if err := json.Unmarshal(resp.Output[1], &call); err != nil || call.CallID != "toolu_1" || call.Name != "exec" || call.Arguments != `{"command":"ls"}` {
		t.Fatalf("call %#v %v", call, err)
	}
	var sent struct {
		MaxTokens int              `json:"max_tokens"`
		System    string           `json:"system"`
		Stream    bool             `json:"stream"`
		Messages  []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatal(err)
	}
	if sent.MaxTokens == 0 || !sent.Stream || !strings.Contains(sent.System, "be brief") {
		t.Fatalf("sent %#v", sent)
	}
	// The tool result must travel as a user turn with a tool_result block.
	last, _ := json.Marshal(sent.Messages[len(sent.Messages)-1])
	if !strings.Contains(string(last), "tool_result") || !strings.Contains(string(last), "call_1") {
		t.Fatalf("last message %s", last)
	}
}

func TestRespondGeminiJSONArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"candidates":[{"content":{"parts":[{"text":"Hi"}]}}]},{"candidates":[{"content":{"parts":[{"text":" there"}]}}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":2}}]`)
	}))
	defer srv.Close()
	c := NewClient("key")
	c.BaseURL = srv.URL + "/google/v1beta"
	resp, err := c.Respond(context.Background(), "gemini-4-argon", "", testHistory(), false, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var msg Item
	if err := json.Unmarshal(resp.Output[0], &msg); err != nil || len(msg.Content) != 1 || msg.Content[0].Text != "Hi there" {
		t.Fatalf("text %#v %v", msg, err)
	}
	if resp.Usage.InputTokens != 3 || resp.Usage.OutputTokens != 2 {
		t.Fatalf("usage %#v", resp.Usage)
	}
}

func TestRespondChatCompletionsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok","tool_calls":[{"id":"call_2","function":{"name":"exec","arguments":"{}"}}]}}],"usage":{"prompt_tokens":4,"completion_tokens":1}}`)
	}))
	defer srv.Close()
	c := NewClient("key")
	c.BaseURL = srv.URL + "/openai/v1/chat/completions"
	resp, err := c.Respond(context.Background(), "glm-5.3", "", testHistory(), false, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Output) != 2 || resp.Usage.InputTokens != 4 {
		t.Fatalf("%#v", resp)
	}
}

func TestRespondGeminiStream(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		if r.URL.Path != "/google/v1beta/models/gemini-4-argon:streamGenerateContent" || r.URL.Query().Get("alt") != "sse" {
			t.Errorf("path %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"Hi\"}]}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"functionCall\":{\"name\":\"exec\",\"args\":{\"command\":\"ls\"}}}]}}],\"usageMetadata\":{\"promptTokenCount\":5,\"candidatesTokenCount\":4}}\n\n")
	}))
	defer srv.Close()
	c := NewClient("key")
	c.BaseURL = srv.URL + "/google/v1beta"
	resp, err := c.Respond(context.Background(), "gemini-4-argon", "high", testHistory(), false, "be brief", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Usage.InputTokens != 5 || resp.Usage.OutputTokens != 4 || len(resp.Output) != 2 {
		t.Fatalf("%#v", resp)
	}
	var call Item
	if err := json.Unmarshal(resp.Output[1], &call); err != nil || call.Name != "exec" || call.Arguments != `{"command":"ls"}` {
		t.Fatalf("call %#v %v", call, err)
	}
	var sent struct {
		SystemInstruction struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"systemInstruction"`
		Contents []struct {
			Role  string `json:"role"`
			Parts []struct {
				FunctionResponse *struct {
					Name string `json:"name"`
				} `json:"functionResponse"`
			} `json:"parts"`
		} `json:"contents"`
		Tools []struct {
			FunctionDeclarations []struct {
				Name       string         `json:"name"`
				Parameters map[string]any `json:"parameters"`
			} `json:"functionDeclarations"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sent.SystemInstruction.Parts[0].Text, "be brief") {
		t.Fatalf("system %#v", sent.SystemInstruction)
	}
	if len(sent.Contents) < 3 || sent.Contents[len(sent.Contents)-1].Parts[0].FunctionResponse == nil || sent.Contents[len(sent.Contents)-1].Parts[0].FunctionResponse.Name != "exec" {
		t.Fatalf("contents %#v", sent.Contents)
	}
	if len(sent.Tools) == 0 || len(sent.Tools[0].FunctionDeclarations) == 0 {
		t.Fatalf("tools %#v", sent.Tools)
	}
	if _, has := sent.Tools[0].FunctionDeclarations[0].Parameters["additionalProperties"]; has {
		t.Fatalf("gemini schema kept additionalProperties")
	}
}
