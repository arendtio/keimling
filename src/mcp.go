package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type mcpToolRoute struct {
	Server string
	Tool   string
}

type MCPServerStatus struct {
	Name       string   `json:"name"`
	Transport  string   `json:"transport"`
	Status     string   `json:"status"` // connected | error | disabled
	Error      string   `json:"error,omitempty"`
	Tools      []string `json:"tools,omitempty"`
	Allow      []string `json:"allow,omitempty"`
	Command    string   `json:"command,omitempty"`
	Args       []string `json:"args,omitempty"`
	Cwd        string   `json:"cwd,omitempty"`
	URL        string   `json:"url,omitempty"`
	Disabled   bool     `json:"disabled,omitempty"`
	Grant      string   `json:"grant,omitempty"`
	Auth       string   `json:"auth,omitempty"`
	Before     string   `json:"before,omitempty"`
	Note       string   `json:"note,omitempty"`
	OAuth      string   `json:"oauth,omitempty"`
	Source     string   `json:"source,omitempty"`      // user, config, project, cursor, plugin
	SourceName string   `json:"source_name,omitempty"` // plugin name when Source is plugin
}

type mcpSession struct {
	name         string
	cfg          MCPServerConfig
	client       *mcpclient.Client
	tools        []mcp.Tool
	instructions string
	status       string
	err          string
}

// MCPManager holds backend MCP clients (stdio/HTTP/SSE). The browser never talks to MCP servers.
type MCPManager struct {
	mu          sync.RWMutex
	sessions    map[string]*mcpSession
	routes      map[string]mcpToolRoute
	defs        []map[string]any
	loaded      map[string]struct{}
	sources     map[string]string
	sourceNames map[string]string
	logs        map[string]*mcpLogBuf
	// oauthMu guards oauthLocks. One refresh per server, so a rotated
	// refresh token is not presented twice.
	oauthMu    sync.Mutex
	oauthLocks map[string]*sync.Mutex
	// startMu guards startCmd against a settings change while a run reads it.
	// The success itself lives in mcpStartGate: MCP servers belong to the
	// process, so every workspace worker shares the same start hook.
	startMu  sync.Mutex
	startCmd string
	startRan string
	startOK  bool
	// scriptRunning is set while a start or before script is inside its
	// command, so the UI can say the servers are not connected yet.
	scriptRunning bool
	scriptKind    string
	scriptServer  string
	// startDone is open while a start runs and closed when it ends, so a run
	// that begins right after a restart waits for the servers instead of
	// reading an empty tool list.
	startDone chan struct{}
}

// mcpLogLimit is the newest process output and connection errors kept per server.
const mcpLogLimit = 64 << 10

type mcpLogBuf struct {
	mu sync.Mutex
	b  []byte
}

func (l *mcpLogBuf) Write(p []byte) (int, error) {
	if l == nil || len(p) == 0 {
		return len(p), nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.b = append(l.b, p...)
	if len(l.b) > mcpLogLimit {
		l.b = append([]byte(nil), l.b[len(l.b)-mcpLogLimit:]...)
		if i := bytes.IndexByte(l.b, '\n'); i >= 0 && i < 2048 {
			l.b = l.b[i+1:]
		}
	}
	return len(p), nil
}

func (l *mcpLogBuf) String() string {
	if l == nil {
		return ""
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return string(l.b)
}

func NewMCPManager() *MCPManager {
	return &MCPManager{
		sessions: map[string]*mcpSession{},
		routes:   map[string]mcpToolRoute{},
	}
}

func (m *MCPManager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.client != nil {
			_ = s.client.Close()
			s.client = nil
		}
	}
	m.sessions = map[string]*mcpSession{}
	m.routes = map[string]mcpToolRoute{}
	m.defs = nil
}

func (m *MCPManager) Reload(ctx context.Context, configs map[string]MCPServerConfig) {
	m.Close()
	m.connectAll(ctx, configs, true)
}

// ReconnectOne replaces a single server session. The others stay connected.
func (m *MCPManager) ReconnectOne(ctx context.Context, name string, cfg MCPServerConfig) {
	if m == nil {
		return
	}
	m.logLine(name, "reload")
	m.mu.Lock()
	if s := m.sessions[name]; s != nil && s.client != nil {
		_ = s.client.Close()
		s.client = nil
	}
	delete(m.sessions, name)
	m.mu.Unlock()
	if !cfg.Disabled {
		if err := m.runStart(ctx, false); err != nil {
			m.noteBlocked(name, cfg, err)
			m.rebuildIndex()
			return
		}
	}
	m.connectOne(ctx, name, cfg)
	m.rebuildIndex()
}

func (m *MCPManager) serverLog(name string) *mcpLogBuf {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.logs == nil {
		m.logs = map[string]*mcpLogBuf{}
	}
	if m.logs[name] == nil {
		m.logs[name] = &mcpLogBuf{}
	}
	return m.logs[name]
}

func (m *MCPManager) logLine(name, line string) {
	if m == nil {
		return
	}
	line = strings.TrimRight(line, "\n")
	if strings.TrimSpace(line) == "" {
		return
	}
	_, _ = m.serverLog(name).Write([]byte(line + "\n"))
}

// Log is the recent stderr and connection errors for one server.
func (m *MCPManager) Log(name string) string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	l := m.logs[name]
	m.mu.RUnlock()
	if l == nil {
		return ""
	}
	return l.String()
}

