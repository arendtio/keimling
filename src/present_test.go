package main

import (
	"context"
	"encoding/json"
	"path/filepath"
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
	a.presentToolResult("exec", "echo hi", `{"command":"echo hi"}`, "hi", "", true, false)
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

func TestToolResultPublishesContextGrowth(t *testing.T) {
	j, err := OpenJournal(filepath.Join(t.TempDir(), "session.jsonl"), false)
	if err != nil {
		t.Fatal(err)
	}
	var pcts []float64
	a := &Agent{
		Journal: j,
		History: j.History,
		Cfg:     Config{ContextLimit: 100000},
		Emit: func(kind, text string) {
			if kind != "growth" {
				return
			}
			var m map[string]any
			if json.Unmarshal([]byte(text), &m) != nil {
				t.Fatalf("growth %s", text)
			}
			pct, ok := m["used_pct"].(float64)
			if !ok {
				t.Fatalf("growth %v", m)
			}
			pcts = append(pcts, pct)
		},
	}
	before := a.contextBudget().UsedPct
	body := strings.Repeat("word ", 8000)
	if err := a.recordToolResult(context.Background(), Item{Type: "function_call", Name: "read_file", CallID: "c1", Arguments: `{"path":"a"}`}, body, "", true); err != nil {
		t.Fatal(err)
	}
	if len(pcts) != 1 {
		t.Fatalf("growth events %v", pcts)
	}
	if pcts[0] <= before {
		t.Fatalf("before %.2f after %.2f", before, pcts[0])
	}
}
