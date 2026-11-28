package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseModelCatalogOpenRouterShape(t *testing.T) {
	raw := []byte(`{"data":[{"id":"acme/fast","name":"Fast","context_length":200000,"pricing":{"prompt":"0.000001","completion":"0.000002"}},{"id":"plain"}]}`)
	got, err := parseModelCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%#v", got)
	}
	if got[0].ContextLimit != 200000 || got[0].PromptPrice != "0.000001" {
		t.Fatalf("%#v", got[0])
	}
	if got[1].ID != "plain" || got[1].ContextLimit != 0 {
		t.Fatalf("%#v", got[1])
	}
}

func TestParseModelCatalogCapabilities(t *testing.T) {
	raw := []byte(`{"data":[
		{"id":"see","architecture":{"input_modalities":["text","image"],"output_modalities":["text"]}},
		{"id":"hear","architecture":{"input_modalities":["text","audio"],"output_modalities":["text"]}},
		{"id":"stt","architecture":{"input_modalities":["audio"],"output_modalities":["transcription"]},"endpoints":["/api/v1/audio/transcriptions"]},
		{"id":"tts","architecture":{"input_modalities":["text"],"output_modalities":["speech"]},"endpoints":["/api/v1/audio/speech"]},
		{"id":"live","endpoints":["/api/v1/realtime"]}
	]}`)
	got, err := parseModelCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]catalogModel{
		"see":  {SeesImages: true},
		"hear": {HearsAudio: true},
		"stt":  {Transcribes: true},
		"tts":  {Speaks: true},
		"live": {Realtime: true, CapabilitiesKnown: true},
	}
	if len(got) != len(want) {
		t.Fatalf("%#v", got)
	}
	for _, m := range got {
		w := want[m.ID]
		if m.SeesImages != w.SeesImages || m.HearsAudio != w.HearsAudio || m.Transcribes != w.Transcribes || m.Speaks != w.Speaks || m.Realtime != w.Realtime || !m.CapabilitiesKnown {
			t.Fatalf("%s %#v", m.ID, m)
		}
	}
}

func TestParseModelCatalogNumericPrice(t *testing.T) {
	raw := []byte(`{"data":[{"id":"n","context_length":128000,"pricing":{"prompt":0.000001,"completion":0.000002}}]}`)
	got, err := parseModelCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "n" || got[0].ContextLimit != 128000 {
		t.Fatalf("%#v", got)
	}
	pin, ok := parseUSDPerToken(got[0].PromptPrice)
	if !ok || pin != 0.000001 {
		t.Fatalf("prompt %q", got[0].PromptPrice)
	}
}

func TestParseModelCatalogModelsDevShape(t *testing.T) {
	raw := []byte(`{"data":[
		{"id":"paid","limit":{"context":200000,"output":32000},"cost":{"text":{"input":{"price":3},"output":{"price":15}}}},
		{"id":"free","limit":{"context":128000},"cost":{"text":{"input":{"price":0},"output":{"price":0}}}}
	]}`)
	got, err := parseModelCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ContextLimit != 200000 {
		t.Fatalf("%#v", got)
	}
	if pin, ok := parseUSDPerToken(got[0].PromptPrice); !ok || pin != 0.000003 {
		t.Fatalf("prompt %q", got[0].PromptPrice)
	}
	if pout, ok := parseUSDPerToken(got[0].CompletionPrice); !ok || pout != 0.000015 {
		t.Fatalf("completion %q", got[0].CompletionPrice)
	}
	if got[1].ContextLimit != 128000 || got[1].PromptPrice != "0" {
		t.Fatalf("%#v", got[1])
	}
}

func TestLiteLLMCatalogUsesModelInfo(t *testing.T) {
	var modelsQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Fatalf("auth %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/v1/models":
			modelsQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"data":[{"id":"vision-model"}]}`))
		case "/model_group/info":
			http.Error(w, "no", http.StatusNotFound)
		case "/model/info":
			_, _ = w.Write([]byte(`{"data":[{"model_name":"vision-model","model_info":{"max_input_tokens":128000,"input_cost_per_token":0.000002,"output_cost_per_token":0.000008,"supports_vision":true}}]}`))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	got, err := fetchProviderModelCatalog(context.Background(), srv.URL+"/v1", "tok", "litellm")
	if err != nil {
		t.Fatal(err)
	}
	if modelsQuery != "" {
		t.Fatalf("non-openrouter query %q", modelsQuery)
	}
	if len(got) != 1 || !got[0].SeesImages || got[0].ContextLimit != 128000 {
		t.Fatalf("%#v", got)
	}
	pin, ok := parseUSDPerToken(got[0].PromptPrice)
	if !ok || pin != 0.000002 {
		t.Fatalf("prompt %q", got[0].PromptPrice)
	}
}

func TestLiteLLMCatalogKeepsNamesWhenInfoIsForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/model/info", "/model_group/info":
			http.Error(w, "no", http.StatusForbidden)
		case "/v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"plain"}]}`))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	got, err := fetchProviderModelCatalog(context.Background(), srv.URL+"/v1", "tok", "litellm")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "plain" || got[0].SeesImages || got[0].ContextLimit != 0 {
		t.Fatalf("%#v", got)
	}
}

