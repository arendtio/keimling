package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	agentPluginsSchema    = "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json"
	agentPluginsMCPSchema = "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json"
)

var pluginNameRe = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+)*$`)

type pluginRecord struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"` // git | dir
	Location string   `json:"location"`
	Enabled  bool     `json:"enabled"`
	Grants   []string `json:"grants,omitempty"`
}

type pluginFile struct {
	Sources []pluginRecord `json:"sources"`
}

type pluginStatus struct {
	ID          string             `json:"id"`
	Kind        string             `json:"kind"`
	Layout      string             `json:"layout,omitempty"` // plugin | marketplace
	Location    string             `json:"location"`
	Enabled     bool               `json:"enabled"`
	Name        string             `json:"name,omitempty"`
	Description string             `json:"description,omitempty"`
	Version     string             `json:"version,omitempty"`
	Plugins     []string           `json:"plugins,omitempty"`
	Skills      []string           `json:"skills,omitempty"`
	Rules       []string           `json:"rules,omitempty"`
	Commands    []string           `json:"commands,omitempty"`
	Agents      []string           `json:"agents,omitempty"`
	Servers     []string           `json:"servers,omitempty"`
	Code        []pluginCodeStatus `json:"code,omitempty"`
	Error       string             `json:"error,omitempty"`
	Notes       []string           `json:"notes,omitempty"`
}

func pluginsPath() string {
	return filepath.Join(keimlingConfigDir(), "plugins.json")
}

func pluginCheckoutDir(id string) string {
	return filepath.Join(dataHome(), "keimling", "plugins", "checkouts", id)
}

func pluginDataDir(id string) string {
	return filepath.Join(dataHome(), "keimling", "plugins", "data", id)
}

func readPluginFile() (pluginFile, error) {
	b, err := os.ReadFile(pluginsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return pluginFile{}, nil
		}
		return pluginFile{}, err
	}
	var f pluginFile
	if err := json.Unmarshal(b, &f); err != nil {
		return pluginFile{}, err
	}
	return f, nil
}

func writePluginFile(f pluginFile) error {
	if f.Sources == nil {
		f.Sources = []pluginRecord{}
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := pluginsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func classifyPluginLocation(raw string) (kind, location string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("location required")
	}
	if strings.ContainsAny(raw, "\r\n") || strings.Contains(raw, " ") {
		return "", "", fmt.Errorf("invalid location")
	}
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "ssh://"):
		return "git", raw, nil
	case strings.HasPrefix(raw, "git@") && strings.Contains(raw, ":"):
		return "git", raw, nil
	}
	path := raw
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", "", herr
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, path[2:])
		}
	}
	if !filepath.IsAbs(path) {
		return "", "", fmt.Errorf("directory must be an absolute path")
	}
	return "dir", filepath.Clean(path), nil
}

func addPluginSource(location string) (pluginStatus, error) {
	kind, loc, err := classifyPluginLocation(location)
	if err != nil {
		return pluginStatus{}, err
	}
	f, err := readPluginFile()
	if err != nil {
		return pluginStatus{}, err
	}
	for _, src := range f.Sources {
		if src.Location == loc {
			return inspectPlugin(src, false)
		}
	}
	rec := pluginRecord{ID: newStampedID("plugin"), Kind: kind, Location: loc, Enabled: true}
	if kind == "git" {
		if err := syncPluginGit(rec, true); err != nil {
			return pluginStatus{}, err
		}
	}
	f.Sources = append(f.Sources, rec)
	if err := writePluginFile(f); err != nil {
		return pluginStatus{}, err
	}
	return inspectPlugin(rec, false)
}

func setPluginEnabled(id string, enabled bool) (pluginStatus, error) {
	f, err := readPluginFile()
	if err != nil {
		return pluginStatus{}, err
	}
	for i := range f.Sources {
		if f.Sources[i].ID != id {
			continue
		}
		f.Sources[i].Enabled = enabled
		if err := writePluginFile(f); err != nil {
			return pluginStatus{}, err
		}
		return inspectPlugin(f.Sources[i], false)
	}
	return pluginStatus{}, fmt.Errorf("unknown plugin %q", id)
}

