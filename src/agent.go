package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
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

func (a *Agent) Run(ctx context.Context, message string, reader *bufio.Reader) error {
	if message != "" {
		if err := a.add(map[string]any{"role": "user", "content": message}); err != nil {
			return err
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
					fmt.Printf("\nResumable plan paused: reached %d API rounds.\n", a.Cfg.MaxSteps)
				}
				return fmt.Errorf("maximum API rounds (%d) reached", a.Cfg.MaxSteps)
			}
			if repeatedReports >= 3 || pilotCycles >= 200 {
				fmt.Println("\n◈ Autopilot stopped: repeated result or 200-supervision limit; human review needed")
				return nil
			}
			pilotCycles++
			fmt.Printf("\n◌ Autopilot reviewing %d-round limit (supervision %d/200)…\n", a.Cfg.MaxSteps, pilotCycles)
			decision, pilotErr := a.Client.Pilot(ctx, a.Cfg.PilotModel, a.Cfg.PilotReasoning, message, "Worker reached its API-round limit without completing the active task. Review progress and decide whether to continue for another block of rounds.", a.History)
			if pilotErr != nil {
				return fmt.Errorf("autopilot round-limit review: %w", pilotErr)
			}
			fmt.Printf("◇ Autopilot: %s", decision.Action)
			if decision.Reason != "" {
				fmt.Printf(" · %s", truncate(decision.Reason, 180))
			}
			fmt.Println()
			if decision.Action == "done" {
				// Do not mark an unverified unfinished task complete merely because the supervisor says so.
				fmt.Println("Autopilot requested completion at round limit; worker must verify with finish_task.")
			} else if decision.Action == "stop" {
				fmt.Println("Autopilot requested human guidance:", decision.Message)
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
		var receivedChars atomic.Int64
		var streamingStarted atomic.Bool
		var streamedText strings.Builder
		a.Client.StreamText = func(delta string) {
			if streamingStarted.CompareAndSwap(false, true) {
				fmt.Print("\r\033[2K\n")
			}
			streamedText.WriteString(delta)
			fmt.Print(delta)
		}
		a.Client.StreamProgress = func(chars int, elapsed time.Duration) { receivedChars.Store(int64(chars)) }
		finished := make(chan struct{})
		var statusWG sync.WaitGroup
		statusWG.Add(1)
		fmt.Print("\r\033[2K\033[36m◌\033[0m Working… 0s")
		go func() {
			defer statusWG.Done()
			tick := time.NewTicker(time.Second)
			defer tick.Stop()
			for {
				select {
				case <-finished:
					return
				case <-tick.C:
					if !streamingStarted.Load() {
						count := receivedChars.Load()
						if count > 0 {
							elapsed := time.Since(started).Seconds()
							rate := float64(count) / 4 / elapsed
							fmt.Printf("\r\033[2K\033[36m◌\033[0m Working… %s · ~%.1f tok/s", time.Since(started).Truncate(time.Second), rate)
						} else {
							fmt.Printf("\r\033[2K\033[36m◌\033[0m Working… %s", time.Since(started).Truncate(time.Second))
						}
					}
				}
			}
		}()
		resp, err := a.Client.Respond(ctx, a.Cfg.Model, a.Cfg.Reasoning, a.History, a.Cfg.Web)
		close(finished)
		statusWG.Wait()
		a.Client.StreamText = nil
		a.Client.StreamProgress = nil
		if streamingStarted.Load() {
			duration := time.Since(started).Seconds()
			if duration > 0 {
				estimated := receivedChars.Load() / 4
				if resp.Usage.OutputTokens > 0 {
					fmt.Printf("\n[Generated: %d output tokens · %.1f tok/s over %.1fs]\n", resp.Usage.OutputTokens, float64(resp.Usage.OutputTokens)/duration, duration)
				} else {
					fmt.Printf("\n[Streaming: ~%d visible tokens · ~%.1f tok/s over %.1fs]\n", estimated, float64(estimated)/duration, duration)
				}
			}
		} else {
			fmt.Print("\r\033[2K")
		}
		if err != nil {
			return err
		}
		if len(resp.Output) == 0 {
			return fmt.Errorf("empty response")
		}
		calls := 0
		for _, raw := range resp.Output {
			if err := a.handleControls(ctx); err != nil {
				return err
			}
			// Persist response item before any local action; never blindly retry actions after interruption.
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
							fmt.Println("\n\033[33m! Model returned tool-call markup as text, not a native Responses API function_call. No command executed. Check server tool-calling format/template.\033[0m")
							continue
						}
						if !resp.StreamedText || !strings.Contains(streamedText.String(), part.Text) {
							fmt.Println("\n" + formatTerminal(part.Text))
						}
					}
				}
			case "web_search_call":
				fmt.Println("\033[36m⌕\033[0m Web search completed")
			case "function_call":
				if a.PendingSteering {
					// A newly received instruction may invalidate an action planned before it arrived.
					// Every tool call still receives a corresponding tool result.
					if err := a.add(map[string]any{"type": "function_call_output", "call_id": item.CallID, "output": "SKIPPED: New user steering arrived. Re-evaluate the plan and this action before retrying."}); err != nil {
						return err
					}
					fmt.Println("↻ Action deferred: new steering received")
					calls++
					continue
				}
				calls++
				if item.CallID == "" {
					return fmt.Errorf("function call without call_id")
				}
				if item.Name != "finish_task" && item.Name != "update_plan" {
					fmt.Println(renderToolCall(item.Name, item.Arguments))
				}
				allowed := item.Name == "web_search" || a.Cfg.Mode == "yolo" || item.Name == "update_plan" || item.Name == "finish_task" || item.Name == "delegate"
				reason := ""
				if a.Cfg.Mode == "auto" && !allowed {
					if forced, why := forceApproval(item.Name, item.Arguments); forced {
						reason = why
					} else {
						cwd, _ := os.Getwd()
						verdict, e := a.Client.Review(ctx, a.Cfg.ReviewModel, a.Cfg.ApprovalPolicy, item.Name, item.Arguments, cwd)
						if e != nil {
							reason = "review unavailable: " + e.Error()
						} else if verdict.Decision == "approve" {
							allowed = true
						} else {
							reason = verdict.Reason
						}
					}
				}
				if a.Cfg.Mode == "ask" && item.Name == "read_file" {
					allowed = true
				}
				if reason != "" {
					fmt.Println("Approval required:", truncate(reason, 240))
				}
				if !allowed {
					if a.Control != nil && isTerminal(0) {
						var e error
						allowed, e = a.Control.requestApproval(ctx)
						if e != nil {
							return e
						}
					} else {
						fmt.Print("Approve? [Y/n] ")
						answer, e := reader.ReadString('\n')
						if e != nil {
							return e
						}
						answer = strings.ToLower(strings.TrimSpace(answer))
						allowed = answer == "" || answer == "y" || answer == "yes" || answer == "j" || answer == "ja"
					}
				}
				result := "DENIED by user"
				rawExecResult := ""
				if allowed {
					if item.Name == "update_plan" {
						var p struct {
							Steps []PlanStep `json:"steps"`
						}
						if e := decodeArgs(item.Arguments, &p); e != nil {
							result = e.Error()
						} else if len(p.Steps) > 20 {
							result = "plan too long"
						} else {
							a.Plan = p.Steps
							a.TaskActive = true
							result = "plan updated"
							fmt.Print(renderPlan(a.Plan))
						}
					} else if item.Name == "finish_task" {
						var f struct {
							Status  string `json:"status"`
							Summary string `json:"summary"`
						}
						if e := decodeArgs(item.Arguments, &f); e != nil {
							result = e.Error()
						} else if f.Status != "done" && f.Status != "blocked" {
							result = "invalid status"
						} else {
							a.TaskDone = f.Status == "done"
							a.TaskBlocked = f.Status == "blocked"
							a.TaskActive = false
							a.PilotReport = f.Status + ": " + f.Summary
							result = "task status recorded: " + f.Status
							fmt.Println(renderFinish(f.Status, f.Summary))
						}
					} else if item.Name == "web_search" {
						var args struct {
							Query string `json:"query"`
						}
						if e := decodeArgs(item.Arguments, &args); e != nil {
							result = "web search error: " + e.Error()
						} else if a.Cfg.ExternalClient == nil || !a.Cfg.Web {
							result = "web search disabled or unavailable"
						} else {
							fmt.Println("⌕ Searching web…")
							answer, e := a.Cfg.ExternalClient.SearchWeb(ctx, args.Query)
							if e != nil {
								result = "web search error: " + e.Error()
							} else {
								result = answer
							}
						}
					} else if item.Name == "search_history" {
						result = a.searchHistory(item.Arguments)
					} else if item.Name == "read_history" {
						result = a.readHistory(item.Arguments)
					} else if item.Name == "delegate" {
						var args DelegateArgs
						if e := decodeArgs(item.Arguments, &args); e != nil {
							result = "delegate error: " + e.Error()
						} else {
							model, effort, e := delegateProfile(a.Cfg, args.Profile)
							if e != nil {
								result = "delegate error: " + e.Error()
							} else {
								fmt.Printf("\033[35m◇ Delegate\033[0m · %s · %s\n", args.Profile, truncate(args.Task, 160))
								answer, e := a.Client.Delegate(ctx, model, effort, args)
								if e != nil {
									result = "delegate error: " + e.Error()
								} else {
									result = answer
								}
							}
						}
					} else {
						if a.Control != nil && toolNeedsTerminal(item.Name, item.Arguments) {
							a.Control.takeTerminal()
							result = executeTool(ctx, item.Name, item.Arguments)
							a.Control.releaseTerminal()
						} else {
							result = executeTool(ctx, item.Name, item.Arguments)
						}
						if item.Name == "exec" {
							rawExecResult = result
							result = a.filterExecResult(ctx, message, item.Arguments, result)
						}
					}
				}
				if item.Name == "exec" {
					code := "?"
					displayedResult := result
					if rawExecResult != "" {
						displayedResult = rawExecResult
					}
					if i := strings.Index(displayedResult, "exit_code="); i >= 0 {
						fields := strings.Fields(displayedResult[i+len("exit_code="):])
						if len(fields) > 0 {
							code = fields[0]
						}
					}
					output := strings.TrimPrefix(displayedResult, "exit_code="+code+"\n")
					if code != "0" {
						fmt.Printf("\033[31mFAIL: Return Value %s\033[0m\n", code)
					}
					if strings.TrimSpace(output) != "" {
						fmt.Print(output)
						if !strings.HasSuffix(output, "\n") {
							fmt.Println()
						}
					}
					if rawExecResult != "" && result != rawExecResult {
						modelVisible := truncate(result, 32<<10)
						fmt.Printf("\033[36m◈ Filtered output → model\033[0m (%d → %d bytes):\n%s\n", len(rawExecResult), len(modelVisible), modelVisible)
					}
				} else {
					if strings.HasPrefix(result, "patch failed:") || strings.HasPrefix(result, "hash mismatch") || strings.HasPrefix(result, "file hash mismatch") || strings.HasPrefix(result, "invalid line range") || strings.HasPrefix(result, "access denied:") {
						fmt.Printf("\033[31m✗\033[0m %s\n", truncate(result, 1800))
					} else {
						fmt.Printf("\033[32m✓\033[0m %s\n", truncate(result, 1800))
					}
				}
				if err := a.add(map[string]any{"type": "function_call_output", "call_id": item.CallID, "output": truncate(result, 32<<10)}); err != nil {
					return err
				}
			}
		}
		a.PendingSteering = false
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
				fmt.Println("\n◈ Autopilot stopped: repeated result or 200-supervision limit; human review needed")
				return nil
			}
			pilotCycles++
			fmt.Println("\033[36m◌\033[0m Autopilot reviewing outcome…")
			decision, pilotErr := a.Client.Pilot(ctx, a.Cfg.PilotModel, a.Cfg.PilotReasoning, message, report, a.History)
			if pilotErr != nil {
				return fmt.Errorf("autopilot review: %w", pilotErr)
			}
			fmt.Printf("◇ Autopilot: %s", decision.Action)
			if decision.Reason != "" {
				fmt.Printf(" · %s", truncate(decision.Reason, 180))
			}
			fmt.Println()
			if decision.Action == "done" {
				return nil
			}
			if decision.Action == "stop" {
				fmt.Println("Autopilot requested human guidance:", decision.Message)
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

var mdLink = regexp.MustCompile(`\[([^]\n]+)\]\((https?://[^)\s]+)\)`)

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
	if err := a.Journal.NewChat(); err != nil {
		return err
	}
	a.History = nil
	a.Plan = nil
	a.TaskActive = false
	a.LastTaskMessage = ""
	a.ResumeAvailable = false
	_ = os.Remove(indexPath(a.Cfg.SessionPath))
	return nil
}
