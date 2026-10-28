package main

import (
	"path/filepath"
	"testing"
)

func TestClientConnectionsMigrateFromGuiPrefs(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	if err := writeGuiPrefs(cfg, guiPrefs{Connect: "http://host:7420", AccessSecret: "s"}); err != nil {
		t.Fatal(err)
	}
	f, err := readClientConnections(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Connections) != 1 || f.Connections[0].URL != "http://host:7420" {
		t.Fatalf("migrated: %#v", f.Connections)
	}
	if f.Connections[0].AccessSecret != "s" {
		t.Fatalf("secret: %#v", f.Connections[0])
	}
	// Second read uses connections.json, not re-copy duplicates.
	f2, err := readClientConnections(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(f2.Connections) != 1 {
		t.Fatalf("dupes: %#v", f2.Connections)
	}
}

func TestUpsertClientRemote(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	if err := upsertClientRemote(cfg, "Lab", "http://lab:7420", "tok"); err != nil {
		t.Fatal(err)
	}
	if err := upsertClientRemote(cfg, "Lab2", "http://lab:7420/", "tok2"); err != nil {
		t.Fatal(err)
	}
	f, err := readClientConnections(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Connections) != 1 {
		t.Fatalf("want 1, got %#v", f.Connections)
	}
	if f.Connections[0].Name != "Lab2" || f.Connections[0].AccessSecret != "tok2" {
		t.Fatalf("%#v", f.Connections[0])
	}
}

func TestResolveConnectReference(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	_ = upsertClientRemote(cfg, "Prod", "http://prod:7420", "sec")
	f, _ := readClientConnections(cfg)
	id := f.Connections[0].ID

	u, s, ok := resolveConnectReference(cfg, "Prod")
	if !ok || u != "http://prod:7420" || s != "sec" {
		t.Fatalf("by name: %v %q %q", ok, u, s)
	}
	u, s, ok = resolveConnectReference(cfg, id)
	if !ok || u != "http://prod:7420" {
		t.Fatalf("by id: %v %q", ok, u)
	}
	u, s, ok = resolveConnectReference(cfg, "http://x/")
	if ok || u != "http://x/" {
		t.Fatalf("url passthrough: %v %q %q", ok, u, s)
	}
}

func TestNormalizeClientDropsLocal(t *testing.T) {
	f := clientConnectionsFile{Connections: []connectionProfile{
		defaultLocalConnection(),
		{ID: "r", Name: "R", Kind: connectionKindRemote, URL: "http://x"},
	}}
	normalizeClientConnections(&f)
	if len(f.Connections) != 1 || f.Connections[0].ID != "r" {
		t.Fatalf("%#v", f.Connections)
	}
}

func TestClearClientActiveConnectionKeepsProfiles(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	if err := upsertClientRemote(cfg, "Prod", "http://prod:7420", "sec"); err != nil {
		t.Fatal(err)
	}
	if err := clearClientActiveConnection(cfg); err != nil {
		t.Fatal(err)
	}
	f, err := readClientConnections(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if f.ActiveID != "" {
		t.Fatalf("active_id=%q", f.ActiveID)
	}
	if len(f.Connections) != 1 {
		t.Fatalf("profiles: %#v", f.Connections)
	}
	var connect, secret string
	if err := applyClientConnectionsConnect(cfg, &connect, &secret, false); err != nil {
		t.Fatal(err)
	}
	if connect != "" {
		t.Fatalf("should not auto-connect, got %q", connect)
	}
}
