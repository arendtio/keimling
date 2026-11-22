package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGateToolCallAgentAutoReviewsEditsAndMCP(t *testing.T) {
	edit := gateToolCall(true, "auto", "edit_file", `{"path":"policy.go"}`, false)
	if !edit.Review || edit.Allow || edit.Ask || edit.Deny {
		t.Fatalf("agent edit: %+v", edit)
	}
	mcp := gateToolCall(true, "auto", "mcp__demo__echo", `{}`, false)
	if !mcp.Review {
		t.Fatalf("agent mcp: %+v", mcp)
	}
	if g := gateToolCall(true, "auto", "mcp__demo__echo", `{}`, true); !g.Allow {
		t.Fatalf("whitelisted agent mcp: %+v", g)
	}
	read := gateToolCall(true, "auto", "read_file", `{"path":"policy.go"}`, false)
	if !read.Allow {
		t.Fatalf("agent read: %+v", read)
	}
}

func TestGateToolCallWorkspaceKeepsTheFloor(t *testing.T) {
	edit := gateToolCall(false, "auto", "edit_file", `{"path":"policy.go"}`, false)
	if !edit.Allow {
		t.Fatalf("workspace edit: %+v", edit)
	}
	outside := gateToolCall(false, "auto", "edit_file", `{"path":"../outside.go"}`, false)
	if !outside.Ask {
		t.Fatalf("outside edit: %+v", outside)
	}
	history := gateToolCall(false, "ask", "search_history", `{"query":"x"}`, false)
	if !history.Allow {
		t.Fatalf("search history: %+v", history)
	}
	mcp := gateToolCall(false, "auto", "mcp__demo__echo", `{}`, false)
	if !mcp.Ask || mcp.Reason != "MCP tool call" {
		t.Fatalf("workspace mcp: %+v", mcp)
	}
	if g := gateToolCall(false, "ask", "mcp__demo__echo", `{}`, true); !g.Allow {
		t.Fatalf("whitelisted ask: %+v", g)
	}
	if g := gateToolCall(false, "ask", "mcp__demo__echo", `{}`, false); !g.Ask {
		t.Fatalf("ask mcp: %+v", g)
	}
	if g := gateToolCall(true, "yolo", "mcp__demo__echo", `{}`, false); !g.Allow {
		t.Fatalf("yolo: %+v", g)
	}
}

func TestSettleReviewAgentRefusesInsteadOfAsking(t *testing.T) {
	denied := settleReview(true, true, "ask", "touches ssh", nil)
	if !denied.Deny || denied.Ask || denied.Reason != "touches ssh" {
		t.Fatalf("deny: %+v", denied)
	}
	down := settleReview(true, false, "", "", errReviewDown)
	if !down.Deny || !strings.Contains(down.Reason, "review unavailable") {
		t.Fatalf("down: %+v", down)
	}
	asked := settleReview(false, true, "ask", "touches ssh", nil)
	if !asked.Ask || asked.Deny {
		t.Fatalf("workspace ask: %+v", asked)
	}
	if g := settleReview(true, true, "approve", "", nil); !g.Allow {
		t.Fatalf("approve: %+v", g)
	}
}

var errReviewDown = errString("down")

type errString string

func (e errString) Error() string { return string(e) }

