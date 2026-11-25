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

func TestAgentSpendLimitLastsForTheLifeOfTheInstance(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Capped", "Work.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Capped-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setAgentInstanceSpend(inst.ID, "0.05", ""); err != nil {
		t.Fatal(err)
	}
	child, err := createAgentInstance(recipe, ws.ID, "Child")
	if err != nil {
		t.Fatal(err)
	}
	usd := 0.06
	old := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
	recordCloudSpend(spendRecord{Time: old, Model: "m", InputTokens: 10, USD: &usd, Priced: true, AgentID: inst.ID})
	recordCloudSpend(spendRecord{Time: time.Now().UTC().Format(time.RFC3339), Model: "m", InputTokens: 10, USD: &usd, Priced: true, AgentID: child.ID})
	parent := &Agent{Cfg: Config{AgentInstanceID: inst.ID}}
	spentUSD, spentTokens := agentSpendLifetime(inst.ID)
	if err := agentSpendLimitError(parent); err == nil || !strings.Contains(err.Error(), "spend limit reached") {
		t.Fatalf("yesterday still counts: %v spent %v tokens %d", err, spentUSD, spentTokens)
	}
	if err := agentSpendLimitError(&Agent{Cfg: Config{AgentInstanceID: child.ID}}); err != nil {
		t.Fatalf("child spend is its own: %v", err)
	}
	if err := agentSpendLimitError(&Agent{Cfg: Config{}}); err != nil {
		t.Fatalf("a chat has no ceiling: %v", err)
	}
	if again := agentSpendLimitError(parent); again == nil {
		t.Fatal("still over the ceiling")
	}
	list := listEscalations(0)
	n := 0
	for _, e := range list {
		if e.AgentID == inst.ID && strings.HasPrefix(e.Text, "spend limit reached") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("escalations = %d", n)
	}
}

func TestSetAgentLimitFollowsTheEditGrant(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Runner", "Run.")
	if err != nil {
		t.Fatal(err)
	}
	tools := []string{"save_agent", "delete_agent"}
	recipe.Tools = &tools
	if err := writeAgentRecipe(recipe); err != nil {
		t.Fatal(err)
	}
	self, err := createAgentInstance(recipe, ws.ID, "Self")
	if err != nil {
		t.Fatal(err)
	}
	other, err := createAgentInstance(recipe, ws.ID, "Other")
	if err != nil {
		t.Fatal(err)
	}
	editor := &Agent{Cfg: Config{AgentInstanceID: self.ID, WorkspaceID: ws.ID}}
	if got := editor.setAgentLimitTool(`{"instance_id":"` + self.ID + `","spend_limit_usd":"1","spend_limit_tokens":"0"}`); !strings.Contains(got, "cannot change its own") {
		t.Fatalf("self: %s", got)
	}
	onlyDelete := []string{"delete_agent"}
	self.Tools = &onlyDelete
	if err := writeAgentInstanceMeta(self); err != nil {
		t.Fatal(err)
	}
	if got := editor.setAgentLimitTool(`{"instance_id":"` + other.ID + `","spend_limit_usd":"1","spend_limit_tokens":"0"}`); !strings.Contains(got, "not granted") {
		t.Fatalf("delete-only: %s", got)
	}
	self.Tools = nil
	if err := writeAgentInstanceMeta(self); err != nil {
		t.Fatal(err)
	}
	if got := editor.setAgentLimitTool(`{"instance_id":"` + other.ID + `","spend_limit_usd":"1.5","spend_limit_tokens":"100"}`); strings.Contains(got, "error") {
		t.Fatalf("edit: %s", got)
	}
	got, err := readAgentInstanceMeta(other.ID)
	if err != nil || got.SpendLimitUSD == nil || *got.SpendLimitUSD != 1.5 || got.SpendLimitTokens == nil || *got.SpendLimitTokens != 100 {
		t.Fatalf("stored %+v %v", got, err)
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
