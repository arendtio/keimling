package main

import (
	"os"
	"strings"
	"testing"
)

func TestPatchUnprefixedContextLines(t *testing.T) {
	patchTestFile(t, "alpha\nbeta\ngamma\n")
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ ~2\nbeta\n-gamma\n+delta\n"
	notes, err := applyPatchText(patch)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes, "leading space") {
		t.Fatalf("expected note about missing space prefix, got %q", notes)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "alpha\nbeta\ndelta\n" {
		t.Fatal(string(b))
	}
}

func TestPatchContextKeepsFileWhitespace(t *testing.T) {
	patchTestFile(t, "func f() {\n\treturn\n}\n")
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ ~1\n    func f() {\n        return\n    }\n+// done\n"
	if _, err := applyPatchText(patch); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "func f() {\n\treturn\n}\n// done\n" {
		t.Fatalf("context lines were rewritten: %q", string(b))
	}
}

func TestPatchNoNewlineMarker(t *testing.T) {
	patchTestFile(t, "alpha\nbeta")
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ ~2\n beta\n\\ No newline at end of file\n+gamma\n"
	if _, err := applyPatchText(patch); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "alpha\nbetagamma\n" {
		t.Fatalf("unexpected result: %q", string(b))
	}
}

func TestPatchSecondFileSectionInsideHunk(t *testing.T) {
	dir := patchTestFile(t, "alpha\n")
	if err := os.WriteFile(dir+"/b.txt", []byte("beta\n"), 0600); err != nil {
		t.Fatal(err)
	}
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ ~1\n-alpha\n+one\n--- a/b.txt\n+++ b/b.txt\n@@ ~1\n-beta\n+two\n"
	if _, err := applyPatchText(patch); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"a.txt": "one\n", "b.txt": "two\n"} {
		b, _ := os.ReadFile(name)
		if string(b) != want {
			t.Fatalf("%s: %q", name, string(b))
		}
	}
}
