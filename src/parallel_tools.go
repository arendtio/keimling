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

// toolCallParallel reports whether this call only reads, or is a non-interactive
// exec with no credential placeholder. Those may run beside each other.
// Anything that writes, takes the terminal, or injects a secret runs alone.
func toolCallParallel(name, args string) bool {
	switch name {
	case "read_file", "search_history", "read_history", "web_search", "list_agents", "list_watches", "agent_instructions":
		return true
	case "exec":
		if toolNeedsTerminal(name, args) {
			return false
		}
		var x struct {
			Command string `json:"command"`
		}
		if json.Unmarshal([]byte(args), &x) != nil {
			return false
		}
		return len(parseCredPlaceholders(x.Command)) == 0
	default:
		return false
	}
}

type toolAdmit struct {
	Allowed  bool
	Finished bool
	Deny     bool
	Reason   string
}

// beginToolCall applies steering, the grant, and approval. Finished means the
// tool result is already recorded.
func (a *Agent) beginToolCall(ctx context.Context, reader *bufio.Reader, item Item) (toolAdmit, error) {
	if a.PendingSteering {
		if err := a.add(map[string]any{"type": "function_call_output", "call_id": item.CallID, "output": "SKIPPED: New user steering arrived. Re-evaluate the plan and this action before retrying."}); err != nil {
			return toolAdmit{}, err
		}
		a.println("↻ Action deferred: new steering received")
		return toolAdmit{Finished: true}, nil
	}
	if item.CallID == "" {
		return toolAdmit{}, fmt.Errorf("function call without call_id")
	}
	if !a.instanceToolGranted(item.Name) {
		msg := "not granted to this agent"
		if item.Name != "finish_task" && item.Name != "update_plan" {
			a.presentToolStart(item.Name, item.Arguments)
		}
		if err := a.add(map[string]any{"type": "function_call_output", "call_id": item.CallID, "output": msg, "ok": false}); err != nil {
			return toolAdmit{}, err
		}
		a.presentToolResult(item.Name, toolCommandLabel(item.Name, item.Arguments), msg, "", false, false)
		return toolAdmit{Finished: true}, nil
	}
	if item.Name != "finish_task" && item.Name != "update_plan" {
		a.presentToolStart(item.Name, item.Arguments)
	}
	a.applyDefinitionApproval()
	agentRun := strings.TrimSpace(a.Cfg.AgentInstanceID) != ""
	g := gateToolCall(agentRun, a.Cfg.Mode, item.Name, item.Arguments, mcpToolAlwaysAllowed(a.Cfg.ConfigPath, item.Name))
	reason := g.Reason
	if g.Review {
		cwd, _ := os.Getwd()
		var sus *SoftSuspicion
		if hit, ok := softSuspicion(item.Name, item.Arguments); ok {
			sus = &hit
			a.println("◌ Soft rule", hit.ID, "— reviewer checking for false positive…")
		}
		if agentPurposeClears(agentRun, a.Cfg.Mode, a.Cfg.AgentInstructions, item.Name, item.Arguments) {
			g = toolGate{Allow: true}
		} else {
			purpose := ""
			if agentRun {
				purpose = a.Cfg.AgentInstructions
			}
			verdict, e := a.Client.Review(ctx, a.Cfg.ReviewModel, a.Cfg.ApprovalPolicy, item.Name, item.Arguments, cwd, purpose, sus, a.Cfg.ApprovalPolicyReplace)
			why := verdict.Reason
			if e == nil {
				if sus != nil && why == "" {
					why = sus.Reason
				}
				if why == "" && verdict.Decision != "approve" {
					why = "review asked for approval"
				}
			}
			g = settleReview(agentRun, verdict.Decision, why, e)
		}
		reason = g.Reason
	}
	if g.Ask && reason != "" {
		a.println("Approval required:", truncate(reason, 240))
	}
	if g.Deny {
		if reason == "" {
			reason = "auto mode refused this call"
		}
		a.println("Approval refused:", truncate(reason, 240))
	}
	allowed := g.Allow
	if g.Ask {
		preview := renderApprovalPreview(item.Name, item.Arguments)
		if preview != "" {
			a.print(preview)
		}
		if a.ApprovalCh != nil {
			a.emitJSON("approval", map[string]any{
				"name":    item.Name,
				"command": toolCommandLabel(item.Name, item.Arguments),
				"reason":  reason,
				"preview": preview,
			})
			a.emit("status", "awaiting_approval")
			select {
			case <-ctx.Done():
				return toolAdmit{}, ctx.Err()
			case allowed = <-a.ApprovalCh:
			}
			a.emitJSON("approval", map[string]any{"clear": true})
			a.emit("status", "running")
		} else if a.Control != nil && isTerminal(0) {
			var e error
			allowed, e = a.Control.requestApproval(ctx)
			if e != nil {
				return toolAdmit{}, e
			}
		} else {
			a.print("Approve? [Y/n] ")
			if reader == nil {
				return toolAdmit{}, fmt.Errorf("approval requires a terminal reader")
			}
			answer, e := reader.ReadString('\n')
			if e != nil {
				return toolAdmit{}, e
			}
			answer = strings.ToLower(strings.TrimSpace(answer))
			allowed = answer == "" || answer == "y" || answer == "yes" || answer == "j" || answer == "ja"
		}
	}
	return toolAdmit{Allowed: allowed, Deny: g.Deny, Reason: reason}, nil
}

