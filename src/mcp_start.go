package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// mcpScriptTimeout bounds a start script and a per-server script.
// The command often waits on a sign-in, so a short limit stops it too soon.
const mcpScriptTimeout = 5 * time.Minute

// mcpStartGate is process wide. Every workspace worker opens its own MCP
// manager, but the MCP servers and their start hook are shared by the whole
// process, so a successful start is reused instead of running again when
// another workspace is opened.
var mcpStartGate struct {
	mu  sync.Mutex
	cmd string
	ok  bool
}

func readMCPStart(configPath string) string {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return ""
	}
	fc, err := readConfiguration(configPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(fc.MCPStart)
}

func (m *MCPManager) beginScript(kind, server string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.scriptRunning = true
	m.scriptKind = kind
	m.scriptServer = server
	m.mu.Unlock()
}

func (m *MCPManager) endScript() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.scriptRunning = false
	m.mu.Unlock()
}

// ScriptBusy reports that a start or before script is inside its command.
func (m *MCPManager) ScriptBusy() bool {
	if m == nil {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.scriptRunning
}

// ScriptNotice is what the chat header shows while a script is unfinished
// or has failed. An empty state means the header stays quiet.
func (m *MCPManager) ScriptNotice() (state, message string) {
	if m == nil {
		return "", ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.scriptRunning {
		if m.scriptKind == "before" && m.scriptServer != "" {
			return "running", fmt.Sprintf("MCP script for %s is still running. That server stays disconnected until it finishes.", m.scriptServer)
		}
		return "running", "MCP start script is still running. Servers stay disconnected until it finishes."
	}
	var startErr string
	type beforeFail struct{ name, err string }
	var befores []beforeFail
	for name, s := range m.sessions {
		if s == nil || s.err == "" {
			continue
		}
		if strings.Contains(s.err, "start script failed") {
			if startErr == "" {
				startErr = s.err
			}
			continue
		}
		if strings.Contains(s.err, "before script failed") {
			befores = append(befores, beforeFail{name: name, err: s.err})
		}
	}
	if startErr != "" {
		return "failed", "MCP start script failed. No server is connected. " + startErr
	}
	if len(befores) == 0 {
		return "", ""
	}
	sort.Slice(befores, func(i, j int) bool { return befores[i].name < befores[j].name })
	parts := make([]string, 0, len(befores))
	for _, f := range befores {
		parts = append(parts, fmt.Sprintf("MCP script for %s failed. That server is not connected. %s", f.name, f.err))
	}
	return "failed", strings.Join(parts, " ")
}

func (m *MCPManager) setStartScript(command string) {
	if m == nil {
		return
	}
	command = strings.TrimSpace(command)
	m.startMu.Lock()
	m.startCmd = command
	m.startMu.Unlock()
}

// runStart runs the process start script. force runs it again after a success.
// An empty command is success. A failure is shared by every server in that pass.
// The cache and the running command are process wide, so opening another
// workspace does not repeat a hook that already succeeded.
func (m *MCPManager) runStart(ctx context.Context, force bool) error {
	if m == nil {
		return nil
	}
	m.startMu.Lock()
	defer m.startMu.Unlock()
	cmd := strings.TrimSpace(m.startCmd)
	if cmd == "" {
		return nil
	}
	mcpStartGate.mu.Lock()
	defer mcpStartGate.mu.Unlock()
	if !force && mcpStartGate.ok && mcpStartGate.cmd == cmd {
		return nil
	}
	m.beginScript("start", "")
	defer m.endScript()
	err := runShellCommand(ctx, cmd)
	mcpStartGate.cmd = cmd
	mcpStartGate.ok = err == nil
	if err != nil {
		return fmt.Errorf("start script failed: %w", err)
	}
	return nil
}

func runShellCommand(ctx context.Context, command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, mcpScriptTimeout)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", command)
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("timed out")
	}
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		return err
	}
	return fmt.Errorf("%s", truncate(msg, 300))
}

func mcpBeforePath() string {
	return filepath.Join(keimlingConfigDir(), "mcp-before.json")
}

func readMCPBeforeOverlay() map[string]string {
	b, err := os.ReadFile(mcpBeforePath())
	if err != nil {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

func writeMCPBeforeOverlay(server, command string) error {
	m := readMCPBeforeOverlay()
	if m == nil {
		m = map[string]string{}
	}
	server = strings.TrimSpace(server)
	command = strings.TrimSpace(command)
	if command == "" {
		delete(m, server)
	} else {
		m[server] = command
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := mcpBeforePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func applyMCPBefore(configs map[string]MCPServerConfig, sources map[string]string) {
	for name, command := range readMCPBeforeOverlay() {
		if sources[name] != "plugin" || strings.TrimSpace(command) == "" {
			continue
		}
		cfg, ok := configs[name]
		if !ok {
			continue
		}
		cfg.Before = strings.TrimSpace(command)
		configs[name] = cfg
	}
}

// setMCPBefore stores the command that runs before one server connects.
// A plugin server keeps it beside the plugin, not in the plugin's files.
func setMCPBefore(server, command, configPath string) error {
	server = strings.TrimSpace(server)
	command = strings.TrimSpace(command)
	if server == "" {
		return fmt.Errorf("server name required")
	}
	_, sources, _ := loadMCPServers(configPath)
	if sources[server] == "" {
		return fmt.Errorf("unknown MCP server")
	}
	if sources[server] == "plugin" {
		return writeMCPBeforeOverlay(server, command)
	}
	if sources[server] == "config" {
		return updateConfiguration(configPath, func(fc *fileConfig) {
			if fc.MCPServers == nil {
				return
			}
			cur := fc.MCPServers[server]
			cur.Before = command
			fc.MCPServers[server] = cur
		})
	}
	path := winningMCPFile(server, configPath)
	if path == "" {
		return fmt.Errorf("unknown MCP server")
	}
	return updateMCPServerInFile(path, server, func(c *MCPServerConfig) {
		c.Before = command
	})
}
