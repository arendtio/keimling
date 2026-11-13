package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// dataFile is a file DataLang scans instead of loading. It is used when the
// file is too large to pass as text.
type dataFile struct {
	path string
	size int64
}

type rowStream struct {
	pull func() (any, error)
	shut func()
	env  *dlEnv
	done bool
}

func (s *rowStream) next() (any, error) {
	if s == nil || s.done {
		return nil, io.EOF
	}
	if s.env != nil {
		if err := s.env.tick(); err != nil {
			s.close()
			return nil, err
		}
	}
	v, err := s.pull()
	if err != nil {
		s.close()
		return nil, err
	}
	return v, nil
}

func (s *rowStream) close() {
	if s == nil || s.done {
		return
	}
	s.done = true
	if s.shut != nil {
		s.shut()
	}
}

func fileArg(args []any) (*dataFile, bool) {
	if len(args) != 1 {
		return nil, false
	}
	f, ok := args[0].(*dataFile)
	return f, ok
}

func sampleUTF8(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 8192)
	n, _ := f.Read(buf)
	return utf8.Valid(buf[:n])
}

func realizeValue(v any) (any, error) {
	switch x := v.(type) {
	case *dataFile:
		return nil, dataErr("RESULT_TOO_LARGE", "this file is large; use csv, json, xml, or outline and return the rows you need")
	case *rowStream:
		return collectStream(x, dataStoreByteLimit)
	default:
		return v, nil
	}
}

func collectStream(s *rowStream, limit int) ([]any, error) {
	defer s.close()
	var out []any
	var n int
	for {
		item, err := s.next()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		if limit > 0 {
			b, err := json.Marshal(item)
			if err != nil {
				return nil, err
			}
			n += len(b)
			if n > limit {
				return nil, dataErr("RESULT_TOO_LARGE", "the result is larger than the result store; filter it further")
			}
		}
		out = append(out, item)
	}
}

func streamHigher(name string, s *rowStream, pred *ex, env *dlEnv) (any, error) {
	s.env = env
	switch name {
	case "filter":
		return filterStream(s, pred, env), nil
	case "map":
		return mapStream(s, pred, env), nil
	case "sort":
		arr, err := collectStream(s, 0)
		if err != nil {
			return nil, err
		}
		return sortArr(arr, pred, env)
	default:
		arr, err := collectStream(s, 0)
		if err != nil {
			return nil, err
		}
		return groupArr(arr, pred, env)
	}
}

func streamColl(name string, s *rowStream, rest []any, env *dlEnv) (any, error) {
	s.env = env
	switch name {
	case "count":
		return countStream(s)
	case "first":
		return firstStream(s)
	case "unique":
		arr, err := collectStream(s, 0)
		if err != nil {
			return nil, err
		}
		return uniqueArr(arr), nil
	default:
		if len(rest) != 1 {
			return nil, dataErr("TYPE_ERROR", "take expects a count")
		}
		n, ok := asFloat(rest[0])
		if !ok || n < 0 {
			return nil, dataErr("TYPE_ERROR", "take expects a non-negative count")
		}
		return takeStream(s, int(n))
	}
}

func filterStream(src *rowStream, pred *ex, env *dlEnv) *rowStream {
	return &rowStream{
		env: env,
		pull: func() (any, error) {
			for {
				item, err := src.next()
				if err != nil {
					return nil, err
				}
				v, err := withDot(env, item, pred)
				if err != nil {
					src.close()
					return nil, err
				}
				b, ok := v.(bool)
				if !ok {
					src.close()
					return nil, dataErr("TYPE_ERROR", "filter predicate must be a boolean")
				}
				if b {
					return item, nil
				}
			}
		},
		shut: src.close,
	}
}

func mapStream(src *rowStream, pred *ex, env *dlEnv) *rowStream {
	return &rowStream{
		env: env,
		pull: func() (any, error) {
			item, err := src.next()
			if err != nil {
				return nil, err
			}
			v, err := withDot(env, item, pred)
			if err != nil {
				src.close()
				return nil, err
			}
			return v, nil
		},
		shut: src.close,
	}
}