func TestConfirmedPurposeAllowsTheNamedWatch(t *testing.T) {
	job := "Monitor server 192.168.0.101 using a recurring watch every 20 seconds running `ping -c 1 -W 2 192.168.0.101`. After each check, if the command succeeds, use message_agent to send a short German message that the server is reachable to thread 20261107-110041-595951692."
	inner, _ := json.Marshal(map[string]any{"command": "ping -c 1 -W 2 192.168.0.101", "interactive": false})
	watchBytes, _ := json.Marshal(map[string]any{
		"command": "", "condition": "", "message": "Bei exit_code 0 melden.", "path": "", "source": "",
		"steps": []map[string]string{{"name": "exec", "arguments": string(inner)}},
		"value": "", "when": "every 20s",
	})
	watch := string(watchBytes)
	if !agentPurposeClears(true, "auto", job, "create_watch", watch) {
		t.Fatal("named ping was not allowed")
	}
	other := strings.ReplaceAll(watch, "192.168.0.101", "10.0.0.8")
	if agentPurposeClears(true, "auto", job, "create_watch", other) {
		t.Fatal("a different host was allowed")
	}
	if agentPurposeClears(true, "auto", "Monitor the network and inform the user.", "create_watch", watch) {
		t.Fatal("a vague job allowed the ping")
	}
	if agentPurposeClears(false, "auto", job, "create_watch", watch) {
		t.Fatal("a workspace thread was granted the job")
	}
	if agentPurposeClears(true, "ask", job, "create_watch", watch) {
		t.Fatal("ask mode skipped the question")
	}
	rules, user := reviewRequest("", "create_watch", watch, "/tmp", job, nil, false)
	if !strings.Contains(rules, "AGENT PURPOSE is present") || !strings.Contains(user, "192.168.0.101") {
		t.Fatal("review did not receive the job")
	}
	plain, _ := reviewRequest("", "create_watch", watch, "/tmp", "", nil, false)
	if strings.Contains(plain, "AGENT PURPOSE is present") {
		t.Fatal("a review without a job was given the purpose rule")
	}
}

func TestReviewRulesReplaceTheFactoryPrompt(t *testing.T) {
	if reviewRules("", false) != defaultPolicy {
		t.Fatal("empty policy should be the factory prompt")
	}
	got := reviewRules("ONLY THIS", true)
	if got != "ONLY THIS" || strings.Contains(got, "ALWAYS APPROVE") {
		t.Fatalf("replaced prompt: %q", got)
	}
	added := reviewRules("extra", false)
	if !strings.Contains(added, "ALWAYS APPROVE") || !strings.Contains(added, "extra") {
		t.Fatal("additional policy should keep the factory prompt")
	}
}

func TestRecipeApprovalModeAndPrompt(t *testing.T) {
	withTempAgentHomes(t)
	rec, err := createAgentRecipe("Reviewer", "Look at the diff.")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Mode != "auto" {
		t.Fatalf("default mode %q", rec.Mode)
	}
	rec, err = setRecipeApproval(rec.ID, "ask", nil, false)
	if err != nil || rec.Mode != "ask" {
		t.Fatalf("mode: %+v %v", rec, err)
	}
	text := "Approve only reads."
	rec, err = setRecipeApproval(rec.ID, "auto", &text, false)
	if err != nil || rec.ReviewPrompt != text || rec.Mode != "auto" {
		t.Fatalf("prompt: %+v %v", rec, err)
	}
	factory := defaultPolicy
	rec, err = setRecipeApproval(rec.ID, "auto", &factory, false)
	if err != nil || rec.ReviewPrompt != "" {
		t.Fatalf("factory text should stay bound: %+v %v", rec, err)
	}
	blank := "  "
	if _, err := setRecipeApproval(rec.ID, "auto", &blank, false); err == nil {
		t.Fatal("empty prompt should be refused")
	}
	rec, err = setRecipeApproval(rec.ID, "auto", &text, false)
	if err != nil {
		t.Fatal(err)
	}
	rec, err = setRecipeApproval(rec.ID, "", nil, true)
	if err != nil || rec.ReviewPrompt != "" || rec.Mode != "auto" {
		t.Fatalf("reset: %+v %v", rec, err)
	}
	inst := AgentInstanceMeta{ID: "agent-test", RecipeID: rec.ID, Name: "One"}
	if err := writeAgentInstanceMeta(inst); err != nil {
		t.Fatal(err)
	}
	rec, err = setRecipeApproval(rec.ID, "yolo", &text, false)
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{AgentInstanceID: inst.ID, Mode: "ask", ConversationMode: "ask"}}
	a.applyDefinitionApproval()
	if a.Cfg.Mode != "yolo" || !a.Cfg.ApprovalPolicyReplace || a.Cfg.ApprovalPolicy != text {
		t.Fatalf("applied: mode %q replace %v policy %q", a.Cfg.Mode, a.Cfg.ApprovalPolicyReplace, a.Cfg.ApprovalPolicy)
	}
	if a.Cfg.storedMode() != "ask" {
		t.Fatalf("conversation mode changed: %q", a.Cfg.storedMode())
	}
}

