package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestConsumeJPEGsSplitsPictures(t *testing.T) {
	var raw []byte
	raw = append(raw, 0x00, 0x11)
	raw = append(raw, 0xff, 0xd8, 0x01, 0x02, 0xff, 0xd9)
	raw = append(raw, 0xff, 0xd8, 0xff, 0xd9)
	var got [][]byte
	err := consumeJPEGs(bytes.NewReader(raw), func(frame []byte) error {
		got = append(got, append([]byte(nil), frame...))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || len(got[0]) != 6 || len(got[1]) != 4 {
		t.Fatalf("frames %#v", got)
	}
	if got[0][0] != 0xff || got[0][1] != 0xd8 || got[1][2] != 0xff || got[1][3] != 0xd9 {
		t.Fatalf("markers %#v", got)
	}
}

func TestConsumeJPEGsStopsWhenTheWriterStops(t *testing.T) {
	raw := []byte{0xff, 0xd8, 0xff, 0xd9, 0xff, 0xd8, 0xff, 0xd9}
	err := consumeJPEGs(bytes.NewReader(raw), func([]byte) error {
		return errors.New("client left")
	})
	if err == nil || err.Error() != "client left" {
		t.Fatalf("err %v", err)
	}
}

func TestScreenShareFrameReachesTheChatModel(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	endScreenShare(beginScreenShare())
	if got := withScreenShare(nil); len(got) != 0 {
		t.Fatalf("idle share attached %#v", got)
	}
	gen := beginScreenShare()
	defer endScreenShare(gen)
	noteScreenShare(gen, []byte{0xff, 0xd8, 0x01, 0xff, 0xd9})
	uploads := withScreenShare(nil)
	if len(uploads) != 1 || uploads[0].Name != screenShareFileName {
		t.Fatalf("upload %#v", uploads)
	}
	raw, err := os.ReadFile(uploads[0].Path)
	if err != nil || !bytes.Equal(raw, []byte{0xff, 0xd8, 0x01, 0xff, 0xd9}) {
		t.Fatalf("stored %q %v", raw, err)
	}
	item, _ := jsonMarshalUser(joinChatFiles("what is on the screen?", uploads))
	on := expandChatFiles([]json.RawMessage{item}, true, false)
	if !bytes.Contains(on[0], []byte("input_image")) || !bytes.Contains(on[0], []byte("data:image/jpeg;base64,")) {
		t.Fatalf("model input %s", on[0])
	}
	off := expandChatFiles([]json.RawMessage{item}, false, false)
	if bytes.Contains(off[0], []byte("input_image")) || bytes.Contains(off[0], []byte("base64")) {
		t.Fatalf("seeing off embedded the picture: %s", off[0])
	}
	endScreenShare(gen)
	if got := withScreenShare(nil); len(got) != 0 {
		t.Fatalf("stopped share attached %#v", got)
	}
}

func TestOnlyTheLatestScreenFrameIsSentToTheModel(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "old", screenShareFileName)
	newer := filepath.Join(dir, "new", screenShareFileName)
	if err := os.MkdirAll(filepath.Dir(older), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(newer), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(older, []byte{0xff, 0xd8, 0x01, 0xff, 0xd9}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, []byte{0xff, 0xd8, 0x02, 0xff, 0xd9}, 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := json.Marshal(map[string]any{"role": "user", "content": joinChatFiles("earlier", []chatFile{{Name: screenShareFileName, Path: older, Source: "temp"}})})
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(map[string]any{"role": "user", "content": joinChatFiles("now", []chatFile{{Name: screenShareFileName, Path: newer, Source: "temp"}})})
	if err != nil {
		t.Fatal(err)
	}
	out := expandChatFiles([]json.RawMessage{first, second}, true, false)
	if bytes.Contains(out[0], []byte("input_image")) || bytes.Contains(out[0], []byte("base64")) {
		t.Fatalf("older frame stayed in context: %s", out[0])
	}
	if !bytes.Contains(out[0], []byte("replaced")) {
		t.Fatalf("older frame note: %s", out[0])
	}
	if !bytes.Contains(out[1], []byte("input_image")) || !bytes.Contains(out[1], []byte("data:image/jpeg;base64,")) {
		t.Fatalf("latest frame: %s", out[1])
	}
}

func jsonMarshalUser(content string) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"role": "user", "content": content})
}
