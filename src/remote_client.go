package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type remoteClient struct {
	base   string
	secret string
	http   *http.Client
}

func newRemoteClient(base, secret string) (*remoteClient, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(base), "/"))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid --connect URL %q", base)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("--connect URL must be http or https")
	}
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("--access-secret is required with --connect")
	}
	return &remoteClient{
		base:   strings.TrimRight(u.String(), "/"),
		secret: secret,
		http:   &http.Client{Timeout: 0},
	}, nil
}

func (c *remoteClient) auth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.secret)
}

func (c *remoteClient) Health(ctx context.Context) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/health", nil)
	if err != nil {
		return nil, err
	}
	c.auth(req)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health: %s: %s", res.Status, truncate(string(body), 200))
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) Session(ctx context.Context) (lines []string, model, mode string, busy bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/session", nil)
	if err != nil {
		return nil, "", "", false, err
	}
	c.auth(req)
	cli := *c.http
	cli.Timeout = 30 * time.Second
	res, err := cli.Do(req)
	if err != nil {
		return nil, "", "", false, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode != http.StatusOK {
		return nil, "", "", false, fmt.Errorf("session: %s: %s", res.Status, truncate(string(body), 200))
	}
	var payload struct {
		Lines []string `json:"lines"`
		Model string   `json:"model"`
		Mode  string   `json:"mode"`
		Busy  bool     `json:"busy"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", "", false, err
	}
	return payload.Lines, payload.Model, payload.Mode, payload.Busy, nil
}

func (c *remoteClient) Message(ctx context.Context, text string) error {
	b, _ := json.Marshal(map[string]string{"text": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/message", bytes.NewReader(b))
	if err != nil {
		return err
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
	cli := *c.http
	cli.Timeout = 30 * time.Second
	res, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != http.StatusAccepted && res.StatusCode != http.StatusOK {
		return fmt.Errorf("message: %s: %s", res.Status, truncate(string(body), 200))
	}
	return nil
}

func (c *remoteClient) Approve(ctx context.Context, allow bool) error {
	b, _ := json.Marshal(map[string]bool{"allow": allow})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/approve", bytes.NewReader(b))
	if err != nil {
		return err
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
	cli := *c.http
	cli.Timeout = 15 * time.Second
	res, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("approve: %s: %s", res.Status, truncate(string(body), 200))
	}
	return nil
}

type sseHandler struct {
	OnReady  func()
	OnLine   func(text string)
	OnStatus func(state string)
}

func (c *remoteClient) StreamEvents(ctx context.Context, h sseHandler) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/events", nil)
	if err != nil {
		return err
	}
	c.auth(req)
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		return fmt.Errorf("events: %s: %s", res.Status, truncate(string(body), 200))
	}
	return readSSE(res.Body, h)
}

func readSSE(r io.Reader, h sseHandler) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 2<<20)
	var event string
	var data strings.Builder
	flush := func() {
		ev := event
		payload := strings.TrimSpace(data.String())
		event = ""
		data.Reset()
		if ev == "" && payload == "" {
			return
		}
		if ev == "" {
			ev = "message"
		}
		switch ev {
		case "ready":
			if h.OnReady != nil {
				h.OnReady()
			}
		case "line":
			var m struct {
				Text string `json:"text"`
			}
			if json.Unmarshal([]byte(payload), &m) == nil && h.OnLine != nil {
				h.OnLine(m.Text)
			}
		case "status":
			var m struct {
				State string `json:"state"`
			}
			if json.Unmarshal([]byte(payload), &m) == nil && h.OnStatus != nil {
				h.OnStatus(m.State)
			}
		}
	}
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	flush()
	return sc.Err()
}
