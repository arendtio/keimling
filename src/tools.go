package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

func hashBytes(b []byte) string { h := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(h[:]) }
func fileLines(b []byte) [][]byte {
	if len(b) == 0 {
		return nil
	}
	parts := bytes.SplitAfter(b, []byte("\n"))
	if len(parts[len(parts)-1]) == 0 {
		parts = parts[:len(parts)-1]
	}
	return parts
}
func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + fmt.Sprintf("\n...[truncated %d bytes]", len(s)-n)
	}
	return s
}
func privatePath(path string) bool {
	p, err := filepath.Abs(path)
	if err != nil {
		return true
	}
	home, _ := os.UserHomeDir()
	for _, s := range []string{filepath.Join(home, ".credentials_openai"), filepath.Join(home, ".ssh"), filepath.Join(home, ".aws"), filepath.Join(home, ".config", "gcloud"), filepath.Join(home, ".local", "share", "keyrings"), "/proc/self/environ"} {
		if p == s || strings.HasPrefix(p, s+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}
func executeTool(ctx context.Context, name, args string) string {
	switch name {
	case "exec":
		var x struct {
			Command        string `json:"command"`
			TimeoutSeconds int    `json:"timeout_seconds"`
			Interactive    bool   `json:"interactive"`
		}
		if err := decodeArgs(args, &x); err != nil {
			return err.Error()
		}
		if strings.TrimSpace(x.Command) == "" {
			return "empty command"
		}
		if x.TimeoutSeconds < 1 {
			if x.Interactive && !nonInteractiveDiagnostic(x.Command) {
				x.TimeoutSeconds = 900
			} else {
				x.TimeoutSeconds = 20
			}
		}
		if x.TimeoutSeconds > 120 && !x.Interactive {
			x.TimeoutSeconds = 120
		}
		if x.Interactive && x.TimeoutSeconds > 3600 {
			x.TimeoutSeconds = 3600
		}
		runCtx, cancel := context.WithTimeout(ctx, time.Duration(x.TimeoutSeconds)*time.Second)
		defer cancel()
		shell := selectedShell()
		if x.Interactive && !nonInteractiveDiagnostic(x.Command) {
			return interactiveExec(runCtx, shell, x.Command, time.Duration(x.TimeoutSeconds)*time.Second)
		}
		cmd := exec.Command(shell, "-c", x.Command)
		// sudo prompts use /dev/tty during a separate credential refresh.
		if commandUsesSudo(x.Command) {
			if err := authorizeSudo(ctx); err != nil {
				return "sudo authorization failed: " + err.Error()
			}
		}
		cmd.Stdin = nil
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		// Combined output preserves the order of stdout/stderr chunks.
		var output limitedBuffer
		output.Limit = 36 << 10
		cmd.Stdout = &output
		cmd.Stderr = &output
		if err := cmd.Start(); err != nil {
			return fmt.Sprintf("start error: %v", err)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		var err error
		select {
		case err = <-done:
		case <-runCtx.Done():
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			err = <-done
			return fmt.Sprintf("exit_status=cancelled (%v); error=%v\n%s", runCtx.Err(), err, output.String())
		}
		exit := 0
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				exit = ee.ExitCode()
			} else {
				return fmt.Sprintf("execution error: %v", err)
			}
		}
		return fmt.Sprintf("exit_code=%d\n%s", exit, output.String())
	case "read_file":
		var x struct {
			Path      string `json:"path"`
			StartLine int    `json:"start_line"`
			EndLine   int    `json:"end_line"`
		}
		if err := decodeArgs(args, &x); err != nil {
			return err.Error()
		}
		if privatePath(x.Path) {
			return "access denied: protected path"
		}
		st, err := os.Stat(x.Path)
		if err != nil {
			return err.Error()
		}
		if st.Size() > 8<<20 {
			return "file too large (>8 MiB); use exec for targeted inspection"
		}
		data, err := os.ReadFile(x.Path)
		if err != nil {
			return err.Error()
		}
		if !utf8.Valid(data) {
			return "not a UTF-8 text file"
		}
		lines := fileLines(data)
		if x.StartLine == 0 && x.EndLine == 0 {
			x.StartLine = 1
			x.EndLine = len(lines)
		}
		if x.StartLine < 1 || x.EndLine < x.StartLine || x.EndLine > len(lines) {
			return "invalid line range"
		}
		raw := bytes.Join(lines[x.StartLine-1:x.EndLine], nil)
		if len(raw) > 64<<10 {
			return "selected range exceeds 64 KiB; select fewer lines"
		}
		return fmt.Sprintf("%s:%d-%d\nhash: %s\n%s", x.Path, x.StartLine, x.EndLine, hashBytes(raw), string(raw))
	case "replace_lines":
		var x struct {
			Path             string `json:"path"`
			StartLine        int    `json:"start_line"`
			EndLine          int    `json:"end_line"`
			ExpectedHash     string `json:"expected_hash"`
			ExpectedFileHash string `json:"expected_file_hash"`
			Content          string `json:"content"`
		}
		if err := decodeArgs(args, &x); err != nil {
			return err.Error()
		}
		if privatePath(x.Path) {
			return "access denied: protected path"
		}
		data, err := os.ReadFile(x.Path)
		if err != nil {
			return err.Error()
		}
		if len(data) > 8<<20 {
			return "file too large"
		}
		lines := fileLines(data)
		if x.StartLine < 1 || x.EndLine < x.StartLine || x.EndLine > len(lines) {
			return "invalid line range; read again"
		}
		if x.ExpectedFileHash != "" && x.ExpectedFileHash != hashBytes(data) {
			return "file hash mismatch; read again"
		}
		old := bytes.Join(lines[x.StartLine-1:x.EndLine], nil)
		if x.ExpectedHash == "" || x.ExpectedHash != hashBytes(old) {
			return "hash mismatch; read the current lines again"
		}
		start := len(bytes.Join(lines[:x.StartLine-1], nil))
		end := start + len(old)
		if len(x.Content) > 1<<20 {
			return "replacement too large"
		}
		replacement := []byte(x.Content)
		if end < len(data) && len(replacement) > 0 && replacement[len(replacement)-1] != '\n' {
			return "replacement must end with newline when more lines follow"
		}
		next := append(append(append([]byte{}, data[:start]...), replacement...), data[end:]...)
		if err := atomicWrite(x.Path, next); err != nil {
			return err.Error()
		}
		return "replaced successfully"
	case "apply_patch":
		var x struct {
			Patch string `json:"patch"`
		}
		if err := decodeArgs(args, &x); err != nil {
			return err.Error()
		}
		notes, err := applyPatchText(x.Patch)
		if err != nil {
			return "patch failed: " + err.Error()
		}
		if notes != "" {
			return "patch applied successfully\nnotes:\n" + notes
		}
		return "patch applied successfully"
	}
	return "unknown tool"
}

type limitedBuffer struct {
	Data  []byte
	Limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if len(b.Data) < b.Limit {
		remaining := b.Limit - len(b.Data)
		if len(p) > remaining {
			p = p[:remaining]
		}
		b.Data = append(b.Data, p...)
	}
	return n, nil
}
func (b *limitedBuffer) String() string { return string(b.Data) }
func atomicWrite(path string, data []byte) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	mode := os.FileMode(0644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
		if !st.Mode().IsRegular() {
			return fmt.Errorf("target is not regular file")
		}
	}
	tmp, err := os.CreateTemp(dir, "."+base+".keimling-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
func RecordVoice(ctx context.Context, seconds int) (string, error) {
	f, err := os.CreateTemp("", "keimling-*.pcm")
	if err != nil {
		return "", err
	}
	path := f.Name()
	_ = f.Close()
	defer os.Remove(path)
	cmd := exec.CommandContext(ctx, "arecord", "-q", "-f", "S16_LE", "-r", "12000", "-c", "1", "-d", fmt.Sprint(seconds), "-t", "raw", path)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("arecord: %w: %s", err, truncate(string(b), 500))
	}
	return pcmFileToWAV(path, 12000)
}

func selectedShell() string {
	if st, err := os.Stat("/bin/bash"); err == nil && !st.IsDir() {
		return "/bin/bash"
	}
	return "/bin/sh"
}

func commandUsesSudo(command string) bool {
	// Conservative: only a hint for authentication; not a security boundary.
	// Only a token in command position counts, so text inside quotes, heredocs or
	// arguments does not trigger a password prompt.
	for _, segment := range strings.FieldsFunc(stripQuoted(command), func(r rune) bool {
		return r == '\n' || r == ';' || r == '|' || r == '&'
	}) {
		fields := strings.Fields(segment)
		// Skip leading VAR=value assignments and harmless wrappers.
		for len(fields) > 0 && (strings.Contains(fields[0], "=") || fields[0] == "env" || fields[0] == "command" || fields[0] == "time" || fields[0] == "nohup") {
			fields = fields[1:]
		}
		if len(fields) > 0 {
			switch fields[0] {
			case "sudo", "/usr/bin/sudo", "/bin/sudo":
				return true
			}
		}
	}
	return false
}

// stripQuoted removes single- and double-quoted spans so that quoted text is
// never mistaken for a command. Unterminated quotes are dropped to the end.
func stripQuoted(s string) string {
	var b strings.Builder
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// toolNeedsTerminal checks decoded arguments, not JSON formatting.
// Even a non-interactive exec needs exclusive terminal ownership while sudo -v
// reads the password from /dev/tty.
func toolNeedsTerminal(name, args string) bool {
	if name != "exec" {
		return false
	}
	var x struct {
		Command     string `json:"command"`
		Interactive bool   `json:"interactive"`
	}
	if json.Unmarshal([]byte(args), &x) != nil {
		return false
	}
	return (x.Interactive && !nonInteractiveDiagnostic(x.Command)) || commandUsesSudo(x.Command)
}

func authorizeSudo(ctx context.Context) error {
	// No stdin injection to the model; password is entered into sudo via the controlling TTY.
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("no interactive terminal: %w", err)
	}
	defer tty.Close()
	check := exec.CommandContext(ctx, "sudo", "-n", "-v")
	if check.Run() == nil {
		return nil
	}
	fmt.Fprintln(tty, "[sudo] Terminal authentication required:")
	auth := exec.CommandContext(ctx, "sudo", "-v")
	auth.Stdin, auth.Stdout, auth.Stderr = tty, tty, tty
	return auth.Run()
}

// safeDiagnostic only overrides accidental interactive=true for trivial read-only commands.
// Keep this allowlist deliberately narrow: shell operators, substitutions and extra
// commands must never be silently reclassified as safe.
func safeDiagnostic(command string) bool {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "lscpu", "/usr/bin/lscpu", "uname", "/usr/bin/uname", "date", "/usr/bin/date", "whoami", "/usr/bin/whoami":
	default:
		return false
	}
	for _, arg := range fields[1:] {
		if !strings.HasPrefix(arg, "-") || strings.ContainsAny(arg, ";|&<>$`\\'\"(){}") {
			return false
		}
	}
	return true
}

