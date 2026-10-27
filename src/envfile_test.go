package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvOrFilePrefersEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "from-env")
	t.Setenv("OPENAI_API_KEY_FILE", "/no/such/file")
	got, err := envOrFile("OPENAI_API_KEY")
	if err != nil || got != "from-env" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestEnvOrFileReadsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("  file-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_API_KEY_FILE", path)
	got, err := envOrFile("OPENAI_API_KEY")
	if err != nil || got != "file-secret" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestEnvOrFileMissingFileErrors(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_API_KEY_FILE", filepath.Join(t.TempDir(), "missing"))
	if _, err := envOrFile("OPENAI_API_KEY"); err == nil {
		t.Fatal("expected error")
	}
}

func TestAccessSecretFromEnvAliases(t *testing.T) {
	t.Setenv("KEIMLING_ACCESS_SECRET", "")
	t.Setenv("ACCESS_SECRET", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "s")
	if err := os.WriteFile(path, []byte("compose-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KEIMLING_SECRET", "")
	t.Setenv("KEIMLING_SECRET_FILE", path)
	got, err := accessSecretFromEnv()
	if err != nil || got != "compose-secret" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestAPIKeyUsesFileEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key")
	if err := os.WriteFile(path, []byte("sk-test"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_API_KEY_FILE", path)
	// Avoid falling through to real home credentials.
	t.Setenv("HOME", dir)
	got, err := apiKey()
	if err != nil || got != "sk-test" {
		t.Fatalf("got %q err=%v", got, err)
	}
}
