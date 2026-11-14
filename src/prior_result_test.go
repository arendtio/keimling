package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPriorResultKeepsTheBodyOutOfTheJournal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	j, err := OpenJournal(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &Agent{Journal: j, History: j.History, Cfg: Config{SessionPath: path}}
	body := strings.Repeat("BODY-SECRET-", 400)
	doc := `{"user":"Look up the open issues.","tool":"mcp__jira__search","arguments":{"jql":"ORDER BY key"},"value":[{"key":"KEY-001","body":"` + body + `"}]}`
	file := filepath.Join(dir, "result.json")
	if err := os.WriteFile(file, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := loadPriorResult(a, file); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "BODY-SECRET") || !strings.Contains(string(raw), "r1") {
		t.Fatalf("journal %s", raw)
	}
	found := false
	for _, v := range a.toolResults().exportValues() {
		b, _ := json.Marshal(v)
		if strings.Contains(string(b), "BODY-SECRET") && strings.Contains(string(b), "KEY-001") {
			found = true
		}
	}
	if !found {
		t.Fatal("store lost the result")
	}
	if err := writeResultStore(path, a.toolResults()); err != nil {
		t.Fatal(err)
	}
	dumped, err := os.ReadFile(path + ".results.json")
	if err != nil || !strings.Contains(string(dumped), "KEY-001") {
		t.Fatalf("%v %s", err, dumped)
	}
}
