package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type sshAgentSession struct {
	sock string
	pid  string
	dir  string
}

func startSSHAgentWithKeys(keys []Credential) (*sshAgentSession, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	out, err := exec.Command("ssh-agent", "-s").Output()
	if err != nil {
		return nil, fmt.Errorf("ssh-agent: %w", err)
	}
	sock, pid := parseSSHAgentEnv(string(out))
	if sock == "" || pid == "" {
		return nil, fmt.Errorf("ssh-agent: could not parse env")
	}
	dir, err := os.MkdirTemp("", "keimling-ssh-*")
	if err != nil {
		_ = killSSHAgent(pid)
		return nil, err
	}
	sess := &sshAgentSession{sock: sock, pid: pid, dir: dir}
	for i, k := range keys {
		keyPath := filepath.Join(dir, fmt.Sprintf("key-%d", i))
		if err := os.WriteFile(keyPath, []byte(normalizeSSHKey(k.Secret)), 0o600); err != nil {
			sess.Close()
			return nil, err
		}
		cmd := exec.Command("ssh-add", keyPath)
		cmd.Env = append(os.Environ(), "SSH_AUTH_SOCK="+sock, "SSH_AGENT_PID="+pid)
		if out, err := cmd.CombinedOutput(); err != nil {
			sess.Close()
			return nil, fmt.Errorf("ssh-add %s: %v (%s)", k.ID, err, strings.TrimSpace(string(out)))
		}
		_ = os.Remove(keyPath) // best-effort; agent holds key
	}
	return sess, nil
}

func (s *sshAgentSession) Close() {
	if s == nil {
		return
	}
	if s.dir != "" {
		_ = os.RemoveAll(s.dir)
	}
	if s.pid != "" {
		_ = killSSHAgent(s.pid)
	}
}

func (s *sshAgentSession) Environ() []string {
	if s == nil {
		return nil
	}
	return []string{"SSH_AUTH_SOCK=" + s.sock, "SSH_AGENT_PID=" + s.pid}
}

func parseSSHAgentEnv(out string) (sock, pid string) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SSH_AUTH_SOCK=") {
			rest := strings.TrimPrefix(line, "SSH_AUTH_SOCK=")
			if i := strings.IndexByte(rest, ';'); i >= 0 {
				rest = rest[:i]
			}
			sock = strings.TrimSpace(rest)
		}
		if strings.HasPrefix(line, "SSH_AGENT_PID=") {
			rest := strings.TrimPrefix(line, "SSH_AGENT_PID=")
			if i := strings.IndexByte(rest, ';'); i >= 0 {
				rest = rest[:i]
			}
			pid = strings.TrimSpace(rest)
		}
	}
	return sock, pid
}

func killSSHAgent(pid string) error {
	return exec.Command("kill", pid).Run()
}

func normalizeSSHKey(secret string) string {
	s := strings.ReplaceAll(secret, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.TrimSpace(s)
	if fixed, ok := reflowPEM(s); ok {
		s = fixed
	}
	if s == "" {
		return "\n"
	}
	return s + "\n"
}

// reflowPEM puts a private key back into PEM lines. A one-line paste drops
// the breaks an HTML text field removes, and ssh-add then fails in libcrypto.
func reflowPEM(s string) (string, bool) {
	begin := strings.Index(s, "-----BEGIN ")
	if begin < 0 {
		return "", false
	}
	headerClose := strings.Index(s[begin+5:], "-----")
	if headerClose < 0 {
		return "", false
	}
	headerEnd := begin + 5 + headerClose + 5
	header := s[begin:headerEnd]
	end := strings.Index(s[headerEnd:], "-----END ")
	if end < 0 {
		return "", false
	}
	end += headerEnd
	footerClose := strings.Index(s[end+5:], "-----")
	if footerClose < 0 {
		return "", false
	}
	footerEnd := end + 5 + footerClose + 5
	footer := s[end:footerEnd]
	body := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\t', '\r':
			return -1
		default:
			return r
		}
	}, s[headerEnd:end])
	if body == "" {
		return "", false
	}
	var b strings.Builder
	b.WriteString(header)
	b.WriteByte('\n')
	for len(body) > 70 {
		b.WriteString(body[:70])
		b.WriteByte('\n')
		body = body[70:]
	}
	b.WriteString(body)
	b.WriteByte('\n')
	b.WriteString(footer)
	return b.String(), true
}

// ensureSSHAgentAvailable is used by tests to skip when ssh-agent missing.
func ensureSSHAgentAvailable() error {
	ctxDone := time.After(2 * time.Second)
	done := make(chan error, 1)
	go func() {
		_, err := exec.LookPath("ssh-agent")
		if err != nil {
			done <- err
			return
		}
		_, err = exec.LookPath("ssh-add")
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-ctxDone:
		return fmt.Errorf("timeout looking up ssh-agent")
	}
}
