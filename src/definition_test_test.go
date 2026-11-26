package main

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func definitionTestAgent(t *testing.T, instructions string, tools []string) (*Agent, AgentRecipe) {
	t.Helper()
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rec, err := createAgentRecipe("Subject", instructions)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setRecipeTools(rec.ID, tools); err != nil {
		t.Fatal(err)
	}
	rec, err = readAgentRecipe(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(rec, ws.ID, "Subject")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = deleteAgentInstance(inst.ID) })
	session := filepath.Join(t.TempDir(), "session.jsonl")
	j, err := OpenJournal(session, false)
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Cfg: Config{
			Model:             "test-model",
			Reasoning:         "low",
			AgentInstanceID:   inst.ID,
			AgentInstructions: rec.Instructions,
			WorkspaceID:       ws.ID,
			WorkspaceRoot:     ws.Root,
			SessionPath:       session,
			ThreadID:          "main",
		},
		Client:  NewClient("test"),
		Journal: j,
		History: j.History,
	}
	a.defTestJudge = func(check, message string) (bool, string, error) {
		if strings.Contains(strings.ToLower(check), "green") {
			if strings.Contains(strings.ToLower(message), "green") {
				return true, "passed", nil
			}
			return false, "the status is missing", nil
		}
		if strings.TrimSpace(message) == "" {
			return false, "empty message", nil
		}
		return true, "passed", nil
	}
	return a, rec
}

func messageCall(id, text, callID string) Response {
	args, _ := json.Marshal(map[string]string{"instance_id": id, "text": text})
	raw, _ := json.Marshal(map[string]any{
		"type":      "function_call",
		"name":      "message_agent",
		"call_id":   callID,
		"arguments": string(args),
	})
	return Response{Output: []json.RawMessage{raw}, Status: "completed"}
}

func toolCall(name, args, callID string) Response {
	raw, _ := json.Marshal(map[string]any{
		"type":      "function_call",
		"name":      name,
		"call_id":   callID,
		"arguments": args,
	})
	return Response{Output: []json.RawMessage{raw}, Status: "completed"}
}

func historyUserTexts(history []json.RawMessage) []string {
	var out []string
	for _, raw := range history {
		var item struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		if json.Unmarshal(raw, &item) != nil {
			continue
		}
		if strings.EqualFold(item.Role, "user") {
			out = append(out, item.Content)
		}
	}
	return out
}

