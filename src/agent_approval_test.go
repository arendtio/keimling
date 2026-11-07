package main

import (
	"encoding/json"
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
	if !edit.Ask || edit.Review {
		t.Fatalf("workspace edit: %+v", edit)
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
	denied := settleReview(true, "ask", "touches ssh", nil)
	if !denied.Deny || denied.Ask || denied.Reason != "touches ssh" {
		t.Fatalf("deny: %+v", denied)
	}
	down := settleReview(true, "", "", errReviewDown)
	if !down.Deny || !strings.Contains(down.Reason, "review unavailable") {
		t.Fatalf("down: %+v", down)
	}
	asked := settleReview(false, "ask", "touches ssh", nil)
	if !asked.Ask || asked.Deny {
		t.Fatalf("workspace ask: %+v", asked)
	}
	if g := settleReview(true, "approve", "", nil); !g.Allow {
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
