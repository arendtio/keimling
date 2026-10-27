package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultWindowWidth  = 1248 // ~30% larger than prior 960
	defaultWindowHeight = 936  // ~30% larger than prior 720
)

// guiPrefs remembers desktop UI state across ephemeral localhost ports
// (localStorage cannot — each launch uses a new origin).
type guiPrefs struct {
	Connect      string `json:"connect,omitempty"`
	AccessSecret string `json:"access_secret,omitempty"`
	WindowWidth  int    `json:"window_width,omitempty"`
	WindowHeight int    `json:"window_height,omitempty"`
}

func guiPrefsPath(configPath string) string {
	dir := filepath.Dir(strings.TrimSpace(configPath))
	if dir == "" || dir == "." {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config", "keimling")
	}
	return filepath.Join(dir, "gui-prefs.json")
}

func readGuiPrefs(configPath string) (guiPrefs, error) {
	var p guiPrefs
	path := guiPrefsPath(configPath)
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, err
	}
	p.Connect = strings.TrimSpace(p.Connect)
	p.AccessSecret = strings.TrimSpace(p.AccessSecret)
	return p, nil
}

func writeGuiPrefs(configPath string, p guiPrefs) error {
	path := guiPrefsPath(configPath)
	p.Connect = strings.TrimSpace(p.Connect)
	p.AccessSecret = strings.TrimSpace(p.AccessSecret)
	if p.Connect == "" {
		p.AccessSecret = ""
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func updateGuiPrefs(configPath string, mut func(*guiPrefs)) error {
	p, err := readGuiPrefs(configPath)
	if err != nil {
		return err
	}
	mut(&p)
	return writeGuiPrefs(configPath, p)
}

// mergeGuiPrefsConnect updates only connection fields; window size is preserved.
func mergeGuiPrefsConnect(configPath, connect, secret string) error {
	return updateGuiPrefs(configPath, func(p *guiPrefs) {
		p.Connect = strings.TrimSpace(connect)
		if p.Connect == "" {
			p.AccessSecret = ""
		} else {
			p.AccessSecret = strings.TrimSpace(secret)
		}
	})
}

func saveGuiWindowSize(configPath string, width, height int) error {
	if width < 400 || height < 300 || width > 10000 || height > 10000 {
		return nil
	}
	return updateGuiPrefs(configPath, func(p *guiPrefs) {
		p.WindowWidth = width
		p.WindowHeight = height
	})
}

func windowSizeFromPrefs(p guiPrefs) (width, height int) {
	width, height = p.WindowWidth, p.WindowHeight
	if width < 400 || height < 300 {
		return defaultWindowWidth, defaultWindowHeight
	}
	return width, height
}

func applyGuiPrefsConnect(configPath string, connectURL, accessSecret *string, connectFlagSet bool) error {
	if connectFlagSet || strings.TrimSpace(*connectURL) != "" {
		return nil
	}
	p, err := readGuiPrefs(configPath)
	if err != nil {
		return fmt.Errorf("gui-prefs: %w", err)
	}
	if p.Connect == "" {
		return nil
	}
	*connectURL = p.Connect
	if strings.TrimSpace(*accessSecret) == "" && p.AccessSecret != "" {
		*accessSecret = p.AccessSecret
	}
	return nil
}
