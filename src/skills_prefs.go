package main

import (
	"fmt"
	"strings"
)

var errNoSkillConfig = fmt.Errorf("no config file")

func errUnknownSkill(name string) error {
	return fmt.Errorf("unknown skill %q", name)
}

// skillPrefsConfig remembers which skills are in the prompt.
// A skill is recorded the first time it is seen. AutoEnableNew applies only then.
type skillPrefsConfig struct {
	Seeded        bool     `json:"seeded,omitempty"`
	AutoEnableNew bool     `json:"auto_enable_new"`
	Enabled       []string `json:"enabled,omitempty"`
	Disabled      []string `json:"disabled,omitempty"`
}

func skillNameListed(list []string, name string) bool {
	name = strings.TrimSpace(name)
	for _, s := range list {
		if s == name {
			return true
		}
	}
	return false
}

func skillNameDrop(list []string, name string) []string {
	name = strings.TrimSpace(name)
	out := make([]string, 0, len(list))
	for _, s := range list {
		if s != name {
			out = append(out, s)
		}
	}
	return out
}

func (p *skillPrefsConfig) note(name string) {
	name = strings.TrimSpace(name)
	if name == "" || skillNameListed(p.Enabled, name) || skillNameListed(p.Disabled, name) {
		return
	}
	if p.AutoEnableNew {
		p.Enabled = append(p.Enabled, name)
		return
	}
	p.Disabled = append(p.Disabled, name)
}

func (a *Agent) syncSkillPrefs() skillPrefsConfig {
	if a == nil || strings.TrimSpace(a.Cfg.ConfigPath) == "" {
		return skillPrefsConfig{Seeded: true, AutoEnableNew: true}
	}
	var names []string
	for _, sk := range a.ensureSkills().List() {
		if sk != nil {
			names = append(names, sk.Name)
		}
	}
	fc, err := readConfiguration(a.Cfg.ConfigPath)
	if err != nil {
		return skillPrefsConfig{Seeded: true, AutoEnableNew: true}
	}
	next := fc.Skills
	dirty := !next.Seeded
	if !next.Seeded {
		next.Seeded = true
		next.AutoEnableNew = false
		next.Enabled = append([]string(nil), names...)
		next.Disabled = nil
	} else {
		beforeE, beforeD := len(next.Enabled), len(next.Disabled)
		for _, name := range names {
			next.note(name)
		}
		dirty = len(next.Enabled) != beforeE || len(next.Disabled) != beforeD
	}
	if dirty {
		_ = updateConfiguration(a.Cfg.ConfigPath, func(cfg *fileConfig) {
			if !cfg.Skills.Seeded {
				cfg.Skills.Seeded = true
				cfg.Skills.AutoEnableNew = false
				cfg.Skills.Enabled = append([]string(nil), names...)
				cfg.Skills.Disabled = nil
				return
			}
			for _, name := range names {
				cfg.Skills.note(name)
			}
		})
		if again, err := readConfiguration(a.Cfg.ConfigPath); err == nil {
			return again.Skills
		}
	}
	return next
}

func (a *Agent) skillEnabled(name string) bool {
	if a == nil || strings.TrimSpace(a.Cfg.ConfigPath) == "" {
		return true
	}
	fc, err := readConfiguration(a.Cfg.ConfigPath)
	if err != nil || !fc.Skills.Seeded {
		return true
	}
	name = strings.TrimSpace(name)
	if skillNameListed(fc.Skills.Disabled, name) {
		return false
	}
	return skillNameListed(fc.Skills.Enabled, name)
}

func (a *Agent) setSkillEnabled(name string, on bool) error {
	name = strings.TrimSpace(name)
	if a == nil || strings.TrimSpace(a.Cfg.ConfigPath) == "" {
		return errNoSkillConfig
	}
	if a.ensureSkills().Get(name) == nil {
		return errUnknownSkill(name)
	}
	return updateConfiguration(a.Cfg.ConfigPath, func(cfg *fileConfig) {
		if !cfg.Skills.Seeded {
			cfg.Skills.Seeded = true
		}
		cfg.Skills.Enabled = skillNameDrop(cfg.Skills.Enabled, name)
		cfg.Skills.Disabled = skillNameDrop(cfg.Skills.Disabled, name)
		if on {
			cfg.Skills.Enabled = append(cfg.Skills.Enabled, name)
			return
		}
		cfg.Skills.Disabled = append(cfg.Skills.Disabled, name)
	})
}

func (a *Agent) setSkillAutoEnable(on bool) error {
	if a == nil || strings.TrimSpace(a.Cfg.ConfigPath) == "" {
		return errNoSkillConfig
	}
	return updateConfiguration(a.Cfg.ConfigPath, func(cfg *fileConfig) {
		cfg.Skills.Seeded = true
		cfg.Skills.AutoEnableNew = on
	})
}