func TestDefinitionTestRecipientComesFromTheScriptOrTheDefinition(t *testing.T) {
	cases := []struct {
		name         string
		instructions string
		test         DefinitionTest
		reply        func(instructions string, users []string, call int) Response
	}{
		{
			name:         "message names the recipient",
			instructions: "Do the job in the message.",
			test: DefinitionTest{
				Name: "message",
				Parties: []DefinitionTestParty{
					{ID: "requester", Name: "Requester", Kind: "thread"},
					{ID: "party-b", Name: "Party B", Kind: "agent"},
				},
				Steps: []DefinitionTestStep{{
					From: "requester", Text: "answer party-b", To: "party-b", Check: "answers",
				}},
			},
			reply: func(instructions string, users []string, call int) Response {
				last := ""
				if len(users) > 0 {
					last = users[len(users)-1]
				}
				if strings.Contains(last, "answer party-b") {
					return messageCall("party-b", "ok", "c1")
				}
				return messageCall("requester", "ok", "c1")
			},
		},
		{
			name:         "definition names the standing recipient",
			instructions: "write status to party-a",
			test: DefinitionTest{
				Name: "standing",
				Parties: []DefinitionTestParty{
					{ID: "party-a", Name: "Party A", Kind: "agent"},
					{ID: "party-b", Name: "Party B", Kind: "agent"},
				},
				Steps: []DefinitionTestStep{{
					From: "party-b", Text: "hello", To: "party-a", Check: "answers",
				}},
			},
			reply: func(instructions string, users []string, call int) Response {
				if strings.Contains(instructions, "write status to party-a") {
					return messageCall("party-a", "ok", "c1")
				}
				return messageCall("party-b", "ok", "c1")
			},
		},
		{
			name:         "a later party does not replace the standing recipient",
			instructions: "report results to party-a",
			test: DefinitionTest{
				Name: "relay",
				Parties: []DefinitionTestParty{
					{ID: "party-a", Name: "Party A", Kind: "thread"},
					{ID: "party-b", Name: "Party B", Kind: "agent"},
				},
				Steps: []DefinitionTestStep{
					{From: "party-a", Text: "Ask party-b for the status.", To: "party-b", Check: "answers"},
					{From: "party-b", Text: "green", To: "party-a", Check: "green"},
				},
			},
			reply: func(instructions string, users []string, call int) Response {
				last := ""
				if len(users) > 0 {
					last = users[len(users)-1]
				}
				if strings.Contains(last, "Ask party-b") {
					return messageCall("party-b", "what is the status?", "c"+string(rune('0'+call)))
				}
				if strings.Contains(instructions, "report results to party-a") {
					return messageCall("party-a", "status is green", "c"+string(rune('0'+call)))
				}
				return messageCall("party-b", "ok", "c1")
			},
		},
		{
			name:         "no further message",
			instructions: "send a status to party-a at the start",
			test: DefinitionTest{
				Name:    "start",
				Parties: []DefinitionTestParty{{ID: "party-a", Name: "Party A", Kind: "agent"}},
				Steps:   []DefinitionTestStep{{To: "party-a", Check: "answers"}},
			},
			reply: func(instructions string, users []string, call int) Response {
				if len(users) == 0 && strings.Contains(instructions, "send a status to party-a at the start") {
					return messageCall("party-a", "ok", "c1")
				}
				return messageCall("party-a", "late", "c1")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := definitionTestAgent(t, tc.instructions, []string{"message_agent"})
			var seen string
			var hinted bool
			calls := 0
			a.Client.ScriptReply = func(ctx context.Context, instructions string, history []json.RawMessage) (Response, error) {
				calls++
				seen = instructions
				blob, _ := json.Marshal(history)
				if strings.Contains(string(blob), "Call message_agent. instance_id:") {
					hinted = true
				}
				return tc.reply(instructions, historyUserTexts(history), calls), nil
			}
			before, _ := listAgentInstances()
			updated, err := a.runDefinitionTest(context.Background(), tc.test)
			if err != nil {
				t.Fatal(err)
			}
			after, _ := listAgentInstances()
			if len(after) != len(before) {
				t.Fatalf("instances %d to %d", len(before), len(after))
			}
			if hinted {
				t.Fatal("the test named a recipient")
			}
			if seen != a.modelInstructions() {
				t.Fatal("the test changed the instructions")
			}
			if !strings.Contains(seen, tc.instructions) {
				t.Fatal("definition instructions missing")
			}
			if !definitionTestPassed(updated.LastRuns) {
				t.Fatalf("runs: %+v", updated.LastRuns)
			}
		})
	}
}

