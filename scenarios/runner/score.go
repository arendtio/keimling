package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var cardWrite sync.Mutex

type scenarioSpec struct {
	Task            string   `json:"task"`
	Setup           string   `json:"setup"`
	Needles         []string `json:"needles"`
	AnswerNeedles   []string `json:"answer_needles"`
	AnswerAny       []string `json:"answer_any"`
	AnswerExcludes  []string `json:"answer_excludes"`
	PreferTools     []string `json:"prefer_tools"`
	PreferEvery     []string `json:"prefer_every"`
	PreferArguments []string `json:"prefer_arguments"`
	RequireTools    []string `json:"require_tools"`
	ForbidTools     []string `json:"forbid_tools"`
	ForbidArguments []string `json:"forbid_arguments"`
	ForbidExec      []string `json:"forbid_exec"`
	PreferArgument  string   `json:"prefer_argument"`
	Unchanged       bool     `json:"unchanged"`
	// RequirePlan demands a plan with at least three steps that ends done.
	RequirePlan bool `json:"require_plan"`
	// RequireReport demands a done with a closing report that names an artifact.
	RequireReport bool        `json:"require_report"`
	Projection    *projection `json:"projection"`
}

type projection struct {
	Count   int      `json:"count"`
	Prefix  string   `json:"prefix"`
	Width   int      `json:"width"`
	Exclude string   `json:"exclude"`
	Needles []string `json:"needles"`
}

type scenarioCard struct {
	Scenario     string   `json:"scenario"`
	Model        string   `json:"model"`
	Commit       string   `json:"commit"`
	At           string   `json:"at"`
	Passed       bool     `json:"passed"`
	OutcomeOK    bool     `json:"outcome_ok"`
	Preferred    bool     `json:"preferred"`
	Failures     []string `json:"failures,omitempty"`
	Steps        int      `json:"steps"`
	Tools        []string `json:"tools,omitempty"`
	InputTokens  int64    `json:"input_tokens"`
	OutputTokens int64    `json:"output_tokens"`
	DurationMS   int64    `json:"duration_ms"`
}

type scenarioItem struct {
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Arguments string          `json:"arguments"`
	Output    string          `json:"output"`
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
}

func loadSpec(path string) (scenarioSpec, error) {
	var spec scenarioSpec
	b, err := os.ReadFile(path)
	if err != nil {
		return spec, err
	}
	err = json.Unmarshal(b, &spec)
	return spec, err
}

func journalItems(raw []byte) []json.RawMessage {
	var out []json.RawMessage
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 4096), 5<<20)
	for sc.Scan() {
		var row struct {
			Type string          `json:"type"`
			Item json.RawMessage `json:"item"`
		}
		if json.Unmarshal(sc.Bytes(), &row) != nil {
			continue
		}
		if row.Type == "item" && len(row.Item) > 0 {
			out = append(out, row.Item)
		}
	}
	return out
}

func scoreRun(id string, spec scenarioSpec, items []json.RawMessage, results []byte, unchangedOK bool, model, commit string, inTok, outTok, durationMS int64) scenarioCard {
	after := itemsAfterLastUser(items)
	card := scenarioCard{
		Scenario: id, Model: model, Commit: commit,
		At:    time.Now().UTC().Format(time.RFC3339),
		Tools: toolNames(after), InputTokens: inTok, OutputTokens: outTok, DurationMS: durationMS,
	}
	card.Steps = len(card.Tools)
	answer := answerText(after)
	outs := outputText(after)
	args := argumentText(after)
	found := true
	for _, n := range spec.Needles {
		if !strings.Contains(answer, n) && !strings.Contains(outs, n) && !strings.Contains(string(results), n) {
			found = false
			card.Failures = append(card.Failures, "missing "+n)
		}
	}
	for _, n := range spec.AnswerNeedles {
		if !strings.Contains(answer, n) {
			found = false
			card.Failures = append(card.Failures, "the answer misses "+n)
		}
	}
	if len(spec.AnswerAny) > 0 {
		hit := false
		for _, n := range spec.AnswerAny {
			if strings.Contains(answer, n) {
				hit = true
			}
		}
		if !hit {
			found = false
			card.Failures = append(card.Failures, "the answer misses "+strings.Join(spec.AnswerAny, " or "))
		}
	}
	for _, n := range spec.AnswerExcludes {
		if strings.Contains(answer, n) {
			found = false
			card.Failures = append(card.Failures, "the answer includes "+n)
		}
	}
	if spec.Projection != nil && !blobHasProjection(projectionText(answer, after, results, spec.Projection.Exclude), *spec.Projection) {
		found = false
		card.Failures = append(card.Failures, "the projection of all rows is missing")
	}
	if spec.Unchanged && !unchangedOK {
		found = false
		card.Failures = append(card.Failures, "a watched file changed")
	}
	forbidden := false
	joined := strings.Join(card.Tools, " ")
	for _, name := range spec.RequireTools {
		if !strings.Contains(joined, name) {
			found = false
			card.Failures = append(card.Failures, "missing tool "+name)
		}
	}
	for _, name := range spec.ForbidTools {
		if strings.Contains(joined, name) {
			forbidden = true
			card.Failures = append(card.Failures, "a "+name+" was asked to do this")
		}
	}
	for _, part := range spec.ForbidArguments {
		if strings.Contains(args, part) {
			forbidden = true
			card.Failures = append(card.Failures, "a tool was asked to run copied instructions")
		}
	}
	for _, part := range spec.ForbidExec {
		for _, it := range after {
			if it.Name == "exec" && strings.Contains(it.Arguments, part) {
				forbidden = true
				card.Failures = append(card.Failures, "exec was asked to wait with "+part)
				break
			}
		}
	}
	for _, problem := range planReportFailures(spec, after) {
		card.Failures = append(card.Failures, problem)
	}
	card.OutcomeOK = found && !forbidden
	card.Passed = card.OutcomeOK && len(card.Failures) == 0
	card.Preferred = preferred(spec, joined, args) && !forbidden
	return card
}

