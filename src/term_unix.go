//go:build linux || darwin

package main

import (
	"io"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func readTerminalByte(fd int, timeout time.Duration) (byte, error) {
	if timeout > 0 {
		var set unix.FdSet
		set.Set(fd)
		tv := unix.NsecToTimeval(timeout.Nanoseconds())
		n, err := unix.Select(fd+1, &set, nil, nil, &tv)
		if err == syscall.EINTR {
			return 0, errKeyTimeout
		}
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, errKeyTimeout
		}
	}
	var one [1]byte
	for {
		n, e := syscall.Read(fd, one[:])
		if e == syscall.EINTR {
			continue
		}
		if e != nil {
			return 0, e
		}
		if n == 0 {
			return 0, io.EOF
		}
		return one[0], nil
	}
}

func terminalColumns(fd int) int {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil || ws.Col < 12 {
		return 80
	}
	return int(ws.Col)
}
