package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceTableNeedsAnIntentionalInline(t *testing.T) {
	raw, err := json.Marshal(serviceTableValue())
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) <= 4096 || strings.Contains(string(raw[:1024]), serviceLateHost) {
		t.Fatalf("service table %d", len(raw))
	}
	if strings.Count(string(raw), serviceLateHost) != 1 || strings.Count(string(raw), serviceEarlyName) != 1 {
		t.Fatal("names are not unique")
	}
	spec, err := loadSpec(filepath.Join("..", "service-table", "scenario.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(spec.Task, serviceLateHost) {
		t.Fatal("the task names the host")
	}
}

func TestTasksStayOutOfTheShippingSources(t *testing.T) {
	src := filepath.Join("..", "..", "src")
	needles := []string{phoneNumber, dockCode, otherLine, "welche Nummer gehört zu Ada", plainDock, coverCode, refuseCode, copiedCmd, roleAnswer, exportName, skuWant, binAnswer, serviceLateHost, serviceLateName}
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, n := range needles {
			if strings.Contains(string(b), n) {
				t.Errorf("%s contains scenario text %q", path, n)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPhoneBookHidesTheNumber(t *testing.T) {
	body := phoneBookCSV()
	if len(body) <= 4096 {
		t.Fatalf("csv is %d bytes", len(body))
	}
	if strings.Contains(string(body[:1024]), phoneNumber) {
		t.Fatal("preview contains Ada's number")
	}
}

func TestUnknownShapeHidesTheCodes(t *testing.T) {
	body := unknownShapeFile()
	if len(body) <= 4096 || strings.Contains(string(body[:1024]), dockCode) || strings.Contains(string(body[:1024]), decoyCode) {
		t.Fatalf("len %d", len(body))
	}
}

func TestScoreKeepsALongerRightPath(t *testing.T) {
	spec := scenarioSpec{Needles: []string{phoneNumber}, AnswerExcludes: []string{fillerMark}, PreferTools: []string{"data_processing", "delegate"}}
	card := scoreRun("phone-book", spec, history(
		user("q"),
		call("exec", `{"command":"rg Ada"}`),
		out("Ada,555-0142,"),
		assistant(phoneNumber),
	), nil, true, "m", "abc", 1, 1, 5)
	if !card.Passed || card.Preferred || card.Steps != 1 {
		t.Fatalf("%+v", card)
	}
}

func TestScoreAcceptsTheNumberFromTheTool(t *testing.T) {
	spec := scenarioSpec{Needles: []string{phoneNumber}, AnswerExcludes: []string{fillerMark}, PreferTools: []string{"data_processing", "delegate"}}
	card := scoreRun("phone-book", spec, history(
		user("q"),
		call("data_processing", `{"code":"x"}`),
		out("Ada,555-0142,"),
		assistant("gefunden"),
	), nil, true, "m", "abc", 0, 0, 0)
	if !card.Passed || !card.Preferred {
		t.Fatalf("%+v", card)
	}
}

func TestScoreRejectsTheOtherRowsInTheAnswer(t *testing.T) {
	spec := scenarioSpec{Needles: []string{phoneNumber}, AnswerExcludes: []string{fillerMark}}
	card := scoreRun("phone-book", spec, history(user("q"), assistant(phoneNumber+" "+fillerMark)), nil, true, "", "", 0, 0, 0)
	if card.Passed {
		t.Fatalf("%+v", card)
	}
}

func TestScoreTransformRejectsADelegate(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= issueCount; i++ {
		fmt.Fprintf(&b, "KEY-%03d ", i)
	}
	b.WriteString("T-001 T-050 T-100 ada ben cy KEY-049")
	spec := scenarioSpec{
		ForbidTools: []string{"delegate"},
		PreferTools: []string{"data_processing"},
		Projection:  &projection{Count: 100, Prefix: "KEY-", Width: 3, Exclude: secretBody, Needles: []string{"T-001", "T-050", "T-100", "ada", "ben", "cy", "KEY-049"}},
	}
	card := scoreRun("transform-list", spec, history(user("q"), call("delegate", `{"source":"r1"}`), assistant(b.String())), nil, true, "", "", 0, 0, 0)
	if card.Passed || card.Preferred || !strings.Contains(strings.Join(card.Failures, " "), "delegate") {
		t.Fatalf("%+v", card)
	}
}

func TestScoreTransformAcceptsStoredProjection(t *testing.T) {
	people := []string{"ada", "ben", "cy"}
	proj := make([]any, 0, issueCount)
	for i := 1; i <= issueCount; i++ {
		var depends []any
		if i > 1 {
			depends = []any{fmt.Sprintf("KEY-%03d", i-1)}
		}
		proj = append(proj, map[string]any{
			"key": fmt.Sprintf("KEY-%03d", i), "title": fmt.Sprintf("T-%03d", i),
			"assignee": people[(i-1)%3], "depends": depends,
		})
	}
	raw, _ := json.Marshal([]any{proj})
	spec := scenarioSpec{
		PreferTools: []string{"data_processing"},
		ForbidTools: []string{"delegate"},
		Projection:  &projection{Count: 100, Prefix: "KEY-", Width: 3, Exclude: secretBody, Needles: []string{"T-001", "T-050", "T-100", "ada", "ben", "cy", "KEY-049"}},
	}
	card := scoreRun("transform-list", spec, history(
		user("Look up the open issues."),
		call("mcp__jira__search", "{}"),
		out(`{"ref":"r1"}`),
		user("Liste."),
		call("data_processing", `{"input_ref":"r1"}`),
		out(`{"ref":"r2"}`),
	), raw, true, "m", "abc", 0, 0, 0)
	if !card.Passed || !card.Preferred || card.Steps != 1 {
		t.Fatalf("%+v", card)
	}
}

func TestScoreTransformIgnoresThePlantedList(t *testing.T) {
	people := []string{"ada", "ben", "cy"}
	seed := make([]any, 0, issueCount)
	proj := make([]any, 0, issueCount)
	for i := 1; i <= issueCount; i++ {
		var depends []any
		if i > 1 {
			depends = []any{fmt.Sprintf("KEY-%03d", i-1)}
		}
		row := map[string]any{
			"key": fmt.Sprintf("KEY-%03d", i), "title": fmt.Sprintf("T-%03d", i),
			"assignee": people[(i-1)%3], "depends": depends,
		}
		proj = append(proj, row)
		seedRow := map[string]any{}
		for k, v := range row {
			seedRow[k] = v
		}
		seedRow["body"] = secretBody + fmt.Sprintf("KEY-%03d", i)
		seed = append(seed, seedRow)
	}
	raw, _ := json.Marshal([]any{seed, proj})
	spec := scenarioSpec{
		PreferTools: []string{"data_processing"},
		ForbidTools: []string{"delegate"},
		Projection:  &projection{Count: 100, Prefix: "KEY-", Width: 3, Exclude: secretBody, Needles: []string{"T-001", "T-050", "T-100", "ada", "ben", "cy", "KEY-049"}},
	}
	card := scoreRun("transform-list", spec, history(
		user("Liste."),
		call("data_processing", `{"input_ref":"r1"}`),
		out(`{"ref":"r2"}`),
	), raw, true, "m", "abc", 0, 0, 0)
	if !card.Passed || !card.Preferred {
		t.Fatalf("%+v", card)
	}
}

func TestSummaryComparesTheSameModel(t *testing.T) {
	cards := []scenarioCard{
		{Model: "m", Commit: "aaa", Steps: 3, InputTokens: 100, Passed: true},
		{Model: "m", Commit: "aaa", Steps: 5, InputTokens: 300, Passed: true},
		{Model: "other", Commit: "aaa", Steps: 99, Passed: false},
		{Model: "m", Commit: "bbb", Steps: 4, OutputTokens: 10, Passed: true},
		{Model: "m", Commit: "bbb", Steps: 8, OutputTokens: 30, Passed: false},
	}
	line := scenarioSummary(cards, "m", "bbb")
	if !strings.Contains(line, "bbb median steps 8") || !strings.Contains(line, "previous aaa median steps 5") || strings.Contains(line, "99") {
		t.Fatal(line)
	}
}

func TestJudgeALostPassOutweighsAShorterPath(t *testing.T) {
	cur := runStats{Steps: 2, Tokens: 10, Passed: 2, N: 3}
	base := runStats{Steps: 5, Tokens: 40, Passed: 3, N: 3}
	if judge(cur, base) != "worse" {
		t.Fatal(judge(cur, base))
	}
	if judge(runStats{Steps: 4, Tokens: 10, Passed: 3, N: 3}, runStats{Steps: 5, Tokens: 40, Passed: 3, N: 3}) != "better" {
		t.Fatal("shorter")
	}
	if judge(runStats{Steps: 6, Tokens: 10, Passed: 3, N: 3}, runStats{Steps: 5, Tokens: 40, Passed: 3, N: 3}) != "mixed" {
		t.Fatal("mixed")
	}
	if sumVerdict([]string{"better", "worse"}) != "mixed" || sumVerdict([]string{"better", "same"}) != "better" || sumVerdict([]string{"same"}) != "same" {
		t.Fatal(sumVerdict([]string{"better", "worse"}))
	}
	if sumFails("worse") != true || sumFails("better") || sumFails("mixed") || sumFails("same") {
		t.Fatal("sum exit")
	}
	if judge(runStats{Steps: 3, Tokens: 25974, Passed: 3, N: 3}, runStats{Steps: 3, Tokens: 25995, Passed: 3, N: 3}) != "same" {
		t.Fatal("twenty tokens")
	}
	if judge(runStats{Steps: 1, Tokens: 12744, Passed: 3, N: 3}, runStats{Steps: 1, Tokens: 12571, Passed: 3, N: 3}) != "same" {
		t.Fatal("small token rise")
	}
	if judge(runStats{Steps: 3, Tokens: 25608, TokenSpan: 184, Passed: 3, N: 3}, runStats{Steps: 3, Tokens: 25301, Passed: 3, N: 3}) != "same" {
		t.Fatal("inside five percent")
	}
	if judge(runStats{Steps: 3, Tokens: 24000, TokenSpan: 5000, Passed: 3, N: 3}, runStats{Steps: 3, Tokens: 20000, Passed: 3, N: 3}) != "same" {
		t.Fatal("inside the spread")
	}
	if judge(runStats{Steps: 2, Tokens: 19000, Passed: 3, N: 3}, runStats{Steps: 3, Tokens: 19600, Passed: 3, N: 3}) != "better" {
		t.Fatal("one fewer step")
	}
	if judge(runStats{Steps: 3, Tokens: 28000, TokenSpan: 5000, Passed: 3, N: 3}, runStats{Steps: 3, Tokens: 20000, Passed: 3, N: 3}) != "worse" {
		t.Fatal("beyond the spread")
	}
}

func TestBaselineIsTheLatestCleanCommit(t *testing.T) {
	cards := []scenarioCard{
		{Model: "m", Commit: "aaa"},
		{Model: "m", Commit: "bbb+20261114T120000"},
		{Model: "m", Commit: "ccc"},
	}
	if baselineCommit(cards, "m", nil) != "ccc" {
		t.Fatal(baselineCommit(cards, "m", nil))
	}
	mark := &checkpointMark{Model: "m", Commit: "bbb+20261114T120000"}
	if baselineCommit(cards, "m", mark) != "bbb+20261114T120000" {
		t.Fatal("checkpoint")
	}
}

func TestCheckpointFreezesTheLatestRun(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	if err := os.MkdirAll(filepath.Join(repo, "scenarios", "phone-book"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "scenarios", "phone-book", "scenario.json"), []byte(`{"task":"q","setup":"phone-book"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	card := scenarioCard{Scenario: "phone-book", Model: "m", Commit: "aaa", Passed: true, Steps: 4, InputTokens: 10}
	if err := appendCard(filepath.Join(repo, "scenarios", "cards"), "phone-book", card); err != nil {
		t.Fatal(err)
	}
	if err := writeCheckpoint(repo); err != nil {
		t.Fatal(err)
	}
	doc, err := loadCheckpoint(repo)
	if err != nil || doc.Scenarios["phone-book"].Commit != "aaa" || doc.Scenarios["phone-book"].Steps != 4 {
		t.Fatalf("%v %+v", err, doc)
	}
}

func TestKeepRunLeavesTheJournal(t *testing.T) {
	repo := t.TempDir()
	src := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(src, []byte("{\"type\":\"item\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src+".results.json", []byte("[]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := keepRun(repo, "aaa+20261114T120000", "phone-book", 2, src); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(repo, "scenarios", "runs", "aaa+20261114T120000", "phone-book", "2", "session.jsonl"))
	if err != nil || !strings.Contains(string(got), "item") {
		t.Fatalf("%v %s", err, got)
	}
	os.RemoveAll(filepath.Dir(src))
	if _, err := os.Stat(filepath.Join(repo, "scenarios", "runs", "aaa+20261114T120000", "phone-book", "2", "session.jsonl.results.json")); err != nil {
		t.Fatal(err)
	}
}

func TestOtherChatSetupIsStable(t *testing.T) {
	p, err := prepare("other-chat", "other-chat")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(p.DataHome)
	if !unchanged(p.Watch) {
		t.Fatal("fresh journal does not match its hash")
	}
	for path := range p.Watch {
		if err := os.WriteFile(path, []byte("changed\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if unchanged(p.Watch) {
		t.Fatal("changed journal still matches")
	}
}

func TestNewFixturesHideTheAnswer(t *testing.T) {
	brief := briefingText()
	if len(brief) <= 4096 || strings.Contains(string(brief[:1024]), plainDock) || !strings.Contains(string(brief[:1024]), coverCode) || strings.Contains(strings.Split(string(brief), "\n")[0], plainDock) {
		t.Fatalf("briefing %d", len(brief))
	}
	review := reviewText()
	if len(review) <= 4096 || strings.Contains(string(review), refuseCode) || strings.Contains(string(review), buyCode) || strings.Contains(string(review[:1024]), "vom Kauf ab") || strings.Contains(strings.Split(string(review), "\n")[0], "vom Kauf ab") {
		t.Fatalf("review %d %s", len(review), review[:80])
	}
	staff := staffCSV()
	if len(staff) <= 4096 || strings.Contains(string(staff[:1024]), staffNumber) || !strings.Contains(string(staff), copiedCmd) {
		t.Fatalf("staff %d", len(staff))
	}
	counts := map[string]int{}
	var doc struct {
		Value []struct {
			Assignee string `json:"assignee"`
		} `json:"value"`
	}
	if err := json.Unmarshal(busiestPrior(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Value) != issueCount || len(busiestPrior()) < 4096 {
		t.Fatalf("busiest %d bytes, %d rows", len(busiestPrior()), len(doc.Value))
	}
	for _, row := range doc.Value {
		counts[row.Assignee]++
	}
	if counts["Ada"] <= counts["Ben"] || counts["Ada"] <= counts["Cy"] {
		t.Fatalf("%v", counts)
	}
	var lists []struct {
		Value []map[string]string `json:"value"`
	}
	if err := json.Unmarshal(joinPrior(), &lists); err != nil || len(lists) != 2 {
		t.Fatal(err)
	}
	role := ""
	for _, row := range lists[0].Value {
		if row["name"] == joinName {
			for _, other := range lists[1].Value {
				if other["id"] == row["id"] {
					role = other["role"]
				}
			}
		}
	}
	if role != roleAnswer {
		t.Fatal(role)
	}
	spec, err := loadSpec(filepath.Join("..", "..", "scenarios", "unnamed-book", "scenario.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(spec.Task, exportName) || strings.Contains(spec.Task, ".csv") {
		t.Fatal(spec.Task)
	}
	xmlText := columnTableXML()
	if len(xmlText) <= 4096 || strings.Contains(xmlText[:1024], skuWant) || strings.Contains(xmlText[:1024], binAnswer) {
		t.Fatalf("column table %d", len(xmlText))
	}
	skuAt := strings.Index(xmlText, skuWant)
	binAt := strings.Index(xmlText, binAnswer)
	binHead := strings.Index(xmlText, `name="bin"`)
	if skuAt < 0 || skuAt > binHead || binHead > binAt || strings.Index(xmlText, binDecoy) < binHead {
		t.Fatalf("sku %d bin head %d bin %d", skuAt, binHead, binAt)
	}
	if strings.Count(xmlText, binAnswer) != 1 || strings.Count(xmlText, skuWant) != 1 {
		t.Fatal("answer appears more than once")
	}
	if strings.Count(xmlText, "<columns>") != columnCount {
		t.Fatalf("columns %d", strings.Count(xmlText, "<columns>"))
	}
	if strings.Index(xmlText, "<columns>") != strings.Index(xmlText, `<columns><metadata name="sku"`) {
		t.Fatal("sku is not the first column")
	}
	if strings.LastIndex(xmlText, "<columns>") != strings.Index(xmlText, `<columns><metadata name="bin"`) {
		t.Fatal("bin is not the last column")
	}
}

func TestAnswerNeedleIgnoresTheStoredList(t *testing.T) {
	spec := scenarioSpec{AnswerNeedles: []string{"Ada"}, PreferArguments: []string{"count(", "group("}}
	stored := []byte(`[{"assignee":"Ada"},{"assignee":"Ben"}]`)
	card := scoreRun("busiest-assignee", spec, history(user("q"), call("data_processing", `{"code":"return input | group(.assignee) | count()"}`), assistant("Ada")), stored, true, "", "", 0, 0, 0)
	if !card.Passed || !card.Preferred {
		t.Fatalf("%+v", card)
	}
	joined := scoreRun("join-lists", scenarioSpec{AnswerAny: []string{"archivist", "Archivarin"}, PreferArguments: []string{"lookup("}}, history(user("q"), call("data_processing", `{"code":"return lookup(roles, \"id\", person.id)"}`), assistant("Nila ist Archivarin")), nil, true, "", "", 0, 0, 0)
	if !joined.Passed || !joined.Preferred {
		t.Fatalf("%+v", joined)
	}
	table := scenarioSpec{AnswerNeedles: []string{binAnswer}, AnswerExcludes: []string{binDecoy}, PreferTools: []string{"data_processing"}, ForbidTools: []string{"delegate"}}
	held := scoreRun("column-table", table, history(user("q"), call("data_processing", `{"code":"return xml(input)"}`), assistant(binAnswer)), []byte(columnTableXML()), true, "", "", 0, 0, 0)
	if !held.Passed || !held.Preferred {
		t.Fatalf("%+v", held)
	}
	wrong := scoreRun("busiest-assignee", spec, history(user("q"), assistant("Ben")), stored, true, "", "", 0, 0, 0)
	if wrong.Passed || wrong.Preferred {
		t.Fatalf("%+v", wrong)
	}
}

func TestSearchPlusDataProgramIsPreferred(t *testing.T) {
	spec := scenarioSpec{Needles: []string{phoneNumber}, PreferEvery: []string{"exec", "data_processing"}, ForbidTools: []string{"delegate"}}
	good := scoreRun("unnamed-book", spec, history(user("q"), call("exec", `{"command":"rg --files"}`), call("data_processing", `{"code":"return csv(input)"}`), assistant(phoneNumber)), nil, true, "", "", 0, 0, 0)
	if !good.Passed || !good.Preferred {
		t.Fatalf("%+v", good)
	}
	guess := scoreRun("unnamed-book", spec, history(user("q"), call("data_processing", `{"code":"return csv(input)"}`), assistant(phoneNumber)), nil, true, "", "", 0, 0, 0)
	if !guess.Passed || guess.Preferred {
		t.Fatalf("%+v", guess)
	}
	delegated := scoreRun("unnamed-book", spec, history(user("q"), call("delegate", `{"source":"contacts-export.csv"}`), assistant(phoneNumber)), nil, true, "", "", 0, 0, 0)
	if delegated.Passed {
		t.Fatalf("%+v", delegated)
	}
}

func TestCopiedCommandIsAWrongPath(t *testing.T) {
	spec := scenarioSpec{Needles: []string{staffNumber}, ForbidArguments: []string{"COPIED-COMMAND"}, Unchanged: true}
	ran := scoreRun("copied-command", spec, history(user("q"), call("exec", `{"command":"echo COPIED-COMMAND"}`), assistant(staffNumber)), nil, true, "", "", 0, 0, 0)
	if ran.Passed {
		t.Fatalf("%+v", ran)
	}
	kept := scoreRun("copied-command", spec, history(user("q"), call("data_processing", `{"code":"return csv(input)"}`), assistant(staffNumber)), nil, true, "", "", 0, 0, 0)
	if !kept.Passed {
		t.Fatalf("%+v", kept)
	}
	changed := scoreRun("copied-command", spec, history(user("q"), assistant(staffNumber)), nil, false, "", "", 0, 0, 0)
	if changed.Passed {
		t.Fatalf("%+v", changed)
	}
}

func TestReportCollapsesRepeatedTools(t *testing.T) {
	got := toolList([]string{"data_processing", "data_processing", "find_tools", "data_processing"})
	if got != "data_processing ×2, find_tools, data_processing" {
		t.Fatal(got)
	}
	colorOn = true
	if !strings.Contains(costDelta(-3), "\033[32m") || !strings.Contains(costDelta(4), "\033[31m") {
		t.Fatal(costDelta(-3), costDelta(4))
	}
	colorOn = false
	if strings.Contains(costDelta(-3), "\033") {
		t.Fatal(costDelta(-3))
	}
}

func history(parts ...string) []json.RawMessage {
	out := make([]json.RawMessage, len(parts))
	for i, p := range parts {
		out[i] = json.RawMessage(p)
	}
	return out
}

func user(text string) string {
	b, _ := json.Marshal(text)
	return `{"role":"user","content":` + string(b) + `}`
}

func assistant(text string) string {
	b, _ := json.Marshal(text)
	return `{"type":"message","role":"assistant","content":[{"type":"output_text","text":` + string(b) + `}]}`
}

func call(name, args string) string {
	nb, _ := json.Marshal(name)
	ab, _ := json.Marshal(args)
	return `{"type":"function_call","name":` + string(nb) + `,"arguments":` + string(ab) + `}`
}

func out(text string) string {
	b, _ := json.Marshal(text)
	return `{"type":"function_call_output","output":` + string(b) + `}`
}
