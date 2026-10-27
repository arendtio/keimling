package main

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

// openExternalURL opens http(s) URLs in the desktop browser (not the WebView).
func openExternalURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported url scheme")
	}
	href := u.String()
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd", "netbsd":
		cmd = exec.Command("xdg-open", href)
	case "darwin":
		cmd = exec.Command("open", href)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", href)
	default:
		return fmt.Errorf("open external url unsupported on %s", runtime.GOOS)
	}
	return cmd.Start()
}
