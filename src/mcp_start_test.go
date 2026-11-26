package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStartScriptFailureConnectsNothing(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	m := NewMCPManager()
	m.setStartScript("false")
	m.Reload(context.Background(), map[string]MCPServerConfig{
		"demo":  {Command: "true", Before: "touch " + marker},
		"quiet": {Command: "true", Disabled: true},
	})
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("before script ran after the start script failed")
	}
	byName := map[string]MCPServerStatus{}
	for _, st := range m.Status() {
		byName[st.Name] = st
	}
	if st := byName["demo"]; st.Status != "error" || !strings.Contains(st.Error, "start script") {
		t.Fatalf("demo %+v", st)
	}
	if st := byName["quiet"]; st.Status != "disabled" {
		t.Fatalf("quiet %+v", st)
	}
	state, msg := m.ScriptNotice()
	if state != "failed" || !strings.Contains(msg, "start script") {
		t.Fatalf("notice %s %s", state, msg)
	}
}

func TestBeforeScriptFailureSkipsThatServer(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	m := NewMCPManager()
	m.setStartScript("true")
	m.Reload(context.Background(), map[string]MCPServerConfig{
		"held": {Command: "true", Before: "false"},
		"ran":  {Command: "true", Before: "touch " + marker},
	})
	if _, err := os.Stat(marker); err != nil {
		t.Fatal(err)
	}
	byName := map[string]MCPServerStatus{}
	for _, st := range m.Status() {
		byName[st.Name] = st
	}
	if st := byName["held"]; st.Status != "error" || !strings.Contains(st.Error, "before script") {
		t.Fatalf("held %+v", st)
	}
	if st := byName["ran"]; strings.Contains(st.Error, "before script") {
		t.Fatalf("ran %+v", st)
	}
	state, msg := m.ScriptNotice()
	if state != "failed" || !strings.Contains(msg, "held") || strings.Contains(msg, "ran") {
		t.Fatalf("notice %s %s", state, msg)
	}
}

func TestScriptNoticeWhileTheStartScriptRuns(t *testing.T) {
	m := NewMCPManager()
	m.setStartScript("sleep 5")
	done := make(chan struct{})
	go func() {
		m.Reload(context.Background(), map[string]MCPServerConfig{
			"quiet": {Command: "true", Disabled: true},
		})
		close(done)
	}()
	deadline := time.Now().Add(4 * time.Second)
	var state, msg string
	for time.Now().Before(deadline) {
		state, msg = m.ScriptNotice()
		if state == "running" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if state != "running" || !strings.Contains(msg, "still running") {
		t.Fatalf("while running %s %s", state, msg)
	}
	<-done
	state, msg = m.ScriptNotice()
	if state != "" || msg != "" {
		t.Fatalf("after %s %s", state, msg)
	}
}

func TestStartScriptRunsOncePerProcess(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "runs")
	cmd := "printf x >> " + marker
	configs := map[string]MCPServerConfig{"quiet": {Command: "true", Disabled: true}}
	first := NewMCPManager()
	first.setStartScript(cmd)
	first.ConnectAll(context.Background(), configs)
	second := NewMCPManager()
	second.setStartScript(cmd)
	second.ConnectAll(context.Background(), configs)
	b, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "x" {
		t.Fatalf("start script ran %d times across workspaces", len(b))
	}
	if err := second.runStart(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(marker)
	if string(b) != "xx" {
		t.Fatalf("force did not run the start script again: %d", len(b))
	}
}

func TestMCPScriptTimeoutIsFiveMinutes(t *testing.T) {
	if mcpScriptTimeout != 5*time.Minute {
		t.Fatal(mcpScriptTimeout)
	}
}

func TestPluginBeforeStaysOutOfThePlugin(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := writeMCPBeforeOverlay("plug", "echo hi"); err != nil {
		t.Fatal(err)
	}
	configs := map[string]MCPServerConfig{
		"plug": {URL: "http://example"},
		"mine": {URL: "http://example"},
	}
	sources := map[string]string{"plug": "plugin", "mine": "user"}
	applyMCPBefore(configs, sources)
	if configs["plug"].Before != "echo hi" {
		t.Fatalf("plugin %#v", configs["plug"])
	}
	if configs["mine"].Before != "" {
		t.Fatalf("user %#v", configs["mine"])
	}
}

func TestKeepBeforeWhenTheEditOmitsIt(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := UpsertUserMCPServer("demo", MCPServerConfig{Command: "echo", Before: "echo keep"}); err != nil {
		t.Fatal(err)
	}
	kept := keepUserMCPFields("demo", MCPServerConfig{Command: "echo"}, false, false, false, false)
	if kept.Before != "echo keep" {
		t.Fatalf("%#v", kept)
	}
	replaced := keepUserMCPFields("demo", MCPServerConfig{Command: "echo", Before: "echo new"}, false, false, true, false)
	if replaced.Before != "echo new" {
		t.Fatalf("%#v", replaced)
	}
}
