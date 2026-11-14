package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// priorResultFile is one tool result to place in the store before a oneshot run.
// The file stays outside the binary. The journal receives a reference, not the bytes.
type priorResultFile struct {
	User      string          `json:"user"`
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
	Value     json.RawMessage `json:"value"`
}

func loadPriorResult(a *Agent, path string) error {
	if a == nil || a.Journal == nil {
		return fmt.Errorf("prior result needs a journal")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc priorResultFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	if strings.TrimSpace(doc.Tool) == "" || len(doc.Value) == 0 {
		return fmt.Errorf("prior result needs a tool name and a value")
	}
	var value any
	if err := json.Unmarshal(doc.Value, &value); err != nil {
		return err
	}
	shown, err := a.toolResults().hold("json", value, string(doc.Value))
	if err != nil {
		return err
	}
	user := strings.TrimSpace(doc.User)
	if user == "" {
		user = "A tool returned this result."
	}
	args := strings.TrimSpace(string(doc.Arguments))
	if args == "" || args == "null" {
		args = "{}"
	}
	if err := a.add(map[string]any{"role": "user", "content": user}); err != nil {
		return err
	}
	if err := a.add(map[string]any{"type": "function_call", "name": doc.Tool, "call_id": "prior-result", "arguments": args}); err != nil {
		return err
	}
	return a.add(map[string]any{"type": "function_call_output", "call_id": "prior-result", "output": shown, "ok": true})
}

func (s *dataStore) exportValues() []any {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]any, 0, len(s.order))
	for _, id := range s.order {
		if it := s.items[id]; it != nil {
			out = append(out, it.value)
		}
	}
	return out
}

func writeResultStore(session string, store *dataStore) error {
	if session == "" || store == nil {
		return nil
	}
	b, err := json.Marshal(store.exportValues())
	if err != nil {
		return err
	}
	return os.WriteFile(session+".results.json", append(b, '\n'), 0o600)
}

// runOneshot reads one message, runs it on this agent, and prints token counts.
func runOneshot(a *Agent, priorPath string) error {
	if strings.TrimSpace(priorPath) != "" {
		if err := loadPriorResult(a, priorPath); err != nil {
			return err
		}
	}
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil && len(strings.TrimSpace(line)) == 0 {
		return fmt.Errorf("oneshot needs one message on stdin")
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return fmt.Errorf("oneshot needs one message on stdin")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	start := time.Now()
	runErr := a.Run(ctx, line, bufio.NewReader(bytes.NewReader(nil)))
	_ = writeResultStore(a.Cfg.SessionPath, a.toolResults())
	inTok, outTok := int64(0), int64(0)
	if a.Client != nil {
		inTok, outTok = a.Client.InputTokens, a.Client.OutputTokens
	}
	fmt.Printf("run finished input_tokens=%d output_tokens=%d duration_ms=%d\n", inTok, outTok, time.Since(start).Milliseconds())
	return runErr
}
