package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const skillOriginFile = ".keimling-origin.json"

// SkillOrigin records where an installed skill came from (for update checks).
type SkillOrigin struct {
	SourceURL string `json:"source_url"`
	Hash      string `json:"hash"`
	FetchedAt string `json:"fetched_at"`
}

// SkillUpdateStatus is the result of comparing a local skill to its remote source.
type SkillUpdateStatus struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // up_to_date | update_available | no_source | unavailable
	LocalHash  string `json:"local_hash,omitempty"`
	RemoteHash string `json:"remote_hash,omitempty"`
	SourceURL  string `json:"source_url,omitempty"`
	Message    string `json:"message,omitempty"`
}

func skillOriginPath(root string) string {
	return filepath.Join(root, skillOriginFile)
}

func hashSkillDir(root string) (string, error) {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if filepath.Base(rel) == skillOriginFile {
			return nil
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, rel := range paths {
		_, _ = io.WriteString(h, rel)
		_, _ = h.Write([]byte{0})
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		_, _ = h.Write(b)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeSkillOrigin(root, sourceURL string) error {
	sourceURL = strings.TrimSpace(sourceURL)
	if root == "" {
		return fmt.Errorf("empty root")
	}
	hash, err := hashSkillDir(root)
	if err != nil {
		return err
	}
	o := SkillOrigin{
		SourceURL: sourceURL,
		Hash:      hash,
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}
	b, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(skillOriginPath(root), b, 0o600)
}

func readSkillOrigin(root string) (*SkillOrigin, error) {
	b, err := os.ReadFile(skillOriginPath(root))
	if err != nil {
		return nil, err
	}
	var o SkillOrigin
	if err := json.Unmarshal(b, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func attachSkillOrigin(sk *SkillMeta) {
	if sk == nil || sk.Root == "" {
		return
	}
	if o, err := readSkillOrigin(sk.Root); err == nil && o != nil {
		sk.OriginURL = strings.TrimSpace(o.SourceURL)
		sk.ContentHash = o.Hash
	}
}

func DeleteSkill(name, cwd string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("skill name required")
	}
	reg := LoadSkillRegistry(cwd)
	sk := reg.Get(name)
	if sk == nil {
		return fmt.Errorf("skill %q not found", name)
	}
	if !sk.Writable {
		return fmt.Errorf("skill %q is read-only (%s)", name, sk.Source)
	}
	if sk.Root == "" {
		return fmt.Errorf("skill %q has no root", name)
	}
	return os.RemoveAll(sk.Root)
}

func CheckSkillUpdate(ctx context.Context, name, cwd string) (SkillUpdateStatus, error) {
	name = strings.TrimSpace(name)
	st := SkillUpdateStatus{Name: name}
	reg := LoadSkillRegistry(cwd)
	sk := reg.Get(name)
	if sk == nil {
		return st, fmt.Errorf("skill %q not found", name)
	}
	origin, err := readSkillOrigin(sk.Root)
	if err != nil || origin == nil || strings.TrimSpace(origin.SourceURL) == "" {
		st.Status = "no_source"
		st.Message = "no recorded download URL (local path install or missing origin metadata)"
		if h, e := hashSkillDir(sk.Root); e == nil {
			st.LocalHash = h
		}
		return st, nil
	}
	st.SourceURL = origin.SourceURL
	localHash, err := hashSkillDir(sk.Root)
	if err != nil {
		return st, err
	}
	st.LocalHash = localHash

	tmp, err := os.MkdirTemp("", "keimling-skill-check-*")
	if err != nil {
		return st, err
	}
	defer os.RemoveAll(tmp)
	remote, _, err := fetchSkillBundle(ctx, origin.SourceURL, tmp)
	if err != nil {
		st.Status = "unavailable"
		st.Message = err.Error()
		return st, nil
	}
	remoteHash, err := hashSkillDir(filepath.Join(tmp, remote.Name))
	if err != nil {
		return st, err
	}
	st.RemoteHash = remoteHash
	if remoteHash == localHash {
		st.Status = "up_to_date"
		st.Message = "matches remote source"
	} else {
		st.Status = "update_available"
		st.Message = "remote content differs from installed skill"
	}
	return st, nil
}

func UpdateSkill(ctx context.Context, name, cwd string) (*SkillMeta, SkillUpdateStatus, error) {
	st, err := CheckSkillUpdate(ctx, name, cwd)
	if err != nil {
		return nil, st, err
	}
	if st.Status == "no_source" {
		return nil, st, fmt.Errorf("%s", st.Message)
	}
	if st.Status == "unavailable" {
		return nil, st, fmt.Errorf("remote unavailable: %s", st.Message)
	}
	if st.Status == "up_to_date" {
		reg := LoadSkillRegistry(cwd)
		return reg.Get(name), st, nil
	}
	reg := LoadSkillRegistry(cwd)
	sk := reg.Get(name)
	if sk == nil {
		return nil, st, fmt.Errorf("skill %q not found", name)
	}
	if !sk.Writable {
		return nil, st, fmt.Errorf("skill %q is read-only", name)
	}
	scope := SkillScopeUser
	if sk.Source == skillSourceProject {
		scope = SkillScopeProject
	}
	updated, err := InstallSkillFromURL(st.SourceURL, scope, cwd)
	if err != nil {
		return nil, st, err
	}
	st.Status = "updated"
	st.Message = "installed latest from source"
	if h, e := hashSkillDir(updated.Root); e == nil {
		st.LocalHash = h
		st.RemoteHash = h
	}
	return updated, st, nil
}
