package main

import (
	"path/filepath"
	"testing"
)

func TestConnectionProfileLegacyMigration(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	if err := writeGuiPrefs(cfg, guiPrefs{Connect: "http://host:7420", AccessSecret: "s"}); err != nil {
		t.Fatal(err)
	}
	got, err := readGuiPrefs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Connections) < 2 {
		t.Fatalf("want local+remote, got %#v", got.Connections)
	}
	local := connectionByID(got.Connections, localConnectionID)
	if local == nil || local.Kind != connectionKindLocal {
		t.Fatalf("local missing: %#v", got.Connections)
	}
	if got.Connect != "http://host:7420" || got.AccessSecret != "s" {
		t.Fatalf("legacy mirror: %#v", got)
	}
	active := connectionByID(got.Connections, got.ActiveConnectionID)
	if active == nil || active.Kind != connectionKindRemote {
		t.Fatalf("active should be remote: %#v", got)
	}
}

func TestConnectionProfileLocalActiveClearsLegacy(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	p := guiPrefs{
		Connections: []connectionProfile{
			defaultLocalConnection(),
			{ID: "r1", Name: "Lab", Kind: connectionKindRemote, URL: "http://x", AccessSecret: "t"},
		},
		ActiveConnectionID: localConnectionID,
	}
	if err := writeGuiPrefs(cfg, p); err != nil {
		t.Fatal(err)
	}
	got, err := readGuiPrefs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.Connect != "" || got.AccessSecret != "" {
		t.Fatalf("local active should clear legacy connect: %#v", got)
	}
}
