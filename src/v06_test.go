package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestInsertRuneMiddle(t *testing.T) {
	got := insertRune([]rune("Hllo"), 1, 'e')
	if string(got) != "Hello" {
		t.Fatalf("%q", string(got))
	}
	got = insertRune(got, 5, '✓')
	if string(got) != "Hello✓" {
		t.Fatalf("%q", string(got))
	}
}
func TestExecJSONArgumentsDecoded(t *testing.T) {
	var arg struct {
		Command string `json:"command"`
	}
	if e := json.Unmarshal([]byte(`{"command":"printf 'hello\\nworld\\n'"}`), &arg); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(arg.Command, "printf 'hello\\nworld\\n'") {
		t.Fatalf("unexpected: %q", arg.Command)
	}
}
func TestPlanDefinitions(t *testing.T) {
	names := map[string]bool{}
	for _, x := range definitions(false) {
		name, _ := x["name"].(string)
		names[name] = true
	}
	if !names["update_plan"] || !names["finish_task"] {
		t.Fatalf("missing tools: %v", names)
	}
}
