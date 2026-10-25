package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPortableHistoryPreservesVisibleWork(t *testing.T) {
	input := []json.RawMessage{
		json.RawMessage(`{"type":"reasoning","id":"r1","encrypted_content":"secret"}`),
		json.RawMessage(`{"type":"message","role":"assistant","content":[{"type":"redacted_thinking","data":"secret"},{"type":"output_text","text":"Useful conclusion"}]}`),
		json.RawMessage(`{"type":"function_call","call_id":"c1","name":"exec","arguments":"{}"}`),
		json.RawMessage(`{"type":"function_call_output","call_id":"c1","output":"ok"}`),
		json.RawMessage(`{"role":"user","content":"Continue"}`),
	}
	out, removed, err := portableHistoryView(input)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 || len(out) != 4 {
		t.Fatalf("removed=%d len=%d", removed, len(out))
	}
	combined := string(bytesJoin(out))
	for _, required := range []string{"Useful conclusion", "function_call_output", "Continue"} {
		if !strings.Contains(combined, required) {
			t.Errorf("missing %q", required)
		}
	}
	if strings.Contains(combined, "redacted_thinking") || strings.Contains(combined, "encrypted_content") {
		t.Fatal("reasoning leaked")
	}
	if !strings.Contains(string(input[0]), "encrypted_content") {
		t.Fatal("original modified")
	}
}
func bytesJoin(items []json.RawMessage) []byte {
	var out []byte
	for _, b := range items {
		out = append(out, b...)
	}
	return out
}
func TestSlashCompletion(t *testing.T) {
	cases := []struct {
		in      string
		pos     int
		want    string
		matches int
	}{
		{"/stat", 5, "/status", 1},
		{"/continue-n", 11, "/continue-new-model", 1},
		{"/con", 4, "/continue", 2},
		{"/mode y", 7, "/mode y", 0},
		{"hi /sta", 7, "hi /sta", 0},
	}
	for _, tc := range cases {
		got, _, matches := completeSlash(tc.in, tc.pos)
		if got != tc.want || len(matches) != tc.matches {
			t.Errorf("%q got %q (%v), want %q (%d)", tc.in, got, matches, tc.want, tc.matches)
		}
	}
}
