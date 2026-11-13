package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDelegateProfiles(t *testing.T) {
	c := Config{DelegateFast: "small", DelegateBalanced: "regular", DelegateDeep: "large"}
	for _, test := range []struct{ p, m, e string }{{"fast", "small", "low"}, {"balanced", "regular", "medium"}, {"deep", "large", "high"}} {
		m, e, err := delegateProfile(c, test.p)
		if err != nil || m != test.m || e != test.e {
			t.Fatalf("%s: %s %s %v", test.p, m, e, err)
		}
	}
	if _, _, err := delegateProfile(c, "unknown"); err == nil {
		t.Fatal("unknown profile accepted")
	}
}
func TestDelegateIsolatedAndCounted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p map[string]json.RawMessage
		if json.Unmarshal(body, &p) != nil {
			t.Error("invalid JSON")
		}
		if _, ok := p["tools"]; ok {
			t.Error("delegate must not have tools")
		}
		if string(p["tool_choice"]) != "\"none\"" {
			t.Errorf("tool choice %s", p["tool_choice"])
		}
		if strings.Contains(string(body), "previous chat") {
			t.Error("history leaked")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"Found two issues"}]}],"usage":{"input_tokens":120,"output_tokens":22}}`))
	}))
	defer server.Close()
	c := NewClient("test-key")
	c.HTTP = &http.Client{Transport: rewriteTransport{endpoint: server.URL, base: http.DefaultTransport}}
	answer, err := c.Delegate(context.Background(), "mock-model", "medium", DelegateArgs{Task: "Review config", Context: "unit content", Profile: "balanced"}, nil)
	if err != nil || answer != "Found two issues" {
		t.Fatalf("answer %q err %v", answer, err)
	}
	if c.InputTokens != 120 || c.OutputTokens != 22 {
		t.Fatalf("usage %d %d", c.InputTokens, c.OutputTokens)
	}
}

func TestDelegateImageUsesTheSeeingModelOnce(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "shot.png")
	pixels := []byte("pixel-unique-vision-bytes")
	if err := os.WriteFile(img, pixels, 0o644); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(note, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	encoded := base64.StdEncoding.EncodeToString(pixels)
	var hits int
	var lastAuth string
	var lastBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		body, _ := io.ReadAll(r.Body)
		lastAuth = r.Header.Get("Authorization")
		lastBody = string(body)
		var p map[string]json.RawMessage
		if json.Unmarshal(body, &p) != nil {
			t.Error("invalid JSON")
		}
		if _, ok := p["tools"]; ok {
			t.Error("delegate must not have tools")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"A green sprout"}]}],"usage":{"input_tokens":3,"output_tokens":2}}`))
	}))
	defer server.Close()
	a := &Agent{
		Client: NewClient("settings-key"),
		Cfg:    Config{DelegateFast: "text-model", DelegateBalanced: "balanced-model", DelegateDeep: "deep-model"},
	}
	a.Client.BaseURL = "http://127.0.0.1:1/v1"
	a.runChat = &chatTarget{ProviderID: "chat", Model: "see-model", BaseURL: server.URL, APIKey: "chat-key", Reasoning: "low", SeeImages: true}

	answer, err := a.runDelegate(context.Background(), DelegateArgs{Task: "What is shown?", Context: "none", Profile: "fast", Image: img})
	if err != nil || answer != "A green sprout" {
		t.Fatalf("answer %q err %v", answer, err)
	}
	if hits != 1 || !strings.Contains(lastBody, encoded) || !strings.Contains(lastBody, "input_image") || !strings.Contains(lastBody, "What is shown?") {
		t.Fatalf("hits %d body %s", hits, lastBody)
	}
	if strings.Contains(lastBody, "text-model") || lastAuth != "Bearer chat-key" || strings.Contains(lastBody, "old conversation") {
		t.Fatalf("auth %s body %s", lastAuth, lastBody)
	}
	if strings.Contains(answer, encoded) {
		t.Fatal("the text answer contains the image")
	}
	if a.Client.APIKey != "settings-key" || a.Client.BaseURL != "http://127.0.0.1:1/v1" {
		t.Fatalf("client endpoint not restored: %s %s", a.Client.BaseURL, a.Client.APIKey)
	}

	if _, err := a.runDelegate(context.Background(), DelegateArgs{Task: "read this", Profile: "fast", Image: note}); err == nil || hits != 1 {
		t.Fatalf("text file err %v hits %d", err, hits)
	}
	a.runChat.SeeImages = false
	if _, err := a.runDelegate(context.Background(), DelegateArgs{Task: "What is shown?", Profile: "fast", Image: img}); err == nil || hits != 1 {
		t.Fatalf("seeing off err %v hits %d", err, hits)
	}

	a.Client.BaseURL = server.URL
	text, err := a.runDelegate(context.Background(), DelegateArgs{Task: "Add one and one", Context: "digits", Profile: "fast"})
	if err != nil || text != "A green sprout" || hits != 2 {
		t.Fatalf("text delegate %q err %v hits %d", text, err, hits)
	}
	if strings.Contains(lastBody, "input_image") || strings.Contains(lastBody, encoded) || !strings.Contains(lastBody, "text-model") || lastAuth != "Bearer settings-key" {
		t.Fatalf("auth %s body %s", lastAuth, lastBody)
	}

	raw, _ := json.Marshal(map[string]any{"role": "user", "content": joinChatFiles("look", []chatFile{{Name: "shot.png", Path: img, Source: "temp"}})})
	main := expandChatFiles([]json.RawMessage{raw}, true, false)
	mainText := string(main[0])
	if strings.Contains(mainText, encoded) || strings.Contains(mainText, "input_image") || !strings.Contains(mainText, img) || !strings.Contains(mainText, imageDelegateHint) {
		t.Fatalf("main input: %s", mainText)
	}
	label := delegateLabel(`{"task":"What is shown?","image":"` + img + `","context":"","profile":"fast"}`)
	if !strings.Contains(label, "image="+img) || !strings.Contains(label, "What is shown?") {
		t.Fatal(label)
	}
}

