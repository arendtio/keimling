package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// watchHub keeps the processes and file watches for one agent instance.
type watchHub struct {
	mu       sync.Mutex
	instance string
	busy     map[string]bool
	cmds     map[string]*exec.Cmd
	files    map[string]*fsnotify.Watcher
}

var watchHubs sync.Map

func watchHubFor(instanceID string) *watchHub {
	v, _ := watchHubs.LoadOrStore(strings.TrimSpace(instanceID), &watchHub{})
	return v.(*watchHub)
}

func (h *watchHub) tryStart(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.busy == nil {
		h.busy = map[string]bool{}
	}
	if h.busy[id] {
		return false
	}
	h.busy[id] = true
	return true
}

func (h *watchHub) finish(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.busy != nil {
		delete(h.busy, id)
	}
}

func (h *watchHub) stop(id string) {
	h.mu.Lock()
	cmd := h.cmds[id]
	watcher := h.files[id]
	inst := h.instance
	delete(h.cmds, id)
	delete(h.files, id)
	h.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		killProcessGroup(cmd.Process.Pid)
		forgetWatchPID(inst, id)
	}
	if watcher != nil {
		_ = watcher.Close()
	}
}

func (h *watchHub) stopAll() {
	h.mu.Lock()
	ids := make([]string, 0, len(h.cmds)+len(h.files))
	seen := map[string]bool{}
	for id := range h.cmds {
		seen[id] = true
		ids = append(ids, id)
	}
	for id := range h.files {
		if !seen[id] {
			ids = append(ids, id)
		}
	}
	h.mu.Unlock()
	for _, id := range ids {
		h.stop(id)
	}
}

func stopInstanceWatches(instanceID string) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return
	}
	if v, ok := watchHubs.LoadAndDelete(instanceID); ok {
		v.(*watchHub).stopAll()
	}
	killStoredWatchPIDs(instanceID)
}

var watchPIDMu sync.Mutex

func watchPIDPath(instanceID string) string {
	return filepath.Join(agentInstanceDir(instanceID), "watch-pids.json")
}

func readWatchPIDs(instanceID string) map[string]int {
	b, err := os.ReadFile(watchPIDPath(instanceID))
	if err != nil {
		return map[string]int{}
	}
	var m map[string]int
	if json.Unmarshal(b, &m) != nil || m == nil {
		return map[string]int{}
	}
	return m
}

