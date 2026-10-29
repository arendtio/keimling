package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanExecEnvPlaceholders(t *testing.T) {
	dir := t.TempDir()
	b := NewCredentialBroker(filepath.Join(dir, "config.json"))
	_ = b.Create("pw")
	_, err := b.Upsert(Credential{
		ID: "tok1", Name: "GH", Type: "token", AllowedTools: []string{"exec"}, InjectEnv: "GITHUB_TOKEN",
	}, "ghp_secret", true)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := b.planExecCredentials(`git push https://x {{keimling:tok1}}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.Command, "$GITHUB_TOKEN") || strings.Contains(plan.Command, "{{keimling:") {
		t.Fatalf("command=%q", plan.Command)
	}
	if plan.Env["GITHUB_TOKEN"] != "ghp_secret" || !plan.NeedsExfil {
		t.Fatalf("%#v", plan)
	}
}

func TestPlanExecSSHKey(t *testing.T) {
	dir := t.TempDir()
	b := NewCredentialBroker(filepath.Join(dir, "config.json"))
	_ = b.Create("pw")
	_, err := b.Upsert(Credential{
		ID: "k1", Name: "Deploy", Type: "ssh_key", AllowedTools: []string{"exec"},
	}, "-----BEGIN OPENSSH PRIVATE KEY-----\nfake\n-----END OPENSSH PRIVATE KEY-----", true)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := b.planExecCredentials(`git fetch {{keimling:k1}}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.SSHKeys) != 1 || plan.NeedsExfil {
		t.Fatalf("%#v", plan)
	}
	if strings.Contains(plan.Command, "{{keimling:") {
		t.Fatalf("placeholder left: %q", plan.Command)
	}
}

func TestValidateInjectEnv(t *testing.T) {
	if err := validateInjectEnv("PATH"); err == nil {
		t.Fatal("PATH blocked")
	}
	if err := validateInjectEnv("GITHUB_TOKEN"); err != nil {
		t.Fatal(err)
	}
}

func TestParseSSHAgentEnv(t *testing.T) {
	sock, pid := parseSSHAgentEnv("SSH_AUTH_SOCK=/tmp/ssh-abc/agent.1; export SSH_AUTH_SOCK;\nSSH_AGENT_PID=1234; export SSH_AGENT_PID;\necho Agent pid 1234;\n")
	if sock != "/tmp/ssh-abc/agent.1" || pid != "1234" {
		t.Fatalf("%q %q", sock, pid)
	}
}
