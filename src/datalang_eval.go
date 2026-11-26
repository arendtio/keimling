package main

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type ex struct {
	kind string
	name string
	str  string
	num  any
	b    bool
	recv *ex
	args []*ex
	keys []string
	rule *gRule
}

type stmt struct {
	kind  string
	name  string
	expr  *ex
	rules map[string]*gRule
}

type program struct{ stmts []stmt }

// dlHost is the process side of mcp(), exec(), and read_file(). Nil in a
// program that only transforms values already in hand.
type dlHost struct {
	call func(name string, args any) (any, error)
	// allowed names the MCP tools that were part of the approval.
	allowed map[string]bool

	exec        func(command string) (any, error)
	execAllowed map[string]bool

	read        func(path string) (any, error)
	readAllowed map[string]bool
}

type dlEnv struct {
	vars     map[string]any
	dot      any
	hasDot   bool
	deadline time.Time
	host     *dlHost
}

func (e *dlEnv) tick() error {
	if time.Now().After(e.deadline) {
		return dataErr("TIMEOUT", "DataLang timed out")
	}
	return nil
}

// mcpProgramNames lists the tool names written as string literals in mcp().
// calls is true when the program contains any mcp() call, including one whose
// name is computed. A computed name is not part of the approval.
func mcpProgramNames(p *program) (literals []string, calls bool) {
	if p == nil {
		return nil, false
	}
	seen := map[string]bool{}
	var walk func(*ex)
	walk = func(e *ex) {
		if e == nil {
			return
		}
		if e.kind == "call" && e.name == "mcp" && e.recv == nil {
			calls = true
			if len(e.args) > 0 && e.args[0] != nil && e.args[0].kind == "str" && e.args[0].str != "" && !seen[e.args[0].str] {
				seen[e.args[0].str] = true
				literals = append(literals, e.args[0].str)
			}
		}
		walk(e.recv)
		for _, arg := range e.args {
			walk(arg)
		}
	}
	for _, st := range p.stmts {
		walk(st.expr)
	}
	return literals, calls
}

// programStringCalls lists the literal arguments of exec() and read_file().
// computed is true for a call whose argument is not a literal, which cannot be
// part of an approval.
func programStringCalls(p *program, name string) (literals []string, computed bool) {
	if p == nil {
		return nil, false
	}
	seen := map[string]bool{}
	var walk func(*ex)
	walk = func(e *ex) {
		if e == nil {
			return
		}
		if e.kind == "call" && e.name == name && e.recv == nil {
			if len(e.args) != 1 || e.args[0] == nil || e.args[0].kind != "str" || strings.TrimSpace(e.args[0].str) == "" {
				computed = true
			} else if !seen[e.args[0].str] {
				seen[e.args[0].str] = true
				literals = append(literals, e.args[0].str)
			}
		}
		walk(e.recv)
		for _, arg := range e.args {
			walk(arg)
		}
	}
	for _, st := range p.stmts {
		walk(st.expr)
	}
	return literals, computed
}

func runDataLang(code string, input any, timeout time.Duration) (any, error) {
	return evalDataLang(code, map[string]any{"input": input}, timeout)
}

func evalDataLang(code string, vars map[string]any, timeout time.Duration) (any, error) {
	return evalDataLangHost(code, vars, timeout, nil)
}

func evalDataLangHost(code string, vars map[string]any, timeout time.Duration, host *dlHost) (any, error) {
	if timeout <= 0 {
		timeout = dataDefaultTimeout
	}
	if vars == nil {
		vars = map[string]any{}
	}
	prog, err := parseProgram(code)
	if err != nil {
		return nil, err
	}
	env := &dlEnv{vars: vars, deadline: time.Now().Add(timeout), host: host}
	var result any
	returned := false
	for _, st := range prog.stmts {
		switch st.kind {
		case "let":
			v, err := eval(st.expr, env)
			if err != nil {
				return nil, err
			}
			env.vars[st.name] = v
		case "grammar":
			for name, rule := range st.rules {
				env.vars[name] = rule
			}
		case "return":
			v, err := eval(st.expr, env)
			if err != nil {
				return nil, err
			}
			v, err = realizeValue(v)
			if err != nil {
				return nil, err
			}
			result = v
			returned = true
		default:
			return nil, dataErr("SYNTAX", "unknown statement")
		}
	}
	if !returned {
		return nil, dataErr("SYNTAX", "program has no return")
	}
	return result, nil
}

