package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MCPServerConfig mirrors Cursor's mcpServers entry (stdio or remote URL).
type MCPServerConfig struct {
	Command  string            `json:"command,omitempty"`
	Args     []string          `json:"args,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	URL      string            `json:"url,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Type     string            `json:"type,omitempty"` // stdio | sse | streamableHttp | http
	Disabled bool              `json:"disabled,omitempty"`
}

type mcpFile struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

func (c MCPServerConfig) transportKind() string {
	t := strings.ToLower(strings.TrimSpace(c.Type))
	switch t {
	case "stdio", "sse", "streamablehttp", "http":
		if t == "http" {
			return "streamableHttp"
		}
		if t == "streamablehttp" {
			return "streamableHttp"
		}
		return t
	}
	if strings.TrimSpace(c.Command) != "" {
		return "stdio"
	}
	if strings.TrimSpace(c.URL) != "" {
		return "streamableHttp"
	}
	return ""
}

func (c MCPServerConfig) validate() error {
	kind := c.transportKind()
	switch kind {
	case "stdio":
		if strings.TrimSpace(c.Command) == "" {
			return fmt.Errorf("stdio server requires command")
		}
	case "sse", "streamableHttp":
		if strings.TrimSpace(c.URL) == "" {
			return fmt.Errorf("%s server requires url", kind)
		}
	default:
		return fmt.Errorf("need command (stdio) or url (http/sse)")
	}
	return nil
}

func defaultUserMCPPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "keimling", "mcp.json")
}

func mcpConfigPaths(configPath string) []string {
	var paths []string
	if p := defaultUserMCPPath(); p != "" {
		paths = append(paths, p)
	}
	if configPath = strings.TrimSpace(configPath); configPath != "" {
		paths = append(paths, filepath.Join(filepath.Dir(configPath), "mcp.json"))
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, ".keimling", "mcp.json"))
		paths = append(paths, filepath.Join(cwd, ".cursor", "mcp.json"))
	}
	return paths
}

func readMCPFile(path string) (map[string]MCPServerConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var f mcpFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return f.MCPServers, nil
}

func writeMCPFile(path string, servers map[string]MCPServerConfig) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("mcp path is empty")
	}
	if servers == nil {
		servers = map[string]MCPServerConfig{}
	}
	b, err := json.MarshalIndent(mcpFile{MCPServers: servers}, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadMCPServerConfigs merges mcpServers from config.json and mcp.json files.
// Later paths override earlier ones for the same server name.
func LoadMCPServerConfigs(configPath string) map[string]MCPServerConfig {
	out := map[string]MCPServerConfig{}
	merge := func(m map[string]MCPServerConfig) {
		for name, cfg := range m {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			out[name] = cfg
		}
	}
	if configPath = strings.TrimSpace(configPath); configPath != "" {
		if fc, err := readConfiguration(configPath); err == nil && len(fc.MCPServers) > 0 {
			merge(fc.MCPServers)
		}
	}
	for _, p := range mcpConfigPaths(configPath) {
		m, err := readMCPFile(p)
		if err != nil || len(m) == 0 {
			continue
		}
		merge(m)
	}
	return out
}

func UpsertUserMCPServer(name string, cfg MCPServerConfig) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("server name required")
	}
	if err := cfg.validate(); err != nil {
		return err
	}
	path := defaultUserMCPPath()
	servers, err := readMCPFile(path)
	if err != nil {
		return err
	}
	if servers == nil {
		servers = map[string]MCPServerConfig{}
	}
	servers[name] = cfg
	return writeMCPFile(path, servers)
}

func RemoveUserMCPServer(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("server name required")
	}
	path := defaultUserMCPPath()
	servers, err := readMCPFile(path)
	if err != nil {
		return err
	}
	if servers == nil {
		return fmt.Errorf("server %q not found in %s", name, path)
	}
	if _, ok := servers[name]; !ok {
		return fmt.Errorf("server %q not found in %s", name, path)
	}
	delete(servers, name)
	return writeMCPFile(path, servers)
}

func sanitizeMCPToken(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "x"
	}
	return out
}

func mcpExposedName(server, tool string) string {
	return "mcp__" + sanitizeMCPToken(server) + "__" + sanitizeMCPToken(tool)
}

func isMCPTool(name string) bool {
	return strings.HasPrefix(name, "mcp__")
}

func parseMCPToolName(exposed string) (server, tool string, ok bool) {
	if !isMCPTool(exposed) {
		return "", "", false
	}
	rest := strings.TrimPrefix(exposed, "mcp__")
	i := strings.Index(rest, "__")
	if i <= 0 || i+2 >= len(rest) {
		return "", "", false
	}
	return rest[:i], rest[i+2:], true
}
