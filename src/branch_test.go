package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func histBlob(a *Agent) string {
	b, _ := json.Marshal(a.History)
	return string(b)
}

func planStep(a *Agent, step string) bool {
	if a.Charter == nil {
		return false
	}
	for _, s := range a.Charter.Plan {
		if s.Step == step {
			return true
		}
	}
	return false
}

func TestChatBranchesKeepPrefixAndCharter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	note := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(note, []byte("keep-me"), 0600); err != nil {
		t.Fatal(err)
	}
	j, err := OpenJournal(path, true)
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Journal: j,
		History: j.History,
		Cfg:     Config{SessionPath: path},
		Charter: &Charter{Version: charterVersion, Goal: "ship"},
	}
	if err := a.persistCharter(); err != nil {
		t.Fatal(err)
	}
	if err := a.addBranchableUser("ALPHA-PREFIX"); err != nil {
		t.Fatal(err)
	}
	if err := a.add(map[string]any{"role": "assistant", "content": "reply-one"}); err != nil {
		t.Fatal(err)
	}
	a.Charter.Plan = []CharterStep{{Step: "later", Status: "pending"}}
	if err := a.persistCharter(); err != nil {
		t.Fatal(err)
	}
	rows, err := readJournalRows(path)
	if err != nil {
		t.Fatal(err)
	}
	source := 0
	for _, ln := range rows {
		if strings.Contains(string(ln.Item), "ALPHA-PREFIX") {
			source = ln.Line
		}
	}
	if source == 0 {
		t.Fatal("missing source line")
	}
	if err := a.prepareFork(source, "BETA-NEXT"); err != nil {
		t.Fatal(err)
	}
	if planStep(a, "later") {
		t.Fatal("new continuation kept the later plan")
	}
	a.Charter.Plan = []CharterStep{{Step: "other", Status: "pending"}}
	if err := a.persistCharter(); err != nil {
		t.Fatal(err)
	}
	if err := a.prepareFork(source, "GAMMA-THIRD"); err != nil {
		t.Fatal(err)
	}
	if planStep(a, "later") || planStep(a, "other") {
		t.Fatalf("third continuation saw a later plan: %+v", a.Charter.Plan)
	}
	st, err := loadBranchState(path)
	if err != nil {
		t.Fatal(err)
	}
	g := st.Forks[lineKey(source)]
	if g == nil || len(g.IDs) != 3 {
		t.Fatalf("siblings %#v", g)
	}
	if err := a.selectContinuation(source, 0); err != nil {
		t.Fatal(err)
	}
	blob := histBlob(a)
	if !strings.Contains(blob, "ALPHA-PREFIX") || !strings.Contains(blob, "reply-one") || strings.Contains(blob, "BETA-NEXT") || strings.Contains(blob, "GAMMA-THIRD") {
		t.Fatal(blob)
	}
	if !planStep(a, "later") {
		t.Fatal("original plan missing")
	}
	if err := a.selectContinuation(source, 1); err != nil {
		t.Fatal(err)
	}
	blob = histBlob(a)
	if !strings.Contains(blob, "BETA-NEXT") || strings.Contains(blob, "ALPHA-PREFIX") || strings.Contains(blob, "reply-one") || strings.Contains(blob, "GAMMA-THIRD") {
		t.Fatal(blob)
	}
	if !planStep(a, "other") {
		t.Fatalf("second plan missing: %+v", a.Charter.Plan)
	}
	if err := a.selectContinuation(source, 2); err != nil {
		t.Fatal(err)
	}
	blob = histBlob(a)
	if !strings.Contains(blob, "GAMMA-THIRD") || strings.Contains(blob, "BETA-NEXT") || strings.Contains(blob, "ALPHA-PREFIX") {
		t.Fatal(blob)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), "ALPHA-PREFIX") != 1 {
		t.Fatalf("prefix copied: %d", strings.Count(string(raw), "ALPHA-PREFIX"))
	}
	idx, err := os.ReadFile(branchIndexPath(path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(idx), "ALPHA-PREFIX") {
		t.Fatal("branch index stored the message text")
	}
	kept, err := os.ReadFile(note)
	if err != nil {
		t.Fatal(err)
	}
	if string(kept) != "keep-me" {
		t.Fatalf("workspace file changed: %q", kept)
	}
	if err := a.prepareFork(source+1, "nope"); err == nil {
		t.Fatal("forked a reply")
	}
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := OpenJournal(path, true)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := json.Marshal(again.History)
	if strings.Contains(string(root), "BETA-NEXT") || strings.Contains(string(root), "GAMMA-THIRD") {
		t.Fatal("resume loaded a continuation into the root view")
	}
	if !strings.Contains(string(root), "ALPHA-PREFIX") {
		t.Fatal("root transcript missing")
	}
	a.Journal = again
	a.History = again.History
	a.restoreActiveBranch()
	blob = histBlob(a)
	if !strings.Contains(blob, "GAMMA-THIRD") || strings.Contains(blob, "ALPHA-PREFIX") {
		t.Fatal(blob)
	}
	if planStep(a, "later") {
		t.Fatal("restored continuation kept the original later plan")
	}
	items, lines, err := visibleJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	msgs := sessionMessages(items, 120)
	annotateBranchMessages(path, msgs, lines)
	var user *sessionMessage
	for i := range msgs {
		if msgs[i].Role == "user" {
			user = &msgs[i]
		}
	}
	if user == nil || user.Text != "GAMMA-THIRD" || user.BranchCount != 3 || user.BranchLine != source || user.BranchIndex != 2 {
		t.Fatalf("visible %+v", user)
	}
	defer again.Close()
}

func TestOpenJournalSkipsBranchCheckpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	j, err := OpenJournal(path, true)
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Journal: j, History: j.History, Cfg: Config{SessionPath: path}, Charter: &Charter{Version: charterVersion}}
	if err := a.add(map[string]any{"role": "user", "content": "ROOT-MSG"}); err != nil {
		t.Fatal(err)
	}
	j.Branch = "b9"
	item, _ := json.Marshal(map[string]any{"role": "assistant", "content": "SECRET-SNAP"})
	if err := j.AppendCheckpoint([]json.RawMessage{item}, "task"); err != nil {
		t.Fatal(err)
	}
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := OpenJournal(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	blob, _ := json.Marshal(again.History)
	if strings.Contains(string(blob), "SECRET-SNAP") {
		t.Fatal("tagged checkpoint replaced the root view")
	}
	if !strings.Contains(string(blob), "ROOT-MSG") {
		t.Fatal("root message missing")
	}
}