func TestDelegateImageKeepsTheUserQuestion(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(img, []byte("pixel-unique-vision-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"Ein Ladebildschirm"}]}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()
	ask := "Wieviele Sachen in meiner Lieblingsfarbe siehst Du?"
	a := &Agent{Client: NewClient("settings-key")}
	a.runChat = &chatTarget{ProviderID: "chat", Model: "see-model", BaseURL: server.URL, APIKey: "chat-key", Reasoning: "low", SeeImages: true}
	a.History = []json.RawMessage{
		mustMessage("user", "Meine Lieblingsfarbe ist blau."),
		mustMessage("assistant", "Verstanden."),
		mustMessage("user", joinChatFiles(ask, []chatFile{{Name: "shot.png", Path: img, Source: "temp"}})),
	}
	rewritten := "Describe everything visible in this image in detail, including any text, UI elements, subjects, numbers, and layout."
	answer, err := a.runDelegate(context.Background(), DelegateArgs{Task: rewritten, Context: "the title is DRG", Profile: "fast", Image: img})
	if err != nil || answer != "Ein Ladebildschirm" {
		t.Fatalf("answer %q err %v", answer, err)
	}
	if !strings.Contains(body, ask) || !strings.Contains(body, "Meine Lieblingsfarbe ist blau.") || !strings.Contains(body, "the title is DRG") {
		t.Fatalf("delegate missed the question or the context: %s", body)
	}
	if strings.Contains(body, rewritten) {
		t.Fatal("the rewritten description replaced the user question")
	}
	if !strings.Contains(body, "at the level it was asked") {
		t.Fatal("delegate was not told to keep the question's scope")
	}
}

func mustMessage(role, content string) json.RawMessage {
	raw, err := json.Marshal(map[string]string{"role": role, "content": content})
	if err != nil {
		panic(err)
	}
	return raw
}

func TestDelegateToolRequiresAnImageArgument(t *testing.T) {
	var found bool
	for _, tool := range definitions(false) {
		if tool["name"] != "delegate" {
			continue
		}
		found = true
		params, _ := tool["parameters"].(map[string]any)
		props, _ := params["properties"].(map[string]any)
		if _, ok := props["image"]; !ok {
			t.Fatal("image argument missing")
		}
		required, _ := params["required"].([]string)
		got := map[string]bool{}
		for _, name := range required {
			got[name] = true
		}
		if !got["image"] || !got["task"] || !got["source"] {
			t.Fatalf("required %#v", required)
		}
		desc, _ := tool["description"].(string)
		if !strings.Contains(desc, "user's words") || !strings.Contains(desc, "Do not rewrite") {
			t.Fatal(desc)
		}
	}
	if !found {
		t.Fatal("delegate tool missing")
	}
	if !strings.Contains(sharedWorkInstructions(), "An image is not included as bytes") {
		t.Fatal("work prompt still treats an image as inline content")
	}
}

type rewriteTransport struct {
	endpoint string
	base     http.RoundTripper
}

func TestDelegateSourceSendsTheFileOnlyOnTheCall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("UNIQUE-FILE-SENTENCE in the note."), 0o600); err != nil {
		t.Fatal(err)
	}
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"the sentence is there"}]}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()
	a := &Agent{
		Client: NewClient("settings-key"),
		Cfg:    Config{DelegateFast: "text-model", ContextLimit: 8000},
	}
	a.Client.BaseURL = server.URL
	answer, err := a.runDelegate(context.Background(), DelegateArgs{Task: "What sentence is in the file?", Profile: "fast", Source: path})
	if err != nil || answer != "the sentence is there" {
		t.Fatalf("answer %q err %v", answer, err)
	}
	if !strings.Contains(body, "UNIQUE-FILE-SENTENCE") || strings.Contains(body, `"source"`) {
		t.Fatalf("body %s", body)
	}
	if _, err := a.runDelegate(context.Background(), DelegateArgs{Task: "both", Profile: "fast", Source: path, Image: path}); err == nil {
		t.Fatal("picture and source accepted together")
	}
	label := delegateLabel(`{"task":"What sentence is in the file?","source":"` + path + `","image":"","context":"","profile":"fast"}`)
	if !strings.Contains(label, "source="+path) {
		t.Fatal(label)
	}
}

