package main

import "testing"

func TestApprovalFloors(t *testing.T) {
	cases := []struct {
		name, args string
		forced     bool
	}{
		{"exec", `{"command":"ls -la","interactive":false}`, false},
		{"exec", `{"command":"ip -brief address show 2>/dev/null || ip address show 2>/dev/null","interactive":false}`, false},
		{"exec", `{"command":"cat /etc/os-release","interactive":false}`, false},
		{"exec", `{"command":"systemctl status sshd","interactive":false}`, false}, // not a hard-floor command
		{"exec", `{"command":"cat /etc/shadow","interactive":false}`, true},
		{"exec", `{"command":"sudo pacman -S foo","interactive":false}`, true},
		{"exec", `{"command":"git reset --hard HEAD~1","interactive":false}`, true},
		{"exec", `{"command":"scp file remote:/tmp","interactive":true}`, true},
		{"edit_file", `{"path":"local-new.txt"}`, true},
		{"write_file", `{"path":"local-new.txt"}`, false},
	}
	for _, c := range cases {
		got, _ := forceApproval(c.name, c.args)
		if got != c.forced {
			t.Errorf("%s %s: forced=%v want %v", c.name, c.args, got, c.forced)
		}
	}
}
