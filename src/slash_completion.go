package main

import (
	"sort"
	"strings"
)

var slashCommands = []string{
	"/help", "/status", "/skills", "/mcp", "/mode", "/compact", "/newchat", "/stop",
	"/resume", "/continue", "/continue-new-model", "/autopilot", "/voice", "/exit",
}

// completeSlash returns the longest common prefix and the possible matches.
// Completion is restricted to a command word at the start of the draft.
func completeSlash(draft string, cursor int) (string, int, []string) {
	r := []rune(draft)
	if cursor < 1 || cursor > len(r) {
		return draft, cursor, nil
	}
	before := string(r[:cursor])
	if !strings.HasPrefix(before, "/") || strings.ContainsAny(before, " \t\n") {
		return draft, cursor, nil
	}
	matches := []string{}
	for _, cmd := range slashCommands {
		if strings.HasPrefix(cmd, before) {
			matches = append(matches, cmd)
		}
	}
	if len(matches) == 0 {
		return draft, cursor, nil
	}
	sort.Strings(matches)
	prefix := matches[0]
	for _, m := range matches[1:] {
		for !strings.HasPrefix(m, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	next := prefix + string(r[cursor:])
	return next, len([]rune(prefix)), matches
}
