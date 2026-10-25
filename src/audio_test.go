package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestPCMtoWAV(t *testing.T) {
	p := filepath.Join(t.TempDir(), "test.pcm")
	if err := os.WriteFile(p, make([]byte, 12000), 0600); err != nil {
		t.Fatal(err)
	}
	wav, err := pcmFileToWAV(p, 12000)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(wav)
	b, err := os.ReadFile(wav)
	if err != nil {
		t.Fatal(err)
	}
	if string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" || string(b[36:40]) != "data" || binary.LittleEndian.Uint32(b[40:44]) != 12000 {
		t.Fatal("bad wav header")
	}
}
func TestPCMTooShort(t *testing.T) {
	p := filepath.Join(t.TempDir(), "empty.pcm")
	os.WriteFile(p, []byte{0}, 0600)
	if _, err := pcmFileToWAV(p, 12000); err == nil {
		t.Fatal("expected error")
	}
}
