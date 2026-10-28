package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func handleMCPCommand(a *Agent, arg string) {
	arg = strings.TrimSpace(arg)
	if arg == "" || arg == "list" || arg == "status" {
		printMCPStatus(a)
		return
	}
	if arg == "reload" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		a.reloadMCP(ctx)
		fmt.Println("MCP reloaded.")
		printMCPStatus(a)
		return
	}
	if strings.HasPrefix(arg, "add ") {
		rest := strings.TrimSpace(strings.TrimPrefix(arg, "add "))
		name, cfg, err := parseMCPAddArgs(rest)
		if err != nil {
			fmt.Println("mcp add:", err)
			return
		}
		if err := UpsertUserMCPServer(name, cfg); err != nil {
			fmt.Println("mcp add:", err)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		a.reloadMCP(ctx)
		fmt.Printf("Added MCP server %q\n", name)
		printMCPStatus(a)
		return
	}
	if strings.HasPrefix(arg, "remove ") || strings.HasPrefix(arg, "rm ") {
		name := strings.TrimSpace(strings.TrimPrefix(arg, "remove "))
		if strings.HasPrefix(arg, "rm ") {
			name = strings.TrimSpace(strings.TrimPrefix(arg, "rm "))
		}
		if err := RemoveUserMCPServer(name); err != nil {
			fmt.Println("mcp remove:", err)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		a.reloadMCP(ctx)
		fmt.Printf("Removed MCP server %q from user mcp.json\n", name)
		return
	}
	fmt.Println("Usage: /mcp | /mcp reload | /mcp add NAME -- command args… | /mcp add NAME --url URL | /mcp remove NAME")
}

func parseMCPAddArgs(rest string) (string, MCPServerConfig, error) {
	fields := strings.Fields(rest)
	if len(fields) < 2 {
		return "", MCPServerConfig{}, fmt.Errorf("need NAME and -- command… or --url URL")
	}
	name := fields[0]
	rest = strings.TrimSpace(strings.TrimPrefix(rest, name))
	if strings.HasPrefix(rest, "--url ") {
		url := strings.TrimSpace(strings.TrimPrefix(rest, "--url "))
		if url == "" {
			return "", MCPServerConfig{}, fmt.Errorf("url required")
		}
		cfg := MCPServerConfig{URL: url}
		return name, cfg, cfg.validate()
	}
	if strings.HasPrefix(rest, "--sse ") {
		url := strings.TrimSpace(strings.TrimPrefix(rest, "--sse "))
		cfg := MCPServerConfig{URL: url, Type: "sse"}
		return name, cfg, cfg.validate()
	}
	if !strings.HasPrefix(rest, "-- ") {
		return "", MCPServerConfig{}, fmt.Errorf("use: /mcp add NAME -- command args…  or  /mcp add NAME --url URL")
	}
	cmdLine := strings.TrimSpace(strings.TrimPrefix(rest, "-- "))
	parts := strings.Fields(cmdLine)
	if len(parts) == 0 {
		return "", MCPServerConfig{}, fmt.Errorf("command required after --")
	}
	cfg := MCPServerConfig{Command: parts[0], Args: parts[1:]}
	return name, cfg, cfg.validate()
}

func printMCPStatus(a *Agent) {
	st := a.ensureMCP().Status()
	if len(st) == 0 {
		fmt.Println("No MCP servers configured. Add via ~/.config/keimling/mcp.json or /mcp add …")
		return
	}
	fmt.Println("\033[1mMCP servers\033[0m (backend clients)")
	for _, s := range st {
		line := fmt.Sprintf("  %s  [%s] %s", s.Name, s.Transport, s.Status)
		if s.Error != "" {
			line += " — " + s.Error
		}
		fmt.Println(line)
		if len(s.Tools) > 0 {
			fmt.Println("    tools:", strings.Join(s.Tools, ", "))
		}
	}
}
