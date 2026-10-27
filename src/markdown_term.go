package main

import (
	"regexp"
	"strings"
)

// Minimal Markdown → ANSI for the CLI. No parser dependency: only the subset
// models actually use in chat (emphasis, code, headings, lists, links).
// Unknown syntax is left as plain text.

var (
	mdLink   = regexp.MustCompile(`\[([^]\n]+)\]\((https?://[^)\s]+)\)`)
	mdBold   = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	mdItalic = regexp.MustCompile(`(^|[^*])\*([^*\n]+)\*`)
	mdCode   = regexp.MustCompile("`([^`\\n]+)`")
)

func renderMarkdown(s string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	var b strings.Builder
	b.Grow(len(s) + 32)
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			if inFence {
				lang := strings.TrimSpace(strings.TrimPrefix(line, "```"))
				if lang != "" {
					b.WriteString("\033[2m")
					b.WriteString(lang)
					b.WriteString("\033[0m\n")
				}
			} else {
				// closing fence: blank visual break
			}
			if i < len(lines)-1 {
				b.WriteByte('\n')
			}
			continue
		}
		if inFence {
			b.WriteString("\033[36m")
			b.WriteString(line)
			b.WriteString("\033[0m")
			if i < len(lines)-1 {
				b.WriteByte('\n')
			}
			continue
		}
		b.WriteString(renderMarkdownLine(line))
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func renderMarkdownLine(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]

	if h := headingText(trimmed); h != "" {
		return indent + "\033[1m" + renderMarkdownInline(h) + "\033[0m"
	}
	if rest, ok := listPrefix(trimmed); ok {
		return indent + "\033[2m•\033[0m " + renderMarkdownInline(rest)
	}
	return indent + renderMarkdownInline(trimmed)
}

func headingText(trimmed string) string {
	if len(trimmed) == 0 || trimmed[0] != '#' {
		return ""
	}
	i := 0
	for i < len(trimmed) && trimmed[i] == '#' {
		i++
		if i > 6 {
			return ""
		}
	}
	if i >= len(trimmed) || trimmed[i] != ' ' {
		return ""
	}
	return strings.TrimSpace(trimmed[i:])
}

func listPrefix(trimmed string) (rest string, ok bool) {
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
		return trimmed[2:], true
	}
	// ordered: 1. 2. …
	i := 0
	for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
		i++
	}
	if i > 0 && i+1 < len(trimmed) && trimmed[i] == '.' && trimmed[i+1] == ' ' {
		return trimmed[i+2:], true
	}
	return "", false
}

func renderMarkdownInline(s string) string {
	s = mdLink.ReplaceAllString(s, "$1 ($2)")
	s = mdBold.ReplaceAllString(s, "\033[1m$1\033[0m")
	s = mdCode.ReplaceAllString(s, "\033[36m$1\033[0m")
	s = mdItalic.ReplaceAllString(s, "$1\033[3m$2\033[0m")
	return s
}
