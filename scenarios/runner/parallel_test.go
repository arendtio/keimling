package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCopyConfigSkipsTheVault(t *testing.T) {
	src := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", src)
	dir := filepath.Join(src, "keimling")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{\"model\":\"m\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mcp.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "credentials.vault"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, home, err := copyUserConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(cfg)
	if err != nil || !strings.Contains(string(b), `"m"`) {
		t.Fatalf("%s %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(home, "keimling", "credentials.vault")); !os.IsNotExist(err) {
		t.Fatal("vault was copied")
	}
	if _, err := os.Stat(filepath.Join(home, "keimling", "mcp.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("{\"model\":\"changed\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil || strings.Contains(string(b), "changed") {
		t.Fatalf("source changed: %s %v", b, err)
	}
}

func TestChildEnvReplacesConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/real")
	t.Setenv("XDG_DATA_HOME", "/real-data")
	env := childEnv("/tmp/data", "/tmp/config")
	data, cfg := 0, 0
	for _, e := range env {
		switch {
		case strings.HasPrefix(e, "XDG_DATA_HOME="):
			data++
			if e != "XDG_DATA_HOME=/tmp/data" {
				t.Fatal(e)
			}
		case strings.HasPrefix(e, "XDG_CONFIG_HOME="):
			cfg++
			if e != "XDG_CONFIG_HOME=/tmp/config" {
				t.Fatal(e)
			}
		}
	}
	if data != 1 || cfg != 1 {
		t.Fatalf("data %d config %d", data, cfg)
	}
}

func TestRunLimitedCapsInFlight(t *testing.T) {
	if gateSize(11, 0) != 11 || gateSize(11, 4) != 4 || gateSize(3, 9) != 3 {
		t.Fatal(gateSize(11, 0))
	}
	var mu sync.Mutex
	var active, peak int
	err := runLimited(6, 2, func(int) error {
		mu.Lock()
		active++
		if active > peak {
			peak = active
		}
		mu.Unlock()
		time.Sleep(30 * time.Millisecond)
		mu.Lock()
		active--
		mu.Unlock()
		return nil
	})
	if err != nil || peak != 2 {
		t.Fatalf("peak %d err %v", peak, err)
	}
}
