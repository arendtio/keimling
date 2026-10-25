package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRetryableNetworkError(t *testing.T) {
	for _, e := range []error{syscall.ENETUNREACH, syscall.EHOSTUNREACH, syscall.ECONNREFUSED, syscall.ECONNRESET, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ENETUNREACH}, &net.DNSError{Err: "temporary", IsTemporary: true}} {
		if !retryableNetworkError(fmt.Errorf("wrapped: %w", e)) {
			t.Fatalf("not retried: %v", e)
		}
	}
	for _, e := range []error{context.Canceled, errors.New("bad auth"), syscall.EINVAL} {
		if retryableNetworkError(e) {
			t.Fatalf("incorrect retry: %v", e)
		}
	}
}
func TestRespondRetriesNetworkUnreachable(t *testing.T) {
	calls := 0
	c := NewClient("test")
	c.BaseURL = "https://test.invalid/v1"
	c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ENETUNREACH}
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"status":"completed","output":[]}`))}, nil
	})}
	_, err := c.Respond(context.Background(), "test-model", "", nil, false)
	if err != nil || calls != 2 {
		t.Fatalf("calls=%d error=%v", calls, err)
	}
}
