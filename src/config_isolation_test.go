package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestConfigPathsStayInTheTestArea proves that the test process resolves every
// config path inside the temporary directory TestMain created. The productive
// ~/.config/keimling is never touched here: no existence check, no listing, no
// mtime comparison. Containment of every derived path is the evidence.
func TestConfigPathsStayInTheTestArea(t *testing.T) {
	if testConfigHome == "" || testConfigDir == "" {
		t.Fatal("TestMain did not create an isolated config directory")
	}
	if got := configHome(); got != testConfigHome {
		t.Fatalf("config home %q, want the test area %q", got, testConfigHome)
	}
	if got := keimlingConfigDir(); got != testConfigDir {
		t.Fatalf("config dir %q, want the test area %q", got, testConfigDir)
	}
	dirs := map[string]string{
		"spendDir":           spendDir(),
		"spendConfigFile":    filepath.Dir(spendConfigFile),
		"userSkillsDir":      userSkillsDir(),
		"agentRecipesDir":    agentRecipesDir(),
		"agentTeamsDir":      agentTeamsDir(),
		"pluginsPath":        filepath.Dir(pluginsPath()),
		"defaultUserMCPPath": filepath.Dir(defaultUserMCPPath()),
		"mcpAllowPath":       filepath.Dir(mcpAllowPath()),
		"mcpBeforePath":      filepath.Dir(mcpBeforePath()),
		"guiPrefsPath":       filepath.Dir(guiPrefsPath("")),
		"vaultPath":          filepath.Dir(credentialsVaultPath("")),
		"connectionsPath":    filepath.Dir(clientConnectionsPath("")),
	}
	for name, dir := range dirs {
		if !insideDir(testConfigDir, dir) {
			t.Errorf("%s = %q is outside the test config dir %q", name, dir, testConfigDir)
		}
	}
	// On Unix the productive default is $HOME/.config/keimling. The test area
	// must not resolve to it.
	if runtime.GOOS != "windows" {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			if productive := filepath.Join(home, ".config", "keimling"); keimlingConfigDir() == productive {
				t.Fatalf("tests resolve the productive config dir %q", productive)
			}
		}
	}
}

// TestFixtureConfigIsSeededInTheTestArea shows that config reads land on the
// checked-in fixture inside the test area, not on a developer config.
func TestFixtureConfigIsSeededInTheTestArea(t *testing.T) {
	path := filepath.Join(keimlingConfigDir(), "config.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fixture config not seeded at %q: %v", path, err)
	}
	var fc fileConfig
	if err := json.Unmarshal(b, &fc); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if len(fc.ModelProviders) != 2 {
		t.Fatalf("model providers %d, want 2", len(fc.ModelProviders))
	}
	if fc.ActiveModelProviderID == "" {
		t.Fatal("active_model_provider_id is empty")
	}
	if fc.Reasoning == "" || fc.MaxSteps == 0 || fc.ContextLimit == 0 {
		t.Fatalf("reasoning %q, max_steps %d, context_limit %d", fc.Reasoning, fc.MaxSteps, fc.ContextLimit)
	}
	for _, secret := range []string{"api_key", "management_key", "key_file", "access_secret"} {
		if strings.Contains(string(b), secret) {
			t.Fatalf("fixture carries the secret field %q", secret)
		}
	}
}

// insideDir reports whether path lies in dir or below it.
func insideDir(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
