package main

import (
	"encoding/json"
	"strings"
)

// fileDropEval is the page call for a file the window was asked to open.
// Dolphin drops a file:// address, and WebKit would otherwise replace the UI with that file.
func fileDropEval(uri string) (string, bool) {
	uri = strings.TrimSpace(uri)
	if !strings.HasPrefix(strings.ToLower(uri), "file:") {
		return "", false
	}
	encoded, err := json.Marshal(uri)
	if err != nil {
		return "", false
	}
	return "if (window.keimlingAcceptDrop) window.keimlingAcceptDrop(" + string(encoded) + ");", true
}
