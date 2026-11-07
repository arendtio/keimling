package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AgentWatch is a persisted wake for an agent instance.
// A time watch fires on the clock. An interval watch runs its steps.
// A file or command watch waits for an event.
type AgentWatch struct {
	ID        string        `json:"id"`
	Kind      string        `json:"kind,omitempty"` // time, interval, file, command; empty is time
	FireAt    time.Time     `json:"fire_at"`
	Message   string        `json:"message"`
	Every     time.Duration `json:"every_ns,omitempty"`
	EveryRaw  string        `json:"every,omitempty"`
	Condition string        `json:"condition,omitempty"` // equals, changed, or empty
	Value     string        `json:"value,omitempty"`
	Steps     []WatchStep   `json:"steps,omitempty"`
	Path      string        `json:"path,omitempty"`
	Command   string        `json:"command,omitempty"`
	Last      string        `json:"last,omitempty"`
	Seen      bool          `json:"seen,omitempty"`
	Exited    string        `json:"exited,omitempty"`
}

// WatchStep is one tool call in a watch chain.
type WatchStep struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// AgentAlarm is the old name of a time watch.
type AgentAlarm = AgentWatch

func agentInstanceWatchesPath(id string) string {
	return filepath.Join(agentInstanceDir(id), "watches.json")
}

// watchPlace is where one chat stores its watches and where a match is delivered.
type watchPlace struct {
	key      string
	path     string
	instance string
	ws       string
	thread   string
}

func (a *Agent) watchPlace() (watchPlace, bool) {
	if a == nil {
		return watchPlace{}, false
	}
	id := strings.TrimSpace(a.Cfg.AgentInstanceID)
	if id == "" {
		return watchPlace{}, false
	}
	return watchPlace{key: id, path: agentInstanceWatchesPath(id), instance: id}, true
}

func (p watchPlace) held() bool {
	return p.instance != "" && agentWakesHeld(p.instance)
}

func (p watchPlace) enqueue(message, body string) {
	text := strings.TrimSpace(body)
	message = strings.TrimSpace(message)
	if message != "" {
		if text != "" {
			text += "\n\n"
		}
		text += message
	}
	if len(text) > 4000 {
		text = text[:4000]
	}
	if text == "" {
		return
	}
	if p.instance != "" {
		_ = enqueueAgentMessage(p.instance, text, "watch", "watch")
		return
	}
	_ = enqueueThreadMessage(p.ws, p.thread, text, "watch", "watch")
}

var agentAlarmsMu sync.Mutex

func readAgentAlarms(instanceID string) ([]AgentAlarm, error) {
	return readWatches(agentInstanceWatchesPath(instanceID))
}

func readWatches(path string) ([]AgentWatch, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var list []AgentWatch
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func writeAgentAlarms(instanceID string, list []AgentAlarm) error {
	return writeWatches(agentInstanceWatchesPath(instanceID), list)
}

func writeWatches(path string, list []AgentWatch) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(filepath.Join(dir, "meta.json")); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("agent instance is gone")
		}
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if list == nil {
		list = []AgentWatch{}
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func addAgentAlarm(instanceID string, fireAt time.Time, message string, every time.Duration) (AgentAlarm, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "Alarm fired."
	}
	if fireAt.Before(time.Now().Add(-time.Second)) {
		return AgentAlarm{}, fmt.Errorf("fire time is in the past")
	}
	if every < 0 {
		every = 0
	}
	agentAlarmsMu.Lock()
	defer agentAlarmsMu.Unlock()
	list, err := readAgentAlarms(instanceID)
	if err != nil {
		return AgentAlarm{}, err
	}
	al := AgentAlarm{
		ID:      newStampedID("watch"),
		Kind:    "time",
		FireAt:  fireAt.UTC(),
		Message: message,
		Every:   every,
	}
	if every > 0 {
		al.EveryRaw = every.String()
	}
	list = append(list, al)
	if err := writeAgentAlarms(instanceID, list); err != nil {
		return AgentAlarm{}, err
	}
	return al, nil
}

func cancelAgentAlarm(instanceID, alarmID string) error {
	alarmID = strings.TrimSpace(alarmID)
	agentAlarmsMu.Lock()
	defer agentAlarmsMu.Unlock()
	list, err := readAgentAlarms(instanceID)
	if err != nil {
		return err
	}
	out := list[:0]
	found := false
	for _, a := range list {
		if a.ID == alarmID {
			found = true
			continue
		}
		out = append(out, a)
	}
	if !found {
		return fmt.Errorf("unknown alarm %q", alarmID)
	}
	return writeAgentAlarms(instanceID, out)
}

