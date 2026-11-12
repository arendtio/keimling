package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func mcpTestManager(kinds map[string]string, tools ...string) *MCPManager {
	if len(tools) == 0 {
		tools = []string{"ping", "pong"}
	}
	m := NewMCPManager()
	for name, kind := range kinds {
		cfg := MCPServerConfig{Type: kind}
		if kind == "stdio" {
			cfg.Command = "true"
		} else {
			cfg.URL = "http://127.0.0.1"
		}
		m.sessions[name] = &mcpSession{
			name: name, status: "connected", cfg: cfg, tools: mcpToolsNamed(tools...),
		}
	}
	m.rebuildIndex()
	return m
}

func waitSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for an MCP call")
	}
}

func TestMCPParallelLimitDefaultsToTen(t *testing.T) {
	if mcpParallelLimit(0) != 10 || mcpParallelLimit(-3) != 10 || mcpParallelLimit(4) != 4 {
		t.Fatalf("limits %d %d %d", mcpParallelLimit(0), mcpParallelLimit(-3), mcpParallelLimit(4))
	}
}

func TestHTTPCallsOnOneServerOverlap(t *testing.T) {
	a := &Agent{Cfg: Config{Mode: "yolo", MCPParallel: 10}, MCP: mcpTestManager(map[string]string{"mail": "http"})}
	entered := map[string]chan struct{}{"mcp__mail__ping": make(chan struct{}), "mcp__mail__pong": make(chan struct{})}
	release := make(chan struct{})
	a.mcpInvoke = func(ctx context.Context, name, args string) (string, string, any, bool) {
		close(entered[name])
		<-release
		return "ok", "text", "ok", true
	}
	var wg sync.WaitGroup
	for _, name := range []string{"mcp__mail__ping", "mcp__mail__pong"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			a.callMCPTool(context.Background(), name, "{}")
		}(name)
	}
	waitSignal(t, entered["mcp__mail__ping"])
	waitSignal(t, entered["mcp__mail__pong"])
	close(release)
	wg.Wait()
}

func TestStdioWaiterDoesNotTakeASlot(t *testing.T) {
	a := &Agent{Cfg: Config{Mode: "yolo", MCPParallel: 2}, MCP: mcpTestManager(map[string]string{"a": "stdio", "b": "http"})}
	entered := map[string]chan struct{}{
		"mcp__a__ping": make(chan struct{}),
		"mcp__a__pong": make(chan struct{}),
		"mcp__b__ping": make(chan struct{}),
	}
	release := map[string]chan struct{}{
		"mcp__a__ping": make(chan struct{}),
		"mcp__a__pong": make(chan struct{}),
		"mcp__b__ping": make(chan struct{}),
	}
	a.mcpInvoke = func(ctx context.Context, name, args string) (string, string, any, bool) {
		close(entered[name])
		<-release[name]
		return "ok", "text", "ok", true
	}
	go a.callMCPTool(context.Background(), "mcp__a__ping", "{}")
	waitSignal(t, entered["mcp__a__ping"])
	go a.callMCPTool(context.Background(), "mcp__a__pong", "{}")
	go a.callMCPTool(context.Background(), "mcp__b__ping", "{}")
	waitSignal(t, entered["mcp__b__ping"])
	select {
	case <-entered["mcp__a__pong"]:
		t.Fatal("second stdio call started while the first still held the server")
	case <-time.After(50 * time.Millisecond):
	}
	a.mcpSlots.mu.Lock()
	inflight := a.mcpSlots.inflight
	a.mcpSlots.mu.Unlock()
	if inflight != 2 {
		t.Fatalf("inflight %d, want the running stdio call and the other server", inflight)
	}
	close(release["mcp__a__ping"])
	waitSignal(t, entered["mcp__a__pong"])
	close(release["mcp__a__pong"])
	close(release["mcp__b__ping"])
}

