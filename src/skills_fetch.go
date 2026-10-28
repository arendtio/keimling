package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	skillFetchMaxFiles = 100
	skillFetchMaxBytes = 12 << 20
	skillFetchFileCap  = 1 << 20
)

type githubSkillLoc struct {
	Owner string
	Repo  string
	Ref   string
	Dir   string // directory containing SKILL.md (slash-separated, no trailing slash)
}

type SkillFileInfo struct {
	Path string `json:"path"`
	Size int    `json:"size"`
}

var (
	relMarkdownLinkRe = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	relPathTokenRe    = regexp.MustCompile("`([A-Za-z0-9_./-]+\\.[A-Za-z0-9]+)`")
	relBarePathRe     = regexp.MustCompile(`(?m)(?:^|[\s(\"'])((?:references|scripts|assets|templates|examples)\/[A-Za-z0-9_./-]+)`)
)

func parseGitHubSkillLocation(raw string) (githubSkillLoc, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return githubSkillLoc{}, err
	}
	host := strings.ToLower(u.Host)
	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")
	loc := githubSkillLoc{Ref: "main"}

	if host == "raw.githubusercontent.com" {
		if len(parts) < 4 {
			return githubSkillLoc{}, fmt.Errorf("unrecognized raw GitHub URL")
		}
		loc.Owner, loc.Repo, loc.Ref = parts[0], parts[1], parts[2]
		rest := parts[3:]
		if len(rest) == 0 {
			return githubSkillLoc{}, fmt.Errorf("missing path in raw URL")
		}
		if strings.EqualFold(rest[len(rest)-1], "SKILL.md") {
			rest = rest[:len(rest)-1]
		}
		loc.Dir = strings.Join(rest, "/")
		return loc, nil
	}
	if host != "github.com" && host != "www.github.com" {
		return githubSkillLoc{}, fmt.Errorf("not a GitHub URL")
	}
	if len(parts) < 2 {
		return githubSkillLoc{}, fmt.Errorf("unrecognized GitHub URL")
	}
	loc.Owner, loc.Repo = parts[0], strings.TrimSuffix(parts[1], ".git")
	if len(parts) >= 4 && (parts[2] == "tree" || parts[2] == "blob") {
		loc.Ref = parts[3]
		rest := parts[4:]
		if parts[2] == "blob" && len(rest) > 0 && strings.EqualFold(rest[len(rest)-1], "SKILL.md") {
			rest = rest[:len(rest)-1]
		}
		loc.Dir = strings.Join(rest, "/")
	}
	return loc, nil
}

func (loc githubSkillLoc) rawFile(rel string) string {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "/")
	base := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", loc.Owner, loc.Repo, loc.Ref)
	if loc.Dir != "" {
		base += "/" + loc.Dir
	}
	return base + "/" + rel
}

func extractSkillRelativePaths(md string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		raw = strings.SplitN(raw, "#", 2)[0]
		raw = strings.SplitN(raw, "?", 2)[0]
		raw = strings.Trim(raw, " \"'")
		if raw == "" || strings.Contains(raw, "://") || strings.HasPrefix(raw, "/") {
			return
		}
		raw = strings.TrimPrefix(raw, "./")
		if raw == "" || strings.Contains(raw, "..") {
			return
		}
		raw = filepath.ToSlash(raw)
		if raw == "SKILL.md" || seen[raw] {
			return
		}
		seen[raw] = true
		out = append(out, raw)
	}
	for _, m := range relMarkdownLinkRe.FindAllStringSubmatch(md, -1) {
		add(m[1])
	}
	for _, m := range relPathTokenRe.FindAllStringSubmatch(md, -1) {
		add(m[1])
	}
	for _, m := range relBarePathRe.FindAllStringSubmatch(md, -1) {
		add(m[1])
	}
	sort.Strings(out)
	return out
}

type ghTreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type ghTreeResp struct {
	Tree []ghTreeEntry `json:"tree"`
}

func githubListSkillFiles(ctx context.Context, loc githubSkillLoc) ([]string, error) {
	api := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", loc.Owner, loc.Repo, url.PathEscape(loc.Ref))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "keimling-skills/1.0")
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 45 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("GitHub API HTTP %d: %s", res.StatusCode, truncate(string(body), 200))
	}
	var tree ghTreeResp
	if err := json.Unmarshal(body, &tree); err != nil {
		return nil, err
	}
	prefix := loc.Dir
	if prefix != "" {
		prefix += "/"
	}
	var files []string
	for _, e := range tree.Tree {
		if e.Type != "blob" {
			continue
		}
		p := e.Path
		if loc.Dir == "" {
			if p == "SKILL.md" || strings.HasPrefix(p, "references/") || strings.HasPrefix(p, "scripts/") ||
				strings.HasPrefix(p, "assets/") || strings.HasPrefix(p, "templates/") || strings.HasPrefix(p, "examples/") {
				files = append(files, p)
			}
			continue
		}
		if p == loc.Dir+"/SKILL.md" || strings.HasPrefix(p, prefix) {
			rel := strings.TrimPrefix(p, prefix)
			if rel != "" {
				files = append(files, rel)
			}
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no files under skill path")
	}
	if len(files) > skillFetchMaxFiles {
		files = files[:skillFetchMaxFiles]
	}
	sort.Strings(files)
	return files, nil
}