func firstStream(s *rowStream) (any, error) {
	defer s.close()
	item, err := s.next()
	if err == io.EOF {
		return nil, nil
	}
	return item, err
}

func countStream(s *rowStream) (int64, error) {
	defer s.close()
	var n int64
	for {
		_, err := s.next()
		if err == io.EOF {
			return n, nil
		}
		if err != nil {
			return 0, err
		}
		n++
	}
}

func takeStream(s *rowStream, k int) ([]any, error) {
	defer s.close()
	out := []any{}
	for len(out) < k {
		item, err := s.next()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func openCSVStream(f *dataFile, env *dlEnv) (*rowStream, error) {
	file, r, head, err := openCSV(f.path)
	if err != nil {
		return nil, err
	}
	return &rowStream{
		env: env,
		pull: func() (any, error) {
			row, err := r.Read()
			if err == io.EOF {
				return nil, io.EOF
			}
			if err != nil {
				return nil, &dataError{Code: "PARSE_ERROR", Message: err.Error()}
			}
			obj := map[string]any{}
			for i, h := range head {
				val := ""
				if i < len(row) {
					val = row[i]
				}
				obj[h] = val
			}
			return obj, nil
		},
		shut: func() { file.Close() },
	}, nil
}

func openCSV(path string) (*os.File, *csv.Reader, []string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, dataErr("INVALID_REF", err.Error())
	}
	r := csv.NewReader(file)
	r.FieldsPerRecord = -1
	head, err := r.Read()
	if err != nil {
		file.Close()
		return nil, nil, nil, dataErr("PARSE_ERROR", "csv has no header")
	}
	return file, r, append([]string(nil), head...), nil
}

func openJSONStream(f *dataFile, env *dlEnv) (any, error) {
	file, err := os.Open(f.path)
	if err != nil {
		return nil, dataErr("INVALID_REF", err.Error())
	}
	br := bufio.NewReader(file)
	kind, err := firstJSONByte(br)
	if err != nil {
		file.Close()
		return nil, err
	}
	dec := json.NewDecoder(br)
	dec.UseNumber()
	switch kind {
	case '[':
		tok, err := dec.Token()
		if err != nil || tok != json.Delim('[') {
			file.Close()
			return nil, dataErr("PARSE_ERROR", "json array expected")
		}
		return jsonArrayStream(dec, file, env), nil
	case '{':
		if !secondJSONValue(f.path) {
			file.Close()
			return nil, dataErr("PARSE_ERROR", "a large JSON object is not a list of records; use an array, one value per line, or CSV")
		}
		return jsonValueStream(dec, file, env), nil
	default:
		file.Close()
		return nil, dataErr("PARSE_ERROR", "json value expected")
	}
}

func jsonArrayStream(dec *json.Decoder, file *os.File, env *dlEnv) *rowStream {
	return &rowStream{
		env: env,
		pull: func() (any, error) {
			if !dec.More() {
				return nil, io.EOF
			}
			var v any
			if err := dec.Decode(&v); err != nil {
				return nil, &dataError{Code: "PARSE_ERROR", Message: err.Error()}
			}
			return foldJSON(v), nil
		},
		shut: func() { file.Close() },
	}
}

func jsonValueStream(dec *json.Decoder, file *os.File, env *dlEnv) *rowStream {
	return &rowStream{
		env: env,
		pull: func() (any, error) {
			var v any
			if err := dec.Decode(&v); err != nil {
				if err == io.EOF {
					return nil, io.EOF
				}
				return nil, &dataError{Code: "PARSE_ERROR", Message: err.Error()}
			}
			return foldJSON(v), nil
		},
		shut: func() { file.Close() },
	}
}

func firstJSONByte(br *bufio.Reader) (byte, error) {
	for {
		c, err := br.ReadByte()
		if err != nil {
			return 0, dataErr("PARSE_ERROR", "empty json")
		}
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			continue
		}
		if err := br.UnreadByte(); err != nil {
			return 0, err
		}
		return c, nil
	}
}

