package main

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func openPTY() (*os.File, *os.File, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	closeMaster := func(err error) (*os.File, *os.File, error) { master.Close(); return nil, nil, err }
	fd := master.Fd()
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(unix.TIOCPTYGRANT), 0); errno != 0 {
		return closeMaster(errno)
	}
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(unix.TIOCPTYUNLK), 0); errno != 0 {
		return closeMaster(errno)
	}
	var name [128]byte
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(unix.TIOCPTYGNAME), uintptr(unsafe.Pointer(&name[0]))); errno != 0 {
		return closeMaster(errno)
	}
	n := 0
	for n < len(name) && name[n] != 0 {
		n++
	}
	slave, err := os.OpenFile(string(name[:n]), os.O_RDWR, 0)
	if err != nil {
		return closeMaster(err)
	}
	return master, slave, nil
}
