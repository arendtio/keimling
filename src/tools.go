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
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// commandGroups are process groups started by exec. A shell can exit and leave
// a background child in the same group; Stop kills those too.
var commandGroups struct {
	sync.Mutex
	pgids map[int]struct{}
}

func trackCommandGroup(pgid int) {
	if pgid <= 0 {
		return
	}
	commandGroups.Lock()
	defer commandGroups.Unlock()
	if commandGroups.pgids == nil {
		commandGroups.pgids = map[int]struct{}{}
	}
	commandGroups.pgids[pgid] = struct{}{}
}

func releaseCommandGroup(pgid int) {
	if pgid <= 0 {
		return
	}
	if processGroupAlive(pgid) {
		return
	}
	commandGroups.Lock()
	delete(commandGroups.pgids, pgid)
	commandGroups.Unlock()
}

func killTrackedCommands() {
	commandGroups.Lock()
	ids := make([]int, 0, len(commandGroups.pgids))
	for id := range commandGroups.pgids {
		ids = append(ids, id)
	}
	commandGroups.pgids = nil
	commandGroups.Unlock()
	for _, id := range ids {
		killProcessGroup(id)
	}
}

func hashBytes(b []byte) string { h := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(h[:]) }
func readFilePreview(path string, st os.FileInfo) string {
	const preview = 1024
	if st.Size() > 8<<20 {
		f, err := os.Open(path)
		if err != nil {
			return err.Error()
		}
		defer f.Close()
		buf := make([]byte, preview)
		n, err := f.Read(buf)
		if n == 0 && err != nil {
			return err.Error()
		}
		if !utf8.Valid(buf[:n]) {
			return binaryFileNote(path, st.Size(), buf[:n])
		}
		return fmt.Sprintf("path: %s\nbytes: %d\npreview:\n%s", path, st.Size(), buf[:n])
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err.Error()
	}
	if !utf8.Valid(data) {
		return binaryFileNote(path, st.Size(), data)
	}
	if shape, ok := structuredOutline(string(data)); ok {
		return fmt.Sprintf("path: %s\nbytes: %d\nlines: %d\nshape:\n%s", path, len(data), len(fileLines(data)), shape)
	}
	prev := data
	if len(prev) > preview {
		prev = prev[:preview]
		if i := bytes.LastIndexByte(prev, '\n'); i > 0 {
			prev = prev[:i+1]
		}
	}
	return fmt.Sprintf("path: %s\nbytes: %d\nlines: %d\npreview:\n%s", path, len(data), len(fileLines(data)), prev)
}

func binaryFileNote(path string, size int64, sample []byte) string {
	text := printableRuns(sample, 240)
	if text == "" {
		text = "(no readable text)"
	}
	return fmt.Sprintf("path: %s\nbytes: %d\nkind: binary\nsample:\n%s", path, size, text)
}

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
	return executeToolWithEnv(ctx, name, args, nil)
}

func executeToolWithEnv(ctx context.Context, name, args string, extraEnv []string) string {
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
			if x.Interactive {
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
		binDir := pluginExecDir()
		if x.Interactive {
			env := append([]string{}, extraEnv...)
			if binDir != "" {
				env = append(env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			}
			return interactiveExec(runCtx, shell, x.Command, time.Duration(x.TimeoutSeconds)*time.Second, env...)
		}
		cmd := shellCommand(shell, x.Command)
		if len(extraEnv) > 0 || binDir != "" {
			cmd.Env = commandEnv(extraEnv, binDir)
		}
		// sudo prompts use /dev/tty during a separate credential refresh.
		if commandUsesSudo(x.Command) {
			if err := authorizeSudo(ctx); err != nil {
				return "sudo authorization failed: " + err.Error()
			}
		}
		cmd.Stdin = nil
		setProcessGroup(cmd)
		// Combined output preserves the order of stdout/stderr chunks.
		var output limitedBuffer
		output.Limit = 36 << 10
		cmd.Stdout = &output
		cmd.Stderr = &output
		if err := cmd.Start(); err != nil {
			return fmt.Sprintf("start error: %v", err)
		}
		trackCommandGroup(cmd.Process.Pid)
		defer releaseCommandGroup(cmd.Process.Pid)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		var err error
		select {
		case err = <-done:
		case <-runCtx.Done():
			killProcessGroup(cmd.Process.Pid)
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
		if x.StartLine == 0 && x.EndLine == 0 && st.Size() > int64(dataModelLimit) {
			return readFilePreview(x.Path, st)
		}
		if st.Size() > 8<<20 {
			return "file too large (>8 MiB); use a line range"
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
	case "edit_file":
		var x struct {
			Path       string `json:"path"`
			OldString  string `json:"old_string"`
			NewString  string `json:"new_string"`
			ReplaceAll bool   `json:"replace_all"`
		}
		if err := decodeArgs(args, &x); err != nil {
			return err.Error()
		}
		msg, err := editFile(x.Path, x.OldString, x.NewString, x.ReplaceAll)
		if err != nil {
			return err.Error()
		}
		return msg
	case "write_file":
		var x struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := decodeArgs(args, &x); err != nil {
			return err.Error()
		}
		msg, err := writeFile(x.Path, x.Content)
		if err != nil {
			return err.Error()
		}
		return msg
	case "screen_click", "screen_scroll", "screen_key", "screen_type":
		return runScreenTool(name, args)
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
	cmd, err := captureCommandContext(ctx, path, fmt.Sprint(seconds))
	if err != nil {
		return "", err
	}
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("voice capture: %w: %s", err, truncate(string(b), 500))
	}
	return pcmFileToWAV(path, 12000)
}

func selectedShell() string {
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("pwsh"); err == nil {
			return p
		}
		if p, err := exec.LookPath("powershell"); err == nil {
			return p
		}
		return "powershell"
	}
	if st, err := os.Stat("/bin/bash"); err == nil && !st.IsDir() {
		return "/bin/bash"
	}
	return "/bin/sh"
}

func shellCommand(shell, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", command)
	}
	return exec.Command(shell, "-c", command)
}

func hostOSName() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}

func hostSystemPrompt() string {
	return "Operating system: " + hostOSName() + ". Shell for exec: " + selectedShell() + ". Write commands for this operating system and this shell. "
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
	return x.Interactive || commandUsesSudo(x.Command)
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
