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

func TestSkillOriginHashAndDelete(t *testing.T) {
	home := t.TempDir()
	old := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", old) })
	_ = os.Setenv("HOME", home)
	cwd := t.TempDir()

	dir := writeSkillFixture(t, filepath.Join(home, ".config", "keimling", "skills"), "manage-me", "Manage skill for delete and origin tests when checking updates.", "body")
	_ = os.MkdirAll(filepath.Join(dir, "references"), 0o700)
	_ = os.WriteFile(filepath.Join(dir, "references", "x.md"), []byte("x"), 0o600)

	h1, err := hashSkillDir(dir)
	if err != nil || h1 == "" {
		t.Fatal(err)
	}
	if err := writeSkillOrigin(dir, "https://example.com/skill"); err != nil {
		t.Fatal(err)
	}
	h2, err := hashSkillDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatal("origin sidecar must not affect content hash")
	}
	sk, err := loadSkillDir(dir, skillSourceUser, true)
	if err != nil || sk.OriginURL != "https://example.com/skill" {
		t.Fatalf("%+v %v", sk, err)
	}
	if err := DeleteSkill("manage-me", cwd); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("expected removed")
	}
}

func TestCheckSkillUpdate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("---\nname: up-skill\ndescription: Update check skill for remote hash comparison tests.\n---\n\nv1\n"))
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
	st, err := CheckSkillUpdate(context.Background(), sk.Name, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != "up_to_date" {
		t.Fatalf("status %+v", st)
	}

	// Change remote content.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {}) // keep server
	// Replace handler by restarting — simpler: rewrite file via new server path.
	mux2 := http.NewServeMux()
	mux2.HandleFunc("/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("---\nname: up-skill\ndescription: Update check skill for remote hash comparison tests.\n---\n\nv2 changed\n"))
	})
	srv2 := httptest.NewServer(mux2)
	defer srv2.Close()
	_ = writeSkillOrigin(sk.Root, srv2.URL+"/SKILL.md")

	st, err = CheckSkillUpdate(context.Background(), sk.Name, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != "update_available" {
		t.Fatalf("expected update_available, got %+v", st)
	}
	_, st2, err := UpdateSkill(context.Background(), sk.Name, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if st2.Status != "updated" && st2.Status != "up_to_date" {
		t.Fatalf("after update %+v", st2)
	}
	b, _ := os.ReadFile(filepath.Join(sk.Root, "SKILL.md"))
	if !strings.Contains(string(b), "v2 changed") {
		// Root may have been replaced; reload path from registry.
		reg := LoadSkillRegistry(cwd)
		fresh := reg.Get("up-skill")
		b, _ = os.ReadFile(filepath.Join(fresh.Root, "SKILL.md"))
		if !strings.Contains(string(b), "v2 changed") {
			t.Fatalf("content not updated: %q", b)
		}
	}
}
