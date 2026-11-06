package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	URL        string   `json:"url,omitempty"`
	Disabled   bool     `json:"disabled,omitempty"`
	Grant      string   `json:"grant,omitempty"`
	Auth       string   `json:"auth,omitempty"`
	OAuth      string   `json:"oauth,omitempty"`
	Source     string   `json:"source,omitempty"`      // user, config, project, cursor, plugin
	SourceName string   `json:"source_name,omitempty"` // plugin name when Source is plugin
}

type mcpSession struct {
	name   string
	cfg    MCPServerConfig
	client *mcpclient.Client
	tools  []mcp.Tool
	status string
	err    string
}

// MCPManager holds backend MCP clients (stdio/HTTP/SSE). The browser never talks to MCP servers.
type MCPManager struct {
	mu          sync.RWMutex
	sessions    map[string]*mcpSession
	routes      map[string]mcpToolRoute
	defs        []map[string]any
	sources     map[string]string
	sourceNames map[string]string
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
	m.ConnectAll(ctx, configs)
}

func (m *MCPManager) ConnectAll(ctx context.Context, configs map[string]MCPServerConfig) {
	if m == nil {
		return
	}
	names := make([]string, 0, len(configs))
	for name := range configs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		m.connectOne(ctx, name, configs[name])
	}
	m.rebuildIndex()
}

func (m *MCPManager) connectOne(ctx context.Context, name string, cfg MCPServerConfig) {
	sess := &mcpSession{name: name, cfg: cfg, status: "error"}
	if cfg.Disabled {
		sess.status = "disabled"
		m.mu.Lock()
		m.sessions[name] = sess
		m.mu.Unlock()
		return
	}
	if err := cfg.validate(); err != nil {
		sess.err = err.Error()
		m.mu.Lock()
		m.sessions[name] = sess
		m.mu.Unlock()
		return
	}
	cfg, err := prepareMCPConnection(ctx, name, cfg)
	if err != nil {
		sess.err = err.Error()
		sess.cfg = cfg
		m.mu.Lock()
		m.sessions[name] = sess
		m.mu.Unlock()
		return
	}
	sess.cfg = cfg
	cctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cli, kind, err := dialMCP(cctx, cfg)
	if kind != "" {
		sess.cfg.Type = kind
	}
	if err != nil {
		sess.err = err.Error()
		m.mu.Lock()
		m.sessions[name] = sess
		m.mu.Unlock()
		return
	}
	listed, err := cli.ListTools(cctx, mcp.ListToolsRequest{})
	if err != nil {
		_ = cli.Close()
		sess.err = "list tools: " + err.Error()
		m.mu.Lock()
		m.sessions[name] = sess
		m.mu.Unlock()
		return
	}
	sess.client = cli
	sess.status = "connected"
	if listed != nil {
		sess.tools = listed.Tools
	}
	m.mu.Lock()
	m.sessions[name] = sess
	m.mu.Unlock()
}

func dialMCP(ctx context.Context, cfg MCPServerConfig) (*mcpclient.Client, string, error) {
	kind := cfg.transportKind()
	if kind == "streamableHttp" && !cfg.remoteTypeSet() {
		cli, err := connectMCP(ctx, cfg)
		if err == nil {
			return cli, "streamableHttp", nil
		}
		if legacySSEServer(err) {
			sse := cfg
			sse.Type = "sse"
			cli, err2 := connectMCP(ctx, sse)
			if err2 == nil {
				return cli, "sse", nil
			}
			return nil, "sse", err2
		}
		return nil, "streamableHttp", err
	}
	cli, err := connectMCP(ctx, cfg)
	return cli, kind, err
}

func legacySSEServer(err error) bool {
	return err != nil && (errors.Is(err, transport.ErrLegacySSEServer) || strings.Contains(err.Error(), "legacy SSE"))
}

func connectMCP(ctx context.Context, cfg MCPServerConfig) (*mcpclient.Client, error) {
	cli, err := openMCPClient(cfg)
	if err != nil {
		return nil, err
	}
	if err := cli.Start(ctx); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("start: %w", err)
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "keimling", Version: "1.0.0"}
	if _, err := cli.Initialize(ctx, initReq); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}
	return cli, nil
}

func openMCPClient(cfg MCPServerConfig) (*mcpclient.Client, error) {
	switch cfg.transportKind() {
	case "stdio":
		env := make([]string, 0, len(cfg.Env))
		for k, v := range cfg.Env {
			env = append(env, k+"="+v)
		}
		sort.Strings(env)
		cwd := strings.TrimSpace(cfg.Cwd)
		if cwd == "" {
			return mcpclient.NewStdioMCPClient(cfg.Command, env, cfg.Args...)
		}
		return mcpclient.NewStdioMCPClientWithOptions(cfg.Command, env, cfg.Args, transport.WithCommandFunc(func(ctx context.Context, command string, env []string, args []string) (*exec.Cmd, error) {
			cmd := exec.CommandContext(ctx, command, args...)
			cmd.Env = append(os.Environ(), env...)
			cmd.Dir = cwd
			return cmd, nil
		}))
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
			URL:        s.cfg.URL,
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

func (m *MCPManager) Call(ctx context.Context, exposedName, argumentsJSON string) string {
	if m == nil {
		return "mcp: not configured"
	}
	m.mu.RLock()
	route, ok := m.routes[exposedName]
	var sess *mcpSession
	if ok {
		sess = m.sessions[route.Server]
	}
	m.mu.RUnlock()
	if !ok || sess == nil || sess.client == nil {
		return "mcp: unknown tool " + exposedName
	}
	var args map[string]any
	if strings.TrimSpace(argumentsJSON) != "" {
		if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
			return "mcp: invalid arguments: " + err.Error()
		}
	}
	req := mcp.CallToolRequest{}
	req.Params.Name = route.Tool
	req.Params.Arguments = args
	cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	result, err := sess.client.CallTool(cctx, req)
	if err != nil {
		return "mcp error: " + err.Error()
	}
	return formatMCPResult(result)
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
	return a.MCP.ToolDefs()
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
	m.Reload(ctx, configs)
}