func TestDelegateSourceSplitsALargeList(t *testing.T) {
	dir := t.TempDir()
	var records []string
	for i, mark := range []string{"MARKER-A", "MARKER-B", "MARKER-C"} {
		records = append(records, `{"n":`+string(rune('0'+i))+`,"body":"`+strings.Repeat("x", 2400)+mark+`"}`)
	}
	path := filepath.Join(dir, "rows.json")
	if err := os.WriteFile(path, []byte("["+strings.Join(records, ",")+"]"), 0o600); err != nil {
		t.Fatal(err)
	}
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(raw))
		text := "part"
		if strings.Contains(string(raw), "Combine these partial answers") {
			text = "combined"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"` + text + `"}]}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()
	a := &Agent{
		Client:  NewClient("settings-key"),
		Cfg:     Config{DelegateFast: "text-model", ContextLimit: 1000},
		runChat: &chatTarget{ContextLimit: 1000},
	}
	a.Client.BaseURL = server.URL
	answer, err := a.runDelegate(context.Background(), DelegateArgs{Task: "Name the markers.", Profile: "fast", Source: path})
	if err != nil || answer != "combined" {
		t.Fatalf("answer %q err %v calls %d", answer, err, len(bodies))
	}
	if len(bodies) < 2 {
		t.Fatalf("calls %d", len(bodies))
	}
	budget := delegateSourceBudget(1000)
	seen := map[string]int{}
	for _, body := range bodies {
		if len(body) > budget+4000 {
			t.Fatalf("body %d over budget %d", len(body), budget)
		}
		for _, mark := range []string{"MARKER-A", "MARKER-B", "MARKER-C"} {
			if strings.Contains(body, mark) {
				seen[mark]++
			}
		}
	}
	if seen["MARKER-A"] != 1 || seen["MARKER-B"] != 1 || seen["MARKER-C"] != 1 {
		t.Fatalf("markers %#v calls %d", seen, len(bodies))
	}
}

func TestDelegateSourceBudgetFollowsTheModelWindow(t *testing.T) {
	if got, want := delegateSourceBudget(1_000_000), 1_000_000*4*8/10; got != want || got <= 400_000 {
		t.Fatalf("million-token budget %d", got)
	}
	if got, want := delegateSourceBudget(128_000), 128_000*4*8/10; got != want {
		t.Fatalf("smaller window %d", got)
	}
}

func TestDelegateSourceUsesTheWholeModelWindow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wide.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(bytes.Repeat([]byte("x"), 8<<20)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("TAIL-MARKER")); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	var hits int
	var sawTail bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		hits++
		sawTail = strings.Contains(string(raw), "TAIL-MARKER")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"read it"}]}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()
	a := &Agent{
		Client:  NewClient("settings-key"),
		Cfg:     Config{DelegateFast: "text-model", ContextLimit: 3_000_000},
		runChat: &chatTarget{ContextLimit: 3_000_000},
	}
	a.Client.BaseURL = server.URL
	answer, err := a.runDelegate(context.Background(), DelegateArgs{Task: "Is the tail there?", Profile: "fast", Source: path})
	if err != nil || answer != "read it" || hits != 1 || !sawTail {
		t.Fatalf("answer %q err %v hits %d tail %v", answer, err, hits, sawTail)
	}
}

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	u, _ := copy.URL.Parse(r.endpoint)
	copy.URL = u
	return r.base.RoundTrip(copy)
}
