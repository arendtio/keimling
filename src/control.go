package main

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
)

var errApprovalPending = errors.New("approval waiting for terminal UI")
var errUIHandoff = errors.New("terminal handed to foreground process")
var errAgentIdle = errors.New("agent finished")

type controlEvent struct{ kind, text string }
type approvalRequest struct{ response chan bool }
type taskControl struct {
	events    chan controlEvent
	approvals chan approvalRequest
	done      chan error
	finished  chan struct{}
	// state=1: agent needs exclusive stdin; state=2: UI has released it.
	state  atomic.Int32
	resume chan struct{}
	cancel context.CancelFunc
}

func newTaskControl(cancel context.CancelFunc) *taskControl {
	return &taskControl{events: make(chan controlEvent, 64), approvals: make(chan approvalRequest, 1), done: make(chan error, 1), finished: make(chan struct{}), resume: make(chan struct{}, 1), cancel: cancel}
}
func (c *taskControl) takeTerminal() {
	if !isTerminal(0) {
		return
	}
	c.state.Store(1)
	// The UI reader must acknowledge relinquishing stdin before PTY/approval reads it.
	<-c.resume
}
func (c *taskControl) releaseTerminal() { c.state.Store(0) }
func (a *Agent) handleControls(ctx context.Context) error {
	if a.Control == nil {
		return ctx.Err()
	}
	for {
		select {
		case ev := <-a.Control.events:
			switch ev.kind {
			case "steer":
				a.PendingSteering = true
				if strings.TrimSpace(ev.text) != "" {
					if err := a.add(map[string]any{"role": "user", "content": "[User steering during ongoing task] " + ev.text}); err != nil {
						return err
					}
				}
			case "compact":
				level, instruction := "hard", ev.text
				if instruction == "soft" {
					level = "soft"
					instruction = ""
				}
				if err := a.compactContext(ctx, level, instruction); err != nil {
					return err
				}
			}
		default:
			return ctx.Err()
		}
	}
}

func (c *taskControl) requestApproval(ctx context.Context) (bool, error) {
	req := approvalRequest{response: make(chan bool, 1)}
	c.state.Store(3) // UI reader yields; it does not hand stdin to the agent.
	select {
	case c.approvals <- req:
	case <-ctx.Done():
		c.state.Store(0)
		return false, ctx.Err()
	}
	select {
	case allow := <-req.response:
		return allow, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
