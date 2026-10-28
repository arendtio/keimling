package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const keimlingRoleEnv = "KEIMLING_ROLE"

func isWorkerRole() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(keimlingRoleEnv)), "worker")
}

type hostWorker struct {
	Key    string // ws:<id> or agent:<id>
	Kind   string // workspace | agent
	Ref    string // bare workspace or instance id
	Root   string
	Addr   string // host:port
	Secret string
	Cmd    *exec.Cmd
}

func workspaceWorkerKey(id string) string { return "ws:" + strings.TrimSpace(id) }
func agentWorkerKey(id string) string     { return "agent:" + strings.TrimSpace(id) }

// Host supervises workspace worker processes and proxies the UI to the attached one.
type Host struct {
	mu             sync.Mutex
	secret         string
	config         string
	exe            string
	workers        map[string]*hostWorker
	attached       string
	focusWorkspace string // sidebar workspace; kept when an agent instance is attached
	client         *http.Client
}

func newHost(secret, configPath string) (*Host, error) {
	if strings.TrimSpace(secret) == "" {
		s, err := randomAccessSecret()
		if err != nil {
			return nil, err
		}
		secret = s
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return &Host{
		secret:  secret,
		config:  configPath,
		exe:     exe,
		workers: map[string]*hostWorker{},
		client:  &http.Client{Timeout: 5 * time.Second},
	}, nil
}

func (h *Host) Secret() string { return h.secret }

func (h *Host) Attached() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.attached
}

func reserveLoopbackAddr() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr, nil
}

func (h *Host) EnsureWorker(ctx context.Context, workspaceID string) (*hostWorker, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id required")
	}
	meta, err := readWorkspaceMeta(workspaceID)
	if err != nil {
		return nil, err
	}
	key := workspaceWorkerKey(workspaceID)

	h.mu.Lock()
	if w := h.workers[key]; w != nil && w.Cmd != nil && w.Cmd.Process != nil {
		alive := w.Cmd.ProcessState == nil || !w.Cmd.ProcessState.Exited()
		if alive {
			addr, secret := w.Addr, w.Secret
			h.mu.Unlock()
			if err := h.waitHealthy(ctx, addr, secret); err == nil {
				h.mu.Lock()
				w := h.workers[key]
				h.mu.Unlock()
				return w, nil
			}
			h.mu.Lock()
			h.dropWorkerLocked(key)
		}
	}
	h.mu.Unlock()

	addr, err := reserveLoopbackAddr()
	if err != nil {
		return nil, err
	}
	args := []string{
		"--server",
		"--listen", addr,
		"--workspace", workspaceID,
		"--access-secret", h.secret,
		"--check-model=false",
	}
	if strings.TrimSpace(h.config) != "" {
		args = append(args, "--config", h.config)
	}
	cmd := exec.Command(h.exe, args...)
	cmd.Env = append(os.Environ(), keimlingRoleEnv+"=worker")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = meta.Root
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	w := &hostWorker{Key: key, Kind: "workspace", Ref: workspaceID, Root: meta.Root, Addr: addr, Secret: h.secret, Cmd: cmd}
	h.mu.Lock()
	h.workers[key] = w
	h.mu.Unlock()

	if err := h.waitHealthy(ctx, addr, h.secret); err != nil {
		h.StopWorker(key)
		return nil, fmt.Errorf("workspace %s worker: %w", workspaceID, err)
	}
	go h.reap(key, cmd)
	return w, nil
}

