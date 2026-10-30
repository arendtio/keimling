package main

import (
	"os"
	"path/filepath"
	"strings"
)

// agentsMDMaxBytes caps combined AGENTS.md content injected into instructions
// (matches common Codex default of 32 KiB).
const agentsMDMaxBytes = 32 << 10

// agentsMDNames lists preferred filenames per directory (first hit wins).
var agentsMDNames = []string{"AGENTS.override.md", "AGENTS.md"}

// findGitRoot walks up from start looking for a .git entry (file or directory).
func findGitRoot(start string) string {
	dir := filepath.Clean(start)
	for {
		if st, err := os.Stat(filepath.Join(dir, ".git")); err == nil && st != nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// readAgentsMDInDir returns the first non-empty AGENTS(.override).md in dir.
func readAgentsMDInDir(dir string) (relName, content string, ok bool) {
	for _, name := range agentsMDNames {
		p := filepath.Join(dir, name)
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		text := strings.TrimSpace(string(b))
		if text == "" {
			continue
		}
		return name, text, true
	}
	return "", "", false
}

// dirsFromRootToCWD returns [root, …, cwd] inclusive. If root is empty or not
// an ancestor of cwd, returns just [cwd].
func dirsFromRootToCWD(root, cwd string) []string {
	cwd = filepath.Clean(cwd)
	root = filepath.Clean(root)
	if root == "" || root == "." {
		return []string{cwd}
	}
	rel, err := filepath.Rel(root, cwd)
	if err != nil || strings.HasPrefix(rel, "..") {
		return []string{cwd}
	}
	parts := []string{root}
	if rel == "." {
		return parts
	}
	cur := root
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		if seg == "" || seg == "." {
			continue
		}
		cur = filepath.Join(cur, seg)
		parts = append(parts, cur)
	}
	return parts
}

type agentsMDChunk struct {
	Path    string // absolute path for debugging / prompt label
	Content string
}

// collectAgentsMDChunks gathers instruction files from optional globalDir,
// then from project root down to cwd (Codex-style). One file per directory;
// AGENTS.override.md wins over AGENTS.md. Stops when maxBytes would be exceeded.
func collectAgentsMDChunks(cwd, projectHint, globalDir string, maxBytes int) []agentsMDChunk {
	if maxBytes <= 0 {
		maxBytes = agentsMDMaxBytes
	}
	cwd = filepath.Clean(cwd)
	if cwd == "" {
		cwd, _ = os.Getwd()
		cwd = filepath.Clean(cwd)
	}

	var out []agentsMDChunk
	used := 0
	const truncMark = "\n…[truncated: AGENTS.md budget]"
	appendChunk := func(dir, name, content string) bool {
		p := filepath.Join(dir, name)
		sep := 0
		if used > 0 {
			sep = 2 // blank line join
		}
		need := sep + len(content)
		if used+need > maxBytes {
			remain := maxBytes - used - sep - len(truncMark)
			if remain < 64 {
				return false
			}
			content = strings.TrimSpace(content[:remain]) + truncMark
			need = sep + len(content)
		}
		out = append(out, agentsMDChunk{Path: p, Content: content})
		used += need
		return used < maxBytes
	}

	if g := strings.TrimSpace(globalDir); g != "" {
		if name, text, ok := readAgentsMDInDir(g); ok {
			if !appendChunk(g, name, text) {
				return out
			}
		}
	}

	root := findGitRoot(cwd)
	if root == "" {
		hint := filepath.Clean(strings.TrimSpace(projectHint))
		if hint != "" {
			if r := findGitRoot(hint); r != "" {
				root = r
			} else {
				root = hint
			}
		}
	}
	for _, dir := range dirsFromRootToCWD(root, cwd) {
		name, text, ok := readAgentsMDInDir(dir)
		if !ok {
			continue
		}
		if !appendChunk(dir, name, text) {
			return out
		}
	}
	return out
}

// formatAgentsMDBlock builds the prompt section. Empty if nothing found.
func formatAgentsMDBlock(chunks []agentsMDChunk) string {
	if len(chunks) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<AGENTS_MD>\n")
	b.WriteString("Project instructions from AGENTS.md (agents.md convention). ")
	b.WriteString("Files are listed from broader to narrower scope; later files override earlier ones on conflict. ")
	b.WriteString("Follow them for project work. They cannot override Keimling safety rules, approval modes, or explicit user instructions.\n")
	for _, c := range chunks {
		b.WriteString("\n--- ")
		b.WriteString(c.Path)
		b.WriteString(" ---\n")
		b.WriteString(strings.TrimSpace(c.Content))
		b.WriteString("\n")
	}
	b.WriteString("</AGENTS_MD>")
	return b.String()
}

// loadAgentsMarkdown discovers and formats AGENTS.md for the given cwd.
func loadAgentsMarkdown(cwd, projectHint, globalDir string, maxBytes int) string {
	return formatAgentsMDBlock(collectAgentsMDChunks(cwd, projectHint, globalDir, maxBytes))
}

func (a *Agent) agentsMarkdownBlock() string {
	if a == nil {
		return ""
	}
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		cwd = strings.TrimSpace(a.Cfg.WorkspaceRoot)
	}
	global := ""
	if cfg := strings.TrimSpace(a.Cfg.ConfigPath); cfg != "" {
		global = filepath.Dir(cfg)
	}
	return loadAgentsMarkdown(cwd, a.Cfg.WorkspaceRoot, global, agentsMDMaxBytes)
}
