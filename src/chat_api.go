package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Wire formats Keimling can speak. The configured base URL decides, so a
// provider profile only pastes the endpoint its vendor documents: an
// ".../anthropic/v1" URL talks Messages, ".../google/v1beta" talks Gemini, a
// URL ending in "/chat/completions" talks Chat Completions, anything else is
// the OpenAI Responses API Keimling was built on.
const (
	apiResponses       = "responses"
	apiChatCompletions = "chat-completions"
	apiMessages        = "messages"
	apiGemini          = "gemini"
)

// shortcut: side tasks (compaction, filter, transcription) still post Responses
// requests on the provider base URL, so a side model from another API family
// needs a provider profile of its own; upgrade by routing them through the
// same dispatch.
func apiFamily(baseURL string) string {
	u := strings.ToLower(strings.TrimSpace(baseURL))
	switch {
	case strings.Contains(u, "anthropic"), strings.HasSuffix(u, "/messages"):
		return apiMessages
	case strings.Contains(u, "google"), strings.Contains(u, "generativelanguage"):
		return apiGemini
	case strings.HasSuffix(u, "/chat/completions"), strings.HasSuffix(u, "/completions"):
		return apiChatCompletions
	}
	return apiResponses
}

// apiEndpoint is the full request URL for one model.
func (c *Client) apiEndpoint(model string) string {
	switch apiFamily(c.BaseURL) {
	case apiMessages:
		if strings.HasSuffix(strings.TrimRight(c.BaseURL, "/"), "/messages") {
			return strings.TrimRight(c.BaseURL, "/")
		}
		return c.endpoint("messages")
	case apiGemini:
		base := strings.TrimRight(c.BaseURL, "/")
		if i := strings.LastIndex(base, "/models/"); i >= 0 {
			base = base[:i]
		}
		return base + "/models/" + model + ":streamGenerateContent?alt=sse"
	case apiChatCompletions:
		if strings.HasSuffix(strings.TrimRight(c.BaseURL, "/"), "/chat/completions") {
			return strings.TrimRight(c.BaseURL, "/")
		}
		return c.endpoint("chat/completions")
	}
	return c.endpoint("responses")
}

// apiAuth sets the headers the family's vendor expects. Every family takes the
// same bearer token; Anthropic's own API also wants its version header.
func (c *Client) apiAuth(req *http.Request, family string) {
	c.authorize(req)
	req.Header.Set("Content-Type", "application/json")
	if family == apiMessages {
		req.Header.Set("anthropic-version", "2023-06-01")
		if strings.HasSuffix(strings.ToLower(req.URL.Hostname()), "anthropic.com") {
			req.Header.Set("x-api-key", c.APIKey)
		}
	}
}

