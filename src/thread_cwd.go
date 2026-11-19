package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// threadCwd is one thread's working directory. Commands in that thread start
// here, and a directory change is written back for the next command.
type threadCwd struct {
	mu  sync.Mutex
	dir string
}

type (
	cwdSlotKey struct{}
	wsRootKey  struct{}
)

func (s *threadCwd) get() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dir
}

func (s *threadCwd) set(dir string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.dir = dir
	s.mu.Unlock()
}

func (a *Agent) bindDirs(ctx context.Context) context.Context {
	if a == nil {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	root := strings.TrimSpace(a.Cfg.WorkspaceRoot)
	ctx = context.WithValue(ctx, wsRootKey{}, root)
	ctx = context.WithValue(ctx, cwdSlotKey{}, a.cwdSlot())
	return ctx
}

// dirTable is the working directory of each thread in one workspace.
// A forked run shares the table, so a cd sticks to the thread.
type dirTable struct {
	mu   sync.Mutex
	dirs map[string]*threadCwd
}

func (a *Agent) dirTable() *dirTable {
	if a == nil {
		return nil
	}
	a.dirsMu.Lock()
	defer a.dirsMu.Unlock()
	if a.dirs == nil {
		a.dirs = &dirTable{}
	}
	return a.dirs
}

func (a *Agent) cwdSlot() *threadCwd {
	t := a.dirTable()
	if t == nil {
		return nil
	}
	key := strings.TrimSpace(a.Cfg.ThreadID)
	if key == "" {
		key = "main"
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.dirs == nil {
		t.dirs = map[string]*threadCwd{}
	}
	s := t.dirs[key]
	if s == nil {
		s = &threadCwd{dir: strings.TrimSpace(a.Cfg.WorkspaceRoot)}
		t.dirs[key] = s
	}
	return s
}

func (a *Agent) threadDir() string {
	s := a.cwdSlot()
	if s == nil {
		return ""
	}
	if dir := strings.TrimSpace(s.get()); dir != "" {
		return dir
	}
	return strings.TrimSpace(a.Cfg.WorkspaceRoot)
}

func (a *Agent) resetThreadDir(dir string) {
	if s := a.cwdSlot(); s != nil {
		s.set(strings.TrimSpace(dir))
	}
}

func cwdSlotFrom(ctx context.Context) *threadCwd {
	if ctx == nil {
		return nil
	}
	s, _ := ctx.Value(cwdSlotKey{}).(*threadCwd)
	return s
}

func threadDirFrom(ctx context.Context) string {
	if s := cwdSlotFrom(ctx); s != nil {
		return strings.TrimSpace(s.get())
	}
	return ""
}

func workspaceRootFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	s, _ := ctx.Value(wsRootKey{}).(string)
	return strings.TrimSpace(s)
}

func locatePath(root, cwd, path string) string {
	path = strings.TrimSpace(path)
	if strings.TrimSpace(root) == "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return path
		}
		return abs
	}
	abs := path
	if !filepath.IsAbs(path) {
		base := strings.TrimSpace(cwd)
		if base == "" {
			base = root
		}
		abs = filepath.Join(base, path)
	}
	out, err := filepath.Abs(abs)
	if err != nil {
		return abs
	}
	return out
}

func inWorkspace(root, cwd, path string) bool {
	abs := locatePath(root, cwd, path)
	if strings.TrimSpace(root) == "" {
		return insideWorkingDir(abs)
	}
	return pathInside(root, abs)
}

func pathInWorkspace(ctx context.Context, path string) bool {
	return inWorkspace(workspaceRootFrom(ctx), threadDirFrom(ctx), path)
}

func resolveToolPath(ctx context.Context, path string) string {
	if strings.TrimSpace(path) == "" || filepath.IsAbs(path) || chatRef(path) {
		return path
	}
	base := threadDirFrom(ctx)
	if base == "" {
		base = workspaceRootFrom(ctx)
	}
	if base == "" {
		return path
	}
	return filepath.Join(base, path)
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// wrapDirCommand runs command with dir as its starting directory and writes
// the directory it finished in to marker, including when the command exits.
func wrapDirCommand(dir, command, marker string) string {
	if runtime.GOOS == "windows" {
		return "Set-Location -LiteralPath " + psQuote(dir) + "; try { " + command + " } finally { Set-Content -LiteralPath " + psQuote(marker) + " -Value (Get-Location).Path }"
	}
	record := "pwd -P > " + shQuote(marker)
	return "trap " + shQuote(record) + " EXIT; cd " + shQuote(dir) + " || exit 126; " + command
}

func noteThreadDir(slot *threadCwd, marker string) {
	if slot == nil || strings.TrimSpace(marker) == "" {
		return
	}
	b, err := os.ReadFile(marker)
	_ = os.Remove(marker)
	if err != nil {
		return
	}
	dir := strings.TrimSpace(string(b))
	if dir == "" {
		return
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return
	}
	slot.set(abs)
}