func (m *MCPManager) ConnectAll(ctx context.Context, configs map[string]MCPServerConfig) {
	m.connectAll(ctx, configs, false)
}

func (m *MCPManager) connectAll(ctx context.Context, configs map[string]MCPServerConfig, forceStart bool) {
	if m == nil {
		return
	}
	startErr := m.runStart(ctx, forceStart)
	names := make([]string, 0, len(configs))
	for name := range configs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cfg := configs[name]
		if !cfg.Disabled && startErr != nil {
			m.noteBlocked(name, cfg, startErr)
			continue
		}
		m.connectOne(ctx, name, cfg)
	}
	m.rebuildIndex()
}

func (m *MCPManager) noteBlocked(name string, cfg MCPServerConfig, err error) {
	sess := &mcpSession{name: name, cfg: cfg, status: "error"}
	if err != nil {
		sess.err = err.Error()
		m.logLine(name, sess.err)
	}
	m.mu.Lock()
	m.sessions[name] = sess
	m.mu.Unlock()
}

func (m *MCPManager) connectOne(ctx context.Context, name string, cfg MCPServerConfig) {
	sess := &mcpSession{name: name, cfg: cfg, status: "error"}
	store := func() {
		m.mu.Lock()
		m.sessions[name] = sess
		m.mu.Unlock()
	}
	fail := func(err error) {
		if err != nil {
			sess.err = err.Error()
			m.logLine(name, sess.err)
		}
		store()
	}
	if cfg.Disabled {
		sess.status = "disabled"
		store()
		return
	}
	if err := cfg.validate(); err != nil {
		fail(err)
		return
	}
	if before := strings.TrimSpace(cfg.Before); before != "" {
		err := func() error {
			m.beginScript("before", name)
			defer m.endScript()
			return runShellCommand(ctx, before)
		}()
		if err != nil {
			fail(fmt.Errorf("before script failed: %w", err))
			return
		}
	}
	cfg, err := prepareMCPConnection(ctx, name, cfg)
	if err != nil {
		sess.cfg = cfg
		fail(err)
		return
	}
	sess.cfg = cfg
	cctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cli, kind, instructions, err := dialMCP(cctx, cfg, m.serverLog(name))
	if kind != "" {
		sess.cfg.Type = kind
	}
	if err != nil {
		fail(err)
		return
	}
	listed, err := cli.ListTools(cctx, mcp.ListToolsRequest{})
	if err != nil {
		_ = cli.Close()
		fail(fmt.Errorf("list tools: %w", err))
		return
	}
	sess.client = cli
	sess.instructions = strings.TrimSpace(instructions)
	sess.status = "connected"
	sess.err = ""
	if listed != nil {
		sess.tools = listed.Tools
	}
	store()
}

