package main

import "errors"

// errGUIUnavailable is returned when this binary has no native WebView
// (CGO/nogui build) or the window cannot be created.
var errGUIUnavailable = errors.New("native GUI unavailable")

// openWebView opens a system webview window navigated to url and blocks until closed.
// configPath is used to restore/persist window size in gui-prefs.json.
// Implementations: gui_webview.go (cgo) or gui_stub.go.
func openWebView(url, title, configPath string) error {
	return openWebViewImpl(url, title, configPath)
}
