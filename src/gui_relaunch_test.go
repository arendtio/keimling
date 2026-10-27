package main

import "testing"

func TestFilterConnectArgs(t *testing.T) {
	got := filterConnectArgs([]string{"--ui", "--connect", "http://remote:7420", "--access-secret", "x"})
	want := []string{"--ui", "--access-secret", "x"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	got = filterConnectArgs([]string{"--connect=http://r", "--cli"})
	if len(got) != 1 || got[0] != "--cli" {
		t.Fatalf("%v", got)
	}
}

func TestIsLoopbackURL(t *testing.T) {
	if !isLoopbackURL("http://127.0.0.1:1234/?access_secret=a") {
		t.Fatal("127.0.0.1")
	}
	if isLoopbackURL("https://example.com/") {
		t.Fatal("remote")
	}
}