func dialMCP(ctx context.Context, cfg MCPServerConfig, stderr io.Writer) (*mcpclient.Client, string, string, error) {
	kind := cfg.transportKind()
	if kind == "streamableHttp" && !cfg.remoteTypeSet() {
		cli, instructions, err := connectMCP(ctx, cfg, stderr)
		if err == nil {
			return cli, "streamableHttp", instructions, nil
		}
		if legacySSEServer(err) {
			sse := cfg
			sse.Type = "sse"
			cli, instructions, err2 := connectMCP(ctx, sse, stderr)
			if err2 == nil {
				return cli, "sse", instructions, nil
			}
			return nil, "sse", "", err2
		}
		return nil, "streamableHttp", "", err
	}
	cli, instructions, err := connectMCP(ctx, cfg, stderr)
	return cli, kind, instructions, err
}

func legacySSEServer(err error) bool {
	return err != nil && (errors.Is(err, transport.ErrLegacySSEServer) || strings.Contains(err.Error(), "legacy SSE"))
}

func connectMCP(ctx context.Context, cfg MCPServerConfig, stderr io.Writer) (*mcpclient.Client, string, error) {
	cli, err := openMCPClient(cfg, stderr)
	if err != nil {
		return nil, "", err
	}
	if err := cli.Start(ctx); err != nil {
		_ = cli.Close()
		return nil, "", fmt.Errorf("start: %w", err)
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "keimling", Version: "1.0.0"}
	res, err := cli.Initialize(ctx, initReq)
	if err != nil {
		_ = cli.Close()
		return nil, "", fmt.Errorf("initialize: %w", err)
	}
	return cli, res.Instructions, nil
}

func openMCPClient(cfg MCPServerConfig, stderr io.Writer) (*mcpclient.Client, error) {
	switch cfg.transportKind() {
	case "stdio":
		env := make([]string, 0, len(cfg.Env))
		for k, v := range cfg.Env {
			env = append(env, k+"="+v)
		}
		sort.Strings(env)
		var opts []transport.StdioOption
		if stderr != nil {
			opts = append(opts, transport.WithCommandStderrWriter(stderr))
		}
		if cwd := strings.TrimSpace(cfg.Cwd); cwd != "" {
			opts = append(opts, transport.WithCommandFunc(func(ctx context.Context, command string, env []string, args []string) (*exec.Cmd, error) {
				cmd := exec.CommandContext(ctx, command, args...)
				cmd.Env = append(os.Environ(), env...)
				cmd.Dir = cwd
				return cmd, nil
			}))
		}
		if len(opts) == 0 {
			return mcpclient.NewStdioMCPClient(cfg.Command, env, cfg.Args...)
		}
		return mcpclient.NewStdioMCPClientWithOptions(cfg.Command, env, cfg.Args, opts...)
	case "sse":
		opts := []transport.ClientOption{}
		if len(cfg.Headers) > 0 {
			opts = append(opts, mcpclient.WithHeaders(cfg.Headers))
		}
		return mcpclient.NewSSEMCPClient(cfg.URL, opts...)
	case "streamableHttp":
		opts := []transport.StreamableHTTPCOption{}
		if len(cfg.Headers) > 0 {
			opts = append(opts, transport.WithHTTPHeaders(cfg.Headers))
		}
		return mcpclient.NewStreamableHttpClient(cfg.URL, opts...)
	default:
		return nil, fmt.Errorf("unsupported transport")
	}
}

func (m *MCPManager) rebuildIndex() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routes = map[string]mcpToolRoute{}
	m.defs = nil
	names := make([]string, 0, len(m.sessions))
	for name := range m.sessions {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		sess := m.sessions[name]
		if sess == nil || sess.status != "connected" {
			continue
		}
		for _, tool := range sess.tools {
			exposed := mcpExposedName(name, tool.Name)
			base := exposed
			for n := 2; ; n++ {
				if _, taken := m.routes[exposed]; !taken {
					break
				}
				exposed = fmt.Sprintf("%s_%d", base, n)
			}
			m.routes[exposed] = mcpToolRoute{Server: name, Tool: tool.Name}
			m.defs = append(m.defs, mcpToolDefinition(exposed, tool))
		}
	}
}

