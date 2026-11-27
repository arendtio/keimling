package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A config without voice_model leaves transcription off. Both entry points used
// to inject gpt-transcribe, so this guards the default from coming back.
func TestRuntimeAgentKeepsTranscriptionOffWithoutAVoiceModel(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agent, cancel, err := openRuntimeAgent(cfgPath, ws.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	if got := agent.Cfg.VoiceModel; got != "" {
		t.Fatalf("voice model = %q, want empty", got)
	}
	if _, _, err := agent.voiceClient(); err == nil || !strings.Contains(err.Error(), "voice transcription is off") {
		t.Fatalf("voiceClient error = %v, want transcription off", err)
	}
}

func TestRuntimeAgentUsesASavedVoiceModel(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	cfg := `{"voice_provider_id":"oa","voice_model":"gpt-transcribe",` +
		`"model_providers":[{"id":"oa","name":"OpenAI","kind":"openai","base_url":"https://api.openai.com/v1"}]}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	ws, err := ensureWorkspaceForRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agent, cancel, err := openRuntimeAgent(cfgPath, ws.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	if got := agent.Cfg.VoiceModel; got != "gpt-transcribe" {
		t.Fatalf("voice model = %q, want gpt-transcribe", got)
	}
	if _, model, err := agent.voiceClient(); err != nil || model != "gpt-transcribe" {
		t.Fatalf("voiceClient = %q, %v", model, err)
	}
}
