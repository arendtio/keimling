package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// respondChatCompletions speaks the classic /chat/completions format. Models
// that only offer it, most open-weight ones, become usable without the
// Responses API.
func (c *Client) respondChatCompletions(ctx context.Context, model, instructions string, history []json.RawMessage, tools []map[string]any) (Response, error) {
	system, turns, err := historyTurns(history)
	if err != nil {
		return Response{}, err
	}
	messages := make([]map[string]any, 0, len(turns)+1)
	if s := strings.TrimSpace(strings.TrimSpace(instructions) + "\n\n" + system); s != "" {
		messages = append(messages, map[string]any{"role": "system", "content": s})
	}
	messages = append(messages, chatCompletionsMessages(turns)...)
	body := map[string]any{
		"model":    model,
		"stream":   true,
		"messages": messages,
	}
	if len(tools) > 0 {
		body["tools"] = chatCompletionsTools(tools)
		body["tool_choice"] = "auto"
	}
	b, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	c.builtChatBody = append([]byte(nil), b...)
	res, err := c.openChatAPI(ctx, c.apiEndpoint(model), b, apiChatCompletions)
	if err != nil {
		return Response{}, err
	}
	if res.StatusCode/100 == 2 && strings.Contains(strings.ToLower(res.Header.Get("Content-Type")), "text/event-stream") {
		defer res.Body.Close()
		return c.readChatCompletionsStream(model, res.Body)
	}
	data, err := chatReadAnswer(res, apiChatCompletions)
	if err != nil {
		return Response{}, err
	}
	return c.chatCompletionsJSON(model, data)
}

func chatCompletionsTools(tools []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		fn := map[string]any{"name": tool["name"], "description": tool["description"], "parameters": tool["parameters"]}
		if strict, ok := tool["strict"]; ok {
			fn["strict"] = strict
		}
		out = append(out, map[string]any{"type": "function", "function": fn})
	}
	return out
}

func chatImageURL(p chatPart) string {
	if p.ImageURL != "" {
		return p.ImageURL
	}
	return "data:" + p.ImageMIME + ";base64," + p.ImageData
}

func chatCompletionsMessages(turns []chatTurn) []map[string]any {
	out := make([]map[string]any, 0, len(turns))
	for _, turn := range turns {
		switch {
		case len(turn.Results) > 0:
			for _, res := range turn.Results {
				out = append(out, map[string]any{"role": "tool", "tool_call_id": res.CallID, "content": res.Output})
			}
		case turn.Role == "assistant":
			msg := map[string]any{"role": "assistant"}
			if turn.Text != "" {
				msg["content"] = turn.Text
			}
			if len(turn.Calls) > 0 {
				calls := make([]map[string]any, 0, len(turn.Calls))
				for _, call := range turn.Calls {
					calls = append(calls, map[string]any{
						"id":       call.ID,
						"type":     "function",
						"function": map[string]any{"name": call.Name, "arguments": call.Arguments},
					})
				}
				msg["tool_calls"] = calls
			}
			out = append(out, msg)
		default:
			if len(turn.Parts) == 0 {
				out = append(out, map[string]any{"role": "user", "content": turn.Text})
				continue
			}
			parts := make([]map[string]any, 0, len(turn.Parts)+1)
			if turn.Text != "" {
				parts = append(parts, map[string]any{"type": "text", "text": turn.Text})
			}
			for _, p := range turn.Parts {
				parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]any{"url": chatImageURL(p)}})
			}
			out = append(out, map[string]any{"role": "user", "content": parts})
		}
	}
	return out
}

// readChatCompletionsStream accumulates deltas; only the finished answer is
// returned, so nothing half-streamed reaches the journal.
func (c *Client) readChatCompletionsStream(model string, r io.Reader) (Response, error) {
	start := time.Now()
	var text strings.Builder
	var thinking strings.Builder
	visible := false
	var inTokens, outTokens int64
	calls := map[int]*chatCall{}
	var order []int
	err := sseEvents(r, func(event, payload string) error {
		payload = strings.TrimSpace(payload)
		if payload == "" || payload == "[DONE]" {
			return nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					Reasoning        string `json:"reasoning"`
					ToolCalls        []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int64 `json:"prompt_tokens"`
				CompletionTokens int64 `json:"completion_tokens"`
			} `json:"usage"`
			Error json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return fmt.Errorf("invalid Chat Completions SSE event: %w", err)
		}
		if len(chunk.Error) > 0 && string(chunk.Error) != "null" {
			return fmt.Errorf("Chat Completions error: %s", truncate(string(chunk.Error), 800))
		}
		if chunk.Usage != nil {
			inTokens, outTokens = chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens
		}
		for _, choice := range chunk.Choices {
			if s := choice.Delta.ReasoningContent + choice.Delta.Reasoning; s != "" {
				thinking.WriteString(s)
				if c.StreamThinking != nil {
					c.StreamThinking(s)
				}
				c.streamProgress(start, s, "thinking")
			}
			if s := choice.Delta.Content; s != "" {
				visible = true
				text.WriteString(s)
				if c.StreamText != nil {
					c.StreamText(s)
				}
				c.streamProgress(start, s, "text")
			}
			for _, tc := range choice.Delta.ToolCalls {
				call, ok := calls[tc.Index]
				if !ok {
					call = &chatCall{}
					calls[tc.Index] = call
					order = append(order, tc.Index)
				}
				if tc.ID != "" {
					call.ID = tc.ID
				}
				if tc.Function.Name != "" {
					call.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					call.Arguments += tc.Function.Arguments
					c.streamProgress(start, tc.Function.Arguments, "tool")
				}
			}
		}
		return nil
	})
	if err != nil {
		return Response{}, err
	}
	if text.Len() == 0 && len(order) == 0 {
		return Response{}, errors.New("Chat Completions stream ended without content")
	}
	callsOut := make([]chatCall, 0, len(order))
	for _, i := range order {
		callsOut = append(callsOut, *calls[i])
	}
	return c.finishChatResponse(model, text.String(), callsOut, inTokens, outTokens, visible), nil
}

func (c *Client) chatCompletionsJSON(model string, data []byte) (Response, error) {
	var doc struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return Response{}, fmt.Errorf("invalid Chat Completions answer: %w", err)
	}
	if len(doc.Choices) == 0 {
		return Response{}, errors.New("Chat Completions answer has no choices")
	}
	msg := doc.Choices[0].Message
	var calls []chatCall
	for _, tc := range msg.ToolCalls {
		calls = append(calls, chatCall{ID: tc.ID, Name: tc.Function.Name, Arguments: tc.Function.Arguments})
	}
	var in, out int64
	if doc.Usage != nil {
		in, out = doc.Usage.PromptTokens, doc.Usage.CompletionTokens
	}
	return c.finishChatResponse(model, msg.Content, calls, in, out, false), nil
}