func TestDefinitionTestTwoScriptsRunSeparately(t *testing.T) {
	a, rec := definitionTestAgent(t, "Answer.", []string{"message_agent"})
	a.Client.ScriptReply = func(ctx context.Context, instructions string, history []json.RawMessage) (Response, error) {
		return messageCall("requester", "ok", "c1"), nil
	}
	first := DefinitionTest{
		Name:    "first",
		Parties: []DefinitionTestParty{{ID: "requester", Name: "Requester", Kind: "thread"}},
		Steps:   []DefinitionTestStep{{From: "requester", Text: "go", To: "requester", Check: "answers"}},
	}
	second := DefinitionTest{
		Name:    "second",
		Parties: []DefinitionTestParty{{ID: "requester", Name: "Requester", Kind: "thread"}},
		Steps:   []DefinitionTestStep{{From: "requester", Text: "stop", To: "requester", Check: "answers"}},
	}
	var err error
	rec, err = saveDefinitionTest(rec.ID, first)
	if err != nil {
		t.Fatal(err)
	}
	rec, err = saveDefinitionTest(rec.ID, second)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Tests) != 2 {
		t.Fatalf("tests: %+v", rec.Tests)
	}
	updated, err := a.runDefinitionTest(context.Background(), rec.Tests[0])
	if err != nil {
		t.Fatal(err)
	}
	for i := range rec.Tests {
		if rec.Tests[i].ID == updated.ID {
			rec.Tests[i].LastRuns = updated.LastRuns
		}
	}
	if err := writeAgentRecipe(rec); err != nil {
		t.Fatal(err)
	}
	rec, err = readAgentRecipe(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !definitionTestPassed(rec.Tests[0].LastRuns) {
		t.Fatalf("first: %+v", rec.Tests[0].LastRuns)
	}
	if len(rec.Tests[1].LastRuns) != 0 {
		t.Fatalf("second test ran: %+v", rec.Tests[1].LastRuns)
	}
	plain, err := createAgentRecipe("Plain", "Do.")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setRecipeTools(plain.ID, []string{"message_agent"}); err != nil {
		t.Fatal(err)
	}
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createAgentInstance(plain, ws.ID, "Plain"); err != nil {
		t.Fatal(err)
	}
}

