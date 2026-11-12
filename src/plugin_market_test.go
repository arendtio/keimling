package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMarketplaceFixture(t *testing.T, root string) {
	t.Helper()
	mustMk := func(path string) {
		t.Helper()
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite := func(path, body string) {
		t.Helper()
		mustMk(filepath.Dir(path))
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mustMk(filepath.Join(root, ".cursor-plugin"))
	mustWrite(filepath.Join(root, ".cursor-plugin", "marketplace.json"), `{
  "name": "team-market",
  "owner": {"name": "Team"},
  "plugins": [
    {"name": "review-kit", "source": "plugin-one"},
    {"name": "notes-kit", "source": "./plugin-two"},
    {"name": "remote-kit", "source": {"source": "github", "repo": "org/remote"}}
  ]
}`)
	mustWrite(filepath.Join(root, "plugin.json"), `{
  "$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json",
  "name": "not-the-catalog"
}`)
	mustWrite(filepath.Join(root, "skills", "root-only", "SKILL.md"), "---\nname: root-only\ndescription: Should stay out.\n---\nNo.\n")
	one := filepath.Join(root, "plugin-one")
	mustWrite(filepath.Join(one, ".cursor-plugin", "plugin.json"), `{"name": "review-kit", "description": "Review tools"}`)
	mustWrite(filepath.Join(one, "skills", "review-diff", "SKILL.md"), "---\nname: review-diff\ndescription: Review a diff.\n---\nLook carefully.\n")
	mustWrite(filepath.Join(one, "rules", "style.mdc"), "---\ndescription: Keep tests small\nglobs: *.go\n---\nPrefer table tests.\n")
	mustWrite(filepath.Join(one, "commands", "deploy.md"), "---\nname: deploy\ndescription: Deploy the service\n---\nDeploy the service now.\n")
	mustWrite(filepath.Join(one, "agents", "reviewer.md"), "---\nname: reviewer\ndescription: Reviews changes\n---\nReview the diff carefully.\n")
	mustWrite(filepath.Join(one, "mcp.json"), `{"mcpServers": {"docs": {"command": "echo", "args": ["hi"]}}}`)
	mustWrite(filepath.Join(one, "hooks", "hooks.json"), `{
  "hooks": {"sessionStart": [{"name": "scan", "command": "sh ./scripts/scan.sh"}]}
}`)
	mustWrite(filepath.Join(one, "scripts", "scan.sh"), "printf 'first\\n' >> \"$PLUGIN_ROOT/ran.txt\"\n")
	mustWrite(filepath.Join(one, "bin", "keimling-plugin-hello"), "#!/bin/sh\necho hello-plugin\n")
	if err := os.Chmod(filepath.Join(one, "bin", "keimling-plugin-hello"), 0o700); err != nil {
		t.Fatal(err)
	}
	two := filepath.Join(root, "plugin-two")
	mustWrite(filepath.Join(two, ".claude-plugin", "plugin.json"), `{"name": "notes-kit"}`)
	mustWrite(filepath.Join(two, "skills", "note-taking", "SKILL.md"), "---\nname: note-taking\ndescription: Take notes.\n---\nWrite it down.\n")
}

func TestMarketplaceRecognizesTheCatalogAndLoadsEveryPlugin(t *testing.T) {
	withTempAgentHomes(t)
	root := t.TempDir()
	writeMarketplaceFixture(t, root)
	st, err := addPluginSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if st.Layout != "marketplace" || st.Name != "team-market" {
		t.Fatalf("status: %+v", st)
	}
	if len(st.Plugins) != 2 || st.Plugins[0] != "notes-kit" || st.Plugins[1] != "review-kit" {
		t.Fatalf("plugins: %+v notes %v", st.Plugins, st.Notes)
	}
	if len(st.Active) != 0 {
		t.Fatalf("plugins started on: %+v", st.Active)
	}
	if LoadSkillRegistry(t.TempDir()).Get("review-diff") != nil || LoadSkillRegistry(t.TempDir()).Get("note-taking") != nil {
		t.Fatal("marketplace plugin contributed before it was activated")
	}
	if _, err := setPluginUnit(st.ID, "review-kit", true); err != nil {
		t.Fatal(err)
	}
	if _, err := setPluginUnit(st.ID, "notes-kit", true); err != nil {
		t.Fatal(err)
	}
	reg := LoadSkillRegistry(t.TempDir())
	if reg.Get("review-diff") == nil || reg.Get("note-taking") == nil {
		t.Fatal("marketplace skills missing")
	}
	if reg.Get("root-only") != nil {
		t.Fatal("root plugin beside the catalog was loaded")
	}
	if !strings.Contains(pluginRulesText(), "Prefer table tests.") || !strings.Contains(pluginRulesText(), "style") {
		t.Fatalf("rules: %s", pluginRulesText())
	}
	cmds := listPluginCommands()
	if len(cmds) != 1 || cmds[0].Name != "deploy" || !strings.Contains(cmds[0].Body, "Deploy the service now.") {
		t.Fatalf("commands: %+v", cmds)
	}
	cfg := LoadMCPServerConfigs("")["docs"]
	if cfg.Type != "stdio" || cfg.Command != "echo" {
		t.Fatalf("mcp: %+v", cfg)
	}
	joined := strings.Join(st.Notes, " ")
	if !strings.Contains(joined, "remote-kit") || !strings.Contains(joined, "outside this repository") {
		t.Fatalf("notes: %v", st.Notes)
	}
	if len(st.Code) != 2 {
		t.Fatalf("code: %+v", st.Code)
	}
	for _, code := range st.Code {
		if code.Granted {
			t.Fatalf("granted before activation: %+v", code)
		}
	}
	if runPluginHooks("sessionStart", "") != "" {
		t.Fatal("hook ran before activation")
	}
	if _, err := os.Stat(filepath.Join(root, "plugin-one", "ran.txt")); !os.IsNotExist(err) {
		t.Fatal("hook wrote before activation")
	}
	if out := executeTool(t.Context(), "exec", `{"command":"keimling-plugin-hello"}`); strings.Contains(out, "hello-plugin") {
		t.Fatalf("exec ran before activation: %s", out)
	}

	st, err = grantPluginCode(st.ID, []string{"scan", "keimling-plugin-hello"})
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range st.Code {
		if !code.Granted {
			t.Fatalf("not granted: %+v", code)
		}
	}
	if note := runPluginHooks("sessionStart", ""); note != "" && !strings.Contains(note, "scan") {
		t.Fatalf("hook note: %s", note)
	}
	ran, err := os.ReadFile(filepath.Join(root, "plugin-one", "ran.txt"))
	if err != nil || !strings.Contains(string(ran), "first") {
		t.Fatalf("ran: %s %v", ran, err)
	}
	if out := executeTool(t.Context(), "exec", `{"command":"keimling-plugin-hello"}`); !strings.Contains(out, "hello-plugin") {
		t.Fatalf("exec after grant: %s", out)
	}

	if err := os.WriteFile(filepath.Join(root, "plugin-one", "scripts", "scan.sh"), []byte("printf 'second\\n' >> \"$PLUGIN_ROOT/ran.txt\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = runPluginHooks("sessionStart", "")
	ran, err = os.ReadFile(filepath.Join(root, "plugin-one", "ran.txt"))
	if err != nil || !strings.Contains(string(ran), "second") {
		t.Fatalf("updated hook did not keep the grant: %s %v", ran, err)
	}
	if err := os.WriteFile(filepath.Join(root, "plugin-one", "hooks", "hooks.json"), []byte(`{
  "hooks": {
    "sessionStart": [
      {"name": "scan", "command": "sh ./scripts/scan.sh"},
      {"name": "other", "command": "sh ./scripts/other.sh"}
    ]
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "plugin-one", "scripts", "other.sh"), []byte("printf 'other\\n' >> \"$PLUGIN_ROOT/ran.txt\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = runPluginHooks("sessionStart", "")
	ran, _ = os.ReadFile(filepath.Join(root, "plugin-one", "ran.txt"))
	if strings.Contains(string(ran), "other") {
		t.Fatalf("new hook name ran without a grant: %s", ran)
	}

	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	inst, err := startPluginAgent(st.ID, "reviewer", ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := readAgentRecipe(inst.RecipeID)
	if err != nil || !strings.Contains(recipe.Instructions, "Review the diff carefully.") {
		t.Fatalf("agent: %+v %v", recipe, err)
	}

	if _, err := setPluginEnabled(st.ID, false); err != nil {
		t.Fatal(err)
	}
	if LoadSkillRegistry(t.TempDir()).Get("review-diff") != nil || LoadSkillRegistry(t.TempDir()).Get("note-taking") != nil {
		t.Fatal("disabled marketplace still contributed skills")
	}
	if pluginRulesText() != "" || len(listPluginCommands()) != 0 {
		t.Fatal("disabled marketplace still contributed rules or commands")
	}
}

func TestCursorPluginDirectoryIsASinglePlugin(t *testing.T) {
	withTempAgentHomes(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".cursor-plugin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".cursor-plugin", "plugin.json"), []byte(`{"name":"solo-kit"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "skills", "solo"), 0o700); err != nil {
		t.Fatal(err)
	}
	skill := "---\nname: solo\ndescription: One skill.\n---\nAlone.\n"
	if err := os.WriteFile(filepath.Join(root, "skills", "solo", "SKILL.md"), []byte(skill), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := addPluginSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if st.Layout != "plugin" || st.Name != "solo-kit" {
		t.Fatalf("status: %+v", st)
	}
	if LoadSkillRegistry(t.TempDir()).Get("solo") == nil {
		t.Fatal("cursor plugin skill missing")
	}
}

func TestClaudeMarketplaceUsesARelativeSource(t *testing.T) {
	withTempAgentHomes(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude-plugin"), 0o700); err != nil {
		t.Fatal(err)
	}
	catalog := `{
  "name": "claude-market",
  "owner": {"name": "Team"},
  "plugins": [{"name": "docs-kit", "source": "./plugins/docs"}]
}`
	if err := os.WriteFile(filepath.Join(root, ".claude-plugin", "marketplace.json"), []byte(catalog), 0o600); err != nil {
		t.Fatal(err)
	}
	plug := filepath.Join(root, "plugins", "docs")
	if err := os.MkdirAll(filepath.Join(plug, ".claude-plugin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plug, ".claude-plugin", "plugin.json"), []byte(`{"name":"docs-kit"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plug, ".mcp.json"), []byte(`{"mcpServers":{"docs":{"command":"echo","args":["claude"]}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := addPluginSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if st.Layout != "marketplace" || len(st.Plugins) != 1 || st.Plugins[0] != "docs-kit" {
		t.Fatalf("status: %+v", st)
	}
	if _, ok := LoadMCPServerConfigs("")["docs"]; ok {
		t.Fatal("claude mcp loaded before the plugin was activated")
	}
	if _, err := setPluginUnit(st.ID, "docs-kit", true); err != nil {
		t.Fatal(err)
	}
	if LoadMCPServerConfigs("")["docs"].Command != "echo" {
		t.Fatal("claude mcp was not loaded")
	}
}

func TestEnabledMarketplaceKeepsPluginsUntilOneIsTurnedOff(t *testing.T) {
	withTempAgentHomes(t)
	root := t.TempDir()
	writeMarketplaceFixture(t, root)
	rec := pluginRecord{ID: "plugin-old", Kind: "dir", Location: root, Enabled: true}
	if err := writePluginFile(pluginFile{Sources: []pluginRecord{rec}}); err != nil {
		t.Fatal(err)
	}
	if LoadSkillRegistry(t.TempDir()).Get("review-diff") == nil || LoadSkillRegistry(t.TempDir()).Get("note-taking") == nil {
		t.Fatal("existing marketplace lost its plugins")
	}
	if _, err := setPluginUnit("plugin-old", "notes-kit", false); err != nil {
		t.Fatal(err)
	}
	reg := LoadSkillRegistry(t.TempDir())
	if reg.Get("note-taking") != nil {
		t.Fatal("turned-off plugin still contributed")
	}
	if reg.Get("review-diff") == nil {
		t.Fatal("the other plugin was turned off too")
	}
	if _, ok := LoadMCPServerConfigs("")["docs"]; !ok {
		t.Fatal("the plugin that stayed on lost its server")
	}
}
