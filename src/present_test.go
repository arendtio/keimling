package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPresentAssistantEmitsStructuredMessage(t *testing.T) {
	var kinds []string
	var payloads []map[string]any
	a := &Agent{
		Emit: func(kind, text string) {
			kinds = append(kinds, kind)
			var m map[string]any
			if json.Unmarshal([]byte(text), &m) == nil {
				payloads = append(payloads, m)
			}
		},
	}
	a.presentAssistant("hello **world**", false, false)
	if len(kinds) != 1 || kinds[0] != "message" {
		t.Fatalf("kinds=%v", kinds)
	}
	if payloads[0]["role"] != "assistant" || payloads[0]["text"] != "hello **world**" {
		t.Fatalf("%v", payloads[0])
	}
	kinds = nil
	payloads = nil
	a.presentAssistant("chunk", true, false)
	if !payloads[0]["delta"].(bool) {
		t.Fatal("expected delta")
	}
}

func TestPresentToolStartAndResult(t *testing.T) {
	var kinds []string
	a := &Agent{
		Emit: func(kind, text string) {
			kinds = append(kinds, kind)
		},
	}
	a.presentToolStart("exec", `{"command":"echo hi"}`)
	a.presentToolResult("exec", "echo hi", "hi", "", true, false)
	if len(kinds) != 2 || kinds[0] != "tool" || kinds[1] != "tool" {
		t.Fatalf("%v", kinds)
	}
}

func TestPresentFinish(t *testing.T) {
	var payload map[string]any
	a := &Agent{
		Emit: func(kind, text string) {
			if kind == "message" {
				_ = json.Unmarshal([]byte(text), &payload)
			}
		},
	}
	a.presentFinish("done", "shipped")
	if payload["role"] != "assistant" {
		t.Fatalf("%v", payload)
	}
	text, _ := payload["text"].(string)
	if !strings.Contains(text, "Task complete") || !strings.Contains(text, "shipped") {
		t.Fatalf("%q", text)
	}
}
