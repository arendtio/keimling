package main

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func discardPendingInput() {
	arg := 1 // FREAD
	_, _, _ = unix.Syscall(unix.SYS_IOCTL, os.Stdin.Fd(), uintptr(unix.TIOCFLUSH), uintptr(unsafe.Pointer(&arg)))
}

func getTermios(fd int) (*unix.Termios, error) {
	return unix.IoctlGetTermios(fd, unix.TIOCGETA)
}
