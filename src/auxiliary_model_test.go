package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// replyServer answers every Responses API call with text and records the model
// each request named.
func replyServer(t *testing.T, text string) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var models []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		mu.Lock()
		models = append(models, payload.Model)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "completed",
			"output": []map[string]any{{
				"type":    "message",
				"content": []map[string]any{{"type": "output_text", "text": text}},
			}},
		})
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string{}, models...)
	}
}

func TestAuxiliaryModelPrecedence(t *testing.T) {
	for _, test := range []struct {
		name       string
		flag       string
		configured string
		chat       string
		runChat    string
		want       string
	}{
		{"flag beats everything", "flag-model", "aux-model", "chat-model", "run-model", "flag-model"},
		{"blank flag is not a choice", "   ", "aux-model", "chat-model", "", "aux-model"},
		{"configured beats the chat model", "", "aux-model", "chat-model", "", "aux-model"},
		{"configured beats the run chat model", "", "aux-model", "chat-model", "run-model", "aux-model"},
		{"empty configured uses the chat model", "", "", "chat-model", "", "chat-model"},
		{"empty configured uses the run chat model", "", "", "chat-model", "run-model", "run-model"},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := &Agent{Cfg: Config{AuxiliaryModel: test.configured, Model: test.chat}}
			if test.runChat != "" {
				target := chatTarget{Model: test.runChat}
				a.runChat = &target
			}
			if got := a.auxiliaryModel(test.flag); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestAuxiliaryModelUnknownFallsBackAndLogsOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	fc := fileConfig{
		ModelProviders: []modelProvider{{
			ID:            "p1",
			Name:          "Provider one",
			BaseURL:       "https://example.invalid/v1",
			ActiveModelID: "known-model",
			Models:        []providerModel{{ID: "known-model"}},
		}},
		ActiveModelProviderID: "p1",
	}
	if err := writeFileConfigForTest(path, fc); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var lines []string
	a := &Agent{
		Cfg:  Config{Model: "chat-model", AuxiliaryModel: "ghost-model", ConfigPath: path},
		Emit: func(kind, text string) { mu.Lock(); lines = append(lines, text); mu.Unlock() },
	}
	for i := 0; i < 3; i++ {
		if got := a.auxiliaryModel(""); got != "chat-model" {
			t.Fatalf("unknown model must fall back to the chat model, got %q", got)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(lines) != 1 || !strings.Contains(lines[0], "ghost-model") {
		t.Fatalf("expected one fallback notice, got %#v", lines)
	}
	if got := a.auxiliaryModel("ghost-flag"); got != "ghost-flag" {
		t.Fatalf("an explicit flag is not validated, got %q", got)
	}
}

func TestAuxiliaryModelSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	a := &Agent{Cfg: Config{Model: "chat-model", Mode: "auto", Reasoning: "high", ConfigPath: path, ContextLimit: 128000, MaxSteps: 200}}
	s := &appServer{agent: a}
	saved := "aux-model"
	if err := s.applySettings(agentSettings{AuxiliaryModel: &saved}); err != nil {
		t.Fatal(err)
	}
	fc, err := readConfiguration(path)
	if err != nil {
		t.Fatal(err)
	}
	if fc.AuxiliaryModel != "aux-model" || a.Cfg.AuxiliaryModel != "aux-model" {
		t.Fatalf("saved %q, config %q", a.Cfg.AuxiliaryModel, fc.AuxiliaryModel)
	}
	if got := s.settingsSnapshot()["auxiliary_model"]; got != "aux-model" {
		t.Fatalf("snapshot %v", got)
	}
	// A payload without the field keeps the saved value.
	if err := s.applySettings(agentSettings{}); err != nil {
		t.Fatal(err)
	}
	if a.Cfg.AuxiliaryModel != "aux-model" {
		t.Fatalf("an absent field cleared the setting: %q", a.Cfg.AuxiliaryModel)
	}
	// An empty value means the chat model, and it is stored that way.
	empty := ""
	if err := s.applySettings(agentSettings{AuxiliaryModel: &empty}); err != nil {
		t.Fatal(err)
	}
	fc, err = readConfiguration(path)
	if err != nil || fc.AuxiliaryModel != "" {
		t.Fatalf("stored %q, err %v", fc.AuxiliaryModel, err)
	}
	if got := a.auxiliaryModel(""); got != "chat-model" {
		t.Fatalf("cleared setting must use the chat model, got %q", got)
	}
}

func TestSideTasksUseTheResolvedModel(t *testing.T) {
	ctx := context.Background()

	t.Run("output filter", func(t *testing.T) {
		srv, models := replyServer(t, "only the facts")
		a := &Agent{
			Cfg:    Config{Model: "chat-model", AuxiliaryModel: "aux-model"},
			Client: &Client{BaseURL: srv.URL + "/v1", HTTP: srv.Client()},
		}
		args := `{"command":"go test ./...","output_focus":"failures"}`
		result := "exit_code=0\n" + strings.Repeat("a long output line\n", 20)
		got := a.filterExecResult(ctx, "run the tests", args, result)
		if !strings.Contains(got, "only the facts") {
			t.Fatalf("filter output %q", got)
		}
		if got := models(); len(got) != 1 || got[0] != "aux-model" {
			t.Fatalf("filter models %v", got)
		}
	})

	t.Run("approval review", func(t *testing.T) {
		srv, models := replyServer(t, `{"decision":"approve","reason":"safe"}`)
		a := &Agent{
			Cfg:    Config{Model: "chat-model", AuxiliaryModel: "aux-model"},
			Client: &Client{BaseURL: srv.URL + "/v1", HTTP: srv.Client()},
		}
		if _, err := a.approvalReview(ctx, "exec", `{"command":"ls"}`, "/tmp", "", nil); err != nil {
			t.Fatal(err)
		}
		if got := models(); len(got) != 1 || got[0] != "aux-model" {
			t.Fatalf("review models %v", got)
		}
	})

	t.Run("autopilot", func(t *testing.T) {
		srv, models := replyServer(t, `{"action":"done","message":"verified","reason":"finished"}`)
		a := &Agent{
			Cfg:    Config{Model: "chat-model", AuxiliaryModel: "aux-model", PilotReasoning: "high"},
			Client: &Client{BaseURL: srv.URL + "/v1", HTTP: srv.Client()},
		}
		if _, err := a.pilotReview(ctx, "objective", "report"); err != nil {
			t.Fatal(err)
		}
		if got := models(); len(got) != 1 || got[0] != "aux-model" {
			t.Fatalf("autopilot models %v", got)
		}
	})

	t.Run("fast delegate", func(t *testing.T) {
		a := &Agent{Cfg: Config{Model: "chat-model", AuxiliaryModel: "aux-model", DelegateBalanced: "balanced-model", DelegateDeep: "deep-model"}}
		if model, effort, err := a.delegateProfile("fast"); err != nil || model != "aux-model" || effort != "low" {
			t.Fatalf("%q %q %v", model, effort, err)
		}
		// The balanced and deep profiles keep their own models.
		if model, _, err := a.delegateProfile("balanced"); err != nil || model != "balanced-model" {
			t.Fatalf("%q %v", model, err)
		}
	})

	t.Run("web search", func(t *testing.T) {
		srv, models := replyServer(t, "Result https://example.com")
		a := &Agent{Cfg: Config{
			Model: "chat-model", AuxiliaryModel: "aux-model", Web: true,
			Search: searchConfig{Kind: "openai", URL: srv.URL, APIKey: "search-key"},
		}}
		text, err := a.searchWeb(ctx, "keimling")
		if err != nil || !strings.Contains(text, "example.com") {
			t.Fatalf("%q %v", text, err)
		}
		if got := models(); len(got) != 1 || got[0] != "aux-model" {
			t.Fatalf("search models %v", got)
		}
	})

	t.Run("web search falls back to the chat model", func(t *testing.T) {
		srv, models := replyServer(t, "Result https://example.com")
		a := &Agent{Cfg: Config{
			Model: "chat-model", Web: true,
			Search: searchConfig{Kind: "openai", URL: srv.URL, APIKey: "search-key"},
		}}
		if _, err := a.searchWeb(ctx, "keimling"); err != nil {
			t.Fatal(err)
		}
		if got := models(); len(got) != 1 || got[0] != "chat-model" {
			t.Fatalf("search models %v", got)
		}
	})
}

// writeFileConfigForTest saves a configuration the way the app writes it.
func writeFileConfigForTest(path string, fc fileConfig) error {
	return updateConfiguration(path, func(target *fileConfig) {
		*target = fc
	})
}
