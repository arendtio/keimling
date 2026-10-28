package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// reportCompaction publishes compaction progress to the GUI (SSE) and CLI spinner.
func (a *Agent) reportCompaction(phase, message string, started time.Time, done bool, err error) {
	if a == nil {
		return
	}
	elapsed := time.Since(started)
	payload := map[string]any{
		"phase":       phase,
		"message":     message,
		"elapsed_sec": int(elapsed.Seconds()),
		"done":        done,
	}
	if err != nil {
		payload["error"] = err.Error()
	}
	a.emitJSON("compaction", payload)
	sec := elapsed.Truncate(time.Second)
	if !done {
		fmt.Fprintf(os.Stdout, "\r\033[2K\033[36m◌\033[0m Compaction: %s… %s", message, sec)
		return
	}
	fmt.Fprint(os.Stdout, "\r\033[2K")
	if err != nil {
		a.printf("\033[31m✗\033[0m Compaction %s failed · %s: %v\n", phase, sec, err)
		return
	}
	a.printf("\033[36m◈\033[0m Compaction: %s · %s\n", message, sec)
}

// withCompactionProgress runs fn while emitting periodic progress ticks.
func (a *Agent) withCompactionProgress(phase, message string, fn func() error) error {
	started := time.Now()
	a.reportCompaction(phase, message, started, false, nil)
	finished := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-finished:
				return
			case <-tick.C:
				a.reportCompaction(phase, message, started, false, nil)
			}
		}
	}()
	err := fn()
	close(finished)
	wg.Wait()
	if err != nil {
		a.reportCompaction(phase, message, started, true, err)
		return err
	}
	// Phase finished; caller continues with the next phase (no done=true).
	a.reportCompaction(phase, message+" done", started, false, nil)
	return nil
}
