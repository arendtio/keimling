package main

import (
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
