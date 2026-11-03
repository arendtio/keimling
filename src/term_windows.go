package main

import (
	"io"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func readTerminalByte(fd int, timeout time.Duration) (byte, error) {
	h := windows.Handle(fd)
	wait := uint32(windows.INFINITE)
	if timeout > 0 {
		wait = uint32(timeout.Milliseconds())
	}
	ev, err := windows.WaitForSingleObject(h, wait)
	if ev == uint32(windows.WAIT_TIMEOUT) {
		return 0, errKeyTimeout
	}
	if err != nil {
		return 0, err
	}
	var one [1]byte
	var n uint32
	if err := windows.ReadFile(h, one[:], &n, nil); err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, io.EOF
	}
	return one[0], nil
}

func discardPendingInput() {
	_ = windows.FlushConsoleInputBuffer(windows.Handle(os.Stdin.Fd()))
}

func terminalColumns(fd int) int {
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(windows.Handle(fd), &info); err != nil {
		return 80
	}
	w := int(info.Window.Right - info.Window.Left + 1)
	if w < 12 {
		return 80
	}
	return w
}
