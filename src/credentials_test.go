package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestVaultRoundTripAndWrongPassword(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	b := NewCredentialBroker(cfg)
	if err := b.Create("master-secret"); err != nil {
		t.Fatal(err)
	}
	_, err := b.Upsert(Credential{
		Name: "XMPP", Type: "password", Service: "chat.example.org",
		AllowedTools: []string{"mcp__xmpp__send"},
	}, "hunter2", true)
	if err != nil {
		t.Fatal(err)
	}
	b2 := NewCredentialBroker(cfg)
	if err := b2.Unlock("wrong"); err == nil {
		t.Fatal("expected wrong password error")
	}
	if err := b2.Unlock("master-secret"); err != nil {
		t.Fatal(err)
	}
	list, err := b2.ListForTool("mcp__xmpp__send")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %#v", err, list)
	}
	if _, ok := list[0]["secret"]; ok {
		t.Fatal("secret leaked in list")
	}
	_, sec, err := b2.ResolveSecret("mcp__xmpp__send", "")
	if err != nil || sec != "hunter2" {
		t.Fatalf("auto-pick: %v %q", err, sec)
	}
}

func TestUnauthorizedCredentialID(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	b := NewCredentialBroker(cfg)
	_ = b.Create("pw")
	c, err := b.Upsert(Credential{
		ID: "a", Name: "A", Type: "api_key", AllowedTools: []string{"mcp__a__t"},
	}, "sekrit", true)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = b.ResolveSecret("mcp__other__t", c.ID)
	if err == nil {
		t.Fatal("expected reject")
	}
	_, _, err = b.ResolveSecret("mcp__a__t", "nope")
	if err == nil {
		t.Fatal("expected reject unknown id")
	}
}

func TestPrepareToolArgsInjectAndStrip(t *testing.T) {
	dir := t.TempDir()
	b := NewCredentialBroker(filepath.Join(dir, "config.json"))
	_ = b.Create("pw")
	_, err := b.Upsert(Credential{
		ID: "x", Name: "X", Type: "password", AllowedTools: []string{"mcp__xmpp__send"},
		InjectArg: "password",
	}, "s3cret", true)
	if err != nil {
		t.Fatal(err)
	}
	out, sec, err := b.prepareToolArgs("mcp__xmpp__send", `{"recipient":"a@b","message":"hi","credential_id":"x"}`)
	if err != nil {
		t.Fatal(err)
	}
	if sec != "s3cret" {
		t.Fatalf("secret %q", sec)
	}
	if strings.Contains(out, "credential_id") {
		t.Fatalf("credential_id not stripped: %s", out)
	}
	if !strings.Contains(out, `"password":"s3cret"`) {
		t.Fatalf("not injected: %s", out)
	}
	red := redactSecret("got s3cret back", sec)
	if strings.Contains(red, "s3cret") {
		t.Fatalf("redact failed: %s", red)
	}
}

func TestInjectArgBlocklist(t *testing.T) {
	if err := validateInjectArg("url"); err == nil {
		t.Fatal("url should be blocked")
	}
	if err := validateInjectArg("password"); err != nil {
		t.Fatal(err)
	}
}

func TestMultipleRequiresID(t *testing.T) {
	dir := t.TempDir()
	b := NewCredentialBroker(filepath.Join(dir, "config.json"))
	_ = b.Create("pw")
	_, _ = b.Upsert(Credential{ID: "1", Name: "1", Type: "password", AllowedTools: []string{"t"}}, "a", true)
	_, _ = b.Upsert(Credential{ID: "2", Name: "2", Type: "password", AllowedTools: []string{"t"}}, "b", true)
	_, _, err := b.ResolveSecret("t", "")
	if err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("want multiple error, got %v", err)
	}
}
