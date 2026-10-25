package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// filterExecResult changes only what enters the model-facing journal; no raw archive.
func (a *Agent) filterExecResult(ctx context.Context, task, arguments, result string) string {
	var args struct {
		Command     string `json:"command"`
		OutputFocus string `json:"output_focus"`
		RawOutput   bool   `json:"raw_output"`
		Interactive bool   `json:"interactive"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		fmt.Println("Output filtering skipped · invalid exec arguments")
		return result
	}
	if args.RawOutput {
		fmt.Println("Output filtering skipped · raw_output=true")
		return result
	}
	if args.Interactive {
		fmt.Println("Output filtering skipped · interactive=true")
		return result
	}
	if !strings.HasPrefix(result, "exit_code=") {
		fmt.Println("Output filtering skipped · no structured exec result")
		return result
	}
	idx := strings.IndexByte(result, '\n')
	if idx < 0 {
		return result
	}
	header, body := result[:idx], result[idx+1:]
	if strings.Count(strings.TrimSuffix(body, "\n"), "\n")+1 <= 10 || strings.TrimSpace(body) == "" {
		return result
	}
	// A failed filter must not cause lost diagnostics.
	if a.Client == nil {
		fmt.Println("Output filtering unavailable · no API client")
		return result
	}
	focus := args.OutputFocus
	if focus == "" {
		focus = "Identify facts needed for the user's current goal"
	}
	// Keep activity visible throughout the extra LLM request. The agent is still busy.
	started := time.Now()
	done := make(chan struct{})
	stopped := make(chan struct{})
	fmt.Print("\r\033[2K\033[36m◌\033[0m Filtering tool output… 0s")
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Printf("\r\033[2K\033[36m◌\033[0m Filtering tool output… %s", time.Since(started).Truncate(time.Second))
			}
		}
	}()
	filtered, err := a.Client.FilterOutput(ctx, "gpt-6-luna", task, args.Command, focus, body)
	close(done)
	<-stopped
	fmt.Print("\r\033[2K")
	if err != nil {
		fmt.Println("Output filtering unavailable; original output retained:", err)
		return result
	}
	filtered = strings.TrimSpace(filtered)
	// Never silently discard a successful filter because of its compression ratio.
	// A too-long summary is a filter-quality issue, not permission to pass raw output.
	if len(filtered) == 0 {
		return result
	}
	return header + "\n[Filtered]\n" + filtered
}
