package main

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestStopKillsActiveCommand(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pgid := cmd.Process.Pid
	trackCommandGroup(pgid)
	t.Cleanup(func() { _ = syscall.Kill(-pgid, syscall.SIGKILL) })
	killTrackedCommands()
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("active command still running after stop")
	}
}

func TestStopKillsCommandLeftInBackground(t *testing.T) {
	cmd := exec.Command("bash", "-c", "sleep 60 & disown; exit")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pgid := cmd.Process.Pid
	trackCommandGroup(pgid)
	t.Cleanup(func() { _ = syscall.Kill(-pgid, syscall.SIGKILL) })
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	select {
	case <-wait:
	case <-time.After(2 * time.Second):
		t.Fatal("shell did not return")
	}
	if err := syscall.Kill(-pgid, 0); err != nil {
		t.Fatal("background command was not still running")
	}
	releaseCommandGroup(pgid)
	killTrackedCommands()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(-pgid, 0) != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("background command still running after stop")
}
