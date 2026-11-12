package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// mcpParallelLimit is how many MCP calls may talk to servers at once.
// Zero and negative values use the starting cap of 10.
func mcpParallelLimit(n int) int {
	if n < 1 {
		return 10
	}
	return n
}

// mcpRunGate is the one scheduler for chat MCP calls and DataLang mcp().
// A slot is a call that is talking to a server. Waiting does not take a slot.
// A stdio server keeps its key before it takes a slot, so a queued call to a
// busy stdio server does not fill the cap.
type mcpRunGate struct {
	mu       sync.Mutex
	cond     *sync.Cond
	inflight int
	stdio    map[string]bool
	limit    func() int
}

func newMCPRunGate(limit func() int) *mcpRunGate {
	g := &mcpRunGate{stdio: map[string]bool{}, limit: limit}
	g.cond = sync.NewCond(&g.mu)
	return g
}

func (a *Agent) mcpGate() *mcpRunGate {
	a.mcpGateOnce.Do(func() {
		a.mcpSlots = newMCPRunGate(func() int { return mcpParallelLimit(a.Cfg.MCPParallel) })
	})
	return a.mcpSlots
}

func (g *mcpRunGate) acquire(ctx context.Context, server string, stdio bool) error {
	if g == nil {
		return nil
	}
	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-ctx.Done():
			g.mu.Lock()
			g.cond.Broadcast()
			g.mu.Unlock()
		case <-stopped:
		}
	}()
	g.mu.Lock()
	defer g.mu.Unlock()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		limit := 10
		if g.limit != nil {
			limit = mcpParallelLimit(g.limit())
		}
		busy := stdio && server != "" && g.stdio[server]
		if g.inflight < limit && !busy {
			g.inflight++
			if stdio && server != "" {
				g.stdio[server] = true
			}
			return nil
		}
		g.cond.Wait()
	}
}

func (g *mcpRunGate) release(server string, stdio bool) {
	if g == nil {
		return
	}
	g.mu.Lock()
	if g.inflight > 0 {
		g.inflight--
	}
	if stdio && server != "" {
		delete(g.stdio, server)
	}
	g.cond.Broadcast()
	g.mu.Unlock()
}

type mcpDone struct {
	display string
	kind    string
	value   any
	stored  bool
}

// callMCPTool takes a slot, calls the server, and redacts a credential.
// It does not record history. The caller does that on one goroutine.
func (a *Agent) callMCPTool(ctx context.Context, name, args string) mcpDone {
	secret := ""
	if a.Credentials != nil {
		prepared, sec, err := a.Credentials.prepareToolArgs(name, args)
		if err != nil {
			return mcpDone{display: "credential error: " + err.Error()}
		}
		args = prepared
		secret = sec
	}
	server, transport := "", ""
	if a.MCP != nil {
		if s, t, ok := a.MCP.callRoute(name); ok {
			server, transport = s, t
		}
	}
	if server == "" {
		if s, _, ok := parseMCPToolName(name); ok {
			server = s
		}
	}
	stdio := transport == "stdio"
	if err := a.mcpGate().acquire(ctx, server, stdio); err != nil {
		return mcpDone{display: "INTERRUPTED: " + err.Error()}
	}
	defer a.mcpGate().release(server, stdio)
	var done mcpDone
	if a.mcpInvoke != nil {
		done.display, done.kind, done.value, done.stored = a.mcpInvoke(ctx, name, args)
	} else {
		done.display, done.kind, done.value, done.stored = a.ensureMCP().callStored(ctx, name, args)
	}
	done.display = redactSecret(done.display, secret)
	if done.stored && secret != "" {
		if b, err := json.Marshal(done.value); err == nil {
			var redacted any
			if json.Unmarshal([]byte(redactSecret(string(b), secret)), &redacted) == nil {
				done.value = redacted
			}
		}
	}
	return done
}