func TestMCPToolAllowRoundTrip(t *testing.T) {
	withTempAgentHomes(t)
	if err := UpsertUserMCPServer("allow-demo", MCPServerConfig{Command: "true"}); err != nil {
		t.Fatal(err)
	}
	if _, err := setMCPToolAllow("allow-demo", "mcp__allow-demo__list", true, ""); err != nil {
		t.Fatal(err)
	}
	if !mcpToolAlwaysAllowed("", "mcp__allow-demo__list") {
		t.Fatal("marked tool should be allowed")
	}
	if mcpToolAlwaysAllowed("", "mcp__allow-demo__write") {
		t.Fatal("unmarked tool should still ask")
	}
	if err := UpsertUserMCPServer("allow-demo", MCPServerConfig{Command: "true", Args: []string{"-n"}}); err != nil {
		t.Fatal(err)
	}
	if !mcpToolAlwaysAllowed("", "mcp__allow-demo__list") {
		t.Fatal("saving the server again should keep the mark")
	}
	if _, err := setMCPToolAllow("allow-demo", "mcp__allow-demo__list", false, ""); err != nil {
		t.Fatal(err)
	}
	if mcpToolAlwaysAllowed("", "mcp__allow-demo__list") {
		t.Fatal("cleared mark should ask again")
	}
}

func TestCallCommandsReadsFilePaths(t *testing.T) {
	if got := callCommands("write_file", `{"path":"src/a.go","content":"x"}`); len(got) != 1 || got[0] != "src/a.go" {
		t.Fatalf("write_file: %v", got)
	}
	if got := callCommands("edit_file", `{"path":" src/b.go ","old_string":"a","new_string":"b"}`); len(got) != 1 || got[0] != "src/b.go" {
		t.Fatalf("edit_file: %v", got)
	}
	for _, args := range []string{`{"path":""}`, `{}`, `{"content":"x"}`, `not json`} {
		if got := callCommands("write_file", args); got != nil {
			t.Fatalf("write_file %s: %v", args, got)
		}
		if got := callCommands("edit_file", args); got != nil {
			t.Fatalf("edit_file %s: %v", args, got)
		}
	}
}

func TestPurposeAllowsANamedFilePath(t *testing.T) {
	job := "Read the diff and fix it." + "\n\nERLAUBTE PFADE:\n- src/policy.go\n- src/agent_approval_test.go"
	write := `{"path":"src/policy.go","content":"package main\n"}`
	if !agentPurposeClears(true, "auto", job, "write_file", write) {
		t.Fatal("a named file was not allowed")
	}
	edit := `{"path":"src/agent_approval_test.go","old_string":"a","new_string":"b"}`
	if !agentPurposeClears(true, "auto", job, "edit_file", edit) {
		t.Fatal("a named edit was not allowed")
	}
	other := `{"path":"src/main.go","content":"package main\n"}`
	if agentPurposeClears(true, "auto", job, "write_file", other) {
		t.Fatal("an unnamed file was allowed")
	}
	inside := `{"path":"/data/work/keimling-wt/011/src/policy.go","content":"package main\n"}`
	if !agentPurposeClears(true, "auto", job, "write_file", inside) {
		t.Fatal("an absolute path inside the job was not allowed")
	}
	elsewhere := `{"path":"/etc/policy.go","content":"x"}`
	if agentPurposeClears(true, "auto", job, "write_file", elsewhere) {
		t.Fatal("a bare file name allowed another directory")
	}
	if agentPurposeClears(true, "auto", "Rewrite the code.", "write_file", write) {
		t.Fatal("a vague job allowed the write")
	}
	if agentPurposeClears(true, "ask", job, "write_file", write) {
		t.Fatal("ask mode skipped the question")
	}
	if agentPurposeClears(false, "auto", job, "write_file", write) {
		t.Fatal("a workspace thread was granted the job")
	}
}

