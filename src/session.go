package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Journal struct {
	File    *os.File
	History []json.RawMessage
}

func OpenJournal(path string, resume bool) (*Journal, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	var history []json.RawMessage
	var view []json.RawMessage
	if resume {
		f, err := os.Open(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if err == nil {
			scanner := bufio.NewScanner(f)
			scanner.Buffer(make([]byte, 4096), 5<<20)
			for scanner.Scan() {
				var row struct {
					Type string          `json:"type"`
					Item json.RawMessage `json:"item"`
				}
				if json.Unmarshal(scanner.Bytes(), &row) == nil {
					if row.Type == "item" && len(row.Item) > 0 {
						history = append(history, row.Item)
						view = append(view, row.Item)
					}
					if row.Type == "checkpoint" && len(row.Item) > 0 {
						var snap []json.RawMessage
						if json.Unmarshal(row.Item, &snap) == nil {
							view = snap
						}
					}
				}
			}
			err = scanner.Err()
			_ = f.Close()
			if err != nil {
				return nil, err
			}
		}
	}
	// Reconcile interrupted calls against the compacted model-facing view.
	if resume {
		pending := map[string]bool{}
		for _, raw := range view {
			var x struct {
				Type   string `json:"type"`
				CallID string `json:"call_id"`
			}
			if json.Unmarshal(raw, &x) != nil {
				continue
			}
			if x.Type == "function_call" && x.CallID != "" {
				pending[x.CallID] = true
			}
			if x.Type == "function_call_output" {
				delete(pending, x.CallID)
			}
		}
		for id := range pending {
			output, _ := json.Marshal(map[string]string{"type": "function_call_output", "call_id": id, "output": "INTERRUPTED: execution state unknown. Inspect before retrying."})
			view = append(view, output)
		}
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if !resume {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		return nil, err
	}
	if view == nil {
		view = history
	}
	return &Journal{File: f, History: view}, nil
}
func (j *Journal) AppendItem(item json.RawMessage) error {
	row, _ := json.Marshal(map[string]any{"type": "item", "item": item})
	if _, err := fmt.Fprintln(j.File, string(row)); err != nil {
		return err
	}
	if err := j.File.Sync(); err != nil {
		return err
	}
	j.History = append(j.History, item)
	return nil
}
func (j *Journal) AppendCheckpoint(history []json.RawMessage, level string) error {
	snapshot, _ := json.Marshal(history)
	row, _ := json.Marshal(map[string]any{"type": "checkpoint", "level": level, "item": json.RawMessage(snapshot)})
	if _, err := fmt.Fprintln(j.File, string(row)); err != nil {
		return err
	}
	if err := j.File.Sync(); err != nil {
		return err
	}
	j.History = history
	return nil
}
func (j *Journal) Close() error { return j.File.Close() }

// NewChat preserves the previous journal as a timestamped archive.
func (j *Journal) NewChat() error {
	path := j.File.Name()
	if err := j.File.Close(); err != nil {
		return err
	}
	archive := fmt.Sprintf("%s.%s.bak", path, time.Now().Format("20060102-150405.000000000"))
	if err := os.Rename(path, archive); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	j.File = f
	j.History = nil
	return nil
}
