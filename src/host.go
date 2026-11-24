package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"time"
)

const keimlingRoleEnv = "KEIMLING_ROLE"

func isWorkerRole() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(keimlingRoleEnv)), "worker")
}

type hostWorker struct {
	Key     string // ws:<id> or agent:<id>
	Kind    string // workspace | agent
	Ref     string // bare workspace or instance id
	Root    string
	Secret  string
	Handler http.Handler
	Agent   *Agent
	cancel  context.CancelFunc
}

func (h *Host) publishLoopback() {
	if base := strings.TrimSpace(h.baseURL); base != "" {
		_ = os.Setenv("KEIMLING_HOST", base)
		_ = os.Setenv("KEIMLING_SECRET", h.secret)
	}
}

func hostBaseForWorkers(listen string) string {
	listen = strings.TrimSpace(listen)
	if listen == "" {
		return ""
	}
	if strings.HasPrefix(listen, ":") {
		return "http://127.0.0.1" + listen
	}
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "http://" + listen
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func workspaceWorkerKey(id string) string { return "ws:" + strings.TrimSpace(id) }
func agentWorkerKey(id string) string     { return "agent:" + strings.TrimSpace(id) }

// Host runs every workspace and every agent instance in this process and serves the UI.
type Host struct {
	mu             sync.Mutex
	secret         string
	config         string
	workers        map[string]*hostWorker
	waking         map[string]bool
	attached       string
	focusWorkspace string // sidebar workspace; kept when an agent instance is attached
	baseURL        string // http://127.0.0.1:port this process serves
}

func newHost(secret, configPath string) (*Host, error) {
	if strings.TrimSpace(secret) == "" {
		s, err := randomAccessSecret()
		if err != nil {
			return nil, err
		}
		secret = s
	}
	return &Host{
		secret:  secret,
		config:  configPath,
		workers: map[string]*hostWorker{},
	}, nil
}

func (h *Host) Secret() string { return h.secret }

func (h *Host) Attached() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.attached
}

func (h *Host) EnsureWorker(ctx context.Context, workspaceID string) (*hostWorker, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id required")
	}
	if _, err := readWorkspaceMeta(workspaceID); err != nil {
		return nil, err
	}
	return h.ensure(workspaceWorkerKey(workspaceID), func() (*hostWorker, error) {
		return h.startWorkspace(workspaceID)
	})
}

func (h *Host) liveAgentWorker(instanceID string) (*hostWorker, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	w := h.workers[agentWorkerKey(instanceID)]
	if w == nil || w.Handler == nil {
		return nil, false
	}
	return w, true
}

func (h *Host) EnsureAgentWorker(ctx context.Context, instanceID string) (*hostWorker, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return nil, fmt.Errorf("agent instance id required")
	}
	if _, err := readAgentInstanceMeta(instanceID); err != nil {
		return nil, err
	}
	return h.ensure(agentWorkerKey(instanceID), func() (*hostWorker, error) {
		return h.startAgent(instanceID)
	})
}

func (h *Host) ensure(key string, start func() (*hostWorker, error)) (*hostWorker, error) {
	h.mu.Lock()
	if w := h.workers[key]; w != nil && w.Handler != nil {
		h.mu.Unlock()
		return w, nil
	}
	h.mu.Unlock()
	w, err := start()
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	if old := h.workers[key]; old != nil && old.Handler != nil {
		h.mu.Unlock()
		closeWorker(w)
		return old, nil
	}
	h.workers[key] = w
	h.mu.Unlock()
	return w, nil
}

func (h *Host) startWorkspace(workspaceID string) (*hostWorker, error) {
	meta, err := readWorkspaceMeta(workspaceID)
	if err != nil {
		return nil, err
	}
	agent, cancel, err := openRuntimeAgent(h.config, workspaceID, "")
	if err != nil {
		return nil, err
	}
	srv := newAppServer(appServerOpts{AccessSecret: h.secret, Agent: agent})
	return &hostWorker{
		Key: workspaceWorkerKey(workspaceID), Kind: "workspace", Ref: workspaceID, Root: meta.Root,
		Secret: h.secret, Handler: srv.Handler(), Agent: agent, cancel: cancel,
	}, nil
}

func (h *Host) startAgent(instanceID string) (*hostWorker, error) {
	meta, err := readAgentInstanceMeta(instanceID)
	if err != nil {
		return nil, err
	}
	ws, err := readWorkspaceMeta(meta.CurrentWorkspace)
	if err != nil {
		return nil, err
	}
	agent, cancel, err := openRuntimeAgent(h.config, "", instanceID)
	if err != nil {
		return nil, err
	}
	srv := newAppServer(appServerOpts{AccessSecret: h.secret, Agent: agent})
	return &hostWorker{
		Key: agentWorkerKey(instanceID), Kind: "agent", Ref: instanceID, Root: ws.Root,
		Secret: h.secret, Handler: srv.Handler(), Agent: agent, cancel: cancel,
	}, nil
}

