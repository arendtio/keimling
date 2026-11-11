package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

// screenTarget is one screen or window the user can share.
type screenTarget struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// screenShareFileName marks the live frame attached to a chat message.
const screenShareFileName = "keimling-screen.jpg"

var screenShareState struct {
	mu   sync.Mutex
	gen  int
	jpeg []byte
}

func beginScreenShare() int {
	screenShareState.mu.Lock()
	defer screenShareState.mu.Unlock()
	screenShareState.gen++
	screenShareState.jpeg = nil
	return screenShareState.gen
}

func noteScreenShare(gen int, jpeg []byte) {
	if gen == 0 || len(jpeg) == 0 {
		return
	}
	screenShareState.mu.Lock()
	defer screenShareState.mu.Unlock()
	if screenShareState.gen != gen {
		return
	}
	screenShareState.jpeg = append([]byte(nil), jpeg...)
}

func endScreenShare(gen int) {
	screenShareState.mu.Lock()
	defer screenShareState.mu.Unlock()
	if screenShareState.gen != gen {
		return
	}
	screenShareState.jpeg = nil
}

func latestScreenJPEG() []byte {
	screenShareState.mu.Lock()
	defer screenShareState.mu.Unlock()
	if len(screenShareState.jpeg) == 0 {
		return nil
	}
	return append([]byte(nil), screenShareState.jpeg...)
}

// withScreenShare appends the current live frame so the chat model can see it.
func withScreenShare(uploads []chatFile) []chatFile {
	jpeg := latestScreenJPEG()
	if len(jpeg) == 0 {
		return uploads
	}
	saved, err := saveChatUpload(attachmentRoot(), screenShareFileName, bytes.NewReader(jpeg))
	if err != nil {
		return uploads
	}
	return append(uploads, saved)
}

var errScreenCancelled = errors.New("cancelled")

// screenCast is one live share chosen in the desktop dialog.
type screenCast struct {
	fd      int
	run     func(context.Context, func([]byte) error) error
	stop    func()
	control *screenControl
}

func (s *appServer) handleScreenTargets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	kind, targets := screenShareChooser()
	if targets == nil {
		targets = []screenTarget{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{"chooser": kind, "targets": targets})
}

func (s *appServer) handleScreenShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pick, stopPick := context.WithTimeout(r.Context(), 3*time.Minute)
	input := r.URL.Query().Get("input") == "1" || r.URL.Query().Get("input") == "true"
	cast, err := openScreenCast(pick, input, r.URL.Query().Get("target"))
	stopPick()
	if errors.Is(err, errScreenCancelled) || errors.Is(err, context.DeadlineExceeded) {
		http.Error(w, "cancelled", http.StatusConflict)
		return
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer cast.stop()
	gen := beginScreenShare()
	defer endScreenShare(gen)
	if cast.control != nil {
		attachScreenControl(cast.control)
		defer detachScreenControl(cast.control)
	}

	frames := make(chan []byte, 1)
	done := make(chan struct{})
	wrote := false
	go func() {
		defer close(done)
		fl, _ := w.(http.Flusher)
		var lastSend time.Time
		for frame := range frames {
			now := time.Now()
			if !lastSend.IsZero() && now.Sub(lastSend) < 200*time.Millisecond {
				continue
			}
			lastSend = now
			if !wrote {
				wrote = true
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.WriteHeader(http.StatusOK)
			}
			var hdr [4]byte
			binary.LittleEndian.PutUint32(hdr[:], uint32(len(frame)))
			if _, err := w.Write(hdr[:]); err != nil {
				return
			}
			if _, err := w.Write(frame); err != nil {
				return
			}
			if fl != nil {
				fl.Flush()
			}
		}
	}()
	runErr := cast.run(r.Context(), func(frame []byte) error {
		noteScreenShare(gen, frame)
		if cast.control != nil {
			if w, h, ok := jpegSize(frame); ok {
				cast.control.setSize(w, h)
			}
		}
		select {
		case frames <- frame:
		default:
			select {
			case <-frames:
			default:
			}
			select {
			case frames <- frame:
			default:
			}
		}
		return nil
	})
	close(frames)
	<-done
	if wrote || r.Context().Err() != nil || errors.Is(runErr, context.Canceled) {
		return
	}
	msg := "screen share produced no picture"
	if runErr != nil {
		msg = runErr.Error()
	}
	http.Error(w, msg, http.StatusBadGateway)
}

// consumeJPEGs reads a concatenated JPEG stream and emits each picture.
func consumeJPEGs(r io.Reader, emit func([]byte) error) error {
	buf := make([]byte, 0, 64*1024)
	tmp := make([]byte, 32*1024)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		for {
			start := markerIndex(buf, 0xd8)
			if start < 0 {
				if len(buf) > 1 {
					buf = buf[len(buf)-1:]
				}
				break
			}
			if start > 0 {
				buf = buf[start:]
			}
			end := markerIndex(buf[2:], 0xd9)
			if end < 0 {
				if len(buf) > 8<<20 {
					return errors.New("screen share picture is too large")
				}
				break
			}
			end += 4
			frame := append([]byte(nil), buf[:end]...)
			buf = buf[end:]
			if err := emit(frame); err != nil {
				return err
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func markerIndex(b []byte, second byte) int {
	for i := 0; i+1 < len(b); i++ {
		if b[i] == 0xff && b[i+1] == second {
			return i
		}
	}
	return -1
}