// A LiteLLM proxy usually listens on localhost with kind "auto", which resolves
// to "local". Prefill still has to read the proxy's model metadata there.
func TestLiteLLMCatalogReadsModelGroupInfo(t *testing.T) {
	var gotGroupPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4o","max_input_tokens":128000}]}`))
		case "/model_group/info":
			gotGroupPath = r.URL.Path
			_, _ = w.Write([]byte(`{"data":[{"model_group":"gpt-4o","description":"GPT-4o","max_input_tokens":128000.0,"max_output_tokens":16384.0,"input_cost_per_token":2.5e-06,"output_cost_per_token":1e-05,"mode":"chat","supports_vision":true}]}`))
		case "/model/info":
			http.Error(w, "no", http.StatusNotFound)
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	got, err := fetchProviderModelCatalog(context.Background(), srv.URL+"/v1", "tok", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if gotGroupPath != "/model_group/info" {
		t.Fatalf("group info path %q", gotGroupPath)
	}
	if len(got) != 1 || got[0].ID != "gpt-4o" || got[0].Name != "GPT-4o" {
		t.Fatalf("%#v", got)
	}
	if got[0].ContextLimit != 128000 || !got[0].SeesImages {
		t.Fatalf("%#v", got[0])
	}
	pin, ok := parseUSDPerToken(got[0].PromptPrice)
	if !ok || pin != 0.0000025 {
		t.Fatalf("prompt %q", got[0].PromptPrice)
	}
	pout, ok := parseUSDPerToken(got[0].CompletionPrice)
	if !ok || pout != 0.00001 {
		t.Fatalf("completion %q", got[0].CompletionPrice)
	}
}

func TestLiteLLMCatalogModelInfoAddsAudio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"voice-model"}]}`))
		case "/model_group/info":
			http.Error(w, "no", http.StatusNotFound)
		case "/model/info":
			_, _ = w.Write([]byte(`{"data":[{"model_name":"voice-model","model_info":{"max_input_tokens":32000,"supports_audio_input":true}}]}`))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	got, err := fetchProviderModelCatalog(context.Background(), srv.URL+"/v1", "tok", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].HearsAudio || got[0].ContextLimit != 32000 {
		t.Fatalf("%#v", got)
	}
}

// OpenRouter keeps its own catalog: no LiteLLM management route is probed.
func TestOpenRouterCatalogSkipsLiteLLMRoutes(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			gotQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"data":[{"id":"acme/fast","context_length":200000,"pricing":{"prompt":"0.000001","completion":"0.000002"},"architecture":{"input_modalities":["text","image"],"output_modalities":["speech"]}},{"id":"acme/chat","architecture":{"output_modalities":["text"]}}]}`))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	got, err := fetchProviderModelCatalog(context.Background(), srv.URL+"/v1", "tok", "openrouter")
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery != "output_modalities=all" {
		t.Fatalf("query %q", gotQuery)
	}
	if len(got) != 2 || got[0].ID != "acme/fast" || !got[0].SeesImages || got[0].ContextLimit != 200000 || got[0].Outputs != "speech" || !got[0].Speaks {
		t.Fatalf("%#v", got)
	}
	if got[1].Outputs != "" {
		t.Fatalf("text output label %q", got[1].Outputs)
	}
}

