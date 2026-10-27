package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The session journal remains authoritative. This sidecar contains only a
// rebuildable navigation view. Events are numbered by their item records.
type HistorySection struct {
	ID         string `json:"id"`
	Parent     string `json:"parent,omitempty"`
	Level      string `json:"level"`
	Title      string `json:"title"`
	Start      int    `json:"start"`
	End        int    `json:"end"`
	Status     string `json:"status"`
	Summary    string `json:"summary,omitempty"`
	Intro      string `json:"intro,omitempty"`
	Conclusion string `json:"conclusion,omitempty"`
	// Memory: raw | keep | distill | summarize | collapse — how compaction treated this span.
	Memory string `json:"memory,omitempty"`
	// Hidden chapters are superseded (e.g. merged denser) and must not appear in the GUI.
	Hidden bool `json:"hidden,omitempty"`
}
type HistoryIndex struct {
	Version      int              `json:"version"`
	Sections     []HistorySection `json:"sections"`
	WorkingState string           `json:"working_state,omitempty"`
}

const historyIndexVersion = 2

func originalEvents(path string) ([]json.RawMessage, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 4096), 64<<20)
	events := []json.RawMessage{}
	for scan.Scan() {
		var record struct {
			Type string          `json:"type"`
			Item json.RawMessage `json:"item"`
		}
		if json.Unmarshal(scan.Bytes(), &record) == nil && record.Type == "item" {
			events = append(events, record.Item)
		}
	}
	return events, scan.Err()
}
func eventLabel(raw json.RawMessage) string {
	var v struct {
		Role      string          `json:"role"`
		Content   json.RawMessage `json:"content"`
		Type      string          `json:"type"`
		Name      string          `json:"name"`
		Arguments string          `json:"arguments"`
	}
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	if v.Role == "user" {
		var s string
		if json.Unmarshal(v.Content, &s) == nil {
			return strings.TrimSpace(s)
		}
	}
	if v.Type == "function_call" && v.Name == "finish_task" {
		return "[task finished]"
	}
	return ""
}
func trimLabel(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 110 {
		s = s[:110] + "…"
	}
	return s
}
func buildHistoryIndex(events []json.RawMessage, previous HistoryIndex) HistoryIndex {
	idx := HistoryIndex{Version: historyIndexVersion, WorkingState: previous.WorkingState}
	if len(events) == 0 {
		return idx
	}
	taskStart := 0
	count := 0
	for i, e := range events {
		label := eventLabel(e)
		// New human turns start top-level tasks. Internal supervisor messages do not.
		if label != "" && label != "[task finished]" && !strings.HasPrefix(label, "[AUTOPILOT") && !strings.HasPrefix(label, "Continue the current plan autonomously.") && !strings.HasPrefix(label, "<compacted_conversation_memory>") && i > taskStart {
			addTaskSections(&idx, events, taskStart, i, count)
			count++
			taskStart = i
		}
	}
	addTaskSections(&idx, events, taskStart, len(events), count)
	// Preserve title / summary / memory for segments with unchanged event boundaries.
	old := map[string]HistorySection{}
	for _, s := range previous.Sections {
		old[fmt.Sprintf("%s:%d:%d", s.Level, s.Start, s.End)] = s
	}
	for i := range idx.Sections {
		s := &idx.Sections[i]
		if prev, ok := old[fmt.Sprintf("%s:%d:%d", s.Level, s.Start, s.End)]; ok {
			if prev.Title != "" {
				s.Title = prev.Title
			}
			s.Summary = prev.Summary
			s.Intro = prev.Intro
			s.Conclusion = prev.Conclusion
			s.Hidden = prev.Hidden
			if prev.Memory != "" {
				s.Memory = prev.Memory
			}
		}
		if s.Level == "task" && s.Memory == "" {
			s.Memory = memoryRaw
		}
	}
	return idx
}
func addTaskSections(idx *HistoryIndex, events []json.RawMessage, start, end, id int) {
	if end <= start {
		return
	}
	taskID := fmt.Sprintf("task-%d", start+1)
	// Title is generated separately — never use the first user message as the heading.
	title := fmt.Sprintf("Chapter %d", id+1)
	status := "active"
	for _, raw := range events[start:end] {
		var call struct {
			Type      string `json:"type"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}
		if json.Unmarshal(raw, &call) == nil && call.Type == "function_call" && call.Name == "finish_task" {
			var args struct {
				Status string `json:"status"`
			}
			if json.Unmarshal([]byte(call.Arguments), &args) == nil {
				if args.Status == "blocked" {
					status = "blocked"
				} else if args.Status == "done" {
					status = "completed"
				}
			}
		}
	}
	idx.Sections = append(idx.Sections, HistorySection{
		ID: taskID, Level: "task", Title: title, Start: start + 1, End: end,
		Status: status, Memory: memoryRaw,
	})
	// Episodes and segments form a bounded hierarchical index, not a complete AST.
	episodeSize := 100
	segmentSize := 25
	for es := start; es < end; es += episodeSize {
		ee := es + episodeSize
		if ee > end {
			ee = end
		}
		epID := fmt.Sprintf("episode-%d", es+1)
		idx.Sections = append(idx.Sections, HistorySection{ID: epID, Parent: taskID, Level: "episode", Title: fmt.Sprintf("Events %d–%d", es+1, ee), Start: es + 1, End: ee, Status: status})
		for ss := es; ss < ee; ss += segmentSize {
			se := ss + segmentSize
			if se > ee {
				se = ee
			}
			idx.Sections = append(idx.Sections, HistorySection{ID: fmt.Sprintf("segment-%d", ss+1), Parent: epID, Level: "segment", Title: fmt.Sprintf("Events %d–%d", ss+1, se), Start: ss + 1, End: se, Status: status})
		}
	}
}
func indexPath(session string) string { return session + ".index.json" }
func loadHistoryIndex(session string) HistoryIndex {
	var idx HistoryIndex
	b, e := os.ReadFile(indexPath(session))
	if e == nil && json.Unmarshal(b, &idx) == nil && idx.Version == historyIndexVersion {
		return idx
	}
	return HistoryIndex{Version: historyIndexVersion}
}
func saveHistoryIndex(session string, idx HistoryIndex) error {
	b, e := json.MarshalIndent(idx, "", "  ")
	if e != nil {
		return e
	}
	p := indexPath(session)
	if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(p), ".index-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e != nil {
		f.Close()
		return e
	}
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), p)
}
func (a *Agent) updateHistoryIndex(summary string) error {
	events, e := originalEvents(a.Cfg.SessionPath)
	if e != nil {
		return e
	}
	prev := loadHistoryIndex(a.Cfg.SessionPath)
	idx := buildHistoryIndex(events, prev)
	if summary != "" {
		idx.WorkingState = summary
		applyMemoryFromCompaction(&idx, summary)
		applyVisibleChaptersFromCompaction(&idx, summary)
	}
	a.ensureChapterTitles(context.Background(), &idx, events)
	return saveHistoryIndex(a.Cfg.SessionPath, idx)
}

// ensureChapterTitles replaces provisional "Chapter N" labels with a short generated title.
// The first user message stays a message in the transcript — never the heading itself.
func (a *Agent) ensureChapterTitles(ctx context.Context, idx *HistoryIndex, events []json.RawMessage) {
	if a == nil || a.Client == nil || idx == nil {
		return
	}
	for i := range idx.Sections {
		s := &idx.Sections[i]
		if s.Level != "task" || s.Hidden || !isProvisionalChapterTitle(s.Title) {
			continue
		}
		seed := firstUserTextInRange(events, s.Start, s.End)
		if seed == "" {
			continue
		}
		askCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		title, err := a.Client.compactionAsk(askCtx, a.compactionModel(),
			`Write a short chapter title for this agent task (3–8 words). Do NOT copy or lightly paraphrase the user message. Prefer a topic label (e.g. "PTY deadlock fix"). Title only — no quotes, no punctuation fluff.`,
			"User request:\n"+truncate(seed, 800), 64)
		cancel()
		if err != nil {
			continue
		}
		if cleaned := cleanGeneratedTitle(title); cleaned != "" && !isProvisionalChapterTitle(cleaned) {
			s.Title = cleaned
		}
	}
}
func (a *Agent) searchHistory(args string) string {
	var q struct {
		Query string `json:"query"`
	}
	if e := decodeArgs(args, &q); e != nil {
		return e.Error()
	}
	events, e := originalEvents(a.Cfg.SessionPath)
	if e != nil {
		return e.Error()
	}
	idx := buildHistoryIndex(events, loadHistoryIndex(a.Cfg.SessionPath))
	if len(idx.Sections) == 0 {
		return "No history sections."
	}
	terms := strings.Fields(strings.ToLower(q.Query))
	type hit struct {
		score int
		s     HistorySection
	}
	hits := []hit{}
	for _, s := range idx.Sections {
		if s.Level != "task" && s.Level != "episode" {
			continue
		}
		hay := strings.ToLower(s.Title + " " + s.Summary)
		score := 0
		for _, t := range terms {
			if strings.Contains(hay, t) {
				score += 3
			}
		}
		// Search original events as a fallback when no summary contains the keywords.
		if score == 0 {
			for _, raw := range events[s.Start-1 : s.End] {
				v := strings.ToLower(string(raw))
				for _, t := range terms {
					if len(t) > 2 && strings.Contains(v, t) {
						score++
						break
					}
				}
				if score >= 3 {
					break
				}
			}
		}
		if score > 0 {
			hits = append(hits, hit{score, s})
		}
	}
	for i := 0; i < len(hits); i++ {
		for j := i + 1; j < len(hits); j++ {
			if hits[j].score > hits[i].score {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}
	if len(hits) == 0 {
		return "No matching sections; try different keywords."
	}
	var out strings.Builder
	for i, h := range hits {
		if i >= 10 {
			break
		}
		fmt.Fprintf(&out, "%s (%s; events %d-%d; %s): %s\n", h.s.ID, h.s.Level, h.s.Start, h.s.End, h.s.Status, h.s.Title)
		if h.s.Summary != "" {
			fmt.Fprintf(&out, "  %s\n", trimLabel(h.s.Summary))
		}
	}
	return out.String()
}
func (a *Agent) readHistory(args string) string {
	var q struct {
		SectionID string `json:"section_id"`
		Start     int    `json:"start_event"`
		Count     int    `json:"count"`
	}
	if e := decodeArgs(args, &q); e != nil {
		return e.Error()
	}
	events, e := originalEvents(a.Cfg.SessionPath)
	if e != nil {
		return e.Error()
	}
	idx := buildHistoryIndex(events, loadHistoryIndex(a.Cfg.SessionPath))
	start, end := q.Start, len(events)
	if q.SectionID != "" {
		found := false
		for _, s := range idx.Sections {
			if s.ID == q.SectionID {
				start = s.Start
				if q.Start >= s.Start && q.Start <= s.End {
					start = q.Start
				}
				end = s.End
				found = true
				break
			}
		}
		if !found {
			return "Unknown section_id; use search_history."
		}
	}
	if q.Count <= 0 || q.Count > 20 {
		q.Count = 20
	}
	if start < 1 || start > len(events) {
		return "Invalid start_event"
	}
	if end > start+q.Count-1 {
		end = start + q.Count - 1
	}
	var out strings.Builder
	for i := start; i <= end; i++ {
		entry := string(events[i-1])
		if len(entry) > 12000 {
			entry = entry[:12000] + " [TRUNCATED]"
		}
		fmt.Fprintf(&out, "[%d] %s\n", i, entry)
		if out.Len() > 45000 {
			out.WriteString("[Read limit reached; request next range]\n")
			break
		}
	}
	return out.String()
}
func (a *Agent) compactIndexed(ctx context.Context, history []json.RawMessage, instruction string) (string, error) {
	return a.compactSemantic(ctx, history, instruction)
}

// Locate the last unfinished user task in the immutable journal. A native
// finish_task call marks a task as completed or blocked; do not replay it.
func pendingTask(session string) (string, bool) {
	events, err := originalEvents(session)
	if err != nil {
		return "", false
	}
	last := ""
	finished := true
	for _, raw := range events {
		var v struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
			Type    string          `json:"type"`
			Name    string          `json:"name"`
		}
		if json.Unmarshal(raw, &v) != nil {
			continue
		}
		if v.Role == "user" {
			var text string
			if json.Unmarshal(v.Content, &text) == nil && !strings.HasPrefix(text, "[AUTOPILOT") && !strings.HasPrefix(text, "Continue the current plan autonomously.") && !strings.HasPrefix(text, "<compacted_conversation_memory>") {
				last = text
				finished = false
			}
		}
		if v.Type == "function_call" && v.Name == "finish_task" {
			finished = true
		}
	}
	return last, last != "" && !finished
}
