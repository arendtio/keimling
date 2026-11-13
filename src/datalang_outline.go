package main

import (
	"encoding/csv"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// outlineAny is the shape of a value. A string that is JSON, XML, or CSV is
// read first. Values are not copied into the shape, and the text stops at the
// inline limit.
func outlineAny(v any) string {
	if s, ok := v.(string); ok {
		if shape, ok := structuredOutline(s); ok {
			return shape
		}
		return textOutline(s)
	}
	return outlineValue(v)
}

// structuredOutline reports the shape of JSON, XML, or CSV text.
func structuredOutline(s string) (string, bool) {
	t := strings.TrimSpace(s)
	if t == "" {
		return "", false
	}
	switch t[0] {
	case '{', '[':
		if v, err := parseJSONText(s); err == nil {
			return outlineValue(v), true
		}
	case '<':
		if v, err := parseXMLText(s); err == nil {
			return outlineValue(v), true
		}
	}
	if looksLikeCSV(s) {
		if v, err := parseCSVText(s); err == nil {
			return outlineValue(v), true
		}
	}
	return "", false
}

func looksLikeCSV(s string) bool {
	r := csv.NewReader(strings.NewReader(s))
	r.FieldsPerRecord = -1
	head, err := r.Read()
	if err != nil || len(head) < 2 {
		return false
	}
	next, err := r.Read()
	if err != nil || len(next) != len(head) {
		return false
	}
	return true
}

func textOutline(s string) string {
	if !utf8.ValidString(s) {
		return binaryOutline(len(s), []byte(s))
	}
	n := strings.Count(s, "\n")
	if s != "" && !strings.HasSuffix(s, "\n") {
		n++
	}
	if s == "" {
		n = 0
	}
	return fmt.Sprintf("text lines=%d chars=%d", n, len(s))
}

func binaryOutline(n int, sample []byte) string {
	text := printableRuns(sample, 240)
	if text == "" {
		text = "(no readable text)"
	}
	return fmt.Sprintf("binary bytes=%d\nsample:\n%s", n, text)
}

func printableRuns(b []byte, limit int) string {
	var runs []string
	var cur []byte
	flush := func() {
		if len(cur) >= 4 {
			runs = append(runs, string(cur))
		}
		cur = cur[:0]
	}
	for _, c := range b {
		if c >= 32 && c < 127 {
			cur = append(cur, c)
			continue
		}
		flush()
	}
	flush()
	text := strings.Join(runs, "\n")
	if len(text) > limit {
		text = text[:limit]
	}
	return text
}

type outl struct {
	b   strings.Builder
	lim int
	cut bool
}

func outlineValue(v any) string {
	o := &outl{lim: dataModelLimit}
	writeOutline(o, v, 0)
	return strings.TrimRight(o.b.String(), "\n")
}

func (o *outl) line(depth int, s string) {
	if o.cut {
		return
	}
	indent := strings.Repeat("  ", depth)
	row := indent + s + "\n"
	if o.b.Len()+len(row) > o.lim {
		mark := indent + "…\n"
		if o.b.Len()+len(mark) <= o.lim {
			o.b.WriteString(mark)
		}
		o.cut = true
		return
	}
	o.b.WriteString(row)
}

func writeOutline(o *outl, v any, depth int) {
	if o.cut {
		return
	}
	if depth > 6 {
		o.line(depth, typePhrase(v))
		return
	}
	switch x := v.(type) {
	case map[string]any:
		if isXMLText(x) {
			o.line(depth, "text")
			return
		}
		if isXMLNode(x) {
			writeXML(o, x, depth)
			return
		}
		o.line(depth, fmt.Sprintf("object n=%d", len(x)))
		writeObjectFields(o, x, depth+1)
	case []any:
		if len(x) == 0 {
			o.line(depth, "array n=0")
			return
		}
		o.line(depth, fmt.Sprintf("array n=%d of %s", len(x), typePhrase(x[0])))
		if unfolds(x[0]) {
			writeInner(o, x[0], depth+1)
		}
	default:
		o.line(depth, typePhrase(v))
	}
}

func writeInner(o *outl, v any, depth int) {
	switch x := v.(type) {
	case map[string]any:
		if isXMLNode(x) {
			writeXMLChildren(o, x, depth)
			return
		}
		writeObjectFields(o, x, depth)
	case []any:
		if len(x) > 0 && unfolds(x[0]) {
			writeInner(o, x[0], depth)
		}
	}
}

func writeObjectFields(o *outl, m map[string]any, depth int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		o.line(depth, k+": "+typePhrase(m[k]))
		if unfolds(m[k]) {
			writeInner(o, m[k], depth+1)
		}
	}
}

func writeXML(o *outl, m map[string]any, depth int) {
	tag, _ := m["tag"].(string)
	attrs := attrNames(m["attrs"])
	line := tag
	if attrs != "" {
		line += " attrs=" + attrs
	}
	o.line(depth, line)
	writeXMLChildren(o, m, depth+1)
}

func writeXMLChildren(o *outl, m map[string]any, depth int) {
	children, _ := m["children"].([]any)
	type group struct {
		tag  string
		n    int
		node map[string]any
		text bool
	}
	var groups []group
	for _, child := range children {
		node, ok := child.(map[string]any)
		if !ok {
			continue
		}
		if isXMLText(node) {
			if len(groups) > 0 && groups[len(groups)-1].text {
				groups[len(groups)-1].n++
				continue
			}
			groups = append(groups, group{n: 1, text: true})
			continue
		}
		tag, _ := node["tag"].(string)
		if tag == "" {
			continue
		}
		if len(groups) > 0 && !groups[len(groups)-1].text && groups[len(groups)-1].tag == tag {
			groups[len(groups)-1].n++
			continue
		}
		groups = append(groups, group{tag: tag, n: 1, node: node})
	}
	for _, g := range groups {
		if g.text {
			if g.n > 1 {
				o.line(depth, fmt.Sprintf("text n=%d", g.n))
			} else {
				o.line(depth, "text")
			}
			continue
		}
		line := g.tag
		if g.n > 1 {
			line += fmt.Sprintf(" n=%d", g.n)
		}
		if attrs := attrNames(g.node["attrs"]); attrs != "" {
			line += " attrs=" + attrs
		}
		o.line(depth, line)
		writeXMLChildren(o, g.node, depth+1)
	}
}

func isXMLNode(m map[string]any) bool {
	_, ok := m["tag"].(string)
	_, ok2 := m["children"].([]any)
	return ok && ok2
}

func isXMLText(m map[string]any) bool {
	_, ok := m["text"].(string)
	return ok && len(m) == 1
}

func attrNames(v any) string {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return ""
	}
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

func unfolds(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		return !isXMLText(x)
	case []any:
		return len(x) > 0 && unfolds(x[0])
	default:
		return false
	}
}

func typePhrase(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case string:
		if len(x) > 80 {
			return fmt.Sprintf("string n=%d", len(x))
		}
		return "string"
	case int, int64, float64:
		return "number"
	case map[string]any:
		if isXMLText(x) {
			return "text"
		}
		if isXMLNode(x) {
			tag, _ := x["tag"].(string)
			return tag
		}
		return fmt.Sprintf("object n=%d", len(x))
	case []any:
		if len(x) == 0 {
			return "array n=0"
		}
		return fmt.Sprintf("array n=%d of %s", len(x), typePhrase(x[0]))
	default:
		if s, ok := v.(fmt.Stringer); ok && strings.IndexFunc(s.String(), unicode.IsSpace) < 0 {
			return "number"
		}
		return "value"
	}
}