func TestMCPCapHoldsFurtherCalls(t *testing.T) {
	a := &Agent{Cfg: Config{Mode: "yolo", MCPParallel: 1}, MCP: mcpTestManager(map[string]string{"a": "http", "b": "http"})}
	entered := map[string]chan struct{}{"mcp__a__ping": make(chan struct{}), "mcp__b__ping": make(chan struct{})}
	release := make(chan struct{})
	a.mcpInvoke = func(ctx context.Context, name, args string) (string, string, any, bool) {
		close(entered[name])
		<-release
		return "ok", "text", "ok", true
	}
	go a.callMCPTool(context.Background(), "mcp__a__ping", "{}")
	waitSignal(t, entered["mcp__a__ping"])
	go a.callMCPTool(context.Background(), "mcp__b__ping", "{}")
	select {
	case <-entered["mcp__b__ping"]:
		t.Fatal("second call started at a cap of 1")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	waitSignal(t, entered["mcp__b__ping"])
}

func TestStopCancelsAWaitingMCPCall(t *testing.T) {
	a := &Agent{Cfg: Config{Mode: "yolo", MCPParallel: 1}, MCP: mcpTestManager(map[string]string{"a": "http"})}
	entered := make(chan struct{})
	release := make(chan struct{})
	a.mcpInvoke = func(ctx context.Context, name, args string) (string, string, any, bool) {
		close(entered)
		<-release
		return "ok", "text", "ok", true
	}
	go a.callMCPTool(context.Background(), "mcp__a__ping", "{}")
	waitSignal(t, entered)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan mcpDone, 1)
	go func() {
		done <- a.callMCPTool(ctx, "mcp__a__pong", "{}")
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case got := <-done:
		if !strings.HasPrefix(got.display, "INTERRUPTED:") {
			t.Fatalf("waiter %q", got.display)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancel did not release the waiting call")
	}
	close(release)
}

func TestChatMCPWaveOverlapsAndKeepsADenial(t *testing.T) {
	a := parallelAgent(t)
	a.Cfg.MCPParallel = 10
	a.MCP = mcpTestManager(map[string]string{"a": "http", "b": "http"})
	a.Cfg.Mode = "ask"
	a.ApprovalCh = make(chan bool, 2)
	a.ApprovalCh <- false
	a.ApprovalCh <- true
	entered := make(chan string, 2)
	release := make(chan struct{})
	a.mcpInvoke = func(ctx context.Context, name, args string) (string, string, any, bool) {
		entered <- name
		<-release
		return name, "text", name, true
	}
	items := []Item{
		{Type: "function_call", Name: "mcp__a__ping", CallID: "c1", Arguments: "{}"},
		{Type: "function_call", Name: "mcp__b__ping", CallID: "c2", Arguments: "{}"},
	}
	done := make(chan error, 1)
	go func() { done <- a.runFunctionCalls(context.Background(), "", nil, items) }()
	select {
	case name := <-entered:
		if name != "mcp__b__ping" {
			t.Fatalf("denied call ran: %s", name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("allowed call did not start")
	}
	select {
	case name := <-entered:
		t.Fatalf("extra call %s", name)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var denied, allowed bool
	for _, raw := range a.History {
		var item map[string]any
		if json.Unmarshal(raw, &item) != nil || item["type"] != "function_call_output" {
			continue
		}
		out, _ := item["output"].(string)
		switch item["call_id"] {
		case "c1":
			denied = strings.HasPrefix(out, "DENIED")
		case "c2":
			allowed = strings.Contains(out, "mcp__b__ping")
		}
	}
	if !denied || !allowed {
		t.Fatalf("history denied=%v allowed=%v", denied, allowed)
	}
}

func TestDataLangMCPProgram(t *testing.T) {
	code := func(body string) string {
		return `{"input_ref":"","inputs":[],"code":` + jsonString(body) + `}`
	}
	t.Run("runs after one approval", func(t *testing.T) {
		a := &Agent{Cfg: Config{Mode: "ask", MCPParallel: 10}, MCP: mcpTestManager(map[string]string{"mail": "http"})}
		var calls int
		a.mcpInvoke = func(ctx context.Context, name, args string) (string, string, any, bool) {
			calls++
			if name != "mcp__mail__ping" {
				t.Errorf("name %s", name)
			}
			return "ok", "text", "pong", true
		}
		a.ApprovalCh = make(chan bool, 1)
		started := make(chan struct{})
		go func() {
			close(started)
			time.Sleep(40 * time.Millisecond)
			if calls != 0 {
				t.Errorf("call ran before approval: %d", calls)
			}
			a.ApprovalCh <- true
		}()
		<-started
		out := a.runDataProgram(context.Background(), nil, code(`return mcp("mcp__mail__ping", {})`))
		if calls != 1 || !strings.Contains(out, "pong") {
			t.Fatalf("calls %d out %s", calls, out)
		}
	})
	t.Run("deny runs none", func(t *testing.T) {
		a := &Agent{Cfg: Config{Mode: "ask"}, MCP: mcpTestManager(map[string]string{"mail": "http"})}
		calls := 0
		a.mcpInvoke = func(ctx context.Context, name, args string) (string, string, any, bool) {
			calls++
			return "ok", "text", "pong", true
		}
		a.ApprovalCh = make(chan bool, 1)
		a.ApprovalCh <- false
		out := a.runDataProgram(context.Background(), nil, code(`return all(mcp("mcp__mail__ping", {}), mcp("mcp__mail__pong", {}))`))
		if calls != 0 || !strings.HasPrefix(out, "DENIED") {
			t.Fatalf("calls %d out %s", calls, out)
		}
	})
	t.Run("no question when every tool is allowed", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.json")
		if err := os.WriteFile(filepath.Join(dir, "mcp.json"), []byte(`{"mcpServers":{"mail":{"command":"true","allow_tools":["mcp__mail__ping"]}}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		a := &Agent{Cfg: Config{Mode: "ask", ConfigPath: cfgPath}, MCP: mcpTestManager(map[string]string{"mail": "http"})}
		a.mcpInvoke = func(ctx context.Context, name, args string) (string, string, any, bool) {
			return "ok", "text", "pong", true
		}
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		out := a.runDataProgram(ctx, nil, code(`return mcp("mcp__mail__ping", {})`))
		if strings.Contains(out, "deadline") || !strings.Contains(out, "pong") {
			t.Fatalf("out %s", out)
		}
	})
	t.Run("asks once when one tool would ask", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.json")
		if err := os.WriteFile(filepath.Join(dir, "mcp.json"), []byte(`{"mcpServers":{"mail":{"command":"true","allow_tools":["mcp__mail__ping"]}}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		a := &Agent{Cfg: Config{Mode: "ask", ConfigPath: cfgPath, MCPParallel: 10}, MCP: mcpTestManager(map[string]string{"mail": "http", "cal": "http"})}
		var mu sync.Mutex
		var got []string
		a.mcpInvoke = func(ctx context.Context, name, args string) (string, string, any, bool) {
			mu.Lock()
			got = append(got, name)
			mu.Unlock()
			return "ok", "text", name, true
		}
		a.ApprovalCh = make(chan bool, 1)
		go func() {
			time.Sleep(40 * time.Millisecond)
			mu.Lock()
			n := len(got)
			mu.Unlock()
			if n != 0 {
				t.Errorf("ran before approval: %d", n)
			}
			a.ApprovalCh <- true
		}()
		out := a.runDataProgram(context.Background(), nil, code(`return all(mcp("mcp__mail__ping", {}), mcp("mcp__cal__ping", {}))`))
		if !strings.Contains(out, "mcp__mail__ping") || !strings.Contains(out, "mcp__cal__ping") {
			t.Fatalf("out %s", out)
		}
	})
	t.Run("computed name does not run", func(t *testing.T) {
		a := &Agent{Cfg: Config{Mode: "yolo"}, MCP: mcpTestManager(map[string]string{"mail": "http"})}
		calls := 0
		a.mcpInvoke = func(ctx context.Context, name, args string) (string, string, any, bool) {
			calls++
			return "ok", "text", "pong", true
		}
		out := a.runDataProgram(context.Background(), nil, code("let name = \"mcp__mail__ping\"\nreturn mcp(name, {})"))
		if calls != 0 || !strings.Contains(out, "not part of the approval") {
			t.Fatalf("calls %d out %s", calls, out)
		}
	})
	t.Run("all overlaps and the same stdio server does not", func(t *testing.T) {
		a := &Agent{Cfg: Config{Mode: "yolo", MCPParallel: 10}, MCP: mcpTestManager(map[string]string{"a": "stdio"})}
		entered := make(chan string, 2)
		release := make(chan struct{})
		a.mcpInvoke = func(ctx context.Context, name, args string) (string, string, any, bool) {
			entered <- name
			<-release
			return "ok", "text", name, true
		}
		done := make(chan string, 1)
		go func() {
			done <- a.runDataProgram(context.Background(), nil, code(`return all(mcp("mcp__a__ping", {}), mcp("mcp__a__pong", {}))`))
		}()
		first := ""
		select {
		case first = <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for an MCP call")
		}
		select {
		case second := <-entered:
			t.Fatalf("stdio tools in one program overlapped: %s and %s", first, second)
		case <-time.After(50 * time.Millisecond):
		}
		close(release)
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("second stdio call did not start after the first finished")
		}
		out := <-done
		if !strings.Contains(out, "mcp__a__ping") || !strings.Contains(out, "mcp__a__pong") {
			t.Fatalf("out %s", out)
		}
	})
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestMCPParallelSettingPersists(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	session := filepath.Join(dir, "s.jsonl")
	j, err := OpenJournal(session, false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &Agent{
		Cfg:     Config{Model: "m", Mode: "auto", MaxSteps: 20, ContextLimit: 8000, SessionPath: session, ConfigPath: cfgPath},
		Journal: j,
	}
	h := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a}).Handler()
	get := func() map[string]any {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
		req.Header.Set("Authorization", "Bearer tok")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("get %d %s", rec.Code, rec.Body.String())
		}
		var snap map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
			t.Fatal(err)
		}
		return snap
	}
	if got, _ := get()["mcp_parallel"].(float64); got != 10 {
		t.Fatalf("default %v", get()["mcp_parallel"])
	}
	put := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer tok")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if rec := put(`{"mcp_parallel":0}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("zero accepted: %d %s", rec.Code, rec.Body.String())
	}
	if rec := put(`{"mcp_parallel":4}`); rec.Code != http.StatusOK {
		t.Fatalf("put %d %s", rec.Code, rec.Body.String())
	}
	if a.Cfg.MCPParallel != 4 {
		t.Fatalf("memory %d", a.Cfg.MCPParallel)
	}
	fc, err := readConfiguration(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if fc.MCPParallel != 4 {
		t.Fatalf("file %d", fc.MCPParallel)
	}
	if got, _ := get()["mcp_parallel"].(float64); got != 4 {
		t.Fatalf("snapshot %v", get()["mcp_parallel"])
	}
}
