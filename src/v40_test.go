package main

import (
	"os"
	"strings"
	"testing"
)

func TestApproximatePatchAndLegacyCounts(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	if err := os.WriteFile("a.txt", []byte("start\nfoo\nbar\nfoo\nbar\nend\n"), 0600); err != nil {
		t.Fatal(err)
	}
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ ~5\n foo\n-bar\n+updated\n"
	if err := applyStrictPatch(patch); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "start\nfoo\nbar\nfoo\nupdated\nend\n" {
		t.Fatal(string(b))
	}
	if err := applyStrictPatch("--- a/a.txt\n+++ b/a.txt\n@@ -2,99 +2,99 @@\n foo\n-bar\n+new\n"); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile("a.txt")
	if !strings.Contains(string(b), "foo\nnew\n") {
		t.Fatal(string(b))
	}
}
func TestApproximatePatchTieAndAtomicity(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(cwd)
	os.WriteFile("a.txt", []byte("x\na\nx\na\n"), 0600)
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ ~1\n-x\n+y\n"
	// Closest occurrence is line 1, not ambiguous.
	if err := applyStrictPatch(patch); err != nil {
		t.Fatal(err)
	}
	os.WriteFile("a.txt", []byte("x\na\nx\na\n"), 0600)
	// At line 2, positions 1 and 3 are equally close.
	patch = "--- a/a.txt\n+++ b/a.txt\n@@ ~2\n-x\n+y\n"
	// The patch must fail on equal-distance candidates. The old source is unchanged.
	if err := applyStrictPatch(patch); err == nil {
		t.Fatal("expected tie rejection")
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "x\na\nx\na\n" {
		t.Fatal(string(b))
	}
}
func TestApproximatePatchNoPartialWrite(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(cwd)
	os.WriteFile("a.txt", []byte("alpha\n"), 0600)
	os.WriteFile("b.txt", []byte("beta\n"), 0600)
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ ~1\n-alpha\n+changed\n--- a/b.txt\n+++ b/b.txt\n@@ ~1\n-missing\n+bad\n"
	if err := applyStrictPatch(patch); err == nil {
		t.Fatal("expected failure")
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "alpha\n" {
		t.Fatal("partial write")
	}
}
