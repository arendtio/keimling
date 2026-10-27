package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const charterVersion = 1
const charterRecentCap = 24
const charterFindingsCap = 40
const charterQuestionsCap = 12
const charterConstraintsCap = 12

// Charter is the durable structured model context (JSON source of truth).
type Charter struct {
	Version       int           `json:"version"`
	Goal          string        `json:"goal,omitempty"`
	Plan          []CharterStep `json:"plan,omitempty"`
	Position      int           `json:"position,omitempty"` // 1-based active step; 0 = unset
	Findings      []string      `json:"findings,omitempty"`
	OpenQuestions []string      `json:"open_questions,omitempty"`
	Constraints   []string      `json:"constraints,omitempty"`
	Recent        []string      `json:"recent,omitempty"` // short notes not yet folded into fields
}

type CharterStep struct {
	Step   string `json:"step"`
	Status string `json:"status"` // pending | in_progress | done
}

func charterPath(sessionPath string) string {
	return sessionPath + ".charter.json"
}

func loadCharter(sessionPath string) (*Charter, error) {
	p := charterPath(sessionPath)
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Charter{Version: charterVersion}, nil
		}
		return nil, err
	}
	var c Charter
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.Version == 0 {
		c.Version = charterVersion
	}
	return &c, nil
}

func saveCharter(sessionPath string, c *Charter) error {
	if c == nil {
		c = &Charter{Version: charterVersion}
	}
	c.Version = charterVersion
	c.clamp()
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	p := charterPath(sessionPath)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (c *Charter) clamp() {
	if c == nil {
		return
	}
	if len(c.Findings) > charterFindingsCap {
		c.Findings = c.Findings[len(c.Findings)-charterFindingsCap:]
	}
	if len(c.OpenQuestions) > charterQuestionsCap {
		c.OpenQuestions = c.OpenQuestions[len(c.OpenQuestions)-charterQuestionsCap:]
	}
	if len(c.Constraints) > charterConstraintsCap {
		c.Constraints = c.Constraints[len(c.Constraints)-charterConstraintsCap:]
	}
	if len(c.Recent) > charterRecentCap {
		c.Recent = c.Recent[len(c.Recent)-charterRecentCap:]
	}
	if c.Position < 0 {
		c.Position = 0
	}
	if c.Position > len(c.Plan) {
		c.Position = len(c.Plan)
	}
}

func (c *Charter) Empty() bool {
	if c == nil {
		return true
	}
	return strings.TrimSpace(c.Goal) == "" &&
		len(c.Plan) == 0 &&
		len(c.Findings) == 0 &&
		len(c.OpenQuestions) == 0 &&
		len(c.Constraints) == 0 &&
		len(c.Recent) == 0
}

func (c *Charter) SyncPlan(steps []PlanStep) {
	if c == nil {
		return
	}
	c.Plan = make([]CharterStep, 0, len(steps))
	pos := 0
	for i, s := range steps {
		st := strings.ToLower(strings.TrimSpace(s.Status))
		if st != "pending" && st != "in_progress" && st != "done" {
			st = "pending"
		}
		c.Plan = append(c.Plan, CharterStep{Step: strings.TrimSpace(s.Step), Status: st})
		if st == "in_progress" {
			pos = i + 1
		}
	}
	if pos == 0 {
		for i, s := range c.Plan {
			if s.Status != "done" {
				pos = i + 1
				break
			}
		}
	}
	c.Position = pos
}

func (c *Charter) AppendFinding(note string) {
	note = strings.TrimSpace(note)
	if c == nil || note == "" {
		return
	}
	c.Findings = append(c.Findings, note)
	c.clamp()
}

func (c *Charter) AppendQuestion(q string) {
	q = strings.TrimSpace(q)
	if c == nil || q == "" {
		return
	}
	c.OpenQuestions = append(c.OpenQuestions, q)
	c.clamp()
}

func (c *Charter) AppendRecent(note string) {
	note = strings.TrimSpace(note)
	if c == nil || note == "" {
		return
	}
	c.Recent = append(c.Recent, note)
	c.clamp()
}

// RenderMarkdown is the model- and GUI-facing view of the charter.
func (c *Charter) RenderMarkdown() string {
	if c == nil || c.Empty() {
		return "_Charter empty._"
	}
	var b strings.Builder
	b.WriteString("# Project charter\n")
	if g := strings.TrimSpace(c.Goal); g != "" {
		b.WriteString("\n## Goal\n")
		b.WriteString(g)
		b.WriteByte('\n')
	}
	if len(c.Plan) > 0 {
		b.WriteString("\n## Plan\n")
		for i, s := range c.Plan {
			mark := "- "
			if c.Position == i+1 {
				mark = "→ "
			}
			fmt.Fprintf(&b, "%s[%s] %s\n", mark, s.Status, s.Step)
		}
		if c.Position > 0 && c.Position <= len(c.Plan) {
			fmt.Fprintf(&b, "\nActive step: %d/%d\n", c.Position, len(c.Plan))
		}
	}
	if len(c.Findings) > 0 {
		b.WriteString("\n## Findings\n")
		for _, f := range c.Findings {
			fmt.Fprintf(&b, "- %s\n", f)
		}
	}
	if len(c.OpenQuestions) > 0 {
		b.WriteString("\n## Open questions\n")
		for _, q := range c.OpenQuestions {
			fmt.Fprintf(&b, "- %s\n", q)
		}
	}
	if len(c.Constraints) > 0 {
		b.WriteString("\n## Constraints\n")
		for _, x := range c.Constraints {
			fmt.Fprintf(&b, "- %s\n", x)
		}
	}
	if len(c.Recent) > 0 {
		b.WriteString("\n## Recent (not yet folded)\n")
		for _, r := range c.Recent {
			fmt.Fprintf(&b, "- %s\n", r)
		}
	}
	return strings.TrimSpace(b.String())
}

func (a *Agent) ensureCharter() *Charter {
	if a == nil {
		return &Charter{Version: charterVersion}
	}
	if a.Charter == nil {
		c, err := loadCharter(a.Cfg.SessionPath)
		if err != nil || c == nil {
			c = &Charter{Version: charterVersion}
		}
		a.Charter = c
	}
	return a.Charter
}

func (a *Agent) persistCharter() error {
	if a == nil {
		return nil
	}
	c := a.ensureCharter()
	if a.Cfg.SessionPath == "" {
		return nil
	}
	return saveCharter(a.Cfg.SessionPath, c)
}