func writeWatchPIDs(instanceID string, m map[string]int) {
	path := watchPIDPath(instanceID)
	if len(m) == 0 {
		_ = os.Remove(path)
		return
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o600) != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

func rememberWatchPID(instanceID, watchID string, pid int) {
	instanceID = strings.TrimSpace(instanceID)
	watchID = strings.TrimSpace(watchID)
	if instanceID == "" || watchID == "" || pid <= 0 {
		return
	}
	watchPIDMu.Lock()
	defer watchPIDMu.Unlock()
	m := readWatchPIDs(instanceID)
	m[watchID] = pid
	writeWatchPIDs(instanceID, m)
}

func forgetWatchPID(instanceID, watchID string) {
	instanceID = strings.TrimSpace(instanceID)
	watchID = strings.TrimSpace(watchID)
	if instanceID == "" || watchID == "" {
		return
	}
	watchPIDMu.Lock()
	defer watchPIDMu.Unlock()
	m := readWatchPIDs(instanceID)
	if _, ok := m[watchID]; !ok {
		return
	}
	delete(m, watchID)
	writeWatchPIDs(instanceID, m)
}

func killStoredWatchPIDs(instanceID string) {
	watchPIDMu.Lock()
	m := readWatchPIDs(instanceID)
	watchPIDMu.Unlock()
	for _, pid := range m {
		if pid > 0 {
			killProcessGroup(pid)
		}
	}
}

func (a *Agent) ensureLiveWatches() {
	p, ok := a.watchPlace()
	if !ok {
		return
	}
	a.ensureLiveWatchesAt(p)
}

func (a *Agent) ensureLiveWatchesAt(p watchPlace) {
	if a == nil || p.path == "" {
		return
	}
	list, err := readWatches(p.path)
	if err != nil {
		return
	}
	hub := watchHubFor(p.key)
	live := map[string]bool{}
	for _, w := range list {
		if w.Exited != "" {
			continue
		}
		switch w.Kind {
		case "file":
			live[w.ID] = true
			hub.ensureFile(a, p, w)
		case "command":
			live[w.ID] = true
			hub.ensureCommand(a, p, w)
		}
	}
	hub.stopMissing(live)
}

func (h *watchHub) stopMissing(live map[string]bool) {
	h.mu.Lock()
	var gone []string
	for id := range h.cmds {
		if !live[id] {
			gone = append(gone, id)
		}
	}
	for id := range h.files {
		if !live[id] {
			gone = append(gone, id)
		}
	}
	h.mu.Unlock()
	for _, id := range gone {
		h.stop(id)
	}
}

func (h *watchHub) ensureFile(a *Agent, p watchPlace, w AgentWatch) {
	h.mu.Lock()
	if h.files == nil {
		h.files = map[string]*fsnotify.Watcher{}
	}
	if _, ok := h.files[w.ID]; ok {
		h.mu.Unlock()
		return
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		h.mu.Unlock()
		return
	}
	if err := watcher.Add(w.Path); err != nil {
		_ = watcher.Close()
		h.mu.Unlock()
		deliverWatchResult(a, p, w.ID, err.Error())
		return
	}
	h.files[w.ID] = watcher
	h.mu.Unlock()
	go func() {
		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				h.onSource(a, p, w.ID, ev.Op.String()+"\t"+ev.Name)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				if err != nil {
					h.onSource(a, p, w.ID, err.Error())
				}
			}
		}
	}()
}

func (h *watchHub) ensureCommand(a *Agent, p watchPlace, w AgentWatch) {
	h.mu.Lock()
	if h.cmds == nil {
		h.cmds = map[string]*exec.Cmd{}
	}
	if _, ok := h.cmds[w.ID]; ok {
		h.mu.Unlock()
		return
	}
	cmd := shellCommand(selectedShell(), w.Command)
	setProcessGroup(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		h.mu.Unlock()
		return
	}
	if err := cmd.Start(); err != nil {
		h.mu.Unlock()
		_ = saveWatchProgress(p.path, w.ID, "", false, err.Error())
		if !p.held() {
			p.enqueue(w.Message, "command ended: "+err.Error())
		}
		return
	}
	h.cmds[w.ID] = cmd
	if p.instance != "" {
		h.instance = p.instance
	}
	rememberWatchPID(p.instance, w.ID, cmd.Process.Pid)
	h.mu.Unlock()
	go func() {
		readWatchLines(h, a, p, w.ID, stdout)
		_ = cmd.Wait()
		h.mu.Lock()
		still := h.cmds[w.ID] == cmd
		if still {
			delete(h.cmds, w.ID)
		}
		h.mu.Unlock()
		if !still {
			return
		}
		forgetWatchPID(p.instance, w.ID)
		note := "command ended"
		_ = saveWatchProgress(p.path, w.ID, "", false, note)
		if !p.held() {
			p.enqueue(w.Message, note)
		}
	}()
}

func readWatchLines(h *watchHub, a *Agent, p watchPlace, id string, r io.Reader) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		h.onSource(a, p, id, sc.Text())
	}
}

func (h *watchHub) onSource(a *Agent, p watchPlace, id, text string) {
	if !h.tryStart(id) {
		return
	}
	defer h.finish(id)
	list, err := readWatches(p.path)
	if err != nil {
		return
	}
	var w AgentWatch
	found := false
	for _, item := range list {
		if item.ID == id {
			w = item
			found = true
			break
		}
	}
	if !found || w.Exited != "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, runErr := a.runWatchChain(ctx, w.Steps, text)
	if runErr != nil {
		deliverWatchResult(a, p, id, runErr.Error())
		return
	}
	if len(w.Steps) == 0 {
		result = text
	}
	deliverWatchResult(a, p, id, result)
}
