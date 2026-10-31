package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	skillBodyCap       = 120_000
	skillFileCap       = 64_000
	skillCatalogMax    = 80
	skillSourceProject = "project"
	skillSourceUser    = "user"
	skillSourceCursorP = "cursor-project"
	skillSourceCursorU = "cursor-user"
)

var skillNameRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// SkillMeta is agentskills.io frontmatter (+ body for activation).
type SkillMeta struct {
	Name          string            `json:"name" yaml:"name"`
	Description   string            `json:"description" yaml:"description"`
	License       string            `json:"license,omitempty" yaml:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty" yaml:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	AllowedTools  string            `json:"allowed_tools,omitempty" yaml:"allowed-tools,omitempty"`
	Body          string            `json:"-" yaml:"-"`
	Root          string            `json:"root" yaml:"-"`
	Source        string            `json:"source" yaml:"-"` // project|user|cursor-project|cursor-user
	Writable      bool              `json:"writable" yaml:"-"`
	OriginURL     string            `json:"origin_url,omitempty" yaml:"-"`
	ContentHash   string            `json:"content_hash,omitempty" yaml:"-"`
}

type SkillRegistry struct {
	mu     sync.RWMutex
	byName map[string]*SkillMeta
	order  []string
}

func userSkillsDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "keimling", "skills")
}

func projectSkillsDir(cwd string) string {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	return filepath.Join(cwd, ".keimling", "skills")
}

func cursorUserSkillsDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".cursor", "skills")
}

func cursorProjectSkillsDir(cwd string) string {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	return filepath.Join(cwd, ".cursor", "skills")
}

func parseSkillMarkdown(raw, dirName, root, source string, writable bool) (*SkillMeta, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(strings.TrimSpace(raw), "---") {
		return nil, fmt.Errorf("SKILL.md missing YAML frontmatter")
	}
	rest := strings.TrimSpace(raw)
	rest = strings.TrimPrefix(rest, "---")
	rest = strings.TrimPrefix(rest, "\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, fmt.Errorf("SKILL.md frontmatter not closed")
	}
	fm := rest[:end]
	body := strings.TrimPrefix(rest[end+len("\n---"):], "\n")
	var meta SkillMeta
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return nil, fmt.Errorf("SKILL.md frontmatter: %w", err)
	}
	meta.Name = strings.TrimSpace(meta.Name)
	meta.Description = strings.TrimSpace(meta.Description)
	meta.Body = strings.TrimSpace(body)
	meta.Root = root
	meta.Source = source
	meta.Writable = writable
	if err := validateSkillMeta(&meta, dirName); err != nil {
		return nil, err
	}
	return &meta, nil
}

func validateSkillMeta(m *SkillMeta, dirName string) error {
	if m == nil {
		return fmt.Errorf("nil skill")
	}
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(m.Name) > 64 {
		return fmt.Errorf("name longer than 64 characters")
	}
	if !skillNameRe.MatchString(m.Name) {
		return fmt.Errorf("invalid name %q (lowercase alphanumeric and single hyphens only)", m.Name)
	}
	if dirName != "" && m.Name != dirName {
		return fmt.Errorf("name %q must match directory %q", m.Name, dirName)
	}
	if m.Description == "" {
		return fmt.Errorf("description is required")
	}
	if len(m.Description) > 1024 {
		return fmt.Errorf("description longer than 1024 characters")
	}
	if m.Compatibility != "" && len(m.Compatibility) > 500 {
		return fmt.Errorf("compatibility longer than 500 characters")
	}
	return nil
}

func loadSkillDir(root, source string, writable bool) (*SkillMeta, error) {
	root = filepath.Clean(root)
	st, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", root)
	}
	dirName := filepath.Base(root)
	b, err := os.ReadFile(filepath.Join(root, "SKILL.md"))
	if err != nil {
		return nil, err
	}
	sk, err := parseSkillMarkdown(string(b), dirName, root, source, writable)
	if err != nil {
		return nil, err
	}
	attachSkillOrigin(sk)
	return sk, nil
}

func scanSkillsRoot(root, source string, writable bool) []*SkillMeta {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []*SkillMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sk, err := loadSkillDir(filepath.Join(root, e.Name()), source, writable)
		if err != nil {
			continue
		}
		out = append(out, sk)
	}
	return out
}

// LoadSkillRegistry scans roots with precedence: project keimling → user keimling →
// project cursor → user cursor (first wins on name collision).
func LoadSkillRegistry(cwd string) *SkillRegistry {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	type root struct {
		path     string
		source   string
		writable bool
	}
	roots := []root{
		{projectSkillsDir(cwd), skillSourceProject, true},
		{userSkillsDir(), skillSourceUser, true},
		{cursorProjectSkillsDir(cwd), skillSourceCursorP, false},
		{cursorUserSkillsDir(), skillSourceCursorU, false},
	}
	reg := &SkillRegistry{byName: map[string]*SkillMeta{}}
	for _, r := range roots {
		for _, sk := range scanSkillsRoot(r.path, r.source, r.writable) {
			if _, exists := reg.byName[sk.Name]; exists {
				continue
			}
			reg.byName[sk.Name] = sk
			reg.order = append(reg.order, sk.Name)
		}
	}
	return reg
}

func (r *SkillRegistry) List() []*SkillMeta {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*SkillMeta, 0, len(r.order))
	for _, name := range r.order {
		if sk := r.byName[name]; sk != nil {
			cp := *sk
			out = append(out, &cp)
		}
	}
	return out
}

func (r *SkillRegistry) Get(name string) *SkillMeta {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	sk := r.byName[strings.TrimSpace(name)]
	if sk == nil {
		return nil
	}
	cp := *sk
	return &cp
}

func (r *SkillRegistry) CatalogBlock() string {
	list := r.List()
	if len(list) == 0 {
		return ""
	}
	if len(list) > skillCatalogMax {
		list = list[:skillCatalogMax]
	}
	var b strings.Builder
	b.WriteString("<agent_skills>\n")
	b.WriteString("Available Agent Skills (agentskills.io). Call activate_skill(name) to load SKILL.md. For another file in that skill, call read_skill_file(name, path). path is a file relative to the skill directory; do not include skills/ or the skill name.\n")
	for _, sk := range list {
		fmt.Fprintf(&b, "- %s: %s\n", sk.Name, sk.Description)
	}
	b.WriteString("</agent_skills>")
	return b.String()
}

func (r *SkillRegistry) Activate(name string) (string, error) {
	sk := r.Get(name)
	if sk == nil {
		return "", fmt.Errorf("unknown skill %q", name)
	}
	// Re-read from disk so edits apply without restart.
	fresh, err := loadSkillDir(sk.Root, sk.Source, sk.Writable)
	if err != nil {
		return "", err
	}
	body := fresh.Body
	if body == "" {
		body = "(skill has empty body)"
	}
	if len(body) > skillBodyCap {
		body = body[:skillBodyCap] + "\n…(truncated)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Activated skill %q (source=%s root=%s)\n\n", fresh.Name, fresh.Source, fresh.Root)
	if fresh.AllowedTools != "" {
		fmt.Fprintf(&b, "allowed-tools (informational): %s\n\n", fresh.AllowedTools)
	}
	b.WriteString(body)
	return b.String(), nil
}

func (r *SkillRegistry) ReadFile(name, rel string) (string, error) {
	sk := r.Get(name)
	if sk == nil {
		return "", fmt.Errorf("unknown skill %q", name)
	}
	rel = strings.TrimSpace(rel)
	rel = strings.TrimPrefix(rel, "./")
	if rel == "" || strings.Contains(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("invalid skill-relative path")
	}
	rootAbs, err := filepath.Abs(sk.Root)
	if err != nil {
		return "", err
	}
	target := filepath.Clean(filepath.Join(rootAbs, rel))
	if target != rootAbs && !strings.HasPrefix(target, rootAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes skill root")
	}
	st, err := os.Lstat(target)
	if err != nil {
		return "", err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlinks not allowed inside skills")
	}
	if !st.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file")
	}
	b, err := os.ReadFile(target)
	if err != nil {
		return "", err
	}
	if len(b) > skillFileCap {
		return string(b[:skillFileCap]) + "\n…(truncated)", nil
	}
	return string(b), nil
}

func (a *Agent) ensureSkills() *SkillRegistry {
	if a == nil {
		return LoadSkillRegistry("")
	}
	if a.Skills == nil {
		a.Skills = LoadSkillRegistry("")
	}
	return a.Skills
}

func (a *Agent) refreshSkills() {
	if a == nil {
		return
	}
	a.Skills = LoadSkillRegistry("")
}

func (a *Agent) skillsCatalogBlock() string {
	cat := a.ensureSkills().CatalogBlock()
	instr := strings.TrimSpace(a.Cfg.AgentInstructions)
	agents := ""
	if a != nil {
		agents = a.agentsMarkdownBlock()
	}
	var parts []string
	if instr != "" {
		parts = append(parts, "Agent definition instructions:\n"+instr)
	}
	if agents != "" {
		parts = append(parts, agents)
	}
	if cat != "" {
		parts = append(parts, cat)
	}
	return strings.Join(parts, "\n\n")
}
