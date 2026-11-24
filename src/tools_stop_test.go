//go:build linux || darwin

package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
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
		// A killed child whose parent is gone becomes a zombie until the
		// reaper collects it. A zombie keeps the process group visible but
		// is not a running command.
		if !processGroupHasLiveMember(pgid) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("background command still running after stop")
}

// processGroupHasLiveMember reports whether any process in the group is more
// than a zombie. Reads /proc, so it is Linux-only; other systems fall back to
// the plain group check.
func processGroupHasLiveMember(pgid int) bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return true
	}
	for _, e := range entries {
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		b, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue
		}
		// stat: pid (comm) state ppid pgrp ...
		fields := strings.Fields(string(b))
		if len(fields) < 5 {
			continue
		}
		if fields[2] == "Z" {
			continue
		}
		if fields[4] == strconv.Itoa(pgid) {
			return true
		}
	}
	return false
}
