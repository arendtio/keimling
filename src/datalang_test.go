package main

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDataLangFiltersMCPResultWithoutTheOriginal(t *testing.T) {
	marker := strings.Repeat("UNIQUE-ISSUE-BODY-", 400)
	issues := []any{
		map[string]any{"number": int64(101), "title": "Crash beim Start", "state": "open", "labels": []any{"bug", "critical"}, "body": marker},
		map[string]any{"number": int64(102), "title": "Dokumentation ergänzen", "state": "open", "labels": []any{"documentation"}, "body": marker},
		map[string]any{"number": int64(103), "title": "Login-Fehler", "state": "closed", "labels": []any{"bug"}, "body": marker},
	}
	payload := map[string]any{"issues": issues}
	a := &Agent{}
	shown, err := a.toolResults().hold("json", payload, marker+marker)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(shown, "UNIQUE-ISSUE-BODY") || strings.Contains(shown, "Dokumentation") {
		t.Fatalf("model text contains the payload: %s", shown)
	}
	if !strings.Contains(shown, `"ref"`) || !strings.Contains(shown, "3 issues") {
		t.Fatalf("preview: %s", shown)
	}
	var env map[string]any
	if err := decodeArgs(shown, &env); err != nil {
		t.Fatal(err)
	}
	got := a.runDataTool(`{"input_ref":"` + env["ref"].(string) + `","code":"return input.issues | filter(.state == \"open\" && contains(.labels, \"bug\")) | map({id: .number, title: .title})","output_mode":"inline","timeout_ms":5000}`)
	if strings.Contains(got, "UNIQUE-ISSUE-BODY") || strings.Contains(got, "Dokumentation") || !strings.Contains(got, "Crash beim Start") {
		t.Fatalf("filtered: %s", got)
	}
}

func TestLargeMCPResultStaysOutOfTheJournal(t *testing.T) {
	j, err := OpenJournal(filepath.Join(t.TempDir(), "session.jsonl"), false)
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Journal: j, History: j.History}
	marker := strings.Repeat("UNIQUE-MCP-BODY-", 400)
	a.toolDataPending = &dataPending{kind: "json", value: map[string]any{
		"issues": []any{map[string]any{"title": marker, "state": "open"}},
	}}
	if err := a.recordToolResult(context.Background(), Item{Type: "function_call", Name: "tracker.search_issues", CallID: "c1", Arguments: `{"query":"is:open"}`}, marker, "", false); err != nil {
		t.Fatal(err)
	}
	raw := string(a.History[len(a.History)-1])
	if strings.Contains(raw, "UNIQUE-MCP-BODY") || !strings.Contains(raw, "r1") || !strings.Contains(raw, "1 issues") {
		t.Fatalf("journal kept the payload: %s", raw)
	}
}

func TestDataLangChainsReferences(t *testing.T) {
	a := &Agent{}
	first := a.runDataTool(`{"input_ref":"r0","code":"return 1","timeout_ms":1000}`)
	if !strings.Contains(first, "INVALID_REF") {
		t.Fatal(first)
	}
	text, err := a.toolResults().hold("json", []any{map[string]any{"id": int64(7), "title": "A"}, map[string]any{"id": int64(8), "title": "B"}}, "small")
	if err != nil {
		t.Fatal(err)
	}
	ref := strings.TrimPrefix(text, "small\nref: ")
	second := a.runDataTool(`{"input_ref":"` + ref + `","code":"return input | map(.id)","output_mode":"inline","timeout_ms":1000}`)
	if !strings.Contains(second, "7") || !strings.Contains(second, `"ref"`) {
		t.Fatal(second)
	}
	var env map[string]any
	if err := decodeArgs(second, &env); err != nil {
		t.Fatal(err)
	}
	third := a.runDataTool(`{"input_ref":"` + env["ref"].(string) + `","code":"return input | take(1)","output_mode":"inline","timeout_ms":1000}`)
	if !strings.Contains(third, "7") || strings.Contains(third, "8") {
		t.Fatal(third)
	}
}

