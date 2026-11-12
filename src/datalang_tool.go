package main

import (
	"bufio"
	"context"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const dataRunDescription = `Execute DataLang to filter, parse, aggregate or transform
previous tool outputs. Use this instead of Python/shell
for data processing.

INPUT
input_ref: reference returned by an MCP, CLI or data_processing call. Empty string when every value is named in inputs.
inputs: [{name, ref}, ...] further results. Each name becomes a variable. Use [] when only input_ref is needed.
code: DataLang program
timeout_ms?: execution timeout in milliseconds (default 30000). Any positive integer; there is no shorter cap.
output_mode?: auto | inline | ref (default auto)

The input_ref value is available as ` + "`input`" + `.
Never copy large tool outputs into code; pass references.

SYNTAX
let x = expression
return expression

Values: null, bool, number, string, array, object.
Access: .field, object.field, [index].
Pipelines: value | operation(args).
Inside filter/map/sort, ` + "`.`" + ` is the current item.
Tests combine with &&, ||, and, or, and not.

OPERATIONS
filter(predicate), map(expression), sort(key), group(key),
take(n), count(), first(), unique(), contains(value, item),
lookup(rows, field, key), split(text, sep), join(array, sep),
trim(text), json(text), csv(text), xml(text).
lookup returns the first object whose field equals key, or null.

EXAMPLE — FILTER AN MCP RESULT
Suppose an issue-search tool returned ref "r1" with:
{
  "issues": [
    {"number": 101, "state": "open", "labels": ["bug"]},
    {"number": 102, "state": "open", "labels": ["docs"]}
  ]
}

Call data_processing with:
{
  "input_ref": "r1",
  "code": "return input.issues | filter(contains(.labels, \"bug\")) | map(.number)"
}

Result: [101]

EXAMPLE — LOOK UP AND COMPARE
{
  "input_ref": "r1",
  "inputs": [{"name": "users", "ref": "r2"}, {"name": "done", "ref": "r3"}],
  "code": "return input | filter(contains(done, .id)) | map({id: .id, name: lookup(users, \"id\", .user).name})"
}

TEXT PARSING
extract(text, /regex/) returns each match. /regex/ is a value.
parse(text, rule) must match the whole text.
extract(text, rule) returns non-overlapping matches.

Grammar, also valid as a rule passed to parse or extract:
  "abc"        literal
  /[a-z]+/     regex
  char(10)     character
  A B          sequence
  A | B        ordered alternative
  A?, A*, A+   repetition
  name:A       named capture
  rule = ...   reusable/recursive rule

Captures become object fields. Repeated captures become
arrays. Uncaptured delimiters are discarded.

Example for CLI stdout:
return extract(input.stdout,
  pattern"""PID {pid:int} CPU {cpu:decimal}%""")

For recursive grammars, lookahead, backreferences, multiline
templates and detailed capture semantics, call
data_processing_help(topic="parser").

MCP
mcp(name, args) calls one connected MCP tool and returns its value.
name is the exposed tool name, written as a string. args is an object.
all(mcp("mcp__a__one", {}), mcp("mcp__b__two", {})) runs those calls together.
The program is approved once before it starts, from those string names.
A name computed while the program runs is not called.
A program that only calls MCP tools may leave input_ref empty and inputs [].

OUTPUT
Small results are returned inline. Large results receive
a new reference for subsequent data_processing calls.`

const dataHelpParser = `DataLang grammar

Operators
  "abc"       literal
  /[a-z]+/    regular expression, matched at the current position
  char(10)    one Unicode code point
  A B         sequence
  A | B       ordered alternative
  A? A* A+    optional, zero or more, one or more
  A{2,5}      bounded repetition
  &A          positive lookahead, consumes nothing
  !A          negative lookahead, consumes nothing
  name:A      named capture
  =name       match the text previously captured as name, in this rule call
  text(A)     the original matched text
  sepBy(A, B) one or more A separated by B
  until("X")  text up to but not including X

Primitives
  int         integer
  decimal     number with a fractional part
  word        letters

Rules
  name = expression
  name = expression => value

Captures
  Named captures are fields of an object.
  A named repetition is an array. Each iteration has its own fields.
  An optional capture is the value or null.
  Nested rules produce nested objects.
  Unnamed pieces are not fields.
  A rule does not merge its fields into the caller. Capture the rule if you want its value.
  An alternative returns only the fields of the branch that matched.
  Setting the same capture twice in one sequence is an error.
  => evaluates a DataLang expression over the captures and becomes the rule result.
  Without =>, named captures make an object. A rule with no captures returns the matched text.

Recursion
  Rules may call themselves or each other after consuming input.
  A rule that calls itself at the same position is rejected as left recursion.

Lookahead and backreferences
  &A and !A test without moving.
  =name matches the text bound to name in this rule invocation, so nested tags can repeat the opener.

Templates
  pattern"""
  Name: {name:until("\n")}
  Status: {status:word}
  """
  The first and last pure delimiter lines are removed and the common indent is stripped.
  {name:rule} inserts a capture. {{ and }} are literal braces. \n and \u000A are escapes.
  A template is the same grammar, not a second language.

parse and extract
  parse(text, rule) must consume the entire text.
  extract(text, rule) returns every non-overlapping match.
  A match that consumes nothing is an error.

Example
  grammar {
    ws = /[ \t\r\n]*/
    word = /[a-z]+/
    node = ws ( "(" children:node* ws ")" | value:word )
  }
  return parse(input, node)

Errors include the offset, line, column, and the tokens that were expected.

mcp(name, args) calls one MCP tool. name is a string literal.
all(...) evaluates each argument at the same time.
A tool name that is not a string literal in the program is not called.`

func dataLangTools() []map[string]any {
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	return []map[string]any{
		toolDef("data_processing", dataRunDescription, map[string]any{
			"input_ref": str("Reference returned by an earlier MCP, CLI, or data_processing result. Empty string when every value is named in inputs."),
			"inputs": map[string]any{
				"type":        "array",
				"description": "Named extra references. Use an empty array when only input_ref is needed. Each name becomes a variable.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": str("Variable name"),
						"ref":  str("Result reference"),
					},
					"required":             []string{"name", "ref"},
					"additionalProperties": false,
				},
			},
			"code":        str("DataLang program. input_ref is `input`. Each inputs name is a variable."),
			"timeout_ms":  map[string]any{"type": "integer", "description": "Execution timeout in milliseconds. Default 30000. Any positive integer."},
			"output_mode": str("auto, inline, or ref. Default auto."),
		}, "input_ref", "inputs", "code", "timeout_ms", "output_mode"),
		toolDef("data_processing_help", "Return the DataLang grammar documentation. topic must be \"parser\".", map[string]any{
			"topic": str("Documentation topic. Use parser."),
		}, "topic"),
	}
}

