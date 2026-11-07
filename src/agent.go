package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Agent struct {
	Cfg                  Config
	Client               *Client
	Journal              *Journal
	History              []json.RawMessage
	Charter              *Charter
	Skills               *SkillRegistry
	MCP                  *MCPManager
	TaskActive           bool
	TaskDone             bool
	TaskBlocked          bool
	PilotReport          string
	Control              *taskControl
	PendingSteering      bool
	Plan                 []PlanStep
	Compaction           *compactionJob
	SessionCtx           context.Context
	CompactionFailures   int
	CompactionRetryAfter time.Time
	LastTaskMessage      string
	ResumeAvailable      bool
	// Emit, when set, receives UI events (kind "line" or "status") for HTTP/SSE clients.
	Emit func(kind, text string)
	// ApprovalCh, when non-nil, receives approval decisions instead of reading stdin.
	ApprovalCh chan bool
	// Shares holds capability-URL file shares for GET /files/{token}.
	Shares *fileShareHub
	// Artifacts are HTML/viewable files opened in the GUI panel via show_artifact.
	Artifacts *artifactHub
	// Credentials is the process-lifetime credential vault (nil = disabled).
	Credentials *CredentialBroker
	// runChat is the main chat model frozen for the current Run. Nil when idle.
	runChat *chatTarget
	// notedChat is the agent chat model last shown, so a later change can say
	// when encrypted reasoning will be left out. Idle only.
	modelNoted bool
	notedChat  chatTarget
	// toolData holds tool results the model can name without receiving the bytes.
	toolData        *dataStore
	toolDataPending *dataPending
}

func (a *Agent) emit(kind, text string) {
	if a == nil || a.Emit == nil {
		return
	}
	a.Emit(kind, text)
}

func (a *Agent) emitJSON(kind string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	a.emit(kind, string(b))
}

func (a *Agent) emitLine(text string) {
	// Skip terminal status redraws ("\r\033[2K…"); keep real content and normalize CRs.
	if strings.Contains(text, "\r") && strings.Contains(text, "\033[2K") {
		return
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if text == "" {
		return
	}
	a.emit("line", text)
}

func (a *Agent) print(args ...any) {
	s := fmt.Sprint(args...)
	a.emitLine(s)
	fmt.Fprint(os.Stdout, s)
}

func (a *Agent) printf(format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	a.emitLine(s)
	fmt.Fprint(os.Stdout, s)
}

func (a *Agent) println(args ...any) {
	a.print(fmt.Sprintln(args...))
}

type PlanStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

func decodeArgs(input string, out any) error {
	if err := json.Unmarshal([]byte(input), out); err != nil {
		return fmt.Errorf("invalid tool arguments: %w", err)
	}
	return nil
}
func (a *Agent) add(item any) error {
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}
	if err := a.Journal.AppendItem(b); err != nil {
		return err
	}
	a.History = a.Journal.History
	return nil
}

