package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// anthropicMaxTokens is required by the Messages API and caps one answer. It
// stays well under every Claude model's output limit.
const anthropicMaxTokens = 32000

// respondMessages speaks the Anthropic Messages format, which Claude and some
// Qwen models require.
func (c *Client) respondMessages(ctx context.Context, model, instructions string, history []json.RawMessage, tools []map[string]any) (Response, error) {
	system, turns, err := historyTurns(history)
	if err != nil {
		return Response{}, err
	}
	body := map[string]any{
		"model":      model,
		"max_tokens": anthropicMaxTokens,
		"stream":     true,
		"messages":   messagesTurns(turns),
	}
	if s := strings.TrimSpace(strings.TrimSpace(instructions) + "\n\n" + system); s != "" {
		body["system"] = s
	}
	if len(tools) > 0 {
		body["tools"] = messagesTools(tools)
	}
	b, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	c.builtChatBody = append([]byte(nil), b...)
	res, err := c.openChatAPI(ctx, c.apiEndpoint(model), b, apiMessages)
	if err != nil {
		return Response{}, err
	}
	if res.StatusCode/100 == 2 && strings.Contains(strings.ToLower(res.Header.Get("Content-Type")), "text/event-stream") {
		defer res.Body.Close()
		return c.readMessagesStream(model, res.Body)
	}
	data, err := chatReadAnswer(res, apiMessages)
	if err != nil {
		return Response{}, err
	}
	return c.messagesJSON(model, data)
}

func messagesTools(tools []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		out = append(out, map[string]any{
			"name":         tool["name"],
			"description":  tool["description"],
			"input_schema": tool["parameters"],
		})
	}
	return out
}

func messagesTurns(turns []chatTurn) []map[string]any {
	out := make([]map[string]any, 0, len(turns))
	for _, turn := range turns {
		switch {
		case len(turn.Results) > 0:
			blocks := make([]map[string]any, 0, len(turn.Results))
			for _, res := range turn.Results {
				blocks = append(blocks, map[string]any{
					"type":        "tool_result",
					"tool_use_id": res.CallID,
					"content":     res.Output,
				})
			}
			out = append(out, map[string]any{"role": "user", "content": blocks})
		case turn.Role == "assistant":
			blocks := make([]map[string]any, 0, len(turn.Calls)+1)
			if turn.Text != "" {
				blocks = append(blocks, map[string]any{"type": "text", "text": turn.Text})
			}
			for _, call := range turn.Calls {
				var input any = map[string]any{}
				if strings.TrimSpace(call.Arguments) != "" {
					if err := json.Unmarshal([]byte(call.Arguments), &input); err != nil {
						input = map[string]any{}
					}
				}
				blocks = append(blocks, map[string]any{"type": "tool_use", "id": call.ID, "name": call.Name, "input": input})
			}
			out = append(out, map[string]any{"role": "assistant", "content": blocks})
		default:
			blocks := make([]map[string]any, 0, len(turn.Parts)+1)
			if turn.Text != "" {
				blocks = append(blocks, map[string]any{"type": "text", "text": turn.Text})
			}
			for _, p := range turn.Parts {
				if p.ImageData != "" {
					blocks = append(blocks, map[string]any{
						"type":   "image",
						"source": map[string]any{"type": "base64", "media_type": p.ImageMIME, "data": p.ImageData},
					})
					continue
				}
				blocks = append(blocks, map[string]any{"type": "image", "source": map[string]any{"type": "url", "url": p.ImageURL}})
			}
			content := any(turn.Text)
			if len(blocks) > 0 {
				content = blocks
			}
			out = append(out, map[string]any{"role": "user", "content": content})
		}
	}
	return out
}

