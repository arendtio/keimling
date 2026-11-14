package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// chatRef is a read-only name for a chat in any workspace: chat:<workspace>/<thread>.
// The workspace is its name or id. The thread is its title or id. A title may contain slashes.
func chatRef(ref string) bool {
	return strings.HasPrefix(strings.TrimSpace(ref), "chat:")
}

func resolveChatJournal(ref string) (string, error) {
	body := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ref), "chat:"))
	wsName, threadName, ok := strings.Cut(body, "/")
	wsName, threadName = strings.TrimSpace(wsName), strings.TrimSpace(threadName)
	if !ok || wsName == "" || threadName == "" {
		return "", fmt.Errorf("chat reference needs a workspace and a thread, like chat:keimling/Review")
	}
	ws, err := matchWorkspace(wsName)
	if err != nil {
		return "", err
	}
	th, err := matchThread(ws, threadName)
	if err != nil {
		return "", err
	}
	path := threadSessionPath(ws.ID, th.ID)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("chat %s/%s has no journal", ws.Name, th.Title)
	}
	return path, nil
}

func matchWorkspace(name string) (WorkspaceMeta, error) {
	all, err := allWorkspaceMetas()
	if err != nil {
		return WorkspaceMeta{}, err
	}
	var byID, byName []WorkspaceMeta
	for _, m := range all {
		if m.ID == name {
			byID = append(byID, m)
		}
		if strings.EqualFold(m.Name, name) {
			byName = append(byName, m)
		}
	}
	if len(byID) == 1 {
		return byID[0], nil
	}
	if len(byName) == 1 {
		return byName[0], nil
	}
	if len(byName) > 1 {
		return WorkspaceMeta{}, fmt.Errorf("workspace %q matches %s", name, workspaceChoices(byName))
	}
	return WorkspaceMeta{}, fmt.Errorf("unknown workspace %q; workspaces: %s", name, workspaceChoices(all))
}

func matchThread(ws WorkspaceMeta, name string) (ThreadMeta, error) {
	list, err := listThreads(ws.ID)
	if err != nil {
		return ThreadMeta{}, err
	}
	var byID, byTitle []ThreadMeta
	for _, th := range list {
		if th.ID == name {
			byID = append(byID, th)
		}
		if strings.EqualFold(th.Title, name) {
			byTitle = append(byTitle, th)
		}
	}
	if len(byID) == 1 {
		return byID[0], nil
	}
	if len(byTitle) == 1 {
		return byTitle[0], nil
	}
	if len(byTitle) > 1 {
		return ThreadMeta{}, fmt.Errorf("thread %q in %s matches %s", name, ws.Name, threadChoices(byTitle))
	}
	return ThreadMeta{}, fmt.Errorf("unknown thread %q in %s; threads: %s", name, ws.Name, threadChoices(list))
}

func allWorkspaceMetas() ([]WorkspaceMeta, error) {
	entries, err := os.ReadDir(workspacesRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no workspaces")
		}
		return nil, err
	}
	var out []WorkspaceMeta
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
	return out, nil
}

func workspaceChoices(list []WorkspaceMeta) string {
	if len(list) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(list))
	for _, m := range list {
		parts = append(parts, m.Name+" ("+m.ID+")")
	}
	return strings.Join(parts, ", ")
}

func threadChoices(list []ThreadMeta) string {
	if len(list) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(list))
	for _, th := range list {
		parts = append(parts, th.Title+" ("+th.ID+")")
	}
	return strings.Join(parts, ", ")
}

// chatRecordPath is a file inside a workspace thread directory, including the
// journal and the files beside it.
func chatRecordPath(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(workspacesRoot()), abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	return len(parts) >= 2 && parts[1] == "threads"
}

func showChatPath(body, real, label string) string {
	if label == "" || label == real {
		return body
	}
	prefix := "path: " + real
	if strings.HasPrefix(body, prefix) {
		return "path: " + label + strings.TrimPrefix(body, prefix)
	}
	return body
}
