package main

import (
	"bufio"
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestApprovalEditorHandoffPTY(t *testing.T) {
	master, slave, err := openPTY()
	if err != nil {
		t.Skipf("PTY unavailable: %v", err)
	}
	defer master.Close()
	defer slave.Close()
	original, err := syscall.Dup(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = syscall.Dup2(original, 0); _ = syscall.Close(original); activeControl = nil }()
	if err := syscall.Dup2(int(slave.Fd()), 0); err != nil {
		t.Fatal(err)
	}
	control := newTaskControl(func() {})
	activeControl = control
	editorDone := make(chan error, 1)
	go func() { _, err := readInteractiveLine(bufio.NewReader(os.Stdin), nil, ""); editorDone <- err }()
	result := make(chan bool, 1)
	go func() { allow, _ := control.requestApproval(context.Background()); result <- allow }()
	select {
	case e := <-editorDone:
		if e != errApprovalPending {
			t.Fatalf("editor: %v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("editor did not yield")
	}
	select {
	case req := <-control.approvals:
		go func() { time.Sleep(75 * time.Millisecond); _, _ = master.Write([]byte("y\r")) }()
		allow := readApproval(context.Background())
		control.state.Store(0)
		req.response <- allow
	case <-time.After(time.Second):
		t.Fatal("approval request missing")
	}
	select {
	case allow := <-result:
		if !allow {
			t.Fatal("Y not approved")
		}
	case <-time.After(time.Second):
		t.Fatal("agent did not receive approval")
	}
}
