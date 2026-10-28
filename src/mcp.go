package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Status    string   `json:"status"` // connected | error | disabled
	Error     string   `json:"error,omitempty"`
	Tools     []string `json:"tools,omitempty"`
	Command   string   `json:"command,omitempty"`
	URL       string   `json:"url,omitempty"`
	Disabled  bool     `json:"disabled,omitempty"`
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
	mu       sync.RWMutex
	sessions map[string]*mcpSession
	routes   map[string]mcpToolRoute
	defs     []map[string]any
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
	cctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cli, err := openMCPClient(cfg)
	if err != nil {
		sess.err = err.Error()
		m.mu.Lock()
		m.sessions[name] = sess
		m.mu.Unlock()
		return
	}
	if err := cli.Start(cctx); err != nil {
		_ = cli.Close()
		sess.err = "start: " + err.Error()
		m.mu.Lock()
		m.sessions[name] = sess
		m.mu.Unlock()
		return
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "keimling", Version: "1.0.0"}
	if _, err := cli.Initialize(cctx, initReq); err != nil {
		_ = cli.Close()
		sess.err = "initialize: " + err.Error()
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

func openMCPClient(cfg MCPServerConfig) (*mcpclient.Client, error) {
	switch cfg.transportKind() {
	case "stdio":
		env := make([]string, 0, len(cfg.Env))
		for k, v := range cfg.Env {
			env = append(env, k+"="+v)
		}
		sort.Strings(env)
		return mcpclient.NewStdioMCPClient(cfg.Command, env, cfg.Args...)
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
	params := schemaToParams(tool.InputSchema)
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
			Name:      name,
			Transport: s.cfg.transportKind(),
			Status:    s.status,
			Error:     s.err,
			Command:   s.cfg.Command,
			URL:       s.cfg.URL,
			Disabled:  s.cfg.Disabled,
		}
		for _, t := range s.tools {
			st.Tools = append(st.Tools, mcpExposedName(name, t.Name))
		}
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

func (a *Agent) reloadMCP(ctx context.Context) {
	if a == nil {
		return
	}
	a.ensureMCP().Reload(ctx, LoadMCPServerConfigs(a.Cfg.ConfigPath))
}
