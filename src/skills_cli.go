package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

func handleSkillsCommand(a *Agent, arg string) {
	arg = strings.TrimSpace(arg)
	if arg == "" || arg == "list" {
		a.refreshSkills()
		list := a.ensureSkills().List()
		if len(list) == 0 {
			fmt.Println("No skills installed. Use /skills add <path> or /skills search <query>.")
			return
		}
		fmt.Println("\033[1mSkills\033[0m")
		for _, sk := range list {
			fmt.Printf("  %s  [%s]\n    %s\n", sk.Name, sk.Source, truncate(sk.Description, 160))
		}
		return
	}
	if strings.HasPrefix(arg, "add ") {
		path := strings.TrimSpace(strings.TrimPrefix(arg, "add "))
		scope := SkillScopeUser
		if strings.HasPrefix(path, "--project ") {
			scope = SkillScopeProject
			path = strings.TrimSpace(strings.TrimPrefix(path, "--project "))
		}
		cwd, _ := os.Getwd()
		sk, err := InstallSkillFromPath(path, scope, cwd)
		if err != nil {
			fmt.Println("skills add:", err)
			return
		}
		a.refreshSkills()
		fmt.Printf("Installed skill %q → %s\n", sk.Name, sk.Root)
		return
	}
	if strings.HasPrefix(arg, "search ") {
		q := strings.TrimSpace(strings.TrimPrefix(arg, "search "))
		client := a.Cfg.ExternalClient
		if client == nil {
			fmt.Println("skills search: web search unavailable (configure OpenAI external key)")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		fmt.Println("⌕ Searching for skills…")
		cands, _, err := SearchSkillsOnline(ctx, client, q)
		if err != nil {
			fmt.Println("skills search:", err)
			return
		}
		lastSkillSearch = cands
		if len(cands) == 0 {
			fmt.Println("No installable skill URLs found in search results.")
			return
		}
		fmt.Println("\033[1mCandidates\033[0m ( /skills install <n> )")
		for _, c := range cands {
			fmt.Printf("  %s  %s\n    %s\n    %s\n", c.ID, c.Title, truncate(c.Description, 120), c.SourceURL)
		}
		return
	}
	if strings.HasPrefix(arg, "install ") {
		target := strings.TrimSpace(strings.TrimPrefix(arg, "install "))
		scope := SkillScopeUser
		if strings.HasPrefix(target, "--project ") {
			scope = SkillScopeProject
			target = strings.TrimSpace(strings.TrimPrefix(target, "--project "))
		}
		cwd, _ := os.Getwd()
		var sk *SkillMeta
		var err error
		if c := skillCandidateByID(target); c != nil {
			sk, err = InstallSkillFromURL(c.SourceURL, scope, cwd)
		} else {
			sk, err = InstallSkillFromURL(target, scope, cwd)
		}
		if err != nil {
			fmt.Println("skills install:", err)
			return
		}
		a.refreshSkills()
		fmt.Printf("Installed skill %q → %s\n", sk.Name, sk.Root)
		return
	}
	fmt.Println("Usage: /skills | /skills add [--project] PATH | /skills search QUERY | /skills install [--project] N|URL")
}
