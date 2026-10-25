package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestV45DistillInOutline(t *testing.T) {
	nodes, err := parseSemanticOutline(`{"sections":[{"start":1,"end":2,"title":"Current relevant work","action":"distill"}]}`, 2)
	if err != nil || nodes[0].Action != "distill" {
		t.Fatalf("distill rejected: %v", err)
	}
	a := &Agent{Cfg: Config{Model: "test"}}
	// Empty requests would use network: inspect no implicit fallthrough instead.
	if err := validateSemanticNodes(nodes, 1, 2, 1); err != nil {
		t.Fatal(err)
	}
	_ = a
	_ = context.Background()
	_ = json.RawMessage(nil)
}
func TestV45DynamicOutputCeiling(t *testing.T) {
	small := compactionOutputCeiling("abc", 0)
	large := compactionOutputCeiling(strings.Repeat("x", 600000), 0)
	if small < 8192 || large <= small || large > 65536 {
		t.Fatalf("unexpected limits %d/%d", small, large)
	}
}
func TestV45SelectCompactionModel(t *testing.T) {
	a := &Agent{Cfg: Config{Model: "primary", CompactionModel: "secondary"}}
	if a.compactionModel() != "secondary" {
		t.Fatal("override ignored")
	}
	a.Cfg.CompactionModel = ""
	if a.compactionModel() != "primary" {
		t.Fatal("fallback invalid")
	}
}
func TestV45DistillInReport(t *testing.T) {
	r := inspectCompaction("[section depth=2 1-5: Relevant work | distill]")
	if r.Distilled != 1 || r.MaxDepth != 2 {
		t.Fatalf("unexpected report: %+v", r)
	}
}
