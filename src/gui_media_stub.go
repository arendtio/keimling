//go:build cgo && !nogui && !(linux && webkitgtk_41)

package main

import webview "github.com/samcharles93/webview_go"

func allowMediaPermissions(w webview.WebView) {}
