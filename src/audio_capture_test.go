package main

import (
	"runtime"
	"testing"
)

func TestLinuxVoiceCaptureUsesArecord(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("linux recorder")
	}
	name, args, err := captureArgs("clip.pcm", "3")
	if err != nil {
		t.Fatal(err)
	}
	if name != "arecord" || len(args) == 0 || args[len(args)-1] != "clip.pcm" {
		t.Fatalf("got %s %v", name, args)
	}
}
