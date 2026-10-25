package main

import (
	"os"
	"strings"
	"testing"
)

func patchTestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	if err := os.WriteFile("a.txt", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestPatchBareHeaderUsesFirstMatch(t *testing.T) {
	patchTestFile(t, "alpha\nbeta\ngamma\nbeta\ngamma\ndelta\n")
	notes, err := applyPatchText("--- a/a.txt\n+++ b/a.txt\n@@\n beta\n-gamma\n+changed\n")
	if err != nil {
		t.Fatal(err)
	}
	if notes != "" {
		t.Fatalf("unexpected notes: %s", notes)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "alpha\nbeta\nchanged\nbeta\ngamma\ndelta\n" {
		t.Fatal(string(b))
	}
}

func TestPatchPureInsertionUsesHint(t *testing.T) {
	patchTestFile(t, "one\ntwo\nthree\n")
	notes, err := applyPatchText("--- a/a.txt\n+++ b/a.txt\n@@ ~2\n+inserted\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes, "inserted 1 line") {
		t.Fatalf("expected insertion note, got %q", notes)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "one\ninserted\ntwo\nthree\n" {
		t.Fatal(string(b))
	}
}

func TestPatchWhitespaceFallbackIsReported(t *testing.T) {
	patchTestFile(t, "func main() {\n\tif x {\n\t\trun()\n\t}\n}\n")
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ ~2\n    if x {\n-        run()\n+        stop()\n    }\n"
	notes, err := applyPatchText(patch)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes, "indentation") {
		t.Fatalf("expected indentation note, got %q", notes)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "func main() {\n\tif x {\n        stop()\n\t}\n}\n" {
		t.Fatal(string(b))
	}
}

func TestPatchDiagnosticShowsActualLines(t *testing.T) {
	patchTestFile(t, "alpha\nbeta\ngamma\n")
	_, err := applyPatchText("--- a/a.txt\n+++ b/a.txt\n@@ ~2\nwrong context\n-gamma\n+delta\n")
	if err == nil {
		t.Fatal("expected failure")
	}
	msg := err.Error()
	for _, want := range []string{"no context match near line 2", "actual file at", "beta", "read_file"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in %q", want, msg)
		}
	}
}

func TestPatchMissingFileNamesPath(t *testing.T) {
	patchTestFile(t, "alpha\n")
	_, err := applyPatchText("--- a/missing.txt\n+++ b/missing.txt\n@@ ~1\n alpha\n")
	if err == nil || !strings.Contains(err.Error(), "missing.txt") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPatchAcceptsCRLFInput(t *testing.T) {
	patchTestFile(t, "one\ntwo\n")
	_, err := applyPatchText("--- a/a.txt\r\n+++ b/a.txt\r\n@@ ~2\r\n-two\r\n+deux\r\n")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "one\ndeux\n" {
		t.Fatal(string(b))
	}
}

func TestPatchEmptyInputRejected(t *testing.T) {
	patchTestFile(t, "one\n")
	if _, err := applyPatchText("   \n"); err == nil {
		t.Fatal("expected error for empty patch")
	}
}