// Resume must not submit dangling native function calls to the Responses API,
// nor retry an interrupted side-effectful tool operation automatically.
func (a *Agent) repairInterruptedCalls() error {
	pending := map[string]bool{}
	order := []string{}
	for _, raw := range a.History {
		typ, id := itemKind(raw)
		if id == "" {
			continue
		}
		if typ == "function_call" {
			if !pending[id] {
				order = append(order, id)
			}
			pending[id] = true
		}
		if typ == "function_call_output" {
			delete(pending, id)
		}
	}
	for _, id := range order {
		if pending[id] {
			if err := a.add(map[string]any{"type": "function_call_output", "call_id": id, "output": "INTERRUPTED: previous action's execution state is unknown. Inspect live state before deciding whether to repeat. Do not automatically replay a side-effectful command."}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Agent) finishRun(errp *error) {
	if a == nil || errp == nil || *errp == nil {
		return
	}
	err := *errp
	// Close dangling tool calls before the note so the transcript pairs
	// each call with its interrupted result.
	_ = a.repairInterruptedCalls()
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		a.note("Stopped")
		return
	}
	a.note("Run stopped: " + err.Error())
}

func (a *Agent) noteAutopilot(action, reason, extra string) {
	line := "Autopilot: " + strings.TrimSpace(action)
	if s := strings.TrimSpace(reason); s != "" {
		line += " · " + s
	}
	if s := strings.TrimSpace(extra); s != "" {
		line += "\n" + s
	}
	a.note(line)
}

func (a *Agent) Run(ctx context.Context, message string, reader *bufio.Reader) (err error) {
	defer func() { _ = a.updateHistoryIndex("") }()
	defer a.finishRun(&err)
	// Agent instances own a fixed main thread under their data dir — don't rename workspace threads.
	shown, attached := splitChatFiles(message)
	if strings.TrimSpace(shown) == "" && len(attached) > 0 {
		shown = attached[0].Name
	}
	if a.Cfg.WorkspaceID != "" && a.Cfg.ThreadID != "" && strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		go a.maybeGenerateThreadTitle(ctx, shown)
	}
	if message != "" {
		if err := a.add(map[string]any{"role": "user", "content": message}); err != nil {
			return err
		}
		if c := a.ensureCharter(); strings.TrimSpace(c.Goal) == "" {
			c.Goal = truncate(strings.TrimSpace(shown), 400)
			_ = a.persistCharter()
		}
	} else {
		message = a.LastTaskMessage
		if err := a.repairInterruptedCalls(); err != nil {
			return err
		}
	}
	a.TaskDone = false
	a.TaskBlocked = false
	a.PilotReport = ""
	a.emit("status", "running")
	defer a.emit("status", "idle")
	defer func() { _ = runPluginHooks("stop", "") }()
	if note := runPluginHooks("sessionStart", message); note != "" {
		a.note(note)
	}
	a.beginRunChat()
	defer a.endRunChat()
	pilotCycles := 0
	lastPilotReport := ""
	repeatedReports := 0
	for step := 0; ; step++ {
		if err := a.handleControls(ctx); err != nil {
			return err
		}
		if step >= a.Cfg.MaxSteps {
			if !a.Cfg.Autopilot || !a.TaskActive {
				if a.TaskActive {
					return fmt.Errorf("resumable plan paused: maximum API rounds (%d) reached", a.Cfg.MaxSteps)
				}
				return fmt.Errorf("maximum API rounds (%d) reached", a.Cfg.MaxSteps)
			}
			if repeatedReports >= 3 || pilotCycles >= 200 {
				a.note("Autopilot stopped: repeated result or supervision limit. Human review needed.")
				return nil
			}
			pilotCycles++
			a.printf("\n◌ Autopilot reviewing %d-round limit (supervision %d/200)…\n", a.Cfg.MaxSteps, pilotCycles)
			decision, pilotErr := a.Client.Pilot(ctx, a.Cfg.PilotModel, a.Cfg.PilotReasoning, message, "Worker reached its API-round limit without completing the active task. Review progress and decide whether to continue for another block of rounds.", a.autopilotPrompt(), a.modelInput())
			if pilotErr != nil {
				return fmt.Errorf("autopilot round-limit review: %w", pilotErr)
			}
			extra := ""
			if decision.Action == "done" {
				// Do not mark an unverified unfinished task complete merely because the supervisor says so.
				extra = "Autopilot requested completion at the round limit. The worker must verify with finish_task."
			} else if decision.Action == "stop" {
				extra = strings.TrimSpace(decision.Message)
				if extra == "" {
					extra = "Autopilot requested human guidance."
				}
			}
			a.noteAutopilot(decision.Action, decision.Reason, extra)
			if decision.Action == "stop" {
				return nil
			}
			if decision.Action != "continue" && decision.Action != "done" {
				return fmt.Errorf("autopilot returned unknown action %q", decision.Action)
			}
			// Preserve plan, history and tool approval policy across round blocks.
			// The supervisor's suggestion is untrusted advice, never an authorization.
			if err := a.add(map[string]any{"role": "user", "content": "[AUTOPILOT SUPERVISOR — not a human authorization or user request] API-round limit reached. Review the existing plan and continue toward the original objective. Do not repeat side-effectful actions without inspecting their results. Supervisor suggestion: " + decision.Message + ". Existing user approval requirements remain unchanged."}); err != nil {
				return err
			}
			step = -1 // loop increment starts the next full block at zero
			continue
		}
		if err := a.maybeCompact(ctx); err != nil {
			return err
		}
		started := time.Now()
		var thinkingChars, textChars, toolChars atomic.Int64
		var streamingStarted atomic.Bool
		var streamPhase atomic.Value
		var phaseStarted atomic.Value
		streamPhase.Store("waiting")
		phaseStarted.Store(started)
		var streamedText strings.Builder
		emitProgress := func() {
			phase, _ := streamPhase.Load().(string)
			if phase == "" {
				phase = "waiting"
			}
			var count int64
			switch phase {
			case "thinking":
				count = thinkingChars.Load()
			case "text":
				count = textChars.Load()
			case "tool":
				count = toolChars.Load()
			default:
				count = thinkingChars.Load() + textChars.Load() + toolChars.Load()
			}
			ps, _ := phaseStarted.Load().(time.Time)
			if ps.IsZero() {
				ps = started
			}
			phaseElapsed := time.Since(ps).Seconds()
			rate := 0.0
			if phaseElapsed > 0 && count > 0 {
				rate = float64(count) / 4 / phaseElapsed
			}
			a.emitJSON("progress", map[string]any{
				"phase":          phase,
				"elapsed_sec":    int(time.Since(started).Seconds()),
				"chars":          count,
				"approx_tokens":  count / 4,
				"tok_per_s":      rate,
				"thinking_chars": thinkingChars.Load(),
				"text_chars":     textChars.Load(),
				"tool_chars":     toolChars.Load(),
			})
		}
		// Buffer streamed markdown; print a formatted block when the turn finishes
		// so **bold**/code fences are not left as raw markers in the terminal.
		// GUI gets live deltas via structured message events.
		a.Client.StreamText = func(delta string) {
			streamingStarted.Store(true)
			streamedText.WriteString(delta)
			a.presentAssistant(delta, true, false)
		}
		a.Client.StreamProgress = func(deltaChars int, elapsed time.Duration, kind string) {
			switch kind {
			case "thinking":
				thinkingChars.Add(int64(deltaChars))
			case "tool":
				toolChars.Add(int64(deltaChars))
			default:
				textChars.Add(int64(deltaChars))
				kind = "text"
			}
			prev, _ := streamPhase.Load().(string)
			if prev != kind {
				phaseStarted.Store(time.Now())
			}
			streamPhase.Store(kind)
			if prev != kind {
				emitProgress()
			}
		}
		finished := make(chan struct{})
		var statusWG sync.WaitGroup
		statusWG.Add(1)
		a.print("\r\033[2K\033[36m◌\033[0m Working… 0s")
		emitProgress()
		go func() {
			defer statusWG.Done()
			tick := time.NewTicker(time.Second)
			defer tick.Stop()
			for {
				select {
				case <-finished:
					return
				case <-tick.C:
					phase, _ := streamPhase.Load().(string)
					var count int64
					label := "Working"
					switch phase {
					case "thinking":
						count = thinkingChars.Load()
						label = "Thinking"
					case "text":
						count = textChars.Load()
						label = "Generating"
					case "tool":
						count = toolChars.Load()
						label = "Tool args"
					default:
						count = thinkingChars.Load() + textChars.Load() + toolChars.Load()
					}
					ps, _ := phaseStarted.Load().(time.Time)
					if ps.IsZero() {
						ps = started
					}
					phaseElapsed := time.Since(ps).Seconds()
					if count > 0 && phaseElapsed > 0 {
						rate := float64(count) / 4 / phaseElapsed
						a.printf("\r\033[2K\033[36m◌\033[0m %s… %s · ~%.1f tok/s", label, time.Since(started).Truncate(time.Second), rate)
					} else {
						a.printf("\r\033[2K\033[36m◌\033[0m %s… %s", label, time.Since(started).Truncate(time.Second))
					}
					emitProgress()
				}
			}
		}()
		extraTools := append(a.mcpToolDefs(), a.credentialToolDefs()...)
		extraTools = append(extraTools, a.agentExtraTools()...)
		resp, err := a.respondMain(ctx, a.modelInput(), extraTools)
		close(finished)
		statusWG.Wait()
		a.Client.StreamText = nil
		a.Client.StreamProgress = nil
		a.emitJSON("progress", map[string]any{"clear": true})
		a.print("\r\033[2K")
		if streamingStarted.Load() {
			full := streamedText.String()
			a.presentAssistant(full, false, true) // CLI formatted block; GUI replaces stream bubble
			duration := time.Since(started).Seconds()
			if duration > 0 {
				estimated := textChars.Load() / 4
				if resp.Usage.OutputTokens > 0 {
					a.printf("[Generated: %d output tokens · %.1f tok/s over %.1fs]\n", resp.Usage.OutputTokens, float64(resp.Usage.OutputTokens)/duration, duration)
				} else if estimated > 0 {
					a.printf("[Streaming: ~%d visible tokens · ~%.1f tok/s over %.1fs]\n", estimated, float64(estimated)/duration, duration)
				}
			}
		}
		if err != nil {
			return err
		}
		if len(resp.Output) == 0 {
			return fmt.Errorf("empty response")
		}
		calls := 0
		for i := 0; i < len(resp.Output); i++ {
			raw := resp.Output[i]
			if err := a.handleControls(ctx); err != nil {
				return err
			}
			// Persist response item before any local action; never blindly retry actions after interruption.
			raw = a.annotateReasoning(raw)
			if err := a.add(json.RawMessage(raw)); err != nil {
				return err
			}
			var item Item
			if err := json.Unmarshal(raw, &item); err != nil {
				return err
			}
			switch item.Type {
			case "message":
				for _, part := range item.Content {
					if part.Type == "output_text" && part.Text != "" {
						if strings.Contains(part.Text, "<tool_call>") || strings.Contains(part.Text, "<arg_key>") {
							a.note("Model returned tool-call markup as text, not a native function call. No command was executed.")
							continue
						}
						if !resp.StreamedText || !strings.Contains(streamedText.String(), part.Text) {
							a.presentAssistant(part.Text, false, false)
						}
					}
				}
			case "web_search_call":
				a.println("\033[36m⌕\033[0m Web search completed")
			case "function_call":
				n := 1
				for i+n < len(resp.Output) {
					var peek Item
					if json.Unmarshal(resp.Output[i+n], &peek) != nil || peek.Type != "function_call" {
						break
					}
					n++
				}
				items := []Item{item}
				for k := 1; k < n; k++ {
					rawK := a.annotateReasoning(resp.Output[i+k])
					if err := a.add(json.RawMessage(rawK)); err != nil {
						return err
					}
					var it Item
					if err := json.Unmarshal(rawK, &it); err != nil {
						return err
					}
					items = append(items, it)
				}
				if err := a.runFunctionCalls(ctx, message, reader, items); err != nil {
					return err
				}
				calls += n
				i += n - 1
				continue
			}
		}
		a.PendingSteering = false
		if err := ctx.Err(); err != nil {
			return err
		}
		if a.Cfg.Autopilot && (a.TaskDone || a.TaskBlocked || calls == 0) {
			report := a.PilotReport
			if report == "" {
				report = "Worker stopped without a finish_task call."
			}
			if report == lastPilotReport {
				repeatedReports++
			} else {
				repeatedReports = 0
				lastPilotReport = report
			}
			if repeatedReports >= 3 || pilotCycles >= 200 {
				a.note("Autopilot stopped: repeated result or supervision limit. Human review needed.")
				return nil
			}
			pilotCycles++
			a.println("\033[36m◌\033[0m Autopilot reviewing outcome…")
			decision, pilotErr := a.Client.Pilot(ctx, a.Cfg.PilotModel, a.Cfg.PilotReasoning, message, report, a.autopilotPrompt(), a.modelInput())
			if pilotErr != nil {
				return fmt.Errorf("autopilot review: %w", pilotErr)
			}
			extra := ""
			if decision.Action == "stop" {
				extra = strings.TrimSpace(decision.Message)
				if extra == "" {
					extra = "Autopilot requested human guidance."
				}
			}
			a.noteAutopilot(decision.Action, decision.Reason, extra)
			if decision.Action == "done" || decision.Action == "stop" {
				return nil
			}
			a.TaskDone = false
			a.TaskBlocked = false
			a.PilotReport = ""
			a.TaskActive = true
			if err := a.add(map[string]any{"role": "user", "content": "[AUTOPILOT SUPERVISOR — not a human authorization or user request] Reassess objective, verify facts and proceed if safe. Supervisor suggestion: " + decision.Message + ". Existing user approval requirements remain unchanged."}); err != nil {
				return err
			}
			continue
		}
		if a.TaskDone || a.TaskBlocked {
			return nil
		}
		if calls == 0 {
			if a.TaskActive {
				if err := a.add(map[string]any{"role": "user", "content": "Continue the current plan autonomously. Inspect results, take the next concrete step, or use finish_task(status=blocked) to request guidance. Do not give a user-facing progress message."}); err != nil {
					return err
				}
				continue
			}
			return nil
		}
	}
}

func formatTerminal(s string) string {
	return renderMarkdown(s)
}

func (a *Agent) NewChat() error {
	if a.Compaction != nil {
		a.Compaction.cancel()
		a.Compaction = nil
	}
	if a.Client != nil {
		a.Client.LastMainUsage = MainUsage{}
	}
	if a.Cfg.WorkspaceID != "" {
		th, session, err := createThread(a.Cfg.WorkspaceID)
		if err != nil {
			return err
		}
		if a.Journal != nil {
			_ = a.Journal.Close()
		}
		j, err := OpenJournal(session, false)
		if err != nil {
			return err
		}
		a.Journal = j
		a.Cfg.SessionPath = session
		a.Cfg.ThreadID = th.ID
	} else if err := a.Journal.NewChat(); err != nil {
		return err
	}
	a.History = nil
	a.Plan = nil
	a.Charter = &Charter{Version: charterVersion}
	a.TaskActive = false
	a.LastTaskMessage = ""
	a.ResumeAvailable = false
	_ = os.Remove(indexPath(a.Cfg.SessionPath))
	_ = os.Remove(charterPath(a.Cfg.SessionPath))
	if a.Shares != nil {
		a.Shares.Clear()
	}
	if a.Artifacts != nil {
		a.Artifacts.Clear()
	}
	return nil
}

// resetInstanceContext archives this instance's transcript and drops the charter
// and pending queue. The instance, its contacts, tools, and alarms stay.
func (a *Agent) resetInstanceContext() error {
	if a == nil || strings.TrimSpace(a.Cfg.AgentInstanceID) == "" {
		return fmt.Errorf("not an agent instance")
	}
	if a.Journal != nil {
		if err := a.Journal.NewChat(); err != nil {
			return err
		}
	}
	a.History = nil
	a.Plan = nil
	a.Charter = &Charter{Version: charterVersion}
	a.TaskActive = false
	a.LastTaskMessage = ""
	a.ResumeAvailable = false
	if a.Client != nil {
		a.Client.LastMainUsage = MainUsage{}
	}
	_ = os.Remove(charterPath(a.Cfg.SessionPath))
	_ = os.Remove(indexPath(a.Cfg.SessionPath))
	_ = os.Remove(agentInstanceQueuePath(a.Cfg.AgentInstanceID))
	return setAgentWakesHeld(a.Cfg.AgentInstanceID, false)
}

// DeleteThread permanently removes a thread. If it was active, switches to another
// existing thread or creates a fresh one.
func (a *Agent) DeleteThread(threadID string) error {
	threadID = strings.TrimSpace(threadID)
	if a.Cfg.WorkspaceID == "" || threadID == "" {
		return fmt.Errorf("thread delete requires workspace")
	}
	wasActive := threadID == a.Cfg.ThreadID
	if wasActive {
		if a.Compaction != nil {
			a.Compaction.cancel()
			a.Compaction = nil
		}
		if a.Journal != nil {
			_ = a.Journal.Close()
			a.Journal = nil
		}
	}
	if err := deleteThreadDir(a.Cfg.WorkspaceID, threadID); err != nil {
		return err
	}
	if !wasActive {
		return nil
	}
	list, err := listThreads(a.Cfg.WorkspaceID)
	if err != nil {
		return err
	}
	if len(list) > 0 {
		return a.SwitchThread(list[0].ID)
	}
	th, session, err := createThread(a.Cfg.WorkspaceID)
	if err != nil {
		return err
	}
	j, err := OpenJournal(session, false)
	if err != nil {
		return err
	}
	a.Journal = j
	a.History = nil
	a.Cfg.SessionPath = session
	a.Cfg.ThreadID = th.ID
	a.Charter = &Charter{Version: charterVersion}
	a.Plan = nil
	a.TaskActive = false
	a.LastTaskMessage = ""
	a.ResumeAvailable = false
	if a.Shares != nil {
		a.Shares.Clear()
	}
	if a.Artifacts != nil {
		a.Artifacts.Clear()
	}
	return nil
}

// SwitchThread loads another thread in the same workspace (same process/CWD).
func (a *Agent) SwitchThread(threadID string) error {
	threadID = strings.TrimSpace(threadID)
	if a.Cfg.WorkspaceID == "" || threadID == "" {
		return fmt.Errorf("thread switch requires workspace")
	}
	if threadID == a.Cfg.ThreadID {
		return nil
	}
	if !isWorkspaceThreadID(threadID) {
		return fmt.Errorf("unknown thread %q", threadID)
	}
	session := threadSessionPath(a.Cfg.WorkspaceID, threadID)
	if _, err := os.Stat(filepath.Dir(session)); err != nil {
		return fmt.Errorf("unknown thread %q", threadID)
	}
	if a.Compaction != nil {
		a.Compaction.cancel()
		a.Compaction = nil
	}
	if a.Journal != nil {
		_ = a.Journal.Close()
	}
	j, err := OpenJournal(session, true)
	if err != nil {
		return err
	}
	a.Journal = j
	a.History = j.History
	a.Cfg.SessionPath = session
	a.Cfg.ThreadID = threadID
	_ = writeActiveThreadID(a.Cfg.WorkspaceID, threadID)
	if ch, err := loadCharter(session); err == nil {
		a.Charter = ch
	} else {
		a.Charter = &Charter{Version: charterVersion}
	}
	a.Plan = nil
	a.TaskActive = false
	a.LastTaskMessage, a.ResumeAvailable = pendingTask(session)
	if a.Shares != nil {
		a.Shares.Clear()
	}
	if a.Artifacts != nil {
		a.Artifacts.Clear()
	}
	return nil
}
