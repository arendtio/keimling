package main

import "testing"

func TestUIURL(t *testing.T) {
	got := uiURL("http://127.0.0.1:9/", "abc")
	if got != "http://127.0.0.1:9/?access_secret=abc" {
		t.Fatalf("%q", got)
	}
}

func TestRandomAccessSecret(t *testing.T) {
	a, err := randomAccessSecret()
	if err != nil || len(a) != 32 {
		t.Fatalf("%q %v", a, err)
	}
}
