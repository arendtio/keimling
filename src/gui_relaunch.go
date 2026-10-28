package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

func isLoopbackURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// filterConnectArgs drops --connect / -connect and the following value so a
// relaunch starts the embedded local agent instead of re-attaching remotely.
func filterConnectArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--connect" || a == "-connect":
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
		case strings.HasPrefix(a, "--connect=") || strings.HasPrefix(a, "-connect="):
			// skip
		default:
			out = append(out, a)
		}
	}
	return out
}

func filterWorkspaceArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--workspace" || a == "-workspace":
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
		case strings.HasPrefix(a, "--workspace=") || strings.HasPrefix(a, "-workspace="):
			// skip
		default:
			out = append(out, a)
		}
	}
	return out
}

func relaunchLocalGUI() error {
	return relaunchLocalGUIWorkspace("")
}

func relaunchLocalGUIWorkspace(workspaceID string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := filterWorkspaceArgs(filterConnectArgs(os.Args[1:]))
	if id := strings.TrimSpace(workspaceID); id != "" {
		args = append([]string{"--workspace", id}, args...)
	}
	cmd := exec.Command(exe, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "keimling: relaunched local UI (pid", cmd.Process.Pid, ")")
	return nil
}

// scheduleRelaunchLocal clears remembered remote connect and starts a new local UI process.
func scheduleRelaunchLocal(configPath string, terminate func()) {
	_ = mergeGuiPrefsConnect(configPath, "", "")
	scheduleRelaunchWorkspace(configPath, "", terminate)
}

func scheduleRelaunchWorkspace(configPath, workspaceID string, terminate func()) {
	if id := strings.TrimSpace(workspaceID); id != "" {
		_ = mergeGuiPrefsWorkspace(configPath, id)
	}
	go func() {
		time.Sleep(150 * time.Millisecond)
		if err := relaunchLocalGUIWorkspace(workspaceID); err != nil {
			fmt.Fprintln(os.Stderr, "keimling: could not relaunch local UI:", err)
			return
		}
		if terminate != nil {
			terminate()
		}
	}()
}
