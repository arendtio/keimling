package main

import (
	"os"
	"os/exec"
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

func TestFlattenedOpenSSHKeyLoads(t *testing.T) {
	if err := ensureSSHAgentAvailable(); err != nil {
		t.Skip(err)
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "id")
	out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", path, "-q").CombinedOutput()
	if err != nil {
		t.Fatalf("%v %s", err, out)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	flat := strings.ReplaceAll(string(raw), "\n", "")
	if strings.Contains(flat, "\n") || !strings.Contains(flat, "BEGIN OPENSSH PRIVATE KEY") {
		t.Fatal("fixture was not flattened")
	}
	sess, err := startSSHAgentWithKeys([]Credential{{ID: "k", Type: "ssh_key", Secret: flat}})
	if err != nil {
		t.Fatal(err)
	}
	sess.Close()
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
