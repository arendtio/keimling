package main

import (
	"encoding/json"
	"sync"
)

type hubEvent struct {
	Type string
	Data any
}

type eventHub struct {
	mu   sync.Mutex
	subs map[chan hubEvent]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{subs: make(map[chan hubEvent]struct{})}
}

func (h *eventHub) Subscribe(buf int) chan hubEvent {
	if buf < 1 {
		buf = 16
	}
	ch := make(chan hubEvent, buf)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *eventHub) Unsubscribe(ch chan hubEvent) {
	h.mu.Lock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
	h.mu.Unlock()
}

func (h *eventHub) Broadcast(typ string, data any) {
	ev := hubEvent{Type: typ, Data: data}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- ev:
		default:
			// Drop if subscriber is slow; keep producers non-blocking.
		}
	}
}

func (h *eventHub) EmitJSON(typ string, data any) {
	h.Broadcast(typ, data)
}

func marshalEventData(data any) []byte {
	b, err := json.Marshal(data)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}
