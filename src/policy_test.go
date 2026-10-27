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
		{"exec", `{"command":"systemctl status sshd","interactive":false}`, false}, // soft, not hard
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

func TestSoftSuspicion(t *testing.T) {
	hit, ok := softSuspicion("exec", `{"command":"systemctl status sshd"}`)
	if !ok || hit.ID != "host_service_ctl" {
		t.Fatalf("systemctl: %+v ok=%v", hit, ok)
	}
	hit, ok = softSuspicion("exec", `{"command":"grep foo /var/log/syslog"}`)
	if !ok || hit.ID != "system_paths" {
		t.Fatalf("system_paths: %+v ok=%v", hit, ok)
	}
	// /dev/null scrubbed → no soft hit from empty leftovers for ip-only command
	if _, ok := softSuspicion("exec", `{"command":"ip addr show 2>/dev/null"}`); ok {
		t.Fatal("ip with /dev/null should not soft-hit after scrub")
	}
	if _, ok := softSuspicion("exec", `{"command":"ls /dev/sda"}`); !ok {
		t.Fatal("expected soft or hard path — /dev/sda should soft-hit if not hard")
	}
	// Hard floor still owns secrets; soft may also match /etc/ but force wins first.
	if forced, _ := forceApproval("exec", `{"command":"cat /etc/shadow"}`); !forced {
		t.Fatal("shadow must stay hard")
	}
}