func TestDefinitionTestWordingAndAddress(t *testing.T) {
	a, _ := definitionTestAgent(t, "Report green.", []string{"message_agent"})
	for _, wording := range []string{"Status is green", "All green here"} {
		ok, _, err := a.judgeDefinitionMessage(context.Background(), "reports green", wording)
		if err != nil || !ok {
			t.Fatalf("%q: %v %v", wording, ok, err)
		}
	}
	ok, reason, err := a.judgeDefinitionMessage(context.Background(), "reports green", "The server is down")
	if err != nil || ok || !strings.Contains(reason, "status is missing") {
		t.Fatalf("miss: %v %q %v", ok, reason, err)
	}
	a.Client.ScriptReply = func(ctx context.Context, instructions string, history []json.RawMessage) (Response, error) {
		return messageCall("requester", "The server is down", "c1"), nil
	}
	updated, err := a.runDefinitionTest(context.Background(), DefinitionTest{
		Name:    "wording",
		Parties: []DefinitionTestParty{{ID: "requester", Name: "Requester", Kind: "thread"}},
		Steps:   []DefinitionTestStep{{From: "requester", Text: "status?", To: "requester", Check: "reports green"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if definitionTestPassed(updated.LastRuns) {
		t.Fatal("a miss passed")
	}
	for _, run := range updated.LastRuns {
		if run.Pass || !strings.Contains(run.Detail, "step 1") {
			t.Fatalf("detail: %+v", run)
		}
	}
	a.Client.ScriptReply = func(ctx context.Context, instructions string, history []json.RawMessage) (Response, error) {
		return messageCall("other", "status is green", "c1"), nil
	}
	updated, err = a.runDefinitionTest(context.Background(), DefinitionTest{
		Name: "address",
		Parties: []DefinitionTestParty{
			{ID: "requester", Name: "Requester", Kind: "thread"},
			{ID: "other", Name: "Other", Kind: "agent"},
		},
		Steps: []DefinitionTestStep{{From: "requester", Text: "status?", To: "requester", Check: "reports green"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if definitionTestPassed(updated.LastRuns) {
		t.Fatal("wrong address passed")
	}
	if !strings.Contains(updated.LastRuns[0].Detail, "expected requester") {
		t.Fatalf("detail: %+v", updated.LastRuns[0])
	}
}

func TestDefinitionTestSimulatedTools(t *testing.T) {
	a, _ := definitionTestAgent(t, "Use the tool, then answer.", []string{"exec", "read_file", "message_agent"})
	sentinel := filepath.Join(t.TempDir(), "touched")
	var outputs []string
	a.Client.ScriptReply = func(ctx context.Context, instructions string, history []json.RawMessage) (Response, error) {
		for _, raw := range history {
			var item struct {
				Type   string `json:"type"`
				Output string `json:"output"`
			}
			if json.Unmarshal(raw, &item) == nil && item.Type == "function_call_output" {
				outputs = append(outputs, item.Output)
				if item.Output == "SIMULATED-RESULT" {
					return messageCall("requester", "saw it", "c2"), nil
				}
				if item.Output == "" {
					return messageCall("requester", "continued", "c2"), nil
				}
			}
		}
		args, _ := json.Marshal(map[string]any{
			"command": "touch " + sentinel, "timeout_seconds": 1, "interactive": false, "raw_output": false, "output_focus": "",
		})
		return toolCall("exec", string(args), "c1"), nil
	}
	updated, err := a.runDefinitionTest(context.Background(), DefinitionTest{
		Name:    "sim",
		Parties: []DefinitionTestParty{{ID: "requester", Name: "Requester", Kind: "thread"}},
		Steps:   []DefinitionTestStep{{From: "requester", Text: "run", To: "requester", Check: "answers"}},
		Tools:   map[string]definitionToolResults{"exec": {"SIMULATED-RESULT"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !definitionTestPassed(updated.LastRuns) {
		t.Fatalf("runs: %+v outputs %q", updated.LastRuns, outputs)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("exec ran on the host: %v", err)
	}

	a.Client.ScriptReply = func(ctx context.Context, instructions string, history []json.RawMessage) (Response, error) {
		for _, raw := range history {
			var item struct {
				Type   string `json:"type"`
				Output string `json:"output"`
			}
			if json.Unmarshal(raw, &item) == nil && item.Type == "function_call_output" {
				return messageCall("requester", "continued", "c2"), nil
			}
		}
		return toolCall("read_file", `{"path":"missing"}`, "c1"), nil
	}
	updated, err = a.runDefinitionTest(context.Background(), DefinitionTest{
		Name:    "empty",
		Parties: []DefinitionTestParty{{ID: "requester", Name: "Requester", Kind: "thread"}},
		Steps:   []DefinitionTestStep{{From: "requester", Text: "read", To: "requester", Check: "answers"}},
		Tools:   map[string]definitionToolResults{"read_file": {""}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !definitionTestPassed(updated.LastRuns) {
		t.Fatalf("empty result failed the agent: %+v", updated.LastRuns)
	}

	a.Client.ScriptReply = func(ctx context.Context, instructions string, history []json.RawMessage) (Response, error) {
		for _, raw := range history {
			var item struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(raw, &item) == nil && item.Type == "function_call_output" {
				return messageCall("requester", "The server is down", "c2"), nil
			}
		}
		return toolCall("read_file", `{"path":"missing"}`, "c1"), nil
	}
	updated, err = a.runDefinitionTest(context.Background(), DefinitionTest{
		Name:    "empty-then-miss",
		Parties: []DefinitionTestParty{{ID: "requester", Name: "Requester", Kind: "thread"}},
		Steps:   []DefinitionTestStep{{From: "requester", Text: "read", To: "requester", Check: "reports green"}},
		Tools:   map[string]definitionToolResults{"read_file": {""}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if definitionTestPassed(updated.LastRuns) || !strings.Contains(updated.LastRuns[0].Detail, "step 1") {
		t.Fatalf("runs: %+v", updated.LastRuns)
	}
}

func TestDefinitionTestOrdinaryRefusal(t *testing.T) {
	a, _ := definitionTestAgent(t, "Answer.", []string{"message_agent"})
	ordinary := func(name, args string) string {
		j, err := OpenJournal(filepath.Join(t.TempDir(), name+".jsonl"), false)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = j.Close() })
		twin := &Agent{Cfg: Config{AgentInstanceID: a.Cfg.AgentInstanceID}, Journal: j, History: nil}
		item := Item{Type: "function_call", Name: name, CallID: "ord", Arguments: args}
		if err := twin.add(item); err != nil {
			t.Fatal(err)
		}
		if err := twin.handleFunctionCall(context.Background(), "", bufio.NewReader(strings.NewReader("")), item); err != nil {
			t.Fatal(err)
		}
		return toolOutputFromHistory(twin.History)
	}
	for _, call := range []struct{ name, args string }{
		{"exec", `{"command":"true","timeout_seconds":1,"interactive":false,"raw_output":false,"output_focus":""}`},
		{"not_a_tool", `{}`},
	} {
		want := ordinary(call.name, call.args)
		var got string
		a.Client.ScriptReply = func(ctx context.Context, instructions string, history []json.RawMessage) (Response, error) {
			for _, raw := range history {
				var item struct {
					Type   string `json:"type"`
					Output string `json:"output"`
				}
				if json.Unmarshal(raw, &item) == nil && item.Type == "function_call_output" {
					got = item.Output
					return messageCall("requester", "ok", "c2"), nil
				}
			}
			return toolCall(call.name, call.args, "c1"), nil
		}
		updated, err := a.runDefinitionTest(context.Background(), DefinitionTest{
			Name:    call.name,
			Parties: []DefinitionTestParty{{ID: "requester", Name: "Requester", Kind: "thread"}},
			Steps:   []DefinitionTestStep{{From: "requester", Text: "go", To: "requester", Check: "answers"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !definitionTestPassed(updated.LastRuns) {
			t.Fatalf("%s runs: %+v", call.name, updated.LastRuns)
		}
		if got != want {
			t.Fatalf("%s reply %q, ordinary %q", call.name, got, want)
		}
	}
}

func TestDefinitionTestOffersTheGrantedMessageTool(t *testing.T) {
	a, _ := definitionTestAgent(t, "Answer with a joke.", []string{"message_agent"})
	a.Client.ScriptReply = func(ctx context.Context, instructions string, history []json.RawMessage) (Response, error) {
		return messageCall("requester", "a joke", "c1"), nil
	}
	_, err := a.runDefinitionTest(context.Background(), DefinitionTest{
		Name:    "joke",
		Parties: []DefinitionTestParty{{ID: "requester", Name: "Requester", Kind: "thread"}},
		Steps:   []DefinitionTestStep{{From: "requester", Text: "how is the weather", To: "requester", Check: "a joke"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, name := range a.Client.offeredToolNames {
		got[name] = true
	}
	if !got["message_agent"] {
		t.Fatalf("offered tools: %v", a.Client.offeredToolNames)
	}
	if got["exec"] {
		t.Fatalf("exec is not granted: %v", a.Client.offeredToolNames)
	}
}

func TestDefinitionTestAllThreeRunsMustPass(t *testing.T) {
	a, _ := definitionTestAgent(t, "Answer.", []string{"message_agent"})
	n := 0
	a.Client.ScriptReply = func(ctx context.Context, instructions string, history []json.RawMessage) (Response, error) {
		n++
		if n == 2 {
			return messageCall("requester", "The server is down", "c1"), nil
		}
		return messageCall("requester", "status is green", "c1"), nil
	}
	updated, err := a.runDefinitionTest(context.Background(), DefinitionTest{
		Name:    "three",
		Parties: []DefinitionTestParty{{ID: "requester", Name: "Requester", Kind: "thread"}},
		Steps:   []DefinitionTestStep{{From: "requester", Text: "status?", To: "requester", Check: "reports green"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.LastRuns) != 2 {
		t.Fatalf("runs: %+v", updated.LastRuns)
	}
	if !updated.LastRuns[0].Pass || updated.LastRuns[1].Pass {
		t.Fatalf("runs: %+v", updated.LastRuns)
	}
	if n != 2 {
		t.Fatalf("model calls after the failed run: %d", n)
	}
	if definitionTestPassed(updated.LastRuns) {
		t.Fatal("one miss still passed the test")
	}
}

func TestDefinitionTestEmptyToolResultRoundTrip(t *testing.T) {
	withTempAgentHomes(t)
	rec, err := createAgentRecipe("Subject", "Do.")
	if err != nil {
		t.Fatal(err)
	}
	rec, err = saveDefinitionTest(rec.ID, DefinitionTest{
		Name:    "empty",
		Parties: []DefinitionTestParty{{ID: "requester", Name: "Requester", Kind: "thread"}},
		Steps:   []DefinitionTestStep{{To: "requester", Check: "answers"}},
		Tools:   map[string]definitionToolResults{"read_file": {""}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := rec.Tests[0].Tools["read_file"]
	if len(got) != 1 || got[0] != "" {
		t.Fatalf("tools: %#v", rec.Tests[0].Tools)
	}
	renamed, err := updateAgentRecipe(rec.ID, "Renamed", rec.Instructions, true)
	if err != nil {
		t.Fatal(err)
	}
	kept := renamed.Tests
	if len(kept) != 1 || len(kept[0].Tools["read_file"]) != 1 || kept[0].Tools["read_file"][0] != "" {
		t.Fatalf("update dropped the test: %+v", renamed.Tests)
	}
}

func TestDefinitionToolResultsAcceptAStringOrAList(t *testing.T) {
	var one DefinitionTest
	if err := json.Unmarshal([]byte(`{"name":"n","tools":{"read_file":""}}`), &one); err != nil {
		t.Fatal(err)
	}
	if len(one.Tools["read_file"]) != 1 || one.Tools["read_file"][0] != "" {
		t.Fatalf("string: %#v", one.Tools)
	}
	var two DefinitionTest
	if err := json.Unmarshal([]byte(`{"name":"n","tools":{"exec":["one","two"]}}`), &two); err != nil {
		t.Fatal(err)
	}
	if len(two.Tools["exec"]) != 2 || two.Tools["exec"][0] != "one" || two.Tools["exec"][1] != "two" {
		t.Fatalf("list: %#v", two.Tools)
	}
}

func TestDefinitionTestToolCallsInOrderAndKeepsChat(t *testing.T) {
	a, _ := definitionTestAgent(t, "Use the tool twice, then answer.", []string{"exec", "message_agent"})
	var outputs []string
	a.Client.ScriptReply = func(ctx context.Context, instructions string, history []json.RawMessage) (Response, error) {
		outputs = outputs[:0]
		for _, raw := range history {
			var item struct {
				Type   string `json:"type"`
				Output string `json:"output"`
			}
			if json.Unmarshal(raw, &item) == nil && item.Type == "function_call_output" {
				outputs = append(outputs, item.Output)
			}
		}
		if len(outputs) >= 2 {
			return messageCall("requester", "status is green", "c3"), nil
		}
		return toolCall("exec", `{"command":"true","timeout_seconds":1,"interactive":false,"raw_output":false,"output_focus":""}`, "c"+string(rune('1'+len(outputs)))), nil
	}
	updated, err := a.runDefinitionTest(context.Background(), DefinitionTest{
		Name:    "order",
		Parties: []DefinitionTestParty{{ID: "requester", Name: "Requester", Kind: "thread"}},
		Steps:   []DefinitionTestStep{{From: "requester", Text: "check", To: "requester", Check: "reports green"}},
		Tools:   map[string]definitionToolResults{"exec": {"one", "two"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !definitionTestPassed(updated.LastRuns) {
		t.Fatalf("runs: %+v outputs %q", updated.LastRuns, outputs)
	}
	if len(outputs) < 2 || outputs[0] != "one" || outputs[1] != "two" {
		t.Fatalf("outputs: %q", outputs)
	}
	chat := updated.LastRuns[0].Transcript
	var sawTool, sawPeer bool
	for _, msg := range chat {
		if msg.Role == "tool_group" && msg.Name == "exec" && msg.Result == "one" {
			sawTool = true
		}
		if msg.Role == "peer" && strings.Contains(msg.Text, "check") {
			sawPeer = true
		}
	}
	if !sawTool || !sawPeer {
		t.Fatalf("chat: %+v", chat)
	}
}
