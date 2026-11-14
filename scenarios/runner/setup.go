package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	phoneNumber = "555-0142"
	fillerMark  = "FILLER-MARKER"
	dockCode    = "HARBOR-19"
	decoyCode   = "HARBOR-00"
	otherLine   = "UNIQUE-OTHER-SENTENCE"
	secretBody  = "BODY-SECRET"
	issueCount  = 100
)

// prepared is the workspace a scenario run starts in.
type prepared struct {
	Root        string
	DataHome    string
	PriorResult string
	Watch       map[string]string
}

func prepare(id, specSetup string) (*prepared, error) {
	data, err := os.MkdirTemp("", "keimling-scenario-data-")
	if err != nil {
		return nil, err
	}
	root := filepath.Join(data, "workspace")
	if err := os.MkdirAll(root, 0o700); err != nil {
		os.RemoveAll(data)
		return nil, err
	}
	p := &prepared{Root: root, DataHome: data, Watch: map[string]string{}}
	switch specSetup {
	case "phone-book":
		err = os.WriteFile(filepath.Join(root, "phonebook.csv"), phoneBookCSV(), 0o600)
	case "unknown-shape":
		err = os.WriteFile(filepath.Join(root, "notes.dat"), unknownShapeFile(), 0o600)
	case "other-chat":
		err = p.writeOtherChat()
	case "transform-list":
		p.PriorResult = filepath.Join(data, "prior.json")
		err = os.WriteFile(p.PriorResult, transformPrior(), 0o600)
	default:
		err = fmt.Errorf("scenario %s has unknown setup %q", id, specSetup)
	}
	if err != nil {
		os.RemoveAll(data)
		return nil, err
	}
	return p, nil
}

func phoneBookCSV() []byte {
	var b strings.Builder
	b.WriteString("name,number,note\n")
	for i := 1; i <= 200; i++ {
		if i == 200 {
			fmt.Fprintf(&b, "Ada,%s,\n", phoneNumber)
			continue
		}
		fmt.Fprintf(&b, "Person-%03d,555-%04d,%s\n", i, i, fillerMark)
	}
	return []byte(b.String())
}

func unknownShapeFile() []byte {
	pad := strings.Repeat("x", 5000)
	return []byte(fmt.Sprintf("{\"pad\":%q,\"pages\":[{\"label\":\"cover\",\"code\":%q},{\"label\":\"dock\",\"code\":%q}]}", pad, decoyCode, dockCode))
}

func transformPrior() []byte {
	people := []string{"ada", "ben", "cy"}
	issues := make([]any, 0, issueCount)
	for i := 1; i <= issueCount; i++ {
		key := fmt.Sprintf("KEY-%03d", i)
		var depends []any
		if i > 1 {
			depends = []any{fmt.Sprintf("KEY-%03d", i-1)}
		}
		issues = append(issues, map[string]any{
			"key": key, "title": fmt.Sprintf("T-%03d", i), "assignee": people[(i-1)%3],
			"depends": depends, "body": secretBody + "-" + key,
		})
	}
	doc := map[string]any{
		"user":      "Look up the open issues.",
		"tool":      "mcp__jira__search",
		"arguments": map[string]string{"jql": "ORDER BY key"},
		"value":     issues,
	}
	b, _ := json.Marshal(doc)
	return b
}

func (p *prepared) writeOtherChat() error {
	root := filepath.Join(p.DataHome, "archive")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	id := workspaceID(root)
	ws := filepath.Join(p.DataHome, "keimling", "workspaces", id)
	thread := filepath.Join(ws, "threads", "notes")
	if err := os.MkdirAll(thread, 0o700); err != nil {
		return err
	}
	meta := map[string]any{"id": id, "name": "Archive", "root": canonical(root), "updated_at": time.Now().UTC()}
	if err := writeJSON(filepath.Join(ws, "meta.json"), meta); err != nil {
		return err
	}
	th := map[string]any{"id": "notes", "title": "Old notes", "updated_at": time.Now().UTC()}
	if err := writeJSON(filepath.Join(thread, "thread.json"), th); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(ws, "active_thread"), []byte("notes\n"), 0o600); err != nil {
		return err
	}
	session := filepath.Join(thread, "session.jsonl")
	item, _ := json.Marshal(map[string]any{"role": "user", "content": otherLine})
	row, _ := json.Marshal(map[string]any{"type": "item", "item": json.RawMessage(item)})
	if err := os.WriteFile(session, append(row, '\n'), 0o600); err != nil {
		return err
	}
	raw, err := os.ReadFile(session)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	p.Watch[session] = hex.EncodeToString(sum[:])
	return nil
}

func unchanged(watch map[string]string) bool {
	for path, before := range watch {
		raw, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != before {
			return false
		}
	}
	return true
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

// workspaceID matches keimling's id for a directory: sha1 of the canonical path, first 8 bytes hex.
func workspaceID(root string) string {
	root = canonical(root)
	home, _ := os.UserHomeDir()
	if home != "" && canonical(home) == root {
		return "global"
	}
	sum := sha1.Sum([]byte(root))
	return hex.EncodeToString(sum[:8])
}

func canonical(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}
