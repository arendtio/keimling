package main

import "testing"

func TestVersionLineUsesBuildStamp(t *testing.T) {
	old := buildVersion
	t.Cleanup(func() { buildVersion = old })
	buildVersion = "abc1234"
	if got := versionLine(); got != "keimling abc1234" {
		t.Fatalf("got %q", got)
	}
}
