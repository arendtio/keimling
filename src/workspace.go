package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	workspaceGlobalID   = "global"
	workspaceMetaFile   = "meta.json"
	workspaceActiveFile = "active_thread"
	threadMetaFile      = "thread.json"
	defaultThreadTitle  = "New thread"
)

// WorkspaceMeta is persisted under ~/.local/share/keimling/workspaces/<id>/meta.json.
type WorkspaceMeta struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Root      string    `json:"root"`
	Builtin   bool      `json:"builtin,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ThreadMeta struct {
	ID                    string    `json:"id"`
	Title                 string    `json:"title,omitempty"`
	UpdatedAt             time.Time `json:"updated_at"`
	Active                bool      `json:"active,omitempty"`
	AutopilotInstructions string    `json:"autopilot_instructions,omitempty"`
}

func dataHome() string {
	if x := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); x != "" {
		return x
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

func workspacesRoot() string {
	return filepath.Join(dataHome(), "keimling", "workspaces")
}

func workspaceDir(id string) string {
	return filepath.Join(workspacesRoot(), id)
}

func workspaceArtifactsDir(id string) string {
	return filepath.Join(workspaceDir(id), "artifacts")
}

func workspaceThreadsDir(id string) string {
	return filepath.Join(workspaceDir(id), "threads")
}

func threadDir(workspaceID, threadID string) string {
	return filepath.Join(workspaceThreadsDir(workspaceID), threadID)
}

func threadSessionPath(workspaceID, threadID string) string {
	return filepath.Join(threadDir(workspaceID, threadID), "session.jsonl")
}

func workspaceIDForRoot(root string) string {
	root = canonicalPath(root)
	home, _ := os.UserHomeDir()
	if home != "" && canonicalPath(home) == root {
		return workspaceGlobalID
	}
	sum := sha1.Sum([]byte(root))
	return hex.EncodeToString(sum[:8])
}

func canonicalPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

func defaultWorkspaceName(root string) string {
	base := filepath.Base(root)
	if base == "" || base == string(filepath.Separator) || base == "." {
		return "Workspace"
	}
	return base
}

func ensureWorkspaceDir(id string) error {
	for _, d := range []string{
		workspaceDir(id),
		workspaceArtifactsDir(id),
		workspaceThreadsDir(id),
	} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func writeWorkspaceMeta(m WorkspaceMeta) error {
	if err := ensureWorkspaceDir(m.ID); err != nil {
		return err
	}
	m.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := filepath.Join(workspaceDir(m.ID), workspaceMetaFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readWorkspaceMeta(id string) (WorkspaceMeta, error) {
	var m WorkspaceMeta
	b, err := os.ReadFile(filepath.Join(workspaceDir(id), workspaceMetaFile))
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, err
	}
	return m, nil
}

func ensureGlobalWorkspace() (WorkspaceMeta, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return WorkspaceMeta{}, fmt.Errorf("home directory required for global workspace")
	}
	home = canonicalPath(home)
	m := WorkspaceMeta{
		ID:      workspaceGlobalID,
		Name:    "Home",
		Root:    home,
		Builtin: true,
	}
	if existing, err := readWorkspaceMeta(workspaceGlobalID); err == nil {
		if existing.Root == "" {
			existing.Root = home
		}
		existing.Builtin = true
		if existing.Name == "" {
			existing.Name = "Home"
		}
		_ = writeWorkspaceMeta(existing)
		return existing, nil
	}
	if err := writeWorkspaceMeta(m); err != nil {
		return WorkspaceMeta{}, err
	}
	return m, nil
}

func ensureWorkspaceForRoot(root string) (WorkspaceMeta, error) {
	root = canonicalPath(root)
	if root == "" {
		return WorkspaceMeta{}, fmt.Errorf("empty workspace root")
	}
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return WorkspaceMeta{}, fmt.Errorf("workspace root must be an existing directory: %s", root)
	}
	id := workspaceIDForRoot(root)
	if id == workspaceGlobalID {
		return ensureGlobalWorkspace()
	}
	if existing, err := readWorkspaceMeta(id); err == nil {
		existing.Root = root
		if existing.Name == "" {
			existing.Name = defaultWorkspaceName(root)
		}
		_ = writeWorkspaceMeta(existing)
		return existing, nil
	}
	m := WorkspaceMeta{
		ID:   id,
		Name: defaultWorkspaceName(root),
		Root: root,
	}
	if err := writeWorkspaceMeta(m); err != nil {
		return WorkspaceMeta{}, err
	}
	return m, nil
}

func listWorkspaces() ([]WorkspaceMeta, error) {
	if _, err := ensureGlobalWorkspace(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(workspacesRoot())
	if err != nil {
		return nil, err
	}
	out := make([]WorkspaceMeta, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m, err := readWorkspaceMeta(e.Name())
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ID == workspaceGlobalID {
			return true
		}
		if out[j].ID == workspaceGlobalID {
			return false
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func newThreadID() string {
	return time.Now().UTC().Format("20060102-150405.000000000")
}

func listThreads(workspaceID string) ([]ThreadMeta, error) {
	dir := workspaceThreadsDir(workspaceID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	active, _ := readActiveThreadID(workspaceID)
	out := make([]ThreadMeta, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		session := threadSessionPath(workspaceID, id)
		st, err := os.Stat(session)
		updated := time.Time{}
		if err == nil {
			updated = st.ModTime().UTC()
		} else if info, err2 := e.Info(); err2 == nil {
			updated = info.ModTime().UTC()
		}
		meta, _ := readThreadMeta(workspaceID, id)
		title := strings.TrimSpace(meta.Title)
		if title == "" {
			title = defaultThreadTitle
		}
		if !meta.UpdatedAt.IsZero() {
			updated = meta.UpdatedAt
		}
		out = append(out, ThreadMeta{
			ID:        id,
			Title:     title,
			UpdatedAt: updated,
			Active:    id == active,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func threadMetaPath(workspaceID, threadID string) string {
	return filepath.Join(threadDir(workspaceID, threadID), threadMetaFile)
}

func readThreadMeta(workspaceID, threadID string) (ThreadMeta, error) {
	var m ThreadMeta
	b, err := os.ReadFile(threadMetaPath(workspaceID, threadID))
	if err != nil {
		m.ID = threadID
		m.Title = defaultThreadTitle
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		m.ID = threadID
		m.Title = defaultThreadTitle
		return m, err
	}
	m.ID = threadID
	if strings.TrimSpace(m.Title) == "" {
		m.Title = defaultThreadTitle
	}
	return m, nil
}

func writeThreadMeta(workspaceID string, m ThreadMeta) error {
	if err := os.MkdirAll(threadDir(workspaceID, m.ID), 0o700); err != nil {
		return err
	}
	m.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := threadMetaPath(workspaceID, m.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func isProvisionalThreadTitle(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" || strings.EqualFold(title, defaultThreadTitle) {
		return true
	}
	// Legacy: id used as title before named threads.
	if _, err := time.Parse("20060102-150405.000000000", title); err == nil {
		return true
	}
	return false
}

// searchThreads returns threads whose title/id or journal text match query (case-insensitive).
func searchThreads(workspaceID, query string) ([]ThreadMeta, error) {
	query = strings.TrimSpace(query)
	all, err := listThreads(workspaceID)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return all, nil
	}
	q := strings.ToLower(query)
	out := make([]ThreadMeta, 0, len(all))
	for _, th := range all {
		if threadMatchesQuery(workspaceID, th, q) {
			out = append(out, th)
		}
	}
	return out, nil
}

func threadMatchesQuery(workspaceID string, th ThreadMeta, qLower string) bool {
	if strings.Contains(strings.ToLower(th.Title), qLower) || strings.Contains(strings.ToLower(th.ID), qLower) {
		return true
	}
	events, err := originalEvents(threadSessionPath(workspaceID, th.ID))
	if err != nil {
		return false
	}
	for _, raw := range events {
		if strings.Contains(strings.ToLower(threadSearchText(raw)), qLower) {
			return true
		}
	}
	return false
}

func threadSearchText(raw json.RawMessage) string {
	if label := eventLabel(raw); label != "" {
		return label
	}
	var v struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
		Type    string          `json:"type"`
		Output  string          `json:"output"`
	}
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	var parts []string
	if v.Role == "assistant" || v.Role == "user" {
		var s string
		if json.Unmarshal(v.Content, &s) == nil {
			parts = append(parts, s)
		} else {
			var chunks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if json.Unmarshal(v.Content, &chunks) == nil {
				for _, c := range chunks {
					if c.Text != "" {
						parts = append(parts, c.Text)
					}
				}
			}
		}
	}
	if v.Type == "function_call_output" && v.Output != "" {
		parts = append(parts, v.Output)
	}
	return strings.Join(parts, "\n")
}

// deleteThreadDir permanently removes a thread's journal and metadata.
func deleteThreadDir(workspaceID, threadID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	threadID = strings.TrimSpace(threadID)
	if workspaceID == "" || threadID == "" {
		return fmt.Errorf("workspace and thread required")
	}
	dir := threadDir(workspaceID, threadID)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("unknown thread %q", threadID)
		}
		return err
	}
	return os.RemoveAll(dir)
}

func readActiveThreadID(workspaceID string) (string, error) {
	b, err := os.ReadFile(filepath.Join(workspaceDir(workspaceID), workspaceActiveFile))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func writeActiveThreadID(workspaceID, threadID string) error {
	if err := ensureWorkspaceDir(workspaceID); err != nil {
		return err
	}
	path := filepath.Join(workspaceDir(workspaceID), workspaceActiveFile)
	return os.WriteFile(path, []byte(threadID+"\n"), 0o600)
}

func ensureActiveThread(workspaceID string) (threadID string, sessionPath string, err error) {
	if err = ensureWorkspaceDir(workspaceID); err != nil {
		return "", "", err
	}
	if id, e := readActiveThreadID(workspaceID); e == nil && id != "" {
		session := threadSessionPath(workspaceID, id)
		if _, err := os.Stat(filepath.Dir(session)); err == nil {
			return id, session, nil
		}
	}
	threads, err := listThreads(workspaceID)
	if err != nil {
		return "", "", err
	}
	if len(threads) > 0 {
		id := threads[0].ID
		_ = writeActiveThreadID(workspaceID, id)
		return id, threadSessionPath(workspaceID, id), nil
	}
	id := newThreadID()
	if err := os.MkdirAll(threadDir(workspaceID, id), 0o700); err != nil {
		return "", "", err
	}
	_ = writeThreadMeta(workspaceID, ThreadMeta{ID: id, Title: defaultThreadTitle})
	_ = writeActiveThreadID(workspaceID, id)
	return id, threadSessionPath(workspaceID, id), nil
}

func createThread(workspaceID string) (ThreadMeta, string, error) {
	id := newThreadID()
	if err := os.MkdirAll(threadDir(workspaceID, id), 0o700); err != nil {
		return ThreadMeta{}, "", err
	}
	meta := ThreadMeta{ID: id, Title: defaultThreadTitle, UpdatedAt: time.Now().UTC(), Active: true}
	if err := writeThreadMeta(workspaceID, meta); err != nil {
		return ThreadMeta{}, "", err
	}
	if err := writeActiveThreadID(workspaceID, id); err != nil {
		return ThreadMeta{}, "", err
	}
	session := threadSessionPath(workspaceID, id)
	return meta, session, nil
}

// resolveStartupWorkspace picks global or a workspace for launchCWD and chdirs to its root.
func resolveStartupWorkspace(launchCWD string) (WorkspaceMeta, string, string, error) {
	global, err := ensureGlobalWorkspace()
	if err != nil {
		return WorkspaceMeta{}, "", "", err
	}
	cwd := canonicalPath(launchCWD)
	if cwd == "" {
		if d, e := os.Getwd(); e == nil {
			cwd = canonicalPath(d)
		}
	}
	var ws WorkspaceMeta
	if cwd == "" || cwd == canonicalPath(global.Root) {
		ws = global
	} else {
		ws, err = ensureWorkspaceForRoot(cwd)
		if err != nil {
			return WorkspaceMeta{}, "", "", err
		}
	}
	if err := os.Chdir(ws.Root); err != nil {
		return WorkspaceMeta{}, "", "", fmt.Errorf("chdir workspace %s: %w", ws.Root, err)
	}
	threadID, session, err := ensureActiveThread(ws.ID)
	if err != nil {
		return WorkspaceMeta{}, "", "", err
	}
	_ = writeWorkspaceMeta(ws) // touch UpdatedAt
	return ws, threadID, session, nil
}
