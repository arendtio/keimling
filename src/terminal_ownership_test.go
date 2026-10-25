package main

import "testing"

func TestToolNeedsTerminal(t *testing.T) {
	cases := []struct {
		name, args string
		want       bool
	}{
		{"exec", `{"command":"sudo pacman -Syu","interactive":false}`, true},
		{"exec", `{ "interactive" : true, "command" : "scp a b:" }`, true},
		{"exec", `{"command":"ls -la"}`, false},
		{"read_file", `{"path":"x"}`, false},
		{"exec", `{"command":"printf 'sudo'"}`, false},
	}
	for _, c := range cases {
		if got := toolNeedsTerminal(c.name, c.args); got != c.want {
			t.Errorf("%s %q: got %v, want %v", c.name, c.args, got, c.want)
		}
	}
}
