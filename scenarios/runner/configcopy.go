package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func configHome() string {
	if runtime.GOOS == "windows" {
		if d, err := os.UserConfigDir(); err == nil && strings.TrimSpace(d) != "" {
			return d
		}
	}
	if x := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); x != "" {
		return x
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

func userConfigDir() string {
	return filepath.Join(configHome(), "keimling")
}

// copyUserConfig copies the user's keimling configuration into dstData.
// The credential vault stays behind. configFile is the copy's config.json.
// configHome is the value for XDG_CONFIG_HOME so other files resolve beside it.
func copyUserConfig(dstData string) (configFile, configHome string, err error) {
	home := filepath.Join(dstData, "config-home")
	dst := filepath.Join(home, "keimling")
	if err = os.MkdirAll(dst, 0o700); err != nil {
		return "", "", err
	}
	src := userConfigDir()
	if st, statErr := os.Stat(src); statErr == nil && st.IsDir() {
		err = filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, relErr := filepath.Rel(src, path)
			if relErr != nil {
				return relErr
			}
			target := filepath.Join(dst, rel)
			if d.IsDir() {
				return os.MkdirAll(target, 0o700)
			}
			if d.Name() == "credentials.vault" || !d.Type().IsRegular() {
				return nil
			}
			b, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if mkErr := os.MkdirAll(filepath.Dir(target), 0o700); mkErr != nil {
				return mkErr
			}
			return os.WriteFile(target, b, 0o600)
		})
		if err != nil {
			return "", "", err
		}
	}
	cfg := filepath.Join(dst, "config.json")
	if _, statErr := os.Stat(cfg); os.IsNotExist(statErr) {
		if err = os.WriteFile(cfg, []byte("{}\n"), 0o600); err != nil {
			return "", "", err
		}
	}
	return cfg, home, nil
}

func childEnv(dataHome, configHome string) []string {
	out := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "XDG_DATA_HOME=") || strings.HasPrefix(e, "XDG_CONFIG_HOME=") {
			continue
		}
		out = append(out, e)
	}
	return append(out, "XDG_DATA_HOME="+dataHome, "XDG_CONFIG_HOME="+configHome)
}
