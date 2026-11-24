package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAgentSpendLimitRefusesAfterTheCap(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Capped", "Work.")
	if err != nil {
		t.Fatal(err)
	}
	limit := 0.05
	recipe.SpendLimitUSD = &limit
	if err := writeAgentRecipe(recipe); err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Capped-1")
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{AgentInstanceID: inst.ID}}
	if err := agentSpendLimitError(a); err != nil {
		t.Fatalf("an unused agent is not refused: %v", err)
	}
	// One priced call under the cap, one above it.
	usd := 0.01
	recordCloudSpend(spendRecord{Time: time.Now().UTC().Format(time.RFC3339), Model: "m", InputTokens: 10, USD: &usd, Priced: true, AgentID: inst.ID})
	if err := agentSpendLimitError(a); err != nil {
		t.Fatalf("under the cap: %v", err)
	}
	usd = 0.06
	recordCloudSpend(spendRecord{Time: time.Now().UTC().Format(time.RFC3339), Model: "m", InputTokens: 10, USD: &usd, Priced: true, AgentID: inst.ID})
	if err := agentSpendLimitError(a); err == nil || !strings.Contains(err.Error(), "spend limit reached") {
		t.Fatalf("over the cap: %v", err)
	}
}

func TestAgentSpendTodayIgnoresOtherAgentsAndDays(t *testing.T) {
	withTempAgentHomes(t)
	now := time.Now().UTC()
	usd := 0.02
	recordCloudSpend(spendRecord{Time: now.Format(time.RFC3339), Model: "m", InputTokens: 1, USD: &usd, Priced: true, AgentID: "agent-a"})
	recordCloudSpend(spendRecord{Time: now.Format(time.RFC3339), Model: "m", InputTokens: 1, USD: &usd, Priced: true, AgentID: "agent-b"})
	old := now.Add(-48 * time.Hour).Format(time.RFC3339)
	recordCloudSpend(spendRecord{Time: old, Model: "m", InputTokens: 1, USD: &usd, Priced: true, AgentID: "agent-a"})
	if got := agentSpendToday("agent-a"); got < 0.019 || got > 0.021 {
		t.Fatalf("agent-a today = %v", got)
	}
	if got := agentSpendToday("agent-c"); got != 0 {
		t.Fatalf("agent-c = %v", got)
	}
	_ = os.Remove(filepath.Join(spendDir(), spendRecordPrefix+"-"+now.Format("2006-01")+".jsonl"))
}
