package main

import (
	"strings"
	"testing"
)

func TestWorkInstructionsRequireEvidenceAndAdmitIgnorance(t *testing.T) {
	got := sharedWorkInstructions()
	for _, want := range []string{
		"Facts before wording",
		"name its source as path:line",
		"say plainly that you have no information on it",
		"label an inference as unverified",
		"[Filtered]",
		"Never send a message that only announces waiting",
		"three to seven user-visible milestones",
		"Update the plan only when a step changes status",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestCredentialStatusPromptStatesTheVaultState(t *testing.T) {
	dir := t.TempDir()
	if got := credentialStatusPrompt(nil); got != "" {
		t.Fatalf("no broker: %q", got)
	}
	b := NewCredentialBroker(dir + "/config.json")
	if got := credentialStatusPrompt(b); !strings.Contains(got, "none stored") {
		t.Fatalf("missing vault: %q", got)
	}
	if err := b.Create("pw"); err != nil {
		t.Fatal(err)
	}
	if got := credentialStatusPrompt(b); !strings.Contains(got, "no credential allowed for exec") {
		t.Fatalf("empty vault: %q", got)
	}
	if _, err := b.Upsert(Credential{ID: "cred-key", Name: "deploy key", Type: "ssh_key", Service: "github.com", AllowedTools: []string{"exec"}}, "PRIVATE", true); err != nil {
		t.Fatal(err)
	}
	got := credentialStatusPrompt(b)
	for _, want := range []string{"unlocked", "cred-key (ssh_key)", "{{keimling:cred-key}}", "ephemeral ssh-agent"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "PRIVATE") {
		t.Fatal("prompt leaked the secret")
	}
}

func TestLockedVaultPromptAsksOnceAndStopsProbing(t *testing.T) {
	dir := t.TempDir()
	b := NewCredentialBroker(dir + "/config.json")
	if err := b.Create("pw"); err != nil {
		t.Fatal(err)
	}
	locked := NewCredentialBroker(dir + "/config.json")
	if locked.Exists() != true || locked.Unlocked() {
		t.Fatalf("state: exists=%v unlocked=%v", locked.Exists(), locked.Unlocked())
	}
	got := credentialStatusPrompt(locked)
	if !strings.Contains(got, "locked") || !strings.Contains(got, "do not probe ~/.ssh") {
		t.Fatalf("locked prompt: %q", got)
	}
}

func TestStaySilentToolExistsOnlyInLiveCalls(t *testing.T) {
	a := &Agent{Cfg: Config{Web: true}}
	live := realtimeToolList(a)
	if !containsToolName(live, "stay_silent") {
		t.Fatal("live call lost stay_silent")
	}
	if containsToolName(definitions(true), "stay_silent") {
		t.Fatal("stay_silent leaked into the chat tools")
	}
	if n := countToolName(live, "stay_silent"); n != 1 {
		t.Fatalf("stay_silent appears %d times", n)
	}
}

func containsToolName(tools []map[string]any, name string) bool {
	return countToolName(tools, name) > 0
}

func countToolName(tools []map[string]any, name string) int {
	n := 0
	for _, tool := range tools {
		if got, _ := tool["name"].(string); got == name {
			n++
		}
	}
	return n
}