func secondJSONValue(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 1<<20)
	n, _ := f.Read(buf)
	return jsonValueCount(buf[:n]) >= 2
}

func jsonValueCount(buf []byte) int {
	i := 0
	n := 0
	for i < len(buf) && n < 2 {
		for i < len(buf) && (buf[i] == ' ' || buf[i] == '\n' || buf[i] == '\r' || buf[i] == '\t') {
			i++
		}
		if i >= len(buf) {
			break
		}
		end, ok := skipJSONValue(buf, i)
		if !ok {
			if n == 0 {
				return 1
			}
			break
		}
		i = end
		n++
	}
	return n
}

func skipJSONValue(buf []byte, i int) (int, bool) {
	if i >= len(buf) {
		return i, false
	}
	switch buf[i] {
	case '{', '[':
		depth := 1
		i++
		inStr := false
		esc := false
		for i < len(buf) && depth > 0 {
			c := buf[i]
			i++
			if inStr {
				if esc {
					esc = false
					continue
				}
				if c == '\\' {
					esc = true
					continue
				}
				if c == '"' {
					inStr = false
				}
				continue
			}
			switch c {
			case '"':
				inStr = true
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		}
		return i, depth == 0
	case '"':
		i++
		esc := false
		for i < len(buf) {
			c := buf[i]
			i++
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				return i, true
			}
		}
		return i, false
	default:
		start := i
		for i < len(buf) && buf[i] != ',' && buf[i] != '}' && buf[i] != ']' && buf[i] != ' ' && buf[i] != '\n' && buf[i] != '\r' && buf[i] != '\t' {
			i++
		}
		return i, i > start
	}
}

func openXMLStream(f *dataFile, env *dlEnv) (*rowStream, error) {
	if xmlHasDoctype(f.path) {
		return nil, dataErr("PARSE_ERROR", "XML external entities and doctypes are refused")
	}
	file, err := os.Open(f.path)
	if err != nil {
		return nil, dataErr("INVALID_REF", err.Error())
	}
	dec := xml.NewDecoder(file)
	dec.CharsetReader = func(string, io.Reader) (io.Reader, error) {
		return nil, fmt.Errorf("external encoding refused")
	}
	dec.Strict = true
	for {
		tok, err := dec.Token()
		if err != nil {
			file.Close()
			return nil, dataErr("PARSE_ERROR", "empty xml")
		}
		if _, ok := tok.(xml.StartElement); ok {
			break
		}
	}
	return &rowStream{
		env: env,
		pull: func() (any, error) {
			for {
				tok, err := dec.Token()
				if err == io.EOF {
					return nil, io.EOF
				}
				if err != nil {
					return nil, &dataError{Code: "PARSE_ERROR", Message: err.Error()}
				}
				switch t := tok.(type) {
				case xml.StartElement:
					return readXMLElem(dec, t)
				case xml.EndElement:
					return nil, io.EOF
				case xml.Directive:
					if strings.Contains(strings.ToUpper(string(t)), "DOCTYPE") {
						return nil, dataErr("PARSE_ERROR", "XML external entities and doctypes are refused")
					}
				}
			}
		},
		shut: func() { file.Close() },
	}, nil
}

func xmlHasDoctype(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	upper := strings.ToUpper(string(buf[:n]))
	return strings.Contains(upper, "<!DOCTYPE") || strings.Contains(upper, "<!ENTITY")
}

func outlineFile(f *dataFile) (string, error) {
	kind := sniffFile(f.path)
	switch kind {
	case "csv":
		return outlineCSV(f)
	case "json":
		return outlineJSON(f)
	case "xml":
		return outlineXML(f)
	default:
		return outlineText(f)
	}
}

func sniffFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "text"
	}
	defer f.Close()
	buf := make([]byte, 8192)
	n, _ := f.Read(buf)
	t := bytes.TrimSpace(buf[:n])
	if len(t) == 0 {
		return "text"
	}
	switch t[0] {
	case '{', '[':
		return "json"
	case '<':
		return "xml"
	}
	if looksLikeCSV(string(t)) {
		return "csv"
	}
	return "text"
}

func outlineCSV(f *dataFile) (string, error) {
	file, r, head, err := openCSV(f.path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var n int64
	for {
		_, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", &dataError{Code: "PARSE_ERROR", Message: err.Error()}
		}
		n++
	}
	o := &outl{lim: dataModelLimit}
	o.line(0, fmt.Sprintf("array n=%d of object n=%d", n, len(head)))
	for _, name := range head {
		o.line(1, name+": string")
	}
	return strings.TrimRight(o.b.String(), "\n"), nil
}

func outlineJSON(f *dataFile) (string, error) {
	file, err := os.Open(f.path)
	if err != nil {
		return "", dataErr("INVALID_REF", err.Error())
	}
	defer file.Close()
	br := bufio.NewReader(file)
	kind, err := firstJSONByte(br)
	if err != nil {
		return "", err
	}
	dec := json.NewDecoder(br)
	dec.UseNumber()
	rows := kind == '{' && secondJSONValue(f.path)
	if kind == '[' || rows {
		if kind == '[' {
			if _, err := dec.Token(); err != nil {
				return "", &dataError{Code: "PARSE_ERROR", Message: err.Error()}
			}
		}
		var first any
		var n int64
		for {
			if kind == '[' && !dec.More() {
				break
			}
			var v any
			err := dec.Decode(&v)
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", &dataError{Code: "PARSE_ERROR", Message: err.Error()}
			}
			if n == 0 {
				first = foldJSON(v)
			}
			n++
		}
		if n == 0 {
			return "array n=0", nil
		}
		o := &outl{lim: dataModelLimit}
		o.line(0, fmt.Sprintf("array n=%d of %s", n, typePhrase(first)))
		writeInner(o, first, 1)
		return strings.TrimRight(o.b.String(), "\n"), nil
	}
	return outlineJSONObject(dec)
}

func outlineJSONObject(dec *json.Decoder) (string, error) {
	o := &outl{lim: dataModelLimit}
	if err := writeJSONShape(dec, o, 0); err != nil {
		return "", err
	}
	return strings.TrimRight(o.b.String(), "\n"), nil
}

func writeJSONShape(dec *json.Decoder, o *outl, depth int) error {
	tok, err := dec.Token()
	if err != nil {
		return &dataError{Code: "PARSE_ERROR", Message: err.Error()}
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			return writeJSONObject(dec, o, depth)
		case '[':
			return writeJSONArray(dec, o, depth)
		default:
			return dataErr("PARSE_ERROR", "unexpected json")
		}
	default:
		o.line(depth, jsonTokenPhrase(tok))
		return nil
	}
}

func writeJSONObject(dec *json.Decoder, o *outl, depth int) error {
	var n int
	type field struct {
		key   string
		shape string
	}
	var fields []field
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return &dataError{Code: "PARSE_ERROR", Message: err.Error()}
		}
		key, _ := keyTok.(string)
		n++
		shape, err := jsonShapeText(dec)
		if err != nil {
			return err
		}
		if len(fields) < 32 {
			fields = append(fields, field{key, shape})
		}
	}
	if _, err := dec.Token(); err != nil {
		return &dataError{Code: "PARSE_ERROR", Message: err.Error()}
	}
	o.line(depth, fmt.Sprintf("object n=%d", n))
	for _, f := range fields {
		for i, line := range strings.Split(f.shape, "\n") {
			if i == 0 {
				o.line(depth+1, f.key+": "+line)
				continue
			}
			o.line(depth+1, line)
		}
	}
	return nil
}