func (m *MCPManager) callRoute(exposed string) (server, transport string, ok bool) {
	if m == nil {
		return "", "", false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	route, found := m.routes[exposed]
	if !found {
		return "", "", false
	}
	sess := m.sessions[route.Server]
	if sess == nil {
		return route.Server, "", true
	}
	return route.Server, sess.cfg.transportKind(), true
}

// allowMCPProgram decides the program once, before any of its calls run.
// Each literal tool is judged the same way as a single call. When every one
// may run, the program starts. When any would ask or need review, the program
// is that one question. A refusal runs nothing.
// runMCPWave approves each MCP call, then runs the allowed ones through the
// shared scheduler. Results are recorded after the wave, in call order, so the
// next model step sees every result. Stop cancels a call that is waiting or running.
func (a *Agent) runMCPWave(ctx context.Context, reader *bufio.Reader, items []Item) error {
	admits := make([]toolAdmit, len(items))
	for i, item := range items {
		adm, err := a.beginToolCall(ctx, reader, item)
		if err != nil {
			return err
		}
		admits[i] = adm
	}
	type held struct {
		pre  string
		done mcpDone
	}
	rows := make([]held, len(items))
	var wg sync.WaitGroup
	for i, item := range items {
		if admits[i].Finished || !admits[i].Allowed {
			continue
		}
		rows[i].pre = runPluginHooks("preToolUse", item.Name+"\n"+item.Arguments)
		wg.Add(1)
		go func(i int, item Item) {
			defer wg.Done()
			rows[i].done = a.callMCPTool(ctx, item.Name, item.Arguments)
		}(i, item)
	}
	wg.Wait()
	for i, item := range items {
		if admits[i].Finished {
			continue
		}
		result := "DENIED by user"
		if admits[i].Deny {
			result = "DENIED: " + admits[i].Reason
		}
		if admits[i].Allowed {
			done := rows[i].done
			if done.stored {
				a.toolDataPending = &dataPending{kind: done.kind, value: done.value}
			} else {
				a.toolDataPending = nil
			}
			result = done.display
			post := runPluginHooks("postToolUse", item.Name+"\n"+result)
			if extra := strings.TrimSpace(rows[i].pre + "\n" + post); extra != "" {
				result = strings.TrimSpace(result + "\n" + extra)
			}
			a.println("\033[35m⬡ MCP\033[0m " + truncate(result, 240))
		}
		if err := a.recordToolResult(ctx, item, result, "", admits[i].Allowed); err != nil {
			return err
		}
	}
	return nil
}

func (a *Agent) allowMCPProgram(ctx context.Context, reader *bufio.Reader, names []string) (bool, error) {
	if len(names) == 0 {
		return true, nil
	}
	agentRun := strings.TrimSpace(a.Cfg.AgentInstanceID) != ""
	ask := false
	review := false
	var listed []string
	for _, name := range names {
		g := gateToolCall(agentRun, a.Cfg.Mode, name, "{}", mcpToolAlwaysAllowed(a.Cfg.ConfigPath, name))
		if g.Deny {
			why := g.Reason
			if why == "" {
				why = "auto mode refused this call"
			}
			a.println("Approval refused:", truncate(why, 240))
			return false, nil
		}
		if g.Allow && !g.Ask && !g.Review {
			continue
		}
		listed = append(listed, name)
		if g.Ask {
			ask = true
		}
		if g.Review {
			review = true
		}
	}
	if len(listed) == 0 {
		return true, nil
	}
	reason := "MCP program calls " + strings.Join(listed, ", ")
	if review && !ask && a.Client != nil {
		cwd, _ := os.Getwd()
		purpose := ""
		if agentRun {
			purpose = a.Cfg.AgentInstructions
		}
		raw, _ := json.Marshal(map[string]any{"tools": listed})
		verdict, e := a.Client.Review(ctx, a.Cfg.ReviewModel, a.Cfg.ApprovalPolicy, "data_processing", string(raw), cwd, purpose, nil, a.Cfg.ApprovalPolicyReplace)
		g := settleReview(agentRun, verdict.Decision, verdict.Reason, e)
		if g.Allow {
			return true, nil
		}
		if g.Deny {
			why := g.Reason
			if why == "" {
				why = "auto mode refused this call"
			}
			a.println("Approval refused:", truncate(why, 240))
			return false, nil
		}
		if g.Reason != "" {
			reason = g.Reason
		}
	}
	a.println("Approval required:", truncate(reason, 240))
	return a.waitApproval(ctx, reader, "data_processing", reason, reason, "")
}

// waitApproval is the one question for a tool or for a whole MCP program.
func (a *Agent) waitApproval(ctx context.Context, reader *bufio.Reader, name, command, reason, preview string) (bool, error) {
	if preview != "" {
		a.print(preview)
	}
	if a.ApprovalCh != nil {
		a.emitJSON("approval", map[string]any{
			"name":    name,
			"command": command,
			"reason":  reason,
			"preview": preview,
		})
		a.emit("status", "awaiting_approval")
		var allowed bool
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case allowed = <-a.ApprovalCh:
		}
		a.emitJSON("approval", map[string]any{"clear": true})
		a.emit("status", "running")
		return allowed, nil
	}
	if a.Control != nil && isTerminal(0) {
		return a.Control.requestApproval(ctx)
	}
	a.print("Approve? [Y/n] ")
	if reader == nil {
		return false, fmt.Errorf("approval requires a terminal reader")
	}
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "" || answer == "y" || answer == "yes" || answer == "j" || answer == "ja", nil
}
