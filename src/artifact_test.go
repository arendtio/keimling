package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShowArtifact(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	htmlPath := "demo.html"
	if err := os.WriteFile(htmlPath, []byte("<html><body>hi</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	var events []string
	a := &Agent{
		Shares:    newFileShareHub(),
		Artifacts: newArtifactHub(),
		Emit: func(kind, text string) {
			if kind == "artifact" {
				events = append(events, text)
			}
		},
	}
	out := a.showArtifact(`{"path":"demo.html","title":"Demo"}`)
	if !strings.Contains(out, "artifact shown") {
		t.Fatalf("out: %q", out)
	}
	list := a.Artifacts.List()
	if len(list) != 1 || list[0].Title != "Demo" || list[0].Token == "" {
		t.Fatalf("list: %+v", list)
	}
	if len(events) != 1 {
		t.Fatalf("events: %v", events)
	}
	var art Artifact
	if err := json.Unmarshal([]byte(events[0]), &art); err != nil || art.Title != "Demo" {
		t.Fatalf("event: %v %q", err, events[0])
	}

	srv := newAppServer(appServerOpts{AccessSecret: "secret", Agent: a})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/files/"+list[0].Token, nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "hi") {
		t.Fatalf("serve: %d %q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Disposition"), "inline") {
		t.Fatalf("disposition: %q", rec.Header().Get("Content-Disposition"))
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/artifacts", nil)
	req2.Header.Set("Authorization", "Bearer secret")
	srv.Handler().ServeHTTP(rec2, req2)
	if rec2.Code != 200 || !strings.Contains(rec2.Body.String(), "Demo") {
		t.Fatalf("api: %d %q", rec2.Code, rec2.Body.String())
	}
}

func TestShowArtifactRejectsOutside(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "out.html")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	work := t.TempDir()
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	a := &Agent{Shares: newFileShareHub(), Artifacts: newArtifactHub()}
	out := a.showArtifact(`{"path":` + mustJSON(outside) + `,"title":"x"}`)
	if !strings.Contains(out, "error") {
		t.Fatalf("expected error, got %q", out)
	}
}

func mustJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestAutoApprovedShowArtifact(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	_ = os.WriteFile("a.html", []byte("<html></html>"), 0o644)
	if !autoApproved("show_artifact", `{"path":"a.html","title":"t"}`) {
		t.Fatal("expected auto approve inside cwd")
	}
	if autoApproved("show_artifact", `{"path":"../a.html"}`) {
		t.Fatal("expected reject outside")
	}
}
