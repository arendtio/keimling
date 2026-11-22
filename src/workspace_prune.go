package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
)

// workspacesInUse returns the workspace ids an agent instance still lists. A
// workspace in that set is never pruned: an instance that wakes later has to
// find its own directory.
func workspacesInUse() map[string]bool {
	used := map[string]bool{}
	all, err := listAgentInstances()
	if err != nil {
		return used
	}
	for _, m := range all {
		for _, ws := range m.Workspaces {
			if ws = strings.TrimSpace(ws); ws != "" {
				used[ws] = true
			}
		}
	}
	return used
}

// pruneMissingWorkspaces drops workspace entries whose folder is gone, that no
// agent instance still uses, and that are not in keep. The Keimling data of
// that workspace (threads, artifacts) goes with it: a workspace whose folder
// no longer exists has nothing left to show. Returns what was removed.
func pruneMissingWorkspaces(keep map[string]bool) []WorkspaceMeta {
	list, err := listWorkspaces()
	if err != nil {
		return nil
	}
	inUse := workspacesInUse()
	removed := make([]WorkspaceMeta, 0, 2)
	for _, ws := range list {
		if ws.Builtin || ws.ID == workspaceGlobalID {
			continue
		}
		if keep[ws.ID] {
			continue
		}
		if inUse[ws.ID] {
			continue
		}
		root := strings.TrimSpace(ws.Root)
		if root == "" {
			continue
		}
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			continue
		}
		if err := os.RemoveAll(workspaceDir(ws.ID)); err != nil {
			continue
		}
		removed = append(removed, ws)
	}
	sort.Slice(removed, func(i, j int) bool { return removed[i].Root < removed[j].Root })
	return removed
}

// handleWorkspacePrune drops the entries whose folder is gone. The workspace the
// host is focused on stays, even if its folder is gone: removing the place the
// user is standing in is not a cleanup.
func (h *Host) handleWorkspacePrune(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	keep := map[string]bool{}
	if id := strings.TrimSpace(h.focusWorkspaceID()); id != "" {
		keep[id] = true
	}
	removed := pruneMissingWorkspaces(keep)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"removed": removed, "count": len(removed)})
}
