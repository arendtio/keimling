package main

import (
	"context"
	"strings"
	"testing"
)

func TestTenLineOutputBypassesFilter(t *testing.T) {
	a := &Agent{}
	original := "exit_code=0\n" + strings.Repeat("line\n", 10)
	if got := a.filterExecResult(context.Background(), "cpu", `{"command":"lscpu | head -n 10"}`, original); got != original {
		t.Fatal("ten lines must stay unchanged")
	}
}
func TestOneLineOutputBypassesFilter(t *testing.T) {
	a := &Agent{}
	original := "exit_code=0\nTue Oct 20 11:00:00 CEST 2026\n"
	if got := a.filterExecResult(context.Background(), "date", `{"command":"date"}`, original); got != original {
		t.Fatal("one line must stay unchanged")
	}
}
