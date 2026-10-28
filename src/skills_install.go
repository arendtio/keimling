package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SkillScope string

const (
	SkillScopeUser    SkillScope = "user"
	SkillScopeProject SkillScope = "project"
)

func skillsInstallDir(scope SkillScope, cwd string) (string, error) {
	switch scope {
	case SkillScopeProject:
		dir := projectSkillsDir(cwd)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", err
		}
		return dir, nil
	default:
		dir := userSkillsDir()
		if dir == "" {
			return "", fmt.Errorf("cannot resolve user skills directory")
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", err
		}
		return dir, nil
	}
}

// resolveSkillSourcePath accepts a skill directory or a path to SKILL.md.
func resolveSkillSourcePath(path string) (string, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return path, nil
	}
	if strings.EqualFold(filepath.Base(path), "SKILL.md") {
		return filepath.Dir(path), nil
	}
	return "", fmt.Errorf("path must be a skill directory or SKILL.md file")
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to copy symlink %s", path)
		}
		if info.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

// InstallSkillFromPath validates a local skill and copies it into the install dir.
func InstallSkillFromPath(path string, scope SkillScope, cwd string) (*SkillMeta, error) {
	src, err := resolveSkillSourcePath(path)
	if err != nil {
		return nil, err
	}
	sk, err := loadSkillDir(src, string(scope), true)
	if err != nil {
		return nil, fmt.Errorf("invalid skill: %w", err)
	}
	destRoot, err := skillsInstallDir(scope, cwd)
	if err != nil {
		return nil, err
	}
	dest := filepath.Join(destRoot, sk.Name)
	if absSrc, e1 := filepath.Abs(src); e1 == nil {
		if absDst, e2 := filepath.Abs(dest); e2 == nil && absSrc == absDst {
			return sk, nil // already installed in place
		}
	}
	tmpParent := filepath.Join(destRoot, ".tmp-"+sk.Name)
	tmp := filepath.Join(tmpParent, sk.Name)
	_ = os.RemoveAll(tmpParent)
	if err := os.MkdirAll(tmpParent, 0700); err != nil {
		return nil, err
	}
	if err := copyDir(src, tmp); err != nil {
		_ = os.RemoveAll(tmpParent)
		return nil, err
	}
	// Re-validate copy (directory name must match skill name).
	if _, err := loadSkillDir(tmp, string(scope), true); err != nil {
		_ = os.RemoveAll(tmpParent)
		return nil, fmt.Errorf("installed copy invalid: %w", err)
	}
	_ = os.RemoveAll(dest)
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.RemoveAll(tmpParent)
		return nil, err
	}
	_ = os.RemoveAll(tmpParent)
	return loadSkillDir(dest, string(scope), true)
}

func isGitHubSkillURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	return host == "github.com" || host == "www.github.com" || host == "raw.githubusercontent.com"
}

// githubRawSkillURL tries to map a GitHub blob/tree URL to a raw SKILL.md URL.
func githubRawSkillURL(raw string) (string, string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", "", err
	}
	host := strings.ToLower(u.Host)
	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")
	if host == "raw.githubusercontent.com" {
		// owner/repo/ref/.../SKILL.md
		if len(parts) < 4 {
			return "", "", fmt.Errorf("unrecognized raw GitHub URL")
		}
		return raw, parts[1], nil // name guess from repo — refined after download
	}
	if host != "github.com" && host != "www.github.com" {
		return "", "", fmt.Errorf("not a GitHub URL")
	}
	if len(parts) < 2 {
		return "", "", fmt.Errorf("unrecognized GitHub URL")
	}
	owner, repo := parts[0], parts[1]
	repo = strings.TrimSuffix(repo, ".git")
	ref := "main"
	sub := ""
	if len(parts) >= 4 && (parts[2] == "tree" || parts[2] == "blob") {
		ref = parts[3]
		if len(parts) > 4 {
			sub = strings.Join(parts[4:], "/")
		}
	}
	skillPath := "SKILL.md"
	if sub != "" {
		if strings.HasSuffix(strings.ToLower(sub), "skill.md") {
			skillPath = sub
		} else {
			skillPath = filepath.ToSlash(filepath.Join(sub, "SKILL.md"))
		}
	}
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, ref, skillPath)
	return rawURL, repo, nil
}

func httpGetLimited(urlStr string, limit int64) ([]byte, error) {
	client := &http.Client{Timeout: 45 * time.Second}
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "keimling-skills/1.0")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, truncate(string(b), 200))
	}
	return io.ReadAll(io.LimitReader(res.Body, limit))
}

// InstallSkillFromURL fetches a full skill bundle (SKILL.md + referenced/tree files) and installs it.
func InstallSkillFromURL(rawURL string, scope SkillScope, cwd string) (*SkillMeta, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("empty url")
	}
	tmpParent, err := os.MkdirTemp("", "keimling-skill-fetch-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpParent)
	sk, _, err := fetchSkillBundle(context.Background(), rawURL, tmpParent)
	if err != nil {
		return nil, err
	}
	installed, err := InstallSkillFromPath(filepath.Join(tmpParent, sk.Name), scope, cwd)
	if err != nil {
		return nil, err
	}
	if err := writeSkillOrigin(installed.Root, rawURL); err != nil {
		return installed, fmt.Errorf("installed but origin metadata failed: %w", err)
	}
	attachSkillOrigin(installed)
	return installed, nil
}
