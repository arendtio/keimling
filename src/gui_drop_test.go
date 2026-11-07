package main

import (
	"strings"
	"testing"
)

func TestFileDropEvalKeepsTheChat(t *testing.T) {
	js, ok := fileDropEval("file:///home/huck/mpv-shot0001.jpg")
	if !ok || !strings.Contains(js, "keimlingAcceptDrop") || !strings.Contains(js, "mpv-shot0001.jpg") {
		t.Fatalf("%v %s", ok, js)
	}
	if _, ok := fileDropEval("http://127.0.0.1:9/?access_secret=abc"); ok {
		t.Fatal("the app address must stay loaded")
	}
	js, ok = fileDropEval("file:///tmp/a\"b.jpg")
	if !ok || !strings.Contains(js, `a\"b.jpg`) {
		t.Fatalf("quote must stay inside the script: %v %s", ok, js)
	}
}
