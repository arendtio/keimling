package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReasoningForChatKeepsBlobForOwnerAndTextForOthers(t *testing.T) {
	raw := json.RawMessage(`{"type":"reasoning","id":"rs_1","encrypted_content":"opaque","summary":[],"content":[{"type":"reasoning_text","text":"thought"}],"keimling_provider":"p1","keimling_model":"m1"}`)
	original := append(json.RawMessage(nil), raw...)

	same, err := normalizeResponseInput(prepareReasoningInput([]json.RawMessage{raw}, "p1", "m1", nil))
	if err != nil {
		t.Fatal(err)
	}
	sameJoined := joinRaw(same)
	if !bytes.Contains(sameJoined, []byte(`"encrypted_content":"opaque"`)) || !bytes.Contains(sameJoined, []byte("thought")) {
		t.Fatalf("owner call: %s", sameJoined)
	}
	if bytes.Contains(sameJoined, []byte("keimling_")) || bytes.Contains(reasoningItems(same), []byte("thought")) {
		t.Fatalf("reasoning item carried text: %s", sameJoined)
	}

	other, err := normalizeResponseInput(prepareReasoningInput([]json.RawMessage{raw}, "p2", "m2", nil))
	if err != nil {
		t.Fatal(err)
	}
	otherJoined := joinRaw(other)
	if bytes.Contains(otherJoined, []byte("opaque")) || bytes.Contains(otherJoined, []byte(`"type":"reasoning"`)) {
		t.Fatalf("foreign call kept reasoning item: %s", otherJoined)
	}
	if !bytes.Contains(otherJoined, []byte("thought")) {
		t.Fatalf("foreign call dropped text: %s", otherJoined)
	}
	if !bytes.Equal(raw, original) {
		t.Fatal("history item was edited")
	}
}

func TestLlamaReasoningTextIsNotSentAsReasoningContent(t *testing.T) {
	raw := json.RawMessage(`{"type":"reasoning","id":"rs_xd8h","summary":[],"content":[{"text":"The user said Hi","type":"reasoning_text"}],"encrypted_content":""}`)
	out, err := normalizeResponseInput(prepareReasoningInput([]json.RawMessage{raw}, "openai", "luna", nil))
	if err != nil {
		t.Fatal(err)
	}
	joined := joinRaw(out)
	if bytes.Contains(joined, []byte(`"type":"reasoning"`)) || bytes.Contains(joined, []byte(`"content":[`)) {
		t.Fatalf("content array sent: %s", joined)
	}
	if !bytes.Contains(joined, []byte("The user said Hi")) {
		t.Fatalf("dropped text: %s", joined)
	}
}

func reasoningItems(items []json.RawMessage) []byte {
	var out []byte
	for _, item := range items {
		if bytes.Contains(item, []byte(`"type":"reasoning"`)) {
			out = append(out, item...)
		}
	}
	return out
}

