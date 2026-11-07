package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	dataModelLimit     = 4 << 10
	dataStoreByteLimit = 32 << 20
	dataStoreMaxItems  = 64
	dataDefaultTimeout = 30 * time.Second
)

// dataTimeout turns a requested millisecond count into a deadline.
// Zero and negative values use the default. A count past the clock type waits as long as that type allows.
func dataTimeout(ms int64) time.Duration {
	if ms <= 0 {
		return dataDefaultTimeout
	}
	maxMS := int64(math.MaxInt64 / int64(time.Millisecond))
	if ms > maxMS {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(ms) * time.Millisecond
}

// dataError is a failure the model can correct.
type dataError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Example    string `json:"example,omitempty"`
	Offset     int    `json:"offset,omitempty"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
	Expected   string `json:"expected,omitempty"`
	Ref        string `json:"ref,omitempty"`
}

func (e *dataError) Error() string { return e.Message }

func dataErr(code, msg string) *dataError { return &dataError{Code: code, Message: msg} }

func dataErrorJSON(err error) string {
	de, ok := err.(*dataError)
	if !ok {
		de = dataErr("ERROR", err.Error())
	}
	b, _ := json.Marshal(map[string]any{"error": de})
	return string(b)
}

type dataItem struct {
	id      string
	kind    string
	value   any
	preview string
	schema  string
	count   int
	bytes   int
}

func (it *dataItem) envelope(inline bool) map[string]any {
	m := map[string]any{"ref": it.id, "type": it.kind}
	if inline {
		m["value"] = it.value
		return m
	}
	m["preview"] = it.preview
	if it.schema != "" {
		m["schema_hint"] = it.schema
	}
	if it.count != 0 {
		m["count"] = it.count
	}
	return m
}

// dataStore keeps tool results for one agent. Refs are not paths.
type dataStore struct {
	mu       sync.Mutex
	n        int
	order    []string
	items    map[string]*dataItem
	used     int
	maxItems int
	maxBytes int
}

func newDataStore() *dataStore {
	return &dataStore{items: map[string]*dataItem{}, maxItems: dataStoreMaxItems, maxBytes: dataStoreByteLimit}
}

func (s *dataStore) put(kind string, value any) (*dataItem, error) {
	canon, n, err := canonValue(value)
	if err != nil {
		return nil, dataErr("TYPE_ERROR", err.Error())
	}
	if kind == "" {
		kind = kindOf(canon)
	}
	preview, schema, count := describeValue(canon)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items == nil {
		s.items = map[string]*dataItem{}
	}
	s.n++
	it := &dataItem{id: fmt.Sprintf("r%d", s.n), kind: kind, value: canon, preview: preview, schema: schema, count: count, bytes: n}
	s.items[it.id] = it
	s.order = append(s.order, it.id)
	s.used += n
	s.evict()
	if _, ok := s.items[it.id]; !ok {
		return nil, dataErr("RESULT_TOO_LARGE", "result does not fit in the result store")
	}
	return it, nil
}

func (s *dataStore) evict() {
	for len(s.order) > s.maxItems || (s.used > s.maxBytes && len(s.order) > 1) {
		id := s.order[0]
		s.order = s.order[1:]
		if it := s.items[id]; it != nil {
			s.used -= it.bytes
			delete(s.items, id)
		}
	}
}

func (s *dataStore) get(id string) (*dataItem, error) {
	id = strings.TrimSpace(id)
	if id == "" || !strings.HasPrefix(id, "r") {
		return nil, dataErr("INVALID_REF", "missing result reference")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	it := s.items[id]
	if it == nil {
		return nil, &dataError{Code: "INVALID_REF", Message: "unknown or expired result reference " + id, Suggestion: "Use a ref from a tool result in this conversation."}
	}
	return it, nil
}

// hold stores a tool result. Large displays become a reference preview.
// Small displays keep their text and name the reference.
func (s *dataStore) hold(kind string, value any, display string) (string, error) {
	it, err := s.put(kind, value)
	if err != nil {
		return "", err
	}
	if len(display) > dataModelLimit {
		b, _ := json.Marshal(it.envelope(false))
		return string(b), nil
	}
	if strings.TrimSpace(display) == "" {
		b, _ := json.Marshal(it.envelope(true))
		return string(b), nil
	}
	return display + "\nref: " + it.id, nil
}

func (s *dataStore) present(value any, mode string) (string, error) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "auto"
	}
	if mode != "auto" && mode != "inline" && mode != "ref" {
		return "", dataErr("TYPE_ERROR", "output_mode must be auto, inline, or ref")
	}
	it, err := s.put(kindOf(value), value)
	if err != nil {
		return "", err
	}
	inline := mode == "inline" || (mode == "auto" && it.bytes <= dataModelLimit)
	if mode == "inline" && it.bytes > dataModelLimit {
		return "", &dataError{Code: "RESULT_TOO_LARGE", Message: fmt.Sprintf("result is %d bytes and the inline limit is %d", it.bytes, dataModelLimit), Ref: it.id, Suggestion: "Use output_mode ref, or filter the result further."}
	}
	if mode == "ref" {
		inline = false
	}
	b, err := json.Marshal(it.envelope(inline))
	if err != nil {
		return "", dataErr("TYPE_ERROR", err.Error())
	}
	return string(b), nil
}

func canonValue(v any) (any, int, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, 0, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, 0, err
	}
	return foldJSON(out), len(b), nil
}

func foldJSON(v any) any {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i
		}
		f, err := x.Float64()
		if err != nil {
			return x.String()
		}
		return f
	case map[string]any:
		for k, val := range x {
			x[k] = foldJSON(val)
		}
		return x
	case []any:
		for i, val := range x {
			x[i] = foldJSON(val)
		}
		return x
	default:
		return v
	}
}

func kindOf(v any) string {
	switch v.(type) {
	case map[string]any, []any:
		return "json"
	case string:
		return "text"
	default:
		return "json"
	}
}

func describeValue(v any) (preview, schema string, count int) {
	switch x := v.(type) {
	case map[string]any:
		schema = schemaHint(x)
		var arrayKey string
		arrays := 0
		n := 0
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if a, ok := x[k].([]any); ok {
				arrays++
				arrayKey = k
				n = len(a)
			}
		}
		if arrays == 1 {
			return fmt.Sprintf("%d %s", n, arrayKey), schema, n
		}
		return fmt.Sprintf("%d fields", len(x)), schema, len(x)
	case []any:
		return fmt.Sprintf("%d records", len(x)), "array", len(x)
	case string:
		s := x
		if len(s) > 80 {
			s = s[:80]
		}
		return s, "string", 1
	default:
		return fmt.Sprint(v), kindOf(v), 1
	}
}

func schemaHint(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+": "+typeName(m[k]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func typeName(v any) string {
	switch v.(type) {
	case []any:
		return "array"
	case map[string]any:
		return "object"
	case string:
		return "string"
	case bool:
		return "bool"
	case int, int64, float64, json.Number:
		return "number"
	case nil:
		return "null"
	default:
		return "value"
	}
}

func normalizeExec(result string) any {
	code := 0
	body := result
	if strings.HasPrefix(result, "exit_code=") {
		line, rest, found := strings.Cut(result, "\n")
		fmt.Sscanf(strings.TrimPrefix(line, "exit_code="), "%d", &code)
		if found {
			body = rest
		} else {
			body = ""
		}
	}
	return map[string]any{"stdout": body, "stderr": "", "exit_code": code}
}

func locAt(s string, off int) (line, col int) {
	if off < 0 {
		off = 0
	}
	if off > len(s) {
		off = len(s)
	}
	line, col = 1, 1
	for i := 0; i < off; i++ {
		if s[i] == '\n' {
			line++
			col = 1
			continue
		}
		col++
	}
	return line, col
}
