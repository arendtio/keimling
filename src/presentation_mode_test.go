package main

import "testing"

func TestResolvePresentationFlagsOverrideConfig(t *testing.T) {
	got, err := resolvePresentation("server", false, true, false)
	if err != nil || got != presentationCLI {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestResolvePresentationMutualExclusion(t *testing.T) {
	if _, err := resolvePresentation("", true, true, false); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolvePresentationConfig(t *testing.T) {
	got, err := resolvePresentation("server", false, false, false)
	if err != nil || got != presentationServer {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestValidateServerListen(t *testing.T) {
	if err := validateServerListen("", "x"); err == nil {
		t.Fatal("expected listen error")
	}
	if err := validateServerListen("127.0.0.1:9", ""); err == nil {
		t.Fatal("expected secret error")
	}
	if err := validateServerListen("127.0.0.1:9", "x"); err != nil {
		t.Fatal(err)
	}
}
