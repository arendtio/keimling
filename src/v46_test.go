package main

import (
	"strings"
	"testing"
)

func TestCompactionMinimumReduction(t *testing.T) {
	cases := []struct {
		before, after int
		ok            bool
	}{{1000, 700, true}, {1000, 701, false}, {1000, 300, true}, {0, 0, false}}
	for _, c := range cases {
		if got := compactionSufficient(c.before, c.after); got != c.ok {
			t.Errorf("%d -> %d got %v", c.before, c.after, got)
		}
	}
}
func TestCompactionInstructionSameForBackground(t *testing.T) {
	a := compactionInstruction("")
	b := compactionInstruction(" ")
	if a != b || !strings.Contains(a, "semantic sections") {
		t.Fatal("unexpected base instruction")
	}
	if !strings.Contains(compactionInstruction("focus tests"), a) {
		t.Fatal("extra instruction should preserve common criteria")
	}
}
