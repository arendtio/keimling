//go:build !cgo || nogui

package main

func openWebViewImpl(url, title, configPath string) error {
	return errGUIUnavailable
}