func eval(e *ex, env *dlEnv) (any, error) {
	if e == nil {
		return nil, dataErr("SYNTAX", "empty expression")
	}
	if err := env.tick(); err != nil {
		return nil, err
	}
	switch e.kind {
	case "num":
		return e.num, nil
	case "str":
		return e.str, nil
	case "bool":
		return e.b, nil
	case "null":
		return nil, nil
	case "rule":
		return e.rule, nil
	case "ident":
		v, ok := env.vars[e.name]
		if !ok {
			return nil, &dataError{Code: "MISSING_FIELD", Message: "unknown name " + e.name}
		}
		return v, nil
	case "dot":
		if !env.hasDot {
			return nil, dataErr("TYPE_ERROR", "`.` is only valid inside filter, map, sort, or group")
		}
		return env.dot, nil
	case "field":
		base, err := evalBase(e.recv, env)
		if err != nil {
			return nil, err
		}
		return fieldOf(base, e.name)
	case "index":
		base, err := evalBase(e.recv, env)
		if err != nil {
			return nil, err
		}
		idx, err := eval(e.args[0], env)
		if err != nil {
			return nil, err
		}
		return indexOf(base, idx)
	case "list":
		out := make([]any, len(e.args))
		for i, a := range e.args {
			v, err := eval(a, env)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	case "object":
		out := map[string]any{}
		for i, k := range e.keys {
			v, err := eval(e.args[i], env)
			if err != nil {
				return nil, err
			}
			out[k] = v
		}
		return out, nil
	case "unary":
		v, err := eval(e.args[0], env)
		if err != nil {
			return nil, err
		}
		if e.name == "!" {
			b, ok := v.(bool)
			if !ok {
				return nil, dataErr("TYPE_ERROR", "! expects a boolean")
			}
			return !b, nil
		}
		n, ok := asFloat(v)
		if !ok {
			return nil, dataErr("TYPE_ERROR", "unary minus expects a number")
		}
		if i, ok := v.(int64); ok {
			return -i, nil
		}
		return -n, nil
	case "binary":
		return evalBinary(e, env)
	case "ternary":
		cond, err := eval(e.args[0], env)
		if err != nil {
			return nil, err
		}
		b, ok := cond.(bool)
		if !ok {
			return nil, dataErr("TYPE_ERROR", "? expects a boolean condition")
		}
		if b {
			return eval(e.args[1], env)
		}
		return eval(e.args[2], env)
	case "coalesce":
		left, err := eval(e.args[0], env)
		if err != nil {
			return nil, err
		}
		if left == nil {
			return eval(e.args[1], env)
		}
		return left, nil
	case "call":
		return applyFn(e.name, nil, false, e.args, env)
	case "pipe":
		left, err := eval(e.recv, env)
		if err != nil {
			return nil, err
		}
		return applyFn(e.name, left, true, e.args, env)
	default:
		return nil, dataErr("SYNTAX", "cannot evaluate "+e.kind)
	}
}

func evalBase(recv *ex, env *dlEnv) (any, error) {
	if recv == nil {
		if !env.hasDot {
			return nil, dataErr("TYPE_ERROR", "`.` is only valid inside filter, map, sort, or group")
		}
		return env.dot, nil
	}
	return eval(recv, env)
}

func evalBinary(e *ex, env *dlEnv) (any, error) {
	left, err := eval(e.args[0], env)
	if err != nil {
		return nil, err
	}
	switch e.name {
	case "&&":
		b, ok := left.(bool)
		if !ok {
			return nil, dataErr("TYPE_ERROR", "&& expects booleans")
		}
		if !b {
			return false, nil
		}
	case "||":
		b, ok := left.(bool)
		if !ok {
			return nil, dataErr("TYPE_ERROR", "|| expects booleans")
		}
		if b {
			return true, nil
		}
	}
	right, err := eval(e.args[1], env)
	if err != nil {
		return nil, err
	}
	switch e.name {
	case "&&", "||":
		b, ok := right.(bool)
		if !ok {
			return nil, dataErr("TYPE_ERROR", e.name+" expects booleans")
		}
		return b, nil
	case "==":
		return numOrDeepEqual(left, right), nil
	case "!=":
		return !numOrDeepEqual(left, right), nil
	case "<", "<=", ">", ">=":
		return cmpOrdered(left, right, e.name)
	case "+":
		return addValues(left, right)
	case "-", "*", "/", "%":
		return arith(left, right, e.name)
	default:
		return nil, dataErr("SYNTAX", "unknown operator "+e.name)
	}
}

func numOrDeepEqual(a, b any) bool {
	af, aok := asFloat(a)
	bf, bok := asFloat(b)
	if aok && bok {
		return af == bf
	}
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func cmpOrdered(a, b any, op string) (bool, error) {
	af, aok := asFloat(a)
	bf, bok := asFloat(b)
	var less, eq bool
	if aok && bok {
		less, eq = af < bf, af == bf
	} else {
		as, aok := a.(string)
		bs, bok := b.(string)
		if !aok || !bok {
			return false, dataErr("TYPE_ERROR", op+" expects numbers or strings")
		}
		less, eq = as < bs, as == bs
	}
	switch op {
	case "<":
		return less, nil
	case "<=":
		return less || eq, nil
	case ">":
		return !less && !eq, nil
	case ">=":
		return !less, nil
	default:
		return false, dataErr("SYNTAX", op)
	}
}

func addValues(a, b any) (any, error) {
	if af, aok := asFloat(a); aok {
		if bf, bok := asFloat(b); bok {
			return numResult(a, b, af+bf), nil
		}
	}
	return nil, dataErr("TYPE_ERROR", "+ expects numbers")
}

func arith(a, b any, op string) (any, error) {
	af, aok := asFloat(a)
	bf, bok := asFloat(b)
	if !aok || !bok {
		return nil, dataErr("TYPE_ERROR", op+" expects numbers")
	}
	var r float64
	switch op {
	case "-":
		r = af - bf
	case "*":
		r = af * bf
	case "/":
		if bf == 0 {
			return nil, dataErr("TYPE_ERROR", "division by zero")
		}
		r = af / bf
	case "%":
		if bf == 0 {
			return nil, dataErr("TYPE_ERROR", "division by zero")
		}
		ai, bi := int64(af), int64(bf)
		return ai % bi, nil
	}
	return numResult(a, b, r), nil
}

func numResult(a, b any, r float64) any {
	_, ai := a.(int64)
	_, bi := b.(int64)
	if ai && bi && r == float64(int64(r)) {
		return int64(r)
	}
	return r
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func fieldOf(base any, name string) (any, error) {
	m, ok := base.(map[string]any)
	if !ok {
		return nil, dataErr("TYPE_ERROR", "field access expects an object")
	}
	v, ok := m[name]
	if !ok {
		return nil, &dataError{Code: "MISSING_FIELD", Message: "missing field " + name}
	}
	return v, nil
}

func indexOf(base, idx any) (any, error) {
	n, ok := asFloat(idx)
	if !ok || n != float64(int(n)) {
		return nil, dataErr("TYPE_ERROR", "index must be an integer")
	}
	i := int(n)
	arr, ok := base.([]any)
	if !ok {
		return nil, dataErr("TYPE_ERROR", "index expects an array")
	}
	if i < 0 || i >= len(arr) {
		return nil, dataErr("INVALID_INDEX", fmt.Sprintf("index %d out of range", i))
	}
	return arr[i], nil
}

func applyFn(name string, recv any, hasRecv bool, args []*ex, env *dlEnv) (any, error) {
	switch name {
	case "all":
		if hasRecv || len(args) == 0 {
			return nil, dataErr("TYPE_ERROR", "all expects one or more calls")
		}
		out := make([]any, len(args))
		errs := make([]error, len(args))
		var wg sync.WaitGroup
		for i, arg := range args {
			wg.Add(1)
			go func(i int, arg *ex) {
				defer wg.Done()
				out[i], errs[i] = eval(arg, env)
			}(i, arg)
		}
		wg.Wait()
		for _, err := range errs {
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case "mcp":
		if hasRecv || len(args) != 2 {
			return nil, dataErr("TYPE_ERROR", "mcp expects a name and an args object")
		}
		if env.host == nil || env.host.call == nil {
			return nil, dataErr("MCP", "mcp is not available")
		}
		namev, err := eval(args[0], env)
		if err != nil {
			return nil, err
		}
		tool, ok := namev.(string)
		if !ok || strings.TrimSpace(tool) == "" {
			return nil, dataErr("TYPE_ERROR", "mcp name must be a string")
		}
		if env.host.allowed == nil || !env.host.allowed[tool] {
			return nil, dataErr("MCP", "tool was not part of the approval")
		}
		arg, err := eval(args[1], env)
		if err != nil {
			return nil, err
		}
		if arg == nil {
			arg = map[string]any{}
		}
		if _, ok := arg.(map[string]any); !ok {
			return nil, dataErr("TYPE_ERROR", "mcp args must be an object")
		}
		return env.host.call(tool, arg)
	case "exec":
		if hasRecv || len(args) != 1 {
			return nil, dataErr("TYPE_ERROR", "exec expects one command")
		}
		if env.host == nil || env.host.exec == nil {
			return nil, dataErr("TOOL", "exec is not available")
		}
		cmdv, err := eval(args[0], env)
		if err != nil {
			return nil, err
		}
		command, ok := cmdv.(string)
		if !ok || strings.TrimSpace(command) == "" {
			return nil, dataErr("TYPE_ERROR", "exec command must be a string")
		}
		if env.host.execAllowed == nil || !env.host.execAllowed[command] {
			return nil, dataErr("TOOL", "exec was not part of the approval")
		}
		return env.host.exec(command)
	case "read_file":
		if hasRecv || len(args) != 1 {
			return nil, dataErr("TYPE_ERROR", "read_file expects one path")
		}
		if env.host == nil || env.host.read == nil {
			return nil, dataErr("TOOL", "read_file is not available")
		}
		pathv, err := eval(args[0], env)
		if err != nil {
			return nil, err
		}
		path, ok := pathv.(string)
		if !ok || strings.TrimSpace(path) == "" {
			return nil, dataErr("TYPE_ERROR", "read_file path must be a string")
		}
		if env.host.readAllowed == nil || !env.host.readAllowed[path] {
			return nil, dataErr("TOOL", "read_file was not part of the approval")
		}
		return env.host.read(path)
	case "filter", "map", "sort", "group":
		src, pred, err := higherArgs(name, recv, hasRecv, args, env)
		if err != nil {
			return nil, err
		}
		if s, ok := src.(*rowStream); ok {
			return streamHigher(name, s, pred, env)
		}
		arr, ok := src.([]any)
		if !ok {
			return nil, dataErr("TYPE_ERROR", name+" expects an array")
		}
		switch name {
		case "filter":
			return filterArr(arr, pred, env)
		case "map":
			return mapArr(arr, pred, env)
		case "sort":
			return sortArr(arr, pred, env)
		default:
			return groupArr(arr, pred, env)
		}
	case "take", "count", "first", "unique":
		src, rest, err := collArgs(name, recv, hasRecv, args, env)
		if err != nil {
			return nil, err
		}
		if s, ok := src.(*rowStream); ok {
			return streamColl(name, s, rest, env)
		}
		arr, ok := src.([]any)
		if !ok {
			return nil, dataErr("TYPE_ERROR", name+" expects an array")
		}
		switch name {
		case "count":
			return int64(len(arr)), nil
		case "first":
			if len(arr) == 0 {
				return nil, nil
			}
			return arr[0], nil
		case "unique":
			return uniqueArr(arr), nil
		default:
			if len(rest) != 1 {
				return nil, dataErr("TYPE_ERROR", "take expects a count")
			}
			n, ok := asFloat(rest[0])
			if !ok {
				return nil, dataErr("TYPE_ERROR", "take expects a count")
			}
			k := int(n)
			if k < 0 {
				return nil, dataErr("TYPE_ERROR", "take expects a non-negative count")
			}
			if k > len(arr) {
				k = len(arr)
			}
			return arr[:k], nil
		}
	default:
		if !knownPure(name) {
			return nil, &dataError{Code: "UNKNOWN_FUNCTION", Message: "Unknown function: " + name, Suggestion: "Use filter(predicate)", Example: `return input | filter(.status == "ok")`}
		}
		vals := []any{}
		if hasRecv {
			vals = append(vals, recv)
		}
		for _, a := range args {
			v, err := eval(a, env)
			if err != nil {
				return nil, err
			}
			vals = append(vals, v)
		}
		return callPure(name, vals, env)
	}
}

func higherArgs(name string, recv any, hasRecv bool, args []*ex, env *dlEnv) (any, *ex, error) {
	var src any
	var pred *ex
	if hasRecv {
		src = recv
		if len(args) != 1 {
			return nil, nil, dataErr("TYPE_ERROR", name+" expects a predicate")
		}
		pred = args[0]
	} else {
		if len(args) != 2 {
			return nil, nil, dataErr("TYPE_ERROR", name+" expects a collection and a predicate")
		}
		v, err := eval(args[0], env)
		if err != nil {
			return nil, nil, err
		}
		src = v
		pred = args[1]
	}
	return src, pred, nil
}

func collArgs(name string, recv any, hasRecv bool, args []*ex, env *dlEnv) (any, []any, error) {
	var src any
	var rest []*ex
	if hasRecv {
		src = recv
		rest = args
	} else {
		if len(args) < 1 {
			return nil, nil, dataErr("TYPE_ERROR", name+" expects an array")
		}
		v, err := eval(args[0], env)
		if err != nil {
			return nil, nil, err
		}
		src = v
		rest = args[1:]
	}
	out := []any{}
	for _, a := range rest {
		v, err := eval(a, env)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, v)
	}
	return src, out, nil
}

func withDot(env *dlEnv, item any, pred *ex) (any, error) {
	prev, had := env.dot, env.hasDot
	env.dot, env.hasDot = item, true
	v, err := eval(pred, env)
	env.dot, env.hasDot = prev, had
	return v, err
}

func filterArr(arr []any, pred *ex, env *dlEnv) (any, error) {
	out := []any{}
	for _, item := range arr {
		v, err := withDot(env, item, pred)
		if err != nil {
			return nil, err
		}
		b, ok := v.(bool)
		if !ok {
			return nil, dataErr("TYPE_ERROR", "filter predicate must be a boolean")
		}
		if b {
			out = append(out, item)
		}
	}
	return out, nil
}

func mapArr(arr []any, pred *ex, env *dlEnv) (any, error) {
	out := make([]any, len(arr))
	for i, item := range arr {
		v, err := withDot(env, item, pred)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func sortArr(arr []any, pred *ex, env *dlEnv) (any, error) {
	type pair struct {
		item any
		key  any
		i    int
	}
	rows := make([]pair, len(arr))
	for i, item := range arr {
		k, err := withDot(env, item, pred)
		if err != nil {
			return nil, err
		}
		rows[i] = pair{item, k, i}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		less, err := cmpOrdered(rows[i].key, rows[j].key, "<")
		if err != nil {
			return rows[i].i < rows[j].i
		}
		return less
	})
	out := make([]any, len(rows))
	for i, row := range rows {
		out[i] = row.item
	}
	return out, nil
}

func groupArr(arr []any, pred *ex, env *dlEnv) (any, error) {
	out := map[string]any{}
	order := []string{}
	for _, item := range arr {
		k, err := withDot(env, item, pred)
		if err != nil {
			return nil, err
		}
		key := groupKey(k)
		bucket, ok := out[key].([]any)
		if !ok {
			order = append(order, key)
			bucket = []any{}
		}
		out[key] = append(bucket, item)
	}
	_ = order
	return out, nil
}

func groupKey(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if n, ok := asFloat(v); ok {
		if n == float64(int64(n)) {
			return strconv.FormatInt(int64(n), 10)
		}
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func uniqueArr(arr []any) []any {
	seen := map[string]bool{}
	out := []any{}
	for _, item := range arr {
		b, _ := json.Marshal(item)
		if seen[string(b)] {
			continue
		}
		seen[string(b)] = true
		out = append(out, item)
	}
	return out
}

func knownPure(name string) bool {
	switch name {
	case "contains", "split", "join", "trim", "lower", "upper", "json", "csv", "xml", "outline", "parse", "extract", "lookup", "indexOf",
		"keys", "values", "entries", "from_entries", "length", "len", "type", "string", "number", "replace", "has", "get", "set", "omit",
		"slice", "lines", "skip", "startsWith", "endsWith", "sum", "min", "max":
		return true
	default:
		return false
	}
}

func callPure(name string, args []any, env *dlEnv) (any, error) {
	switch name {
	case "contains":
		if len(args) != 2 {
			return nil, dataErr("TYPE_ERROR", "contains expects a value and a search")
		}
		return containsValue(args[0], args[1])
	case "split":
		if len(args) != 2 {
			return nil, dataErr("TYPE_ERROR", "split expects text and a separator")
		}
		text, ok1 := args[0].(string)
		sep, ok2 := args[1].(string)
		if !ok1 || !ok2 {
			return nil, dataErr("TYPE_ERROR", "split expects strings")
		}
		parts := strings.Split(text, sep)
		out := make([]any, len(parts))
		for i, p := range parts {
			out[i] = p
		}
		return out, nil
	case "join":
		if len(args) != 2 {
			return nil, dataErr("TYPE_ERROR", "join expects an array and a separator")
		}
		arr, ok := args[0].([]any)
		sep, ok2 := args[1].(string)
		if !ok || !ok2 {
			return nil, dataErr("TYPE_ERROR", "join expects an array of strings")
		}
		bits := make([]string, len(arr))
		for i, item := range arr {
			s, ok := item.(string)
			if !ok {
				return nil, dataErr("TYPE_ERROR", "join expects an array of strings")
			}
			bits[i] = s
		}
		return strings.Join(bits, sep), nil
	case "trim":
		s, err := oneString("trim", args)
		if err != nil {
			return nil, err
		}
		return strings.TrimSpace(s), nil
	case "lower":
		s, err := oneString("lower", args)
		if err != nil {
			return nil, err
		}
		return strings.ToLower(s), nil
	case "upper":
		s, err := oneString("upper", args)
		if err != nil {
			return nil, err
		}
		return strings.ToUpper(s), nil
	case "json":
		if f, ok := fileArg(args); ok {
			return openJSONStream(f, env)
		}
		s, err := oneString("json", args)
		if err != nil {
			return nil, err
		}
		return parseJSONText(s)
	case "csv":
		if f, ok := fileArg(args); ok {
			return openCSVStream(f, env)
		}
		s, err := oneString("csv", args)
		if err != nil {
			return nil, err
		}
		return parseCSVText(s)
	case "xml":
		if f, ok := fileArg(args); ok {
			return openXMLStream(f, env)
		}
		s, err := oneString("xml", args)
		if err != nil {
			return nil, err
		}
		return parseXMLText(s)
	case "outline":
		if len(args) != 1 {
			return nil, dataErr("TYPE_ERROR", "outline expects one value")
		}
		if f, ok := args[0].(*dataFile); ok {
			return outlineFile(f)
		}
		return outlineAny(args[0]), nil
	case "parse":
		if len(args) != 2 {
			return nil, dataErr("TYPE_ERROR", "parse expects text and a rule")
		}
		return gParse(args[0], args[1], env)
	case "extract":
		if len(args) != 2 {
			return nil, dataErr("TYPE_ERROR", "extract expects text and a rule")
		}
		return gExtract(args[0], args[1], env)
	case "lookup":
		return lookupRow(args)
	case "indexOf":
		return indexOfValue(args)
	case "keys":
		return objectKeys(args)
	case "entries":
		return objectEntries(args)
	case "from_entries":
		return objectFromEntries(args)
	case "length", "len":
		return valueLength(args)
	case "slice":
		return valueSlice(args)
	case "lines":
		return valueLines(args)
	case "skip":
		return valueSkip(args)
	case "startsWith", "endsWith":
		return valueAffix(name, args)
	case "sum":
		return valueSum(args)
	case "min", "max":
		return valueMinMax(name, args)
	case "values":
		return objectValues(args)
	case "type":
		return valueKind(args)
	case "string":
		return valueString(args)
	case "number":
		return valueNumber(args)
	case "replace":
		return replaceText(args)
	case "has":
		return objectHas(args)
	case "get":
		return objectGet(args)
	case "set":
		return objectSet(args)
	case "omit":
		return objectOmit(args)
	default:
		return nil, &dataError{Code: "UNKNOWN_FUNCTION", Message: "Unknown function: " + name, Suggestion: "Use filter(predicate)", Example: `return input | filter(.status == "ok")`}
	}
}

func indexOfValue(args []any) (any, error) {
	if len(args) != 2 {
		return nil, dataErr("TYPE_ERROR", "indexOf expects an array and a value")
	}
	if s, ok := args[0].(*rowStream); ok {
		defer s.close()
		var i int64
		for {
			item, err := s.next()
			if err == io.EOF {
				return nil, nil
			}
			if err != nil {
				return nil, err
			}
			if numOrDeepEqual(item, args[1]) {
				return i, nil
			}
			i++
		}
	}
	arr, ok := args[0].([]any)
	if !ok {
		return nil, dataErr("TYPE_ERROR", "indexOf expects an array and a value")
	}
	for i, item := range arr {
		if numOrDeepEqual(item, args[1]) {
			return int64(i), nil
		}
	}
	return nil, nil
}

func lookupRow(args []any) (any, error) {
	if len(args) != 3 {
		return nil, dataErr("TYPE_ERROR", "lookup expects rows, a field name, and a key")
	}
	field, ok2 := args[1].(string)
	if !ok2 {
		return nil, dataErr("TYPE_ERROR", "lookup expects an array of objects and a field name")
	}
	if s, ok := args[0].(*rowStream); ok {
		defer s.close()
		for {
			item, err := s.next()
			if err == io.EOF {
				return nil, nil
			}
			if err != nil {
				return nil, err
			}
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			v, exists := obj[field]
			if exists && numOrDeepEqual(v, args[2]) {
				return item, nil
			}
		}
	}
	rows, ok := args[0].([]any)
	if !ok {
		return nil, dataErr("TYPE_ERROR", "lookup expects an array of objects and a field name")
	}
	for _, item := range rows {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		v, exists := obj[field]
		if exists && numOrDeepEqual(v, args[2]) {
			return item, nil
		}
	}
	return nil, nil
}

func objectKeys(args []any) (any, error) {
	m, err := oneObject("keys", args)
	if err != nil {
		return nil, err
	}
	raw := make([]string, 0, len(m))
	for k := range m {
		raw = append(raw, k)
	}
	sort.Strings(raw)
	out := make([]any, len(raw))
	for i, k := range raw {
		out[i] = k
	}
	return out, nil
}

func objectEntries(args []any) (any, error) {
	m, err := oneObject("entries", args)
	if err != nil {
		return nil, err
	}
	raw := make([]string, 0, len(m))
	for k := range m {
		raw = append(raw, k)
	}
	sort.Strings(raw)
	out := make([]any, len(raw))
	for i, k := range raw {
		out[i] = map[string]any{"key": k, "value": m[k]}
	}
	return out, nil
}

func objectFromEntries(args []any) (any, error) {
	if len(args) != 1 {
		return nil, dataErr("TYPE_ERROR", "from_entries expects rows of {key, value}")
	}
	arr, ok := args[0].([]any)
	if !ok {
		return nil, dataErr("TYPE_ERROR", "from_entries expects rows of {key, value}")
	}
	out := map[string]any{}
	for _, item := range arr {
		row, ok := item.(map[string]any)
		if !ok {
			return nil, dataErr("TYPE_ERROR", "from_entries expects rows of {key, value}")
		}
		key, ok := row["key"].(string)
		if !ok {
			return nil, dataErr("TYPE_ERROR", "from_entries expects a string key")
		}
		out[key] = row["value"]
	}
	return out, nil
}

func valueLength(args []any) (any, error) {
	if len(args) != 1 {
		return nil, dataErr("TYPE_ERROR", "length expects one value")
	}
	switch v := args[0].(type) {
	case nil:
		return int64(0), nil
	case string:
		return int64(utf8.RuneCountInString(v)), nil
	case []any:
		return int64(len(v)), nil
	case map[string]any:
		return int64(len(v)), nil
	default:
		return nil, dataErr("TYPE_ERROR", "length expects text, an array, an object, or null")
	}
}

func sliceBounds(n int, start, end int64, hasEnd bool) (int, int) {
	if start < 0 {
		start = int64(n) + start
	}
	if start < 0 {
		start = 0
	}
	if start > int64(n) {
		start = int64(n)
	}
	if !hasEnd {
		end = int64(n)
	}
	if end < 0 {
		end = int64(n) + end
	}
	if end < start {
		end = start
	}
	if end > int64(n) {
		end = int64(n)
	}
	return int(start), int(end)
}

func intArg(v any) (int64, bool) {
	f, ok := asFloat(v)
	if !ok || f != float64(int64(f)) {
		return 0, false
	}
	return int64(f), true
}

func valueSlice(args []any) (any, error) {
	if len(args) != 2 && len(args) != 3 {
		return nil, dataErr("TYPE_ERROR", "slice expects a value, a start, and an optional end")
	}
	start, ok := intArg(args[1])
	if !ok {
		return nil, dataErr("TYPE_ERROR", "slice expects integer bounds")
	}
	end := int64(0)
	hasEnd := len(args) == 3
	if hasEnd {
		end, ok = intArg(args[2])
		if !ok {
			return nil, dataErr("TYPE_ERROR", "slice expects integer bounds")
		}
	}
	switch v := args[0].(type) {
	case string:
		r := []rune(v)
		a, b := sliceBounds(len(r), start, end, hasEnd)
		return string(r[a:b]), nil
	case []any:
		a, b := sliceBounds(len(v), start, end, hasEnd)
		out := make([]any, b-a)
		copy(out, v[a:b])
		return out, nil
	default:
		return nil, dataErr("TYPE_ERROR", "slice expects text or an array")
	}
}

func valueLines(args []any) (any, error) {
	if len(args) != 1 {
		return nil, dataErr("TYPE_ERROR", "lines expects text")
	}
	text, ok := args[0].(string)
	if !ok {
		return nil, dataErr("TYPE_ERROR", "lines expects text")
	}
	raw := strings.Split(text, "\n")
	if len(raw) > 0 && raw[len(raw)-1] == "" {
		raw = raw[:len(raw)-1]
	}
	out := make([]any, len(raw))
	for i, line := range raw {
		out[i] = strings.TrimSuffix(line, "\r")
	}
	return out, nil
}

func valueSkip(args []any) (any, error) {
	if len(args) != 2 {
		return nil, dataErr("TYPE_ERROR", "skip expects a value and a count")
	}
	n, ok := intArg(args[1])
	if !ok || n < 0 {
		return nil, dataErr("TYPE_ERROR", "skip expects a non-negative integer count")
	}
	switch v := args[0].(type) {
	case string:
		r := []rune(v)
		k := int(n)
		if k > len(r) {
			k = len(r)
		}
		return string(r[k:]), nil
	case []any:
		k := int(n)
		if k > len(v) {
			k = len(v)
		}
		out := make([]any, len(v)-k)
		copy(out, v[k:])
		return out, nil
	default:
		return nil, dataErr("TYPE_ERROR", "skip expects text or an array")
	}
}

func valueAffix(name string, args []any) (any, error) {
	if len(args) != 2 {
		return nil, dataErr("TYPE_ERROR", name+" expects text and a prefix or suffix")
	}
	text, ok1 := args[0].(string)
	part, ok2 := args[1].(string)
	if !ok1 || !ok2 {
		return nil, dataErr("TYPE_ERROR", name+" expects strings")
	}
	if name == "startsWith" {
		return strings.HasPrefix(text, part), nil
	}
	return strings.HasSuffix(text, part), nil
}

func numberResult(f float64) any {
	if f == float64(int64(f)) {
		return int64(f)
	}
	return f
}

func valueSum(args []any) (any, error) {
	if len(args) != 1 {
		return nil, dataErr("TYPE_ERROR", "sum expects an array of numbers")
	}
	arr, ok := args[0].([]any)
	if !ok {
		return nil, dataErr("TYPE_ERROR", "sum expects an array of numbers")
	}
	total := float64(0)
	for _, item := range arr {
		n, ok := asFloat(item)
		if !ok {
			return nil, dataErr("TYPE_ERROR", "sum expects an array of numbers")
		}
		total += n
	}
	return numberResult(total), nil
}

func valueMinMax(name string, args []any) (any, error) {
	values := args
	if len(args) == 1 {
		if arr, ok := args[0].([]any); ok {
			values = arr
		}
	}
	if len(values) == 0 {
		return nil, nil
	}
	best, ok := asFloat(values[0])
	if !ok {
		return nil, dataErr("TYPE_ERROR", name+" expects numbers")
	}
	for _, item := range values[1:] {
		n, ok := asFloat(item)
		if !ok {
			return nil, dataErr("TYPE_ERROR", name+" expects numbers")
		}
		if (name == "min" && n < best) || (name == "max" && n > best) {
			best = n
		}
	}
	return numberResult(best), nil
}

func objectValues(args []any) (any, error) {
	m, err := oneObject("values", args)
	if err != nil {
		return nil, err
	}
	raw := make([]string, 0, len(m))
	for k := range m {
		raw = append(raw, k)
	}
	sort.Strings(raw)
	out := make([]any, len(raw))
	for i, k := range raw {
		out[i] = m[k]
	}
	return out, nil
}

func valueKind(args []any) (any, error) {
	if len(args) != 1 {
		return nil, dataErr("TYPE_ERROR", "type expects one value")
	}
	switch args[0].(type) {
	case nil:
		return "null", nil
	case bool:
		return "bool", nil
	case string:
		return "string", nil
	case int, int64, float64, json.Number:
		return "number", nil
	case []any:
		return "array", nil
	case map[string]any:
		return "object", nil
	default:
		return nil, dataErr("TYPE_ERROR", "type expects a JSON value")
	}
}

func valueString(args []any) (any, error) {
	if len(args) != 1 {
		return nil, dataErr("TYPE_ERROR", "string expects one value")
	}
	switch n := args[0].(type) {
	case nil:
		return "null", nil
	case string:
		return n, nil
	case bool:
		return strconv.FormatBool(n), nil
	case int:
		return strconv.Itoa(n), nil
	case int64:
		return strconv.FormatInt(n, 10), nil
	case float64:
		if n == float64(int64(n)) {
			return strconv.FormatInt(int64(n), 10), nil
		}
		return strconv.FormatFloat(n, 'f', -1, 64), nil
	case json.Number:
		return n.String(), nil
	default:
		b, err := json.Marshal(n)
		if err != nil {
			return nil, dataErr("TYPE_ERROR", "string expects a JSON value")
		}
		return string(b), nil
	}
}

func valueNumber(args []any) (any, error) {
	if len(args) != 1 {
		return nil, dataErr("TYPE_ERROR", "number expects a number or text")
	}
	if _, ok := asFloat(args[0]); ok {
		return args[0], nil
	}
	s, ok := args[0].(string)
	if !ok {
		return nil, dataErr("TYPE_ERROR", "number expects a number or text")
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, dataErr("TYPE_ERROR", "number expects a number or text")
	}
	return f, nil
}

func replaceText(args []any) (any, error) {
	if len(args) != 3 {
		return nil, dataErr("TYPE_ERROR", "replace expects text, a search, and a replacement")
	}
	text, ok1 := args[0].(string)
	old, ok2 := args[1].(string)
	repl, ok3 := args[2].(string)
	if !ok1 || !ok2 || !ok3 {
		return nil, dataErr("TYPE_ERROR", "replace expects strings")
	}
	if old == "" {
		return nil, dataErr("TYPE_ERROR", "replace search must not be empty")
	}
	return strings.ReplaceAll(text, old, repl), nil
}

func objectHas(args []any) (any, error) {
	m, name, err := objectField("has", args)
	if err != nil {
		return nil, err
	}
	_, ok := m[name]
	return ok, nil
}

func objectGet(args []any) (any, error) {
	m, name, err := objectField("get", args)
	if err != nil {
		return nil, err
	}
	v, ok := m[name]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func objectSet(args []any) (any, error) {
	if len(args) != 3 {
		return nil, dataErr("TYPE_ERROR", "set expects an object, a field name, and a value")
	}
	m, ok := args[0].(map[string]any)
	name, ok2 := args[1].(string)
	if !ok || !ok2 {
		return nil, dataErr("TYPE_ERROR", "set expects an object, a field name, and a value")
	}
	out := copyObject(m)
	out[name] = args[2]
	return out, nil
}

func objectOmit(args []any) (any, error) {
	m, name, err := objectField("omit", args)
	if err != nil {
		return nil, err
	}
	out := copyObject(m)
	delete(out, name)
	return out, nil
}

func objectField(fn string, args []any) (map[string]any, string, error) {
	if len(args) != 2 {
		return nil, "", dataErr("TYPE_ERROR", fn+" expects an object and a field name")
	}
	m, ok := args[0].(map[string]any)
	name, ok2 := args[1].(string)
	if !ok || !ok2 {
		return nil, "", dataErr("TYPE_ERROR", fn+" expects an object and a field name")
	}
	return m, name, nil
}

func oneObject(name string, args []any) (map[string]any, error) {
	if len(args) != 1 {
		return nil, dataErr("TYPE_ERROR", name+" expects an object")
	}
	m, ok := args[0].(map[string]any)
	if !ok {
		return nil, dataErr("TYPE_ERROR", name+" expects an object")
	}
	return m, nil
}

func copyObject(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func oneString(name string, args []any) (string, error) {
	if len(args) != 1 {
		return "", dataErr("TYPE_ERROR", name+" expects text")
	}
	s, ok := args[0].(string)
	if !ok {
		return "", dataErr("TYPE_ERROR", name+" expects text")
	}
	return s, nil
}

func containsValue(value, search any) (any, error) {
	switch v := value.(type) {
	case string:
		s, ok := search.(string)
		if !ok {
			return nil, dataErr("TYPE_ERROR", "contains on a string expects a string")
		}
		return strings.Contains(v, s), nil
	case []any:
		for _, item := range v {
			if numOrDeepEqual(item, search) {
				return true, nil
			}
		}
		return false, nil
	default:
		return nil, dataErr("TYPE_ERROR", "contains expects a string or an array")
	}
}

func parseJSONText(s string) (any, error) {
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, &dataError{Code: "PARSE_ERROR", Message: err.Error()}
	}
	return foldJSON(v), nil
}

func parseCSVText(s string) (any, error) {
	r := csv.NewReader(strings.NewReader(s))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, &dataError{Code: "PARSE_ERROR", Message: err.Error()}
	}
	if len(rows) == 0 {
		return nil, dataErr("PARSE_ERROR", "csv has no header")
	}
	head := rows[0]
	out := []any{}
	for _, row := range rows[1:] {
		obj := map[string]any{}
		for i, h := range head {
			val := ""
			if i < len(row) {
				val = row[i]
			}
			obj[h] = val
		}
		out = append(out, obj)
	}
	return out, nil
}

func parseXMLText(s string) (any, error) {
	upper := strings.ToUpper(s)
	if strings.Contains(upper, "<!DOCTYPE") || strings.Contains(upper, "<!ENTITY") {
		return nil, dataErr("PARSE_ERROR", "XML external entities and doctypes are refused")
	}
	dec := xml.NewDecoder(strings.NewReader(s))
	dec.CharsetReader = func(string, io.Reader) (io.Reader, error) {
		return nil, fmt.Errorf("external encoding refused")
	}
	dec.Strict = true
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil, dataErr("PARSE_ERROR", "empty xml")
		}
		if err != nil {
			return nil, &dataError{Code: "PARSE_ERROR", Message: err.Error()}
		}
		if start, ok := tok.(xml.StartElement); ok {
			return readXMLElem(dec, start)
		}
	}
}

func readXMLElem(dec *xml.Decoder, start xml.StartElement) (any, error) {
	attrs := map[string]any{}
	for _, a := range start.Attr {
		attrs[a.Name.Local] = a.Value
	}
	children := []any{}
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, &dataError{Code: "PARSE_ERROR", Message: err.Error()}
		}
		switch t := tok.(type) {
		case xml.StartElement:
			child, err := readXMLElem(dec, t)
			if err != nil {
				return nil, err
			}
			children = append(children, child)
		case xml.EndElement:
			return map[string]any{"tag": start.Name.Local, "attrs": attrs, "children": children}, nil
		case xml.CharData:
			text := string(t)
			if strings.TrimSpace(text) == "" {
				continue
			}
			children = append(children, map[string]any{"text": text})
		}
	}
}

type parser struct {
	s      string
	i      int
	gdepth int
}

func parseProgram(src string) (*program, error) {
	p := &parser{s: src}
	prog := &program{}
	for {
		p.skip()
		if p.i >= len(p.s) {
			break
		}
		id, ok := p.peekIdent()
		if !ok {
			return nil, p.err("expected let, grammar, or return")
		}
		switch id {
		case "let":
			p.i += len(id)
			name, err := p.ident()
			if err != nil {
				return nil, err
			}
			if !p.consumeOp("=") {
				return nil, p.err("expected =")
			}
			e, err := p.parsePipe()
			if err != nil {
				return nil, err
			}
			prog.stmts = append(prog.stmts, stmt{kind: "let", name: name, expr: e})
		case "return":
			p.i += len(id)
			e, err := p.parsePipe()
			if err != nil {
				return nil, err
			}
			prog.stmts = append(prog.stmts, stmt{kind: "return", expr: e})
		case "grammar":
			p.i += len(id)
			rules, err := p.parseGrammarBlock()
			if err != nil {
				return nil, err
			}
			prog.stmts = append(prog.stmts, stmt{kind: "grammar", rules: rules})
		default:
			return nil, p.err("expected let, grammar, or return")
		}
	}
	return prog, nil
}

func (p *parser) parsePipe() (*ex, error) {
	left, err := p.parseTernary()
	if err != nil {
		return nil, err
	}
	for p.consumeOp("|") {
		name, err := p.ident()
		if err != nil {
			return nil, err
		}
		if !p.consumeOp("(") {
			return nil, p.err("expected (")
		}
		args, err := p.parseArgList()
		if err != nil {
			return nil, err
		}
		left = &ex{kind: "pipe", name: name, recv: left, args: args}
	}
	return left, nil
}

func (p *parser) parseTernary() (*ex, error) {
	left, err := p.parseCoalesce()
	if err != nil {
		return nil, err
	}
	if p.consumeOp("?") {
		thenExpr, err := p.parsePipe()
		if err != nil {
			return nil, err
		}
		if !p.consumeOp(":") {
			return nil, p.err("expected : after the true branch")
		}
		elseExpr, err := p.parsePipe()
		if err != nil {
			return nil, err
		}
		return &ex{kind: "ternary", args: []*ex{left, thenExpr, elseExpr}}, nil
	}
	return left, nil
}

func (p *parser) parseCoalesce() (*ex, error) {
	left, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	for p.consumeOp("??") {
		right, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		left = &ex{kind: "coalesce", args: []*ex{left, right}}
	}
	return left, nil
}

func (p *parser) parseOr() (*ex, error) {
	return p.parseBool([]string{"||"}, []string{"or"}, p.parseAnd)
}
func (p *parser) parseAnd() (*ex, error) {
	return p.parseBool([]string{"&&"}, []string{"and"}, p.parseCmp)
}
func (p *parser) parseCmp() (*ex, error) {
	return p.parseBinary([]string{"==", "!=", "<=", ">=", "<", ">"}, p.parseAdd)
}
func (p *parser) parseAdd() (*ex, error) { return p.parseBinary([]string{"+", "-"}, p.parseMul) }
func (p *parser) parseMul() (*ex, error) {
	return p.parseBinary([]string{"*", "/", "%"}, p.parseUnary)
}

func (p *parser) parseBinary(ops []string, next func() (*ex, error)) (*ex, error) {
	left, err := next()
	if err != nil {
		return nil, err
	}
	for {
		op := ""
		for _, cand := range ops {
			if p.consumeOp(cand) {
				op = cand
				break
			}
		}
		if op == "" {
			return left, nil
		}
		right, err := next()
		if err != nil {
			return nil, err
		}
		left = &ex{kind: "binary", name: op, args: []*ex{left, right}}
	}
}

func (p *parser) parseBool(ops, words []string, next func() (*ex, error)) (*ex, error) {
	left, err := next()
	if err != nil {
		return nil, err
	}
	for {
		op := ""
		for _, cand := range ops {
			if p.consumeOp(cand) {
				op = cand
				break
			}
		}
		if op == "" {
			for _, w := range words {
				if p.consumeWord(w) {
					op = map[string]string{"or": "||", "and": "&&"}[w]
					break
				}
			}
		}
		if op == "" {
			return left, nil
		}
		right, err := next()
		if err != nil {
			return nil, err
		}
		left = &ex{kind: "binary", name: op, args: []*ex{left, right}}
	}
}

func (p *parser) parseUnary() (*ex, error) {
	if p.consumeWord("not") {
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ex{kind: "unary", name: "!", args: []*ex{inner}}, nil
	}
	if p.consumeOp("!") || p.consumeOp("-") {
		op := p.s[p.i-1 : p.i]
		if op == "-" && p.i < len(p.s) && p.s[p.i-1] == '-' {
			// consumeOp already advanced one byte
		}
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		name := "!"
		if strings.HasSuffix(p.recent(1), "-") && op != "!" {
			name = "-"
		}
		if op == "-" {
			name = "-"
		}
		return &ex{kind: "unary", name: name, args: []*ex{inner}}, nil
	}
	return p.parsePostfix()
}

func (p *parser) recent(n int) string {
	if p.i < n {
		return p.s[:p.i]
	}
	return p.s[p.i-n : p.i]
}

func (p *parser) parsePostfix() (*ex, error) {
	e, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		if p.consumeOp(".") {
			name, err := p.ident()
			if err != nil {
				return nil, err
			}
			e = &ex{kind: "field", recv: e, name: name}
			continue
		}
		if p.consumeOp("[") {
			idx, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			if !p.consumeOp("]") {
				return nil, p.err("expected ]")
			}
			e = &ex{kind: "index", recv: e, args: []*ex{idx}}
			continue
		}
		return e, nil
	}
}

func (p *parser) parsePrimary() (*ex, error) {
	p.skip()
	if p.i >= len(p.s) {
		return nil, p.err("expected expression")
	}
	if p.starts(`"""`) || (p.starts("pattern") && p.followedByPattern()) {
		return p.parsePatternExpr()
	}
	switch p.s[p.i] {
	case '"':
		s, err := p.parseString()
		if err != nil {
			return nil, err
		}
		return &ex{kind: "str", str: s}, nil
	case '(':
		p.i++
		e, err := p.parsePipe()
		if err != nil {
			return nil, err
		}
		if !p.consumeOp(")") {
			return nil, p.err("expected )")
		}
		return e, nil
	case '[':
		return p.parseList()
	case '{':
		return p.parseObject()
	case '.':
		p.i++
		if id, ok := p.peekIdent(); ok {
			p.i += len(id)
			return &ex{kind: "field", name: id}, nil
		}
		return &ex{kind: "dot"}, nil
	case '/':
		node, err := p.parseRegex()
		if err != nil {
			return nil, err
		}
		return &ex{kind: "rule", rule: regexRule(node)}, nil
	}
	if p.s[p.i] == '-' || (p.s[p.i] >= '0' && p.s[p.i] <= '9') {
		n, err := p.parseNumber()
		if err != nil {
			return nil, err
		}
		return &ex{kind: "num", num: n}, nil
	}
	id, ok := p.peekIdent()
	if !ok {
		return nil, p.err("expected expression")
	}
	p.i += len(id)
	switch id {
	case "true":
		return &ex{kind: "bool", b: true}, nil
	case "false":
		return &ex{kind: "bool", b: false}, nil
	case "null":
		return &ex{kind: "null"}, nil
	}
	p.skip()
	if p.i < len(p.s) && p.s[p.i] == '(' {
		p.i++
		args, err := p.parseArgList()
		if err != nil {
			return nil, err
		}
		return &ex{kind: "call", name: id, args: args}, nil
	}
	return &ex{kind: "ident", name: id}, nil
}

func (p *parser) followedByPattern() bool {
	save := p.i
	id, ok := p.peekIdent()
	if !ok || id != "pattern" {
		return false
	}
	p.i += len(id)
	p.skip()
	yes := p.starts(`"""`)
	p.i = save
	return yes
}

func (p *parser) parsePatternExpr() (*ex, error) {
	p.skip()
	if id, ok := p.peekIdent(); ok && id == "pattern" {
		p.i += len(id)
	}
	p.skip()
	if !p.starts(`"""`) {
		return nil, p.err("expected pattern string")
	}
	p.i += 3
	raw, err := p.rawUntil(`"""`)
	if err != nil {
		return nil, err
	}
	rule, err := compilePattern(dedentTemplate(raw))
	if err != nil {
		return nil, err
	}
	return &ex{kind: "rule", rule: rule}, nil
}

func (p *parser) parseList() (*ex, error) {
	p.i++
	if p.consumeOp("]") {
		return &ex{kind: "list"}, nil
	}
	var args []*ex
	for {
		e, err := p.parsePipe()
		if err != nil {
			return nil, err
		}
		args = append(args, e)
		if p.consumeOp("]") {
			break
		}
		if !p.consumeOp(",") {
			return nil, p.err("expected ,")
		}
	}
	return &ex{kind: "list", args: args}, nil
}

func (p *parser) parseObject() (*ex, error) {
	p.i++
	if p.consumeOp("}") {
		return &ex{kind: "object"}, nil
	}
	var keys []string
	var args []*ex
	for {
		key, err := p.objectKey()
		if err != nil {
			return nil, err
		}
		if !p.consumeOp(":") {
			return nil, p.err("expected :")
		}
		e, err := p.parsePipe()
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
		args = append(args, e)
		if p.consumeOp("}") {
			break
		}
		if !p.consumeOp(",") {
			return nil, p.err("expected ,")
		}
	}
	return &ex{kind: "object", keys: keys, args: args}, nil
}

func (p *parser) objectKey() (string, error) {
	p.skip()
	if p.i < len(p.s) && p.s[p.i] == '"' {
		return p.parseString()
	}
	return p.ident()
}

func (p *parser) parseArgList() ([]*ex, error) {
	if p.consumeOp(")") {
		return nil, nil
	}
	var args []*ex
	for {
		e, err := p.parsePipe()
		if err != nil {
			return nil, err
		}
		args = append(args, e)
		if p.consumeOp(")") {
			break
		}
		if !p.consumeOp(",") {
			return nil, p.err("expected ,")
		}
	}
	return args, nil
}

func (p *parser) parseNumber() (any, error) {
	start := p.i
	if p.s[p.i] == '-' {
		p.i++
	}
	if p.i >= len(p.s) || p.s[p.i] < '0' || p.s[p.i] > '9' {
		return nil, p.err("expected number")
	}
	for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
		p.i++
	}
	float := false
	if p.i < len(p.s) && p.s[p.i] == '.' {
		float = true
		p.i++
		for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
			p.i++
		}
	}
	lit := p.s[start:p.i]
	if float {
		f, err := strconv.ParseFloat(lit, 64)
		if err != nil {
			return nil, p.err("bad number")
		}
		return f, nil
	}
	n, err := strconv.ParseInt(lit, 10, 64)
	if err != nil {
		return nil, p.err("bad number")
	}
	return n, nil
}