func closeWorker(w *hostWorker) {
	if w == nil {
		return
	}
	if w.cancel != nil {
		w.cancel()
	}
	if w.Agent != nil {
		if w.Agent.MCP != nil {
			w.Agent.MCP.Close()
		}
		if w.Agent.Journal != nil {
			_ = w.Agent.Journal.Close()
		}
	}
}

func (h *Host) serveWorker(w *hostWorker, rw http.ResponseWriter, req *http.Request) {
	if w == nil || w.Handler == nil {
		http.Error(rw, "workspace is not running", http.StatusServiceUnavailable)
		return
	}
	req.Header.Set("Authorization", "Bearer "+w.Secret)
	w.Handler.ServeHTTP(rw, req)
}

func (h *Host) callWorker(ctx context.Context, w *hostWorker, method, path string, body string) (*http.Response, error) {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, path, r)
	if err != nil {
		return nil, err
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.serveWorker(w, rec, req)
	return rec.Result(), nil
}

func (h *Host) dropWorkerLocked(id string) *hostWorker {
	w := h.workers[id]
	delete(h.workers, id)
	if h.attached == id {
		h.attached = ""
	}
	return w
}

func (h *Host) StopWorker(id string) {
	h.mu.Lock()
	w := h.dropWorkerLocked(id)
	h.mu.Unlock()
	closeWorker(w)
}

func (h *Host) StopAll() {
	h.mu.Lock()
	all := h.workers
	h.workers = map[string]*hostWorker{}
	h.attached = ""
	h.mu.Unlock()
	for _, w := range all {
		closeWorker(w)
	}
}

var errHangUpFirst = errors.New("hang up the call first")

func (h *Host) workerCallOpen(ctx context.Context, w *hostWorker) bool {
	if w == nil {
		return false
	}
	res, err := h.callWorker(ctx, w, http.MethodGet, "/api/health", "")
	if err != nil {
		return false
	}
	defer res.Body.Close()
	var body struct {
		Call bool `json:"call"`
	}
	_ = json.NewDecoder(io.LimitReader(res.Body, 1<<16)).Decode(&body)
	return body.Call
}

// refuseCallLeave keeps a live call on the worker that owns it.
func (h *Host) refuseCallLeave(ctx context.Context, nextKey string) error {
	h.mu.Lock()
	cur := h.attached
	w := h.workers[cur]
	h.mu.Unlock()
	if cur == "" || cur == nextKey || w == nil {
		return nil
	}
	if h.workerCallOpen(ctx, w) {
		return errHangUpFirst
	}
	return nil
}

func writeAttachErr(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	code := http.StatusBadGateway
	if errors.Is(err, errHangUpFirst) {
		code = http.StatusConflict
	}
	http.Error(w, err.Error(), code)
}

func (h *Host) Attach(ctx context.Context, workspaceID string) error {
	if err := h.refuseCallLeave(ctx, workspaceWorkerKey(workspaceID)); err != nil {
		return err
	}
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
	if err := h.refuseCallLeave(ctx, agentWorkerKey(instanceID)); err != nil {
		return err
	}
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

type workerPulse struct {
	Busy bool
	Call bool
}

func (h *Host) workerPulse(ctx context.Context, w *hostWorker) workerPulse {
	if w == nil {
		return workerPulse{}
	}
	res, err := h.callWorker(ctx, w, http.MethodGet, "/api/health", "")
	if err != nil {
		return workerPulse{}
	}
	defer res.Body.Close()
	var body struct {
		State string `json:"state"`
		Call  bool   `json:"call"`
	}
	_ = json.NewDecoder(io.LimitReader(res.Body, 1<<16)).Decode(&body)
	return workerPulse{Busy: body.State == "running", Call: body.Call}
}

func (h *Host) workersBusy(ctx context.Context, workers map[string]*hostWorker) map[string]bool {
	pulse := h.workersPulse(ctx, workers)
	out := make(map[string]bool, len(pulse))
	for id, p := range pulse {
		out[id] = p.Busy
	}
	return out
}

func (h *Host) workersPulse(ctx context.Context, workers map[string]*hostWorker) map[string]workerPulse {
	out := make(map[string]workerPulse, len(workers))
	if len(workers) == 0 {
		return out
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for id, w := range workers {
		wg.Add(1)
		go func(id string, w *hostWorker) {
			defer wg.Done()
			p := h.workerPulse(ctx, w)
			mu.Lock()
			out[id] = p
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
			writeAttachErr(rw, err)
			return
		}
		w, err := h.EnsureWorker(ctx, wsID)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadGateway)
			return
		}
		h.serveWorker(w, rw, req)
	})
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
		h.serveWorker(w, rw, req)
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
		if r.URL.Path == "/" {
			writeWebUIPage(w, "login.html")
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	auth := func(next http.HandlerFunc) http.Handler {
		return h.requireSecret(next)
	}
	mux.Handle("/ui", auth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeWebUIPage(w, "index.html")
	}))
	mux.Handle("/api/workspaces", auth(h.handleWorkspaces))
	mux.Handle("/api/workspaces/open", auth(h.handleWorkspaceOpen))
	mux.Handle("/api/workspaces/switch", auth(h.handleWorkspaceSwitch))
	mux.Handle("/api/workspaces/hide", auth(h.handleWorkspaceHide))
	mux.Handle("/api/workspaces/rename", auth(h.handleWorkspaceRename))
	mux.Handle("/api/workspaces/restore", auth(h.handleWorkspaceRestore))
	mux.Handle("/api/workspaces/prune", auth(h.handleWorkspacePrune))
	mux.Handle("/api/agents/recipes", auth(h.handleAgentRecipes))
	mux.Handle("/api/agents/recipes/update", auth(h.handleAgentRecipeUpdate))
	mux.Handle("/api/agents/recipes/delete", auth(h.handleAgentRecipeDelete))
	mux.Handle("/api/agents/instances", auth(h.handleAgentInstances))
	mux.Handle("/api/agents/instances/attach", auth(h.handleAgentAttach))
	mux.Handle("/api/agents/instances/stop", auth(h.handleAgentStop))
	mux.Handle("/api/agents/instances/reset", auth(h.handleAgentReset))
	mux.Handle("/api/agents/instances/delete", auth(h.handleAgentDelete))
	mux.Handle("/api/agents/instances/message", auth(h.handleAgentInstanceMessage))
	mux.Handle("/api/agents/instances/access", auth(h.handleAgentInstanceAccess))
	mux.Handle("/api/agents/teams", auth(h.handleAgentTeams))
	mux.Handle("/api/agents/teams/spawn", auth(h.handleAgentTeamSpawn))
	mux.Handle("/api/agents/teams/delete", auth(h.handleAgentTeamDelete))
	mux.Handle("/api/escalations", auth(h.handleEscalations))
	mux.Handle("/api/escalations/done", auth(h.handleEscalationDone))
	// Workspace threads stay host-owned so attaching an agent does not replace the sidebar list.
	mux.Handle("/api/threads", auth(h.handleHostThreads))
	mux.Handle("/api/threads/switch", auth(h.proxyToWorkspace().ServeHTTP))
	mux.Handle("/api/threads/view", auth(h.handleThreadView))
	mux.Handle("/api/threads/deliver", auth(h.handleThreadDeliver))
	mux.Handle("/api/threads/delete", auth(h.proxyToWorkspace().ServeHTTP))
	mux.Handle("/api/", auth(h.proxy().ServeHTTP))
	mux.Handle("/events", auth(h.proxy().ServeHTTP))
	mux.Handle("/files/", h.proxy()) // capability URLs; worker validates token
	mux.Handle(mcpOAuthCallbackPath, h.proxy())
	mux.Handle(mcpOAuthMetaPath, h.proxy())

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