// agentAlarmDue reports a wake time that has arrived. It does not change the file.
func agentAlarmDue(instanceID string, now time.Time) bool {
	list, err := readAgentAlarms(instanceID)
	if err != nil {
		return false
	}
	for _, a := range list {
		if watchIsTime(a) && !a.FireAt.After(now) {
			return true
		}
	}
	return false
}

func dueAgentAlarms(instanceID string, now time.Time) ([]AgentAlarm, error) {
	return dueWatches(agentInstanceWatchesPath(instanceID), now)
}

func dueWatches(path string, now time.Time) ([]AgentWatch, error) {
	agentAlarmsMu.Lock()
	defer agentAlarmsMu.Unlock()
	list, err := readWatches(path)
	if err != nil {
		return nil, err
	}
	var due []AgentAlarm
	remain := make([]AgentAlarm, 0, len(list))
	for _, a := range list {
		if !watchIsTime(a) || a.FireAt.After(now) {
			remain = append(remain, a)
			continue
		}
		due = append(due, a)
		every := a.Every
		if every <= 0 && a.EveryRaw != "" {
			every, _ = time.ParseDuration(a.EveryRaw)
		}
		if every > 0 {
			next := a
			next.FireAt = now.Add(every).UTC()
			next.Every = every
			remain = append(remain, next)
		}
	}
	if len(due) > 0 {
		if err := writeWatches(path, remain); err != nil {
			return nil, err
		}
	}
	return due, nil
}

func watchIsTime(w AgentWatch) bool {
	return w.Kind == "" || w.Kind == "time"
}

// agentWatchNeedsProcess reports a watch that runs only while this worker is up.
func agentWatchNeedsProcess(instanceID string) bool {
	list, err := readAgentAlarms(instanceID)
	if err != nil {
		return false
	}
	for _, w := range list {
		switch w.Kind {
		case "interval", "file", "command":
			if w.Exited == "" {
				return true
			}
		}
	}
	return false
}

// parseAlarmWhen accepts:
//   - relative: "in 2m", "in 1h30m", or bare Go duration "2m"
//   - repeating: "every 30s" (first fire after one interval, then repeats)
//   - absolute clock today/tomorrow local: "at 15:04", "at 3:04pm"
//   - RFC3339 timestamp
//
// every is non-zero for repeating alarms.
func parseAlarmWhen(raw string, now time.Time) (fireAt time.Time, every time.Duration, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, 0, fmt.Errorf("when required")
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "every ") {
		d, e := time.ParseDuration(strings.TrimSpace(raw[6:]))
		if e != nil {
			return time.Time{}, 0, fmt.Errorf("every duration: %w", e)
		}
		if d <= 0 {
			return time.Time{}, 0, fmt.Errorf("duration must be positive")
		}
		return now.Add(d), d, nil
	}
	if strings.HasPrefix(lower, "in ") {
		d, e := time.ParseDuration(strings.TrimSpace(raw[3:]))
		if e != nil {
			return time.Time{}, 0, fmt.Errorf("relative duration: %w", e)
		}
		if d <= 0 {
			return time.Time{}, 0, fmt.Errorf("duration must be positive")
		}
		return now.Add(d), 0, nil
	}
	if strings.HasPrefix(lower, "at ") {
		clock := strings.TrimSpace(raw[3:])
		for _, layout := range []string{"15:04", "15:04:05", "3:04pm", "3:04:05pm", "3:04PM", "3:04:05PM"} {
			t, e := time.ParseInLocation(layout, clock, now.Location())
			if e != nil {
				continue
			}
			cand := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, now.Location())
			if !cand.After(now) {
				cand = cand.Add(24 * time.Hour)
			}
			return cand, 0, nil
		}
		return time.Time{}, 0, fmt.Errorf("absolute time: use HH:MM")
	}
	if t, e := time.Parse(time.RFC3339, raw); e == nil {
		return t, 0, nil
	}
	if d, e := time.ParseDuration(raw); e == nil && d > 0 {
		return now.Add(d), 0, nil
	}
	return time.Time{}, 0, fmt.Errorf("when: use \"in 2m\", \"every 30s\", \"at 15:04\", duration, or RFC3339")
}