// readMessagesStream accumulates content blocks. Claude sends text, thinking
// and tool arguments as separate delta kinds under one block index.
func (c *Client) readMessagesStream(model string, r io.Reader) (Response, error) {
	start := time.Now()
	type block struct {
		kind string
		text strings.Builder
	}
	blocks := map[int]*block{}
	var order []int
	var inTokens, outTokens int64
	visible := false
	blockFor := func(index int, kind string) *block {
		b, ok := blocks[index]
		if !ok {
			b = &block{kind: kind}
			blocks[index] = b
			order = append(order, index)
		}
		return b
	}
	err := sseEvents(r, func(event, payload string) error {
		var e struct {
			Type    string `json:"type"`
			Index   int    `json:"index"`
			Message *struct {
				Usage struct {
					InputTokens  int64 `json:"input_tokens"`
					OutputTokens int64 `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message"`
			ContentBlock *struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"content_block"`
			Delta *struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				Thinking    string `json:"thinking"`
				PartialJSON string `json:"partial_json"`
			} `json:"delta"`
			Usage *struct {
				OutputTokens int64 `json:"output_tokens"`
			} `json:"usage"`
			Error *struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(payload), &e); err != nil {
			return fmt.Errorf("invalid Messages SSE event: %w", err)
		}
		typ := e.Type
		if typ == "" {
			typ = event
		}
		switch typ {
		case "error":
			msg := "unknown"
			if e.Error != nil {
				msg = e.Error.Type + ": " + e.Error.Message
			}
			return fmt.Errorf("Messages API error: %s", truncate(msg, 800))
		case "message_start":
			if e.Message != nil {
				inTokens = e.Message.Usage.InputTokens
				outTokens = e.Message.Usage.OutputTokens
			}
		case "content_block_start":
			if e.ContentBlock != nil {
				b := blockFor(e.Index, e.ContentBlock.Type)
				if e.ContentBlock.Type == "tool_use" {
					b.kind = "tool_use"
					b.text.WriteString(e.ContentBlock.ID + "\x00" + e.ContentBlock.Name + "\x00")
				}
			}
		case "content_block_delta":
			if e.Delta == nil {
				return nil
			}
			switch e.Delta.Type {
			case "text_delta":
				b := blockFor(e.Index, "text")
				b.text.WriteString(e.Delta.Text)
				visible = true
				if c.StreamText != nil {
					c.StreamText(e.Delta.Text)
				}
				c.streamProgress(start, e.Delta.Text, "text")
			case "thinking_delta":
				blockFor(e.Index, "thinking").text.WriteString(e.Delta.Thinking)
				if c.StreamThinking != nil {
					c.StreamThinking(e.Delta.Thinking)
				}
				c.streamProgress(start, e.Delta.Thinking, "thinking")
			case "input_json_delta":
				blockFor(e.Index, "tool_use").text.WriteString(e.Delta.PartialJSON)
				c.streamProgress(start, e.Delta.PartialJSON, "tool")
			}
		case "message_delta":
			if e.Usage != nil && e.Usage.OutputTokens > 0 {
				outTokens = e.Usage.OutputTokens
			}
		}
		return nil
	})
	if err != nil {
		return Response{}, err
	}
	var text strings.Builder
	var calls []chatCall
	for _, index := range order {
		b := blocks[index]
		switch b.kind {
		case "text":
			text.WriteString(b.text.String())
		case "tool_use":
			parts := strings.SplitN(b.text.String(), "\x00", 3)
			if len(parts) < 2 {
				continue
			}
			args := "{}"
			if len(parts) == 3 && strings.TrimSpace(parts[2]) != "" {
				args = parts[2]
			}
			calls = append(calls, chatCall{ID: parts[0], Name: parts[1], Arguments: args})
		}
	}
	if text.Len() == 0 && len(calls) == 0 {
		return Response{}, fmt.Errorf("Messages stream ended without content")
	}
	return c.finishChatResponse(model, text.String(), calls, inTokens, outTokens, visible), nil
}

func (c *Client) messagesJSON(model string, data []byte) (Response, error) {
	var doc struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return Response{}, fmt.Errorf("invalid Messages answer: %w", err)
	}
	var text strings.Builder
	var calls []chatCall
	for _, block := range doc.Content {
		switch block.Type {
		case "text":
			text.WriteString(block.Text)
		case "tool_use":
			args := "{}"
			if len(block.Input) > 0 {
				args = string(block.Input)
			}
			calls = append(calls, chatCall{ID: block.ID, Name: block.Name, Arguments: args})
		}
	}
	return c.finishChatResponse(model, text.String(), calls, doc.Usage.InputTokens, doc.Usage.OutputTokens, false), nil
}
