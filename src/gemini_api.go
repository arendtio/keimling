package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// respondGemini speaks the Google generateContent format, which the Gemini
// models require. The model name travels in the URL, not the body.
func (c *Client) respondGemini(ctx context.Context, model, instructions string, history []json.RawMessage, tools []map[string]any) (Response, error) {
	system, turns, err := historyTurns(history)
	if err != nil {
		return Response{}, err
	}
	body := map[string]any{"contents": geminiContents(turns)}
	if s := strings.TrimSpace(strings.TrimSpace(instructions) + "\n\n" + system); s != "" {
		body["systemInstruction"] = map[string]any{"parts": []map[string]any{{"text": s}}}
	}
	if len(tools) > 0 {
		body["tools"] = []map[string]any{{"functionDeclarations": geminiTools(tools)}}
	}
	b, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	c.builtChatBody = append([]byte(nil), b...)
	res, err := c.openChatAPI(ctx, c.apiEndpoint(model), b, apiGemini)
	if err != nil {
		return Response{}, err
	}
	if res.StatusCode/100 == 2 && strings.Contains(strings.ToLower(res.Header.Get("Content-Type")), "text/event-stream") {
		defer res.Body.Close()
		return c.readGeminiStream(model, res.Body)
	}
	data, err := chatReadAnswer(res, apiGemini)
	if err != nil {
		return Response{}, err
	}
	return c.geminiJSON(model, data)
}

// geminiTools drops schema keywords Gemini rejects. additionalProperties and
// strict come from Keimling's OpenAI tool definitions.
func geminiTools(tools []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		out = append(out, map[string]any{
			"name":        tool["name"],
			"description": tool["description"],
			"parameters":  geminiSchema(tool["parameters"]),
		})
	}
	return out
}

func geminiSchema(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			switch key {
			case "additionalProperties", "strict":
				continue
			case "properties":
				props := map[string]any{}
				if m, ok := value.(map[string]any); ok {
					for name, sub := range m {
						props[name] = geminiSchema(sub)
					}
				}
				out[key] = props
			case "items":
				out[key] = geminiSchema(value)
			default:
				out[key] = value
			}
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, geminiSchema(item))
		}
		return out
	}
	return v
}

func geminiContents(turns []chatTurn) []map[string]any {
	out := make([]map[string]any, 0, len(turns))
	for _, turn := range turns {
		switch {
		case len(turn.Results) > 0:
			parts := make([]map[string]any, 0, len(turn.Results))
			for _, res := range turn.Results {
				name := res.Name
				if name == "" {
					name = res.CallID
				}
				parts = append(parts, map[string]any{
					"functionResponse": map[string]any{"name": name, "response": map[string]any{"output": res.Output}},
				})
			}
			out = append(out, map[string]any{"role": "user", "parts": parts})
		default:
			role := "user"
			if turn.Role == "assistant" {
				role = "model"
			}
			parts := make([]map[string]any, 0, len(turn.Calls)+len(turn.Parts)+1)
			if turn.Text != "" {
				parts = append(parts, map[string]any{"text": turn.Text})
			}
			for _, p := range turn.Parts {
				if p.ImageData != "" {
					parts = append(parts, map[string]any{"inlineData": map[string]any{"mimeType": p.ImageMIME, "data": p.ImageData}})
				}
			}
			for _, call := range turn.Calls {
				args := map[string]any{}
				if strings.TrimSpace(call.Arguments) != "" {
					_ = json.Unmarshal([]byte(call.Arguments), &args)
				}
				parts = append(parts, map[string]any{"functionCall": map[string]any{"name": call.Name, "args": args}})
			}
			if len(parts) == 0 {
				parts = append(parts, map[string]any{"text": ""})
			}
			out = append(out, map[string]any{"role": role, "parts": parts})
		}
	}
	return out
}

// geminiChunk is one streamed GenerateContentResponse.
type geminiChunk struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text         string `json:"text"`
				Thought      bool   `json:"thought"`
				FunctionCall *struct {
					Name string          `json:"name"`
					Args json.RawMessage `json:"args"`
				} `json:"functionCall"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int64 `json:"promptTokenCount"`
		CandidatesTokenCount int64 `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func (c *Client) readGeminiStream(model string, r io.Reader) (Response, error) {
	start := time.Now()
	var text, thinking strings.Builder
	var calls []chatCall
	var inTokens, outTokens int64
	visible := false
	err := sseEvents(r, func(event, payload string) error {
		if strings.TrimSpace(payload) == "" {
			return nil
		}
		var chunk geminiChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return fmt.Errorf("invalid Gemini SSE event: %w", err)
		}
		if err := c.consumeGeminiChunk(&chunk, &text, &thinking, &calls, &inTokens, &outTokens, &visible, start); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Response{}, err
	}
	if text.Len() == 0 && len(calls) == 0 {
		return Response{}, fmt.Errorf("Gemini stream ended without content")
	}
	return c.finishChatResponse(model, text.String(), calls, inTokens, outTokens, visible), nil
}

func (c *Client) consumeGeminiChunk(chunk *geminiChunk, text, thinking *strings.Builder, calls *[]chatCall, inTokens, outTokens *int64, visible *bool, start time.Time) error {
	if chunk.Error != nil {
		return fmt.Errorf("Gemini API error: %s", truncate(chunk.Error.Status+": "+chunk.Error.Message, 800))
	}
	if chunk.UsageMetadata != nil {
		*inTokens = chunk.UsageMetadata.PromptTokenCount
		*outTokens = chunk.UsageMetadata.CandidatesTokenCount
	}
	for _, cand := range chunk.Candidates {
		for _, part := range cand.Content.Parts {
			if part.FunctionCall != nil {
				args := "{}"
				if len(part.FunctionCall.Args) > 0 {
					args = string(part.FunctionCall.Args)
				}
				*calls = append(*calls, chatCall{Name: part.FunctionCall.Name, Arguments: args})
				c.streamProgress(start, args, "tool")
				continue
			}
			if part.Text == "" {
				continue
			}
			if part.Thought {
				thinking.WriteString(part.Text)
				if c.StreamThinking != nil {
					c.StreamThinking(part.Text)
				}
				c.streamProgress(start, part.Text, "thinking")
				continue
			}
			*visible = true
			text.WriteString(part.Text)
			if c.StreamText != nil {
				c.StreamText(part.Text)
			}
			c.streamProgress(start, part.Text, "text")
		}
	}
	return nil
}

func (c *Client) geminiJSON(model string, data []byte) (Response, error) {
	var chunk geminiChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return Response{}, fmt.Errorf("invalid Gemini answer: %w", err)
	}
	var text, thinking strings.Builder
	var calls []chatCall
	var inTokens, outTokens int64
	visible := false
	if err := c.consumeGeminiChunk(&chunk, &text, &thinking, &calls, &inTokens, &outTokens, &visible, time.Now()); err != nil {
		return Response{}, err
	}
	return c.finishChatResponse(model, text.String(), calls, inTokens, outTokens, false), nil
}
