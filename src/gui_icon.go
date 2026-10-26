//go:build cgo && !nogui

package main

import _ "embed"

//go:embed assets/keimling.png
var keimlingIconPNG []byte

//go:embed assets/keimling.svg
var keimlingIconSVG []byte

const desktopAppID = "keimling"

// prepareGUIIdentity sets the process name (Wayland app_id) and, on Linux,
// installs XDG icon + .desktop so KDE/Plasma can show the taskbar icon.
func prepareGUIIdentity() {
	setProgramName(desktopAppID)
	_ = installDesktopIcon()
}
