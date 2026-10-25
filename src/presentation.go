package main

import (
	"encoding/json"
	"fmt"
	"regexp"
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
	case "read_file", "replace_lines", "apply_patch":
		var path string
		_ = json.Unmarshal(v["path"], &path)
		icons := map[string]string{"read_file": "◉ Read", "replace_lines": "✎ Replace", "apply_patch": "✎ Patch"}
		return "\n\033[35m" + icons[name] + "\033[0m " + truncate(path, 260)
	default:
		return "\n\033[35m⚙ " + name + "\033[0m"
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

var (
	mdBold    = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	mdCode    = regexp.MustCompile("`([^`\\n]+)`")
	mdItalic  = regexp.MustCompile(`(^|[^*])\*([^*\n]+)\*`)
	mdHeading = regexp.MustCompile(`(?m)^#{1,6} +(.+)$`)
)

func renderMarkdown(s string) string {
	s = mdLink.ReplaceAllString(s, "$1 ($2)")
	s = mdHeading.ReplaceAllString(s, "\033[1m$1\033[0m")
	s = mdBold.ReplaceAllString(s, "\033[1m$1\033[0m")
	s = mdCode.ReplaceAllString(s, "\033[36m$1\033[0m")
	s = mdItalic.ReplaceAllString(s, "$1\033[3m$2\033[0m")
	return s
}