// openChatAPI sends one request and hands back the open response so streaming
// answers stay live. The body is never mutated between retries.
func (c *Client) openChatAPI(ctx context.Context, url string, body []byte, family string) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= 3; attempt++ {
		request, e := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
		if e != nil {
			return nil, e
		}
		c.apiAuth(request, family)
		streamHTTP := *c.HTTP
		streamHTTP.Timeout = 0
		res, e := streamHTTP.Do(request)
		if e == nil {
			return res, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		lastErr = e
		if !retryableNetworkError(e) || attempt == 3 {
			return nil, lastErr
		}
		delay := time.Second * time.Duration(1<<attempt)
		fmt.Printf("\nAPI: transient error; retry %d/3 in %s\n", attempt+1, delay)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil, lastErr
}

// sseEvents walks a text/event-stream body and hands each event's joined data
// lines to consume.
func sseEvents(r io.Reader, consume func(event, payload string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 16<<20)
	var eventName string
	var dataLines []string
	flush := func() error {
		defer func() { eventName = "" }()
		if len(dataLines) == 0 {
			return nil
		}
		payload := strings.Join(dataLines, "\n")
		dataLines = nil
		return consume(eventName, payload)
	}
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if err := flush(); err != nil {
				return err
			}
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return flush()
}

// chatReadAnswer drains a non-streaming answer or the error text.
func chatReadAnswer(res *http.Response, family string) ([]byte, error) {
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode/100 != 2 {
		return data, chatStatusError(family, res.StatusCode, data)
	}
	return data, nil
}

// chatPart is one content part, stripped to what the non-Responses formats need.
type chatPart struct {
	Text      string
	ImageMIME string
	ImageData string
	ImageURL  string
}

type chatCall struct {
	ID        string
	Name      string
	Arguments string
}

type chatResult struct {
	CallID string
	Name   string
	Output string
}

// chatTurn is one message. Tool calls stay on the assistant turn that produced
// them and tool results travel as their own user turn, which is what Messages
// and Gemini require.
type chatTurn struct {
	Role    string // user | assistant
	Text    string
	Parts   []chatPart
	Calls   []chatCall
	Results []chatResult
	IsTool  bool
}

// historyTurns converts normalized history into a system prompt plus turns.
// Consecutive assistant items share a turn, so text plus tool calls stay in one
// message.
func historyTurns(history []json.RawMessage) (string, []chatTurn, error) {
	input, err := normalizeResponseInput(history)
	if err != nil {
		return "", nil, err
	}
	var system []string
	var turns []chatTurn
	callNames := map[string]string{}
	open := func(role string) *chatTurn {
		if n := len(turns); n > 0 && turns[n-1].Role == role && !turns[n-1].IsTool {
			return &turns[n-1]
		}
		turns = append(turns, chatTurn{Role: role})
		return &turns[len(turns)-1]
	}
	for _, raw := range input {
		var item struct {
			Type    string          `json:"type"`
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
			Name    string          `json:"name"`
			CallID  string          `json:"call_id"`
			Args    string          `json:"arguments"`
			Output  json.RawMessage `json:"output"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return "", nil, fmt.Errorf("history item: %w", err)
		}
		switch item.Type {
		case "", "message":
			parts, text, err := chatParts(item.Content)
			if err != nil {
				return "", nil, err
			}
			if role := strings.ToLower(strings.TrimSpace(item.Role)); role == "developer" || role == "system" {
				if text != "" {
					system = append(system, text)
				}
				continue
			}
			role := "user"
			if strings.EqualFold(item.Role, "assistant") {
				role = "assistant"
			}
			t := open(role)
			t.Parts = append(t.Parts, parts...)
			if text != "" {
				if t.Text != "" {
					t.Text += "\n"
				}
				t.Text += text
			}
		case "function_call":
			t := open("assistant")
			callNames[item.CallID] = item.Name
			t.Calls = append(t.Calls, chatCall{ID: item.CallID, Name: item.Name, Arguments: item.Args})
		case "function_call_output":
			var out string
			if json.Unmarshal(item.Output, &out) != nil {
				out = string(item.Output)
			}
			turns = append(turns, chatTurn{Role: "user", IsTool: true, Results: []chatResult{{CallID: item.CallID, Name: callNames[item.CallID], Output: out}}})
		}
	}
	kept := turns[:0]
	for _, turn := range turns {
		if turn.Text == "" && len(turn.Parts) == 0 && len(turn.Calls) == 0 && len(turn.Results) == 0 {
			continue
		}
		kept = append(kept, turn)
	}
	return strings.Join(system, "\n\n"), kept, nil
}

// chatParts reads a content value that is either a string or a part array.
// Images arrive as data URLs, which every family can embed as base64.
func chatParts(raw json.RawMessage) ([]chatPart, string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, "", nil
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, "", err
		}
		return nil, s, nil
	}
	var rows []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		ImageURL string `json:"image_url"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, "", err
	}
	var parts []chatPart
	var texts []string
	for _, row := range rows {
		switch row.Type {
		case "input_text", "output_text", "text":
			if row.Text != "" {
				texts = append(texts, row.Text)
			}
		case "input_image":
			p := chatPart{ImageURL: row.ImageURL}
			if mime, data, ok := splitDataURL(row.ImageURL); ok {
				p.ImageMIME, p.ImageData, p.ImageURL = mime, data, ""
			}
			if p.ImageData != "" || p.ImageURL != "" {
				parts = append(parts, p)
			}
		}
	}
	return parts, strings.Join(texts, "\n"), nil
}

func splitDataURL(u string) (mime, data string, ok bool) {
	if !strings.HasPrefix(u, "data:") {
		return "", "", false
	}
	rest := strings.TrimPrefix(u, "data:")
	i := strings.Index(rest, ";base64,")
	if i < 0 {
		return "", "", false
	}
	return rest[:i], rest[i+len(";base64,"):], true
}

func chatMessageItem(text string) map[string]any {
	return map[string]any{
		"type":    "message",
		"role":    "assistant",
		"content": []map[string]any{{"type": "output_text", "text": text}},
	}
}

func chatCallItem(callID, name, args string) map[string]any {
	if strings.TrimSpace(args) == "" {
		args = "{}"
	}
	return map[string]any{"type": "function_call", "call_id": callID, "name": name, "arguments": args}
}

func newCallID() string {
	return newStampedID("call")
}

// streamProgress reports streamed rune counts for the GUI throughput meter.
func (c *Client) streamProgress(start time.Time, s, kind string) {
	if n := len([]rune(s)); n > 0 && c.StreamProgress != nil {
		c.StreamProgress(n, time.Since(start), kind)
	}
}

// finishChatResponse assembles the Response the agent commits. Status stays
// "completed", so a foreign finish reason never looks like a truncated answer.
func (c *Client) finishChatResponse(model, text string, calls []chatCall, inTokens, outTokens int64, streamed bool) Response {
	out := Response{Status: "completed", StreamedText: streamed}
	if strings.TrimSpace(text) != "" {
		b, _ := json.Marshal(chatMessageItem(text))
		out.Output = append(out.Output, b)
	}
	for _, call := range calls {
		id := call.ID
		if id == "" {
			id = newCallID()
		}
		b, _ := json.Marshal(chatCallItem(id, call.Name, call.Arguments))
		out.Output = append(out.Output, b)
	}
	out.Usage.InputTokens = inTokens
	out.Usage.OutputTokens = outTokens
	c.InputTokens += inTokens
	c.OutputTokens += outTokens
	c.spendUsage("chat", model, inTokens, outTokens)
	if inTokens > 0 || outTokens > 0 {
		c.LastMainUsage = MainUsage{Available: true, InputTokens: inTokens, OutputTokens: outTokens}
	}
	return out
}

func chatStatusError(family string, status int, data []byte) error {
	return fmt.Errorf("%s API HTTP %d: %s", family, status, truncate(string(data), 1000))
}