// nonInteractiveDiagnostic only classifies the need for a PTY, never the safety
// of executing a command. Command approval remains independent. It handles
// common read-only pipelines even when the model incorrectly requests a PTY.
func nonInteractiveDiagnostic(command string) bool {
	// Conservative: avoid downgrading commands that may prompt for input.
	lower := strings.ToLower(command)
	for _, dangerous := range []string{"ssh ", "scp ", "sftp ", "sudo ", "su ", "passwd", "read ", "askpass", "-i ", "--interactive", "pacman -s ", "pacman -r ", "pacman -u "} {
		if strings.Contains(" "+lower+" ", " "+dangerous) {
			return false
		}
	}
	// Pipes and redirections are common in diagnostics. Only grant the
	// non-PTY route for known read-only commands; no arbitrary shell evaluation.
	segments := strings.FieldsFunc(command, func(r rune) bool { return r == '|' || r == ';' || r == '&' || r == '\n' })
	if len(segments) == 0 {
		return false
	}
	for _, seg := range segments {
		fields := strings.Fields(strings.TrimSpace(seg))
		if len(fields) == 0 {
			return false
		}
		cmd := strings.TrimPrefix(fields[0], "/usr/bin/")
		switch cmd {
		case "lscpu", "uname", "date", "whoami", "hostname", "head", "tail", "cat", "grep", "rg", "wc", "ls", "lspci", "lsusb", "free", "uptime", "findmnt", "lsblk", "stat", "id":
		case "pacman":
			if len(fields) < 2 || !(fields[1] == "-Qi" || fields[1] == "-Q" || fields[1] == "-Qs" || fields[1] == "-Qo" || fields[1] == "-Ql" || fields[1] == "-Qe" || fields[1] == "-Qdt") {
				return false
			}
		default:
			return false
		}
	}
	return true
}
