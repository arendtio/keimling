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
	Connect            string              `json:"connect,omitempty"`
	AccessSecret       string              `json:"access_secret,omitempty"`
	Connections        []connectionProfile `json:"connections,omitempty"`
	ActiveConnectionID string              `json:"active_connection_id,omitempty"`
	WindowWidth        int                 `json:"window_width,omitempty"`
	WindowHeight       int                 `json:"window_height,omitempty"`
	BgLight            string              `json:"bg_light,omitempty"` // #rrggbb page background (light)
	BgDark             string              `json:"bg_dark,omitempty"`  // #rrggbb page background (dark)
	Themes             []themeRecord       `json:"themes,omitempty"`
	ActiveTheme        string              `json:"active_theme,omitempty"`
	ColorScheme        string              `json:"color_scheme,omitempty"` // light, dark, or empty for automatic
	WorkspaceID        string              `json:"workspace_id,omitempty"`
	SidebarOpen        *bool               `json:"sidebar_open,omitempty"`
	SidebarTab         string              `json:"sidebar_tab,omitempty"`
	SidebarWidth       int                 `json:"sidebar_width,omitempty"`
}

// normalizeColorScheme accepts automatic, light, or dark. Automatic is stored empty.
func normalizeColorScheme(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto", "automatic":
		return "", true
	case "light", "dark":
		return strings.ToLower(strings.TrimSpace(s)), true
	default:
		return "", false
	}
}

func colorSchemeOrAuto(s string) string {
	if s == "light" || s == "dark" {
		return s
	}
	return "auto"
}

// normalizeBgColor accepts #rgb / #rrggbb (case-insensitive); empty clears to default.
func normalizeBgColor(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", true
	}
	if !strings.HasPrefix(s, "#") {
		return "", false
	}
	hex := strings.ToLower(s[1:])
	switch len(hex) {
	case 3:
		for _, c := range hex {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return "", false
			}
		}
		return "#" + string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]}), true
	case 6:
		for _, c := range hex {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return "", false
			}
		}
		return "#" + hex, true
	default:
		return "", false
	}
}

// themeRecord is one named palette with a light and a dark scheme.
type themeRecord struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Builtin bool              `json:"builtin,omitempty"`
	Light   map[string]string `json:"light"`
	Dark    map[string]string `json:"dark"`
}

var themeRoles = []string{"bg", "surface", "surface2", "line", "fg", "muted", "accent", "danger", "bubble"}

func cleanThemeRecords(in []themeRecord) []themeRecord {
	out := make([]themeRecord, 0, len(in))
	seen := map[string]bool{}
	for _, t := range in {
		id := strings.TrimSpace(t.ID)
		if id == "" || seen[id] || len(id) > 40 || strings.ContainsAny(id, " \t\n/\\") {
			continue
		}
		name := strings.TrimSpace(t.Name)
		if name == "" {
			name = id
		}
		runes := []rune(name)
		if len(runes) > 40 {
			name = string(runes[:40])
		}
		light := cleanThemePalette(t.Light)
		dark := cleanThemePalette(t.Dark)
		if light == nil || dark == nil {
			continue
		}
		out = append(out, themeRecord{
			ID: id, Name: name, Builtin: id == "keimling", Light: light, Dark: dark,
		})
		seen[id] = true
		if len(out) >= 24 {
			break
		}
	}
	return out
}

func cleanThemePalette(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(themeRoles))
	for _, role := range themeRoles {
		c, ok := normalizeBgColor(in[role])
		if !ok || c == "" {
			return nil
		}
		out[role] = c
	}
	return out
}

func guiPrefsPath(configPath string) string {
	dir := filepath.Dir(strings.TrimSpace(configPath))
	if dir == "" || dir == "." {
		dir = keimlingConfigDir()
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
	normalizeConnectionProfiles(&p)
	return p, nil
}

func writeGuiPrefs(configPath string, p guiPrefs) error {
	path := guiPrefsPath(configPath)
	p.Connect = strings.TrimSpace(p.Connect)
	p.AccessSecret = strings.TrimSpace(p.AccessSecret)
	normalizeConnectionProfiles(&p)
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
// Also syncs the active remote profile (or switches to local when connect is empty).
func mergeGuiPrefsConnect(configPath, connect, secret string) error {
	return updateGuiPrefs(configPath, func(p *guiPrefs) {
		normalizeConnectionProfiles(p)
		connect = strings.TrimSpace(connect)
		secret = strings.TrimSpace(secret)
		if connect == "" {
			p.ActiveConnectionID = localConnectionID
			mirrorActiveConnection(p)
			return
		}
		for i := range p.Connections {
			c := &p.Connections[i]
			if c.Kind == connectionKindRemote && c.URL == connect {
				c.AccessSecret = secret
				p.ActiveConnectionID = c.ID
				mirrorActiveConnection(p)
				return
			}
		}
		id := newConnectionID(connectionKindRemote)
		p.Connections = append(p.Connections, connectionProfile{
			ID: id, Name: "Remote", Kind: connectionKindRemote, URL: connect, AccessSecret: secret,
		})
		p.ActiveConnectionID = id
		mirrorActiveConnection(p)
	})
}

func mergeGuiPrefsWorkspace(configPath, workspaceID string) error {
	return updateGuiPrefs(configPath, func(p *guiPrefs) {
		p.WorkspaceID = strings.TrimSpace(workspaceID)
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