func refreshPlugin(id string) (pluginStatus, error) {
	f, err := readPluginFile()
	if err != nil {
		return pluginStatus{}, err
	}
	for _, src := range f.Sources {
		if src.ID != id {
			continue
		}
		if src.Kind == "git" {
			if err := syncPluginGit(src, true); err != nil {
				st, _ := inspectPlugin(src, false)
				st.Error = err.Error()
				return st, nil
			}
		}
		return inspectPlugin(src, false)
	}
	return pluginStatus{}, fmt.Errorf("unknown plugin %q", id)
}

func removePlugin(id string) error {
	f, err := readPluginFile()
	if err != nil {
		return err
	}
	next := make([]pluginRecord, 0, len(f.Sources))
	var found *pluginRecord
	for _, src := range f.Sources {
		if src.ID == id {
			cp := src
			found = &cp
			continue
		}
		next = append(next, src)
	}
	if found == nil {
		return fmt.Errorf("unknown plugin %q", id)
	}
	f.Sources = next
	if err := writePluginFile(f); err != nil {
		return err
	}
	if found.Kind == "git" {
		_ = os.RemoveAll(pluginCheckoutDir(found.ID))
		_ = os.RemoveAll(pluginDataDir(found.ID))
	}
	return nil
}

func listPluginStatus() ([]pluginStatus, error) {
	f, err := readPluginFile()
	if err != nil {
		return nil, err
	}
	out := make([]pluginStatus, 0, len(f.Sources))
	for _, src := range f.Sources {
		st, err := inspectPlugin(src, false)
		if err != nil {
			st = pluginStatus{ID: src.ID, Kind: src.Kind, Location: src.Location, Enabled: src.Enabled, Error: err.Error()}
		}
		out = append(out, st)
	}
	return out, nil
}

func syncPluginGit(rec pluginRecord, fetch bool) error {
	dest := pluginCheckoutDir(rec.ID)
	if !fetch && dirExists(dest) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if dirExists(filepath.Join(dest, ".git")) {
		cmd := exec.CommandContext(ctx, "git", "-C", dest, "pull", "--ff-only")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git pull: %s", pluginOneLine(string(out)))
		}
		return nil
	}
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--", rec.Location, dest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(dest)
		msg := pluginOneLine(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("git clone: %s", msg)
	}
	return nil
}

func pluginRoot(rec pluginRecord, fetch bool) (string, error) {
	if rec.Kind == "git" {
		if err := syncPluginGit(rec, fetch); err != nil {
			return "", err
		}
		return pluginCheckoutDir(rec.ID), nil
	}
	if !dirExists(rec.Location) {
		return "", fmt.Errorf("directory not found")
	}
	return rec.Location, nil
}

func inspectPlugin(rec pluginRecord, fetch bool) (pluginStatus, error) {
	return describePlugin(rec, fetch), nil
}

