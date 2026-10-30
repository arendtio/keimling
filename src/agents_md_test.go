package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirsFromRootToCWD(t *testing.T) {
	got := dirsFromRootToCWD("/proj", "/proj/src/pkg")
	want := []string{"/proj", "/proj/src", "/proj/src/pkg"}
	if len(got) != len(want) {
		t.Fatalf("got %#v", got)
	}
	for i := range want {
		if filepath.Clean(got[i]) != filepath.Clean(want[i]) {
			t.Fatalf("%d: got %q want %q", i, got[i], want[i])
		}
	}
	if len(dirsFromRootToCWD("/other", "/proj/a")) != 1 {
		t.Fatal("non-ancestor should fall back to cwd only")
	}
}

func TestCollectAgentsMDHierarchyAndOverride(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("root rules"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "AGENTS.override.md"), []byte("pkg override"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "AGENTS.md"), []byte("pkg ignored"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Fake git root via .git file
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("ref: fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	chunks := collectAgentsMDChunks(filepath.Join(root, "pkg"), "", "", agentsMDMaxBytes)
	if len(chunks) != 2 {
		t.Fatalf("chunks=%d %#v", len(chunks), chunks)
	}
	if !strings.Contains(chunks[0].Content, "root rules") {
		t.Fatalf("first %#v", chunks[0])
	}
	if !strings.Contains(chunks[1].Content, "pkg override") {
		t.Fatalf("second %#v", chunks[1])
	}
	if strings.Contains(chunks[1].Content, "pkg ignored") {
		t.Fatal("override should win over AGENTS.md in same dir")
	}

	block := formatAgentsMDBlock(chunks)
	if !strings.Contains(block, "<AGENTS_MD>") || !strings.Contains(block, "root rules") || !strings.Contains(block, "pkg override") {
		t.Fatalf("block=%q", block)
	}
	// Nearer guidance appears later in the prompt.
	if i, j := strings.Index(block, "root rules"), strings.Index(block, "pkg override"); i < 0 || j < 0 || i > j {
		t.Fatalf("order wrong: %d %d", i, j)
	}
}

func TestCollectAgentsMDGlobalAndBudget(t *testing.T) {
	global := t.TempDir()
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(global, "AGENTS.md"), []byte("global guidance"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, ".git"), []byte("ref"), 0o644); err != nil {
		t.Fatal(err)
	}
	big := strings.Repeat("x", 2000)
	if err := os.WriteFile(filepath.Join(proj, "AGENTS.md"), []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}

	chunks := collectAgentsMDChunks(proj, "", global, 1500)
	if len(chunks) < 1 {
		t.Fatal("expected at least global")
	}
	if chunks[0].Content != "global guidance" {
		t.Fatalf("%#v", chunks[0])
	}
	used := 0
	for i, c := range chunks {
		if i > 0 {
			used += 2
		}
		used += len(c.Content)
	}
	if used > 1500 {
		t.Fatalf("content over budget: %d", used)
	}
	if len(chunks) > 1 && !strings.Contains(chunks[len(chunks)-1].Content, "truncated") {
		t.Fatalf("expected truncation marker in %#v", chunks[len(chunks)-1])
	}
}

func TestAgentAgentsMarkdownBlock(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("ref"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("keep binary small"), 0o644); err != nil {
		t.Fatal(err)
	}
	prev, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	a := &Agent{Cfg: Config{WorkspaceRoot: dir}}
	block := a.agentsMarkdownBlock()
	if !strings.Contains(block, "keep binary small") {
		t.Fatalf("%q", block)
	}
}