func (h *Host) EnsureAgentWorker(ctx context.Context, instanceID string) (*hostWorker, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return nil, fmt.Errorf("agent instance id required")
	}
	meta, err := readAgentInstanceMeta(instanceID)
	if err != nil {
		return nil, err
	}
	ws, err := readWorkspaceMeta(meta.CurrentWorkspace)
	if err != nil {
		return nil, fmt.Errorf("agent workspace: %w", err)
	}
	key := agentWorkerKey(instanceID)

	h.mu.Lock()
	if w := h.workers[key]; w != nil && w.Cmd != nil && w.Cmd.Process != nil {
		alive := w.Cmd.ProcessState == nil || !w.Cmd.ProcessState.Exited()
		if alive {
			addr, secret := w.Addr, w.Secret
			h.mu.Unlock()
			if err := h.waitHealthy(ctx, addr, secret); err == nil {
				h.mu.Lock()
				w := h.workers[key]
				h.mu.Unlock()
				return w, nil
			}
			h.mu.Lock()
			h.dropWorkerLocked(key)
		}
	}
	h.mu.Unlock()

	addr, err := reserveLoopbackAddr()
	if err != nil {
		return nil, err
	}
	args := []string{
		"--server",
		"--listen", addr,
		"--agent-instance", instanceID,
		"--access-secret", h.secret,
		"--check-model=false",
	}
	if strings.TrimSpace(h.config) != "" {
		args = append(args, "--config", h.config)
	}
	cmd := exec.Command(h.exe, args...)
	cmd.Env = append(os.Environ(), keimlingRoleEnv+"=worker")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = ws.Root
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	w := &hostWorker{Key: key, Kind: "agent", Ref: instanceID, Root: ws.Root, Addr: addr, Secret: h.secret, Cmd: cmd}
	h.mu.Lock()
	h.workers[key] = w
	h.mu.Unlock()

	if err := h.waitHealthy(ctx, addr, h.secret); err != nil {
		h.StopWorker(key)
		return nil, fmt.Errorf("agent %s worker: %w", instanceID, err)
	}
	go h.reap(key, cmd)
	return w, nil
}

func (h *Host) reap(id string, cmd *exec.Cmd) {
	_ = cmd.Wait()
	h.mu.Lock()
	defer h.mu.Unlock()
	if w := h.workers[id]; w != nil && w.Cmd == cmd {
		delete(h.workers, id)
		if h.attached == id {
			h.attached = ""
		}
	}
}

func (h *Host) dropWorkerLocked(id string) {
	if w := h.workers[id]; w != nil && w.Cmd != nil && w.Cmd.Process != nil {
		_ = w.Cmd.Process.Kill()
	}
	delete(h.workers, id)
	if h.attached == id {
		h.attached = ""
	}
}

func (h *Host) StopWorker(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.dropWorkerLocked(id)
}

func (h *Host) StopAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id := range h.workers {
		h.dropWorkerLocked(id)
	}
}

func (h *Host) Attach(ctx context.Context, workspaceID string) error {
	w, err := h.EnsureWorker(ctx, workspaceID)
	if err != nil {
		return err
	}
	h.mu.Lock()
	h.attached = w.Key
	h.focusWorkspace = w.Ref
	h.mu.Unlock()
	_ = mergeGuiPrefsWorkspace(h.config, w.Ref)
	return nil
}

func (h *Host) AttachAgent(ctx context.Context, instanceID string) error {
	meta, err := readAgentInstanceMeta(instanceID)
	if err != nil {
		return err
	}
	w, err := h.EnsureAgentWorker(ctx, instanceID)
	if err != nil {
		return err
	}
	h.mu.Lock()
	h.attached = w.Key
	if ws := strings.TrimSpace(meta.CurrentWorkspace); ws != "" {
		h.focusWorkspace = ws
	}
	h.mu.Unlock()
	return nil
}

func (h *Host) focusWorkspaceID() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.focusWorkspace != "" {
		return h.focusWorkspace
	}
	if strings.HasPrefix(h.attached, "ws:") {
		return strings.TrimPrefix(h.attached, "ws:")
	}
	return ""
}

func (h *Host) workerBusy(ctx context.Context, w *hostWorker) bool {
	if w == nil {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+w.Addr+"/api/health", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+w.Secret)
	// Short client so sidebar polls stay snappy even if a worker is wedged.
	client := &http.Client{Timeout: 800 * time.Millisecond}
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	var body struct {
		State string `json:"state"`
	}
	_ = json.NewDecoder(io.LimitReader(res.Body, 1<<16)).Decode(&body)
	return body.State == "running"
}

