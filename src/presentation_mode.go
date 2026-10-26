package main

import (
	"fmt"
	"strings"
)

const (
	presentationAuto   = "auto"
	presentationUI     = "ui"
	presentationCLI    = "cli"
	presentationServer = "server"
)

func resolvePresentation(configValue string, ui, cli, server bool) (string, error) {
	n := 0
	flagMode := ""
	if ui {
		n++
		flagMode = presentationUI
	}
	if cli {
		n++
		flagMode = presentationCLI
	}
	if server {
		n++
		flagMode = presentationServer
	}
	if n > 1 {
		return "", fmt.Errorf("--ui, --cli, and --server are mutually exclusive")
	}
	if n == 1 {
		return flagMode, nil
	}
	v := strings.TrimSpace(strings.ToLower(configValue))
	if v == "" {
		return presentationAuto, nil
	}
	switch v {
	case presentationAuto, presentationUI, presentationCLI, presentationServer:
		return v, nil
	default:
		return "", fmt.Errorf("invalid presentation %q (want auto, ui, cli, or server)", configValue)
	}
}

func validateServerListen(listen, secret string) error {
	if strings.TrimSpace(listen) == "" {
		return fmt.Errorf("--listen is required with --server")
	}
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("--access-secret is required with --server")
	}
	return nil
}