func writeSkillFile(root, rel string, data []byte) error {
	rel = filepath.Clean(filepath.FromSlash(rel))
	if rel == "." || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("invalid relative path %q", rel)
	}
	target := filepath.Join(root, rel)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if absTarget != absRoot && !strings.HasPrefix(absTarget, absRoot+string(os.PathSeparator)) {
		return fmt.Errorf("path escapes skill root")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o600)
}

func listSkillFiles(root string) ([]SkillFileInfo, error) {
	var files []SkillFileInfo
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, SkillFileInfo{Path: filepath.ToSlash(rel), Size: int(info.Size())})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// fetchSkillBundle downloads SKILL.md plus bundled files into destRoot/<skill-name>/.
// Prefer GitHub tree listing; fall back to following relative links in markdown.
func fetchSkillBundle(ctx context.Context, rawURL, destRoot string) (*SkillMeta, []SkillFileInfo, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, nil, fmt.Errorf("empty url")
	}
	if err := os.MkdirAll(destRoot, 0o700); err != nil {
		return nil, nil, err
	}

	loc, locErr := parseGitHubSkillLocation(rawURL)
	skillURL := rawURL
	if locErr == nil {
		skillURL = loc.rawFile("SKILL.md")
	} else if isGitHubSkillURL(rawURL) {
		if mapped, _, err := githubRawSkillURL(rawURL); err == nil {
			skillURL = mapped
		}
	}

	skillData, err := httpGetLimited(skillURL, skillFetchFileCap)
	if err != nil {
		return nil, nil, fmt.Errorf("download SKILL.md: %w", err)
	}
	sk, err := parseSkillMarkdown(string(skillData), "", "", "preview", false)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid remote SKILL.md: %w", err)
	}
	skillDir := filepath.Join(destRoot, sk.Name)
	_ = os.RemoveAll(skillDir)
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		return nil, nil, err
	}
	if err := writeSkillFile(skillDir, "SKILL.md", skillData); err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, nil, err
	}

	var rels []string
	if locErr == nil {
		if listed, listErr := githubListSkillFiles(ctx, loc); listErr == nil {
			rels = listed
		}
	}
	if len(rels) == 0 {
		rels = extractSkillRelativePaths(string(skillData))
	} else {
		// Ensure linked paths are included even if tree filter missed them.
		for _, p := range extractSkillRelativePaths(string(skillData)) {
			found := false
			for _, r := range rels {
				if r == p {
					found = true
					break
				}
			}
			if !found {
				rels = append(rels, p)
			}
		}
	}

	total := len(skillData)
	fetched := map[string]bool{"SKILL.md": true}
	queue := append([]string{}, rels...)
	for i := 0; i < len(queue) && len(fetched) < skillFetchMaxFiles && total < skillFetchMaxBytes; i++ {
		rel := queue[i]
		if fetched[rel] {
			continue
		}
		var fileURL string
		if locErr == nil {
			fileURL = loc.rawFile(rel)
		} else {
			fileURL = resolveSiblingURL(skillURL, rel)
			if fileURL == "" {
				continue
			}
		}
		data, getErr := httpGetLimited(fileURL, skillFetchFileCap)
		if getErr != nil {
			continue // missing optional assets should not fail the whole skill
		}
		if err := writeSkillFile(skillDir, rel, data); err != nil {
			continue
		}
		fetched[rel] = true
		total += len(data)
		if strings.HasSuffix(strings.ToLower(rel), ".md") {
			for _, more := range extractSkillRelativePaths(string(data)) {
				base := filepath.ToSlash(filepath.Dir(rel))
				joined := more
				if base != "." && !strings.Contains(more, "/") {
					joined = base + "/" + more
				} else if base != "." && strings.HasPrefix(more, "./") {
					joined = base + "/" + strings.TrimPrefix(more, "./")
				}
				if !fetched[joined] {
					queue = append(queue, joined)
				}
			}
		}
	}

	// Re-parse with directory name enforced.
	sk, err = loadSkillDir(skillDir, "preview", false)
	if err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, nil, err
	}
	files, err := listSkillFiles(skillDir)
	if err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, nil, err
	}
	return sk, files, nil
}

func resolveSiblingURL(skillURL, rel string) string {
	u, err := url.Parse(skillURL)
	if err != nil {
		return ""
	}
	// Drop SKILL.md filename.
	path := strings.TrimSuffix(u.Path, "/")
	if i := strings.LastIndex(path, "/"); i >= 0 {
		path = path[:i+1] + strings.TrimPrefix(filepath.ToSlash(rel), "/")
	} else {
		return ""
	}
	u.Path = path
	return u.String()
}
