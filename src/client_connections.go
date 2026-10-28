package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// clientConnectionsFile is the machine-local Keimling address book (GUI + CLI).
type clientConnectionsFile struct {
	Connections []connectionProfile `json:"connections"`
	ActiveID    string              `json:"active_id,omitempty"`
}

func clientConnectionsPath(configPath string) string {
	dir := filepath.Dir(strings.TrimSpace(configPath))
	if dir == "" || dir == "." {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config", "keimling")
	}
	return filepath.Join(dir, "connections.json")
}

func readClientConnections(configPath string) (clientConnectionsFile, error) {
	var f clientConnectionsFile
	path := clientConnectionsPath(configPath)
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		f.Connections = []connectionProfile{}
		migrateClientConnectionsFromGuiPrefs(configPath, &f)
		return f, nil
	}
	if err != nil {
		return f, err
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return f, err
	}
	normalizeClientConnections(&f)
	if len(f.Connections) == 0 {
		migrateClientConnectionsFromGuiPrefs(configPath, &f)
	}
	return f, nil
}

func writeClientConnections(configPath string, f clientConnectionsFile) error {
	normalizeClientConnections(&f)
	path := clientConnectionsPath(configPath)
	b, err := json.MarshalIndent(f, "", "  ")
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

func updateClientConnections(configPath string, mut func(*clientConnectionsFile)) error {
	f, err := readClientConnections(configPath)
	if err != nil {
		return err
	}
	mut(&f)
	return writeClientConnections(configPath, f)
}

func normalizeClientConnections(f *clientConnectionsFile) {
	if f.Connections == nil {
		f.Connections = []connectionProfile{}
	}
	seen := map[string]bool{}
	out := make([]connectionProfile, 0, len(f.Connections))
	for _, c := range f.Connections {
		c.ID = strings.TrimSpace(c.ID)
		c.Name = strings.TrimSpace(c.Name)
		c.Kind = strings.ToLower(strings.TrimSpace(c.Kind))
		c.URL = strings.TrimSpace(c.URL)
		c.AccessSecret = strings.TrimSpace(c.AccessSecret)
		// Client book is remotes only; drop accidental local rows.
		if c.Kind != connectionKindRemote || c.URL == "" {
			continue
		}
		c.Kind = connectionKindRemote
		if c.ID == "" || seen[c.ID] {
			c.ID = newConnectionID(connectionKindRemote)
		}
		seen[c.ID] = true
		if c.Name == "" {
			c.Name = c.URL
		}
		out = append(out, c)
	}
	f.Connections = out
	active := strings.TrimSpace(f.ActiveID)
	if active != "" && connectionByID(f.Connections, active) == nil {
		active = ""
	}
	f.ActiveID = active
}

func migrateClientConnectionsFromGuiPrefs(configPath string, f *clientConnectionsFile) {
	p, err := readGuiPrefs(configPath)
	if err != nil {
		return
	}
	normalizeConnectionProfiles(&p)
	changed := false
	for _, c := range p.Connections {
		if c.Kind != connectionKindRemote || strings.TrimSpace(c.URL) == "" {
			continue
		}
		if clientConnectionByURL(f.Connections, c.URL) != nil {
			continue
		}
		f.Connections = append(f.Connections, connectionProfile{
			ID:           c.ID,
			Name:         c.Name,
			Kind:         connectionKindRemote,
			URL:          c.URL,
			AccessSecret: c.AccessSecret,
		})
		changed = true
	}
	if f.ActiveID == "" && p.ActiveConnectionID != "" {
		if connectionByID(f.Connections, p.ActiveConnectionID) != nil {
			f.ActiveID = p.ActiveConnectionID
			changed = true
		}
	}
	normalizeClientConnections(f)
	if changed {
		_ = writeClientConnections(configPath, *f)
	}
}

func clientConnectionByURL(list []connectionProfile, url string) *connectionProfile {
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	for i := range list {
		if strings.TrimRight(list[i].URL, "/") == url {
			return &list[i]
		}
	}
	return nil
}

func (f *clientConnectionsFile) public() map[string]any {
	normalizeClientConnections(f)
	items := make([]map[string]any, 0, len(f.Connections))
	for _, c := range f.Connections {
		items = append(items, map[string]any{
			"id":                c.ID,
			"name":              c.Name,
			"kind":              c.Kind,
			"url":               c.URL,
			"access_secret_set": c.AccessSecret != "",
			"access_secret":     c.AccessSecret,
		})
	}
	return map[string]any{
		"connections": items,
		"active_id":   f.ActiveID,
		"source":      "machine",
		"path":        "connections.json",
	}
}

// upsertClientRemote saves or updates a remote in the machine book (adopt-on-connect).
func upsertClientRemote(configPath, name, url, secret string) error {
	url = strings.TrimSpace(url)
	secret = strings.TrimSpace(secret)
	if url == "" {
		return fmt.Errorf("url required")
	}
	return updateClientConnections(configPath, func(f *clientConnectionsFile) {
		if existing := clientConnectionByURL(f.Connections, url); existing != nil {
			if n := strings.TrimSpace(name); n != "" {
				existing.Name = n
			}
			if secret != "" {
				existing.AccessSecret = secret
			}
			f.ActiveID = existing.ID
			return
		}
		id := newConnectionID(connectionKindRemote)
		if strings.TrimSpace(name) == "" {
			name = "Remote"
		}
		f.Connections = append(f.Connections, connectionProfile{
			ID: id, Name: name, Kind: connectionKindRemote, URL: url, AccessSecret: secret,
		})
		f.ActiveID = id
	})
}

// resolveConnectReference expands a --connect value that is a profile name or id.
// URLs (containing "://") are returned unchanged; secretOut is only set when resolved from the book.
func resolveConnectReference(configPath, connect string) (url string, secret string, ok bool) {
	connect = strings.TrimSpace(connect)
	if connect == "" {
		return "", "", false
	}
	if strings.Contains(connect, "://") {
		return connect, "", false
	}
	f, err := readClientConnections(configPath)
	if err != nil {
		return "", "", false
	}
	for _, c := range f.Connections {
		if c.ID == connect || strings.EqualFold(c.Name, connect) {
			return c.URL, c.AccessSecret, true
		}
	}
	return "", "", false
}

// applyClientConnectionsConnect fills --connect from the machine book when unset.
func applyClientConnectionsConnect(configPath string, connectURL, accessSecret *string, connectFlagSet bool) error {
	if connectFlagSet || strings.TrimSpace(*connectURL) != "" {
		return nil
	}
	f, err := readClientConnections(configPath)
	if err != nil {
		return fmt.Errorf("connections.json: %w", err)
	}
	c := connectionByID(f.Connections, f.ActiveID)
	if c == nil || c.URL == "" {
		return nil
	}
	*connectURL = c.URL
	if strings.TrimSpace(*accessSecret) == "" && c.AccessSecret != "" {
		*accessSecret = c.AccessSecret
	}
	return nil
}

// clearClientActiveConnection drops auto-reconnect to a saved remote without
// deleting profiles (used when switching back to the desktop agent).
func clearClientActiveConnection(configPath string) error {
	return updateClientConnections(configPath, func(f *clientConnectionsFile) {
		f.ActiveID = ""
	})
}
