package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestPatchEmptyContextLineWithCounts(t *testing.T) {
	patchTestFile(t, "alpha\n\nbeta\n")
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ -1,3 +1,3 @@\n alpha\n \n-beta\n+gamma\n"
	notes, err := applyPatchText(patch)
	if err != nil {
		t.Fatal(err)
	}
	if notes != "" {
		t.Fatalf("unexpected notes: %s", notes)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "alpha\n\ngamma\n" {
		t.Fatalf("%q", string(b))
	}
}

func TestPatchRemovedLineStartingWithDashes(t *testing.T) {
	patchTestFile(t, "alpha\n-- comment\nbeta\n")
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ -1,3 +1,3 @@\n alpha\n--- comment\n+// comment\n beta\n"
	if _, err := applyPatchText(patch); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "alpha\n// comment\nbeta\n" {
		t.Fatalf("%q", string(b))
	}
}

func TestPatchSkipsGitPreamble(t *testing.T) {
	patchTestFile(t, "alpha\n")
	patch := "diff --git a/a.txt b/a.txt\nindex 1234567..89abcde 100644\n--- a/a.txt\n+++ b/a.txt\n@@ -1,1 +1,1 @@\n-alpha\n+beta\n"
	if _, err := applyPatchText(patch); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "beta\n" {
		t.Fatalf("%q", string(b))
	}
}

func TestPatchCreateRejectsExistingTarget(t *testing.T) {
	patchTestFile(t, "alpha\n")
	_, err := applyPatchText("--- /dev/null\n+++ b/a.txt\n@@ -0,0 +1,1 @@\n+new\n")
	if err == nil {
		t.Fatal("expected failure")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "alpha\n" {
		t.Fatalf("file changed: %q", string(b))
	}
}

func TestPatchCreateHasNoInsertionNote(t *testing.T) {
	patchTestFile(t, "alpha\n")
	notes, err := applyPatchText("--- /dev/null\n+++ b/new.txt\n@@ -0,0 +1,1 @@\n+created\n")
	if err != nil {
		t.Fatal(err)
	}
	if notes != "" {
		t.Fatalf("unexpected notes: %s", notes)
	}
	b, _ := os.ReadFile("new.txt")
	if string(b) != "created\n" {
		t.Fatalf("%q", string(b))
	}
}

func TestPatchCreateRejectsContextLines(t *testing.T) {
	patchTestFile(t, "alpha\n")
	_, err := applyPatchText("--- /dev/null\n+++ b/new.txt\n@@ -0,0 +1,2 @@\n+created\n context\n")
	if err == nil || !strings.Contains(err.Error(), "creation") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPatchKeepsCRLFInExistingFile(t *testing.T) {
	patchTestFile(t, "alpha\r\nbeta\r\ngamma\r\n")
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ -1,3 +1,3 @@\n alpha\n-beta\n+BETA\n gamma\n"
	notes, err := applyPatchText(patch)
	if err != nil {
		t.Fatal(err)
	}
	if notes != "" {
		t.Fatalf("unexpected notes: %s", notes)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "alpha\r\nBETA\r\ngamma\r\n" {
		t.Fatalf("%q", string(b))
	}
}

func TestPatchReportsNearestOfSeveralMatches(t *testing.T) {
	patchTestFile(t, "a\nx\nb\nx\nc\n")
	notes, err := applyPatchText("--- a/a.txt\n+++ b/a.txt\n@@ ~4\n x\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes, "patched the nearest one") {
		t.Fatalf("unexpected notes: %s", notes)
	}
}

func TestPatchDeletesFile(t *testing.T) {
	dir := patchTestFile(t, "alpha\nbeta\n")
	if err := os.WriteFile(dir+"/doomed.txt", []byte("gone\n"), 0600); err != nil {
		t.Fatal(err)
	}
	patch := "--- a/doomed.txt\n+++ /dev/null\n@@ -1,1 +0,0 @@\n-gone\n"
	if _, err := applyPatchText(patch); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("doomed.txt"); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
}

func TestPatchDeleteMissingFile(t *testing.T) {
	patchTestFile(t, "alpha\n")
	_, err := applyPatchText("--- a/missing.txt\n+++ /dev/null\n@@ -1,1 +0,0 @@\n-gone\n")
	if err == nil || !strings.Contains(err.Error(), "missing.txt") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPatchDeleteRollsBackOnFailure(t *testing.T) {
	patchTestFile(t, "alpha\n")
	if err := os.WriteFile("b.txt", []byte("beta\n"), 0600); err != nil {
		t.Fatal(err)
	}
	patch := "--- a/a.txt\n+++ /dev/null\n@@ -1,1 +0,0 @@\n-alpha\n--- a/b.txt\n+++ b/b.txt\n@@ ~1\n-missing\n+bad\n"
	if _, err := applyPatchText(patch); err == nil {
		t.Fatal("expected failure")
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "alpha\n" {
		t.Fatalf("deletion not rolled back: %q", string(b))
	}
}

func TestPatchTargetsAndPreview(t *testing.T) {
	dir := patchTestFile(t, "alpha\n")
	if err := os.WriteFile(dir+"/b.txt", []byte("beta\n"), 0600); err != nil {
		t.Fatal(err)
	}
	patch := "--- a/a.txt\n+++ b/a.txt\n@@ ~1\n-alpha\n+one\n--- /dev/null\n+++ b/c.txt\n@@ -0,0 +1,1 @@\n+new\n--- a/b.txt\n+++ /dev/null\n@@ -1,1 +0,0 @@\n-beta\n"
	targets, err := patchTargets(patch)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 3 || targets[0].path != "a.txt" || targets[1].path != "c.txt" || !targets[1].create || !targets[2].delete {
		t.Fatalf("unexpected targets: %+v", targets)
	}
	args := `{"patch":"` + strings.ReplaceAll(patch, "\n", `\n`) + `"}`
	preview := renderToolCall("apply_patch", args)
	for _, want := range []string{"a.txt", "c.txt", "b.txt", "new", "delete"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview %q misses %q", preview, want)
		}
	}
}

func TestPolicyPatchTargetsAreChecked(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.WriteFile("a.txt", []byte("x\n"), 0600); err != nil {
		t.Fatal(err)
	}
	create := `{"patch":"--- /dev/null\n+++ b/new.txt\n@@ -0,0 +1,1 @@\n+x\n"}`
	if forced, why := forceApproval("apply_patch", create); !forced || !strings.Contains(why, "new.txt") {
		t.Fatalf("unexpected verdict: %v %q", forced, why)
	}
	outside := `{"patch":"--- /dev/null\n+++ b/../x.txt\n@@ -0,0 +1,1 @@\n+x\n"}`
	if forced, why := forceApproval("apply_patch", outside); !forced || why != "invalid patch arguments" {
		t.Fatalf("unexpected verdict: %v %q", forced, why)
	}
	modify := `{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ ~1\n x\n"}`
	if forced, why := forceApproval("apply_patch", modify); !forced || !strings.Contains(why, "a.txt") {
		t.Fatalf("unexpected verdict: %v %q", forced, why)
	}
}

func TestApprovalPreviewShowsPatchContent(t *testing.T) {
	args := `{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ ~2\n context\n-old\n+new\n"}`
	got := renderApprovalPreview("apply_patch", args)
	for _, want := range []string{"1 addition(s), 1 removal(s)", "-old", "+new", "@@ ~2"} {
		if !strings.Contains(got, want) {
			t.Fatalf("preview %q misses %q", got, want)
		}
	}
}

func TestApprovalPreviewTruncatesLongPatch(t *testing.T) {
	var b strings.Builder
	b.WriteString("--- a/a.txt\n+++ b/a.txt\n@@ ~1\n")
	for i := 0; i < 200; i++ {
		b.WriteString("+line\n")
	}
	args := `{"patch":"` + strings.ReplaceAll(b.String(), "\n", `\n`) + `"}`
	got := renderApprovalPreview("apply_patch", args)
	if !strings.Contains(got, "more line(s)") {
		t.Fatalf("expected truncation marker: %q", got)
	}
	if len(got) > 12000 {
		t.Fatalf("preview too large: %d", len(got))
	}
}

func TestApprovalPreviewShowsReplacement(t *testing.T) {
	args := `{"path":"a.txt","start_line":4,"end_line":6,"content":"x\ny\n"}`
	got := renderApprovalPreview("replace_lines", args)
	for _, want := range []string{"a.txt", "lines 4-6", "+x", "+y"} {
		if !strings.Contains(got, want) {
			t.Fatalf("preview %q misses %q", got, want)
		}
	}
	if got := renderApprovalPreview("exec", `{"command":"ls"}`); got != "" {
		t.Fatalf("exec should not add a preview: %q", got)
	}
}

func TestPatchDiagnosticShowsClosestMatch(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= 200; i++ {
		fmt.Fprintf(&b, "line%d\n", i)
	}
	patchTestFile(t, b.String())
	_, err := applyPatchText("--- a/a.txt\n+++ b/a.txt\n@@ ~5\n-line150\n+X\n")
	if err == nil {
		t.Fatal("expected failure")
	}
	msg := err.Error()
	for _, want := range []string{"closest match starts at line 150", "line150", "@@ ~LINE"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in %q", want, msg)
		}
	}
}
