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

// AgentAlarm is a persisted wake for an agent instance.
type AgentAlarm struct {
	ID       string        `json:"id"`
	FireAt   time.Time     `json:"fire_at"`
	Message  string        `json:"message"`
	Every    time.Duration `json:"every_ns,omitempty"` // if >0, reschedule after each fire
	EveryRaw string        `json:"every,omitempty"`    // human duration e.g. "30s"
}

func agentInstanceAlarmsPath(id string) string {
	return filepath.Join(agentInstanceDir(id), "alarms.json")
}

var agentAlarmsMu sync.Mutex

func readAgentAlarms(instanceID string) ([]AgentAlarm, error) {
	b, err := os.ReadFile(agentInstanceAlarmsPath(instanceID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var list []AgentAlarm
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func writeAgentAlarms(instanceID string, list []AgentAlarm) error {
	if err := os.MkdirAll(agentInstanceDir(instanceID), 0o700); err != nil {
		return err
	}
	if list == nil {
		list = []AgentAlarm{}
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := agentInstanceAlarmsPath(instanceID)
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
		ID:      fmt.Sprintf("alarm-%s", time.Now().UTC().Format("20060102-150405.000000000")),
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

func dueAgentAlarms(instanceID string, now time.Time) ([]AgentAlarm, error) {
	agentAlarmsMu.Lock()
	defer agentAlarmsMu.Unlock()
	list, err := readAgentAlarms(instanceID)
	if err != nil {
		return nil, err
	}
	var due []AgentAlarm
	remain := make([]AgentAlarm, 0, len(list))
	for _, a := range list {
		if a.FireAt.After(now) {
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
		if err := writeAgentAlarms(instanceID, remain); err != nil {
			return nil, err
		}
	}
	return due, nil
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
