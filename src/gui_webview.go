//go:build cgo && !nogui

package main

import (
	"fmt"

	webview "github.com/samcharles93/webview_go"
)

func openWebViewImpl(url, title string) error {
	if title == "" {
		title = "Keimling"
	}
	prepareGUIIdentity()
	w, err := webview.NewWithOptions(webview.Options{
		Title:  title,
		Width:  960,
		Height: 720,
		Hint:   webview.HintNone,
		URL:    url,
		Debug:  false,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", errGUIUnavailable, err)
	}
	defer w.Destroy()
	applyWindowIcon(w)
	allowMediaPermissions(w)
	if err := w.Run(); err != nil {
		return fmt.Errorf("%w: %v", errGUIUnavailable, err)
	}
	return nil
}
