package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	agentPluginsSchema    = "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json"
	agentPluginsMCPSchema = "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json"
)

var pluginNameRe = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+)*$`)

type pluginRecord struct {
	ID        string   `json:"id"`
	Kind      string   `json:"kind"` // git | dir
	Location  string   `json:"location"`
	SSHPort   string   `json:"ssh_port,omitempty"`
	Enabled   bool     `json:"enabled"`
	Grants    []string `json:"grants,omitempty"`
	PluginSet bool     `json:"plugin_set,omitempty"` // marketplace choices have been stored
	Units     []string `json:"units,omitempty"`      // marketplace plugin names the user turned on
}

type pluginFile struct {
	Sources []pluginRecord `json:"sources"`
}

type pluginStatus struct {
	ID          string             `json:"id"`
	Kind        string             `json:"kind"`
	Layout      string             `json:"layout,omitempty"` // plugin | marketplace
	Location    string             `json:"location"`
	SSHPort     string             `json:"ssh_port,omitempty"`
	Enabled     bool               `json:"enabled"`
	Name        string             `json:"name,omitempty"`
	Description string             `json:"description,omitempty"`
	Version     string             `json:"version,omitempty"`
	Plugins     []string           `json:"plugins,omitempty"`
	Active      []string           `json:"active,omitempty"`
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

func addPluginSourceWith(location, port string, broker *CredentialBroker) (pluginStatus, error) {
	kind, loc, err := classifyPluginLocation(location)
	if err != nil {
		return pluginStatus{}, err
	}
	sshPort := ""
	if kind == "git" {
		if _, _, err := gitRemoteForClone(loc, port); err != nil {
			return pluginStatus{}, err
		}
		sshPort = normalizePluginSSHPort(port)
	} else if strings.TrimSpace(port) != "" {
		return pluginStatus{}, fmt.Errorf("SSH port applies to a Git address over SSH")
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
	rec := pluginRecord{ID: newStampedID("plugin"), Kind: kind, Location: loc, SSHPort: sshPort, Enabled: true, PluginSet: true}
	if kind == "git" {
		if err := syncPluginGitWith(rec, true, broker); err != nil {
			return pluginStatus{}, err
		}
	}
	f.Sources = append(f.Sources, rec)
	if err := writePluginFile(f); err != nil {
		return pluginStatus{}, err
	}
	return inspectPlugin(rec, false)
}

func addPluginSource(location string) (pluginStatus, error) {
	return addPluginSourceWith(location, "", nil)
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
	return refreshPluginWith(id, nil)
}

func refreshPluginWith(id string, broker *CredentialBroker) (pluginStatus, error) {
	f, err := readPluginFile()
	if err != nil {
		return pluginStatus{}, err
	}
	for _, src := range f.Sources {
		if src.ID != id {
			continue
		}
		if src.Kind == "git" {
			if err := syncPluginGitWith(src, true, broker); err != nil {
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
	if err := migratePluginUnits(); err != nil {
		return nil, err
	}
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
	return syncPluginGitWith(rec, fetch, nil)
}

func syncPluginGitWith(rec pluginRecord, fetch bool, broker *CredentialBroker) error {
	dest := pluginCheckoutDir(rec.ID)
	if !fetch && dirExists(dest) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	remote, sshCommand, err := gitRemoteForClone(rec.Location, rec.SSHPort)
	if err != nil {
		return err
	}
	branch, err := pluginGitDefaultBranch(ctx, remote, rec.Location, sshCommand, broker)
	if err != nil {
		return err
	}
	if dirExists(filepath.Join(dest, ".git")) {
		return fetchPluginGitBranch(ctx, dest, branch, rec.Location, sshCommand, broker)
	}
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", branch, "--", remote, dest)
	cleanup, err := applyPluginGitAuth(cmd, broker, rec.Location, sshCommand)
	if err != nil {
		return err
	}
	defer cleanup()
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(dest)
		return errors.New(pluginGitFailure(string(out), err))
	}
	return nil
}

// pluginGitDefaultBranch is the branch a clone should check out. A server
// whose HEAD still names a deleted branch (often master after a rename to
// main) advertises no usable HEAD; the single remaining branch is used.
func pluginGitDefaultBranch(ctx context.Context, remote, location, sshCommand string, broker *CredentialBroker) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--symref", "--", remote, "HEAD", "refs/heads/*")
	cleanup, err := applyPluginGitAuth(cmd, broker, location, sshCommand)
	if err != nil {
		return "", err
	}
	defer cleanup()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(pluginGitFailure(string(out), err))
	}
	return pickPluginGitBranch(string(out))
}

func pickPluginGitBranch(out string) (string, error) {
	var sym string
	var branches []string
	seen := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ref:") {
			ref, _, _ := strings.Cut(strings.TrimSpace(strings.TrimPrefix(line, "ref:")), "\t")
			name, ok := strings.CutPrefix(strings.TrimSpace(ref), "refs/heads/")
			if ok && validGitBranch(name) {
				sym = name
			}
			continue
		}
		_, ref, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		name, ok := strings.CutPrefix(strings.TrimSpace(ref), "refs/heads/")
		if !ok || !validGitBranch(name) || seen[name] {
			continue
		}
		seen[name] = true
		branches = append(branches, name)
	}
	has := func(name string) bool { return seen[name] }
	if sym != "" && has(sym) {
		return sym, nil
	}
	if len(branches) == 1 {
		return branches[0], nil
	}
	if has("main") {
		return "main", nil
	}
	if has("master") {
		return "master", nil
	}
	if len(branches) == 0 {
		return "", fmt.Errorf("the Git server has no branch")
	}
	return "", fmt.Errorf("the Git server does not name a default branch (%s)", strings.Join(branches, ", "))
}

func validGitBranch(name string) bool {
	if name == "" || name == "HEAD" || strings.HasPrefix(name, "-") || strings.Contains(name, "..") {
		return false
	}
	return !strings.ContainsAny(name, " \t\r\n~^:?*[\\")
}

func fetchPluginGitBranch(ctx context.Context, dest, branch, location, sshCommand string, broker *CredentialBroker) error {
	fetch := exec.CommandContext(ctx, "git", "-C", dest, "fetch", "--depth", "1", "origin", branch)
	cleanup, err := applyPluginGitAuth(fetch, broker, location, sshCommand)
	if err != nil {
		return err
	}
	defer cleanup()
	out, err := fetch.CombinedOutput()
	if err != nil {
		return errors.New(pluginGitFailure(string(out), err))
	}
	co := exec.CommandContext(ctx, "git", "-C", dest, "checkout", "-f", "-B", branch, "FETCH_HEAD")
	out, err = co.CombinedOutput()
	if err != nil {
		return errors.New(pluginGitFailure(string(out), err))
	}
	return nil
}

// gitRemoteForClone keeps a git@host:path address unchanged and carries the
// port as an ssh -p option, so the path stays the one Git already understands.
// An ssh:// address takes the port in the URL. An empty port changes nothing.
func gitRemoteForClone(location, port string) (remote, sshCommand string, err error) {
	remote = strings.TrimSpace(location)
	port = normalizePluginSSHPort(port)
	if port == "" {
		return remote, "", nil
	}
	n, nerr := strconv.Atoi(port)
	if nerr != nil || n < 1 || n > 65535 {
		return "", "", fmt.Errorf("SSH port must be a number from 1 to 65535")
	}
	port = strconv.Itoa(n)
	if strings.HasPrefix(remote, "git@") {
		if _, ok := gitSSHHost(remote); !ok {
			return "", "", fmt.Errorf("SSH port applies to a Git address over SSH")
		}
		return remote, "ssh -p " + port, nil
	}
	u, perr := url.Parse(remote)
	if perr != nil || !strings.EqualFold(u.Scheme, "ssh") || u.Hostname() == "" {
		return "", "", fmt.Errorf("SSH port applies to a Git address over SSH")
	}
	u.Host = net.JoinHostPort(u.Hostname(), port)
	return u.String(), "", nil
}

func normalizePluginSSHPort(port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		return ""
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return port
	}
	return strconv.Itoa(n)
}

// applyPluginGitAuth gives an SSH remote the vault key for that host.
// A missing vault, a locked vault, or no key for the host leaves the process SSH setup in place.
// sshCommand, when set, is GIT_SSH_COMMAND for this one git process.
func applyPluginGitAuth(cmd *exec.Cmd, broker *CredentialBroker, location, sshCommand string) (func(), error) {
	host, ok := gitSSHHost(location)
	var keys []Credential
	if ok {
		keys = broker.sshKeysForHost(host)
	}
	if len(keys) == 0 && sshCommand == "" {
		return func() {}, nil
	}
	env := append([]string{}, os.Environ()...)
	if sshCommand != "" {
		env = append(env, "GIT_SSH_COMMAND="+sshCommand)
	}
	if len(keys) == 0 {
		cmd.Env = env
		return func() {}, nil
	}
	sess, err := startSSHAgentWithKeys(keys)
	if err != nil {
		return func() {}, err
	}
	cmd.Env = append(env, sess.Environ()...)
	return sess.Close, nil
}

func gitSSHHost(location string) (string, bool) {
	location = strings.TrimSpace(location)
	if strings.HasPrefix(location, "git@") {
		rest := strings.TrimPrefix(location, "git@")
		host, _, found := strings.Cut(rest, ":")
		host = strings.TrimSpace(host)
		return host, found && host != ""
	}
	u, err := url.Parse(location)
	if err != nil || u.Hostname() == "" {
		return "", false
	}
	if strings.EqualFold(u.Scheme, "ssh") {
		return u.Hostname(), true
	}
	return "", false
}

func pluginGitFailure(out string, err error) string {
	out = strings.TrimRight(out, "\n")
	if strings.TrimSpace(out) == "" && err != nil {
		return err.Error()
	}
	const max = 64 << 10
	if len(out) > max {
		return out[:max] + "\n… output truncated"
	}
	return out
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
