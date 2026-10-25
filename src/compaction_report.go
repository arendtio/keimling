package main

import (
	"fmt"
	"regexp"
	"strings"
)

var compactSectionMarker = regexp.MustCompile(`(?m)^\[section (?:depth=([0-9]+) )?[0-9]+-[0-9]+: (.*?) \| (keep|distill|summarize|collapse)\]$`)

type compactionReport struct {
	Kept, Distilled, Summarized, Collapsed                         int
	Nodes, MaxDepth                                                int
	KeptTitles, DistilledTitles, SummarizedTitles, CollapsedTitles []string
}

func inspectCompaction(summary string) compactionReport {
	var r compactionReport
	for _, m := range compactSectionMarker.FindAllStringSubmatch(summary, -1) {
		title := strings.TrimSpace(m[2])
		r.Nodes++
		depth := 1
		if m[1] != "" {
			fmt.Sscanf(m[1], "%d", &depth)
		}
		if depth > r.MaxDepth {
			r.MaxDepth = depth
		}
		if len(title) > 65 {
			title = string([]rune(title)[:min(65, len([]rune(title)))]) + "…"
		}
		switch m[3] {
		case "keep":
			r.Kpt(title)
		case "distill":
			r.Distilled++
			if len(r.DistilledTitles) < 2 {
				r.DistilledTitles = append(r.DistilledTitles, title)
			}
		case "summarize":
			r.Sum(title)
		case "collapse":
			r.Col(title)
		}
	}
	return r
}
func (r *compactionReport) Kpt(title string) {
	r.Kept++
	if len(r.KeptTitles) < 2 {
		r.KeptTitles = append(r.KeptTitles, title)
	}
}
func (r *compactionReport) Sum(title string) {
	r.Summarized++
	if len(r.SummarizedTitles) < 2 {
		r.SummarizedTitles = append(r.SummarizedTitles, title)
	}
}
func (r *compactionReport) Col(title string) {
	r.Collapsed++
	if len(r.CollapsedTitles) < 2 {
		r.CollapsedTitles = append(r.CollapsedTitles, title)
	}
}

func formatCompactionReport(summary string, before, after int) string {
	r := inspectCompaction(summary)
	reduction := 0.0
	if before > 0 {
		reduction = 100 * float64(before-after) / float64(before)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "◈ Compaction: ~%d → ~%d tokens (%.1f%% reduction)\n", before, after, reduction)
	fmt.Fprintf(&b, "  Tree: %d nodes · %d levels\n", r.Nodes, r.MaxDepth)
	fmt.Fprintf(&b, "  Sections: %d kept · %d distilled · %d summarized · %d collapsed\n", r.Kept, r.Distilled, r.Summarized, r.Collapsed)
	for _, item := range []struct {
		label  string
		titles []string
		total  int
	}{{"Kept", r.KeptTitles, r.Kept}, {"Distilled", r.DistilledTitles, r.Distilled}, {"Summarized", r.SummarizedTitles, r.Summarized}, {"Collapsed", r.CollapsedTitles, r.Collapsed}} {
		if item.total == 0 {
			continue
		}
		fmt.Fprintf(&b, "  %s: %s", item.label, strings.Join(item.titles, "; "))
		if item.total > len(item.titles) {
			fmt.Fprintf(&b, " (+%d more)", item.total-len(item.titles))
		}
		b.WriteByte('\n')
	}
	return b.String()
}
