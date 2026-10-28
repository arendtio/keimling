package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractSkillRelativePaths(t *testing.T) {
	md := `# Skill
See [guide](references/guide.md) and ` + "`scripts/run.sh`" + `.
Also references/extra.md and https://example.com/skip.md
`
	got := extractSkillRelativePaths(md)
	joined := strings.Join(got, ",")
	if !strings.Contains(joined, "references/guide.md") || !strings.Contains(joined, "scripts/run.sh") || !strings.Contains(joined, "references/extra.md") {
		t.Fatalf("got %v", got)
	}
	for _, p := range got {
		if strings.Contains(p, "example.com") {
			t.Fatal("should skip absolute URLs")
		}
	}
}

func TestParseGitHubSkillLocation(t *testing.T) {
	loc, err := parseGitHubSkillLocation("https://github.com/acme/tools/tree/main/skills/pdf")
	if err != nil || loc.Owner != "acme" || loc.Repo != "tools" || loc.Ref != "main" || loc.Dir != "skills/pdf" {
		t.Fatalf("%+v %v", loc, err)
	}
	loc, err = parseGitHubSkillLocation("https://github.com/acme/tools/blob/main/skills/pdf/SKILL.md")
	if err != nil || loc.Dir != "skills/pdf" {
		t.Fatalf("%+v %v", loc, err)
	}
	loc, err = parseGitHubSkillLocation("https://raw.githubusercontent.com/acme/tools/v1/skills/pdf/SKILL.md")
	if err != nil || loc.Ref != "v1" || loc.Dir != "skills/pdf" {
		t.Fatalf("%+v %v", loc, err)
	}
}

func TestFetchSkillBundleFollowsLinks(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repo/main/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("---\nname: demo-skill\ndescription: Demo skill for fetch tests when bundling files.\n---\n\nSee [notes](references/notes.md).\n"))
	})
	mux.HandleFunc("/repo/main/references/notes.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("# Notes\nAlso `scripts/helper.sh`\n"))
	})
	mux.HandleFunc("/repo/main/scripts/helper.sh", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#!/bin/sh\necho ok\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Non-GitHub URL path: resolveSiblingURL based off SKILL.md URL.
	skillURL := srv.URL + "/repo/main/SKILL.md"
	dest := t.TempDir()
	sk, files, err := fetchSkillBundle(context.Background(), skillURL, dest)
	if err != nil {
		t.Fatal(err)
	}
	if sk.Name != "demo-skill" {
		t.Fatalf("name %q", sk.Name)
	}
	paths := map[string]bool{}
	for _, f := range files {
		paths[f.Path] = true
	}
	for _, need := range []string{"SKILL.md", "references/notes.md", "scripts/helper.sh"} {
		if !paths[need] {
			t.Fatalf("missing %s in %v", need, paths)
		}
	}
	b, err := os.ReadFile(filepath.Join(dest, "demo-skill", "scripts", "helper.sh"))
	if err != nil || !strings.Contains(string(b), "echo ok") {
		t.Fatalf("helper: %v %q", err, b)
	}
}

func TestInstallSkillFromURLUsesBundle(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("---\nname: url-skill\ndescription: URL install skill with a linked reference file.\n---\n\nRead [ref](references/a.md).\n"))
	})
	mux.HandleFunc("/references/a.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("linked"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	home := t.TempDir()
	old := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", old) })
	_ = os.Setenv("HOME", home)
	cwd := t.TempDir()

	sk, err := InstallSkillFromURL(srv.URL+"/SKILL.md", SkillScopeUser, cwd)
	if err != nil {
		t.Fatal(err)
	}
	ref := filepath.Join(sk.Root, "references", "a.md")
	b, err := os.ReadFile(ref)
	if err != nil || string(b) != "linked" {
		t.Fatalf("bundled file missing: %v %q", err, b)
	}
}