func (p *parser) parseString() (string, error) {
	if p.i >= len(p.s) || p.s[p.i] != '"' {
		return "", p.err("expected string")
	}
	p.i++
	var b strings.Builder
	for p.i < len(p.s) {
		c := p.s[p.i]
		if c == '"' {
			p.i++
			return b.String(), nil
		}
		if c == '\\' {
			esc, n, err := decodeEscape(p.s[p.i:])
			if err != nil {
				return "", p.err(err.Error())
			}
			b.WriteString(esc)
			p.i += n
			continue
		}
		b.WriteByte(c)
		p.i++
	}
	return "", p.err("unterminated string")
}

func decodeEscape(s string) (string, int, error) {
	if len(s) < 2 || s[0] != '\\' {
		return "", 0, fmt.Errorf("bad escape")
	}
	switch s[1] {
	case 'n':
		return "\n", 2, nil
	case 'r':
		return "\r", 2, nil
	case 't':
		return "\t", 2, nil
	case '\\', '"', '\'':
		return s[1:2], 2, nil
	case 'u':
		if len(s) < 6 {
			return "", 0, fmt.Errorf("bad unicode escape")
		}
		n, err := strconv.ParseInt(s[2:6], 16, 32)
		if err != nil {
			return "", 0, fmt.Errorf("bad unicode escape")
		}
		return string(rune(n)), 6, nil
	default:
		return "", 0, fmt.Errorf("bad escape")
	}
}

