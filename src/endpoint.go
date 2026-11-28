package main

import (
	"net/http"
	"strings"
)

const defaultBaseURL = "https://api.openai.com/v1"

func (c *Client) endpoint(path string) string {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	return base + "/" + strings.TrimLeft(path, "/")
}

// openCodeSessionHost reports a host that wants x-opencode-session.
func openCodeSessionHost(host string) bool {
	h := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	return h == "opencode.ai" || strings.HasSuffix(h, ".opencode.ai")
}

func (c *Client) authorize(req *http.Request) {
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	// Name the client rather than a generic library, as providers ask. The
	// commit stamp doubles as the version.
	req.Header.Set("User-Agent", "keimling/"+buildVersion)
	// opencode routes and caches per conversation; other providers ignore it.
	if c.SessionID != "" && openCodeSessionHost(req.URL.Hostname()) {
		req.Header.Set("x-opencode-session", c.SessionID)
	}
}
