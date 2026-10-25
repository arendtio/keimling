package main

import (
	"context"
	"strings"
	"testing"
)

func TestResponseStreamCompleted(t *testing.T) {
	s := "event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"Hello \"}\n\n" +
		"data: {\"type\":\"response.output_text.delta\",\"delta\":\"world\"}\n\n" +
		"event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"Hello world\"}]}],\"usage\":{\"input_tokens\":100,\"output_tokens\":5}}}\n\n"
	c := NewClient("test")
	var received strings.Builder
	c.StreamText = func(t string) { received.WriteString(t) }
	got, err := c.readResponseStream(context.Background(), strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	if !got.StreamedText || got.Status != "completed" || got.Usage.OutputTokens != 5 || received.String() != "Hello world" {
		t.Fatalf("response %#v, text %q", got, received.String())
	}
}
func TestResponseStreamRejectsTruncation(t *testing.T) {
	c := NewClient("test")
	_, err := c.readResponseStream(context.Background(), strings.NewReader("data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n"))
	if err == nil {
		t.Fatal("expected stream incomplete error")
	}
}
func TestResponseStreamRejectsError(t *testing.T) {
	c := NewClient("test")
	_, err := c.readResponseStream(context.Background(), strings.NewReader("event: error\ndata: {\"type\":\"error\",\"message\":\"failure\"}\n\n"))
	if err == nil || !strings.Contains(err.Error(), "failure") {
		t.Fatalf("bad error: %v", err)
	}
}
