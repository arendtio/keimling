package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// delegateDefaultMaxSteps is the global default when a caller has no positive
// max_steps. A delegate uses the configured max_steps, not a smaller ceiling.
const delegateDefaultMaxSteps = 200

func delegateStepLimit(maxSteps int) int {
	if maxSteps < 1 {
		return delegateDefaultMaxSteps
	}
	return maxSteps
}

const delegateAgentRules = " You are a bounded subagent. Use the available tools to complete the task. You cannot call delegate or finish_task. Do not ask the user a question: when a tool is denied, report that and continue with what you can verify. Return one concise answer that names what you checked."

func toolDefinitionName(t map[string]any) string {
	name, _ := t["name"].(string)
	return strings.TrimSpace(name)
}

// delegateToolGrant is the caller's grant without the tools a subagent must
// not use: delegate would recurse, and finish_task belongs to the parent task.
// A nil caller grant means every tool, so the result is an explicit list.
func delegateToolGrant(web bool, extra []map[string]any, grant []string) []string {
	skip := map[string]bool{"delegate": true, "finish_task": true}
	seen := map[string]bool{}
	out := []string{}
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || skip[name] || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	if grant == nil {
		for _, t := range definitions(web, extra) {
			add(toolDefinitionName(t))
		}
		return out
	}
	for _, name := range grant {
		add(name)
	}
	return out
}

// runDelegateAgent runs a nested agent loop with the caller's tools. Only the
// final answer returns to the parent; the subagent keeps a temporary journal so
// its tool results do not flood the parent transcript.
func (a *Agent) runDelegateAgent(ctx context.Context, args DelegateArgs, model, effort string) (string, error) {
	if a == nil || a.Client == nil {
		return "", errors.New("no API client")
	}
	task := strings.TrimSpace(args.Task)
	if task == "" {
		return "", errors.New("delegate task required")
	}
	if c := strings.TrimSpace(args.Context); c != "" {
		task = "Context:\n" + c + "\n\nTask:\n" + task
	}

	tmp, err := os.CreateTemp("", "keimling-delegate-*.jsonl")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(path)
	journal, err := OpenJournal(path, false)
	if err != nil {
		return "", err
	}
	defer journal.Close()

	sub := &Agent{
		Cfg:              a.Cfg,
		Client:           a.Client,
		Journal:          journal,
		Skills:           a.Skills,
		MCP:              a.MCP,
		SessionCtx:       a.SessionCtx,
		Emit:             a.Emit,
		ApprovalCh:       a.ApprovalCh,
		Shares:           a.Shares,
		Artifacts:        a.Artifacts,
		Credentials:      a.Credentials,
		dirs:             a.dirTable(),
		toolData:         newDataStore(),
		skipToolFindings: true,
	}
	sub.Cfg.SessionPath = path
	// MCP calls share the caller's cap and stdio locks.
	sub.mcpSlots = a.mcpGate()
	sub.mcpGateOnce.Do(func() {})
	if err := sub.add(map[string]any{"role": "user", "content": task}); err != nil {
		return "", err
	}

	extra := sub.chatExtraTools()
	grant := delegateToolGrant(sub.Cfg.Web, extra, sub.instanceToolGrant())
	instructions := sharedWorkInstructions() + delegateAgentRules
	steps := delegateStepLimit(sub.Cfg.MaxSteps)
	last := ""
	for step := 0; step < steps; step++ {
		resp, err := sub.Client.Respond(ctx, model, effort, sub.History, sub.Cfg.Web, instructions, extra, grant)
		if err != nil {
			return "", err
		}
		var items []Item
		for _, raw := range resp.Output {
			if err := sub.add(json.RawMessage(raw)); err != nil {
				return "", err
			}
			var item Item
			if json.Unmarshal(raw, &item) != nil {
				continue
			}
			switch item.Type {
			case "message":
				if text := itemOutputText(item); text != "" {
					last = text
				}
			case "function_call":
				items = append(items, item)
			}
		}
		if len(items) == 0 {
			if strings.TrimSpace(last) == "" {
				return "", errors.New("delegate returned no answer")
			}
			return strings.TrimSpace(last), nil
		}
		if err := sub.runFunctionCalls(ctx, "", nil, items); err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("delegate reached its step limit (%d)", steps)
}

func itemOutputText(item Item) string {
	var b strings.Builder
	for _, part := range item.Content {
		if part.Type == "output_text" && part.Text != "" {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(part.Text)
		}
	}
	return b.String()
}
