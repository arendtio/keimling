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
		if name == "apply_patch" {
			var patch string
			_ = json.Unmarshal(v["patch"], &patch)
			targets, err := patchTargets(patch)
			if err != nil {
				return "\n\033[35m✎ Patch\033[0m " + truncate(firstLine(patch), 260) + " \033[2m(invalid)\033[0m"
			}
			labels := make([]string, 0, len(targets))
			for _, t := range targets {
				label := t.path
				switch {
				case t.create:
					label += " \033[32m(new)\033[0m"
				case t.delete:
					label += " \033[31m(delete)\033[0m"
				}
				labels = append(labels, label)
			}
			return "\n\033[35m✎ Patch\033[0m " + truncate(strings.Join(labels, ", "), 260)
		}
		var path string
		_ = json.Unmarshal(v["path"], &path)
		icons := map[string]string{"read_file": "◉ Read", "replace_lines": "✎ Replace", "apply_patch": "✎ Patch"}
		return "\n\033[35m" + icons[name] + "\033[0m " + truncate(path, 260)
	default:
		return "\n\033[35m⚙ " + name + "\033[0m"
	}
}

// firstLine keeps an approval preview short when the argument is a multi-line patch.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i]) + " …"
	}
	return s
}

// approvalPreviewLines caps how much of a pending change is shown before approval.
const approvalPreviewLines = 80

// renderApprovalPreview prints the actual change a tool would make so the user
// approves content instead of a bare tool name. It returns "" for tools whose
// arguments already speak for themselves.
func renderApprovalPreview(name, args string) string {
	switch name {
	case "apply_patch":
		var x struct {
			Patch string `json:"patch"`
		}
		if json.Unmarshal([]byte(args), &x) != nil || strings.TrimSpace(x.Patch) == "" {
			return ""
		}
		return renderPatchPreview(x.Patch)
	case "replace_lines":
		var x struct {
			Path      string `json:"path"`
			StartLine int    `json:"start_line"`
			EndLine   int    `json:"end_line"`
			Content   string `json:"content"`
		}
		if json.Unmarshal([]byte(args), &x) != nil || x.Content == "" {
			return ""
		}
		return renderReplacementPreview(x.Path, x.StartLine, x.EndLine, x.Content)
	}
	return ""
}

func renderPatchPreview(patch string) string {
	lines := strings.Split(strings.TrimRight(patch, "\n"), "\n")
	added, removed := 0, 0
	for _, line := range lines {
		if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "+"):
			added++
		case strings.HasPrefix(line, "-"):
			removed++
		}
	}
	var b strings.Builder
	for i, line := range lines {
		if i >= approvalPreviewLines || b.Len() > 8000 {
			fmt.Fprintf(&b, "  \033[2m… %d more line(s)\033[0m\n", len(lines)-i)
			break
		}
		switch {
		case strings.HasPrefix(line, "+++ "), strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "diff --git "), strings.HasPrefix(line, "index "):
			fmt.Fprintf(&b, "  \033[1m%s\033[0m\n", line)
		case strings.HasPrefix(line, "@@"):
			fmt.Fprintf(&b, "  \033[36m%s\033[0m\n", line)
		case strings.HasPrefix(line, "+"):
			fmt.Fprintf(&b, "  \033[32m%s\033[0m\n", line)
		case strings.HasPrefix(line, "-"):
			fmt.Fprintf(&b, "  \033[31m%s\033[0m\n", line)
		default:
			fmt.Fprintf(&b, "  \033[2m%s\033[0m\n", line)
		}
	}
	fmt.Fprintf(&b, "  \033[2m%d addition(s), %d removal(s)\033[0m\n", added, removed)
	return b.String()
}

func renderReplacementPreview(path string, start, end int, content string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  \033[1m%s\033[0m \033[2mlines %d-%d\033[0m\n", path, start, end)
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	for i, line := range lines {
		if i >= approvalPreviewLines || b.Len() > 8000 {
			fmt.Fprintf(&b, "  \033[2m… %d more line(s)\033[0m\n", len(lines)-i)
			break
		}
		fmt.Fprintf(&b, "  \033[32m+%s\033[0m\n", line)
	}
	return b.String()
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