type dataPending struct {
	kind  string
	value any
}

func (a *Agent) toolResults() *dataStore {
	if a == nil {
		return newDataStore()
	}
	if a.toolData == nil {
		a.toolData = newDataStore()
	}
	return a.toolData
}

func (a *Agent) runDataTool(args string) string {
	return a.runDataProgram(context.Background(), nil, args)
}

func (a *Agent) runDataProgram(ctx context.Context, reader *bufio.Reader, args string) string {
	var in struct {
		InputRef string `json:"input_ref"`
		Inputs   []struct {
			Name string `json:"name"`
			Ref  string `json:"ref"`
		} `json:"inputs"`
		Code       string `json:"code"`
		TimeoutMS  int64  `json:"timeout_ms"`
		OutputMode string `json:"output_mode"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return dataErrorJSON(dataErr("SYNTAX", err.Error()))
	}
	prog, err := parseProgram(in.Code)
	if err != nil {
		return dataErrorJSON(err)
	}
	names, calls := mcpProgramNames(prog)
	vars, err := a.dataVars(in.InputRef, in.Inputs, calls)
	if err != nil {
		return dataErrorJSON(err)
	}
	if calls {
		ok, err := a.allowMCPProgram(ctx, reader, names)
		if err != nil {
			return err.Error()
		}
		if !ok {
			return "DENIED by user"
		}
	}
	host := &dlHost{allowed: map[string]bool{}}
	for _, name := range names {
		host.allowed[name] = true
	}
	host.call = func(name string, args any) (any, error) {
		raw, err := json.Marshal(args)
		if err != nil {
			return nil, dataErr("MCP", err.Error())
		}
		done := a.callMCPTool(ctx, name, string(raw))
		if !done.stored {
			msg := strings.TrimSpace(done.display)
			if msg == "" {
				msg = "MCP call failed"
			}
			return nil, dataErr("MCP", msg)
		}
		return done.value, nil
	}
	value, err := evalDataLangHost(in.Code, vars, dataTimeout(in.TimeoutMS), host)
	if err != nil {
		return dataErrorJSON(err)
	}
	text, err := a.toolResults().present(value, in.OutputMode)
	if err != nil {
		return dataErrorJSON(err)
	}
	return text
}

func (a *Agent) dataVars(primary string, extras []struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
}, allowEmpty bool) (map[string]any, error) {
	vars := map[string]any{}
	if strings.TrimSpace(primary) != "" {
		item, err := a.toolResults().get(primary)
		if err != nil {
			return nil, err
		}
		if item.kind == "unsupported" {
			return nil, dataErr("UNSUPPORTED_INPUT", "binary tool output is not interpreted")
		}
		vars["input"] = item.value
	}
	seen := map[string]bool{}
	for _, extra := range extras {
		name := strings.TrimSpace(extra.Name)
		if !dataIdent(name) || name == "true" || name == "false" || name == "null" {
			return nil, &dataError{Code: "SYNTAX", Message: "invalid input name " + name, Suggestion: "Use a name of letters, digits, and underscores."}
		}
		if _, exists := vars[name]; exists || seen[name] {
			return nil, &dataError{Code: "SYNTAX", Message: "duplicate input name " + name}
		}
		seen[name] = true
		item, err := a.toolResults().get(extra.Ref)
		if err != nil {
			return nil, err
		}
		if item.kind == "unsupported" {
			return nil, dataErr("UNSUPPORTED_INPUT", "binary tool output is not interpreted")
		}
		vars[name] = item.value
	}
	if len(vars) == 0 && !allowEmpty {
		return nil, dataErr("INVALID_REF", "missing result reference")
	}
	return vars, nil
}

func dataProcessingLabel(name, args string) string {
	if name == "data_processing_help" {
		var topic struct {
			Topic string `json:"topic"`
		}
		if json.Unmarshal([]byte(args), &topic) != nil || strings.TrimSpace(topic.Topic) == "" {
			return name
		}
		return name + " " + strings.TrimSpace(topic.Topic)
	}
	var in struct {
		InputRef string `json:"input_ref"`
		Inputs   []struct {
			Name string `json:"name"`
			Ref  string `json:"ref"`
		} `json:"inputs"`
		Code       string `json:"code"`
		OutputMode string `json:"output_mode"`
	}
	if json.Unmarshal([]byte(args), &in) != nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(name)
	if ref := strings.TrimSpace(in.InputRef); ref != "" {
		b.WriteString(" input_ref=")
		b.WriteString(ref)
	}
	for _, extra := range in.Inputs {
		n := strings.TrimSpace(extra.Name)
		ref := strings.TrimSpace(extra.Ref)
		if n == "" && ref == "" {
			continue
		}
		b.WriteString(" ")
		b.WriteString(n)
		b.WriteString("=")
		b.WriteString(ref)
	}
	if mode := strings.TrimSpace(in.OutputMode); mode != "" && mode != "auto" {
		b.WriteString(" output_mode=")
		b.WriteString(mode)
	}
	if strings.TrimSpace(in.Code) != "" {
		b.WriteString("\n")
		b.WriteString(in.Code)
	}
	return b.String()
}

func dataIdent(name string) bool {
	if name == "" || !isIdentStart(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !isIdentCont(name[i]) {
			return false
		}
	}
	return true
}

func dataHelp(args string) string {
	var in struct {
		Topic string `json:"topic"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return dataErrorJSON(dataErr("SYNTAX", err.Error()))
	}
	if strings.TrimSpace(in.Topic) != "parser" {
		return dataErrorJSON(&dataError{Code: "SYNTAX", Message: "unknown help topic", Suggestion: `use topic "parser"`})
	}
	return dataHelpParser
}

