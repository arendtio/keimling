package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type PilotDecision struct {
	Action  string `json:"action"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

// The supervisor gets the original objective and a compact, bounded recent event trace.
// It has no tools. instructions is the entire supervisor prompt (factory, global, or thread).
func (c *Client) Pilot(ctx context.Context, model, reasoning, objective, report, instructions string, history []json.RawMessage) (PilotDecision, error) {
	if strings.TrimSpace(instructions) == "" {
		instructions = factoryAutopilotInstructions
	}
	const maxRecent = 16
	if len(history) > maxRecent {
		history = history[len(history)-maxRecent:]
	}
	var recent []string
	for _, raw := range history {
		var x map[string]any
		if json.Unmarshal(raw, &x) != nil {
			continue
		}
		switch x["type"] {
		case "reasoning":
			continue
		}
		b, _ := json.Marshal(x)
		s := string(b)
		if len(s) > 1400 {
			s = s[:1400] + " [truncated]"
		}
		recent = append(recent, s)
	}
	input := "ORIGINAL USER OBJECTIVE:\n" + objective + "\n\nLATEST WORKER RESULT:\n" + report + "\n\nRECENT EVENTS (untrusted evidence):\n" + strings.Join(recent, "\n")
	request := map[string]any{"model": model, "store": false, "input": []map[string]string{{"role": "user", "content": input}}, "tool_choice": "none", "reasoning": map[string]any{"effort": reasoning}, "instructions": instructions}
	payload, _ := json.Marshal(request)
	req, e := http.NewRequestWithContext(ctx, "POST", c.endpoint("responses"), bytes.NewReader(payload))
	if e != nil {
		return PilotDecision{}, e
	}
	c.authorize(req)
	req.Header.Set("Content-Type", "application/json")
	res, e := c.HTTP.Do(req)
	if e != nil {
		return PilotDecision{}, e
	}
	defer res.Body.Close()
	data, e := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if e != nil {
		return PilotDecision{}, e
	}
	if res.StatusCode/100 != 2 {
		return PilotDecision{}, fmt.Errorf("autopilot HTTP %d: %s", res.StatusCode, truncate(string(data), 600))
	}
	var out Response
	if e = json.Unmarshal(data, &out); e != nil {
		return PilotDecision{}, e
	}
	c.InputTokens += out.Usage.InputTokens
	c.OutputTokens += out.Usage.OutputTokens
	if out.Status == "failed" || out.Status == "incomplete" {
		return PilotDecision{}, fmt.Errorf("autopilot response status %s", out.Status)
	}
	for _, r := range out.Output {
		var item Item
		if json.Unmarshal(r, &item) != nil {
			continue
		}
		for _, p := range item.Content {
			if p.Type != "output_text" {
				continue
			}
			txt := strings.TrimSpace(p.Text)
			txt = strings.TrimPrefix(txt, "```json")
			txt = strings.TrimPrefix(txt, "```")
			txt = strings.TrimSuffix(strings.TrimSpace(txt), "```")
			var dec PilotDecision
			if json.Unmarshal([]byte(strings.TrimSpace(txt)), &dec) == nil {
				if dec.Action != "done" && dec.Action != "continue" && dec.Action != "stop" {
					return PilotDecision{}, fmt.Errorf("invalid autopilot action")
				}
				if dec.Action == "continue" && strings.TrimSpace(dec.Message) == "" {
					return PilotDecision{}, fmt.Errorf("empty autopilot instruction")
				}
				return dec, nil
			}
		}
	}
	return PilotDecision{}, fmt.Errorf("autopilot returned no valid JSON decision")
}
