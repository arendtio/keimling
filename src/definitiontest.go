package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const definitionTestMaxRounds = 8

const definitionJudgeInstructions = "You judge one message from an agent. Reply with a JSON object only, no markdown: {\"pass\":true or false,\"reason\":\"one sentence\"}. Pass when the message satisfies the check. Different wording still passes. Fail when the required point is missing."

// DefinitionTest is a script attached to one agent definition.
type DefinitionTest struct {
	ID       string                           `json:"id"`
	Name     string                           `json:"name"`
	Parties  []DefinitionTestParty            `json:"parties"`
	Steps    []DefinitionTestStep             `json:"steps"`
	Tools    map[string]definitionToolResults `json:"tools,omitempty"`
	LastRuns []DefinitionTestRunResult        `json:"last_runs,omitempty"`
}

// definitionToolResults is the result of each successive call of one tool.
// A JSON string is the result of the first call. Calls past the last result
// receive that last result again.
type definitionToolResults []string

func (r *definitionToolResults) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*r = nil
		return nil
	}
	if data[0] == '"' {
		var one string
		if err := json.Unmarshal(data, &one); err != nil {
			return err
		}
		*r = definitionToolResults{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*r = many
	return nil
}

// DefinitionTestParty is a scripted sender. It is not a model instance.
type DefinitionTestParty struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// DefinitionTestStep is one scripted message, one expected reply, or both.
type DefinitionTestStep struct {
	From  string `json:"from,omitempty"`
	Text  string `json:"text,omitempty"`
	To    string `json:"to,omitempty"`
	Check string `json:"check,omitempty"`
}

// DefinitionTestRunResult is one of the three runs.
type DefinitionTestRunResult struct {
	Pass       bool             `json:"pass"`
	Detail     string           `json:"detail"`
	Transcript []sessionMessage `json:"transcript,omitempty"`
}

type definitionTestSend struct {
	to   string
	text string
}

type definitionTestRun struct {
	test      DefinitionTest
	sent      *definitionTestSend
	toolCalls map[string]int
}

func (run *definitionTestRun) nextToolResult(name string) string {
	results := run.test.Tools[name]
	if run.toolCalls == nil {
		run.toolCalls = map[string]int{}
	}
	i := run.toolCalls[name]
	run.toolCalls[name] = i + 1
	if len(results) == 0 {
		return ""
	}
	if i >= len(results) {
		return results[len(results)-1]
	}
	return results[i]
}

func definitionTestPassed(runs []DefinitionTestRunResult) bool {
	if len(runs) != 3 {
		return false
	}
	for _, run := range runs {
		if !run.Pass {
			return false
		}
	}
	return true
}

func normalizeDefinitionTest(test DefinitionTest) (DefinitionTest, error) {
	test.ID = strings.TrimSpace(test.ID)
	test.Name = strings.TrimSpace(test.Name)
	if test.Name == "" {
		return test, fmt.Errorf("test name required")
	}
	if len(test.Parties) == 0 {
		return test, fmt.Errorf("at least one party required")
	}
	if len(test.Steps) == 0 {
		return test, fmt.Errorf("at least one step required")
	}
	seen := map[string]bool{}
	for i := range test.Parties {
		p := &test.Parties[i]
		p.ID = strings.TrimSpace(p.ID)
		p.Name = strings.TrimSpace(p.Name)
		p.Kind = strings.TrimSpace(p.Kind)
		if p.ID == "" {
			return test, fmt.Errorf("party id required")
		}
		if seen[p.ID] {
			return test, fmt.Errorf("party %s is listed twice", p.ID)
		}
		seen[p.ID] = true
		if p.Name == "" {
			p.Name = p.ID
		}
		switch p.Kind {
		case "":
			p.Kind = "agent"
		case "agent", "thread":
		default:
			return test, fmt.Errorf("party %s must be a thread or an agent", p.ID)
		}
	}
	for i, step := range test.Steps {
		step.From = strings.TrimSpace(step.From)
		step.To = strings.TrimSpace(step.To)
		step.Text = strings.TrimSpace(step.Text)
		step.Check = strings.TrimSpace(step.Check)
		if step.From == "" && step.To == "" {
			return test, fmt.Errorf("step %d is empty", i+1)
		}
		if step.From != "" && !seen[step.From] {
			return test, fmt.Errorf("step %d: unknown party %s", i+1, step.From)
		}
		if step.To != "" && !seen[step.To] {
			return test, fmt.Errorf("step %d: unknown party %s", i+1, step.To)
		}
		if step.From != "" && step.Text == "" {
			return test, fmt.Errorf("step %d: the message is empty", i+1)
		}
		test.Steps[i] = step
	}
	cleaned := map[string]definitionToolResults{}
	for name, result := range test.Tools {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if result == nil {
			result = definitionToolResults{}
		}
		cleaned[name] = result
	}
	test.Tools = cleaned
	return test, nil
}

func saveDefinitionTest(recipeID string, test DefinitionTest) (AgentRecipe, error) {
	r, err := readAgentRecipe(recipeID)
	if err != nil {
		return AgentRecipe{}, err
	}
	test, err = normalizeDefinitionTest(test)
	if err != nil {
		return AgentRecipe{}, err
	}
	if test.ID == "" || strings.HasPrefix(test.ID, "draft-") {
		test.ID = newStampedID("deftest")
	}
	replaced := false
	for i := range r.Tests {
		if r.Tests[i].ID == test.ID {
			if test.LastRuns == nil {
				test.LastRuns = r.Tests[i].LastRuns
			}
			r.Tests[i] = test
			replaced = true
			break
		}
	}
	if !replaced {
		r.Tests = append(r.Tests, test)
	}
	if err := writeAgentRecipe(r); err != nil {
		return AgentRecipe{}, err
	}
	return readAgentRecipe(r.ID)
}

func deleteDefinitionTest(recipeID, testID string) (AgentRecipe, error) {
	r, err := readAgentRecipe(recipeID)
	if err != nil {
		return AgentRecipe{}, err
	}
	testID = strings.TrimSpace(testID)
	next := make([]DefinitionTest, 0, len(r.Tests))
	found := false
	for _, test := range r.Tests {
		if test.ID == testID {
			found = true
			continue
		}
		next = append(next, test)
	}
	if !found {
		return AgentRecipe{}, fmt.Errorf("test not found")
	}
	r.Tests = next
	if err := writeAgentRecipe(r); err != nil {
		return AgentRecipe{}, err
	}
	return r, nil
}

func (test DefinitionTest) party(id string) (DefinitionTestParty, bool) {
	id = strings.TrimSpace(id)
	for _, p := range test.Parties {
		if p.ID == id {
			return p, true
		}
	}
	return DefinitionTestParty{}, false
}

func definitionTestWake(test DefinitionTest, from, text string) (string, error) {
	party, ok := test.party(from)
	if !ok {
		return "", fmt.Errorf("unknown party %s", from)
	}
	if party.Kind == "thread" {
		return formatPeerUserText("thread_id="+party.ID+" via message_agent", text), nil
	}
	label := party.Name
	if label == "" {
		label = party.ID
	}
	if label != party.ID {
		label = label + " instance_id=" + party.ID
	}
	return formatPeerUserText(label+" via message_agent", text), nil
}

// simulatedDefinitionTool returns a scripted result for a granted tool.
// A tool the definition does not grant, or a name that is not a tool, returns
// false so the ordinary agent path answers.
func (a *Agent) simulatedDefinitionTool(item Item) (string, bool) {
	run := a.defTest
	if run == nil || !a.instanceToolGranted(item.Name) || !knownAgentTool(item.Name) {
		return "", false
	}
	if item.Name == "message_agent" {
		var x struct {
			InstanceID string `json:"instance_id"`
			Text       string `json:"text"`
		}
		if json.Unmarshal([]byte(item.Arguments), &x) != nil {
			return "", false
		}
		target := strings.TrimSpace(x.InstanceID)
		text := strings.TrimSpace(x.Text)
		if target == "" || text == "" {
			return "", false
		}
		party, ok := run.test.party(target)
		if !ok {
			// A script party is the only recipient. Do not deliver to a live agent.
			return "message_agent error: unknown agent or thread", true
		}
		if run.sent == nil {
			run.sent = &definitionTestSend{to: party.ID, text: text}
		}
		name := party.Name
		if name == "" {
			name = party.ID
		}
		return fmt.Sprintf("Sent to %s: %s", name, text), true
	}
	return run.nextToolResult(item.Name), true
}

func (a *Agent) swapDefinitionJournal(path string) error {
	if a.Journal != nil && a.Journal.File != nil {
		_ = a.Journal.Close()
	}
	j, err := OpenJournal(path, false)
	if err != nil {
		return err
	}
	a.Journal = j
	a.History = j.History
	a.defTest = nil
	return nil
}

func (a *Agent) runDefinitionTest(ctx context.Context, test DefinitionTest) (DefinitionTest, error) {
	test, err := normalizeDefinitionTest(test)
	if err != nil {
		return test, err
	}
	base := strings.TrimSpace(a.Cfg.SessionPath)
	if base == "" {
		base = agentInstanceSessionPath(a.Cfg.AgentInstanceID)
	}
	dir := filepath.Dir(base)
	results := make([]DefinitionTestRunResult, 0, 3)
	for i := 1; i <= 3; i++ {
		if err := ctx.Err(); err != nil {
			return test, err
		}
		path := filepath.Join(dir, fmt.Sprintf("session.run%d.jsonl", i))
		if err := a.swapDefinitionJournal(path); err != nil {
			return test, err
		}
		pass, detail := a.runDefinitionTestOnce(ctx, test)
		results = append(results, DefinitionTestRunResult{
			Pass:       pass,
			Detail:     detail,
			Transcript: sessionMessages(a.History, 400),
		})
		if err := ctx.Err(); err != nil {
			return test, err
		}
		if !pass {
			break
		}
	}
	if a.Journal != nil && a.Journal.File != nil {
		_ = a.Journal.Close()
	}
	test.LastRuns = results
	return test, nil
}

func (a *Agent) runDefinitionTestOnce(ctx context.Context, test DefinitionTest) (pass bool, detail string) {
	run := &definitionTestRun{test: test}
	a.defTest = run
	defer func() { a.defTest = nil }()
	for i, step := range test.Steps {
		n := i + 1
		if step.From != "" {
			line, err := definitionTestWake(test, step.From, step.Text)
			if err != nil {
				return false, fmt.Sprintf("step %d: %s", n, err.Error())
			}
			if err := a.add(map[string]any{"role": "user", "content": line}); err != nil {
				return false, fmt.Sprintf("step %d: %s", n, err.Error())
			}
		}
		if step.To == "" {
			continue
		}
		if strings.TrimSpace(step.Check) == "" {
			return false, fmt.Sprintf("step %d: the check prompt is empty", n)
		}
		run.sent = nil
		if err := a.driveDefinitionExpect(ctx, step.To); err != nil {
			return false, fmt.Sprintf("step %d: %s", n, err.Error())
		}
		sent := run.sent
		if sent == nil || sent.to != step.To {
			got := ""
			if sent != nil {
				got = sent.to
			}
			return false, fmt.Sprintf("step %d: addressed %s, expected %s", n, got, step.To)
		}
		ok, reason, err := a.judgeDefinitionMessage(ctx, step.Check, sent.text)
		if err != nil {
			return false, fmt.Sprintf("step %d: %s", n, err.Error())
		}
		if !ok {
			return false, fmt.Sprintf("step %d: %s", n, reason)
		}
	}
	return true, "passed"
}

func (a *Agent) driveDefinitionExpect(ctx context.Context, expectTo string) error {
	for round := 0; round < definitionTestMaxRounds; round++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		resp, err := a.respondMain(ctx, a.modelInput(), a.chatExtraTools())
		if err != nil {
			return err
		}
		if len(resp.Output) == 0 {
			return fmt.Errorf("expected a message to %s", expectTo)
		}
		sawTool := false
		for _, raw := range resp.Output {
			if err := a.add(json.RawMessage(raw)); err != nil {
				return err
			}
			var item Item
			if json.Unmarshal(raw, &item) != nil || item.Type != "function_call" {
				continue
			}
			sawTool = true
			if result, ok := a.simulatedDefinitionTool(item); ok {
				if err := a.add(map[string]any{
					"type":    "function_call_output",
					"call_id": item.CallID,
					"output":  result,
				}); err != nil {
					return err
				}
				continue
			}
			if err := a.handleFunctionCall(ctx, "", bufio.NewReader(strings.NewReader("")), item); err != nil {
				return err
			}
		}
		if a.defTest != nil && a.defTest.sent != nil {
			return nil
		}
		if !sawTool {
			return fmt.Errorf("expected a message to %s", expectTo)
		}
	}
	return fmt.Errorf("expected a message to %s", expectTo)
}