func (h *Host) workersBusy(ctx context.Context, workers map[string]*hostWorker) map[string]bool {
	out := make(map[string]bool, len(workers))
	if len(workers) == 0 {
		return out
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for id, w := range workers {
		wg.Add(1)
		go func(id string, w *hostWorker) {
			defer wg.Done()
			busy := h.workerBusy(ctx, w)
			mu.Lock()
			out[id] = busy
			mu.Unlock()
		}(id, w)
	}
	wg.Wait()
	return out
}

// proxyToWorkspace routes a request to the focus workspace worker and attaches it
// (leaving any agent instance view). Used for human thread create/switch/delete.
func (h *Host) proxyToWorkspace() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		wsID := strings.TrimSpace(req.URL.Query().Get("workspace"))
		if wsID == "" {
			wsID = h.focusWorkspaceID()
		}
		if wsID == "" {
			http.Error(rw, "no workspace", http.StatusServiceUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
		defer cancel()
		if err := h.Attach(ctx, wsID); err != nil {
			http.Error(rw, err.Error(), http.StatusBadGateway)
			return
		}
		w, err := h.EnsureWorker(ctx, wsID)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadGateway)
			return
		}
		target, err := url.Parse("http://" + w.Addr)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.FlushInterval = 50 * time.Millisecond
		orig := proxy.Director
		secret := w.Secret
		proxy.Director = func(r *http.Request) {
			orig(r)
			r.Host = target.Host
			r.Header.Set("Authorization", "Bearer "+secret)
			r.Header.Del("X-Keimling-Access-Secret")
		}
		proxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, e error) {
			http.Error(rw, "worker unavailable: "+e.Error(), http.StatusBadGateway)
		}
		proxy.ServeHTTP(rw, req)
	})
}

func (h *Host) waitHealthy(ctx context.Context, addr, secret string) error {
	deadline := time.Now().Add(20 * time.Second)
	url := "http://" + addr + "/api/health"
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+secret)
		res, err := h.client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16))
			res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return fmt.Errorf("worker at %s did not become healthy", addr)
}

func (h *Host) attachedWorker() (*hostWorker, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.attached == "" {
		return nil, fmt.Errorf("no workspace attached")
	}
	w := h.workers[h.attached]
	if w == nil {
		return nil, fmt.Errorf("attached workspace worker missing")
	}
	return w, nil
}

func (h *Host) proxy() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		w, err := h.attachedWorker()
		if err != nil {
			http.Error(rw, err.Error(), http.StatusServiceUnavailable)
			return
		}
		target, err := url.Parse("http://" + w.Addr)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.FlushInterval = 50 * time.Millisecond
		orig := proxy.Director
		proxy.Director = func(r *http.Request) {
			orig(r)
			r.Host = target.Host
			r.Header.Set("Authorization", "Bearer "+w.Secret)
			r.Header.Del("X-Keimling-Access-Secret")
		}
		proxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, e error) {
			http.Error(rw, "worker unavailable: "+e.Error(), http.StatusBadGateway)
		}
		proxy.ServeHTTP(rw, req)
	})
}

