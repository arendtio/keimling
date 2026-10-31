package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSkillStaysOutUntilEnabled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, "config.json")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	user := filepath.Join(home, ".config", "keimling", "skills")
	writeSkillFixture(t, user, "alpha-skill", "Alpha skill description for the prompt.", "alpha")
	a := &Agent{Cfg: Config{ConfigPath: cfgPath}}
	if !strings.Contains(a.skillsCatalogBlock(), "alpha-skill") {
		t.Fatal("existing skill was not kept")
	}
	writeSkillFixture(t, user, "beta-skill", "Beta skill description for the prompt.", "beta")
	a.refreshSkills()
	block := a.skillsCatalogBlock()
	if strings.Contains(block, "beta-skill") {
		t.Fatal("new skill was included while auto-enable is off")
	}
	if !strings.Contains(block, "alpha-skill") {
		t.Fatal(block)
	}
	if err := a.setSkillEnabled("alpha-skill", false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(a.skillsCatalogBlock(), "alpha-skill") {
		t.Fatal("turned-off skill still in the prompt")
	}
	if err := a.setSkillAutoEnable(true); err != nil {
		t.Fatal(err)
	}
	writeSkillFixture(t, user, "gamma-skill", "Gamma skill description for the prompt.", "gamma")
	a.refreshSkills()
	block = a.skillsCatalogBlock()
	if !strings.Contains(block, "gamma-skill") {
		t.Fatal("new skill was not included after auto-enable")
	}
	if strings.Contains(block, "alpha-skill") {
		t.Fatal("auto-enable turned a disabled skill back on")
	}
	if a.skillEnabled("alpha-skill") {
		t.Fatal("alpha still counts as enabled")
	}
}
