//go:build cgo && !nogui && !linux

package main

import webview "github.com/samcharles93/webview_go"

func setProgramName(name string) {}

func applyWindowIcon(w webview.WebView) {}

func installDesktopIcon() error { return nil }
