//go:build linux || darwin

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// interactiveExec gives the foreground child exclusive use of the terminal.
// The child owns its PTY slave, while this process forwards bytes via its master.
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
	if ws, werr := unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ); werr == nil {
		_ = unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, ws)
	}
	cmd := shellCommand(shell, command)
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	raw, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "terminal setup error: " + err.Error()
	}
	defer term.Restore(int(os.Stdin.Fd()), raw)
	fmt.Println("[Interactive terminal attached; Ctrl+C interrupts child]")
	if err := cmd.Start(); err != nil {
		return "start error: " + err.Error()
	}
	trackCommandGroup(cmd.Process.Pid)
	defer releaseCommandGroup(cmd.Process.Pid)
	_ = slave.Close()
	flags, err := unix.FcntlInt(os.Stdin.Fd(), unix.F_GETFL, 0)
	if err != nil {
		killProcessGroup(cmd.Process.Pid)
		_ = cmd.Wait()
		return "fcntl error: " + err.Error()
	}
	defer unix.FcntlInt(os.Stdin.Fd(), unix.F_SETFL, flags)
	if err := unix.SetNonblock(int(os.Stdin.Fd()), true); err != nil {
		killProcessGroup(cmd.Process.Pid)
		_ = cmd.Wait()
		return "nonblock error: " + err.Error()
	}
	if err := unix.SetNonblock(int(master.Fd()), true); err != nil {
		killProcessGroup(cmd.Process.Pid)
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
	for !childExited || !eof {
		select {
		case waitErr = <-done:
			childExited = true
			done = nil
		case <-ctx.Done():
			cancelRequested = true
			killProcessGroup(cmd.Process.Pid)
		case <-deadline.C:
			cancelRequested = true
			killProcessGroup(cmd.Process.Pid)
		default:
		}
		if childExited && eof {
			break
		}
		var rfds unix.FdSet
		if masterFD >= 1024 || inputFD >= 1024 {
			cancelRequested = true
			killProcessGroup(cmd.Process.Pid)
			break
		}
		rfds.Set(masterFD)
		if !childExited {
			rfds.Set(inputFD)
		}
		tv := unix.NsecToTimeval(50 * time.Millisecond.Nanoseconds())
		_, e := unix.Select(maxFD(inputFD, masterFD)+1, &rfds, nil, nil, &tv)
		if e != nil && e != syscall.EINTR {
			cancelRequested = true
			killProcessGroup(cmd.Process.Pid)
			break
		}
		if rfds.IsSet(masterFD) {
			n, e := syscall.Read(masterFD, buf)
			if n > 0 {
				_, _ = os.Stdout.Write(buf[:n])
			}
			if e == syscall.EIO || (n == 0 && e == nil) {
				eof = true
			}
		}
		if !childExited && rfds.IsSet(inputFD) {
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
					killProcessSignal(cmd.Process.Pid, sig)
					start = i + 1
				}
				if start < n {
					_, _ = master.Write(buf[start:n])
				}
			}
			if e != nil && e != syscall.EAGAIN && e != syscall.EINTR {
				cancelRequested = true
				killProcessGroup(cmd.Process.Pid)
				break
			}
		}
		if childExited && !eof {
			break
		}
	}
	if !childExited {
		killProcessGroup(cmd.Process.Pid)
		waitErr = <-done
	}
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

func killProcessSignal(pid int, sig syscall.Signal) {
	_ = syscall.Kill(-pid, sig)
}