func (p *parser) skip() {
	for p.i < len(p.s) {
		c := p.s[p.i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.i++
			continue
		}
		break
	}
}

func (p *parser) starts(s string) bool {
	return strings.HasPrefix(p.s[p.i:], s)
}

func (p *parser) consumeOp(s string) bool {
	p.skip()
	if p.i+len(s) > len(p.s) || p.s[p.i:p.i+len(s)] != s {
		return false
	}
	rest := p.s[p.i+len(s):]
	switch s {
	case "|":
		if strings.HasPrefix(rest, "|") {
			return false
		}
	case "=":
		if strings.HasPrefix(rest, ">") {
			return false
		}
	case "<", ">", "!":
		if strings.HasPrefix(rest, "=") {
			return false
		}
	case "&":
		if strings.HasPrefix(rest, "&") {
			return false
		}
	}
	p.i += len(s)
	return true
}

func (p *parser) consumeWord(w string) bool {
	p.skip()
	id, ok := p.peekIdent()
	if !ok || id != w {
		return false
	}
	save := p.i
	p.i += len(id)
	p.skip()
	call := p.i < len(p.s) && p.s[p.i] == '('
	p.i = save
	if call {
		return false
	}
	p.i += len(id)
	return true
}

func (p *parser) peekIdent() (string, bool) {
	p.skip()
	if p.i >= len(p.s) || !isIdentStart(p.s[p.i]) {
		return "", false
	}
	j := p.i + 1
	for j < len(p.s) && isIdentCont(p.s[j]) {
		j++
	}
	return p.s[p.i:j], true
}