func (a *Agent) attachDataRef(name, result string) string {
	if a == nil {
		return result
	}
	if name == "data_processing" || name == "data_processing_help" {
		a.toolDataPending = nil
		return result
	}
	if p := a.toolDataPending; p != nil {
		a.toolDataPending = nil
		text, err := a.toolResults().hold(p.kind, p.value, result)
		if err != nil {
			return result
		}
		return text
	}
	if name == "exec" {
		text, err := a.toolResults().hold("json", normalizeExec(result), result)
		if err != nil {
			return result
		}
		return text
	}
	return result
}

func mcpCallOutcome(result *mcp.CallToolResult) (display, kind string, value any, ok bool) {
	display = formatMCPResult(result)
	if result == nil {
		return display, "", nil, false
	}
	if result.IsError {
		return display, "", nil, false
	}
	if result.StructuredContent != nil {
		canon, _, err := canonValue(result.StructuredContent)
		if err != nil {
			return display, "text", display, true
		}
		return display, "json", canon, true
	}
	var texts []string
	var blocks []any
	binary := false
	for _, c := range result.Content {
		switch item := c.(type) {
		case mcp.TextContent:
			texts = append(texts, item.Text)
			blocks = append(blocks, map[string]any{"type": "text", "text": item.Text})
		case mcp.ImageContent, mcp.AudioContent:
			binary = true
			blocks = append(blocks, map[string]any{"type": "binary"})
		default:
			text := mcp.GetTextFromContent(c)
			if strings.TrimSpace(text) == "" {
				binary = true
				blocks = append(blocks, map[string]any{"type": "binary"})
				continue
			}
			texts = append(texts, text)
			blocks = append(blocks, map[string]any{"type": "text", "text": text})
		}
	}
	if binary && len(texts) == 0 {
		return display, "unsupported", nil, true
	}
	if len(blocks) == 1 && !binary {
		return display, "text", texts[0], true
	}
	if len(blocks) > 1 {
		return display, "json", blocks, true
	}
	return display, "text", display, true
}
