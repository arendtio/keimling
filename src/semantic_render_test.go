package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSemanticMaxDepthIsThree(t *testing.T) {
	n := semanticSection{Start: 1, End: 1, Title: "leaf", Action: "keep"}
	for i := 0; i < 2; i++ {
		n = semanticSection{Start: 1, End: 1, Title: "parent", Action: "keep", Children: []semanticSection{n}}
	}
	if err := validateSemanticNodes([]semanticSection{n}, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
	n = semanticSection{Start: 1, End: 1, Title: "too deep", Action: "keep", Children: []semanticSection{n}}
	if err := validateSemanticNodes([]semanticSection{n}, 1, 1, 1); err == nil {
		t.Fatal("accepted more than 3 levels")
	}
}

func TestArcContextListsNeighbors(t *testing.T) {
	sections := []semanticSection{
		{Start: 1, End: 1, Title: "Alpha", Action: "collapse", Note: "old"},
		{Start: 2, End: 2, Title: "Beta", Action: "summarize"},
		{Start: 3, End: 3, Title: "Gamma", Action: "keep"},
	}
	got := buildArcContext(sections, 1, "ship the feature", "")
	for _, want := range []string{"ship the feature", "Alpha", "Gamma", "Beta"} {
		if !strings.Contains(got, want) {
			t.Fatalf("arc missing %q in %q", want, got)
		}
	}
}

func TestRenderFramingForSummarize(t *testing.T) {
	a := &Agent{Cfg: Config{}, Client: &Client{}}
	// Stub: no real API — use keep path for structure; framing unit-tested via formatFramedLeaf.
	body := formatFramedLeaf("Topic", "intro text", "body text", "conclusion text", true)
	for _, want := range []string{"[intro]", "intro text", "[conclusion]", "conclusion text", "body text"} {
		if !strings.Contains(body, want) {
			t.Fatalf("framed leaf missing %q in %q", want, body)
		}
	}
	_ = a
	_ = context.Background()
	_ = json.RawMessage(nil)
}

func TestCollectLeavesForParallel(t *testing.T) {
	sections := []semanticSection{
		{Start: 1, End: 3, Title: "P", Action: "keep", Children: []semanticSection{
			{Start: 1, End: 1, Title: "a", Action: "summarize"},
			{Start: 2, End: 3, Title: "b", Action: "distill"},
		}},
	}
	leaves := collectRenderLeaves(sections, 0, 1, "")
	if len(leaves) != 2 {
		t.Fatalf("got %d leaves", len(leaves))
	}
}

func TestDenserStillHighOccupancy(t *testing.T) {
	if !shouldRunDenserPass(1000, 750, 1000) {
		t.Fatal("expected denser when still at 75%")
	}
	if shouldRunDenserPass(1000, 400, 1000) {
		t.Fatal("should not denser when well under 70%")
	}
}

func TestApplyVisibleChaptersFromCompaction(t *testing.T) {
	idx := HistoryIndex{
		Version: historyIndexVersion,
		Sections: []HistorySection{
			{ID: "task-1", Level: "task", Title: "Chapter 1", Start: 1, End: 10, Status: "done"},
			{ID: "task-11", Level: "task", Title: "Chapter 2", Start: 11, End: 20, Status: "active"},
		},
	}
	summary := `<semantic_history>
[section depth=1 1-20: Merged arc | summarize]
[intro] Why we merged
body
[conclusion] Outcome
</semantic_history>`
	applyVisibleChaptersFromCompaction(&idx, summary)
	visible := 0
	for _, s := range idx.Sections {
		if s.Level == "task" && !s.Hidden {
			visible++
			if s.Title != "Merged arc" {
				t.Fatalf("title %q", s.Title)
			}
			if s.Intro != "Why we merged" || s.Summary == "" {
				t.Fatalf("framing %+v", s)
			}
		}
	}
	if visible != 1 {
		t.Fatalf("visible=%d sections=%+v", visible, idx.Sections)
	}
	hidden := 0
	for _, s := range idx.Sections {
		if s.Hidden {
			hidden++
		}
	}
	if hidden < 2 {
		t.Fatalf("expected old chapters hidden, got hidden=%d", hidden)
	}
}
