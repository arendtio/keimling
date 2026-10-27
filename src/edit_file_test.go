package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	return dir
}

func TestEditFileUniqueReplace(t *testing.T) {
	_ = chdirTemp(t)
	path := "a.txt"
	if err := os.WriteFile(path, []byte("alpha\nbeta\ngamma\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := `{"path":"a.txt","old_string":"beta\n","new_string":"delta\n","replace_all":false}`
	if got := executeTool(context.Background(), "edit_file", args); got != "edited successfully" {
		t.Fatal(got)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "alpha\ndelta\ngamma\n" {
		t.Fatal(string(b))
	}
}

func TestEditFileRequiresUniqueMatch(t *testing.T) {
	_ = chdirTemp(t)
	if err := os.WriteFile("a.txt", []byte("x\nx\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := `{"path":"a.txt","old_string":"x\n","new_string":"y\n","replace_all":false}`
	got := executeTool(context.Background(), "edit_file", args)
	if !strings.Contains(got, "multiple matches") {
		t.Fatal(got)
	}
}

func TestEditFileReplaceAll(t *testing.T) {
	_ = chdirTemp(t)
	if err := os.WriteFile("a.txt", []byte("x\nx\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := `{"path":"a.txt","old_string":"x\n","new_string":"y\n","replace_all":true}`
	if got := executeTool(context.Background(), "edit_file", args); !strings.Contains(got, "edited successfully") {
		t.Fatal(got)
	}
	b, _ := os.ReadFile("a.txt")
	if string(b) != "y\ny\n" {
		t.Fatal(string(b))
	}
}

func TestWriteFileCreateAndOverwrite(t *testing.T) {
	_ = chdirTemp(t)
	path := filepath.Join("sub", "b.txt")
	args := `{"path":` + jsonQuote(path) + `,"content":"hello\n"}`
	if got := executeTool(context.Background(), "write_file", args); got != "wrote successfully" {
		t.Fatal(got)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "hello\n" {
		t.Fatal(string(b))
	}
	args = `{"path":` + jsonQuote(path) + `,"content":"world\n"}`
	if got := executeTool(context.Background(), "write_file", args); got != "wrote successfully" {
		t.Fatal(got)
	}
	b, _ = os.ReadFile(path)
	if string(b) != "world\n" {
		t.Fatal(string(b))
	}
}

func jsonQuote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestPolicyFileEditTargetsAreChecked(t *testing.T) {
	_ = chdirTemp(t)
	if err := os.WriteFile("a.txt", []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if forced, why := forceApproval("write_file", `{"path":"new.txt","content":"x\n"}`); forced {
		t.Fatalf("new write should not force: %v %q", forced, why)
	}
	if forced, why := forceApproval("write_file", `{"path":"../x.txt","content":"x\n"}`); !forced || why != "write outside working directory" {
		t.Fatalf("unexpected verdict: %v %q", forced, why)
	}
	if forced, why := forceApproval("edit_file", `{"path":"a.txt","old_string":"x\n","new_string":"y\n","replace_all":false}`); !forced || !strings.Contains(why, "modifying") {
		t.Fatalf("unexpected verdict: %v %q", forced, why)
	}
	if forced, why := forceApproval("write_file", `{"path":"a.txt","content":"z\n"}`); !forced || !strings.Contains(why, "overwriting") {
		t.Fatalf("unexpected verdict: %v %q", forced, why)
	}
}

func TestApprovalPreviewShowsEditContent(t *testing.T) {
	args := `{"path":"a.txt","old_string":"old\n","new_string":"new\n","replace_all":false}`
	got := renderApprovalPreview("edit_file", args)
	for _, want := range []string{"a.txt", "-old", "+new", "one match"} {
		if !strings.Contains(got, want) {
			t.Fatalf("preview %q misses %q", got, want)
		}
	}
}

func TestApprovalPreviewTruncatesLongEdit(t *testing.T) {
	var old, neu strings.Builder
	for i := 0; i < 200; i++ {
		old.WriteString("old\n")
		neu.WriteString("new\n")
	}
	args := `{"path":"a.txt","old_string":` + jsonQuote(old.String()) + `,"new_string":` + jsonQuote(neu.String()) + `,"replace_all":false}`
	got := renderApprovalPreview("edit_file", args)
	if !strings.Contains(got, "more line(s)") {
		t.Fatalf("expected truncation marker: %q", got)
	}
	if len(got) > 12000 {
		t.Fatalf("preview too large: %d", len(got))
	}
}

func TestApprovalPreviewShowsWrite(t *testing.T) {
	args := `{"path":"a.txt","content":"x\ny\n"}`
	got := renderApprovalPreview("write_file", args)
	for _, want := range []string{"a.txt", "write", "+x", "+y"} {
		if !strings.Contains(got, want) {
			t.Fatalf("preview %q misses %q", got, want)
		}
	}
	if got := renderApprovalPreview("exec", `{"command":"ls"}`); got != "" {
		t.Fatalf("exec should not add a preview: %q", got)
	}
}
