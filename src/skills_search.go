package main

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// SkillCandidate is a search hit the user can install with one click.
type SkillCandidate struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	SourceURL   string `json:"source_url"`
	InstallHint string `json:"install_hint,omitempty"`
}

var (
	mdLinkRe          = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)\s]+)\)`)
	bareURLRe         = regexp.MustCompile(`https?://[^\s)>\]]+`)
	githubSkillPathRe = regexp.MustCompile(`(?i)github\.com/[^/\s]+/[^/\s]+(?:/(?:tree|blob)/[^/\s]+)?(?:/[\w./-]*)?`)
)

// SearchSkillsOnline uses the OpenAI web_search client and extracts installable candidates.
func SearchSkillsOnline(ctx context.Context, client *Client, query string) ([]SkillCandidate, string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, "", fmt.Errorf("empty query")
	}
	if client == nil {
		return nil, "", fmt.Errorf("web search unavailable")
	}
	prompt := fmt.Sprintf(
		`Find public Agent Skills (agentskills.io / SKILL.md directories) for: %s
Prefer GitHub repositories or paths that contain a SKILL.md file with YAML frontmatter (name + description).
Return a short list with title, one-line description, and exact https URLs. Do not invent URLs.`,
		query,
	)
	text, err := client.SearchWeb(ctx, prompt)
	if err != nil {
		return nil, "", err
	}
	cands := extractSkillCandidates(text)
	return cands, text, nil
}

func extractSkillCandidates(text string) []SkillCandidate {
	seen := map[string]bool{}
	var out []SkillCandidate
	add := func(title, desc, rawURL string) {
		rawURL = strings.TrimRight(strings.TrimSpace(rawURL), ".,);\"'")
		if rawURL == "" || seen[rawURL] {
			return
		}
		if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
			return
		}
		u, err := url.Parse(rawURL)
		if err != nil || u.Host == "" {
			return
		}
		host := strings.ToLower(u.Host)
		// Prefer GitHub; still allow other hosts that clearly point at SKILL.md.
		if host != "github.com" && host != "www.github.com" && host != "raw.githubusercontent.com" {
			if !strings.Contains(strings.ToLower(rawURL), "skill.md") {
				return
			}
		}
		seen[rawURL] = true
		if title == "" {
			title = filepathBaseURL(rawURL)
		}
		id := fmt.Sprintf("%d", len(out)+1)
		out = append(out, SkillCandidate{
			ID:          id,
			Title:       truncate(title, 120),
			Description: truncate(strings.TrimSpace(desc), 240),
			SourceURL:   rawURL,
			InstallHint: "Preview downloads the full skill folder, then install after review",
		})
	}

	for _, m := range mdLinkRe.FindAllStringSubmatch(text, -1) {
		add(m[1], "", m[2])
	}
	for _, u := range bareURLRe.FindAllString(text, -1) {
		if githubSkillPathRe.MatchString(u) || strings.Contains(strings.ToLower(u), "skill.md") {
			add("", "", u)
		}
	}
	if len(out) > 20 {
		out = out[:20]
	}
	return out
}

func filepathBaseURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	p := strings.Trim(u.Path, "/")
	parts := strings.Split(p, "/")
	if len(parts) == 0 {
		return u.Host
	}
	return parts[len(parts)-1]
}

// LastSkillSearch holds CLI numbered install targets for the process.
var lastSkillSearch []SkillCandidate

func skillCandidateByID(id string) *SkillCandidate {
	id = strings.TrimSpace(id)
	for i := range lastSkillSearch {
		if lastSkillSearch[i].ID == id {
			cp := lastSkillSearch[i]
			return &cp
		}
	}
	return nil
}