func TestChatModelChangeNoticesOmittedBlobsAndRestoresThem(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	const wsID, thID = "ws-reason", "th-reason"
	if err := writeThreadMeta(wsID, ThreadMeta{ID: thID, Title: "Reason"}); err != nil {
		t.Fatal(err)
	}
	fc := fileConfig{}
	fc.ActiveModelProviderID = "p1"
	fc.ModelProviders = []modelProvider{
		{ID: "p1", Name: "One", BaseURL: "https://one.example/v1", APIKey: "k1", ActiveModelID: "m1", Models: []providerModel{{ID: "m1", Name: "Model One"}}},
		{ID: "p2", Name: "Two", BaseURL: "https://two.example/v1", APIKey: "k2", ActiveModelID: "m2", Models: []providerModel{{ID: "m2", Name: "Model Two"}}},
	}
	writeModelConfig(t, cfgPath, fc)
	journalPath := filepath.Join(t.TempDir(), "session.jsonl")
	j, err := OpenJournal(journalPath, false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &Agent{
		Cfg: Config{
			ConfigPath: cfgPath, WorkspaceID: wsID, ThreadID: thID,
			Model: "m1", BaseURL: "https://one.example/v1", Key: "k1",
		},
		Client:  &Client{BaseURL: "https://one.example/v1", APIKey: "k1"},
		Journal: j,
	}
	if err := a.add(map[string]any{"role": "user", "content": "hello"}); err != nil {
		t.Fatal(err)
	}
	if err := a.add(json.RawMessage(`{"type":"reasoning","id":"rs_1","encrypted_content":"opaque","summary":[],"content":[{"type":"reasoning_text","text":"thought"}]}`)); err != nil {
		t.Fatal(err)
	}

	if err := a.saveThreadChatModel("p2", "m2"); err != nil {
		t.Fatal(err)
	}
	if n := noticeCount(a.History); n != 1 {
		t.Fatalf("notices after switch: %d", n)
	}
	if !bytes.Contains(joinRaw(a.History), []byte("opaque")) {
		t.Fatal("blob left the active context")
	}
	foreign := prepareReasoningInput(a.modelInput(), "p2", "m2", a.reasoningOwnerMap())
	if bytes.Contains(joinRaw(foreign), []byte("opaque")) {
		t.Fatal("foreign model received the blob")
	}
	if !bytes.Contains(joinRaw(foreign), []byte("thought")) {
		t.Fatal("foreign model lost the reasoning text")
	}

	reloaded, err := OpenJournal(journalPath, true)
	if err != nil {
		t.Fatal(err)
	}
	defer reloaded.Close()
	if !bytes.Contains(joinRaw(reloaded.History), []byte("opaque")) {
		t.Fatal("reload dropped the blob")
	}

	if err := a.saveThreadChatModel("p1", "m1"); err != nil {
		t.Fatal(err)
	}
	if n := noticeCount(a.History); n != 1 {
		t.Fatalf("notices after switch back: %d", n)
	}
	home := prepareReasoningInput(a.modelInput(), "p1", "m1", a.reasoningOwnerMap())
	if !bytes.Contains(joinRaw(home), []byte("opaque")) || !bytes.Contains(joinRaw(home), []byte("thought")) {
		t.Fatal("original model did not receive the blob")
	}

	if err := a.Journal.AppendCheckpoint([]json.RawMessage{json.RawMessage(`{"role":"user","content":"hello"}`)}, "semantic"); err != nil {
		t.Fatal(err)
	}
	a.History = a.Journal.History
	if err := a.saveThreadChatModel("p2", "m2"); err != nil {
		t.Fatal(err)
	}
	if err := a.saveThreadChatModel("p1", "m1"); err != nil {
		t.Fatal(err)
	}
	gone := prepareReasoningInput(a.modelInput(), "p1", "m1", a.reasoningOwnerMap())
	if bytes.Contains(joinRaw(gone), []byte("opaque")) {
		t.Fatal("dropped span was restored")
	}
}

func joinRaw(items []json.RawMessage) []byte {
	var out []byte
	for _, item := range items {
		out = append(out, item...)
	}
	return out
}

func TestChatModelChangeWithoutBlobsIsSilent(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	const wsID, thID = "ws-quiet", "th-quiet"
	if err := writeThreadMeta(wsID, ThreadMeta{ID: thID}); err != nil {
		t.Fatal(err)
	}
	fc := fileConfig{}
	fc.ActiveModelProviderID = "p1"
	fc.ModelProviders = []modelProvider{
		{ID: "p1", Name: "One", BaseURL: "https://one.example/v1", ActiveModelID: "m1", Models: []providerModel{{ID: "m1", Name: "One"}}},
		{ID: "p2", Name: "Two", BaseURL: "https://two.example/v1", ActiveModelID: "m2", Models: []providerModel{{ID: "m2", Name: "Two"}}},
	}
	writeModelConfig(t, cfgPath, fc)
	j, err := OpenJournal(filepath.Join(t.TempDir(), "session.jsonl"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &Agent{
		Cfg:     Config{ConfigPath: cfgPath, WorkspaceID: wsID, ThreadID: thID},
		Journal: j,
	}
	if err := a.add(map[string]any{"role": "user", "content": "hello"}); err != nil {
		t.Fatal(err)
	}
	if err := a.saveThreadChatModel("p2", "m2"); err != nil {
		t.Fatal(err)
	}
	if n := noticeCount(a.History); n != 0 {
		t.Fatalf("notices: %d", n)
	}
}

func TestAgentReasoningOwnersStayOnTheInstance(t *testing.T) {
	withTempAgentHomes(t)
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := createAgentRecipe("Owner", "Keep notes on the instance.")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := createAgentInstance(recipe, ws.ID, "Owner 1")
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{Cfg: Config{WorkspaceID: ws.ID, ThreadID: "main", AgentInstanceID: inst.ID}}
	a.persistReasoningOwners(map[string]modelPair{"rs_1": {Provider: "p", Model: "m"}})
	if _, err := os.Stat(threadDir(ws.ID, "main")); !os.IsNotExist(err) {
		t.Fatalf("agent created a workspace thread: %v", err)
	}
	meta, err := readThreadMetaFile(filepath.Join(agentInstanceThreadDir(inst.ID), "thread.json"))
	if err != nil || len(meta.ReasoningOwners) != 1 || meta.ReasoningOwners[0].Key != "rs_1" {
		t.Fatalf("instance meta: %v %#v", err, meta)
	}
}

func noticeCount(history []json.RawMessage) int {
	n := 0
	for _, msg := range sessionMessages(history, 0) {
		if msg.Role == "system" && msg.Text == encryptedReasoningOmittedNotice {
			n++
		}
	}
	return n
}