func mcpToolDefinition(exposed string, tool mcp.Tool) map[string]any {
	desc := strings.TrimSpace(tool.Description)
	if desc == "" {
		desc = "MCP tool " + tool.Name
	} else {
		desc = desc + " (MCP)"
	}
	desc += " Optional credential_id selects a Keimling vault secret (never pass raw secrets)."
	params := schemaToParams(tool.InputSchema)
	if props, ok := params["properties"].(map[string]any); ok {
		props["credential_id"] = map[string]any{
			"type":        "string",
			"description": "Keimling credential id from list_credentials for this tool; omit to auto-pick when exactly one is allowed",
		}
	}
	return map[string]any{
		"type":        "function",
		"name":        exposed,
		"description": desc,
		"parameters":  params,
		"strict":      false,
	}
}

func schemaToParams(schema mcp.ToolInputSchema) map[string]any {
	b, err := json.Marshal(schema)
	if err != nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	var params map[string]any
	if err := json.Unmarshal(b, &params); err != nil || params == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	if _, ok := params["type"]; !ok {
		params["type"] = "object"
	}
	if _, ok := params["properties"]; !ok {
		params["properties"] = map[string]any{}
	}
	return params
}

func (m *MCPManager) ToolDefs() []map[string]any {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.defs) == 0 {
		return nil
	}
	out := make([]map[string]any, len(m.defs))
	copy(out, m.defs)
	return out
}

func (m *MCPManager) noteAllow(name string, tools []string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[name]; ok && s != nil {
		s.cfg.AllowTools = append([]string(nil), tools...)
	}
}

func (m *MCPManager) Status() []MCPServerStatus {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.sessions))
	for name := range m.sessions {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]MCPServerStatus, 0, len(names))
	for _, name := range names {
		s := m.sessions[name]
		st := MCPServerStatus{
			Name:       name,
			Transport:  s.cfg.transportKind(),
			Status:     s.status,
			Error:      s.err,
			Command:    s.cfg.Command,
			Args:       append([]string(nil), s.cfg.Args...),
			Cwd:        s.cfg.Cwd,
			URL:        s.cfg.URL,
			Before:     s.cfg.Before,
			Note:       s.cfg.Note,
			Disabled:   s.cfg.Disabled,
			Grant:      mcpServerGrantID(name),
			Auth:       s.cfg.authKind(),
			OAuth:      s.cfg.oauthState(time.Now()),
			Source:     m.sources[name],
			SourceName: m.sourceNames[name],
		}
		for _, t := range s.tools {
			st.Tools = append(st.Tools, mcpExposedName(name, t.Name))
		}
		st.Allow = append([]string(nil), s.cfg.AllowTools...)
		out = append(out, st)
	}
	return out
}

// mcpCallTimeout is the harness limit for one MCP tool call. A server job may
// run for many minutes; a shorter deadline cancels it before the server does.
// Stopping the run still cancels the call through the parent context.
const mcpCallTimeout = 60 * time.Minute

func (m *MCPManager) Call(ctx context.Context, exposedName, argumentsJSON string) string {
	display, _, _, _ := m.callStored(ctx, exposedName, argumentsJSON)
	return display
}

func mcpAuthRejected(err error) bool {
	return err != nil && errors.Is(err, transport.ErrAuthorizationRequired)
}

func (m *MCPManager) oauthLock(name string) *sync.Mutex {
	m.oauthMu.Lock()
	defer m.oauthMu.Unlock()
	if m.oauthLocks == nil {
		m.oauthLocks = map[string]*sync.Mutex{}
	}
	if m.oauthLocks[name] == nil {
		m.oauthLocks[name] = &sync.Mutex{}
	}
	return m.oauthLocks[name]
}

func (m *MCPManager) sessionSnapshot(name string) (MCPServerConfig, *mcpclient.Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := m.sessions[name]
	if s == nil {
		return MCPServerConfig{}, nil, false
	}
	return s.cfg, s.client, true
}

