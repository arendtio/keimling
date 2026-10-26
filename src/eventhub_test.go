package main

import (
	"testing"
	"time"
)

func TestEventHubBroadcast(t *testing.T) {
	h := newEventHub()
	ch := h.Subscribe(4)
	defer h.Unsubscribe(ch)
	h.Broadcast("line", map[string]string{"text": "hi"})
	select {
	case ev := <-ch:
		if ev.Type != "line" {
			t.Fatalf("type %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
