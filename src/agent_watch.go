package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func watchToolDefs() []map[string]any {
	step := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":      map[string]any{"type": "string", "description": "Tool name"},
			"arguments": map[string]any{"type": "string", "description": "JSON object for that tool, as text"},
		},
		"required":             []string{"name", "arguments"},
		"additionalProperties": false,
	}
	return []map[string]any{
		toolDef("create_watch", "Create a watch that wakes only you. Another agent cannot create it, and it does not wake them. when: \"in 2m\", \"every 30s\", \"at 15:04\", or RFC3339. Leave when empty for a file or a long-running command. message is a reminder for yourself, shown as [Watch]. It is not sent to the user or to another agent. condition is empty, equals, or changed. Empty, with no steps and no source, wakes on the time. equals wakes on every result equal to value, including when that value stays. changed remembers the first result and wakes when a later result differs. steps run in order without the model; a data_processing step receives the previous result as input. Pass an empty array when there are no steps. source is empty, file, or command. path is the file or directory for source file. command is a long-running shell command for source command; each stdout line is a result, and you are told once if it exits. A directory watch sees that directory, not its subdirectories. The watch stays until you cancel_watch it.", map[string]any{
			"when":      map[string]any{"type": "string"},
			"message":   map[string]any{"type": "string"},
			"condition": map[string]any{"type": "string", "description": "Empty string for a time such as every 30s or in 2m. equals or changed only when comparing a step result."},
			"value":     map[string]any{"type": "string", "description": "Result to match when condition is equals. Empty otherwise."},
			"steps":     map[string]any{"type": "array", "items": step},
			"source":    map[string]any{"type": "string", "description": "empty, file, or command"},
			"path":      map[string]any{"type": "string"},
			"command":   map[string]any{"type": "string"},
		}, "when", "message", "condition", "value", "steps", "source", "path", "command"),
		toolDef("list_watches", "List watches for this agent instance.", map[string]any{"unused": map[string]any{"type": "string", "description": "Pass empty string"}}, "unused"),
		toolDef("cancel_watch", "Turn off a watch by id from list_watches. It does not run again.", map[string]any{"id": map[string]any{"type": "string"}}, "id"),
	}
}

type watchRequest struct {
	When      string      `json:"when"`
	Message   string      `json:"message"`
	Condition string      `json:"condition"`
	Value     string      `json:"value"`
	Steps     []WatchStep `json:"steps"`
	Source    string      `json:"source"`
	Path      string      `json:"path"`
	Command   string      `json:"command"`
}

func (a *Agent) createWatch(argsJSON string) string {
	place, ok := a.watchPlace()
	if !ok {
		return "create_watch error: only available on agent instances"
	}
	var req watchRequest
	if err := json.Unmarshal([]byte(argsJSON), &req); err != nil {
		return "create_watch error: invalid arguments"
	}
	w, err := a.prepareWatch(req, time.Now())
	if err != nil {
		return "create_watch error: " + err.Error()
	}
	if err := addAgentWatch(place.path, w); err != nil {
		return "create_watch error: " + err.Error()
	}
	return fmt.Sprintf("watch %s created (%s)", w.ID, w.Kind)
}

func (a *Agent) prepareWatch(req watchRequest, now time.Time) (AgentWatch, error) {
	req.Condition = strings.ToLower(strings.TrimSpace(req.Condition))
	req.Source = strings.ToLower(strings.TrimSpace(req.Source))
	req.Path = strings.TrimSpace(req.Path)
	req.Command = strings.TrimSpace(req.Command)
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		req.Message = "Watch fired."
	}
	switch req.Condition {
	case "", "equals", "changed":
	default:
		return AgentWatch{}, fmt.Errorf("condition must be empty, equals, or changed")
	}
	steps := make([]WatchStep, len(req.Steps))
	for i, step := range req.Steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			return AgentWatch{}, fmt.Errorf("step name required")
		}
		if name == "create_watch" || name == "list_watches" || name == "cancel_watch" {
			return AgentWatch{}, fmt.Errorf("a watch cannot run %s", name)
		}
		if !a.instanceToolGranted(name) {
			return AgentWatch{}, fmt.Errorf("tool %s is not granted", name)
		}
		args := strings.TrimSpace(step.Arguments)
		if args == "" {
			args = "{}"
		}
		var obj map[string]any
		if json.Unmarshal([]byte(args), &obj) != nil {
			return AgentWatch{}, fmt.Errorf("arguments for %s must be a JSON object", name)
		}
		steps[i] = WatchStep{Name: name, Arguments: args}
	}
	w := AgentWatch{
		ID:        newStampedID("watch"),
		Message:   req.Message,
		Condition: req.Condition,
		Value:     req.Value,
		Steps:     steps,
		Path:      req.Path,
		Command:   req.Command,
	}
	switch req.Source {
	case "":
		if len(steps) == 0 {
			fireAt, every, err := parseAlarmWhen(req.When, now)
			if err != nil {
				return AgentWatch{}, err
			}
			w.Kind = "time"
			w.FireAt = fireAt.UTC()
			w.Every = every
			if every > 0 {
				w.EveryRaw = every.String()
			}
			return w, nil
		}
		fireAt, every, err := parseAlarmWhen(req.When, now)
		if err != nil {
			return AgentWatch{}, err
		}
		if every <= 0 {
			return AgentWatch{}, fmt.Errorf("a step watch needs every <duration>")
		}
		w.Kind = "interval"
		w.FireAt = fireAt.UTC()
		w.Every = every
		w.EveryRaw = every.String()
		return w, nil
	case "file":
		if req.Path == "" {
			return AgentWatch{}, fmt.Errorf("path required")
		}
		if st, err := os.Stat(req.Path); err != nil || (!st.IsDir() && !st.Mode().IsRegular()) {
			return AgentWatch{}, fmt.Errorf("path is not a file or directory")
		}
		w.Kind = "file"
		return w, nil
	case "command":
		if req.Command == "" {
			return AgentWatch{}, fmt.Errorf("command required")
		}
		if !a.instanceToolGranted("exec") {
			return AgentWatch{}, fmt.Errorf("tool exec is not granted")
		}
		w.Kind = "command"
		return w, nil
	default:
		return AgentWatch{}, fmt.Errorf("source must be empty, file, or command")
	}
}

