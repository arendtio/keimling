package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShareFileRegisterAndServe(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("hello share"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{PublicURL: "https://keimling.example"}, Shares: newFileShareHub()}
	a.Shares.SetPublicBase(a.Cfg.PublicURL)
	out := a.shareFile(`{"path":"note.txt","as":"auto"}`)
	if !strings.Contains(out, "https://keimling.example/files/") {
		t.Fatalf("url missing: %s", out)
	}
	token := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "token: ") {
			token = strings.TrimPrefix(line, "token: ")
		}
	}
	if token == "" {
		t.Fatal("no token")
	}

	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})
	req := httptest.NewRequest(http.MethodGet, "/files/"+token, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("serve: %d", rec.Code)
	}
	if body := rec.Body.String(); body != "hello share" {
		t.Fatalf("body %q", body)
	}
	if disp := rec.Header().Get("Content-Disposition"); !strings.Contains(disp, "note.txt") || !strings.HasPrefix(disp, "inline") {
		t.Fatalf("disposition %q", disp)
	}

	out = a.shareFile(`{"path":"note.txt","as":"download"}`)
	tokenDL := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "token: ") {
			tokenDL = strings.TrimPrefix(line, "token: ")
		}
	}
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/"+tokenDL, nil))
	if !strings.HasPrefix(rec.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("download disposition %q", rec.Header().Get("Content-Disposition"))
	}

	// Unknown token
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/deadbeef", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown: %d", rec.Code)
	}

	// Outside cwd rejected
	out = a.shareFile(`{"path":"` + filepath.Join(cwd, "nope") + `"}`)
	if !strings.Contains(out, "error") {
		t.Fatalf("expected error: %s", out)
	}

	a.Shares.Clear()
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/"+token, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cleared: %d", rec.Code)
	}
}

func TestPublicBaseFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://internal:7420/", nil)
	req.Host = "internal:7420"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "keimling.example")
	if got := publicBaseFromRequest(req); got != "https://keimling.example" {
		t.Fatal(got)
	}
}

func TestShareFileAutoApproved(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	_ = os.WriteFile("x.txt", []byte("x"), 0o600)
	if !autoApproved("share_file", `{"path":"x.txt"}`) {
		t.Fatal("expected auto approve in cwd")
	}
	if autoApproved("share_file", `{"path":"../x.txt"}`) {
		t.Fatal("expected reject outside cwd")
	}
}
