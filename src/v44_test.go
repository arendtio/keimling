package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestV44HierarchicalOutline(t *testing.T) {
	input := `{"sections":[{"start":1,"end":4,"title":"Parent","action":"keep","children":[{"start":1,"end":2,"title":"old","action":"collapse","note":"Obsolete experiments"},{"start":3,"end":4,"title":"current","action":"keep"}]}]}`
	sections, err := parseSemanticOutline(input, 4)
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{}
	window := []json.RawMessage{json.RawMessage(`{"role":"user","content":"old1"}`), json.RawMessage(`{"role":"user","content":"old2"}`), json.RawMessage(`{"role":"user","content":"current1"}`), json.RawMessage(`{"role":"user","content":"current2"}`)}
	var b strings.Builder
	if err := a.renderSemanticNodes(context.Background(), &b, window, sections, 0, 1, ""); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if strings.Contains(got, "old1") || !strings.Contains(got, "current2") || !strings.Contains(got, "Obsolete experiments") {
		t.Fatalf("incorrect visibility: %s", got)
	}
	report := inspectCompaction(got)
	if report.Nodes != 3 || report.MaxDepth != 2 {
		t.Fatalf("unexpected report %+v", report)
	}
}
func TestV44RejectBadTree(t *testing.T) {
	for _, input := range []string{
		`{"sections":[{"start":1,"end":3,"title":"p","action":"keep","children":[{"start":1,"end":1,"title":"a","action":"keep"}]}]}`,
		`{"sections":[{"start":1,"end":3,"title":"p","action":"keep","children":[{"start":1,"end":2,"title":"a","action":"keep"},{"start":2,"end":3,"title":"b","action":"collapse"}]}]}`,
	} {
		if _, err := parseSemanticOutline(input, 3); err == nil {
			t.Fatal("accepted invalid tree:", input)
		}
	}
}
func TestV44DepthLimit(t *testing.T) {
	n := semanticSection{Start: 1, End: 1, Title: "leaf", Action: "keep"}
	for i := 0; i < 9; i++ {
		n = semanticSection{Start: 1, End: 1, Title: "parent", Action: "keep", Children: []semanticSection{n}}
	}
	if err := validateSemanticNodes([]semanticSection{n}, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
	n = semanticSection{Start: 1, End: 1, Title: "too deep", Action: "keep", Children: []semanticSection{n}}
	if err := validateSemanticNodes([]semanticSection{n}, 1, 1, 1); err == nil {
		t.Fatal("accepted 11 levels")
	}
}