func writeJSONArray(dec *json.Decoder, o *outl, depth int) error {
	var n int64
	var first string
	for dec.More() {
		if n == 0 {
			shape, err := jsonShapeText(dec)
			if err != nil {
				return err
			}
			first = shape
		} else if err := skipJSONDecoder(dec); err != nil {
			return err
		}
		n++
	}
	if _, err := dec.Token(); err != nil {
		return &dataError{Code: "PARSE_ERROR", Message: err.Error()}
	}
	if first == "" {
		o.line(depth, "array n=0")
		return nil
	}
	lines := strings.Split(first, "\n")
	o.line(depth, fmt.Sprintf("array n=%d of %s", n, lines[0]))
	for _, line := range lines[1:] {
		o.line(depth+1, strings.TrimLeft(line, " "))
	}
	return nil
}

func jsonShapeText(dec *json.Decoder) (string, error) {
	tok, err := dec.Token()
	if err != nil {
		return "", &dataError{Code: "PARSE_ERROR", Message: err.Error()}
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return jsonTokenPhrase(tok), nil
	}
	inner := &outl{lim: dataModelLimit}
	switch d {
	case '{':
		if err := writeJSONObject(dec, inner, 0); err != nil {
			return "", err
		}
	case '[':
		if err := writeJSONArray(dec, inner, 0); err != nil {
			return "", err
		}
	default:
		return "", dataErr("PARSE_ERROR", "unexpected json")
	}
	return strings.TrimRight(inner.b.String(), "\n"), nil
}

func jsonTokenPhrase(tok any) string {
	switch tok.(type) {
	case string:
		return "string"
	case json.Number:
		return "number"
	case bool:
		return "bool"
	case nil:
		return "null"
	default:
		return "value"
	}
}

func skipJSONDecoder(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return &dataError{Code: "PARSE_ERROR", Message: err.Error()}
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	if d != '{' && d != '[' {
		return nil
	}
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return &dataError{Code: "PARSE_ERROR", Message: err.Error()}
		}
		switch tok {
		case json.Delim('{'), json.Delim('['):
			depth++
		case json.Delim('}'), json.Delim(']'):
			depth--
		}
	}
	return nil
}

func outlineXML(f *dataFile) (string, error) {
	if xmlHasDoctype(f.path) {
		return "", dataErr("PARSE_ERROR", "XML external entities and doctypes are refused")
	}
	file, err := os.Open(f.path)
	if err != nil {
		return "", dataErr("INVALID_REF", err.Error())
	}
	defer file.Close()
	dec := xml.NewDecoder(file)
	dec.CharsetReader = func(string, io.Reader) (io.Reader, error) {
		return nil, fmt.Errorf("external encoding refused")
	}
	dec.Strict = true
	var root string
	depth := 0
	var order []string
	count := map[string]int{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", &dataError{Code: "PARSE_ERROR", Message: err.Error()}
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if depth == 0 {
				root = t.Name.Local
			}
			if depth == 1 {
				name := t.Name.Local
				if count[name] == 0 {
					order = append(order, name)
				}
				count[name]++
			}
			depth++
		case xml.EndElement:
			depth--
		}
	}
	o := &outl{lim: dataModelLimit}
	if root == "" {
		root = "xml"
	}
	o.line(0, root)
	for _, name := range order {
		line := name
		if count[name] > 1 {
			line += fmt.Sprintf(" n=%d", count[name])
		}
		o.line(1, line)
	}
	return strings.TrimRight(o.b.String(), "\n"), nil
}

func outlineText(f *dataFile) (string, error) {
	file, err := os.Open(f.path)
	if err != nil {
		return "", dataErr("INVALID_REF", err.Error())
	}
	defer file.Close()
	buf := make([]byte, 32<<10)
	var lines int64
	var last byte
	var saw bool
	for {
		n, err := file.Read(buf)
		for _, c := range buf[:n] {
			saw = true
			if c == '\n' {
				lines++
			}
			last = c
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	if saw && last != '\n' {
		lines++
	}
	return fmt.Sprintf("text lines=%d chars=%d", lines, f.size), nil
}