// handleThreadView reads a workspace thread from disk. It does not switch the
// worker, so a running thread keeps its journal.
func (h *Host) handleThreadView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	wsID := strings.TrimSpace(r.URL.Query().Get("workspace"))
	if wsID == "" {
		wsID = h.focusWorkspaceID()
	}
	if id == "" || wsID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	path := threadSessionPath(wsID, id)
	if st, err := os.Stat(threadDir(wsID, id)); err != nil || !st.IsDir() {
		http.Error(w, "unknown thread", http.StatusNotFound)
		return
	}
	events, lines, err := visibleJournal(path)
	if err != nil && !os.IsNotExist(err) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(events) == 0 {
		if fallback, ferr := originalEvents(path); ferr == nil {
			events = fallback
			lines = nil
		}
	}
	msgs := sessionMessages(events, 120)
	annotateBranchMessages(path, msgs, lines)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"thread_id":    id,
		"workspace_id": wsID,
		"preview":      true,
		"messages":     msgs,
		"chapters":     sessionChaptersFor(path),
	})
}

func (h *Host) handleThreadDeliver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		WorkspaceID string `json:"workspace_id"`
		ThreadID    string `json:"thread_id"`
		FromName    string `json:"from_name"`
		FromID      string `json:"from_id"`
		Text        string `json:"text"`
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil || json.Unmarshal(raw, &body) != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	wsID := strings.TrimSpace(body.WorkspaceID)
	if wsID == "" {
		http.Error(w, "workspace_id required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	worker, err := h.EnsureWorker(ctx, wsID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	res, err := h.callWorker(ctx, worker, http.MethodPost, "/api/threads/deliver", string(raw))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, res.Body)
}

func (h *Host) handleWorkspaceOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	ws, err := ensureWorkspaceNamed(body.Path, body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.respondAttach(w, ws.ID)
}

func (h *Host) handleWorkspaceRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	ws, err := renameWorkspace(body.ID, body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ws)
}

func (h *Host) handleWorkspaceRestore(w http.ResponseWriter, r *http.Request) {
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
	if err := showWorkspace(body.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": strings.TrimSpace(body.ID)})
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

func (h *Host) handleWorkspaceHide(w http.ResponseWriter, r *http.Request) {
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
	if id == h.focusWorkspaceID() {
		http.Error(w, "switch to another workspace before removing this one from the menu", http.StatusBadRequest)
		return
	}
	if err := hideWorkspace(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": id})
}

func (h *Host) respondAttach(w http.ResponseWriter, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := h.Attach(ctx, id); err != nil {
		writeAttachErr(w, err)
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
