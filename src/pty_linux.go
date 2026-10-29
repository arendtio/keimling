package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
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

// interactiveExec gives the foreground child exclusive use of the terminal.
// The child owns its PTY slave, while this process forwards bytes via its master.
// No background reader is started, and original fd flags/termios are restored.
func interactiveExec(ctx context.Context, shell, command string, timeout time.Duration, extraEnv ...string) string {
	if !isTerminal(os.Stdin.Fd()) {
		return "interactive execution unavailable: stdin is not a terminal"
	}
	master, slave, err := openPTY()
	if err != nil {
		return "PTY error: " + err.Error()
	}
	defer master.Close()
	defer slave.Close()
	// Keep dimensions consistent with the user's actual terminal.
	var ws [8]byte
	if ioctl(os.Stdin.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws[0]))) == nil {
		_ = ioctl(slave.Fd(), syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(&ws[0])))
	}
	cmd := exec.Command(shell, "-c", command)
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	old, err := getTermios(os.Stdin.Fd())
	if err != nil {
		return "terminal state error: " + err.Error()
	}
	raw := *old
	raw.Lflag &^= syscall.ICANON | syscall.ECHO | syscall.ISIG | syscall.IEXTEN
	raw.Iflag &^= syscall.IXON | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP
	raw.Oflag &^= syscall.OPOST
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if err := setTermios(os.Stdin.Fd(), &raw); err != nil {
		return "terminal setup error: " + err.Error()
	}
	defer setTermios(os.Stdin.Fd(), old)
	fmt.Println("[Interactive terminal attached; Ctrl+C interrupts child]")
	if err := cmd.Start(); err != nil {
		return "start error: " + err.Error()
	}
	_ = slave.Close()
	// Preserve original O_NONBLOCK on stdin: clearing it unconditionally breaks
	// the surrounding line editor when PTY execution ends.
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, os.Stdin.Fd(), syscall.F_GETFL, 0)
	if errno != 0 {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
		return "fcntl error: " + errno.Error()
	}
	defer syscall.Syscall(syscall.SYS_FCNTL, os.Stdin.Fd(), syscall.F_SETFL, flags)
	if err := syscall.SetNonblock(int(os.Stdin.Fd()), true); err != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
		return "nonblock error: " + err.Error()
	}
	if err := syscall.SetNonblock(int(master.Fd()), true); err != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
		return "PTY nonblock error: " + err.Error()
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	buf := make([]byte, 8192)
	inputFD, masterFD := int(os.Stdin.Fd()), int(master.Fd())
	childExited := false
	var waitErr error
	eof := false
	interrupts := 0
	cancelRequested := false
	// Wait for the process *and* drain output. An EIO from the PTY master
	// indicates slave closure, not permission to abandon an unreaped child.
	for !childExited || !eof {
		select {
		case waitErr = <-done:
			childExited = true
			done = nil
		case <-ctx.Done():
			cancelRequested = true
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		case <-deadline.C:
			cancelRequested = true
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		default:
		}
		if childExited && eof {
			break
		}
		var rfds syscall.FdSet
		if masterFD >= 1024 || inputFD >= 1024 {
			cancelRequested = true
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			break
		}
		rfds.Bits[masterFD/64] |= 1 << uint(masterFD%64)
		if !childExited {
			rfds.Bits[inputFD/64] |= 1 << uint(inputFD%64)
		}
		tv := syscall.Timeval{Sec: 0, Usec: 50000}
		_, e := syscall.Select(maxFD(inputFD, masterFD)+1, &rfds, nil, nil, &tv)
		if e != nil && e != syscall.EINTR {
			cancelRequested = true
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			break
		}
		if rfds.Bits[masterFD/64]&(1<<uint(masterFD%64)) != 0 {
			n, e := syscall.Read(masterFD, buf)
			if n > 0 {
				_, _ = os.Stdout.Write(buf[:n])
			}
			if e == syscall.EIO || (n == 0 && e == nil) {
				eof = true
			}
		}
		if !childExited && rfds.Bits[inputFD/64]&(1<<uint(inputFD%64)) != 0 {
			n, e := syscall.Read(inputFD, buf)
			if n > 0 {
				start := 0
				for i, key := range buf[:n] {
					if key != 3 {
						continue
					}
					if i > start {
						_, _ = master.Write(buf[start:i])
					}
					interrupts++
					sig := syscall.SIGINT
					if interrupts == 2 {
						sig = syscall.SIGTERM
					}
					if interrupts >= 3 {
						sig = syscall.SIGKILL
					}
					_ = syscall.Kill(-cmd.Process.Pid, sig)
					start = i + 1
				}
				if start < n {
					_, _ = master.Write(buf[start:n])
				}
			}
			if e != nil && e != syscall.EAGAIN && e != syscall.EINTR {
				cancelRequested = true
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				break
			}
		}
		// Prevent spinning forever if other descendants retain the PTY open.
		if childExited && !eof {
			// A bounded drain will be handled below if output remains open.
			break
		}
	}
	if !childExited {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		waitErr = <-done
	}
	// Drain available output without blocking on descendants holding the slave.
	for i := 0; i < 16; i++ {
		n, e := syscall.Read(masterFD, buf)
		if n > 0 {
			_, _ = os.Stdout.Write(buf[:n])
		}
		if n == 0 || e != nil {
			break
		}
	}
	fmt.Println()
	code := 0
	if waitErr != nil {
		var ee *exec.ExitError
		if errors.As(waitErr, &ee) {
			code = ee.ExitCode()
		} else {
			return "execution error: " + waitErr.Error()
		}
	}
	if cancelRequested {
		return fmt.Sprintf("exit_code=%d\n[Interactive PTY session interrupted; transcript omitted]", code)
	}
	return fmt.Sprintf("exit_code=%d\n[Interactive PTY session completed; input and transcript omitted to protect credentials]", code)
}
func maxFD(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func getTermios(fd uintptr) (*syscall.Termios, error) {
	var t syscall.Termios
	err := ioctl(fd, syscall.TCGETS, uintptr(unsafe.Pointer(&t)))
	return &t, err
}
func setTermios(fd uintptr, t *syscall.Termios) error {
	return ioctl(fd, syscall.TCSETS, uintptr(unsafe.Pointer(t)))
}
func isTerminal(fd uintptr) bool { _, err := getTermios(fd); return err == nil }