func (a *Agent) runFunctionCalls(ctx context.Context, message string, reader *bufio.Reader, items []Item) error {
	for i := 0; i < len(items); {
		if !toolCallParallel(items[i].Name, items[i].Arguments) {
			if err := a.handleFunctionCall(ctx, message, reader, items[i]); err != nil {
				return err
			}
			i++
			continue
		}
		j := i + 1
		for j < len(items) && toolCallParallel(items[j].Name, items[j].Arguments) {
			j++
		}
		wave := items[i:j]
		var err error
		if len(wave) == 1 {
			err = a.handleFunctionCall(ctx, message, reader, wave[0])
		} else {
			err = a.runParallelWave(ctx, message, reader, wave)
		}
		if err != nil {
			return err
		}
		i = j
	}
	return nil
}

func (a *Agent) runParallelWave(ctx context.Context, message string, reader *bufio.Reader, items []Item) error {
	admits := make([]toolAdmit, len(items))
	for i, item := range items {
		adm, err := a.beginToolCall(ctx, reader, item)
		if err != nil {
			return err
		}
		admits[i] = adm
	}
	results := make([]string, len(items))
	for i, item := range items {
		if admits[i].Allowed && item.Name == "web_search" {
			a.println("⌕ Searching web…")
		}
	}
	var wg sync.WaitGroup
	for i, item := range items {
		if !admits[i].Allowed || admits[i].Finished {
			continue
		}
		wg.Add(1)
		go func(i int, item Item) {
			defer wg.Done()
			results[i] = a.parallelToolBody(ctx, item.Name, item.Arguments)
		}(i, item)
	}
	wg.Wait()
	for i, item := range items {
		if admits[i].Finished {
			continue
		}
		result := "DENIED by user"
		raw := ""
		if admits[i].Deny {
			result = "DENIED: " + admits[i].Reason
		}
		if admits[i].Allowed {
			result = results[i]
			if item.Name == "exec" {
				raw = result
				result = a.filterExecResult(ctx, message, item.Arguments, result)
				result = redactAllSecrets(result, parseExecSecretsForRedact(a, item.Arguments))
			}
		}
		if err := a.recordToolResult(ctx, item, result, raw, admits[i].Allowed); err != nil {
			return err
		}
	}
	return nil
}

func (a *Agent) parallelToolBody(ctx context.Context, name, args string) string {
	switch name {
	case "read_file":
		return executeTool(ctx, name, args)
	case "exec":
		return executeTool(ctx, "exec", args)
	case "search_history":
		return a.searchHistory(args)
	case "read_history":
		return a.readHistory(args)
	case "list_agents":
		return a.listAgentsTool(args)
	case "list_watches":
		return a.listWatchesTool(args)
	case "agent_instructions":
		return a.runDefineAgentTool(name, args)
	case "web_search":
		var q struct {
			Query string `json:"query"`
		}
		if e := decodeArgs(args, &q); e != nil {
			return "web search error: " + e.Error()
		}
		if !a.Cfg.Web || !a.Cfg.Search.enabled() {
			return "web search is off"
		}
		answer, e := a.searchWeb(ctx, q.Query)
		if e != nil {
			return "web search error: " + e.Error()
		}
		return answer
	default:
		return "unknown tool"
	}
}

func (a *Agent) handleFunctionCall(ctx context.Context, message string, reader *bufio.Reader, item Item) error {
	adm, err := a.beginToolCall(ctx, reader, item)
	if err != nil || adm.Finished {
		return err
	}
	result := "DENIED by user"
	if adm.Deny {
		result = "DENIED: " + adm.Reason
	}
	raw := ""
	if adm.Allowed {
		result, raw = a.invokeTool(ctx, message, item)
	}
	return a.recordToolResult(ctx, item, result, raw, adm.Allowed)
}

