package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	connectionKindLocal  = "local"
	connectionKindRemote = "remote"
	localConnectionID    = "local"
)

// connectionProfile is a saved UI→Keimling endpoint in gui-prefs.
type connectionProfile struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Kind         string `json:"kind"` // local | remote
	URL          string `json:"url,omitempty"`
	AccessSecret string `json:"access_secret,omitempty"`
}

func newConnectionID(kind string) string {
	return fmt.Sprintf("%s-%s", kind, time.Now().UTC().Format("20060102-150405.000000000"))
}

func defaultLocalConnection() connectionProfile {
	return connectionProfile{
		ID:   localConnectionID,
		Name: "This window",
		Kind: connectionKindLocal,
	}
}

// normalizeConnectionProfiles ensures a local profile exists, migrates legacy
// connect/access_secret into a remote profile when needed, and mirrors the
// active remote back into legacy fields.
func normalizeConnectionProfiles(p *guiPrefs) {
	if p.Connections == nil {
		p.Connections = []connectionProfile{}
	}
	hasLocal := false
	for i := range p.Connections {
		c := &p.Connections[i]
		c.ID = strings.TrimSpace(c.ID)
		c.Name = strings.TrimSpace(c.Name)
		c.Kind = strings.ToLower(strings.TrimSpace(c.Kind))
		c.URL = strings.TrimSpace(c.URL)
		c.AccessSecret = strings.TrimSpace(c.AccessSecret)
		if c.Kind != connectionKindRemote {
			c.Kind = connectionKindLocal
			c.URL = ""
			c.AccessSecret = ""
		}
		if c.ID == "" {
			c.ID = newConnectionID(c.Kind)
		}
		if c.Name == "" {
			if c.Kind == connectionKindLocal {
				c.Name = "This window"
			} else {
				c.Name = c.URL
			}
		}
		if c.Kind == connectionKindLocal {
			hasLocal = true
			c.ID = localConnectionID
		}
	}
	if !hasLocal {
		p.Connections = append([]connectionProfile{defaultLocalConnection()}, p.Connections...)
	}

	// Migrate legacy single connect into a remote profile once.
	legacyURL := strings.TrimSpace(p.Connect)
	if legacyURL != "" {
		found := false
		for _, c := range p.Connections {
			if c.Kind == connectionKindRemote && c.URL == legacyURL {
				found = true
				break
			}
		}
		if !found {
			p.Connections = append(p.Connections, connectionProfile{
				ID:           newConnectionID(connectionKindRemote),
				Name:         "Remote",
				Kind:         connectionKindRemote,
				URL:          legacyURL,
				AccessSecret: strings.TrimSpace(p.AccessSecret),
			})
		}
	}

	active := strings.TrimSpace(p.ActiveConnectionID)
	if active == "" {
		if legacyURL != "" {
			for _, c := range p.Connections {
				if c.Kind == connectionKindRemote && c.URL == legacyURL {
					active = c.ID
					break
				}
			}
		}
		if active == "" {
			active = localConnectionID
		}
	}
	if connectionByID(p.Connections, active) == nil {
		active = localConnectionID
	}
	p.ActiveConnectionID = active
	mirrorActiveConnection(p)
}

func connectionByID(list []connectionProfile, id string) *connectionProfile {
	id = strings.TrimSpace(id)
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

func mirrorActiveConnection(p *guiPrefs) {
	c := connectionByID(p.Connections, p.ActiveConnectionID)
	if c == nil || c.Kind != connectionKindRemote || strings.TrimSpace(c.URL) == "" {
		p.Connect = ""
		p.AccessSecret = ""
		return
	}
	p.Connect = c.URL
	p.AccessSecret = c.AccessSecret
}

func (p *guiPrefs) publicConnections() []map[string]any {
	normalizeConnectionProfiles(p)
	out := make([]map[string]any, 0, len(p.Connections))
	for _, c := range p.Connections {
		item := map[string]any{
			"id":   c.ID,
			"name": c.Name,
			"kind": c.Kind,
		}
		if c.Kind == connectionKindRemote {
			item["url"] = c.URL
			item["access_secret_set"] = c.AccessSecret != ""
			// Return secret only to the authenticated UI that owns gui-prefs
			// (needed to reconnect). Still stored locally with 0600.
			item["access_secret"] = c.AccessSecret
		}
		out = append(out, item)
	}
	return out
}
