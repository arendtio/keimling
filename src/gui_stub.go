//go:build !cgo || nogui

package main

func openWebViewImpl(url, title, configPath string) error {
	return errGUIUnavailable
}

func openWebViewWithHomeImpl(url, title, configPath, localHome string) error {
	return errGUIUnavailable
}

func openWebViewPreparedImpl(url, title, configPath, localHome string, beforeRun func() error) error {
	return errGUIUnavailable
}
