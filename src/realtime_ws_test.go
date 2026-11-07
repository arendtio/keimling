package main

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRealtimeWebSocketURL(t *testing.T) {
	got, err := realtimeWebSocketURL("https://api.openai.com/v1", "gpt-realtime-2.1")
	if err != nil || got != "wss://api.openai.com/v1/realtime?model=gpt-realtime-2.1" {
		t.Fatalf("%q %v", got, err)
	}
	local, err := realtimeWebSocketURL("http://127.0.0.1:9/v1/", "m")
	if err != nil || local != "ws://127.0.0.1:9/v1/realtime?model=m" {
		t.Fatalf("%q %v", local, err)
	}
}

func TestRealtimeWebSocketProxiesText(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/realtime" || r.URL.Query().Get("model") != "gpt-realtime-2.1" {
			http.Error(w, "bad url", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer operator-key" {
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijack", http.StatusInternalServerError)
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		if err := acceptWebSocket(conn, r.Header.Get("Sec-WebSocket-Key")); err != nil {
			return
		}
		fin, op, payload, err := readWSFrame(conn)
		if err != nil || !fin || op != 0x1 || string(payload) != "hello-call" {
			return
		}
		_ = writeWSFrame(conn, true, 0x1, []byte(`{"type":"session.created"}`), false)
	}))
	defer upstream.Close()

	dir := t.TempDir()
	cfgPath := dir + "/config.json"
	raw := []byte(`{"model_providers":[{"id":"oa","name":"OpenAI","base_url":"` + upstream.URL + `/v1","api_key":"operator-key","kind":"openai"}]}`)
	if err := os.WriteFile(cfgPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	j, err := OpenJournal(dir+"/s.jsonl", false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &Agent{Cfg: Config{
		ConfigPath: cfgPath, Mode: "ask", HasModelProvider: true,
		RealtimeProviderID: "oa", RealtimeModel: "gpt-realtime-2.1", RealtimeVoice: "marin",
	}, Journal: j}
	srv := newAppServer(appServerOpts{AccessSecret: "tok", Agent: a})
	app := httptest.NewServer(srv.Handler())
	defer app.Close()

	conn, err := dialRealtimeWebSocket(t.Context(), "ws"+strings.TrimPrefix(app.URL, "http")+"/api/realtime/ws", "tok")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := writeWSFrame(conn, true, 0x1, []byte("hello-call"), true); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, payload, err := readWSFrame(conn)
	if err != nil || string(payload) != `{"type":"session.created"}` {
		t.Fatalf("frame %q %v", payload, err)
	}
	if !srv.realtimeCallOpen() {
		t.Fatal("call was not marked open")
	}
	conn.Close()
	deadline := time.Now().Add(2 * time.Second)
	for srv.realtimeCallOpen() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if srv.realtimeCallOpen() {
		t.Fatal("call stayed open after disconnect")
	}
}

func TestRealtimeSpeakerAppendsPCM(t *testing.T) {
	srv := newAppServer(appServerOpts{AccessSecret: "tok"})
	app := httptest.NewServer(srv.Handler())
	defer app.Close()
	req, err := http.NewRequest(http.MethodGet, app.URL+"/api/realtime/speaker", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.Status)
	}
	if resp.Header.Get("Content-Length") != "" {
		t.Fatal("content-length would make this window wait to start")
	}
	hdr := make([]byte, 44)
	if _, err := io.ReadFull(resp.Body, hdr); err != nil {
		t.Fatal(err)
	}
	if string(hdr[0:4]) != "RIFF" || string(hdr[8:12]) != "WAVE" {
		t.Fatalf("header %q", hdr)
	}
	pcm := []byte{0x34, 0x12, 0x78, 0x56}
	srv.teeRealtimeAudio([]byte(`{"type":"response.output_audio.delta","delta":"` + base64.StdEncoding.EncodeToString(pcm) + `"}`))
	got := make([]byte, len(pcm))
	if _, err := io.ReadFull(resp.Body, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != string(pcm) {
		t.Fatalf("pcm %v", got)
	}
	srv.noteRealtimeClient([]byte(`{"type":"response.cancel"}`))
	srv.teeRealtimeAudio([]byte(`{"type":"response.output_audio.delta","delta":"` + base64.StdEncoding.EncodeToString([]byte{9, 9, 9, 9}) + `"}`))
	next := []byte{5, 6, 7, 8}
	srv.teeRealtimeAudio([]byte(`{"type":"response.created"}`))
	srv.teeRealtimeAudio([]byte(`{"type":"response.output_audio.delta","delta":"` + base64.StdEncoding.EncodeToString(next) + `"}`))
	got = make([]byte, len(next))
	if _, err := io.ReadFull(resp.Body, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != string(next) {
		t.Fatalf("after cancel %v", got)
	}
	srv.teeRealtimeAudio([]byte(`{"type":"input_audio_buffer.speech_started"}`))
	srv.teeRealtimeAudio([]byte(`{"type":"response.output_audio.delta","delta":"` + base64.StdEncoding.EncodeToString([]byte{9, 9, 9, 9}) + `"}`))
	kept := []byte{1, 2, 3, 4}
	srv.teeRealtimeAudio([]byte(`{"type":"response.created"}`))
	srv.teeRealtimeAudio([]byte(`{"type":"response.output_audio.delta","delta":"` + base64.StdEncoding.EncodeToString(kept) + `"}`))
	got = make([]byte, len(kept))
	if _, err := io.ReadFull(resp.Body, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != string(kept) {
		t.Fatalf("after speech started %v", got)
	}
}

func TestRealtimeSpeakerAcceptsPageSecret(t *testing.T) {
	srv := newAppServer(appServerOpts{AccessSecret: "tok"})
	app := httptest.NewServer(srv.Handler())
	defer app.Close()
	resp, err := http.Get(app.URL + "/api/realtime/speaker?access_secret=tok")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.Status)
	}
	hdr := make([]byte, 44)
	if _, err := io.ReadFull(resp.Body, hdr); err != nil {
		t.Fatal(err)
	}
	if string(hdr[0:4]) != "RIFF" {
		t.Fatalf("header %q", hdr[:4])
	}
}
