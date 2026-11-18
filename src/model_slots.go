package main

import (
	"context"
	"sync"
)

// modelSlots limits simultaneous calls to one endpoint. A local model shares
// one GPU; a cloud endpoint does not.
var modelSlots = struct {
	mu sync.Mutex
	ch map[string]chan struct{}
}{ch: map[string]chan struct{}{}}

func (c *Client) modelSlotCount() int {
	if c == nil || c.Parallel < 0 {
		return 0
	}
	if c.Parallel > 0 {
		n := c.Parallel
		if n > 8 {
			n = 8
		}
		return n
	}
	if isLoopbackURL(c.BaseURL) {
		return 1
	}
	return 0
}

func (c *Client) acquireModelSlot(ctx context.Context) (func(), error) {
	n := c.modelSlotCount()
	if n <= 0 {
		return func() {}, nil
	}
	key := ""
	if c != nil {
		key = c.BaseURL
	}
	modelSlots.mu.Lock()
	ch := modelSlots.ch[key]
	if ch == nil || cap(ch) != n {
		ch = make(chan struct{}, n)
		modelSlots.ch[key] = ch
	}
	modelSlots.mu.Unlock()
	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
