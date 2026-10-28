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
	if err := saveGuiWindowSize(cfg, 1400, 1000); err != nil {
		t.Fatal(err)
	}
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
	if got.WindowWidth != 1400 || got.WindowHeight != 1000 {
		t.Fatalf("window size clobbered: %#v", got)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/gui-prefs", strings.NewReader(`{"connect":"","access_secret":""}`))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body.String())
	}
	got, err = readGuiPrefs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.Connect != "" || got.WindowWidth != 1400 {
		t.Fatalf("clear should keep window size: %#v", got)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/gui-prefs", strings.NewReader(`{"bg_light":"#f0f0f0","bg_dark":"#112233"}`))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("theme put: %d %s", rec.Code, rec.Body.String())
	}
	got, err = readGuiPrefs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.BgLight != "#f0f0f0" || got.BgDark != "#112233" || got.WindowWidth != 1400 {
		t.Fatalf("theme merge: %#v", got)
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
	if body.WindowWidth != 1400 || body.BgDark != "#112233" {
		t.Fatalf("%#v", body)
	}
}

func TestNormalizeBgColor(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "", true},
		{"#abc", "#aabbcc", true},
		{"#AABBCC", "#aabbcc", true},
		{"#gg0000", "", false},
		{"red", "", false},
		{"#12345", "", false},
	}
	for _, c := range cases {
		got, ok := normalizeBgColor(c.in)
		if ok != c.ok || got != c.want {
			t.Fatalf("%q: got %q %v want %q %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestWindowSizeFromPrefs(t *testing.T) {
	w, h := windowSizeFromPrefs(guiPrefs{})
	if w != defaultWindowWidth || h != defaultWindowHeight {
		t.Fatalf("%d x %d", w, h)
	}
	w, h = windowSizeFromPrefs(guiPrefs{WindowWidth: 1600, WindowHeight: 900})
	if w != 1600 || h != 900 {
		t.Fatalf("%d x %d", w, h)
	}
}
