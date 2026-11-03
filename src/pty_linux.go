package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func ioctl(fd uintptr, req uintptr, data uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, data)
	if errno != 0 {
		return errno
	}
	return nil
}

func openPTY() (*os.File, *os.File, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	closeMaster := func(err error) (*os.File, *os.File, error) { master.Close(); return nil, nil, err }
	var unlock int32
	if err = ioctl(master.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); err != nil {
		return closeMaster(err)
	}
	var number uint32
	if err = ioctl(master.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&number))); err != nil {
		return closeMaster(err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR, 0)
	if err != nil {
		return closeMaster(err)
	}
	return master, slave, nil
}
