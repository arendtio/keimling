package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// interactiveExec attaches the command to the current console. Windows has no
// POSIX PTY here; the child shares the terminal the user is typing in.
func interactiveExec(ctx context.Context, shell, command string, timeout time.Duration, extraEnv ...string) string {
	if !isTerminal(os.Stdin.Fd()) {
		return "interactive execution unavailable: stdin is not a terminal"
	}
	cmd := shellCommand(shell, command)
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	setProcessGroup(cmd)
	fmt.Println("[Interactive console attached]")
	if err := cmd.Start(); err != nil {
		return "start error: " + err.Error()
	}
	trackCommandGroup(cmd.Process.Pid)
	defer releaseCommandGroup(cmd.Process.Pid)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var waitErr error
	cancelRequested := false
	select {
	case waitErr = <-done:
	case <-ctx.Done():
		cancelRequested = true
		killProcessGroup(cmd.Process.Pid)
		waitErr = <-done
	case <-timer.C:
		cancelRequested = true
		killProcessGroup(cmd.Process.Pid)
		waitErr = <-done
	}
	code := 0
	if waitErr != nil {
		var ee *exec.ExitError
		if errors.As(waitErr, &ee) {
			code = ee.ExitCode()
		} else if !cancelRequested {
			return "execution error: " + waitErr.Error()
		}
	}
	if cancelRequested {
		return fmt.Sprintf("exit_code=%d\n[Interactive console session interrupted]", code)
	}
	return fmt.Sprintf("exit_code=%d\n[Interactive console session completed; input and transcript omitted to protect credentials]", code)
}
