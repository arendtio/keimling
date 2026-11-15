package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writePluginFixture(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "skills", "review-diff"), 0o700); err != nil {
		t.Fatal(err)
	}
	skill := "---\nname: review-diff\ndescription: Review a diff.\n---\nLook carefully.\n"
	if err := os.WriteFile(filepath.Join(root, "skills", "review-diff", "SKILL.md"), []byte(skill), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json",
  "name": "review-kit"
}`
	if err := os.WriteFile(filepath.Join(root, "plugin.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	mcp := `{
  "$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json",
  "mcpServers": {
    "review": {
      "type": "stdio",
      "command": "./bin/review",
      "args": ["--data", "${PLUGIN_DATA}/state"],
      "env": {"CONFIG": "${PLUGIN_ROOT}/config.json"},
      "cwd": "${PLUGIN_ROOT}"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(root, "mcp.json"), []byte(mcp), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPluginDirectoryContributesSkillAndServer(t *testing.T) {
	withTempAgentHomes(t)
	root := t.TempDir()
	writePluginFixture(t, root)
	st, err := addPluginSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if st.Name != "review-kit" || len(st.Skills) != 1 || st.Skills[0] != "review-diff" || len(st.Servers) != 1 || st.Servers[0] != "review" {
		t.Fatalf("status: %+v", st)
	}
	reg := LoadSkillRegistry(t.TempDir())
	if reg.Get("review-diff") == nil {
		t.Fatal("skill missing")
	}
	cfg := LoadMCPServerConfigs("")["review"]
	if cfg.Type != "stdio" || !strings.HasPrefix(cfg.Command, root) || !strings.Contains(cfg.Command, "bin/review") {
		t.Fatalf("command: %+v", cfg)
	}
	if cfg.Cwd != root || cfg.Env["PLUGIN_ROOT"] != root || !strings.Contains(cfg.Args[1], "state") || !strings.Contains(cfg.Env["CONFIG"], "config.json") {
		t.Fatalf("expanded: %+v", cfg)
	}
	if _, err := setPluginEnabled(st.ID, false); err != nil {
		t.Fatal(err)
	}
	if LoadSkillRegistry(t.TempDir()).Get("review-diff") != nil {
		t.Fatal("disabled plugin still contributed the skill")
	}
	if _, ok := LoadMCPServerConfigs("")["review"]; ok {
		t.Fatal("disabled plugin still contributed the server")
	}
}

func TestPluginDoesNotReplaceAnExistingServerOrSkill(t *testing.T) {
	withTempAgentHomes(t)
	root := t.TempDir()
	writePluginFixture(t, root)
	user := filepath.Join(userSkillsDir(), "review-diff")
	if err := os.MkdirAll(user, 0o700); err != nil {
		t.Fatal(err)
	}
	text := "---\nname: review-diff\ndescription: The local one.\n---\nLocal.\n"
	if err := os.WriteFile(filepath.Join(user, "SKILL.md"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpsertUserMCPServer("review", MCPServerConfig{Command: "local-review"}); err != nil {
		t.Fatal(err)
	}
	if _, err := addPluginSource(root); err != nil {
		t.Fatal(err)
	}
	got := LoadSkillRegistry(t.TempDir()).Get("review-diff")
	if got == nil || got.Source != skillSourceUser || !strings.Contains(got.Body, "Local") {
		t.Fatalf("skill: %+v", got)
	}
	if cmd := LoadMCPServerConfigs("")["review"].Command; cmd != "local-review" {
		t.Fatalf("server command %q", cmd)
	}
}

func TestPluginLocationKindsAndRemoval(t *testing.T) {
	withTempAgentHomes(t)
	kind, loc, err := classifyPluginLocation("https://example.com/team/plugin.git")
	if err != nil || kind != "git" || loc != "https://example.com/team/plugin.git" {
		t.Fatalf("git: %s %s %v", kind, loc, err)
	}
	if _, _, err := classifyPluginLocation("relative/plugin"); err == nil {
		t.Fatal("relative path accepted")
	}
	root := t.TempDir()
	writePluginFixture(t, root)
	st, err := addPluginSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := removePlugin(st.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal("directory source was deleted")
	}
	list, err := listPluginStatus()
	if err != nil || len(list) != 0 {
		t.Fatalf("list: %+v %v", list, err)
	}
}

func TestPluginGitURLChecksOut(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git")
	}
	withTempAgentHomes(t)
	src := t.TempDir()
	writePluginFixture(t, src)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = src
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Keimling", "GIT_AUTHOR_EMAIL=keimling@example.com", "GIT_COMMITTER_NAME=Keimling", "GIT_COMMITTER_EMAIL=keimling@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %s", strings.Join(args, " "), out)
		}
	}
	run("init")
	run("add", "plugin.json", "mcp.json", "skills")
	run("commit", "-m", "plugin")
	st, err := addPluginSource(src)
	if err != nil || st.Kind != "dir" {
		t.Fatalf("path is a directory source: %+v %v", st, err)
	}
	if err := removePlugin(st.ID); err != nil {
		t.Fatal(err)
	}
	rec := pluginRecord{ID: "plugin-git", Kind: "git", Location: src, Enabled: true}
	if err := syncPluginGit(rec, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(pluginCheckoutDir(rec.ID), "plugin.json")); err != nil {
		t.Fatal(err)
	}
}

func TestPluginGitFailureSkipsTheCloneBanner(t *testing.T) {
	out := "Cloning into '/data/plugins/checkouts/plugin-1'...\ngit@git.arendt.dev: Permission denied (publickey).\nfatal: Could not read from remote repository.\n"
	got := pluginGitFailure(out, nil)
	if strings.Contains(got, "Cloning into") || !strings.Contains(got, "Permission denied") {
		t.Fatal(got)
	}
}

func TestPluginGitUsesTheVaultKeyForThatHost(t *testing.T) {
	if err := ensureSSHAgentAvailable(); err != nil {
		t.Skip(err)
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id")
	out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath, "-q").CombinedOutput()
	if err != nil {
		t.Fatalf("%v %s", err, out)
	}
	raw, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "config.json")
	b := NewCredentialBroker(cfg)
	if err := b.Create("pw"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Upsert(Credential{
		Name: "Git", Type: "ssh_key", Service: "git.arendt.dev, github.com", AllowedTools: []string{"exec"},
	}, string(raw), true); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "clone")
	cleanup, err := applyPluginGitAuth(cmd, b, "git@git.arendt.dev:skills.git")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if cmd.Env == nil {
		t.Fatal("vault key was not offered")
	}
	list := exec.Command("ssh-add", "-l")
	list.Env = cmd.Env
	listed, err := list.CombinedOutput()
	if err != nil || !strings.Contains(string(listed), "ED25519") {
		t.Fatalf("%v %s", err, listed)
	}
	other := exec.Command("git", "clone")
	cleanupOther, err := applyPluginGitAuth(other, b, "git@example.test:repo.git")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupOther()
	if other.Env != nil {
		t.Fatal("a key for another host was offered")
	}
	web := exec.Command("git", "clone")
	cleanupWeb, err := applyPluginGitAuth(web, b, "https://git.arendt.dev/skills.git")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupWeb()
	if web.Env != nil {
		t.Fatal("an ssh key was offered for https")
	}
	locked := NewCredentialBroker(cfg)
	plain := exec.Command("git", "clone")
	cleanupPlain, err := applyPluginGitAuth(plain, locked, "git@git.arendt.dev:skills.git")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupPlain()
	if plain.Env != nil {
		t.Fatal("a locked vault changed the clone")
	}
}

func TestPluginSkipsAServerOutsideTheRoot(t *testing.T) {
	withTempAgentHomes(t)
	root := t.TempDir()
	writePluginFixture(t, root)
	mcp := `{
  "$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json",
  "mcpServers": {
    "escape": {"type": "stdio", "command": "./../outside"}
  }
}`
	if err := os.WriteFile(filepath.Join(root, "mcp.json"), []byte(mcp), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := addPluginSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Servers) != 0 || len(st.Notes) == 0 {
		t.Fatalf("status: %+v", st)
	}
	if _, ok := LoadMCPServerConfigs("")["escape"]; ok {
		t.Fatal("escaping server was loaded")
	}
}