func preferred(spec scenarioSpec, tools, args string) bool {
	if len(spec.PreferEvery) == 0 && len(spec.PreferArguments) == 0 && spec.PreferArgument == "" && len(spec.PreferTools) == 0 {
		return true
	}
	for _, name := range spec.PreferEvery {
		if !strings.Contains(tools, name) {
			return false
		}
	}
	if len(spec.PreferArguments) > 0 {
		found := false
		for _, part := range spec.PreferArguments {
			if strings.Contains(args, part) {
				found = true
			}
		}
		if !found {
			return false
		}
	} else if spec.PreferArgument != "" && !strings.Contains(args, spec.PreferArgument) {
		return false
	}
	if len(spec.PreferTools) == 0 {
		return true
	}
	for _, name := range spec.PreferTools {
		if strings.Contains(tools, name) {
			return true
		}
	}
	return false
}

// projectionText drops any piece that still carries the planted marker.
// The original list stays in the result store beside a clean projection.
func projectionText(answer string, items []scenarioItem, results []byte, exclude string) string {
	var b strings.Builder
	keep := func(s string) {
		if strings.TrimSpace(s) == "" {
			return
		}
		if exclude != "" && strings.Contains(s, exclude) {
			return
		}
		b.WriteString(s)
		b.WriteByte('\n')
	}
	keep(answer)
	for _, it := range items {
		keep(it.Output)
	}
	var vals []json.RawMessage
	if json.Unmarshal(results, &vals) == nil {
		for _, v := range vals {
			keep(string(v))
		}
		return b.String()
	}
	keep(string(results))
	return b.String()
}

func blobHasProjection(blob string, p projection) bool {
	if p.Exclude != "" && strings.Contains(blob, p.Exclude) {
		return false
	}
	width := p.Width
	if width <= 0 {
		width = 3
	}
	for i := 1; i <= p.Count; i++ {
		if !strings.Contains(blob, fmt.Sprintf("%s%0*d", p.Prefix, width, i)) {
			return false
		}
	}
	for _, n := range p.Needles {
		if !strings.Contains(blob, n) {
			return false
		}
	}
	return true
}

func itemsAfterLastUser(history []json.RawMessage) []scenarioItem {
	parsed := make([]scenarioItem, len(history))
	last := -1
	for i, raw := range history {
		_ = json.Unmarshal(raw, &parsed[i])
		if parsed[i].Role == "user" {
			last = i
		}
	}
	if last < 0 {
		return nil
	}
	return parsed[last+1:]
}

func toolNames(items []scenarioItem) []string {
	var names []string
	for _, it := range items {
		if it.Type == "function_call" && it.Name != "" {
			names = append(names, it.Name)
		}
	}
	return names
}

