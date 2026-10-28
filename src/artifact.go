package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxArtifacts = 32

// Artifact is an HTML (or other viewable) file shown in the GUI panel.
type Artifact struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Path    string    `json:"path"`
	URL     string    `json:"url"`
	Token   string    `json:"token"`
	Created time.Time `json:"created"`
}

type artifactHub struct {
	mu    sync.Mutex
	items []Artifact
}

func newArtifactHub() *artifactHub {
	return &artifactHub{}
}

func (h *artifactHub) Clear() {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.items = nil
	h.mu.Unlock()
}

func (h *artifactHub) List() []Artifact {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Artifact, len(h.items))
	copy(out, h.items)
	return out
}

func (h *artifactHub) Add(a Artifact) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.items = append([]Artifact{a}, h.items...)
	if len(h.items) > maxArtifacts {
		h.items = h.items[:maxArtifacts]
	}
}

func (a *Agent) ensureArtifacts() *artifactHub {
	if a.Artifacts == nil {
		a.Artifacts = newArtifactHub()
	}
	return a.Artifacts
}

func (a *Agent) showArtifact(argsJSON string) string {
	var x struct {
		Path  string `json:"path"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &x); err != nil || strings.TrimSpace(x.Path) == "" {
		return "show_artifact error: path required"
	}
	if a.Shares == nil {
		a.Shares = newFileShareHub()
	}
	if base := strings.TrimSpace(a.Cfg.PublicURL); base != "" {
		a.Shares.SetPublicBase(base)
	}
	storePath := strings.TrimSpace(x.Path)
	if a.Cfg.WorkspaceID != "" {
		if copied, err := persistWorkspaceArtifact(a.Cfg.WorkspaceID, x.Path); err == nil {
			storePath = copied
		}
	}
	token, url, err := a.Shares.Register(storePath, "view")
	if err != nil {
		return "show_artifact error: " + err.Error()
	}
	title := strings.TrimSpace(x.Title)
	if title == "" {
		title = filepath.Base(x.Path)
	}
	art := Artifact{
		ID:      token,
		Title:   title,
		Path:    storePath,
		URL:     url,
		Token:   token,
		Created: time.Now().UTC(),
	}
	a.ensureArtifacts().Add(art)
	a.emitJSON("artifact", art)
	a.println("\033[35m▣ Artifact\033[0m " + title + " → " + url)
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return fmt.Sprintf("artifact shown: %s\nurl: %s\nThe GUI opens this in the artifact panel (sandboxed iframe). Prefer self-contained HTML/CSS/JS.", title, url)
	}
	return fmt.Sprintf("artifact shown: %s\npath: %s\nGUI clients open /files/%s in the artifact panel. Prefer self-contained HTML.", title, url, token)
}

func persistWorkspaceArtifact(workspaceID, src string) (string, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return "", fmt.Errorf("empty path")
	}
	abs, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	in, err := os.Open(abs)
	if err != nil {
		return "", err
	}
	defer in.Close()
	dir := workspaceArtifactsDir(workspaceID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	base := filepath.Base(abs)
	dst := filepath.Join(dir, fmt.Sprintf("%s-%s", time.Now().UTC().Format("20060102-150405"), base))
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	return dst, nil
}
