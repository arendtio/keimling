package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// pcmFileToWAV wraps signed 16-bit mono little-endian PCM with a valid WAV header.
// A short or empty capture is never uploaded to the transcription API.
func pcmFileToWAV(path string, rate uint32) (string, error) {
	in, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return "", err
	}
	n := st.Size() &^ 1
	if n < int64(rate)*2/4 {
		return "", fmt.Errorf("recording too short (%d bytes); check ALSA device and microphone", n)
	}
	if n > 25<<20-44 {
		return "", fmt.Errorf("recording exceeds transcription size limit")
	}
	out, err := os.CreateTemp("", "keimling-*.wav")
	if err != nil {
		return "", err
	}
	keep := false
	defer func() {
		out.Close()
		if !keep {
			os.Remove(out.Name())
		}
	}()
	write := func(v interface{}) error { return binary.Write(out, binary.LittleEndian, v) }
	if _, err = out.Write([]byte("RIFF")); err != nil {
		return "", err
	}
	if err = write(uint32(n + 36)); err != nil {
		return "", err
	}
	if _, err = out.Write([]byte("WAVEfmt ")); err != nil {
		return "", err
	}
	for _, v := range []interface{}{uint32(16), uint16(1), uint16(1), rate, rate * 2, uint16(2), uint16(16)} {
		if err = write(v); err != nil {
			return "", err
		}
	}
	if _, err = out.Write([]byte("data")); err != nil {
		return "", err
	}
	if err = write(uint32(n)); err != nil {
		return "", err
	}
	if _, err = io.CopyN(out, in, n); err != nil {
		return "", err
	}
	if err = out.Close(); err != nil {
		return "", err
	}
	keep = true
	return out.Name(), nil
}