func addAgentWatch(path string, w AgentWatch) error {
	agentAlarmsMu.Lock()
	defer agentAlarmsMu.Unlock()
	list, err := readWatches(path)
	if err != nil {
		return err
	}
	list = append(list, w)
	return writeWatches(path, list)
}

func (a *Agent) listWatchesTool(_ string) string {
	place, ok := a.watchPlace()
	if !ok {
		return "list_watches error: only available on agent instances"
	}
	list, err := readWatches(place.path)
	if err != nil {
		return "list_watches error: " + err.Error()
	}
	if len(list) == 0 {
		return "no watches"
	}
	var b strings.Builder
	for _, w := range list {
		fmt.Fprintf(&b, "- %s\n", watchSummary(w))
	}
	return strings.TrimSpace(b.String())
}

func watchSummary(w AgentWatch) string {
	kind := w.Kind
	if kind == "" {
		kind = "time"
	}
	extra := ""
	if w.Every > 0 {
		extra = " every " + w.Every.String()
	} else if w.EveryRaw != "" {
		extra = " every " + w.EveryRaw
	}
	when := ""
	if !w.FireAt.IsZero() {
		when = " at " + w.FireAt.Local().Format(time.RFC3339)
	}
	cond := ""
	if w.Condition != "" {
		cond = " " + w.Condition
		if w.Condition == "equals" {
			cond += " " + w.Value
		}
	}
	where := ""
	if w.Path != "" {
		where = " " + w.Path
	}
	if w.Command != "" {
		where = " " + w.Command
	}
	ended := ""
	if w.Exited != "" {
		ended = " exited"
	}
	return fmt.Sprintf("%s %s%s%s%s%s%s: %s", w.ID, kind, when, extra, cond, where, ended, w.Message)
}

func (a *Agent) cancelWatch(argsJSON string) string {
	place, ok := a.watchPlace()
	if !ok {
		return "cancel_watch error: only available on agent instances"
	}
	var x struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &x); err != nil || strings.TrimSpace(x.ID) == "" {
		return "cancel_watch error: id required"
	}
	if err := cancelWatchAt(place.path, strings.TrimSpace(x.ID)); err != nil {
		return "cancel_watch error: " + err.Error()
	}
	watchHubFor(place.key).stop(strings.TrimSpace(x.ID))
	return "watch cancelled"
}

func cancelWatchAt(path, id string) error {
	id = strings.TrimSpace(id)
	agentAlarmsMu.Lock()
	defer agentAlarmsMu.Unlock()
	list, err := readWatches(path)
	if err != nil {
		return err
	}
	out := list[:0]
	found := false
	for _, w := range list {
		if w.ID == id {
			found = true
			continue
		}
		out = append(out, w)
	}
	if !found {
		return fmt.Errorf("unknown watch %q", id)
	}
	return writeWatches(path, out)
}

func watchResultBody(result string) string {
	result = strings.TrimSpace(result)
	if i := strings.LastIndex(result, "\nref: "); i >= 0 {
		return strings.TrimSpace(result[:i])
	}
	return result
}

func watchRef(result string) string {
	if i := strings.LastIndex(result, "\nref: "); i >= 0 {
		return strings.TrimSpace(result[i+len("\nref: "):])
	}
	var envelope struct {
		Ref string `json:"ref"`
	}
	if json.Unmarshal([]byte(result), &envelope) == nil {
		return strings.TrimSpace(envelope.Ref)
	}
	return ""
}

