package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkillFixture(t *testing.T, root, name, desc, body string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + desc + "\n---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestParseSkillValidAndInvalid(t *testing.T) {
	raw := "---\nname: pdf-tools\ndescription: Extract PDF text. Use when handling PDFs.\n---\n\n# Hi\n"
	sk, err := parseSkillMarkdown(raw, "pdf-tools", "/tmp/pdf-tools", skillSourceUser, true)
	if err != nil || sk.Name != "pdf-tools" || !strings.Contains(sk.Body, "# Hi") {
		t.Fatalf("%v %+v", err, sk)
	}
	if _, err := parseSkillMarkdown(raw, "other", "/tmp/other", skillSourceUser, true); err == nil {
		t.Fatal("expected name/dir mismatch")
	}
	if _, err := parseSkillMarkdown("---\nname: Bad_Name\ndescription: x\n---\n", "Bad_Name", "", skillSourceUser, true); err == nil {
		t.Fatal("expected invalid name")
	}
	if _, err := parseSkillMarkdown("---\nname: ok\n---\n", "ok", "", skillSourceUser, true); err == nil {
		t.Fatal("expected missing description")
	}
}

func TestSkillRegistryPrecedence(t *testing.T) {
	cwd := t.TempDir()
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", oldHome) })
	_ = os.Setenv("HOME", home)

	proj := filepath.Join(cwd, ".keimling", "skills")
	user := filepath.Join(home, ".config", "keimling", "skills")
	writeSkillFixture(t, proj, "demo", "project skill description here", "project body")
	writeSkillFixture(t, user, "demo", "user skill description here", "user body")
	writeSkillFixture(t, user, "only-user", "user only skill description", "u")

	reg := LoadSkillRegistry(cwd)
	demo := reg.Get("demo")
	if demo == nil || demo.Source != skillSourceProject {
		t.Fatalf("expected project to win: %+v", demo)
	}
	if reg.Get("only-user") == nil {
		t.Fatal("missing user skill")
	}
	cat := reg.CatalogBlock()
	if !strings.Contains(cat, "demo:") || !strings.Contains(cat, "<agent_skills>") {
		t.Fatal(cat)
	}
}

func TestReadSkillFilePathEscape(t *testing.T) {
	root := t.TempDir()
	dir := writeSkillFixture(t, root, "safe-skill", "Safe skill for path tests when reading files.", "body")
	_ = os.WriteFile(filepath.Join(dir, "notes.md"), []byte("secret-ish"), 0600)
	reg := &SkillRegistry{byName: map[string]*SkillMeta{
		"safe-skill": {Name: "safe-skill", Description: "Safe skill for path tests when reading files.", Root: dir, Source: skillSourceUser, Writable: true},
	}, order: []string{"safe-skill"}}
	if _, err := reg.ReadFile("safe-skill", "notes.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.ReadFile("safe-skill", "../notes.md"); err == nil {
		t.Fatal("expected escape rejection")
	}
	if _, err := reg.ReadFile("safe-skill", "/etc/passwd"); err == nil {
		t.Fatal("expected abs rejection")
	}
}

func TestInstallSkillFromPath(t *testing.T) {
	srcRoot := t.TempDir()
	src := writeSkillFixture(t, srcRoot, "install-me", "Installable skill description for fixture copy.", "# Installed")
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", oldHome) })
	_ = os.Setenv("HOME", home)

	sk, err := InstallSkillFromPath(src, SkillScopeUser, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "keimling", "skills", "install-me")
	if sk.Root != want {
		t.Fatalf("root %s want %s", sk.Root, want)
	}
	if _, err := os.Stat(filepath.Join(want, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	// Also accept SKILL.md path
	src2 := writeSkillFixture(t, srcRoot, "via-file", "Second installable skill description here.", "x")
	if _, err := InstallSkillFromPath(filepath.Join(src2, "SKILL.md"), SkillScopeUser, t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestExtractSkillCandidates(t *testing.T) {
	text := `Try [Cool Skill](https://github.com/acme/cool-skill/tree/main) and https://raw.githubusercontent.com/acme/x/main/SKILL.md`
	cands := extractSkillCandidates(text)
	if len(cands) < 2 {
		t.Fatalf("%+v", cands)
	}
}