// renewOAuth refreshes one server's access token and reconnects it.
// rejectedAccess is the bearer the failed call used. A refresh that already
// replaced that bearer is not run again.
func (m *MCPManager) renewOAuth(ctx context.Context, name, rejectedAccess string, force bool) error {
	if m == nil {
		return fmt.Errorf("sign-in expired")
	}
	lock := m.oauthLock(name)
	lock.Lock()
	defer lock.Unlock()

	sessCfg, _, _ := m.sessionSnapshot(name)
	cfg := LoadMCPServerConfigs("")[name]
	if !oauthCanRefresh(cfg.OAuth) {
		cfg = sessCfg
	}
	if !oauthCanRefresh(cfg.OAuth) {
		err := fmt.Errorf("sign-in expired")
		m.noteOAuthFailure(name, rejectedAccess, err.Error())
		return err
	}
	now := time.Now()
	if rejectedAccess != "" && cfg.OAuth.AccessToken != "" && cfg.OAuth.AccessToken != rejectedAccess && !oauthAccessDue(cfg.OAuth, now) {
		m.reconnectIfStale(ctx, name, rejectedAccess, cfg)
		return nil
	}
	if !force && !oauthAccessDue(cfg.OAuth, now) {
		return nil
	}
	prepared, err := prepareMCPConnectionMode(ctx, name, cfg, true)
	if err != nil {
		m.noteOAuthFailure(name, rejectedAccess, err.Error())
		return err
	}
	m.ReconnectOne(ctx, name, prepared)
	return nil
}

func (m *MCPManager) reconnectIfStale(ctx context.Context, name, rejectedAccess string, cfg MCPServerConfig) {
	currentCfg, client, ok := m.sessionSnapshot(name)
	current := ""
	if ok && currentCfg.OAuth != nil {
		current = currentCfg.OAuth.AccessToken
	}
	if !ok || client == nil || current == "" || current == rejectedAccess {
		m.ReconnectOne(ctx, name, cfg)
	}
}

func (m *MCPManager) noteOAuthFailure(name, rejectedAccess, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[name]
	if s == nil {
		return
	}
	if rejectedAccess != "" && s.cfg.OAuth != nil && s.cfg.OAuth.AccessToken != "" && s.cfg.OAuth.AccessToken != rejectedAccess {
		return
	}
	s.status = "error"
	s.err = msg
}

func (m *MCPManager) callStored(ctx context.Context, exposedName, argumentsJSON string) (display, kind string, value any, ok bool) {
	if m == nil {
		return "mcp: not configured", "", nil, false
	}
	m.mu.RLock()
	route, found := m.routes[exposedName]
	var sess *mcpSession
	if found {
		sess = m.sessions[route.Server]
	}
	m.mu.RUnlock()
	if !found || sess == nil || sess.client == nil {
		return "mcp: unknown tool " + exposedName, "", nil, false
	}
	serverName := route.Server
	if sess.cfg.OAuth != nil && oauthAccessDue(sess.cfg.OAuth, time.Now()) {
		rejected := sess.cfg.OAuth.AccessToken
		if err := m.renewOAuth(ctx, serverName, rejected, false); err != nil {
			return "mcp error: " + err.Error(), "", nil, false
		}
		if _, client, okSess := m.sessionSnapshot(serverName); !okSess || client == nil {
			return "mcp error: sign-in expired", "", nil, false
		}
	}
	var args map[string]any
	if strings.TrimSpace(argumentsJSON) != "" {
		if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
			return "mcp: invalid arguments: " + err.Error(), "", nil, false
		}
	}
	req := mcp.CallToolRequest{}
	req.Params.Name = route.Tool
	req.Params.Arguments = args
	call := func() (*mcp.CallToolResult, error) {
		_, client, okSess := m.sessionSnapshot(serverName)
		if !okSess || client == nil {
			return nil, fmt.Errorf("sign-in expired")
		}
		cctx, cancel := context.WithTimeout(ctx, mcpCallTimeout)
		defer cancel()
		return client.CallTool(cctx, req)
	}
	result, err := call()
	if mcpAuthRejected(err) {
		rejected := ""
		if cfg, _, okSess := m.sessionSnapshot(serverName); okSess && cfg.OAuth != nil {
			rejected = cfg.OAuth.AccessToken
		}
		if rerr := m.renewOAuth(ctx, serverName, rejected, true); rerr != nil {
			return "mcp error: " + rerr.Error(), "", nil, false
		}
		result, err = call()
	}
	if err != nil {
		return "mcp error: " + err.Error(), "", nil, false
	}
	return mcpCallOutcome(result)
}

