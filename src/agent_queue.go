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
	Text string    `json:"text"`
	From string    `json:"from,omitempty"` // agent instance id or label
	Kind string    `json:"kind,omitempty"` // message | alarm | peer
	At   time.Time `json:"at"`
}

func agentInstanceQueuePath(id string) string {
	return filepath.Join(agentInstanceDir(id), "queue.jsonl")
}

var agentQueueMu sync.Mutex

func enqueueAgentMessage(instanceID, text, from, kind string) error {
	instanceID = strings.TrimSpace(instanceID)
	text = strings.TrimSpace(text)
	if instanceID == "" || text == "" {
		return fmt.Errorf("instance id and text required")
	}
	if _, err := readAgentInstanceMeta(instanceID); err != nil {
		return err
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

func formatQueueWake(it AgentQueueItem) string {
	switch it.Kind {
	case "alarm":
		return "[Alarm] " + it.Text
	case "peer":
		from := it.From
		if from == "" {
			from = "peer"
		}
		return "[Message from agent " + from + "] " + it.Text
	default:
		if it.From != "" {
			return "[" + it.From + "] " + it.Text
		}
		return it.Text
	}
}
