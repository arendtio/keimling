package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func renderToolCall(name, args string) string {
	var v map[string]json.RawMessage
	if json.Unmarshal([]byte(args), &v) != nil {
		return "\033[35m⚙ " + name + "\033[0m"
	}
	switch name {
	case "exec":
		var cmd string
		_ = json.Unmarshal(v["command"], &cmd)
		var interactive bool
		_ = json.Unmarshal(v["interactive"], &interactive)
		label := "\033[35m❯\033[0m"
		if interactive && !nonInteractiveDiagnostic(cmd) {
			label = "\033[35m⌨ interactive\033[0m"
		}
		return "\n" + label + " " + highlightCommand(truncate(cmd, 1600))
	case "read_file", "edit_file", "write_file", "share_file", "show_artifact":
		var path string
		_ = json.Unmarshal(v["path"], &path)
		icons := map[string]string{"read_file": "◉ Read", "edit_file": "✎ Edit", "write_file": "✎ Write", "share_file": "📎 Share", "show_artifact": "▣ Artifact"}
		return "\n\033[35m" + icons[name] + "\033[0m " + truncate(path, 260)
	default:
		return "\n\033[35m⚙ " + name + "\033[0m"
	}
}

// approvalPreviewLines caps how much of a pending change is shown before approval.
const approvalPreviewLines = 80

// renderApprovalPreview prints the actual change a tool would make so the user
// approves content instead of a bare tool name. It returns "" for tools whose
// arguments already speak for themselves.
func renderApprovalPreview(name, args string) string {
	switch name {
	case "edit_file":
		var x struct {
			Path       string `json:"path"`
			OldString  string `json:"old_string"`
			NewString  string `json:"new_string"`
			ReplaceAll bool   `json:"replace_all"`
		}
		if json.Unmarshal([]byte(args), &x) != nil || x.OldString == "" {
			return ""
		}
		return renderEditPreview(x.Path, x.OldString, x.NewString, x.ReplaceAll)
	case "write_file":
		var x struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if json.Unmarshal([]byte(args), &x) != nil {
			return ""
		}
		return renderWritePreview(x.Path, x.Content)
	}
	return ""
}

func renderEditPreview(path, oldString, newString string, replaceAll bool) string {
	var b strings.Builder
	scope := "one match"
	if replaceAll {
		scope = "all matches"
	}
	fmt.Fprintf(&b, "  \033[1m%s\033[0m \033[2m(%s)\033[0m\n", path, scope)
	appendPreviewLines(&b, oldString, '-')
	appendPreviewLines(&b, newString, '+')
	return b.String()
}

func renderWritePreview(path, content string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  \033[1m%s\033[0m \033[2m(write)\033[0m\n", path)
	appendPreviewLines(&b, content, '+')
	return b.String()
}

func appendPreviewLines(b *strings.Builder, text string, mark rune) {
	if text == "" {
		return
	}
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, line := range lines {
		if i >= approvalPreviewLines || b.Len() > 8000 {
			fmt.Fprintf(b, "  \033[2m… %d more line(s)\033[0m\n", len(lines)-i)
			break
		}
		color := "\033[32m"
		if mark == '-' {
			color = "\033[31m"
		}
		fmt.Fprintf(b, "  %s%c%s\033[0m\n", color, mark, line)
	}
}

func renderPlan(plan []PlanStep) string {
	var b strings.Builder
	count := 0
	for _, p := range plan {
		if p.Status == "done" {
			count++
		}
	}
	fmt.Fprintf(&b, "\n\033[36m◆ Plan\033[0m  \033[2m%d/%d complete\033[0m\n", count, len(plan))
	for _, p := range plan {
		symbol, color := "○", "\033[2m"
		switch p.Status {
		case "done":
			symbol, color = "✓", "\033[32m"
		case "in_progress":
			symbol, color = "◉", "\033[36m"
		}
		fmt.Fprintf(&b, "  %s%s\033[0m %s\n", color, symbol, p.Step)
	}
	return b.String()
}

func renderFinish(status, summary string) string {
	if status == "done" {
		return "\n\033[32m✓ Task complete\033[0m\n" + renderMarkdown(strings.TrimSpace(summary))
	}
	return "\n\033[33m! Action needed\033[0m\n" + renderMarkdown(strings.TrimSpace(summary))
}
