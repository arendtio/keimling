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
func (c *Client) authorize(req *http.Request) {
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
}