// watchDecide reports whether this result wakes the agent.
// changed stores the first result and wakes only on a later difference.
// equals wakes on every equal result.
func watchDecide(w AgentWatch, body string) (match bool, last string, seen bool) {
	switch w.Condition {
	case "equals":
		return body == strings.TrimSpace(w.Value), w.Last, w.Seen
	case "changed":
		if !w.Seen {
			return false, body, true
		}
		if body != w.Last {
			return true, body, true
		}
		return false, w.Last, true
	default:
		return true, body, true
	}
}

func (a *Agent) runWatchChain(ctx context.Context, steps []WatchStep, input string) (string, error) {
	if len(steps) == 0 {
		return input, nil
	}
	ref := ""
	if strings.TrimSpace(input) != "" {
		held, err := a.toolResults().hold("text", input, input)
		if err != nil {
			return "", err
		}
		ref = watchRef(held)
	}
	var last string
	for _, step := range steps {
		name := strings.TrimSpace(step.Name)
		args := step.Arguments
		if strings.TrimSpace(args) == "" {
			args = "{}"
		}
		if name == "data_processing" && ref != "" {
			args = withWatchInputRef(args, ref)
		}
		if !a.instanceToolGranted(name) {
			return "", fmt.Errorf("tool %s is not granted", name)
		}
		var result string
		if name == "exec" {
			result = a.runExecWithCredentials(ctx, args)
		} else {
			result, _ = a.invokeTool(ctx, "", Item{Type: "function_call", Name: name, Arguments: args})
		}
		result = a.attachDataRef(name, result)
		last = result
		if next := watchRef(result); next != "" {
			ref = next
		}
	}
	return last, nil
}

func withWatchInputRef(args, ref string) string {
	var m map[string]any
	if json.Unmarshal([]byte(args), &m) != nil {
		return args
	}
	cur, _ := m["input_ref"].(string)
	if strings.TrimSpace(cur) == "" {
		m["input_ref"] = ref
	}
	b, err := json.Marshal(m)
	if err != nil {
		return args
	}
	return string(b)
}

func saveWatchProgress(path, id, last string, seen bool, exited string) error {
	agentAlarmsMu.Lock()
	defer agentAlarmsMu.Unlock()
	list, err := readWatches(path)
	if err != nil {
		return err
	}
	found := false
	for i := range list {
		if list[i].ID != id {
			continue
		}
		list[i].Last = last
		list[i].Seen = seen
		if exited != "" {
			list[i].Exited = exited
		}
		found = true
		break
	}
	if !found {
		return nil
	}
	return writeWatches(path, list)
}

func scheduleIntervalWatch(path, id string, next time.Time) error {
	agentAlarmsMu.Lock()
	defer agentAlarmsMu.Unlock()
	list, err := readWatches(path)
	if err != nil {
		return err
	}
	for i := range list {
		if list[i].ID == id {
			list[i].FireAt = next.UTC()
			break
		}
	}
	return writeWatches(path, list)
}

func deliverWatchResult(a *Agent, p watchPlace, id, result string) {
	if a == nil || p.path == "" {
		return
	}
	list, err := readWatches(p.path)
	if err != nil {
		return
	}
	var w AgentWatch
	found := false
	for _, item := range list {
		if item.ID == id {
			w = item
			found = true
			break
		}
	}
	if !found || w.Exited != "" {
		return
	}
	body := watchResultBody(result)
	match, last, seen := watchDecide(w, body)
	_ = saveWatchProgress(p.path, id, last, seen, "")
	if !match || p.held() {
		return
	}
	p.enqueue(w.Message, body)
}

func (a *Agent) runDueIntervalWatches(ctx context.Context, now time.Time) {
	p, ok := a.watchPlace()
	if !ok {
		return
	}
	a.runDueIntervalWatchesAt(ctx, now, p)
}

func (a *Agent) runDueIntervalWatchesAt(ctx context.Context, now time.Time, p watchPlace) {
	if a == nil || p.path == "" {
		return
	}
	list, err := readWatches(p.path)
	if err != nil {
		return
	}
	hub := watchHubFor(p.key)
	for _, w := range list {
		if w.Kind != "interval" || w.Exited != "" || w.FireAt.After(now) {
			continue
		}
		if !hub.tryStart(w.ID) {
			continue
		}
		every := w.Every
		if every <= 0 && w.EveryRaw != "" {
			every, _ = time.ParseDuration(w.EveryRaw)
		}
		if every <= 0 {
			every = time.Second
		}
		_ = scheduleIntervalWatch(p.path, w.ID, now.Add(every))
		result, runErr := a.runWatchChain(ctx, w.Steps, "")
		hub.finish(w.ID)
		if runErr != nil {
			deliverWatchResult(a, p, w.ID, runErr.Error())
			continue
		}
		deliverWatchResult(a, p, w.ID, result)
	}
}
