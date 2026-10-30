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
	a.emitJSON("message", map[string]any{
		"role":    "assistant",
		"text":    text,
		"delta":   delta,
		"replace": replace,
	})
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
		"phase":   "start",
		"name":    name,
		"command": cmd,
	})
	banner := renderToolCall(name, args)
	if banner != "" && !strings.HasSuffix(banner, "\n") {
		banner += "\n"
	}
	a.cliWrite(banner)
}

// presentToolResult publishes tool output to GUI and CLI.
// When filtered is true, result is the model-facing text and raw is the full shell output.
// When filtered is false, result is the single display output and raw should be empty.
func (a *Agent) presentToolResult(name, command, result, raw string, ok, filtered bool) {
	payload := map[string]any{
		"phase":    "result",
		"name":     name,
		"command":  command,
		"result":   result,
		"ok":       ok,
		"filtered": filtered,
	}
	if filtered && strings.TrimSpace(raw) != "" {
		payload["raw"] = raw
	}
	a.emitJSON("tool", payload)
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
	a.emitJSON("message", map[string]any{
		"role": "assistant",
		"text": msg,
	})
	out := renderFinish(status, summary)
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	a.cliWrite(out)
}
