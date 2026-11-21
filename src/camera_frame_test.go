package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var testCameraJPEG = []byte{0xff, 0xd8, 0x01, 0xff, 0xd9}

func TestCameraShareFrameReachesTheChatModel(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	clearCameraShare()
	if got := withCameraShare(nil); len(got) != 0 {
		t.Fatalf("idle camera attached %#v", got)
	}
	noteCameraShare(testCameraJPEG)
	uploads := withCameraShare(nil)
	if len(uploads) != 1 || uploads[0].Name != cameraShareFileName {
		t.Fatalf("upload %#v", uploads)
	}
	raw, err := os.ReadFile(uploads[0].Path)
	if err != nil || !bytes.Equal(raw, testCameraJPEG) {
		t.Fatalf("stored %q %v", raw, err)
	}
	item, _ := jsonMarshalUser(joinChatFiles("what is in front of the camera?", uploads))
	on := expandChatFiles([]json.RawMessage{item}, true, false)
	if !bytes.Contains(on[0], []byte("input_image")) || !bytes.Contains(on[0], []byte("data:image/jpeg;base64,")) {
		t.Fatalf("model input %s", on[0])
	}
	if !bytes.Contains(on[0], []byte("current camera frame")) {
		t.Fatalf("camera caption: %s", on[0])
	}
	off := expandChatFiles([]json.RawMessage{item}, false, false)
	if bytes.Contains(off[0], []byte("input_image")) || bytes.Contains(off[0], []byte("base64")) {
		t.Fatalf("seeing off embedded the picture: %s", off[0])
	}
	clearCameraShare()
	if got := withCameraShare(nil); len(got) != 0 {
		t.Fatalf("stopped camera attached %#v", got)
	}
}

func TestCameraAndScreenShareAreIndependent(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	clearCameraShare()
	endScreenShare(beginScreenShare())
	gen := beginScreenShare()
	noteScreenShare(gen, []byte{0xff, 0xd8, 0x02, 0xff, 0xd9})
	noteCameraShare(testCameraJPEG)

	names := map[string]bool{}
	for _, f := range withCameraShare(withScreenShare(nil)) {
		names[f.Name] = true
	}
	if !names[screenShareFileName] || !names[cameraShareFileName] {
		t.Fatalf("both shares attached %v", names)
	}

	clearCameraShare()
	names = map[string]bool{}
	for _, f := range withCameraShare(withScreenShare(nil)) {
		names[f.Name] = true
	}
	if names[cameraShareFileName] || !names[screenShareFileName] {
		t.Fatalf("stopping the camera kept the screen only %v", names)
	}

	noteCameraShare(testCameraJPEG)
	endScreenShare(gen)
	names = map[string]bool{}
	for _, f := range withCameraShare(withScreenShare(nil)) {
		names[f.Name] = true
	}
	if names[screenShareFileName] || !names[cameraShareFileName] {
		t.Fatalf("stopping the screen kept the camera only %v", names)
	}
	clearCameraShare()
}

func TestOnlyTheLatestCameraFrameIsSentToTheModel(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "old", cameraShareFileName)
	newer := filepath.Join(dir, "new", cameraShareFileName)
	for _, path := range []string{older, newer} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(older, []byte{0xff, 0xd8, 0x01, 0xff, 0xd9}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, []byte{0xff, 0xd8, 0x02, 0xff, 0xd9}, 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := json.Marshal(map[string]any{"role": "user", "content": joinChatFiles("earlier", []chatFile{{Name: cameraShareFileName, Path: older, Source: "temp"}})})
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(map[string]any{"role": "user", "content": joinChatFiles("now", []chatFile{{Name: cameraShareFileName, Path: newer, Source: "temp"}})})
	if err != nil {
		t.Fatal(err)
	}
	out := expandChatFiles([]json.RawMessage{first, second}, true, false)
	if bytes.Contains(out[0], []byte("input_image")) || bytes.Contains(out[0], []byte("base64")) {
		t.Fatalf("older camera frame stayed in context: %s", out[0])
	}
	if !bytes.Contains(out[0], []byte("newer live camera frame")) {
		t.Fatalf("older camera frame note: %s", out[0])
	}
	if !bytes.Contains(out[1], []byte("input_image")) || !bytes.Contains(out[1], []byte("data:image/jpeg;base64,")) {
		t.Fatalf("latest camera frame: %s", out[1])
	}
}

func TestCameraSnapshotToolHandsTheModelTheFrame(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	clearCameraShare()
	off := runCameraSnapshotTool()
	if !strings.Contains(off, "turn the camera") {
		t.Fatalf("camera off answer %q", off)
	}
	noteCameraShare(testCameraJPEG)
	defer clearCameraShare()
	out := runCameraSnapshotTool()
	if !strings.Contains(out, cameraSnapshotFileName) || !strings.Contains(out, imageDelegateHint) {
		t.Fatalf("snapshot answer %q", out)
	}
	path := strings.SplitN(strings.SplitN(out, "Current camera frame: ", 2)[1], "\n", 2)[0]
	raw, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(raw, testCameraJPEG) {
		t.Fatalf("snapshot file %q %v", raw, err)
	}
}

func TestHandleCameraShareStoresAndClearsTheFrame(t *testing.T) {
	clearCameraShare()
	defer clearCameraShare()
	s := &appServer{}

	post := func(target string, body []byte) int {
		req := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
		rec := httptest.NewRecorder()
		s.handleCameraShare(rec, req)
		return rec.Code
	}

	if code := post("/api/camera/share", testCameraJPEG); code != http.StatusNoContent {
		t.Fatalf("frame code %d", code)
	}
	if got := latestCameraJPEG(); !bytes.Equal(got, testCameraJPEG) {
		t.Fatalf("stored frame %q", got)
	}
	if code := post("/api/camera/share", []byte("not a picture")); code != http.StatusBadRequest {
		t.Fatalf("bad frame code %d", code)
	}
	if got := latestCameraJPEG(); !bytes.Equal(got, testCameraJPEG) {
		t.Fatalf("a rejected frame replaced the picture %q", got)
	}
	if code := post("/api/camera/share?stop=1", nil); code != http.StatusNoContent {
		t.Fatalf("stop code %d", code)
	}
	if got := latestCameraJPEG(); len(got) != 0 {
		t.Fatalf("stopped share kept %q", got)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/camera/share", nil)
	rec := httptest.NewRecorder()
	s.handleCameraShare(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET code %d", rec.Code)
	}
}