func TestLiteLLMCatalogModelInfoBeatsPlaceholderPrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"priced"}]}`))
		case "/model_group/info":
			_, _ = w.Write([]byte(`{"data":[{"model_group":"priced","input_cost_per_token":0.0,"output_cost_per_token":0.0}]}`))
		case "/model/info":
			_, _ = w.Write([]byte(`{"data":[{"model_name":"priced","model_info":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000009}}]}`))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	got, err := fetchProviderModelCatalog(context.Background(), srv.URL+"/v1", "tok", "litellm")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("%#v", got)
	}
	if pin, ok := parseUSDPerToken(got[0].PromptPrice); !ok || pin != 0.000003 {
		t.Fatalf("prompt %q", got[0].PromptPrice)
	}
	if pout, ok := parseUSDPerToken(got[0].CompletionPrice); !ok || pout != 0.000009 {
		t.Fatalf("completion %q", got[0].CompletionPrice)
	}
}

func TestFetchProviderModelCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			if r.Header.Get("Authorization") != "Bearer tok" {
				t.Fatalf("auth %q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"m1","context_window":8192}]}`))
		case "/model_group/info", "/model/info":
			http.Error(w, "no", http.StatusNotFound)
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	models, err := fetchProviderModelCatalog(context.Background(), srv.URL+"/v1", "tok", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "m1" || models[0].ContextLimit != 8192 {
		t.Fatalf("%#v", models)
	}
}

func TestModelCatalogUsesFormKeyAndURL(t *testing.T) {
	var gotAuth, gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			gotAuth = r.Header.Get("Authorization")
			gotPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"data":[{"id":"gpt-form"}]}`))
		case "/model_group/info", "/model/info":
			http.Error(w, "no", http.StatusNotFound)
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := updateConfiguration(cfgPath, func(fc *fileConfig) {
		fc.ModelProviders = []modelProvider{{
			ID: "or", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", APIKey: "sk-or-stored",
		}}
		fc.ActiveModelProviderID = "or"
		fc.API.Key = "sk-or-stored"
	}); err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Cfg:    Config{ConfigPath: cfgPath, Key: "sk-or-stored"},
		Client: &Client{APIKey: "sk-or-stored"},
	}
	h := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a}).Handler()

	body := `{"base_url":"` + upstream.URL + `/v1","api_key":"sk-form-key","provider_id":"or"}`
	req := httptest.NewRequest(http.MethodPost, "/api/settings/model-catalog", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog: %d %s", rec.Code, rec.Body.String())
	}
	if gotAuth != "Bearer sk-form-key" {
		t.Fatalf("upstream auth %q", gotAuth)
	}
	if gotPath != "/v1/models" {
		t.Fatalf("upstream path %q", gotPath)
	}
	if strings.Contains(rec.Body.String(), "sk-form-key") || strings.Contains(rec.Body.String(), "sk-or-stored") {
		t.Fatalf("response leaked a key: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "gpt-form") {
		t.Fatalf("models: %s", rec.Body.String())
	}
}

func TestModelCatalogBlankKeyKeepsEditedProvider(t *testing.T) {
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"gpt-saved"}]}`))
	}))
	defer upstream.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := updateConfiguration(cfgPath, func(fc *fileConfig) {
		fc.ModelProviders = []modelProvider{
			{ID: "or", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", APIKey: "sk-or-stored"},
			{ID: "oa", Name: "OpenAI", BaseURL: "https://api.openai.com/v1", APIKey: "sk-saved"},
		}
		fc.ActiveModelProviderID = "or"
		fc.API.Key = "sk-or-stored"
	}); err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Cfg:    Config{ConfigPath: cfgPath, Key: "sk-or-stored"},
		Client: &Client{APIKey: "sk-or-stored"},
	}
	h := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a}).Handler()

	body := `{"base_url":"` + upstream.URL + `/v1","provider_id":"oa"}`
	req := httptest.NewRequest(http.MethodPost, "/api/settings/model-catalog", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog: %d %s", rec.Code, rec.Body.String())
	}
	if gotAuth != "Bearer sk-saved" {
		t.Fatalf("upstream auth %q", gotAuth)
	}
}

func TestModelCatalogDoesNotBorrowAnotherKey(t *testing.T) {
	called := false
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"local"}]}`))
	}))
	defer upstream.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := updateConfiguration(cfgPath, func(fc *fileConfig) {
		fc.ModelProviders = []modelProvider{{
			ID: "or", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", APIKey: "sk-or-stored",
		}}
		fc.ActiveModelProviderID = "or"
		fc.API.Key = "sk-or-stored"
	}); err != nil {
		t.Fatal(err)
	}
	a := &Agent{
		Cfg:    Config{ConfigPath: cfgPath, Key: "sk-or-stored"},
		Client: &Client{APIKey: "sk-or-stored"},
	}
	h := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a}).Handler()

	body := `{"base_url":"` + upstream.URL + `/v1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/settings/model-catalog", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog: %d %s", rec.Code, rec.Body.String())
	}
	if !called {
		t.Fatal("upstream not called")
	}
	if gotAuth != "" || strings.Contains(gotAuth, "sk-or-stored") {
		t.Fatalf("borrowed key: %q", gotAuth)
	}
}
