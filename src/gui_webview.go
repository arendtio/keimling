//go:build cgo && !nogui

package main

import (
	"fmt"

	webview "github.com/samcharles93/webview_go"
)

func openWebViewImpl(url, title, configPath string) error {
	if title == "" {
		title = "Keimling"
	}
	prefs, _ := readGuiPrefs(configPath)
	width, height := windowSizeFromPrefs(prefs)
	prepareGUIIdentity()
	w, err := webview.NewWithOptions(webview.Options{
		Title:  title,
		Width:  width,
		Height: height,
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
	_ = w.Bind("keimlingSaveGuiGeometry", func(width, height int) {
		_ = saveGuiWindowSize(configPath, width, height)
	})
	if err := w.Run(); err != nil {
		return fmt.Errorf("%w: %v", errGUIUnavailable, err)
	}
	return nil
}
