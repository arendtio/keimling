package main

import (
	"regexp"
	"strings"
	"unicode"
)

// Highlight only the terminal representation. Never transform executable bytes.
var heredocPattern = regexp.MustCompile(`<<-?\s*['"]?([A-Za-z_][A-Za-z_0-9.]*)['"]?`)
var jsonKeyPattern = regexp.MustCompile(`^\s*:`)
var pythonWords = words("def class import from return if elif else for while try except finally with as lambda raise pass async await yield del global nonlocal assert break continue in is not and or")
var bashWords = words("if then fi for do done while case esac function sudo export set echo printf python python3 bash sh local readonly source trap until select in")
var literals = words("True False None true false null")

func words(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(s) {
		out[w] = true
	}
	return out
}
func colored(s, code string) string { return "\x1b[" + code + "m" + s + "\x1b[0m" }
func identStart(r rune) bool        { return unicode.IsLetter(r) || r == '_' }
func identContinue(r rune) bool     { return identStart(r) || unicode.IsDigit(r) }
func syntaxLine(line, lang string) string {
	r := []rune(line)
	var b strings.Builder
	for i := 0; i < len(r); {
		start := i
		c := r[i]
		if c == '#' && lang != "json" && (i == 0 || r[i-1] != '\\') {
			b.WriteString(colored(string(r[i:]), "90"))
			break
		}
		if c == '\'' || c == '"' {
			quote := c
			i++
			for i < len(r) {
				if r[i] == '\\' && i+1 < len(r) {
					i += 2
					continue
				}
				if r[i] == quote {
					i++
					break
				}
				i++
			}
			token := string(r[start:i])
			color := "32"
			if lang == "json" && jsonKeyPattern.MatchString(string(r[i:])) {
				color = "36"
			}
			b.WriteString(colored(token, color))
			continue
		}
		if unicode.IsDigit(c) && (i == 0 || !identContinue(r[i-1])) {
			i++
			for i < len(r) && (unicode.IsDigit(r[i]) || r[i] == '.' || r[i] == '_') {
				i++
			}
			b.WriteString(colored(string(r[start:i]), "33"))
			continue
		}
		if identStart(c) {
			i++
			for i < len(r) && identContinue(r[i]) {
				i++
			}
			token := string(r[start:i])
			if literals[token] {
				b.WriteString(colored(token, "35"))
			} else if (lang == "python" && pythonWords[token]) || (lang == "bash" && bashWords[token]) {
				b.WriteString(colored(token, map[bool]string{true: "35", false: "34"}[lang == "python"]))
			} else if lang == "python" && i < len(r) && r[i] == '(' {
				b.WriteString(colored(token, "36"))
			} else {
				b.WriteString(token)
			}
			continue
		}
		if lang == "bash" && c == '$' {
			i++
			if i < len(r) && r[i] == '{' {
				i++
				for i < len(r) && r[i] != '}' {
					i++
				}
				if i < len(r) {
					i++
				}
			} else {
				for i < len(r) && identContinue(r[i]) {
					i++
				}
			}
			if i > start+1 {
				b.WriteString(colored(string(r[start:i]), "33"))
			} else {
				b.WriteRune(c)
			}
			continue
		}
		b.WriteRune(c)
		i++
	}
	return b.String()
}
func highlightCommand(command string) string {
	if !strings.Contains(command, "\n") {
		return command
	}
	lines := strings.Split(command, "\n")
	language := "bash"
	delim := ""
	for i, line := range lines {
		if delim != "" && strings.TrimSpace(line) == delim {
			lines[i] = colored(line, "90")
			delim = ""
			language = "bash"
			continue
		}
		current := language
		if delim == "" {
			if m := heredocPattern.FindStringSubmatch(line); len(m) > 0 {
				delim = m[1]
				lower := strings.ToLower(line)
				if strings.Contains(lower, "python") {
					language = "python"
				} else if strings.Contains(lower, "json") || strings.HasSuffix(strings.ToLower(delim), "json") {
					language = "json"
				} else {
					language = "bash"
				}
			}
		}
		// Header is shell syntax; heredoc body uses the selected language.
		lines[i] = syntaxLine(line, current)
	}
	return strings.Join(lines, "\n")
}
