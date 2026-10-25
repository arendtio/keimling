package main

import (
	"strings"
	"testing"
)

func TestCompactionReport(t *testing.T) {
	content := "<working_state>\nStatus\n</working_state>\n<semantic_history>\n[section 1-4: Design | keep]\nabc\n[section 5-9: Discovery | summarize]\ndef\n[section 10-20: Old failure | collapse]\nghi\n</semantic_history>"
	r := inspectCompaction(content)
	if r.Kept != 1 || r.Summarized != 1 || r.Collapsed != 1 {
		t.Fatalf("unexpected counts: %+v", r)
	}
	got := formatCompactionReport(content, 1000, 240)
	if !strings.Contains(got, "76.0% reduction") || !strings.Contains(got, "1 kept · 0 distilled · 1 summarized · 1 collapsed") {
		t.Fatal(got)
	}
}
