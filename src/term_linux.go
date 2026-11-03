package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func discardPendingInput() {
	// TCFLSH with 0 discards queued input (TCIFLUSH).
	_, _, _ = unix.Syscall(unix.SYS_IOCTL, os.Stdin.Fd(), uintptr(unix.TCFLSH), 0)
}

func getTermios(fd int) (*unix.Termios, error) {
	return unix.IoctlGetTermios(fd, unix.TCGETS)
}
