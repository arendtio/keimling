package main

import (
	"context"
	"fmt"
	"strings"
)

// toolPurpose classifies tools for the cheap post-tool retain prompt.
func toolPurpose(name string) string {
	switch strings.TrimSpace(name) {
	case "exec", "edit_file", "write_file", "apply_patch", "update_plan", "finish_task", "share_file", "show_artifact":
		return "mutate"
	default:
		return "query"
	}
}

func shouldRetainTool(name, result string) bool {
	result = strings.TrimSpace(result)
	if result == "" || strings.HasPrefix(result, "DENIED") || strings.HasPrefix(result, "SKIPPED") {
		return false
	}
	switch name {
	case "update_plan", "finish_task":
		return false // handled directly on the charter
	case "search_history", "read_history":
		return false // archival navigation; findings belong in charter from real work
	}
	return true
}

// retainToolFinding asks a tiny model call to fold a tool result into the charter.
func (a *Agent) retainToolFinding(ctx context.Context, name, args, result string) {
	if a == nil || a.Client == nil || !shouldRetainTool(name, result) {
		return
	}
	purpose := toolPurpose(name)
	c := a.ensureCharter()
	instr := "Maintain a project charter. Reply with at most two short lines. English. No markdown fences. If nothing durable, reply exactly SKIP."
	input := fmt.Sprintf(
		"Tool purpose=%s name=%s.\nWrite ONE short finding (≤80 tokens): mutate → 'changed X → success/fail + key side effect'; query → 'answered Q with fact F'. Drop bulk stdout.\nOptional second line starting with Q: for an unresolved blocker.\nArgs: %s\nResult: %s\n",
		purpose, name, truncate(args, 400), truncate(result, 1200),
	)
	text, err := a.Client.compactionAsk(ctx, a.compactionModel(), instr, input, 120)
	if err != nil {
		// Deterministic fallback so a retain failure never blocks the agent loop.
		c.AppendFinding(fmt.Sprintf("%s (%s): %s", name, purpose, truncate(oneLine(result), 160)))
		_ = a.persistCharter()
		return
	}
	text = strings.TrimSpace(text)
	if text == "" || strings.EqualFold(text, "SKIP") {
		return
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.EqualFold(line, "SKIP") {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(line), "Q:") {
			c.AppendQuestion(strings.TrimSpace(line[2:]))
			continue
		}
		c.AppendFinding(truncate(line, 240))
	}
	_ = a.persistCharter()
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}