func formatMCPResult(result *mcp.CallToolResult) string {
	if result == nil {
		return "mcp: empty result"
	}
	var parts []string
	for _, c := range result.Content {
		parts = append(parts, mcp.GetTextFromContent(c))
	}
	text := strings.TrimSpace(strings.Join(parts, "\n"))
	if result.StructuredContent != nil {
		if b, err := json.MarshalIndent(result.StructuredContent, "", "  "); err == nil {
			if text != "" {
				text += "\n"
			}
			text += string(b)
		}
	}
	if text == "" {
		text = "(no content)"
	}
	if result.IsError {
		return "mcp tool error: " + text
	}
	return text
}

func (a *Agent) ensureMCP() *MCPManager {
	if a.MCP == nil {
		a.MCP = NewMCPManager()
	}
	return a.MCP
}

func (a *Agent) mcpToolDefs() []map[string]any {
	if a == nil || a.MCP == nil {
		return nil
	}
	return a.MCP.defsNamed(a.mcpLoadedNames())
}

func (m *MCPManager) setServerSources(sources, names map[string]string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.sources = sources
	m.sourceNames = names
	m.mu.Unlock()
}

func (a *Agent) reloadMCP(ctx context.Context) {
	if a == nil {
		return
	}
	configs, sources, names := loadMCPServers(a.Cfg.ConfigPath)
	m := a.ensureMCP()
	m.setServerSources(sources, names)
	m.setStartScript(readMCPStart(a.Cfg.ConfigPath))
	m.Reload(ctx, configs)
}

// startMCP connects the servers when a worker opens. The start hook is process
// wide, so opening another workspace reuses a start that already succeeded.
func (a *Agent) startMCP(ctx context.Context) {
	if a == nil {
		return
	}
	configs, sources, names := loadMCPServers(a.Cfg.ConfigPath)
	m := a.ensureMCP()
	m.setServerSources(sources, names)
	m.setStartScript(readMCPStart(a.Cfg.ConfigPath))
	m.beginStart()
	defer m.finishStart()
	m.Close()
	m.ConnectAll(ctx, configs)
}

// beginStart and finishStart bracket a connect so a run can wait for it.
func (m *MCPManager) beginStart() {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.startDone == nil {
		m.startDone = make(chan struct{})
	}
	m.mu.Unlock()
}

func (m *MCPManager) finishStart() {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.startDone != nil {
		close(m.startDone)
		m.startDone = nil
	}
	m.mu.Unlock()
}

// waitStart blocks while a connect runs, at most timeout. A caller that finds
// no connect in progress returns at once.
func (m *MCPManager) waitStart(timeout time.Duration) {
	if m == nil {
		return
	}
	m.mu.RLock()
	done := m.startDone
	m.mu.RUnlock()
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

// reconnectMCP replaces one server. A sign-in for that server does not
// restart the others.
func (a *Agent) reconnectMCP(ctx context.Context, name string) {
	if a == nil || strings.TrimSpace(name) == "" {
		return
	}
	configs, sources, names := loadMCPServers(a.Cfg.ConfigPath)
	m := a.ensureMCP()
	m.setServerSources(sources, names)
	m.setStartScript(readMCPStart(a.Cfg.ConfigPath))
	cfg, ok := configs[name]
	if !ok {
		return
	}
	m.ReconnectOne(ctx, name, cfg)
}

// refreshMCPTokens renews access tokens that are about to lapse and
// reconnects only those servers. A still-valid refresh token is the grant.
func (a *Agent) refreshMCPTokens(ctx context.Context) {
	if a == nil {
		return
	}
	configs, _, _ := loadMCPServers(a.Cfg.ConfigPath)
	now := time.Now()
	for name, cfg := range configs {
		if cfg.OAuth == nil || cfg.OAuth.RefreshToken == "" || cfg.OAuth.TokenURL == "" {
			continue
		}
		if cfg.OAuth.AccessToken != "" && (cfg.OAuth.ExpiresAt.IsZero() || now.Before(cfg.OAuth.ExpiresAt.Add(-time.Minute))) {
			continue
		}
		prepared, err := prepareMCPConnection(ctx, name, cfg)
		if err != nil {
			continue
		}
		a.ensureMCP().ReconnectOne(ctx, name, prepared)
	}
}
