package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	OAuth    *MCPOAuth         `json:"oauth,omitempty"`
	// AllowTools are exposed MCP tool names that run without an approval question.
	AllowTools []string `json:"allow_tools,omitempty"`
	// Cwd is the stdio working directory. Empty keeps the process working directory.
	Cwd string `json:"cwd,omitempty"`
}

// MCPOAuth is the browser sign-in result for one remote MCP server.
// Tokens stay in mcp.json. They are not returned to the UI.
type MCPOAuth struct {
	ClientID     string    `json:"client_id,omitempty"`
	AuthURL      string    `json:"auth_url,omitempty"`
	TokenURL     string    `json:"token_url,omitempty"`
	Resource     string    `json:"resource,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

type mcpFile struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// remoteTypeSet is true when the entry names SSE or streamable HTTP.
// An empty type with a URL is probed.
func (c MCPServerConfig) remoteTypeSet() bool {
	switch strings.ToLower(strings.TrimSpace(c.Type)) {
	case "sse", "http", "streamablehttp":
		return true
	default:
		return false
	}
}

func mcpAuthorizationHeader(token string) string {
	token = strings.TrimSpace(token)
	if len(token) >= 7 && strings.EqualFold(token[:7], "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	if token == "" {
		return ""
	}
	return "Bearer " + token
}

func (c *MCPServerConfig) setBearer(token string) {
	value := mcpAuthorizationHeader(token)
	if value == "" {
		return
	}
	if c.Headers == nil {
		c.Headers = map[string]string{}
	}
	c.Headers["Authorization"] = value
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
	if _, err := os.UserHomeDir(); err != nil {
		return ""
	}
	return filepath.Join(keimlingConfigDir(), "mcp.json")
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
	configs, _, _ := loadMCPServers(configPath)
	return configs
}

// loadMCPServers returns the merged configs plus where the winning entry came from.
// sources are user, config, project, cursor, or plugin. names is set for plugins.
func loadMCPServers(configPath string) (map[string]MCPServerConfig, map[string]string, map[string]string) {
	configs := map[string]MCPServerConfig{}
	sources := map[string]string{}
	names := map[string]string{}
	put := func(name, source, sourceName string, cfg MCPServerConfig) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		configs[name] = cfg
		sources[name] = source
		if sourceName == "" {
			delete(names, name)
			return
		}
		names[name] = sourceName
	}
	configPath = strings.TrimSpace(configPath)
	if configPath != "" {
		if fc, err := readConfiguration(configPath); err == nil {
			for name, cfg := range fc.MCPServers {
				put(name, "config", "", cfg)
			}
		}
	}
	for _, p := range mcpConfigPaths(configPath) {
		m, err := readMCPFile(p)
		if err != nil || len(m) == 0 {
			continue
		}
		source := mcpFileSource(p, configPath)
		for name, cfg := range m {
			put(name, source, "", cfg)
		}
	}
	plugins, labels := pluginMCPOverlay()
	for name, cfg := range plugins {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, exists := configs[name]; exists {
			continue
		}
		put(name, "plugin", labels[name], cfg)
	}
	applyPluginAllow(configs, sources)
	return configs, sources, names
}

func mcpAllowPath() string {
	return filepath.Join(keimlingConfigDir(), "mcp-allow.json")
}

func readMCPAllowOverlay() map[string][]string {
	b, err := os.ReadFile(mcpAllowPath())
	if err != nil {
		return nil
	}
	var m map[string][]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

func writeMCPAllowOverlay(server string, tools []string) error {
	m := readMCPAllowOverlay()
	if m == nil {
		m = map[string][]string{}
	}
	if len(tools) == 0 {
		delete(m, server)
	} else {
		m[server] = append([]string(nil), tools...)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := mcpAllowPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func applyPluginAllow(configs map[string]MCPServerConfig, sources map[string]string) {
	overlay := readMCPAllowOverlay()
	for name, tools := range overlay {
		if sources[name] != "plugin" || len(tools) == 0 {
			continue
		}
		cfg, ok := configs[name]
		if !ok {
			continue
		}
		cfg.AllowTools = append([]string(nil), tools...)
		configs[name] = cfg
	}
}

func mcpFileSource(path, configPath string) string {
	path = filepath.Clean(path)
	if user := defaultUserMCPPath(); user != "" && path == filepath.Clean(user) {
		return "user"
	}
	if strings.HasSuffix(path, filepath.Join(".keimling", "mcp.json")) {
		return "project"
	}
	if strings.HasSuffix(path, filepath.Join(".cursor", "mcp.json")) {
		return "cursor"
	}
	if configPath != "" && path == filepath.Clean(filepath.Join(filepath.Dir(configPath), "mcp.json")) {
		return "user"
	}
	return "user"
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
	if prev, ok := servers[name]; ok && cfg.AllowTools == nil {
		cfg.AllowTools = prev.AllowTools
	}
	servers[name] = cfg
	return writeMCPFile(path, servers)
}

// setMCPToolAllow marks or clears one exposed tool name on the file that actually supplies the server.
func setMCPToolAllow(server, tool string, allow bool, configPath string) ([]string, error) {
	server = strings.TrimSpace(server)
	tool = strings.TrimSpace(tool)
	if server == "" || tool == "" {
		return nil, fmt.Errorf("server and tool required")
	}
	configs, sources, _ := loadMCPServers(configPath)
	cfg, ok := configs[server]
	if !ok {
		return nil, fmt.Errorf("unknown MCP server")
	}
	next := make([]string, 0, len(cfg.AllowTools)+1)
	seen := false
	for _, name := range cfg.AllowTools {
		if name == tool {
			seen = true
			if allow {
				next = append(next, name)
			}
			continue
		}
		next = append(next, name)
	}
	if allow && !seen {
		next = append(next, tool)
	}
	if sources[server] == "plugin" {
		if err := writeMCPAllowOverlay(server, next); err != nil {
			return nil, err
		}
		return next, nil
	}
	if sources[server] == "config" {
		err := updateConfiguration(configPath, func(fc *fileConfig) {
			if fc.MCPServers == nil {
				fc.MCPServers = map[string]MCPServerConfig{}
			}
			cur := fc.MCPServers[server]
			cur.AllowTools = next
			fc.MCPServers[server] = cur
		})
		if err != nil {
			return nil, err
		}
		return next, nil
	}
	path := winningMCPFile(server, configPath)
	if path == "" {
		return nil, fmt.Errorf("unknown MCP server")
	}
	if err := updateMCPServerInFile(path, server, func(c *MCPServerConfig) {
		c.AllowTools = next
	}); err != nil {
		return nil, err
	}
	return next, nil
}

func winningMCPFile(name, configPath string) string {
	var found string
	for _, p := range mcpConfigPaths(configPath) {
		servers, err := readMCPFile(p)
		if err != nil || len(servers) == 0 {
			continue
		}
		if _, ok := servers[name]; ok {
			found = p
		}
	}
	return found
}

// mcpToolAlwaysAllowed reports whether this exposed tool is marked on any loaded server.
func mcpToolAlwaysAllowed(configPath, exposed string) bool {
	if !isMCPTool(exposed) {
		return false
	}
	for _, cfg := range LoadMCPServerConfigs(configPath) {
		for _, name := range cfg.AllowTools {
			if name == exposed {
				return true
			}
		}
	}
	return false
}

func RemoveUserMCPServer(name string) error {
	return RemoveMCPServer(name, "")
}

// RemoveMCPServer deletes name from every config file that lists it.
// A server that exists only because a plugin is enabled is left alone.
func RemoveMCPServer(name, configPath string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("server name required")
	}
	removed := false
	if ok, err := removeMCPFromConfig(configPath, name); err != nil {
		return err
	} else if ok {
		removed = true
	}
	seen := map[string]struct{}{}
	for _, p := range mcpConfigPaths(configPath) {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		ok, err := deleteMCPServerKey(p, name)
		if err != nil {
			return err
		}
		if ok {
			removed = true
		}
	}
	if removed {
		return nil
	}
	_, labels := pluginMCPOverlay()
	if label := labels[name]; label != "" {
		return fmt.Errorf("server %q comes from plugin %s; remove or turn off that plugin", name, label)
	}
	return fmt.Errorf("server %q not found", name)
}

func removeMCPFromConfig(configPath, name string) (bool, error) {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return false, nil
	}
	fc, err := readConfiguration(configPath)
	if err != nil {
		return false, err
	}
	if _, ok := fc.MCPServers[name]; !ok {
		return false, nil
	}
	err = updateConfiguration(configPath, func(cfg *fileConfig) {
		if cfg.MCPServers == nil {
			return
		}
		delete(cfg.MCPServers, name)
		if len(cfg.MCPServers) == 0 {
			cfg.MCPServers = nil
		}
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func deleteMCPServerKey(path, name string) (bool, error) {
	doc, servers, err := readMCPDocument(path)
	if err != nil || doc == nil {
		return false, err
	}
	if _, ok := servers[name]; !ok {
		return false, nil
	}
	delete(servers, name)
	return true, writeMCPDocument(path, doc, servers)
}

func updateMCPServerInFile(path, name string, mut func(*MCPServerConfig)) error {
	doc, servers, err := readMCPDocument(path)
	if err != nil {
		return err
	}
	if doc == nil {
		return fmt.Errorf("server %q not found", name)
	}
	raw, ok := servers[name]
	if !ok {
		return fmt.Errorf("server %q not found", name)
	}
	var cfg MCPServerConfig
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	mut(&cfg)
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	servers[name] = encoded
	return writeMCPDocument(path, doc, servers)
}

func readMCPDocument(path string) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	if doc == nil {
		doc = map[string]json.RawMessage{}
	}
	servers := map[string]json.RawMessage{}
	if raw := doc["mcpServers"]; len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	if servers == nil {
		servers = map[string]json.RawMessage{}
	}
	return doc, servers, nil
}

func writeMCPDocument(path string, doc, servers map[string]json.RawMessage) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("mcp path is empty")
	}
	if doc == nil {
		doc = map[string]json.RawMessage{}
	}
	if servers == nil {
		servers = map[string]json.RawMessage{}
	}
	encoded, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	doc["mcpServers"] = encoded
	b, err := json.MarshalIndent(doc, "", "  ")
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