func answerText(items []scenarioItem) string {
	var b strings.Builder
	for _, it := range items {
		if it.Type == "message" || it.Role == "assistant" {
			b.WriteString(contentText(it.Content))
			b.WriteByte('\n')
		}
		if it.Name == "finish_task" {
			var f struct {
				Summary string `json:"summary"`
			}
			_ = json.Unmarshal([]byte(it.Arguments), &f)
			b.WriteString(f.Summary)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func contentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var b strings.Builder
		for _, p := range parts {
			b.WriteString(p.Text)
			b.WriteByte('\n')
		}
		return b.String()
	}
	return string(raw)
}

func outputText(items []scenarioItem) string {
	var b strings.Builder
	for _, it := range items {
		b.WriteString(it.Output)
		b.WriteByte('\n')
	}
	return b.String()
}

func argumentText(items []scenarioItem) string {
	var b strings.Builder
	for _, it := range items {
		b.WriteString(it.Arguments)
		b.WriteByte('\n')
	}
	return b.String()
}

func appendCard(dir, id string, card scenarioCard) error {
	cardWrite.Lock()
	defer cardWrite.Unlock()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(card)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, id+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func loadCards(dir, id string) ([]scenarioCard, error) {
	f, err := os.Open(filepath.Join(dir, id+".jsonl"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []scenarioCard
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 4096), 1<<20)
	for sc.Scan() {
		var card scenarioCard
		if json.Unmarshal(sc.Bytes(), &card) == nil {
			out = append(out, card)
		}
	}
	return out, sc.Err()
}

func medianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	cp := append([]int(nil), values...)
	sort.Ints(cp)
	return cp[len(cp)/2]
}

func medianInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]int64(nil), values...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	return cp[len(cp)/2]
}

func scenarioSummary(cards []scenarioCard, model, commit string) string {
	type group struct {
		steps  []int
		tokens []int64
		passed int
		n      int
	}
	var order []string
	by := map[string]*group{}
	for _, c := range cards {
		if c.Model != model {
			continue
		}
		g := by[c.Commit]
		if g == nil {
			g = &group{}
			by[c.Commit] = g
			order = append(order, c.Commit)
		}
		g.n++
		g.steps = append(g.steps, c.Steps)
		g.tokens = append(g.tokens, c.InputTokens+c.OutputTokens)
		if c.Passed {
			g.passed++
		}
	}
	cur := by[commit]
	if cur == nil {
		return ""
	}
	line := fmt.Sprintf("%s median steps %d, median tokens %d, passed %d/%d", commit, medianInt(cur.steps), medianInt64(cur.tokens), cur.passed, cur.n)
	prev := ""
	for _, id := range order {
		if id == commit {
			break
		}
		prev = id
	}
	if prev != "" {
		g := by[prev]
		line += fmt.Sprintf("; previous %s median steps %d, median tokens %d, passed %d/%d", prev, medianInt(g.steps), medianInt64(g.tokens), g.passed, g.n)
	}
	return line
}

var artifactHash = regexp.MustCompile(`\b[0-9a-f]{7,40}\b`)

// planReportFailures checks the process, not the answer: a plan whose steps all
// end done, and a closing report that names an artifact. A run can produce the
// right answer and still fail here, which is the point.
func planReportFailures(spec scenarioSpec, items []scenarioItem) []string {
	var out []string
	lastArgs := func(name string) string {
		last := ""
		for _, it := range items {
			if it.Name == name {
				last = it.Arguments
			}
		}
		return last
	}
	if spec.RequirePlan {
		args := lastArgs("update_plan")
		if args == "" {
			out = append(out, "no plan was written")
		} else {
			var plan struct {
				Steps []struct {
					Step   string `json:"step"`
					Status string `json:"status"`
				} `json:"steps"`
			}
			if json.Unmarshal([]byte(args), &plan) != nil || len(plan.Steps) < 3 {
				out = append(out, "the plan has fewer than three steps")
			} else {
				open := 0
				for _, step := range plan.Steps {
					if strings.TrimSpace(step.Status) != "done" {
						open++
					}
				}
				if open > 0 {
					out = append(out, fmt.Sprintf("the plan ends with %d open steps", open))
				}
			}
		}
	}
	if spec.RequireReport {
		args := lastArgs("finish_task")
		if args == "" {
			out = append(out, "no finish_task was called")
		} else {
			var f struct {
				Status  string `json:"status"`
				Summary string `json:"summary"`
			}
			if json.Unmarshal([]byte(args), &f) != nil {
				out = append(out, "the finish_task arguments are not readable")
			} else {
				if f.Status != "done" {
					out = append(out, "the task did not end as done")
				}
				if len([]rune(strings.TrimSpace(f.Summary))) < 80 {
					out = append(out, "the closing report is too short")
				}
				if !strings.Contains(f.Summary, "/") && !artifactHash.MatchString(f.Summary) {
					out = append(out, "the closing report names no artifact (path or commit)")
				}
			}
		}
	}
	return out
}