type pluginManifest struct {
	Schema      string `json:"$schema"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

func readPluginManifest(root string) (pluginManifest, error) {
	path := filepath.Join(root, "plugin.json")
	if !withinRoot(root, path) {
		return pluginManifest{}, fmt.Errorf("plugin.json is outside the plugin root")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return pluginManifest{}, fmt.Errorf("plugin.json not found")
		}
		return pluginManifest{}, err
	}
	var m pluginManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return pluginManifest{}, fmt.Errorf("plugin.json: %w", err)
	}
	if m.Schema != agentPluginsSchema {
		return pluginManifest{}, fmt.Errorf("unsupported plugin schema")
	}
	if !validPluginName(m.Name) {
		return pluginManifest{}, fmt.Errorf("invalid plugin name %q", m.Name)
	}
	return m, nil
}

func validPluginName(name string) bool {
	return len(name) >= 1 && len(name) <= 64 && pluginNameRe.MatchString(name)
}

func pluginSkills(root string) []*SkillMeta {
	dir := filepath.Join(root, "skills")
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return nil
	}
	if !withinRoot(root, dir) {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []*SkillMeta
	for _, e := range entries {
		child := filepath.Join(dir, e.Name())
		info, err := os.Stat(child)
		if err != nil || !info.IsDir() || !withinRoot(root, child) {
			continue
		}
		sk, err := loadSkillDir(child, skillSourcePlugin, false)
		if err != nil {
			continue
		}
		out = append(out, sk)
	}
	return out
}

func pluginSkillRoots() []string {
	var roots []string
	eachEnabledUnit(func(_ pluginRecord, unit pluginUnit) {
		dir := filepath.Join(unit.Root, "skills")
		if st, err := os.Stat(dir); err == nil && st.IsDir() && withinRoot(unit.Root, dir) {
			roots = append(roots, dir)
		}
	})
	return roots
}

func pluginMCPOverlay() (map[string]MCPServerConfig, map[string]string) {
	out := map[string]MCPServerConfig{}
	labels := map[string]string{}
	eachEnabledUnit(func(src pluginRecord, unit pluginUnit) {
		servers, _ := pluginServers(src.ID, unit.Root)
		label := unit.Name
		if label == "" {
			label = src.ID
		}
		for name, cfg := range servers {
			if _, exists := out[name]; exists {
				continue
			}
			out[name] = cfg
			labels[name] = label
		}
	})
	return out, labels
}

func pluginServers(id, root string) (map[string]MCPServerConfig, []string) {
	path := filepath.Join(root, "mcp.json")
	if _, err := os.Stat(path); err != nil {
		alt := filepath.Join(root, ".mcp.json")
		if _, err2 := os.Stat(alt); err2 == nil {
			path = alt
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{"mcp.json: " + err.Error()}
	}
	if !withinRoot(root, path) {
		return nil, []string{"mcp.json is outside the plugin root"}
	}
	var doc struct {
		Schema     string                     `json:"$schema"`
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, []string{"mcp.json: " + err.Error()}
	}
	if doc.Schema != "" && doc.Schema != agentPluginsMCPSchema {
		return nil, []string{filepath.Base(path) + " schema does not match the plugin"}
	}
	data := pluginDataDir(id)
	_ = os.MkdirAll(data, 0o700)
	out := map[string]MCPServerConfig{}
	var notes []string
	for name, raw := range doc.MCPServers {
		name = strings.TrimSpace(name)
		if name == "" {
			notes = append(notes, "skipped a server with an empty name")
			continue
		}
		cfg, err := convertPluginServer(root, data, raw)
		if err != nil {
			notes = append(notes, name+": "+err.Error())
			continue
		}
		out[name] = cfg
	}
	return out, notes
}

func convertPluginServer(root, data string, raw json.RawMessage) (MCPServerConfig, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return MCPServerConfig{}, err
	}
	var typ string
	if raw, ok := fields["type"]; ok {
		_ = json.Unmarshal(raw, &typ)
	}
	typ = strings.TrimSpace(typ)
	if typ == "" {
		switch {
		case fields["command"] != nil:
			typ = "stdio"
		case fields["url"] != nil:
			typ = "streamable-http"
		default:
			return MCPServerConfig{}, fmt.Errorf("type required")
		}
	}
	switch typ {
	case "stdio":
		return convertStdioServer(root, data, fields)
	case "streamable-http", "sse":
		return convertRemoteServer(typ, fields)
	default:
		return MCPServerConfig{}, fmt.Errorf("unsupported transport %q", typ)
	}
}

func convertStdioServer(root, data string, fields map[string]json.RawMessage) (MCPServerConfig, error) {
	allowed := map[string]bool{"type": true, "command": true, "args": true, "env": true, "cwd": true}
	for key := range fields {
		if !allowed[key] {
			return MCPServerConfig{}, fmt.Errorf("unknown field %q", key)
		}
	}
	var command string
	if err := json.Unmarshal(fields["command"], &command); err != nil {
		return MCPServerConfig{}, fmt.Errorf("command required")
	}
	command = strings.TrimSpace(command)
	if command == "" || strings.ContainsAny(command, " \t") || strings.Contains(command, "${") {
		return MCPServerConfig{}, fmt.Errorf("command must be one token")
	}
	if strings.HasPrefix(command, "./") {
		abs := filepath.Join(root, filepath.FromSlash(command[2:]))
		if !withinRoot(root, abs) {
			return MCPServerConfig{}, fmt.Errorf("command escapes the plugin root")
		}
		command = abs
	} else if strings.ContainsAny(command, `/\`) {
		return MCPServerConfig{}, fmt.Errorf("command must be a bare name or a ./ path")
	}
	var args []string
	if raw, ok := fields["args"]; ok {
		if err := json.Unmarshal(raw, &args); err != nil {
			return MCPServerConfig{}, fmt.Errorf("args must be strings")
		}
		for i, arg := range args {
			args[i] = expandPluginVars(arg, root, data)
		}
	}
	env := map[string]string{}
	if raw, ok := fields["env"]; ok {
		if err := json.Unmarshal(raw, &env); err != nil {
			return MCPServerConfig{}, fmt.Errorf("env must be strings")
		}
		for k, v := range env {
			env[k] = expandPluginVars(v, root, data)
		}
	}
	cwd := root
	if raw, ok := fields["cwd"]; ok {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return MCPServerConfig{}, fmt.Errorf("cwd must be a string")
		}
		resolved, err := resolvePluginCwd(root, data, text)
		if err != nil {
			return MCPServerConfig{}, err
		}
		cwd = resolved
	}
	env["PLUGIN_ROOT"] = root
	env["PLUGIN_DATA"] = data
	return MCPServerConfig{Command: command, Args: args, Env: env, Cwd: cwd, Type: "stdio"}, nil
}

func convertRemoteServer(typ string, fields map[string]json.RawMessage) (MCPServerConfig, error) {
	allowed := map[string]bool{"type": true, "url": true, "headers": true}
	for key := range fields {
		if !allowed[key] {
			return MCPServerConfig{}, fmt.Errorf("unknown field %q", key)
		}
	}
	var rawURL string
	if err := json.Unmarshal(fields["url"], &rawURL); err != nil {
		return MCPServerConfig{}, fmt.Errorf("url required")
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.Fragment != "" {
		return MCPServerConfig{}, fmt.Errorf("url must be an absolute http(s) URL")
	}
	if u.Scheme == "http" && !loopbackHost(u.Hostname()) {
		return MCPServerConfig{}, fmt.Errorf("url must use https")
	}
	headers := map[string]string{}
	if raw, ok := fields["headers"]; ok {
		if err := json.Unmarshal(raw, &headers); err != nil {
			return MCPServerConfig{}, fmt.Errorf("headers must be strings")
		}
		seen := map[string]bool{}
		for name := range headers {
			key := strings.ToLower(name)
			if seen[key] || strings.ContainsAny(name, " \t\r\n:") {
				return MCPServerConfig{}, fmt.Errorf("invalid header %q", name)
			}
			seen[key] = true
		}
	}
	kind := "streamableHttp"
	if typ == "sse" {
		kind = "sse"
	}
	return MCPServerConfig{URL: rawURL, Headers: headers, Type: kind}, nil
}

func resolvePluginCwd(root, data, text string) (string, error) {
	text = strings.TrimSpace(text)
	switch {
	case text == "${PLUGIN_ROOT}":
		return root, nil
	case strings.HasPrefix(text, "${PLUGIN_ROOT}/"):
		abs := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(text, "${PLUGIN_ROOT}/")))
		if !withinRoot(root, abs) {
			return "", fmt.Errorf("cwd escapes the plugin root")
		}
		return abs, nil
	case text == "${PLUGIN_DATA}":
		return data, nil
	case strings.HasPrefix(text, "${PLUGIN_DATA}/"):
		abs := filepath.Join(data, filepath.FromSlash(strings.TrimPrefix(text, "${PLUGIN_DATA}/")))
		if !withinRoot(data, abs) {
			return "", fmt.Errorf("cwd escapes the plugin data directory")
		}
		return abs, nil
	case strings.HasPrefix(text, "./"):
		abs := filepath.Join(root, filepath.FromSlash(text[2:]))
		if !withinRoot(root, abs) {
			return "", fmt.Errorf("cwd escapes the plugin root")
		}
		return abs, nil
	default:
		return "", fmt.Errorf("cwd must stay inside the plugin")
	}
}

func expandPluginVars(s, root, data string) string {
	for _, key := range []string{"${PLUGIN_ROOT}", "${CURSOR_PLUGIN_ROOT}", "${CLAUDE_PLUGIN_ROOT}"} {
		s = strings.ReplaceAll(s, key, root)
	}
	s = strings.ReplaceAll(s, "${PLUGIN_DATA}", data)
	return s
}

func loopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func withinRoot(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if !relInside(root, path) {
		return false
	}
	rr, err1 := filepath.EvalSymlinks(root)
	pp, err2 := filepath.EvalSymlinks(path)
	if err1 != nil || err2 != nil {
		return true
	}
	return relInside(rr, pp)
}

func relInside(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func pluginOneLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if len(s) > 300 {
		s = s[:300]
	}
	return s
}
