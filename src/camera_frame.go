package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// cameraShareFileName marks the live camera frame attached to a chat message.
const cameraShareFileName = "keimling-camera.jpg"

// cameraSnapshotFileName names the still a camera_snapshot call hands to the model.
const cameraSnapshotFileName = "camera-snapshot.jpg"

// cameraFrameLimit is the largest single JPEG frame the browser may post.
const cameraFrameLimit = 4 << 20

// cameraShareState is the newest frame of the camera the user turned on. It is
// independent of the screen share: both can run at the same time.
var cameraShareState struct {
	mu   sync.Mutex
	jpeg []byte
}

func noteCameraShare(jpeg []byte) {
	if len(jpeg) == 0 {
		return
	}
	cameraShareState.mu.Lock()
	defer cameraShareState.mu.Unlock()
	cameraShareState.jpeg = append([]byte(nil), jpeg...)
}

// clearCameraShare drops the frame so nothing is attached to the next turn.
func clearCameraShare() {
	cameraShareState.mu.Lock()
	defer cameraShareState.mu.Unlock()
	cameraShareState.jpeg = nil
}

func latestCameraJPEG() []byte {
	cameraShareState.mu.Lock()
	defer cameraShareState.mu.Unlock()
	if len(cameraShareState.jpeg) == 0 {
		return nil
	}
	return append([]byte(nil), cameraShareState.jpeg...)
}

// withCameraShare appends the current live camera frame so the chat model can see it.
func withCameraShare(uploads []chatFile) []chatFile {
	jpeg := latestCameraJPEG()
	if len(jpeg) == 0 {
		return uploads
	}
	saved, err := saveChatUpload(attachmentRoot(), cameraShareFileName, bytes.NewReader(jpeg))
	if err != nil {
		return uploads
	}
	return append(uploads, saved)
}

func cameraShareCaption() string {
	return "This is the live camera share. The current camera frame is included in this message and replaces any earlier one. There is no sound."
}

// handleCameraShare takes one JPEG frame per request from the browser, or
// clears the frame when the client stops the share with ?stop=1.
func (s *appServer) handleCameraShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Query().Get("stop") == "1" || r.URL.Query().Get("stop") == "true" {
		clearCameraShare()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, cameraFrameLimit+1))
	if err != nil {
		http.Error(w, "camera frame could not be read", http.StatusBadRequest)
		return
	}
	if len(data) == 0 || len(data) > cameraFrameLimit {
		http.Error(w, "camera frame is empty or too large", http.StatusBadRequest)
		return
	}
	if len(data) < 2 || data[0] != 0xff || data[1] != 0xd8 {
		http.Error(w, "camera frame is not a JPEG picture", http.StatusBadRequest)
		return
	}
	noteCameraShare(data)
	w.WriteHeader(http.StatusNoContent)
}

// runCameraSnapshotTool gives the model the newest camera picture as a file.
// Without a running camera share it asks the user to turn the camera on.
func runCameraSnapshotTool() string {
	jpeg := latestCameraJPEG()
	if len(jpeg) == 0 {
		return "The camera is off. Ask the user to turn the camera share on in the sidebar, then call camera_snapshot again."
	}
	saved, err := saveChatUpload(attachmentRoot(), cameraSnapshotFileName, bytes.NewReader(jpeg))
	if err != nil {
		return fmt.Sprintf("camera snapshot: %s", err)
	}
	return strings.TrimSpace("Current camera frame: " + saved.Path + "\n" + imageDelegateHint)
}

func cameraToolDefs() []map[string]any {
	return []map[string]any{
		toolDef("camera_snapshot", "Get the current camera picture as a JPEG file. The path is in the result; call delegate with image set to that path to look at it. Without a running camera share the result asks the user to turn the camera on. There is no sound.", map[string]any{}),
	}
}

func cameraTool(name string) bool {
	return name == "camera_snapshot"
}