func (a *Agent) judgeDefinitionMessage(ctx context.Context, check, message string) (bool, string, error) {
	if a != nil && a.defTestJudge != nil {
		return a.defTestJudge(check, message)
	}
	if a == nil || a.Client == nil {
		return false, "", fmt.Errorf("no model for the check")
	}
	model := strings.TrimSpace(a.Cfg.Model)
	reasoning := strings.TrimSpace(a.Cfg.Reasoning)
	if reasoning == "" {
		reasoning = "low"
	}
	if t := a.callChatTarget(); t.Model != "" {
		model = t.Model
		if strings.TrimSpace(t.Reasoning) != "" {
			reasoning = t.Reasoning
		}
	}
	raw, err := json.Marshal(map[string]any{
		"role":    "user",
		"content": "Check:\n" + check + "\n\nMessage:\n" + message,
	})
	if err != nil {
		return false, "", err
	}
	resp, err := a.modelClient(model).Respond(ctx, model, reasoning, []json.RawMessage{raw}, false, definitionJudgeInstructions, nil, nil)
	if err != nil {
		return false, "", err
	}
	return parseDefinitionJudge(responsePlainText(resp))
}

func responsePlainText(resp Response) string {
	var b strings.Builder
	for _, raw := range resp.Output {
		var item Item
		if json.Unmarshal(raw, &item) != nil {
			continue
		}
		for _, part := range item.Content {
			if part.Text != "" {
				b.WriteString(part.Text)
			}
		}
	}
	return b.String()
}

func parseDefinitionJudge(text string) (bool, string, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(strings.TrimSpace(text), "```")
	text = strings.TrimSpace(text)
	i := strings.Index(text, "{")
	j := strings.LastIndex(text, "}")
	if i < 0 || j < i {
		return false, "", fmt.Errorf("the check did not return a verdict")
	}
	var v struct {
		Pass   bool   `json:"pass"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(text[i:j+1]), &v); err != nil {
		return false, "", fmt.Errorf("the check did not return a verdict")
	}
	reason := strings.TrimSpace(v.Reason)
	if reason == "" {
		if v.Pass {
			reason = "passed"
		} else {
			reason = "missed the check"
		}
	}
	return v.Pass, reason, nil
}

func toolOutputFromHistory(history []json.RawMessage) string {
	for i := len(history) - 1; i >= 0; i-- {
		var item struct {
			Type   string `json:"type"`
			Output string `json:"output"`
		}
		if json.Unmarshal(history[i], &item) != nil {
			continue
		}
		if item.Type == "function_call_output" {
			return item.Output
		}
	}
	return ""
}
