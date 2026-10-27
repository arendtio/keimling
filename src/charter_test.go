package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCharterRenderAndPersist(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "session.jsonl")
	c := &Charter{
		Version:  charterVersion,
		Goal:     "Ship charter context",
		Plan:     []CharterStep{{Step: "schema", Status: "done"}, {Step: "wire", Status: "in_progress"}},
		Position: 2,
		Findings: []string{"persist beside session"},
	}
	if err := saveCharter(session, c); err != nil {
		t.Fatal(err)
	}
	got, err := loadCharter(session)
	if err != nil {
		t.Fatal(err)
	}
	md := got.RenderMarkdown()
	if !strings.Contains(md, "Ship charter context") || !strings.Contains(md, "→ [in_progress] wire") {
		t.Fatalf("markdown: %s", md)
	}
	if _, err := os.Stat(charterPath(session)); err != nil {
		t.Fatal(err)
	}
}

func TestParseCharterJSONFence(t *testing.T) {
	raw := "```json\n{\"goal\":\"g\",\"findings\":[\"a\"],\"plan\":[],\"open_questions\":[],\"constraints\":[],\"recent\":[]}\n```"
	c, err := parseCharterJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if c.Goal != "g" || len(c.Findings) != 1 {
		t.Fatalf("%+v", c)
	}
}

func TestModelInputSkipsArchivalLeaves(t *testing.T) {
	a := &Agent{
		Cfg:     Config{ContextLimit: 100000},
		Charter: &Charter{Version: 1, Goal: "goal"},
		History: []json.RawMessage{
			[]byte(`{"role":"user","content":"<compacted_conversation_memory>\nold\n</compacted_conversation_memory>"}`),
			[]byte(`{"role":"user","content":"hello"}`),
		},
	}
	in := a.modelInput()
	joined := ""
	for _, raw := range in {
		joined += string(raw)
	}
	if strings.Contains(joined, "compacted_conversation_memory") {
		t.Fatal("archival leaf leaked into model input")
	}
	if !strings.Contains(joined, "context_budget") || !strings.Contains(joined, "project_charter") {
		t.Fatal(joined)
	}
	if !strings.Contains(joined, "hello") {
		t.Fatal("recent user turn missing")
	}
}
