package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

func captureCommand(path, seconds string) (*exec.Cmd, error) {
	name, args, err := captureArgs(path, seconds)
	if err != nil {
		return nil, err
	}
	return exec.Command(name, args...), nil
}

func captureCommandContext(ctx context.Context, path, seconds string) (*exec.Cmd, error) {
	name, args, err := captureArgs(path, seconds)
	if err != nil {
		return nil, err
	}
	return exec.CommandContext(ctx, name, args...), nil
}

func captureArgs(path, seconds string) (string, []string, error) {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("rec"); err == nil {
			return "rec", []string{"-q", "-t", "raw", "-r", "12000", "-e", "signed-integer", "-b", "16", "-c", "1", path, "trim", "0", seconds}, nil
		}
		if _, err := exec.LookPath("ffmpeg"); err == nil {
			return "ffmpeg", []string{"-y", "-loglevel", "error", "-f", "avfoundation", "-i", ":0", "-t", seconds, "-ar", "12000", "-ac", "1", "-f", "s16le", path}, nil
		}
		return "", nil, fmt.Errorf("voice capture needs rec (sox) or ffmpeg")
	case "windows":
		if _, err := exec.LookPath("ffmpeg"); err == nil {
			return "ffmpeg", []string{"-y", "-loglevel", "error", "-f", "dshow", "-i", "audio=Microphone", "-t", seconds, "-ar", "12000", "-ac", "1", "-f", "s16le", path}, nil
		}
		return "", nil, fmt.Errorf("voice capture needs ffmpeg")
	default:
		return "arecord", []string{"-q", "-f", "S16_LE", "-r", "12000", "-c", "1", "-d", seconds, "-t", "raw", path}, nil
	}
}
