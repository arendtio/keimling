package main

import (
	"fmt"
	"os"
	"strings"
)

// cliWrite prints to the terminal only (no GUI SSE line mirror).
func (a *Agent) cliWrite(s string) {
	if s == "" {
		return
	}
	fmt.Fprint(os.Stdout, s)
}

// presentAssistant publishes a structured GUI message and ANSI for the CLI.
// When delta is true, text is a streaming chunk (CLI does not print yet).
// When replace is true, GUI should replace the open streaming bubble with full text.
func (a *Agent) presentAssistant(text string, delta, replace bool) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if !delta && strings.TrimSpace(text) == "" && !replace {
		return
	}
	payload := map[string]any{
		"role":    "assistant",
		"text":    text,
		"delta":   delta,
		"replace": replace,
	}
	if !delta && a.Client != nil {
		payload["used_pct"] = a.contextBudget().UsedPct
	}
	a.emitJSON("message", payload)
	if delta {
		return
	}
	formatted := formatTerminal(text)
	if formatted == "" {
		return
	}
	if !strings.HasPrefix(formatted, "\n") {
		formatted = "\n" + formatted
	}
	if !strings.HasSuffix(formatted, "\n") {
		formatted += "\n"
	}
	a.cliWrite(formatted)
}

// presentToolStart announces a tool invocation to GUI and CLI.
func (a *Agent) presentToolStart(name, args string) {
	cmd := toolCommandLabel(name, args)
	a.emitJSON("tool", map[string]any{
		"phase":     "start",
		"name":      name,
		"command":   cmd,
		"input":     args,
		"thread_id": a.Cfg.ThreadID,
	})
	banner := renderToolCall(name, args)
	if banner != "" && !strings.HasSuffix(banner, "\n") {
		banner += "\n"
	}
	a.cliWrite(banner)
}

// presentToolReview tells GUI and CLI that a call is with the approval review
// and has not started yet. The card shows this state until the gate decides.
func (a *Agent) presentToolReview(name, args string) {
	if quietTool(name) {
		return
	}
	a.emitJSON("tool", map[string]any{
		"phase":     "review",
		"name":      name,
		"command":   toolCommandLabel(name, args),
		"input":     args,
		"thread_id": a.Cfg.ThreadID,
	})
}

// presentToolAsk tells the GUI that the call waits for a human now, so the card
// does not look like it is already running.
func (a *Agent) presentToolAsk(name, args string) {
	if quietTool(name) {
		return
	}
	a.emitJSON("tool", map[string]any{
		"phase":     "ask",
		"name":      name,
		"command":   toolCommandLabel(name, args),
		"input":     args,
		"thread_id": a.Cfg.ThreadID,
	})
}

// presentToolGo tells the GUI that the gate passed and the call starts now, so
// the time spent in review can be shown apart from the time the call runs.
func (a *Agent) presentToolGo(name, args string) {
	if quietTool(name) {
		return
	}
	a.emitJSON("tool", map[string]any{
		"phase":     "go",
		"name":      name,
		"command":   toolCommandLabel(name, args),
		"input":     args,
		"thread_id": a.Cfg.ThreadID,
	})
}

// quietTool has no card of its own: its effect is the plan or the finish line.
func quietTool(name string) bool {
	return name == "finish_task" || name == "update_plan"
}

// presentToolResult publishes tool output to GUI and CLI.
// When filtered is true, result is the model-facing text and raw is the full shell output.
// When filtered is false, result is the single display output and raw should be empty.
func (a *Agent) presentToolResult(name, command, input, result, raw string, ok, filtered bool) {
	payload := map[string]any{
		"phase":     "result",
		"name":      name,
		"command":   command,
		"input":     input,
		"result":    result,
		"ok":        ok,
		"filtered":  filtered,
		"thread_id": a.Cfg.ThreadID,
	}
	if filtered && strings.TrimSpace(raw) != "" {
		payload["raw"] = raw
	}
	a.emitJSON("tool", payload)
}

// note records an outcome in the transcript. The GUI shows it, the journal
// keeps it across reload, and the next model call sees it.
func (a *Agent) note(text string) {
	text = strings.TrimSpace(text)
	if a == nil || text == "" {
		return
	}
	collapsed := noticeCollapsed(text)
	a.emitJSON("message", map[string]any{
		"role":      "system",
		"text":      text,
		"collapsed": collapsed,
	})
	a.cliWrite("\n" + text + "\n")
	if a.Journal == nil {
		return
	}
	_ = a.add(map[string]any{"type": "keimling_notice", "text": text})
}

func noticeCollapsed(text string) bool {
	return strings.Contains(text, "\n") || len(text) > 180
}

// presentFinish shows the task completion summary on both surfaces.
func (a *Agent) presentFinish(status, summary string) {
	title := "Task complete"
	if status == "blocked" {
		title = "Action needed"
	}
	body := strings.TrimSpace(summary)
	msg := "**" + title + "**"
	if body != "" {
		msg += "\n\n" + body
	}
	done := map[string]any{
		"role": "assistant",
		"text": msg,
	}
	if a.Client != nil {
		done["used_pct"] = a.contextBudget().UsedPct
	}
	a.emitJSON("message", done)
	out := renderFinish(status, summary)
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	a.cliWrite(out)
}

// presentContextGrowth tells the sidebar plant the context fill now, so it
// can move while tools are still running.
func (a *Agent) presentContextGrowth() {
	if a == nil {
		return
	}
	a.emitJSON("growth", map[string]any{"used_pct": a.contextBudget().UsedPct})
}
