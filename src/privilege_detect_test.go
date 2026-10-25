package main

import "testing"

func TestCommandUsesPrivilegesOnlyInCommandPosition(t *testing.T) {
	S := "sudo"
	yes := []string{S + " -n ls", S + " pacman -S x", "cd /tmp && " + S + " ls", "ls | " + S + " tee /tmp/x", "FOO=1 " + S + " ls", "/usr/bin/" + S + " ls", "env " + S + " ls"}
	for _, cmd := range yes {
		if !commandUsesSudo(cmd) {
			t.Errorf("expected detection: %q", cmd)
		}
	}
	no := []string{
		`grep -rn "` + S + `" .`,
		`echo '` + S + ` pacman -S x'`,
		`printf "%s" "` + S + `"`,
		`git commit -m "use ` + S + ` carefully"`,
		"python3 - <<'PY'\nprint('" + S + "')\nPY",
		`cat ` + S + `.txt`,
		`rg ` + S,
		"",
	}
	for _, cmd := range no {
		if commandUsesSudo(cmd) {
			t.Errorf("unexpected detection: %q", cmd)
		}
	}
}
