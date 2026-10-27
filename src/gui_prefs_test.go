package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuiPrefsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	if err := writeGuiPrefs(cfg, guiPrefs{Connect: "https://agent.example", AccessSecret: "sekrit"}); err != nil {
		t.Fatal(err)
	}
	path := guiPrefsPath(cfg)
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0077 != 0 {
		t.Fatalf("perms %o", st.Mode().Perm())
	}
	got, err := readGuiPrefs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.Connect != "https://agent.example" || got.AccessSecret != "sekrit" {
		t.Fatalf("%#v", got)
	}
	if err := writeGuiPrefs(cfg, guiPrefs{}); err != nil {
		t.Fatal(err)
	}
	got, err = readGuiPrefs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.Connect != "" || got.AccessSecret != "" {
		t.Fatalf("expected cleared, got %#v", got)
	}
}

func TestApplyGuiPrefsConnect(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	_ = writeGuiPrefs(cfg, guiPrefs{Connect: "http://127.0.0.1:9", AccessSecret: "tok"})
	connect, secret := "", ""
	if err := applyGuiPrefsConnect(cfg, &connect, &secret, false); err != nil {
		t.Fatal(err)
	}
	if connect != "http://127.0.0.1:9" || secret != "tok" {
		t.Fatalf("connect=%q secret=%q", connect, secret)
	}
	connect, secret = "already", ""
	if err := applyGuiPrefsConnect(cfg, &connect, &secret, false); err != nil {
		t.Fatal(err)
	}
	if connect != "already" {
		t.Fatalf("should not override existing connect: %q", connect)
	}
	connect, secret = "", "keep"
	if err := applyGuiPrefsConnect(cfg, &connect, &secret, true); err != nil {
		t.Fatal(err)
	}
	if connect != "" || secret != "keep" {
		t.Fatalf("flag set should skip: %q %q", connect, secret)
	}
}

func TestAppServerGuiPrefsAPI(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	a := &Agent{Cfg: Config{ConfigPath: cfg}}
	h := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a}).Handler()

	req := httptest.NewRequest(http.MethodPut, "/api/gui-prefs", strings.NewReader(`{"connect":"https://r.example","access_secret":"remote-tok"}`))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put: %d %s", rec.Code, rec.Body.String())
	}
	got, err := readGuiPrefs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.Connect != "https://r.example" || got.AccessSecret != "remote-tok" {
		t.Fatalf("%#v", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/gui-prefs", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: %d", rec.Code)
	}
	var body guiPrefs
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Connect != "https://r.example" {
		t.Fatalf("%#v", body)
	}
}