func TestAgentPurposeKeepsTheRolePromptAndAddsTheTask(t *testing.T) {
	role := "You are a focused Keimling agent."
	job := "Fix the reviewer purpose." + "\n\nERLAUBTE PFADE:\n- src/policy.go"
	got := agentPurpose(role, job)
	if !strings.HasPrefix(got, role) || !strings.Contains(got, "AUFTRAG:") || !strings.Contains(got, "src/policy.go") {
		t.Fatalf("purpose: %q", got)
	}
	if agentPurpose(role, "") != role {
		t.Fatal("a thread without a task lost the role prompt")
	}
	long := agentPurpose(role, strings.Repeat("x", agentPurposeTaskLimit+50))
	if strings.Contains(long, strings.Repeat("x", agentPurposeTaskLimit+1)) {
		t.Fatal("a long task was not cut")
	}
	if !strings.HasPrefix(long, role+"\n\nAUFTRAG:\n") {
		t.Fatal("a long task lost the role prompt")
	}
}

// purposeTurn builds one run-history turn for the purpose tests.
func purposeTurn(t *testing.T, role, content string) json.RawMessage {
	t.Helper()
	row, err := json.Marshal(map[string]any{"role": role, "content": content})
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func TestReviewPurposeUsesTheLatestTask(t *testing.T) {
	first := "Read the diff of branch 004 and report."
	second := "Fix the reviewer purpose.\nERLAUBTE PFADE:\n- src/policy.go"
	a := &Agent{Cfg: Config{AgentInstanceID: "agent-1", AgentInstructions: "ROLE-PROMPT"}}
	a.History = []json.RawMessage{
		purposeTurn(t, "user", first),
		purposeTurn(t, "assistant", "The branch is already merged."),
		purposeTurn(t, "user", second),
		purposeTurn(t, "assistant", "Working on it."),
	}
	got := a.reviewPurpose()
	if !strings.HasPrefix(got, "ROLE-PROMPT\n\nAUFTRAG:\n") || !strings.Contains(got, second) {
		t.Fatalf("purpose: %q", got)
	}
	if strings.Contains(got, first) {
		t.Fatalf("a finished task stayed in the purpose: %q", got)
	}
}

func TestReviewPurposeOfASingleTaskIsUnchanged(t *testing.T) {
	task := "Fix the reviewer purpose.\nERLAUBTE PFADE:\n- src/policy.go"
	a := &Agent{Cfg: Config{AgentInstanceID: "agent-1", AgentInstructions: "ROLE-PROMPT"}}
	a.History = []json.RawMessage{
		purposeTurn(t, "user", task),
		purposeTurn(t, "assistant", "Done."),
	}
	if got, want := a.reviewPurpose(), agentPurpose("ROLE-PROMPT", task); got != want {
		t.Fatalf("purpose: %q, want %q", got, want)
	}
	// An instance the user never tasked keeps the bare role prompt.
	b := &Agent{Cfg: Config{AgentInstanceID: "agent-2", AgentInstructions: "ROLE-PROMPT"}}
	if got := b.reviewPurpose(); got != "ROLE-PROMPT" {
		t.Fatalf("a taskless instance: %q", got)
	}
	if p := (&Agent{Cfg: Config{AgentInstructions: "ROLE-PROMPT"}}).reviewPurpose(); p != "" {
		t.Fatalf("a workspace thread got a purpose: %q", p)
	}
}

func TestReviewPurposeSkipsInternalTurns(t *testing.T) {
	task := "Fix the reviewer purpose.\nERLAUBTE PFADE:\n- src/policy.go"
	a := &Agent{Cfg: Config{AgentInstanceID: "agent-1", AgentInstructions: "ROLE-PROMPT"}}
	a.History = []json.RawMessage{
		purposeTurn(t, "user", "[AUTOPILOT wake] keep the plan moving"),
		purposeTurn(t, "assistant", "On it."),
		purposeTurn(t, "user", task),
		purposeTurn(t, "user", "[User steering during ongoing task] hurry up"),
		purposeTurn(t, "user", "[Watch] the runner finished"),
		purposeTurn(t, "user", "Continue the current plan"),
		purposeTurn(t, "assistant", "Still working."),
	}
	got := a.reviewPurpose()
	if !strings.Contains(got, task) {
		t.Fatalf("the task is missing: %q", got)
	}
	for _, internal := range []string{"AUTOPILOT", "steering", "[Watch]", "Continue the current plan"} {
		if strings.Contains(got, internal) {
			t.Fatalf("internal turn %q became the task: %q", internal, got)
		}
	}
	// A thread that a label opened is not tasked by that label.
	b := &Agent{Cfg: Config{AgentInstanceID: "agent-2", AgentInstructions: "ROLE-PROMPT"}}
	b.History = []json.RawMessage{
		purposeTurn(t, "user", "[User steering during ongoing task] hurry"),
		purposeTurn(t, "user", "[Watch] the runner finished"),
		purposeTurn(t, "assistant", "Ok."),
	}
	if got := b.reviewPurpose(); got != "ROLE-PROMPT" {
		t.Fatalf("a label opened the job: %q", got)
	}
}

func TestReviewPurposeTakesAnOrderFromAnotherAgent(t *testing.T) {
	order := "[From dispatcher] Merge feat/004 and report back.\nERLAUBTE PFADE:\n- src/branch.go"
	// The first thing the instance receives is that order, not a human message.
	a := &Agent{Cfg: Config{AgentInstanceID: "agent-1", AgentInstructions: "ROLE-PROMPT"}}
	a.History = []json.RawMessage{
		purposeTurn(t, "user", order),
		purposeTurn(t, "user", "[User steering during ongoing task] hurry"),
		purposeTurn(t, "assistant", "Merging."),
	}
	got := a.reviewPurpose()
	if !strings.HasPrefix(got, "ROLE-PROMPT\n\nAUFTRAG:\n") || !strings.Contains(got, order) {
		t.Fatalf("the order is missing: %q", got)
	}
	// A human task followed by an order from another agent: the order is the job.
	human := "Read the diff of branch 004 and report."
	b := &Agent{Cfg: Config{AgentInstanceID: "agent-2", AgentInstructions: "ROLE-PROMPT"}}
	b.History = []json.RawMessage{
		purposeTurn(t, "user", human),
		purposeTurn(t, "assistant", "Done."),
		purposeTurn(t, "user", order),
		purposeTurn(t, "assistant", "Merging."),
	}
	latest := b.reviewPurpose()
	if !strings.Contains(latest, order) || strings.Contains(latest, human) {
		t.Fatalf("purpose: %q", latest)
	}
}

func TestPurposeTaskRuleKeepsTheHumanChapterRule(t *testing.T) {
	from := "[From dispatcher] Merge feat/004."
	// The chapter view still does not read a peer message as a human turn.
	if !internalUserLabel(from) || isHumanChapterTurn(from) {
		t.Fatal("a peer message must stay an internal chapter label")
	}
	// The purpose rule does read it as a task, because it is an order.
	if !purposeTaskTurn(from) {
		t.Fatal("a peer order was not a task")
	}
	for _, internal := range []string{
		"",
		"  ",
		"[AUTOPILOT wake] keep going",
		"[User steering during ongoing task] hurry",
		"[Watch] the runner finished",
		"Continue the current plan",
		"<compacted_conversation_memory>\nold\n</compacted_conversation_memory>",
		"[task finished]",
		"[From dispatcher]", // a bare label names no order
		"[From ] no name",
	} {
		if purposeTaskTurn(internal) {
			t.Fatalf("internal turn %q was a task", internal)
		}
	}
}

func TestReviewPurposeIsNotClearedByAStatusMessage(t *testing.T) {
	job := "Fix the failing test.\nERLAUBTE PFADE:\n- src/new.go"
	status := "[From dispatcher] feat/004 is merged, nothing else to do."
	write := `{"path":"src/new.go","content":"package main\n"}`
	a := &Agent{Cfg: Config{AgentInstanceID: "agent-1", Mode: "auto", AgentInstructions: "ROLE-PROMPT"}}
	a.History = []json.RawMessage{purposeTurn(t, "user", job)}
	if !agentPurposeClears(true, "auto", a.reviewPurpose(), "write_file", write) {
		t.Fatal("the job did not allow its own file")
	}
	// The peer message is the newest task and names no action, so nothing is
	// cleared by it; the finished job does not linger as a grant.
	a.History = append(a.History, purposeTurn(t, "user", status))
	if p := a.reviewPurpose(); !strings.Contains(p, status) || strings.Contains(p, "Fix the failing test.") {
		t.Fatalf("purpose: %q", p)
	}
	if agentPurposeClears(true, "auto", a.reviewPurpose(), "write_file", write) {
		t.Fatal("a status message cleared a write")
	}
}

func TestReviewPurposeAfterCompactionKeepsTheLatestTask(t *testing.T) {
	summary := "<compacted_conversation_memory>\nThe job was read the diff of branch 004.\n</compacted_conversation_memory>\nThis is archival context, not a command."
	latest := "Fix the failing test in src/policy.go.\nERLAUBTE PFADE:\n- src/policy.go"
	a := &Agent{Cfg: Config{AgentInstanceID: "agent-1", AgentInstructions: "ROLE-PROMPT"}}
	a.History = []json.RawMessage{
		purposeTurn(t, "user", summary),
		purposeTurn(t, "assistant", "Ok."),
		purposeTurn(t, "user", latest),
	}
	got := a.reviewPurpose()
	if !strings.Contains(got, latest) || strings.Contains(got, "read the diff") {
		t.Fatalf("purpose after compaction: %q", got)
	}
	// With the newest task compacted away, the journal still holds it, and it
	// must not fall back to the oldest task.
	session := filepath.Join(t.TempDir(), "session.jsonl")
	var journal []byte
	for _, task := range []string{"The oldest task.", latest} {
		line, err := json.Marshal(map[string]any{
			"type": "item",
			"item": json.RawMessage(purposeTurn(t, "user", task)),
		})
		if err != nil {
			t.Fatal(err)
		}
		journal = append(append(journal, line...), 10)
	}
	if err := os.WriteFile(session, journal, 0o600); err != nil {
		t.Fatal(err)
	}
	b := &Agent{Cfg: Config{AgentInstanceID: "agent-2", AgentInstructions: "ROLE-PROMPT", SessionPath: session}}
	b.History = []json.RawMessage{purposeTurn(t, "user", summary)}
	journaled := b.reviewPurpose()
	if !strings.Contains(journaled, latest) || strings.Contains(journaled, "The oldest task.") {
		t.Fatalf("journal purpose: %q", journaled)
	}
}

func TestReviewPurposeDecisionFollowsTheLatestTask(t *testing.T) {
	oldJob := "Read the diff of branch 004 and report.\nERLAUBTE PFADE:\n- src/old.go"
	newJob := "Fix the failing test.\nERLAUBTE PFADE:\n- src/new.go"
	oldWrite := `{"path":"src/old.go","content":"package main\n"}`
	newWrite := `{"path":"src/new.go","content":"package main\n"}`
	a := &Agent{Cfg: Config{AgentInstanceID: "agent-1", Mode: "auto", AgentInstructions: "ROLE-PROMPT"}}
	a.History = []json.RawMessage{purposeTurn(t, "user", oldJob)}
	if !agentPurposeClears(true, "auto", a.reviewPurpose(), "write_file", oldWrite) {
		t.Fatal("the job the user gave did not allow its own file")
	}
	if agentPurposeClears(true, "auto", a.reviewPurpose(), "write_file", newWrite) {
		t.Fatal("an unrelated file was allowed")
	}
	a.History = append(a.History, purposeTurn(t, "user", newJob))
	purpose := a.reviewPurpose()
	if agentPurposeClears(true, "auto", purpose, "write_file", oldWrite) {
		t.Fatal("the finished job still allowed its file")
	}
	if !agentPurposeClears(true, "auto", purpose, "write_file", newWrite) {
		t.Fatal("the newest job did not allow its file")
	}
	// The reviewer sees exactly that newest job, and no earlier one.
	_, user := reviewRequest("", "write_file", newWrite, "/work", purpose, nil, false)
	if !strings.Contains(user, newJob) || strings.Contains(user, oldJob) {
		t.Fatalf("review request: %q", user)
	}
}