func (a *Agent) invokeTool(ctx context.Context, message string, item Item) (result, rawExec string) {
	switch item.Name {
	case "update_plan":
		var p struct {
			Steps []PlanStep `json:"steps"`
		}
		if e := decodeArgs(item.Arguments, &p); e != nil {
			return e.Error(), ""
		} else if len(p.Steps) > 20 {
			return "plan too long", ""
		}
		a.Plan = p.Steps
		a.TaskActive = true
		a.ensureCharter().SyncPlan(p.Steps)
		_ = a.persistCharter()
		a.print(renderPlan(a.Plan))
		return "plan updated", ""
	case "finish_task":
		var f struct {
			Status  string `json:"status"`
			Summary string `json:"summary"`
		}
		if e := decodeArgs(item.Arguments, &f); e != nil {
			return e.Error(), ""
		} else if f.Status != "done" && f.Status != "blocked" {
			return "invalid status", ""
		}
		a.TaskDone = f.Status == "done"
		a.TaskBlocked = f.Status == "blocked"
		a.TaskActive = false
		a.PilotReport = f.Status + ": " + f.Summary
		a.presentFinish(f.Status, f.Summary)
		return "task status recorded: " + f.Status, ""
	case "web_search":
		var args struct {
			Query string `json:"query"`
		}
		if e := decodeArgs(item.Arguments, &args); e != nil {
			return "web search error: " + e.Error(), ""
		} else if !a.Cfg.Web || !a.Cfg.Search.enabled() {
			return "web search is off", ""
		}
		a.println("⌕ Searching web…")
		answer, e := a.searchWeb(ctx, args.Query)
		if e != nil {
			return "web search error: " + e.Error(), ""
		}
		return answer, ""
	case "share_file":
		result = a.shareFile(item.Arguments)
		a.println("\033[35m📎 Share\033[0m " + truncate(result, 240))
		return result, ""
	case "change_workspace":
		return a.changeWorkspace(item.Arguments), ""
	case "create_watch":
		return a.createWatch(item.Arguments), ""
	case "list_watches":
		return a.listWatchesTool(item.Arguments), ""
	case "cancel_watch":
		return a.cancelWatch(item.Arguments), ""
	case "list_agents":
		return a.listAgentsTool(item.Arguments), ""
	case "allow_peer":
		return a.allowPeer(item.Arguments), ""
	case "message_agent":
		return a.messageAgent(item.Arguments), ""
	case "stop_agent":
		return a.stopAgentTool(item.Arguments), ""
	case "delete_agent":
		return a.deleteAgentTool(item.Arguments), ""
	case "list_credentials":
		return a.listCredentialsTool(item.Arguments), ""
	default:
		if defineAgentTool(item.Name) {
			return a.runDefineAgentTool(item.Name, item.Arguments), ""
		}
		if item.Name == "show_artifact" {
			return a.showArtifact(item.Arguments), ""
		}
		if item.Name == "search_history" {
			return a.searchHistory(item.Arguments), ""
		}
		if item.Name == "read_history" {
			return a.readHistory(item.Arguments), ""
		}
		if item.Name == "activate_skill" {
			var args struct {
				Name string `json:"name"`
			}
			if e := decodeArgs(item.Arguments, &args); e != nil {
				return e.Error(), ""
			} else if !a.skillEnabled(args.Name) {
				return "activate_skill: skill " + args.Name + " is turned off", ""
			}
			text, e := a.ensureSkills().Activate(args.Name)
			if e != nil {
				return "activate_skill: " + e.Error(), ""
			}
			a.println("\033[35m◇ Skill\033[0m " + args.Name)
			return text, ""
		}
		if item.Name == "read_skill_file" {
			var args struct {
				Name string `json:"name"`
				Path string `json:"path"`
			}
			if e := decodeArgs(item.Arguments, &args); e != nil {
				return e.Error(), ""
			} else if !a.skillEnabled(args.Name) {
				return "read_skill_file: skill " + args.Name + " is turned off", ""
			}
			text, e := a.ensureSkills().ReadFile(args.Name, args.Path)
			if e != nil {
				return "read_skill_file: " + e.Error(), ""
			}
			return text, ""
		}
		if item.Name == "delegate" {
			var args DelegateArgs
			if e := decodeArgs(item.Arguments, &args); e != nil {
				return "delegate error: " + e.Error(), ""
			}
			label := args.Profile
			if strings.TrimSpace(args.Image) != "" {
				label = "image"
			}
			a.printf("\033[35m◇ Delegate\033[0m · %s · %s\n", label, truncate(args.Task, 160))
			answer, e := a.runDelegate(ctx, args)
			if e != nil {
				return "delegate error: " + e.Error(), ""
			}
			return answer, ""
		}
		if item.Name == "data_processing" {
			return a.runDataTool(item.Arguments), ""
		}
		if item.Name == "data_processing_help" {
			return dataHelp(item.Arguments), ""
		}
		if isMCPTool(item.Name) {
			argsJSON := item.Arguments
			secret := ""
			if a.Credentials != nil {
				prepared, sec, credErr := a.Credentials.prepareToolArgs(item.Name, item.Arguments)
				if credErr != nil {
					return "credential error: " + credErr.Error(), ""
				}
				argsJSON = prepared
				secret = sec
			}
			display, kind, value, stored := a.ensureMCP().callStored(ctx, item.Name, argsJSON)
			display = redactSecret(display, secret)
			if stored && secret != "" {
				if b, err := json.Marshal(value); err == nil {
					var redacted any
					if json.Unmarshal([]byte(redactSecret(string(b), secret)), &redacted) == nil {
						value = redacted
					}
				}
			}
			if stored {
				a.toolDataPending = &dataPending{kind: kind, value: value}
			}
			result = display
			a.println("\033[35m⬡ MCP\033[0m " + truncate(result, 240))
			return result, ""
		}
		run := func() string {
			if item.Name == "exec" {
				return a.runExecWithCredentials(ctx, item.Arguments)
			}
			return executeTool(ctx, item.Name, item.Arguments)
		}
		if a.Control != nil && toolNeedsTerminal(item.Name, item.Arguments) {
			a.Control.takeTerminal()
			result = run()
			a.Control.releaseTerminal()
		} else {
			result = run()
		}
		if item.Name == "exec" {
			rawExec = result
			result = a.filterExecResult(ctx, message, item.Arguments, result)
			result = redactAllSecrets(result, parseExecSecretsForRedact(a, item.Arguments))
		}
		return result, rawExec
	}
}

