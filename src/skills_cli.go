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
			fmt.Println("No skills installed. Use /skills search <query> then /skills preview <n>.")
			return
		}
		fmt.Println("\033[1mSkills\033[0m")
		for _, sk := range list {
			ro := ""
			if !sk.Writable {
				ro = " read-only"
			}
			origin := ""
			if sk.OriginURL != "" {
				origin = "\n    origin: " + sk.OriginURL
			}
			fmt.Printf("  %s  [%s]%s\n    %s%s\n", sk.Name, sk.Source, ro, truncate(sk.Description, 160), origin)
		}
		return
	}
	if strings.HasPrefix(arg, "remove ") || strings.HasPrefix(arg, "rm ") {
		name := strings.TrimSpace(strings.TrimPrefix(arg, "remove "))
		if strings.HasPrefix(arg, "rm ") {
			name = strings.TrimSpace(strings.TrimPrefix(arg, "rm "))
		}
		cwd, _ := os.Getwd()
		if err := DeleteSkill(name, cwd); err != nil {
			fmt.Println("skills remove:", err)
			return
		}
		a.refreshSkills()
		fmt.Printf("Removed skill %q\n", name)
		return
	}
	if strings.HasPrefix(arg, "check ") {
		name := strings.TrimSpace(strings.TrimPrefix(arg, "check "))
		cwd, _ := os.Getwd()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		st, err := CheckSkillUpdate(ctx, name, cwd)
		if err != nil {
			fmt.Println("skills check:", err)
			return
		}
		fmt.Printf("%s: %s\n", st.Name, st.Status)
		if st.Message != "" {
			fmt.Println(" ", st.Message)
		}
		if st.SourceURL != "" {
			fmt.Println("  source:", st.SourceURL)
		}
		return
	}
	if strings.HasPrefix(arg, "update ") {
		name := strings.TrimSpace(strings.TrimPrefix(arg, "update "))
		cwd, _ := os.Getwd()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		sk, st, err := UpdateSkill(ctx, name, cwd)
		if err != nil {
			fmt.Println("skills update:", err)
			return
		}
		a.refreshSkills()
		if st.Status == "up_to_date" {
			fmt.Printf("%s is already up to date\n", name)
			return
		}
		fmt.Printf("Updated skill %q → %s (%s)\n", sk.Name, sk.Root, st.Status)
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
		fmt.Println("\033[1mCandidates\033[0m ( /skills preview <n> )")
		for _, c := range cands {
			fmt.Printf("  %s  %s\n    %s\n    %s\n", c.ID, c.Title, truncate(c.Description, 120), c.SourceURL)
		}
		return
	}
	if strings.HasPrefix(arg, "preview ") {
		target := strings.TrimSpace(strings.TrimPrefix(arg, "preview "))
		sourceURL := target
		if c := skillCandidateByID(target); c != nil {
			sourceURL = c.SourceURL
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		fmt.Println("↓ Fetching skill bundle…")
		hub := ensureCLISkillPreviews()
		prev, err := StageSkillPreview(ctx, hub, sourceURL)
		if err != nil {
			fmt.Println("skills preview:", err)
			return
		}
		lastSkillPreviewID = prev.ID
		fmt.Printf("Preview %s — %s (%d files)\n", prev.ID, prev.Skill.Name, len(prev.Files))
		fmt.Printf("  %s\n  %s\n", prev.Skill.Description, prev.SourceURL)
		for _, f := range prev.Files {
			fmt.Printf("  · %s (%d)\n", f.Path, f.Size)
		}
		fmt.Println("Next: /skills review   or   /skills install --preview")
		return
	}
	if arg == "review" || strings.HasPrefix(arg, "review ") {
		id := strings.TrimSpace(strings.TrimPrefix(arg, "review"))
		if id == "" {
			id = lastSkillPreviewID
		}
		hub := ensureCLISkillPreviews()
		prev := hub.Get(id)
		if prev == nil {
			fmt.Println("skills review: no preview (run /skills preview <n> first)")
			return
		}
		client := a.Client
		if client == nil {
			fmt.Println("skills review: model client unavailable")
			return
		}
		model := strings.TrimSpace(a.Cfg.ReviewModel)
		if model == "" {
			model = a.Cfg.Model
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		fmt.Println("◌ AI review…")
		rev, err := ReviewSkillPreview(ctx, client, model, prev)
		if err != nil {
			fmt.Println("skills review:", err)
			return
		}
		fmt.Printf("Risk: %s  advisory_ok=%v\n%s\n", rev.RiskLevel, rev.OKToInstall, rev.Summary)
		if len(rev.Dependencies) > 0 {
			fmt.Println("Dependencies:")
			for _, d := range rev.Dependencies {
				fmt.Println("  -", d)
			}
		}
		if len(rev.Concerns) > 0 {
			fmt.Println("Concerns:")
			for _, c := range rev.Concerns {
				fmt.Println("  -", c)
			}
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
		if target == "--preview" || target == "preview" {
			hub := ensureCLISkillPreviews()
			prev := hub.Get(lastSkillPreviewID)
			if prev == nil {
				fmt.Println("skills install: no preview staged")
				return
			}
			sk, err = InstallSkillFromPreview(prev, scope, cwd)
		} else if c := skillCandidateByID(target); c != nil {
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
	fmt.Println("Usage: /skills | /skills search Q | /skills preview N|URL | /skills review | /skills install --preview | /skills check NAME | /skills update NAME | /skills remove NAME | /skills add [--project] PATH")
}

var (
	cliSkillPreviews   *skillPreviewHub
	lastSkillPreviewID string
)

func ensureCLISkillPreviews() *skillPreviewHub {
	if cliSkillPreviews == nil {
		cliSkillPreviews = newSkillPreviewHub()
	}
	return cliSkillPreviews
}