func (h *Host) Handler() http.Handler {
	mux := http.NewServeMux()
	ui, err := fsSubWebUI()
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(ui))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	auth := func(next http.HandlerFunc) http.Handler {
		return h.requireSecret(next)
	}
	mux.Handle("/api/workspaces", auth(h.handleWorkspaces))
	mux.Handle("/api/workspaces/open", auth(h.handleWorkspaceOpen))
	mux.Handle("/api/workspaces/switch", auth(h.handleWorkspaceSwitch))
	mux.Handle("/api/agents/recipes", auth(h.handleAgentRecipes))
	mux.Handle("/api/agents/recipes/update", auth(h.handleAgentRecipeUpdate))
	mux.Handle("/api/agents/recipes/delete", auth(h.handleAgentRecipeDelete))
	mux.Handle("/api/agents/instances", auth(h.handleAgentInstances))
	mux.Handle("/api/agents/instances/attach", auth(h.handleAgentAttach))
	mux.Handle("/api/agents/instances/stop", auth(h.handleAgentStop))
	mux.Handle("/api/agents/instances/delete", auth(h.handleAgentDelete))
	mux.Handle("/api/agents/instances/message", auth(h.handleAgentInstanceMessage))
	mux.Handle("/api/agents/teams", auth(h.handleAgentTeams))
	mux.Handle("/api/agents/teams/spawn", auth(h.handleAgentTeamSpawn))
	mux.Handle("/api/agents/teams/delete", auth(h.handleAgentTeamDelete))
	// Workspace threads stay host-owned so attaching an agent does not replace the sidebar list.
	mux.Handle("/api/threads", auth(h.handleHostThreads))
	mux.Handle("/api/threads/switch", auth(h.proxyToWorkspace().ServeHTTP))
	mux.Handle("/api/threads/delete", auth(h.proxyToWorkspace().ServeHTTP))
	mux.Handle("/api/", auth(h.proxy().ServeHTTP))
	mux.Handle("/events", auth(h.proxy().ServeHTTP))
	mux.Handle("/files/", h.proxy()) // capability URLs; worker validates token

	return mux
}

func fsSubWebUI() (fs.FS, error) {
	return fs.Sub(webuiFS, "webui")
}

func (h *Host) requireSecret(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if got == "" {
			got = strings.TrimSpace(r.Header.Get("X-Keimling-Access-Secret"))
		}
		if got == "" {
			got = strings.TrimSpace(r.URL.Query().Get("access_secret"))
		}
		if subtle.ConstantTimeCompare([]byte(got), []byte(h.secret)) == 1 {
			next(w, r)
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func (h *Host) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := listWorkspaces()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cur := h.focusWorkspaceID()
	h.mu.Lock()
	running := map[string]bool{}
	for key := range h.workers {
		if strings.HasPrefix(key, "ws:") {
			running[strings.TrimPrefix(key, "ws:")] = true
		}
	}
	h.mu.Unlock()
	out := make([]map[string]any, 0, len(list))
	for _, m := range list {
		out = append(out, map[string]any{
			"id":         m.ID,
			"name":       m.Name,
			"root":       m.Root,
			"builtin":    m.Builtin,
			"running":    running[m.ID],
			"updated_at": m.UpdatedAt,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"current": cur, "workspaces": out})
}

func (h *Host) handleHostThreads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		wsID := strings.TrimSpace(r.URL.Query().Get("workspace"))
		if wsID == "" {
			wsID = h.focusWorkspaceID()
		}
		if wsID == "" {
			http.Error(w, "no workspace", http.StatusServiceUnavailable)
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		var (
			list []ThreadMeta
			err  error
		)
		if q != "" {
			list, err = searchThreads(wsID, q)
		} else {
			list, err = listThreads(wsID)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		current := ""
		h.mu.Lock()
		attachedAgent := strings.HasPrefix(h.attached, "agent:")
		h.mu.Unlock()
		if !attachedAgent {
			if active, e := readActiveThreadID(wsID); e == nil {
				current = active
			}
		}
		for i := range list {
			list[i].Active = current != "" && list[i].ID == current
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspace_id": wsID,
			"current":      current,
			"state":        "idle",
			"query":        q,
			"threads":      list,
		})
	case http.MethodPost:
		// Create thread on the workspace worker (not an attached agent).
		h.proxyToWorkspace().ServeHTTP(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Host) handleWorkspaceOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	ws, err := ensureWorkspaceForRoot(body.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.respondAttach(w, ws.ID)
}

func (h *Host) handleWorkspaceSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(body.ID)
	if _, err := readWorkspaceMeta(id); err != nil {
		http.Error(w, "unknown workspace", http.StatusNotFound)
		return
	}
	h.respondAttach(w, id)
}

func (h *Host) respondAttach(w http.ResponseWriter, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := h.Attach(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"id":       id,
		"relaunch": false,
		"attached": true,
	})
}
