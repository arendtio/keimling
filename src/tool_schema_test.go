package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// checkStrictSchema walks one JSON schema and reports every object whose
// properties are not all listed in required, or that allows extra properties.
func checkStrictSchema(path string, schema any, out *[]string) {
	obj, ok := schema.(map[string]any)
	if !ok {
		return
	}
	if props, ok := obj["properties"].(map[string]any); ok {
		names := make([]string, 0, len(props))
		for name := range props {
			names = append(names, name)
		}
		sort.Strings(names)
		required := map[string]bool{}
		if list, ok := obj["required"].([]string); ok {
			for _, name := range list {
				required[name] = true
			}
		}
		for _, name := range names {
			if !required[name] {
				*out = append(*out, fmt.Sprintf("%s.%s is optional", path, name))
			}
		}
		if allowed, ok := obj["additionalProperties"]; !ok || allowed != false {
			*out = append(*out, path+" allows additionalProperties")
		}
		for _, name := range names {
			checkStrictSchema(path+"."+name, props[name], out)
		}
	}
	if items, ok := obj["items"]; ok {
		checkStrictSchema(path+"[]", items, out)
	}
}

func TestToolDefinitionsAreStrictComplete(t *testing.T) {
	var all []string
	for _, tool := range definitions(true) {
		name, _ := tool["name"].(string)
		if strict, ok := tool["strict"].(bool); !ok || !strict {
			all = append(all, name+" is not strict")
			continue
		}
		var problems []string
		checkStrictSchema(name, tool["parameters"], &problems)
		all = append(all, problems...)
	}
	if len(all) > 0 {
		t.Fatalf("tool schemas are not strict-complete: %s", strings.Join(all, "; "))
	}
}

// A strict JSON schema carries `required` as an array. Encoding a nil slice
// writes null, which a proxy rejects even for a tool without parameters.
func TestToolSchemasMarshalRequiredAsArray(t *testing.T) {
	for _, tool := range definitions(true) {
		name, _ := tool["name"].(string)
		b, err := json.Marshal(tool)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Parameters struct {
				Required json.RawMessage `json:"required"`
				Type     string          `json:"type"`
			} `json:"parameters"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Fatal(err)
		}
		req := strings.TrimSpace(string(doc.Parameters.Required))
		if doc.Parameters.Type != "object" || !strings.HasPrefix(req, "[") {
			t.Fatalf("%s type=%s required=%s", name, doc.Parameters.Type, req)
		}
	}
}

// The camera tool takes no parameter. It still needs an object schema with an
// empty properties map and an empty required array.
func TestCameraToolSchemaHasNoParameters(t *testing.T) {
	var found bool
	for _, tool := range cameraToolDefs() {
		if tool["name"] != "camera_snapshot" {
			continue
		}
		found = true
		params, _ := tool["parameters"].(map[string]any)
		props, _ := params["properties"].(map[string]any)
		required, ok := params["required"].([]string)
		if !ok || len(props) != 0 || len(required) != 0 {
			t.Fatalf("parameters=%v", params)
		}
		if strict, _ := tool["strict"].(bool); !strict {
			t.Fatal("camera_snapshot is not strict")
		}
	}
	if !found {
		t.Fatal("camera_snapshot missing")
	}
}
