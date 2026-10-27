package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type compactionResult struct {
	charter                   *Charter
	end                       int
	err                       error
	inputTokens, outputTokens int64
}
type compactionJob struct {
	ctx     context.Context
	cancel  context.CancelFunc
	end     int
	started time.Time
	done    chan compactionResult
}

// Background charter refresh snapshots a completed prefix; commit only on the agent goroutine.
func (a *Agent) startBackgroundCompaction(ctx context.Context) bool {
	if a.Compaction != nil || a.Client == nil || a.Cfg.ContextLimit <= 0 || time.Now().Before(a.CompactionRetryAfter) {
		return false
	}
	end := safePrefixEnd(a.History, charterTailKeep)
	if end == 0 {
		end = safePrefixEnd(a.History, 2)
	}
	if end == 0 {
		return false
	}
	snapshot := forCompaction(a.History[:end])
	if len(snapshot) == 0 {
		return false
	}
	base := a.SessionCtx
	if base == nil {
		base = context.Background()
	}
	jobCtx, cancel := context.WithCancel(base)
	job := &compactionJob{ctx: jobCtx, cancel: cancel, end: end, started: time.Now(), done: make(chan compactionResult, 1)}
	a.Compaction = job
	key := a.Client.APIKey
	model := a.compactionModel()
	baseURL := a.Client.BaseURL
	curJSON, _ := json.MarshalIndent(a.ensureCharter(), "", "  ")
	go func() {
		client := NewClient(key)
		client.BaseURL = baseURL
		var body strings.Builder
		body.WriteString("Current charter JSON:\n")
		body.Write(curJSON)
		body.WriteString("\n\nNew material to fold (truncated JSONL events):\n")
		for i, raw := range snapshot {
			if i >= 80 {
				body.WriteString("…(truncated)\n")
				break
			}
			body.WriteString(truncate(string(raw), 1500))
			body.WriteByte('\n')
		}
		instr := `Update the project charter JSON. Return ONLY valid JSON with keys:
goal (string), plan (array of {step,status:pending|in_progress|done}), position (1-based int),
findings (string array), open_questions (string array), constraints (string array), recent (string array).
Merge durable facts; clear or shrink recent after folding. Do not invent facts. English. No markdown fences.`
		text, err := client.compactionAsk(jobCtx, model, instr, body.String(), 4096)
		if err != nil {
			job.done <- compactionResult{err: err, inputTokens: client.InputTokens, outputTokens: client.OutputTokens}
			return
		}
		next, parseErr := parseCharterJSON(text)
		if parseErr != nil {
			job.done <- compactionResult{err: parseErr, inputTokens: client.InputTokens, outputTokens: client.OutputTokens}
			return
		}
		job.done <- compactionResult{charter: next, end: end, inputTokens: client.InputTokens, outputTokens: client.OutputTokens}
	}()
	fmt.Println("◌ Background charter refresh started")
	return true
}

func (a *Agent) adoptCompaction(job *compactionJob, result compactionResult) error {
	a.Compaction = nil
	defer func() {
		if job.cancel != nil {
			job.cancel()
		}
	}()
	if result.err != nil {
		a.CompactionFailures++
		delay := time.Minute * time.Duration(1<<min(a.CompactionFailures-1, 6))
		a.CompactionRetryAfter = time.Now().Add(delay)
		fmt.Printf("Background charter refresh retry paused for %s\n", delay)
		fmt.Println("Background charter refresh failed:", result.err)
		return result.err
	}
	if result.charter == nil {
		return fmt.Errorf("empty background charter")
	}
	before := a.contextOccupancyEstimate()
	end := result.end
	if end <= 0 || end > len(a.History) {
		end = job.end
	}
	if end <= 0 || end > len(a.History) {
		return fmt.Errorf("stale background snapshot boundary")
	}
	a.Charter = result.charter
	if err := a.persistCharter(); err != nil {
		return err
	}
	tail := a.History[end:]
	if err := a.checkpoint(tail, "charter-background"); err != nil {
		return err
	}
	_ = a.updateHistoryIndex("")
	a.CompactionFailures = 0
	a.CompactionRetryAfter = time.Time{}
	a.Client.InputTokens += result.inputTokens
	a.Client.OutputTokens += result.outputTokens
	after := a.contextOccupancyEstimate()
	fmt.Printf("◈ Background charter refresh applied (%s) · ~%d → ~%d tokens\n", time.Since(job.started).Truncate(time.Second), before, after)
	return nil
}

func (a *Agent) manageBackgroundCompaction(ctx context.Context) error {
	limit := a.Cfg.ContextLimit
	if limit <= 0 {
		return nil
	}
	estimate := a.contextOccupancyEstimate()
	if a.Compaction != nil {
		select {
		case result := <-a.Compaction.done:
			if err := a.adoptCompaction(a.Compaction, result); err != nil && estimate*100 >= 90*limit {
				return a.recoverCompaction(ctx, err)
			}
		default:
		}
	}
	estimate = a.contextOccupancyEstimate()
	if estimate*100 >= 70*limit && a.Compaction == nil {
		a.startBackgroundCompaction(ctx)
	}
	if estimate*100 < 90*limit {
		return nil
	}
	if a.Compaction == nil {
		return a.recoverCompaction(ctx, fmt.Errorf("no safe background snapshot (pending tool call or insufficient completed history)"))
	}
	fmt.Println("◌ Context >=90%: waiting for background charter refresh")
	job := a.Compaction
	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-job.done:
		if err := a.adoptCompaction(job, result); err != nil {
			return a.recoverCompaction(ctx, err)
		}
		return nil
	}
}

func (a *Agent) recoverCompaction(ctx context.Context, original error) error {
	fmt.Println("◌ Background charter refresh did not free enough context; attempting foreground recovery")
	before := a.contextOccupancyEstimate()
	if err := a.compactContext(ctx, "hard", ""); err != nil {
		return fmt.Errorf("background charter refresh failed (%v); foreground recovery failed (%w); context unchanged", original, err)
	}
	if a.contextOccupancyEstimate() >= before {
		return fmt.Errorf("foreground recovery found no foldable history; context remains >=90%% (pending live tool call or insufficient completed history)")
	}
	return nil
}

func jobContext(job *compactionJob) context.Context {
	if job != nil && job.ctx != nil {
		return job.ctx
	}
	return context.Background()
}