func TestDataLangParsesCLIGrammar(t *testing.T) {
	value, err := runDataLang(`return extract(input.stdout, pattern"""PID {pid:int} CPU {cpu:decimal}%""") | filter(.cpu > 5)`, map[string]any{
		"stdout":    "PID 12 CPU 5.4%\nPID 15 CPU 0.7%\nPID 27 CPU 12.1%",
		"stderr":    "",
		"exit_code": 0,
	}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	rows, ok := value.([]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("%#v", value)
	}
	first := rows[0].(map[string]any)
	if first["pid"] != int64(12) || first["cpu"] != 5.4 {
		t.Fatalf("%#v", first)
	}
}

func TestDataLangParsesRecursiveNodes(t *testing.T) {
	code := `
grammar {
  ws = /[ \t\r\n]*/
  word = /[a-z]+/
  node = ws (
      "(" children:node* ws ")"
    | value:word
  )
}
return parse(input, node)`
	value, err := runDataLang(code, "(a (b c) (d (e f)))", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	root := value.(map[string]any)
	children := root["children"].([]any)
	if len(children) != 3 {
		t.Fatalf("%#v", value)
	}
	if children[0].(map[string]any)["value"] != "a" {
		t.Fatalf("%#v", children[0])
	}
	inner := children[1].(map[string]any)["children"].([]any)
	if inner[0].(map[string]any)["value"] != "b" || inner[1].(map[string]any)["value"] != "c" {
		t.Fatalf("%#v", inner)
	}
}

func TestDataLangCSVAndXML(t *testing.T) {
	rows, err := runDataLang(`return csv(input)`, "name,title\nAda,Crash\n", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got := rows.([]any)[0].(map[string]any)
	if got["name"] != "Ada" || got["title"] != "Crash" {
		t.Fatalf("%#v", got)
	}
	tree, err := runDataLang(`return xml(input)`, `<issues><issue id="1">Crash</issue></issues>`, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	root := tree.(map[string]any)
	if root["tag"] != "issues" {
		t.Fatalf("%#v", root)
	}
	issue := root["children"].([]any)[0].(map[string]any)
	if issue["attrs"].(map[string]any)["id"] != "1" {
		t.Fatalf("%#v", issue)
	}
	if _, err := runDataLang(`return xml(input)`, `<!DOCTYPE foo SYSTEM "http://127.0.0.1:1/x"><foo/>`, time.Second); err == nil {
		t.Fatal("doctype should fail")
	}
}

func TestDataLangMultilineTemplate(t *testing.T) {
	code := "grammar {\nrecord = pattern\"\"\"\nName: {name:until(\"\\n\")}\nStatus: {status:word}\nCount: {count:int}\n\"\"\"\n}\nreturn parse(input, record)"
	value, err := runDataLang(code, "Name: Ada\nStatus: open\nCount: 3", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	obj := value.(map[string]any)
	if obj["name"] != "Ada" || obj["status"] != "open" || obj["count"] != int64(3) {
		t.Fatalf("%#v", obj)
	}
}

func TestDataLangDynamicTag(t *testing.T) {
	code := `
grammar {
  ident = /[a-z0-9]+/
  textNode = value:until("<")
  element = "<" tag:ident ">" children:(element | textNode)* "</" =tag ">"
}
return parse(input, element)`
	value, err := runDataLang(code, "<a><b>x</b></a>", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	root := value.(map[string]any)
	if root["tag"] != "a" {
		t.Fatalf("%#v", root)
	}
	if _, err := runDataLang(code, "<a>x</b>", time.Second); err == nil {
		t.Fatal("mismatched tag should fail")
	}
}

func TestDataLangReportsCorrectableErrors(t *testing.T) {
	_, err := runDataLang(`return input | where(.status == "ok")`, []any{}, time.Second)
	de, ok := err.(*dataError)
	if !ok || de.Code != "UNKNOWN_FUNCTION" || !strings.Contains(de.Suggestion, "filter") {
		t.Fatalf("%v", err)
	}
	_, err = runDataLang("return [", nil, time.Second)
	de, ok = err.(*dataError)
	if !ok || de.Code != "SYNTAX" || de.Line == 0 || de.Column == 0 {
		t.Fatalf("%v", err)
	}
	_, err = runDataLang(`return parse("abc", pattern"""Name: {name:int}""")`, nil, time.Second)
	de, ok = err.(*dataError)
	if !ok || de.Code != "PARSE_ERROR" || de.Expected == "" {
		t.Fatalf("%v", err)
	}
}

func TestDataLangTimesOut(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 200000; i++ {
		b.WriteByte('a')
	}
	_, err := runDataLang("grammar { bit = /a/ \n bits = bit* }\nreturn parse(input, bits)", b.String(), time.Millisecond)
	de, ok := err.(*dataError)
	if !ok || de.Code != "TIMEOUT" {
		t.Fatalf("%v", err)
	}
}

func TestDataLangHasNoHostAccess(t *testing.T) {
	for _, code := range []string{
		`return read("/etc/passwd")`,
		`return exec("rm -rf /")`,
		`return dial("127.0.0.1")`,
	} {
		_, err := runDataLang(code, nil, time.Second)
		de, ok := err.(*dataError)
		if !ok || de.Code != "UNKNOWN_FUNCTION" {
			t.Fatalf("%s: %v", code, err)
		}
	}
}

func TestDataLangRejectsForeignAndExpiredRefs(t *testing.T) {
	a := &Agent{}
	other := newDataStore()
	it, err := other.put("text", "secret")
	if err != nil {
		t.Fatal(err)
	}
	got := a.runDataTool(`{"input_ref":"` + it.id + `","code":"return input","timeout_ms":1000}`)
	if !strings.Contains(got, "INVALID_REF") {
		t.Fatal(got)
	}
	store := newDataStore()
	store.maxItems = 1
	first, err := store.put("text", "one")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.put("text", "two"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.get(first.id); err == nil {
		t.Fatal("evicted ref still resolved")
	}
}

func TestDataLangInlineDoesNotTruncate(t *testing.T) {
	a := &Agent{}
	big := strings.Repeat("x", dataModelLimit+10)
	text, err := a.toolResults().hold("text", big, "tiny")
	if err != nil {
		t.Fatal(err)
	}
	ref := strings.TrimPrefix(text, "tiny\nref: ")
	got := a.runDataTool(`{"input_ref":"` + ref + `","code":"return input","output_mode":"inline","timeout_ms":1000}`)
	if !strings.Contains(got, "RESULT_TOO_LARGE") || strings.Contains(got, big[:100]) {
		clip := got
		if len(clip) > 200 {
			clip = clip[:200]
		}
		t.Fatalf("inline of a large value must fail with the ref, not a clipped copy: %s", clip)
	}
	refOnly := a.runDataTool(`{"input_ref":"` + ref + `","code":"return input","output_mode":"ref","timeout_ms":1000}`)
	if strings.Contains(refOnly, big) || !strings.Contains(refOnly, `"ref"`) || strings.Contains(refOnly, `"value"`) {
		t.Fatal(refOnly)
	}
}

func TestDataLangExtractRejectsEmptyMatch(t *testing.T) {
	_, err := runDataLang("grammar { empty = \"\" }\nreturn extract(input, empty)", "abc", time.Second)
	de, ok := err.(*dataError)
	if !ok || de.Code != "EMPTY_MATCH" {
		t.Fatalf("%v", err)
	}
}

func TestDataLangRejectsLeftRecursion(t *testing.T) {
	_, err := runDataLang("grammar { loop = loop }\nreturn parse(input, loop)", "a", time.Second)
	de, ok := err.(*dataError)
	if !ok || de.Code != "LEFT_RECURSION" {
		t.Fatalf("%v", err)
	}
}

func TestDataLangTimeoutFollowsTheInteger(t *testing.T) {
	if dataTimeout(0) != 30*time.Second {
		t.Fatalf("default %s", dataTimeout(0))
	}
	if dataTimeout(45000) != 45*time.Second {
		t.Fatalf("45s was shortened to %s", dataTimeout(45000))
	}
	if dataTimeout(math.MaxInt64) != time.Duration(math.MaxInt64) {
		t.Fatal("the top of the integer range was capped")
	}
	a := &Agent{}
	text, err := a.toolResults().hold("json", []any{int64(1)}, "one")
	if err != nil {
		t.Fatal(err)
	}
	ref := strings.TrimPrefix(text, "one\nref: ")
	got := a.runDataTool(`{"input_ref":"` + ref + `","inputs":[],"code":"return input","timeout_ms":45000,"output_mode":"inline"}`)
	if strings.Contains(got, "error") || !strings.Contains(got, "1") {
		t.Fatal(got)
	}
}

func TestDataLangJoinsNamedInputs(t *testing.T) {
	a := &Agent{}
	orders, err := a.toolResults().hold("json", []any{
		map[string]any{"id": int64(1), "user": int64(7), "title": "Crash"},
		map[string]any{"id": int64(2), "user": int64(8), "title": "Docs"},
	}, "orders")
	if err != nil {
		t.Fatal(err)
	}
	users, err := a.toolResults().hold("json", []any{
		map[string]any{"id": int64(7), "name": "Ada"},
		map[string]any{"id": int64(9), "name": "Bea"},
	}, "users")
	if err != nil {
		t.Fatal(err)
	}
	done, err := a.toolResults().hold("json", []any{int64(1)}, "done")
	if err != nil {
		t.Fatal(err)
	}
	orderRef := strings.TrimPrefix(orders, "orders\nref: ")
	userRef := strings.TrimPrefix(users, "users\nref: ")
	doneRef := strings.TrimPrefix(done, "done\nref: ")
	raw, err := json.Marshal(map[string]any{
		"input_ref":   orderRef,
		"inputs":      []map[string]string{{"name": "users", "ref": userRef}, {"name": "done", "ref": doneRef}},
		"code":        `return input | filter(contains(done, .id)) | map({id: .id, name: lookup(users, "id", .user).name})`,
		"output_mode": "inline",
		"timeout_ms":  1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := a.runDataTool(string(raw))
	if strings.Contains(got, "Docs") || strings.Contains(got, "Bea") || !strings.Contains(got, "Ada") {
		t.Fatal(got)
	}
	missing := a.runDataTool(`{"input_ref":"","inputs":[{"name":"users","ref":"r9"}],"code":"return users","timeout_ms":1000}`)
	if !strings.Contains(missing, "INVALID_REF") {
		t.Fatal(missing)
	}
	dup := a.runDataTool(`{"input_ref":"` + orderRef + `","inputs":[{"name":"input","ref":"` + userRef + `"}],"code":"return input","timeout_ms":1000}`)
	if !strings.Contains(dup, "duplicate input name") {
		t.Fatal(dup)
	}
}

func TestDataLangScansAFilePastTheOldCap(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	f, err := os.Create("phone.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("name,phone\nAda,555-SECRET\n"); err != nil {
		t.Fatal(err)
	}
	filler := "Other,FILLER-MARKER-" + strings.Repeat("x", 180) + "\n"
	var written int
	var rows int64 = 1
	for written < 8<<20 {
		if _, err := f.WriteString(filler); err != nil {
			t.Fatal(err)
		}
		written += len(filler)
		rows++
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat("phone.csv")
	if err != nil || st.Size() <= 8<<20 {
		t.Fatalf("fixture size %v %v", st, err)
	}
	a := &Agent{}
	code := `return csv(input) | filter(.name == "Ada") | first()`
	got := a.runDataTool(`{"input_ref":"phone.csv","inputs":[],"code":` + jsonString(code) + `,"timeout_ms":10000,"output_mode":"inline"}`)
	if !strings.Contains(got, "Ada") || !strings.Contains(got, "555-SECRET") || strings.Contains(got, "FILLER-MARKER") || strings.Contains(got, "larger than 8 MiB") {
		t.Fatalf("search: %s", got)
	}
	code = `return lookup(csv(input), "name", "Ada")`
	got = a.runDataTool(`{"input_ref":"phone.csv","inputs":[],"code":` + jsonString(code) + `,"timeout_ms":10000,"output_mode":"inline"}`)
	if !strings.Contains(got, "555-SECRET") || strings.Contains(got, "FILLER-MARKER") {
		t.Fatalf("lookup: %s", got)
	}
	code = `return outline(input)`
	got = a.runDataTool(`{"input_ref":"phone.csv","inputs":[],"code":` + jsonString(code) + `,"timeout_ms":10000,"output_mode":"inline"}`)
	if !strings.Contains(got, "name") || !strings.Contains(got, "phone") || !strings.Contains(got, "n="+strconv.FormatInt(rows, 10)) || strings.Contains(got, "555-SECRET") || strings.Contains(got, "FILLER-MARKER") {
		t.Fatalf("outline rows %d: %s", rows, got)
	}
	got = a.runDataTool(`{"input_ref":"phone.csv","inputs":[],"code":"return input","timeout_ms":1000,"output_mode":"inline"}`)
	if !strings.Contains(got, "large") || strings.Contains(got, "FILLER-MARKER") {
		t.Fatalf("whole file: %s", got)
	}
}

func TestDataLangReadsAFileAndAHistorySection(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	if err := os.WriteFile("notes.txt", []byte("alpha\nbeta\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := &Agent{}
	got := a.runDataTool(`{"input_ref":"notes.txt","inputs":[],"code":"return input","timeout_ms":1000,"output_mode":"inline"}`)
	if !strings.Contains(got, "alpha") || !strings.Contains(got, "beta") {
		t.Fatalf("file: %s", got)
	}
	denied := a.runDataTool(`{"input_ref":"notes.txt","inputs":[],"code":"return read_file(input)","timeout_ms":1000}`)
	if !strings.Contains(denied, "unknown") && !strings.Contains(strings.ToLower(denied), "error") {
		t.Fatalf("program touched the file itself: %s", denied)
	}
	path := filepath.Join(t.TempDir(), "session.jsonl")
	j, err := OpenJournal(path, false)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"role": "user", "content": "UNIQUE-HISTORY-SENTENCE"})
	if err := j.AppendItem(raw); err != nil {
		t.Fatal(err)
	}
	j.Close()
	a.Cfg.SessionPath = path
	got = a.runDataTool(`{"input_ref":"task-1","inputs":[],"code":"return input","timeout_ms":1000,"output_mode":"inline"}`)
	if !strings.Contains(got, "UNIQUE-HISTORY-SENTENCE") {
		t.Fatalf("history: %s", got)
	}
}

func TestDataLangHelpAndToolDescription(t *testing.T) {
	help := dataHelp(`{"topic":"parser"}`)
	for _, want := range []string{"=name", "pattern", "lookahead", "left recursion"} {
		if !strings.Contains(strings.ToLower(help), strings.ToLower(want)) {
			t.Fatalf("help missing %s", want)
		}
	}
	if !strings.Contains(dataRunDescription, "input_ref") || !strings.Contains(dataRunDescription, "filter(contains(.labels, \\\"bug\\\"))") {
		t.Fatal("tool description lost the worked example")
	}
	if !strings.Contains(dataRunDescription, "lookup") || !strings.Contains(dataRunDescription, "30000") || !strings.Contains(dataRunDescription, "inputs") {
		t.Fatal("tool description lost named inputs or the 30s default")
	}
	if !strings.Contains(sharedWorkInstructions(), "Use data_processing") {
		t.Fatal("work prompt missing data_processing")
	}
}

func TestToolNamesMatchTheResponsesAPI(t *testing.T) {
	ok := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	for _, web := range []bool{false, true} {
		for i, tool := range definitions(web) {
			name, _ := tool["name"].(string)
			if !ok.MatchString(name) {
				t.Fatalf("tools[%d].name %q", i, name)
			}
		}
	}
}

func TestDataLangAcceptsAndOrNot(t *testing.T) {
	log := "fix opcache\nremove compose\nhello\nupgrade nextcloud\n"
	code := `return split(input.log, "\n") | filter(contains(., "opcache") or contains(., "compose") or contains(., "nextcloud"))`
	value, err := runDataLang(code, map[string]any{"log": log}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	rows, ok := value.([]any)
	if !ok || len(rows) != 3 || rows[2] != "upgrade nextcloud" {
		t.Fatalf("%#v", value)
	}
	empty, err := runDataLang(`return split("a\nb", "\n") | filter(contains(., "a") and not contains(., "b"))`, nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got := empty.([]any)
	if len(got) != 1 || got[0] != "a" {
		t.Fatalf("%#v", got)
	}
}

func TestDataLangExtractsWithARegexValue(t *testing.T) {
	log := "0bf859e add check-after-reboot.md\ncf7ff7f arendt.cloud: fix opcache\na46d8d9 upgrade to nextcloud 32\n"
	value, err := runDataLang("return extract(input.log, /[^\\n]*(?:fix|upgrade|repair|scripts)[^\\n]*/)", map[string]any{"log": log}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	rows, ok := value.([]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("%#v", value)
	}
	if rows[0] != "cf7ff7f arendt.cloud: fix opcache" || rows[1] != "a46d8d9 upgrade to nextcloud 32" {
		t.Fatalf("%#v", rows)
	}
	n, err := runDataLang("return 8 / 2", nil, time.Second)
	if err != nil || n != int64(4) {
		t.Fatalf("division %v %v", n, err)
	}
}

func TestDataProcessingCardShowsTheProgram(t *testing.T) {
	code := "return extract(input.log, /[^\\n]*(?:fix|upgrade)[^\\n]*/) | map(.) " + strings.Repeat(" ", 80)
	args, err := json.Marshal(map[string]any{
		"input_ref":   "r1",
		"inputs":      []any{},
		"code":        code,
		"output_mode": "inline",
		"timeout_ms":  30000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) < 160 {
		t.Fatal("fixture is shorter than the old cutoff")
	}
	label := toolCommandLabel("data_processing", string(args))
	if !strings.Contains(label, "input_ref=r1") || !strings.Contains(label, code) {
		t.Fatal(label)
	}
}

func TestDataLangDocumentedFilterExample(t *testing.T) {
	value, err := runDataLang(`return input.issues | filter(contains(.labels, "bug")) | map(.number)`, map[string]any{
		"issues": []any{
			map[string]any{"number": int64(101), "state": "open", "labels": []any{"bug"}},
			map[string]any{"number": int64(102), "state": "open", "labels": []any{"docs"}},
		},
	}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	nums := value.([]any)
	if len(nums) != 1 || nums[0] != int64(101) {
		t.Fatalf("%#v", value)
	}
}

func TestOutlineShowsShapeWithoutValues(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"items":[`)
	for i := 0; i < 40; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"sku":"MARKER-SECRET","qty":1}`)
	}
	b.WriteString(`]}`)
	value, err := runDataLang("return outline(input)", b.String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := value.(string)
	for _, want := range []string{"items", "array", "sku", "qty"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s in %s", want, got)
		}
	}
	if strings.Contains(got, "MARKER-SECRET") || len(got) > dataModelLimit {
		t.Fatalf("shape leaked or grew: %d %s", len(got), got)
	}
	piped, err := runDataLang("return input | outline()", b.String(), time.Second)
	if err != nil || piped != value {
		t.Fatalf("pipe %v %v", piped, err)
	}

	xml := `<stock><item id="a">MARKER-SECRET</item><item id="b">other</item></stock>`
	value, err = runDataLang("return outline(input)", xml, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got, _ = value.(string)
	if !strings.Contains(got, "stock") || !strings.Contains(got, "item") || strings.Contains(got, "MARKER-SECRET") {
		t.Fatal(got)
	}

	csvText := "sku,qty\nMARKER-SECRET,1\npen,2\n"
	value, err = runDataLang("return outline(input)", csvText, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got, _ = value.(string)
	if !strings.Contains(got, "sku") || !strings.Contains(got, "qty") || !strings.Contains(got, "n=2") || strings.Contains(got, "MARKER-SECRET") {
		t.Fatal(got)
	}
}
