package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTasksStayOutOfTheShippingSources(t *testing.T) {
	src := filepath.Join("..", "..", "src")
	needles := []string{phoneNumber, dockCode, otherLine, "welche Nummer gehört zu Ada"}
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
