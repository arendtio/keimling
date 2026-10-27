package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type compactionResult struct {
	summary                   string
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

// A snapshot captures a completed prefix. Its boundary is immutable even as
// the foreground agent appends events. Results are committed only on the agent goroutine.
func (a *Agent) startBackgroundCompaction(ctx context.Context) bool {
	if a.Compaction != nil || a.Client == nil || a.Cfg.ContextLimit <= 0 || time.Now().Before(a.CompactionRetryAfter) {
		return false
	}
	end := safePrefixEnd(a.History, 12)
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
	// Snapshot is independently compacted in bounded sections.
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
	go func() {
		client := NewClient(key)
		client.BaseURL = baseURL
		worker := &Agent{Client: client, Cfg: Config{Model: model}}
		text, err := worker.compactIndexed(jobCtx, snapshot, compactionInstruction(""))
		job.done <- compactionResult{summary: text, err: err, inputTokens: client.InputTokens, outputTokens: client.OutputTokens}
	}()
	fmt.Println("◌ Background compaction started")
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
		fmt.Printf("Background compaction retry paused for %s\n", delay)
		fmt.Println("Background compaction failed:", result.err)
		return result.err
	}
	text := strings.TrimSpace(result.summary)
	if text == "" {
		return fmt.Errorf("empty background summary")
	}
	before := contextEstimate(a.History)
	text, afterSize, passes := a.adaptiveCompaction(jobContext(job), text, before, a.History[job.end:])
	denserUsed := false
	if shouldRunDenserPass(before, afterSize, a.Cfg.ContextLimit) {
		if denser, denserSize, denserErr := a.denserCompaction(jobContext(job), text, before, a.History[job.end:]); denserErr == nil {
			text, afterSize = denser, denserSize
			denserUsed = true
		}
	}
	_ = afterSize
	next := makeCompactionView(text, a.History[job.end:])
	if !compactionAcceptable(before, contextEstimate(next)) {
		a.CompactionFailures++
		a.CompactionRetryAfter = time.Now().Add(time.Minute * time.Duration(1<<min(a.CompactionFailures-1, 6)))
		return fmt.Errorf("background compaction saved only %.1f%% (snapshot ~%d, current ~%d, result ~%d tokens; minimum useful %.0f%%); original context preserved", compactionReduction(before, contextEstimate(next)), contextEstimate(a.History[:job.end]), before, contextEstimate(next), compactionMinimumUsefulReduction)
	}
	fmt.Printf("Background compaction: %s · additional passes: %d", compactionDecision(before, contextEstimate(next)), passes)
	if denserUsed {
		fmt.Print(" · denser merge")
	}
	fmt.Println()
	if err := a.checkpoint(next, "semantic-background"); err != nil {
		return err
	}
	_ = a.updateHistoryIndex(text)
	a.CompactionFailures = 0
	a.CompactionRetryAfter = time.Time{}
	a.Client.InputTokens += result.inputTokens
	a.Client.OutputTokens += result.outputTokens
	fmt.Printf("◈ Background compaction applied (%s)\n", time.Since(job.started).Truncate(time.Second))
	fmt.Print(formatCompactionReport(text, before, contextEstimate(next)))
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
	fmt.Println("◌ Context >=90%: waiting for background compaction")
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

// At critical occupancy attempt one foreground compaction using the latest safe prefix.
// Never proceed with an unsafe oversized context if recovery also fails.
func (a *Agent) recoverCompaction(ctx context.Context, original error) error {
	fmt.Println("◌ Background compaction did not free enough context; attempting foreground recovery")
	before := contextEstimate(a.History)
	if err := a.compactContext(ctx, "hard", ""); err != nil {
		return fmt.Errorf("background compaction failed (%v); foreground recovery failed (%w); context unchanged", original, err)
	}
	if contextEstimate(a.History) >= before {
		return fmt.Errorf("foreground recovery found no compactable history; context remains >=90%% (pending live tool call or insufficient completed history)")
	}
	return nil
}

func jobContext(job *compactionJob) context.Context {
	if job != nil && job.ctx != nil {
		return job.ctx
	}
	return context.Background()
}