func (a *Agent) recordToolResult(ctx context.Context, item Item, result, rawExec string, allowed bool) error {
	toolCmd := toolCommandLabel(item.Name, item.Arguments)
	toolOK := markToolOK(item.Name, result, rawExec)
	filteredExec := false
	if item.Name == "exec" {
		code := "?"
		displayedResult := result
		if rawExec != "" {
			displayedResult = rawExec
		}
		if i := strings.Index(displayedResult, "exit_code="); i >= 0 {
			fields := strings.Fields(displayedResult[i+len("exit_code="):])
			if len(fields) > 0 {
				code = fields[0]
			}
		}
		output := strings.TrimPrefix(displayedResult, "exit_code="+code+"\n")
		if !toolOK {
			a.cliWrite(fmt.Sprintf("\033[31mFAIL: Return Value %s\033[0m\n", code))
		}
		if strings.TrimSpace(output) != "" {
			a.cliWrite(output)
			if !strings.HasSuffix(output, "\n") {
				a.cliWrite("\n")
			}
		}
		filteredExec = rawExec != "" && rawExec != result
		if filteredExec {
			modelVisible := truncate(result, 32<<10)
			a.cliWrite(fmt.Sprintf("\033[36m◈ Filtered output → model\033[0m (%d → %d bytes):\n%s\n", len(rawExec), len(modelVisible), modelVisible))
			a.presentToolResult(item.Name, toolCmd, strings.TrimSpace(result), strings.TrimSpace(rawExec), toolOK, true)
		} else {
			a.presentToolResult(item.Name, toolCmd, strings.TrimSpace(output), "", toolOK, false)
		}
		visible := a.attachDataRef(item.Name, result)
		if visible != result {
			result = visible
		}
	} else {
		visible := a.attachDataRef(item.Name, result)
		if visible != result {
			result = visible
		}
		if !toolOK {
			a.cliWrite(fmt.Sprintf("\033[31m✗\033[0m %s\n", truncate(result, 1800)))
		} else {
			a.cliWrite(fmt.Sprintf("\033[32m✓\033[0m %s\n", truncate(result, 1800)))
		}
		a.presentToolResult(item.Name, toolCmd, truncate(result, 1800), "", toolOK, false)
	}
	outItem := map[string]any{"type": "function_call_output", "call_id": item.CallID, "output": truncate(result, 32<<10), "ok": toolOK}
	if filteredExec && !strings.Contains(result, `"ref"`) {
		outItem["raw_output"] = truncate(rawExec, 32<<10)
	}
	if err := a.add(outItem); err != nil {
		return err
	}
	if allowed {
		a.retainToolFinding(ctx, item.Name, item.Arguments, result)
	}
	return nil
}
