package main

import (
	"encoding/json"
	"regexp"
	"testing"
)

// Every tool schema a strict proxy sees must arrive as valid JSON Schema:
// no null where an array or object is expected.
func TestNoNullInToolSchemas(t *testing.T) {
	b, err := json.Marshal(definitions(true))
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`"([A-Za-z_]+)":null`)
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		t.Errorf("null field: %s", m[1])
	}
}
