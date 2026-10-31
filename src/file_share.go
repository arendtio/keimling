package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const maxShareBytes = 32 << 20 // 32 MiB

type fileShareEntry struct {
	Path        string
	Name        string
	Size        int64
	Disposition string // "inline", "attachment", or "" for auto
}

// fileShareHub maps capability tokens to files for GET /files/{token}.
type fileShareHub struct {
	mu         sync.Mutex
	byToken    map[string]fileShareEntry
	publicBase string // scheme://host, no trailing slash
}

func newFileShareHub() *fileShareHub {
	return &fileShareHub{byToken: make(map[string]fileShareEntry)}
}

func (h *fileShareHub) SetPublicBase(base string) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return
	}
	h.mu.Lock()
	h.publicBase = base
	h.mu.Unlock()
}

func (h *fileShareHub) PublicBase() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.publicBase
}

func (h *fileShareHub) Clear() {
	h.mu.Lock()
	h.byToken = make(map[string]fileShareEntry)
	h.mu.Unlock()
}

func (h *fileShareHub) Lookup(token string) (fileShareEntry, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	e, ok := h.byToken[strings.TrimSpace(token)]
	return e, ok
}

func normalizeShareAs(as string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(as)) {
	case "", "auto":
		return "", nil
	case "view", "inline", "open":
		return "inline", nil
	case "download", "attachment", "save":
		return "attachment", nil
	default:
		return "", fmt.Errorf("as must be auto, view, or download")
	}
}

func (h *fileShareHub) Register(path, as string) (token, url string, err error) {
	return h.register(path, as, nil)
}

// register serves path when it is inside the working directory or one of extraRoots.
// Artifact snapshots live under the workspace data dir, outside the working directory.
func (h *fileShareHub) register(path, as string, extraRoots []string) (token, url string, err error) {
	disp, err := normalizeShareAs(as)
	if err != nil {
		return "", "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("invalid path")
	}
	if !pathAllowed(abs, extraRoots) {
		return "", "", fmt.Errorf("path outside working directory")
	}
	if privatePath(abs) {
		return "", "", fmt.Errorf("protected path")
	}
	st, err := os.Lstat(abs)
	if err != nil {
		return "", "", err
	}
	if !st.Mode().IsRegular() {
		return "", "", fmt.Errorf("not a regular file")
	}
	if st.Size() > maxShareBytes {
		return "", "", fmt.Errorf("file too large (max %d bytes)", maxShareBytes)
	}
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(b[:])
	entry := fileShareEntry{Path: abs, Name: filepath.Base(abs), Size: st.Size(), Disposition: disp}
	h.mu.Lock()
	h.byToken[token] = entry
	base := h.publicBase
	h.mu.Unlock()
	pathURL := "/files/" + token
	if base != "" {
		url = base + pathURL
	} else {
		url = pathURL
	}
	return token, url, nil
}

func pathAllowed(abs string, extraRoots []string) bool {
	if insideWorkingDir(abs) {
		return true
	}
	for _, root := range extraRoots {
		if strings.TrimSpace(root) != "" && pathInside(root, abs) {
			return true
		}
	}
	return false
}

func publicBaseFromRequest(r *http.Request) string {
	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if i := strings.IndexByte(proto, ','); i >= 0 {
		proto = strings.TrimSpace(proto[:i])
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if i := strings.IndexByte(host, ','); i >= 0 {
		host = strings.TrimSpace(host[:i])
	}
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return ""
	}
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	return proto + "://" + host
}

func shareInline(name, contentType string) bool {
	if strings.HasPrefix(contentType, "text/") || strings.HasPrefix(contentType, "image/") {
		return true
	}
	switch contentType {
	case "application/pdf", "application/json", "application/javascript":
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".html", ".htm", ".css", ".js", ".json", ".md", ".txt", ".csv", ".svg", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".pdf":
		return true
	}
	return false
}

func (s *appServer) handleFileShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := strings.TrimPrefix(r.URL.Path, "/files/")
	token = strings.Trim(token, "/")
	if token == "" || strings.Contains(token, "/") {
		http.NotFound(w, r)
		return
	}
	if s.agent == nil || s.agent.Shares == nil {
		http.NotFound(w, r)
		return
	}
	entry, ok := s.agent.Shares.Lookup(token)
	if !ok {
		http.NotFound(w, r)
		return
	}
	st, err := os.Stat(entry.Path)
	if err != nil || !st.Mode().IsRegular() || st.Size() > maxShareBytes {
		http.NotFound(w, r)
		return
	}
	ct := mime.TypeByExtension(filepath.Ext(entry.Name))
	if ct == "" {
		ct = "application/octet-stream"
	}
	disp := entry.Disposition
	if disp == "" {
		disp = "attachment"
		if shareInline(entry.Name, ct) {
			disp = "inline"
		}
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disp, strings.ReplaceAll(entry.Name, `"`, "")))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeFile(w, r, entry.Path)
}

func (a *Agent) shareFile(argsJSON string) string {
	var x struct {
		Path string `json:"path"`
		As   string `json:"as"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &x); err != nil || strings.TrimSpace(x.Path) == "" {
		return "share_file error: path required"
	}
	if a.Shares == nil {
		a.Shares = newFileShareHub()
	}
	if base := strings.TrimSpace(a.Cfg.PublicURL); base != "" {
		a.Shares.SetPublicBase(base)
	}
	token, url, err := a.Shares.Register(x.Path, x.As)
	if err != nil {
		return "share_file error: " + err.Error()
	}
	mode := "auto"
	if d, _ := normalizeShareAs(x.As); d == "inline" {
		mode = "view"
	} else if d == "attachment" {
		mode = "download"
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return fmt.Sprintf("shared %s (%s)\nurl: %s\ntoken: %s\nAnyone with this URL can access the file until the session ends or /newchat. Paste the URL for the user.", x.Path, mode, url, token)
	}
	return fmt.Sprintf("shared %s (%s)\npath: %s\ntoken: %s\nOpen https://<agent-host>%s (or set public_url). Capability URL; no access secret required.", x.Path, mode, url, token, url)
}