func (p *parser) ident() (string, error) {
	id, ok := p.peekIdent()
	if !ok {
		return "", p.err("expected name")
	}
	p.i += len(id)
	return id, nil
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func (p *parser) err(msg string) *dataError {
	line, col := locAt(p.s, p.i)
	return &dataError{Code: "SYNTAX", Message: msg, Offset: p.i, Line: line, Column: col}
}

func (p *parser) rawUntil(end string) (string, error) {
	start := p.i
	idx := strings.Index(p.s[p.i:], end)
	if idx < 0 {
		return "", p.err("unterminated pattern")
	}
	p.i += idx + len(end)
	return p.s[start : start+idx], nil
}

func dedentTemplate(s string) string {
	if strings.HasPrefix(s, "\n") {
		s = s[1:]
	}
	if strings.HasSuffix(s, "\n") {
		s = strings.TrimSuffix(s, "\n")
	}
	lines := strings.Split(s, "\n")
	ind := -1
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		n := 0
		for n < len(ln) && (ln[n] == ' ' || ln[n] == '\t') {
			n++
		}
		if ind < 0 || n < ind {
			ind = n
		}
	}
	if ind > 0 {
		for i, ln := range lines {
			if strings.TrimSpace(ln) == "" {
				lines[i] = ""
				continue
			}
			if len(ln) >= ind {
				lines[i] = ln[ind:]
			}
		}
	}
	return strings.Join(lines, "\n")
}
