package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SkillReview is an advisory model assessment of a staged skill preview.
type SkillReview struct {
	Summary      string   `json:"summary"`
	RiskLevel    string   `json:"risk_level"` // low | medium | high | unknown
	Concerns     []string `json:"concerns"`
	Dependencies []string `json:"dependencies"`
	OKToInstall  bool     `json:"ok_to_install"`
}

// SkillPreview is a staged download awaiting user review/install.
type SkillPreview struct {
	ID        string          `json:"id"`
	SourceURL string          `json:"source_url"`
	Root      string          `json:"-"`
	Skill     *SkillMeta      `json:"skill"`
	Files     []SkillFileInfo `json:"files"`
	Review    *SkillReview    `json:"review,omitempty"`
	Created   time.Time       `json:"created"`
}

type skillPreviewHub struct {
	mu   sync.Mutex
	byID map[string]*SkillPreview
}

func newSkillPreviewHub() *skillPreviewHub {
	return &skillPreviewHub{byID: map[string]*SkillPreview{}}
}

func skillPreviewCacheDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return filepath.Join(os.TempDir(), "keimling-skill-previews")
	}
	return filepath.Join(home, ".cache", "keimling", "skill-previews")
}

func (h *skillPreviewHub) Get(id string) *SkillPreview {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.byID[strings.TrimSpace(id)]
}

func (h *skillPreviewHub) Put(p *SkillPreview) {
	if h == nil || p == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.byID == nil {
		h.byID = map[string]*SkillPreview{}
	}
	h.byID[p.ID] = p
	cutoff := time.Now().Add(-6 * time.Hour)
	for id, old := range h.byID {
		if old.Created.Before(cutoff) {
			_ = os.RemoveAll(filepath.Dir(old.Root))
			delete(h.byID, id)
		}
	}
}

func newPreviewID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// StageSkillPreview downloads the full skill bundle into a cache directory for review.
func StageSkillPreview(ctx context.Context, hub *skillPreviewHub, sourceURL string) (*SkillPreview, error) {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return nil, fmt.Errorf("url required")
	}
	id := newPreviewID()
	dest := filepath.Join(skillPreviewCacheDir(), id)
	_ = os.RemoveAll(dest)
	sk, files, err := fetchSkillBundle(ctx, sourceURL, dest)
	if err != nil {
		_ = os.RemoveAll(dest)
		return nil, err
	}
	p := &SkillPreview{
		ID:        id,
		SourceURL: sourceURL,
		Root:      filepath.Join(dest, sk.Name),
		Skill:     sk,
		Files:     files,
		Created:   time.Now().UTC(),
	}
	if hub != nil {
		hub.Put(p)
	}
	return p, nil
}

func InstallSkillFromPreview(p *SkillPreview, scope SkillScope, cwd string) (*SkillMeta, error) {
	if p == nil || strings.TrimSpace(p.Root) == "" {
		return nil, fmt.Errorf("preview required")
	}
	installed, err := InstallSkillFromPath(p.Root, scope, cwd)
	if err != nil {
		return nil, err
	}
	if url := strings.TrimSpace(p.SourceURL); url != "" {
		if err := writeSkillOrigin(installed.Root, url); err != nil {
			return installed, fmt.Errorf("installed but origin metadata failed: %w", err)
		}
		attachSkillOrigin(installed)
	}
	return installed, nil
}

func ReadSkillPreviewFile(p *SkillPreview, rel string, limit int) (string, error) {
	if p == nil {
		return "", fmt.Errorf("preview required")
	}
	rel = filepath.Clean(filepath.FromSlash(strings.TrimSpace(rel)))
	if rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid path")
	}
	target := filepath.Join(p.Root, rel)
	absRoot, err := filepath.Abs(p.Root)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if absTarget != absRoot && !strings.HasPrefix(absTarget, absRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes preview root")
	}
	if limit <= 0 {
		limit = skillFileCap
	}
	b, err := os.ReadFile(target)
	if err != nil {
		return "", err
	}
	if len(b) > limit {
		b = b[:limit]
	}
	return string(b), nil
}

func ReviewSkillPreview(ctx context.Context, client *Client, model string, p *SkillPreview) (*SkillReview, error) {
	if client == nil {
		return nil, fmt.Errorf("model client unavailable")
	}
	if p == nil || p.Skill == nil {
		return nil, fmt.Errorf("preview required")
	}
	if model == "" {
		model = "gpt-5.4-mini"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Skill name: %s\nDescription: %s\nCompatibility: %s\nAllowed-tools: %s\nSource: %s\nFiles (%d):\n",
		p.Skill.Name, p.Skill.Description, p.Skill.Compatibility, p.Skill.AllowedTools, p.SourceURL, len(p.Files))
	for _, f := range p.Files {
		fmt.Fprintf(&b, "- %s (%d bytes)\n", f.Path, f.Size)
	}
	b.WriteString("\n--- SKILL.md ---\n")
	body, _ := ReadSkillPreviewFile(p, "SKILL.md", 40_000)
	b.WriteString(body)
	nExtra := 0
	for _, f := range p.Files {
		if f.Path == "SKILL.md" || f.Size > 20_000 {
			continue
		}
		lower := strings.ToLower(f.Path)
		if !strings.HasSuffix(lower, ".md") && !strings.HasSuffix(lower, ".txt") && !strings.HasSuffix(lower, ".sh") {
			continue
		}
		text, err := ReadSkillPreviewFile(p, f.Path, 12_000)
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "\n--- %s ---\n%s\n", f.Path, text)
		nExtra++
		if nExtra >= 6 {
			break
		}
	}

	instructions := `You review Agent Skills (agentskills.io) before a user installs them into Keimling.
Assess safety and required dependencies. Be concise and skeptical.
Flag: destructive filesystem/boot changes, secret exfiltration, opaque remote code, package installs, privilege escalation, social engineering.
List software/CLI dependencies the skill expects (from docs/scripts).
ok_to_install is advisory only (true if risk is low/medium with clear caveats).
Return ONLY JSON matching the schema.`

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"summary":       map[string]any{"type": "string"},
			"risk_level":    map[string]any{"type": "string", "enum": []string{"low", "medium", "high", "unknown"}},
			"concerns":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"dependencies":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"ok_to_install": map[string]any{"type": "boolean"},
		},
		"required":             []string{"summary", "risk_level", "concerns", "dependencies", "ok_to_install"},
		"additionalProperties": false,
	}
	payload := map[string]any{
		"model": model, "store": false, "instructions": instructions,
		"input": []map[string]any{{"role": "user", "content": b.String()}},
		"text":  map[string]any{"format": map[string]any{"type": "json_schema", "name": "skill_review", "strict": true, "schema": schema}},
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint("responses"), bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	client.authorize(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("review HTTP %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	text := ""
	var result struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if json.Unmarshal(data, &result) == nil {
		for _, item := range result.Output {
			for _, part := range item.Content {
				if part.Type == "output_text" && part.Text != "" {
					text = part.Text
					break
				}
			}
		}
	}
	var review SkillReview
	if err := json.Unmarshal([]byte(text), &review); err != nil {
		return nil, fmt.Errorf("invalid review JSON: %w", err)
	}
	switch review.RiskLevel {
	case "low", "medium", "high", "unknown":
	default:
		review.RiskLevel = "unknown"
	}
	p.Review = &review
	return &review, nil
}
