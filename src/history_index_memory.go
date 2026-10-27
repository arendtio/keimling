package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Memory on a task chapter: how compaction treated that span (if at all).
const (
	memoryRaw       = "raw"
	memoryKeep      = "keep"
	memoryDistill   = "distill"
	memorySummarize = "summarize"
	memoryCollapse  = "collapse"
)

var sectionMemoryMarker = regexp.MustCompile(`(?m)^\[section (?:depth=\d+ )?(\d+)-(\d+): .*? \| (keep|distill|summarize|collapse)\]$`)
var topLevelSectionMarker = regexp.MustCompile(`(?m)^\[section depth=1 (\d+)-(\d+): (.*?) \| (keep|distill|summarize|collapse)\]$`)

func memoryRank(m string) int {
	switch m {
	case memoryCollapse:
		return 4
	case memorySummarize:
		return 3
	case memoryDistill:
		return 2
	case memoryKeep:
		return 1
	default:
		return 0
	}
}

// applyMemoryFromCompaction marks task sections from semantic section markers in a summary.
// applyVisibleChaptersFromCompaction replaces the visible task list with top-level
// compact chapters when denser/raw compaction produced framed section markers.
// Prior task chapters overlapping those ranges are hidden (GUI must not show them).
func applyVisibleChaptersFromCompaction(idx *HistoryIndex, summary string) {
	if idx == nil || strings.TrimSpace(summary) == "" {
		return
	}
	type top struct {
		start, end              int
		title, action           string
		intro, body, conclusion string
	}
	var tops []top
	matches := topLevelSectionMarker.FindAllStringSubmatchIndex(summary, -1)
	for i, loc := range matches {
		m := topLevelSectionMarker.FindStringSubmatch(summary[loc[0]:loc[1]])
		if len(m) < 5 {
			continue
		}
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		title := strings.TrimSpace(m[3])
		action := m[4]
		bodyStart := loc[1]
		bodyEnd := len(summary)
		if i+1 < len(matches) {
			bodyEnd = matches[i+1][0]
		}
		chunk := summary[bodyStart:bodyEnd]
		intro, body, conclusion := parseFramedSummary(chunk)
		if body == "" {
			body = strings.TrimSpace(chunk)
		}
		tops = append(tops, top{start: start, end: end, title: title, action: action, intro: intro, body: body, conclusion: conclusion})
	}
	if len(tops) == 0 {
		return
	}
	for i := range idx.Sections {
		s := &idx.Sections[i]
		if s.Level != "task" {
			continue
		}
		for _, t := range tops {
			if s.End < t.start || s.Start > t.end {
				continue
			}
			s.Hidden = true
		}
	}
	for _, t := range tops {
		id := fmt.Sprintf("compact-%d-%d", t.start, t.end)
		found := false
		for i := range idx.Sections {
			s := &idx.Sections[i]
			if s.ID == id || (s.Level == "task" && s.Start == t.start && s.End == t.end && strings.HasPrefix(s.ID, "compact-")) {
				s.Hidden = false
				s.Title = t.title
				s.Memory = t.action
				s.Intro = t.intro
				s.Summary = t.body
				s.Conclusion = t.conclusion
				s.Status = "compacted"
				found = true
				break
			}
		}
		if found {
			continue
		}
		idx.Sections = append(idx.Sections, HistorySection{
			ID:         id,
			Level:      "task",
			Title:      t.title,
			Start:      t.start,
			End:        t.end,
			Status:     "compacted",
			Memory:     t.action,
			Intro:      t.intro,
			Summary:    t.body,
			Conclusion: t.conclusion,
		})
	}
}

func applyMemoryFromCompaction(idx *HistoryIndex, summary string) {
	if idx == nil || strings.TrimSpace(summary) == "" {
		return
	}
	type span struct {
		start, end int
		action     string
	}
	var spans []span
	for _, m := range sectionMemoryMarker.FindAllStringSubmatch(summary, -1) {
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		if start < 1 || end < start {
			continue
		}
		spans = append(spans, span{start: start, end: end, action: m[3]})
	}
	for i := range idx.Sections {
		s := &idx.Sections[i]
		if s.Level != "task" {
			continue
		}
		best := memoryRaw
		for _, sp := range spans {
			// Overlap between task [Start,End] and marker [start,end].
			if sp.end < s.Start || sp.start > s.End {
				continue
			}
			if memoryRank(sp.action) > memoryRank(best) {
				best = sp.action
			}
		}
		if best != memoryRaw {
			s.Memory = best
		} else if s.Memory == "" {
			s.Memory = memoryRaw
		}
	}
}

func isProvisionalChapterTitle(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return true
	}
	return strings.HasPrefix(title, "Chapter ")
}

func cleanGeneratedTitle(raw string) string {
	s := strings.TrimSpace(raw)
	if i := strings.IndexAny(s, "\n\r"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	s = strings.Trim(s, `"'“”`)
	// Drop a leading "Title:" label some models add.
	if strings.HasPrefix(strings.ToLower(s), "title:") {
		s = strings.TrimSpace(s[6:])
		s = strings.Trim(s, `"'“”`)
	}
	return trimLabel(s)
}

func firstUserTextInRange(events []json.RawMessage, start, end int) string {
	if start < 1 {
		start = 1
	}
	if end > len(events) {
		end = len(events)
	}
	for i := start - 1; i < end && i < len(events); i++ {
		label := eventLabel(events[i])
		if label == "" || label == "[task finished]" {
			continue
		}
		if strings.HasPrefix(label, "[AUTOPILOT") || strings.HasPrefix(label, "Continue the current plan") {
			continue
		}
		if strings.HasPrefix(label, "<compacted_conversation_memory>") {
			continue
		}
		return label
	}
	return ""
}
