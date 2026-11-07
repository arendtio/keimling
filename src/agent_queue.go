package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AgentQueueItem is one durable inbound wake for an agent instance.
type AgentQueueItem struct {
	Text      string    `json:"text"`
	From      string    `json:"from,omitempty"` // agent instance id, thread id, or label
	FromName  string    `json:"from_name,omitempty"`
	Workspace string    `json:"workspace,omitempty"`
	Kind      string    `json:"kind,omitempty"` // message | alarm | peer | thread
	At        time.Time `json:"at"`
}

func agentInstanceQueuePath(id string) string {
	return filepath.Join(agentInstanceDir(id), "queue.jsonl")
}

var agentQueueMu sync.Mutex

func enqueueAgentMessage(instanceID, text, from, kind string) error {
	return enqueueAgentQueueItem(instanceID, AgentQueueItem{Text: text, From: from, Kind: kind})
}

func enqueueAgentQueueItem(instanceID string, item AgentQueueItem) error {
	instanceID = strings.TrimSpace(instanceID)
	item.Text = strings.TrimSpace(item.Text)
	if instanceID == "" || item.Text == "" {
		return fmt.Errorf("instance id and text required")
	}
	if _, err := readAgentInstanceMeta(instanceID); err != nil {
		return err
	}
	item.From = strings.TrimSpace(item.From)
	item.FromName = strings.TrimSpace(item.FromName)
	item.Workspace = strings.TrimSpace(item.Workspace)
	item.Kind = strings.TrimSpace(item.Kind)
	if item.Kind == "" {
		item.Kind = "message"
	}
	if item.At.IsZero() {
		item.At = time.Now().UTC()
	}
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}
	agentQueueMu.Lock()
	defer agentQueueMu.Unlock()
	if err := os.MkdirAll(agentInstanceDir(instanceID), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(agentInstanceQueuePath(instanceID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func dequeueAgentMessages(instanceID string) ([]AgentQueueItem, error) {
	agentQueueMu.Lock()
	defer agentQueueMu.Unlock()
	path := agentInstanceQueuePath(instanceID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var items []AgentQueueItem
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var it AgentQueueItem
		if json.Unmarshal([]byte(line), &it) != nil || strings.TrimSpace(it.Text) == "" {
			continue
		}
		items = append(items, it)
	}
	_ = f.Close()
	_ = os.Remove(path)
	return items, sc.Err()
}

func threadQueuePath(workspaceID, threadID string) string {
	return filepath.Join(threadDir(workspaceID, threadID), "queue.jsonl")
}

var threadQueueMu sync.Mutex

func enqueueThreadMessage(workspaceID, threadID, text, from, kind string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	threadID = strings.TrimSpace(threadID)
	text = strings.TrimSpace(text)
	if workspaceID == "" || threadID == "" || text == "" {
		return fmt.Errorf("workspace, thread, and text required")
	}
	if st, err := os.Stat(threadDir(workspaceID, threadID)); err != nil || !st.IsDir() {
		return fmt.Errorf("unknown thread %q", threadID)
	}
	item := AgentQueueItem{
		Text: text,
		From: strings.TrimSpace(from),
		Kind: strings.TrimSpace(kind),
		At:   time.Now().UTC(),
	}
	if item.Kind == "" {
		item.Kind = "message"
	}
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}
	threadQueueMu.Lock()
	defer threadQueueMu.Unlock()
	f, err := os.OpenFile(threadQueuePath(workspaceID, threadID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func peekThreadMessages(workspaceID, threadID string) ([]AgentQueueItem, error) {
	threadQueueMu.Lock()
	defer threadQueueMu.Unlock()
	return readThreadQueueLocked(workspaceID, threadID)
}

func dequeueThreadMessage(workspaceID, threadID string) (AgentQueueItem, bool, error) {
	threadQueueMu.Lock()
	defer threadQueueMu.Unlock()
	items, err := readThreadQueueLocked(workspaceID, threadID)
	if err != nil || len(items) == 0 {
		return AgentQueueItem{}, false, err
	}
	rest := items[1:]
	path := threadQueuePath(workspaceID, threadID)
	if len(rest) == 0 {
		_ = os.Remove(path)
		return items[0], true, nil
	}
	var b strings.Builder
	for _, it := range rest {
		line, err := json.Marshal(it)
		if err != nil {
			return AgentQueueItem{}, false, err
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return AgentQueueItem{}, false, err
	}
	return items[0], true, nil
}

func readThreadQueueLocked(workspaceID, threadID string) ([]AgentQueueItem, error) {
	f, err := os.Open(threadQueuePath(workspaceID, threadID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var items []AgentQueueItem
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var it AgentQueueItem
		if json.Unmarshal([]byte(line), &it) != nil || strings.TrimSpace(it.Text) == "" {
			continue
		}
		items = append(items, it)
	}
	return items, sc.Err()
}

func formatWatchWake(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "[Watch]"
	}
	if strings.HasPrefix(text, "[Watch]") {
		return text
	}
	return "[Watch]\n" + text
}

// watchDisplayText is the result a watch shows. The reminder under the blank
// line stays in the journal for the model and is not the visible line.
func watchDisplayText(text string) (string, bool) {
	raw := strings.TrimSpace(text)
	if !strings.HasPrefix(raw, "[Watch]") {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(raw, "[Watch]"))
	if result, reminder, ok := strings.Cut(rest, "\n\n"); ok {
		result = strings.TrimSpace(result)
		if result != "" {
			return result, true
		}
		return strings.TrimSpace(reminder), true
	}
	lines := strings.Split(rest, "\n")
	if len(lines) > 1 {
		return strings.TrimSpace(strings.Join(lines[1:], "\n")), true
	}
	return rest, true
}

func formatQueueWake(it AgentQueueItem) string {
	switch it.Kind {
	case "alarm", "watch":
		return formatWatchWake(it.Text)
	case "peer":
		label := agentInstanceDisplayName(it.From)
		if id := strings.TrimSpace(it.From); id != "" && label != id {
			label = label + " instance_id=" + id
		}
		return formatPeerUserText(label+" via message_agent", it.Text)
	case "thread":
		return formatPeerUserText("thread_id="+strings.TrimSpace(it.From)+" via message_agent", it.Text)
	default:
		if it.From != "" {
			return "[" + it.From + "] " + it.Text
		}
		return it.Text
	}
}
