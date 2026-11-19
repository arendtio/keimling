//go:build cgo && !nogui && darwin

package main

/*
#cgo LDFLAGS: -framework Cocoa

void keimlingEditMenu(void);
*/
import "C"

func prepareDesktopApp() { C.keimlingEditMenu() }
