//go:build !cgo || nogui

package main

func openWebViewImpl(url, title string) error {
	return errGUIUnavailable
}
