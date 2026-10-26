//go:build (linux || openbsd || freebsd || netbsd) && webkitgtk_40 && !webkitgtk_41 && !webkitgtk_60

package webview

/*
#cgo linux openbsd freebsd netbsd pkg-config: gtk+-3.0 webkit2gtk-4.0
*/
import "C"

// Drop this file into a replace'd copy of github.com/samcharles93/webview_go
// (and tighten webkitgtk_41 / webkitgtk_60 build constraints to exclude
// webkitgtk_40) when building on distros that only ship WebKitGTK 4.0.
