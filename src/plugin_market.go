package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// A marketplace catalog wins over a plugin manifest in the same checkout.
// Runnable code (hooks and bin/) stays idle until its name is granted.

type pluginUnit struct {
	Root    string
	Name    string
	Format  string
	Version string
	Summary string
}

type pluginCodeStatus struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Command string `json:"command"`
	Event   string `json:"event,omitempty"`
	Granted bool   `json:"granted"`
}

type pluginCodeEntry struct {
	Name    string
	Kind    string
	Command string
	Event   string
	Root    string
}

type marketplaceCatalog struct {
	Name    string
	Plugins []marketplaceEntry
}

type marketplaceEntry struct {
	Name   string
	Source json.RawMessage
}

func describePlugin(rec pluginRecord, fetch bool) pluginStatus {
	st := pluginStatus{ID: rec.ID, Kind: rec.Kind, Location: rec.Location, Enabled: rec.Enabled, Layout: "plugin"}
	root, err := pluginRoot(rec, fetch)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	units, notes, layout, title, err := loadSourceUnits(root)
	st.Notes = append(st.Notes, notes...)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	st.Layout = layout
	st.Name = title
	seenSkill := map[string]bool{}
	seenServer := map[string]bool{}
	for _, unit := range units {
		if layout == "marketplace" {
			st.Plugins = append(st.Plugins, unit.Name)
		} else {
			st.Description = unit.Summary
			st.Version = unit.Version
		}
		for _, sk := range pluginSkills(unit.Root) {
			if seenSkill[sk.Name] {
				continue
			}
			seenSkill[sk.Name] = true
			st.Skills = append(st.Skills, sk.Name)
		}
		for _, rule := range pluginRuleFiles(unit.Root) {
			st.Rules = append(st.Rules, rule.Name)
		}
		for _, cmd := range pluginCommandFiles(unit.Root) {
			st.Commands = append(st.Commands, cmd.Name)
		}
		for _, ag := range pluginAgentFiles(unit.Root) {
			st.Agents = append(st.Agents, ag.Name)
		}
		servers, serverNotes := pluginServers(rec.ID, unit.Root)
		st.Notes = append(st.Notes, serverNotes...)
		var names []string
		for name := range servers {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if seenServer[name] {
				continue
			}
			seenServer[name] = true
			st.Servers = append(st.Servers, name)
		}
		for _, code := range pluginCodeEntries(unit.Root) {
			st.Code = append(st.Code, pluginCodeStatus{
				Name: code.Name, Kind: code.Kind, Command: code.Command, Event: code.Event,
				Granted: pluginGrantHas(rec, code.Name),
			})
		}
	}
	sort.Strings(st.Skills)
	sort.Strings(st.Rules)
	sort.Strings(st.Commands)
	sort.Strings(st.Agents)
	sort.Strings(st.Plugins)
	if layout == "marketplace" {
		for _, name := range st.Plugins {
			if pluginUnitOn(rec, name) {
				st.Active = append(st.Active, name)
			}
		}
	}
	return st
}

// pluginUnitOn reports whether a marketplace plugin contributes. A source that
// has never stored a choice still contributes every plugin it already had on.
func pluginUnitOn(src pluginRecord, name string) bool {
	if !src.PluginSet {
		return true
	}
	for _, n := range src.Units {
		if n == name {
			return true
		}
	}
	return false
}

// migratePluginUnits records a choice for sources saved before per-plugin
// switches existed. An enabled marketplace keeps the plugins it was already
// running. A source that was off starts with none on.
func migratePluginUnits() error {
	f, err := readPluginFile()
	if err != nil {
		return err
	}
	changed := false
	for i := range f.Sources {
		src := &f.Sources[i]
		if src.PluginSet {
			continue
		}
		if !src.Enabled {
			src.PluginSet = true
			src.Units = nil
			changed = true
			continue
		}
		root, err := pluginRoot(*src, false)
		if err != nil {
			continue
		}
		units, _, layout, _, err := loadSourceUnits(root)
		if err != nil {
			continue
		}
		src.PluginSet = true
		if layout == "marketplace" {
			var names []string
			for _, unit := range units {
				if unit.Name != "" {
					names = append(names, unit.Name)
				}
			}
			sort.Strings(names)
			src.Units = names
		}
		changed = true
	}
	if !changed {
		return nil
	}
	return writePluginFile(f)
}

func setPluginUnit(id, name string, on bool) (pluginStatus, error) {
	if err := migratePluginUnits(); err != nil {
		return pluginStatus{}, err
	}
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return pluginStatus{}, fmt.Errorf("plugin name required")
	}
	f, err := readPluginFile()
	if err != nil {
		return pluginStatus{}, err
	}
	for i := range f.Sources {
		if f.Sources[i].ID != id {
			continue
		}
		root, err := pluginRoot(f.Sources[i], false)
		if err != nil {
			return pluginStatus{}, err
		}
		units, _, layout, _, err := loadSourceUnits(root)
		if err != nil {
			return pluginStatus{}, err
		}
		if layout != "marketplace" {
			return pluginStatus{}, fmt.Errorf("this source is one plugin")
		}
		known := false
		for _, unit := range units {
			if unit.Name == name {
				known = true
				break
			}
		}
		if !known {
			return pluginStatus{}, fmt.Errorf("unknown plugin %q", name)
		}
		have := map[string]bool{}
		var next []string
		for _, n := range f.Sources[i].Units {
			if n == name && !on {
				continue
			}
			if n == "" || have[n] {
				continue
			}
			have[n] = true
			next = append(next, n)
		}
		if on && !have[name] {
			next = append(next, name)
		}
		sort.Strings(next)
		f.Sources[i].PluginSet = true
		f.Sources[i].Units = next
		if err := writePluginFile(f); err != nil {
			return pluginStatus{}, err
		}
		return describePlugin(f.Sources[i], false), nil
	}
	return pluginStatus{}, fmt.Errorf("unknown plugin %q", id)
}

func loadSourceUnits(root string) (units []pluginUnit, notes []string, layout, title string, err error) {
	cats, notes, found, err := readMarketplaceCatalogs(root)
	if err != nil {
		return nil, notes, "", "", err
	}
	if found {
		title = cats[0].Name
		seen := map[string]bool{}
		for _, cat := range cats {
			for _, entry := range cat.Plugins {
				name := strings.TrimSpace(entry.Name)
				if name == "" || seen[name] {
					continue
				}
				seen[name] = true
				dir, skip := resolveMarketplaceSource(root, entry.Source)
				if skip != "" {
					notes = append(notes, name+": "+skip)
					continue
				}
				unit, uerr := readPluginUnit(dir)
				if uerr != nil {
					notes = append(notes, name+": "+uerr.Error())
					continue
				}
				if unit.Name == "" {
					unit.Name = name
				}
				units = append(units, unit)
			}
		}
		return units, notes, "marketplace", title, nil
	}
	unit, err := readPluginUnit(root)
	if err != nil {
		return nil, notes, "plugin", "", err
	}
	return []pluginUnit{unit}, notes, "plugin", unit.Name, nil
}

func readMarketplaceCatalogs(root string) ([]marketplaceCatalog, []string, bool, error) {
	var cats []marketplaceCatalog
	var notes []string
	found := false
	for _, rel := range []string{filepath.Join(".cursor-plugin", "marketplace.json"), filepath.Join(".claude-plugin", "marketplace.json")} {
		path := filepath.Join(root, rel)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		found = true
		cat, err := readMarketplaceCatalog(path)
		if err != nil {
			return nil, notes, true, err
		}
		cats = append(cats, cat)
	}
	return cats, notes, found, nil
}

func readMarketplaceCatalog(path string) (marketplaceCatalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return marketplaceCatalog{}, err
	}
	if len(b) > 10<<20 {
		return marketplaceCatalog{}, fmt.Errorf("%s is larger than 10 MB", filepath.Base(path))
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return marketplaceCatalog{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	for _, key := range []string{"name", "owner", "plugins"} {
		if _, ok := raw[key]; !ok {
			return marketplaceCatalog{}, fmt.Errorf("%s: %s required", filepath.Base(path), key)
		}
	}
	var name string
	if err := json.Unmarshal(raw["name"], &name); err != nil || strings.TrimSpace(name) == "" {
		return marketplaceCatalog{}, fmt.Errorf("%s: name required", filepath.Base(path))
	}
	name = strings.TrimSpace(name)
	if !validPluginName(name) {
		return marketplaceCatalog{}, fmt.Errorf("%s: invalid name %q", filepath.Base(path), name)
	}
	var owner struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw["owner"], &owner); err != nil || strings.TrimSpace(owner.Name) == "" {
		return marketplaceCatalog{}, fmt.Errorf("%s: owner name required", filepath.Base(path))
	}
	var entries []marketplaceEntry
	if err := json.Unmarshal(raw["plugins"], &entries); err != nil {
		return marketplaceCatalog{}, fmt.Errorf("%s: plugins must be a list", filepath.Base(path))
	}
	return marketplaceCatalog{Name: name, Plugins: entries}, nil
}

func resolveMarketplaceSource(root string, raw json.RawMessage) (string, string) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", "source required"
	}
	var text string
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &text); err != nil {
			return "", "source is not a path"
		}
		return marketplaceDir(root, text)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", "source is not a path"
	}
	if rawPath, ok := obj["path"]; ok {
		var path string
		if err := json.Unmarshal(rawPath, &path); err == nil {
			if dir, skip := marketplaceDir(root, path); skip == "" {
				return dir, ""
			}
		}
	}
	var kind string
	if rawKind, ok := obj["source"]; ok {
		_ = json.Unmarshal(rawKind, &kind)
	}
	kind = strings.TrimSpace(kind)
	if dir, skip := marketplaceDir(root, kind); skip == "" {
		return dir, ""
	}
	if kind == "" {
		kind = "external"
	}
	return "", "outside this repository (" + kind + ") and was not loaded"
}

func marketplaceDir(root, source string) (string, string) {
	source = strings.TrimSpace(source)
	if source == "" || source == "." || source == "./" {
		return root, ""
	}
	if !relativePluginSource(source) {
		return "", "outside this repository (" + source + ") and was not loaded"
	}
	clean := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(source, "./")))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", "source escapes the repository"
	}
	dir := filepath.Join(root, clean)
	if !withinRoot(root, dir) {
		return "", "source escapes the repository"
	}
	if !dirExists(dir) {
		return "", "directory not found"
	}
	return dir, ""
}

func relativePluginSource(source string) bool {
	if source == "" || strings.ContainsAny(source, "\r\n") {
		return false
	}
	lower := strings.ToLower(source)
	switch lower {
	case "github", "git", "git-subdir", "url", "archive", "npm", "pip", "command":
		return false
	}
	if strings.Contains(source, "://") || strings.HasPrefix(source, "git@") {
		return false
	}
	if filepath.IsAbs(source) {
		return false
	}
	return true
}

func readPluginUnit(root string) (pluginUnit, error) {
	if m, err := readLooseManifest(filepath.Join(root, ".cursor-plugin", "plugin.json")); err == nil {
		return pluginUnit{Root: root, Name: m.Name, Format: "cursor", Version: m.Version, Summary: m.Description}, nil
	}
	if m, err := readLooseManifest(filepath.Join(root, ".claude-plugin", "plugin.json")); err == nil {
		return pluginUnit{Root: root, Name: m.Name, Format: "claude", Version: m.Version, Summary: m.Description}, nil
	}
	m, err := readPluginManifest(root)
	if err != nil {
		return pluginUnit{}, err
	}
	return pluginUnit{Root: root, Name: m.Name, Format: "agent", Version: m.Version, Summary: m.Description}, nil
}

func readLooseManifest(path string) (pluginManifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return pluginManifest{}, err
	}
	var m pluginManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return pluginManifest{}, err
	}
	m.Name = strings.TrimSpace(m.Name)
	if !validPluginName(m.Name) {
		return pluginManifest{}, fmt.Errorf("invalid plugin name %q", m.Name)
	}
	return m, nil
}

func eachEnabledUnit(fn func(pluginRecord, pluginUnit)) {
	if err := migratePluginUnits(); err != nil {
		return
	}
	f, err := readPluginFile()
	if err != nil {
		return
	}
	for _, src := range f.Sources {
		if !src.Enabled {
			continue
		}
		root, err := pluginRoot(src, false)
		if err != nil {
			continue
		}
		units, _, layout, _, err := loadSourceUnits(root)
		if err != nil {
			continue
		}
		for _, unit := range units {
			if layout == "marketplace" && !pluginUnitOn(src, unit.Name) {
				continue
			}
			fn(src, unit)
		}
	}
}

type pluginText struct {
	Name        string
	Description string
	Body        string
	Globs       string
	Plugin      string
}

func pluginRuleFiles(root string) []pluginText {
	return pluginMarkdownDir(root, "rules", map[string]bool{".md": true, ".mdc": true, ".markdown": true})
}

func pluginCommandFiles(root string) []pluginText {
	return pluginMarkdownDir(root, "commands", map[string]bool{".md": true, ".txt": true, ".markdown": true})
}

func pluginAgentFiles(root string) []pluginText {
	return pluginMarkdownDir(root, "agents", map[string]bool{".md": true, ".mdc": true, ".markdown": true})
}

func pluginMarkdownDir(root, dir string, exts map[string]bool) []pluginText {
	base := filepath.Join(root, dir)
	entries, err := os.ReadDir(base)
	if err != nil || !withinRoot(root, base) {
		return nil
	}
	var out []pluginText
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !exts[ext] {
			continue
		}
		path := filepath.Join(base, e.Name())
		if !withinRoot(root, path) {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil || len(b) > 64_000 {
			continue
		}
		meta, body := splitPluginFrontmatter(string(b))
		name := strings.TrimSpace(meta["name"])
		if name == "" {
			name = strings.TrimSuffix(e.Name(), ext)
		}
		name = strings.TrimSpace(name)
		if name == "" || strings.ContainsAny(name, "\r\n") || len(name) > 64 {
			continue
		}
		out = append(out, pluginText{
			Name: name, Description: strings.TrimSpace(meta["description"]), Body: strings.TrimSpace(body),
			Globs: pluginGlobText(meta["globs"]),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func splitPluginFrontmatter(raw string) (map[string]string, string) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return map[string]string{}, strings.TrimSpace(raw)
	}
	rest := raw[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return map[string]string{}, strings.TrimSpace(raw)
	}
	body := rest[end+4:]
	body = strings.TrimPrefix(body, "\n")
	var node map[string]any
	if err := yaml.Unmarshal([]byte(rest[:end]), &node); err != nil {
		return map[string]string{}, strings.TrimSpace(raw)
	}
	out := map[string]string{}
	for k, v := range node {
		switch t := v.(type) {
		case string:
			out[k] = t
		case bool:
			if t {
				out[k] = "true"
			} else {
				out[k] = "false"
			}
		default:
			b, err := yaml.Marshal(t)
			if err == nil {
				out[k] = strings.TrimSpace(string(b))
			}
		}
	}
	return out, strings.TrimSpace(body)
}

func pluginGlobText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return ""
	}
	return raw
}

func pluginRulesText() string {
	var b strings.Builder
	eachEnabledUnit(func(_ pluginRecord, unit pluginUnit) {
		for _, rule := range pluginRuleFiles(unit.Root) {
			if b.Len() > 32<<10 {
				return
			}
			fmt.Fprintf(&b, "\n\n## %s (%s)\n", rule.Name, unit.Name)
			if rule.Globs != "" {
				fmt.Fprintf(&b, "Applies when the files match: %s\n", rule.Globs)
			}
			if rule.Description != "" {
				fmt.Fprintf(&b, "%s\n", rule.Description)
			}
			b.WriteString(rule.Body)
		}
	})
	text := strings.TrimSpace(b.String())
	if text == "" {
		return ""
	}
	if len(text) > 32<<10 {
		text = text[:32<<10]
	}
	return "Plugin rules:\n" + text
}

type pluginCommand struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Body        string `json:"body"`
	Plugin      string `json:"plugin,omitempty"`
}

func listPluginCommands() []pluginCommand {
	seen := map[string]bool{}
	var out []pluginCommand
	eachEnabledUnit(func(_ pluginRecord, unit pluginUnit) {
		for _, cmd := range pluginCommandFiles(unit.Root) {
			name := strings.ToLower(cmd.Name)
			if seen[name] || !validSlashName(name) {
				continue
			}
			seen[name] = true
			body := cmd.Body
			if body == "" {
				body = cmd.Description
			}
			out = append(out, pluginCommand{Name: name, Description: cmd.Description, Body: body, Plugin: unit.Name})
		}
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func validSlashName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '-' && i > 0:
		default:
			return false
		}
	}
	return true
}

func pluginCodeEntries(root string) []pluginCodeEntry {
	var out []pluginCodeEntry
	out = append(out, pluginHookEntries(root)...)
	out = append(out, pluginBinEntries(root)...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
	seen := map[string]bool{}
	var uniq []pluginCodeEntry
	for _, e := range out {
		if seen[e.Name] {
			continue
		}
		seen[e.Name] = true
		e.Root = root
		uniq = append(uniq, e)
	}
	return uniq
}

func pluginHookEntries(root string) []pluginCodeEntry {
	path := filepath.Join(root, "hooks", "hooks.json")
	b, err := os.ReadFile(path)
	if err != nil || !withinRoot(root, path) || len(b) > 1<<20 {
		return nil
	}
	var doc struct {
		Hooks map[string]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil
	}
	var out []pluginCodeEntry
	for event, raw := range doc.Hooks {
		canonical, ok := canonicalHookEvent(event)
		if !ok {
			continue
		}
		for _, cmd := range hookCommands(raw) {
			name := cmd.name
			if name == "" {
				name = cmd.command
			}
			if !validCodeName(name) || !validHookCommand(cmd.command) {
				continue
			}
			out = append(out, pluginCodeEntry{Name: name, Kind: "hook", Command: cmd.command, Event: canonical, Root: root})
		}
	}
	return out
}

type hookCommand struct {
	name    string
	command string
}

func hookCommands(raw json.RawMessage) []hookCommand {
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	var out []hookCommand
	for _, item := range items {
		if nested, ok := item["hooks"]; ok {
			out = append(out, hookCommands(nested)...)
			continue
		}
		var command, name string
		if rawCmd, ok := item["command"]; ok {
			_ = json.Unmarshal(rawCmd, &command)
		}
		if rawName, ok := item["name"]; ok {
			_ = json.Unmarshal(rawName, &name)
		}
		command = strings.TrimSpace(command)
		name = strings.TrimSpace(name)
		if command == "" {
			continue
		}
		out = append(out, hookCommand{name: name, command: command})
	}
	return out
}

func canonicalHookEvent(event string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(event)) {
	case "sessionstart":
		return "sessionStart", true
	case "sessionend", "stop":
		return "stop", true
	case "pretooluse":
		return "preToolUse", true
	case "posttooluse":
		return "postToolUse", true
	case "beforeshellexecution":
		return "beforeShellExecution", true
	case "aftershellexecution":
		return "afterShellExecution", true
	case "precompact":
		return "preCompact", true
	default:
		return "", false
	}
}

func validCodeName(name string) bool {
	if name == "" || len(name) > 200 || strings.ContainsAny(name, "\r\n") {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.' || r == '/' || r == ':' || r == ' ':
		default:
			return false
		}
	}
	return true
}

func validHookCommand(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" || len(command) > 500 || strings.ContainsAny(command, "\r\n") {
		return false
	}
	return true
}

func pluginBinEntries(root string) []pluginCodeEntry {
	dir := filepath.Join(root, "bin")
	entries, err := os.ReadDir(dir)
	if err != nil || !withinRoot(root, dir) {
		return nil
	}
	var out []pluginCodeEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !validCodeName(name) || strings.Contains(name, "/") {
			continue
		}
		path := filepath.Join(dir, name)
		if !withinRoot(root, path) {
			continue
		}
		out = append(out, pluginCodeEntry{Name: name, Kind: "exec", Command: "./bin/" + name, Root: root})
	}
	return out
}

func pluginGrantHas(rec pluginRecord, name string) bool {
	for _, g := range rec.Grants {
		if g == name {
			return true
		}
	}
	return false
}

func grantPluginCode(id string, names []string) (pluginStatus, error) {
	return setPluginGrants(id, names, true)
}

func revokePluginCode(id string, names []string) (pluginStatus, error) {
	return setPluginGrants(id, names, false)
}

func setPluginGrants(id string, names []string, grant bool) (pluginStatus, error) {
	f, err := readPluginFile()
	if err != nil {
		return pluginStatus{}, err
	}
	want := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			want[name] = true
		}
	}
	if len(want) == 0 {
		return pluginStatus{}, fmt.Errorf("name required")
	}
	for i := range f.Sources {
		if f.Sources[i].ID != id {
			continue
		}
		root, err := pluginRoot(f.Sources[i], false)
		if err != nil {
			return pluginStatus{}, err
		}
		units, _, _, _, err := loadSourceUnits(root)
		if err != nil {
			return pluginStatus{}, err
		}
		known := map[string]bool{}
		for _, unit := range units {
			for _, code := range pluginCodeEntries(unit.Root) {
				known[code.Name] = true
			}
		}
		for name := range want {
			if !known[name] {
				return pluginStatus{}, fmt.Errorf("unknown code %q", name)
			}
		}
		have := map[string]bool{}
		var next []string
		for _, g := range f.Sources[i].Grants {
			if want[g] && !grant {
				continue
			}
			if have[g] {
				continue
			}
			have[g] = true
			next = append(next, g)
		}
		if grant {
			var add []string
			for name := range want {
				if !have[name] {
					add = append(add, name)
				}
			}
			sort.Strings(add)
			next = append(next, add...)
		}
		f.Sources[i].Grants = next
		if err := writePluginFile(f); err != nil {
			return pluginStatus{}, err
		}
		return describePlugin(f.Sources[i], false), nil
	}
	return pluginStatus{}, fmt.Errorf("unknown plugin %q", id)
}

func runPluginHooks(event, payload string) string {
	event = strings.TrimSpace(event)
	if event == "" {
		return ""
	}
	var parts []string
	eachEnabledUnit(func(src pluginRecord, unit pluginUnit) {
		for _, code := range pluginCodeEntries(unit.Root) {
			if code.Kind != "hook" || code.Event != event || !pluginGrantHas(src, code.Name) {
				continue
			}
			if text := runOneHook(src, unit, code, payload); text != "" {
				parts = append(parts, text)
			}
		}
	})
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func runOneHook(src pluginRecord, unit pluginUnit, code pluginCodeEntry, payload string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", code.Command)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", code.Command)
	}
	cmd.Dir = unit.Root
	cmd.Stdin = strings.NewReader(payload)
	data := pluginDataDir(src.ID)
	_ = os.MkdirAll(data, 0o700)
	cmd.Env = append(os.Environ(),
		"PLUGIN_ROOT="+unit.Root,
		"PLUGIN_DATA="+data,
		"KEIMLING_HOOK_EVENT="+code.Event,
	)
	var output limitedBuffer
	output.Limit = 4 << 10
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	text := strings.TrimSpace(output.String())
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return "hook " + code.Name + " failed: " + text
	}
	if text == "" {
		return ""
	}
	return "hook " + code.Name + ":\n" + text
}

var pluginBinMu sync.Mutex

func pluginExecDir() string {
	pluginBinMu.Lock()
	defer pluginBinMu.Unlock()
	dir := filepath.Join(dataHome(), "keimling", "plugins", "bin")
	_ = os.RemoveAll(dir)
	n := 0
	eachEnabledUnit(func(src pluginRecord, unit pluginUnit) {
		for _, code := range pluginCodeEntries(unit.Root) {
			if code.Kind != "exec" || !pluginGrantHas(src, code.Name) {
				continue
			}
			target := filepath.Join(unit.Root, "bin", code.Name)
			if !withinRoot(unit.Root, target) {
				continue
			}
			if n == 0 {
				if err := os.MkdirAll(dir, 0o700); err != nil {
					return
				}
			}
			link := filepath.Join(dir, code.Name)
			if err := os.Symlink(target, link); err != nil {
				continue
			}
			n++
		}
	})
	if n == 0 {
		return ""
	}
	return dir
}

func overlayEnv(base, extra []string) []string {
	if len(extra) == 0 {
		return base
	}
	out := append([]string{}, base...)
	for _, kv := range extra {
		key, _, ok := strings.Cut(kv, "=")
		if !ok || key == "" {
			continue
		}
		prefix := key + "="
		replaced := false
		for i, cur := range out {
			if strings.HasPrefix(cur, prefix) {
				out[i] = kv
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, kv)
		}
	}
	return out
}

func commandEnv(extra []string, binDir string) []string {
	env := append(os.Environ(), extra...)
	if binDir == "" {
		return env
	}
	prefix := binDir + string(os.PathListSeparator)
	for i, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			env[i] = "PATH=" + prefix + strings.TrimPrefix(kv, "PATH=")
			return env
		}
	}
	return append(env, "PATH="+binDir)
}

func startPluginAgent(sourceID, agentName, workspaceID string) (AgentInstanceMeta, error) {
	sourceID = strings.TrimSpace(sourceID)
	agentName = strings.TrimSpace(agentName)
	workspaceID = strings.TrimSpace(workspaceID)
	if sourceID == "" || agentName == "" {
		return AgentInstanceMeta{}, fmt.Errorf("plugin and agent name required")
	}
	f, err := readPluginFile()
	if err != nil {
		return AgentInstanceMeta{}, err
	}
	var rec *pluginRecord
	for i := range f.Sources {
		if f.Sources[i].ID == sourceID {
			rec = &f.Sources[i]
			break
		}
	}
	if rec == nil {
		return AgentInstanceMeta{}, fmt.Errorf("unknown plugin %q", sourceID)
	}
	if !rec.Enabled {
		return AgentInstanceMeta{}, fmt.Errorf("plugin is off")
	}
	root, err := pluginRoot(*rec, false)
	if err != nil {
		return AgentInstanceMeta{}, err
	}
	units, _, layout, _, err := loadSourceUnits(root)
	if err != nil {
		return AgentInstanceMeta{}, err
	}
	var found *pluginText
	var pluginName string
	inactive := false
	for _, unit := range units {
		for _, ag := range pluginAgentFiles(unit.Root) {
			if ag.Name != agentName {
				continue
			}
			if layout == "marketplace" && !pluginUnitOn(*rec, unit.Name) {
				inactive = true
				continue
			}
			cp := ag
			found = &cp
			pluginName = unit.Name
			break
		}
	}
	if found == nil && inactive {
		return AgentInstanceMeta{}, fmt.Errorf("plugin is off")
	}
	if found == nil {
		return AgentInstanceMeta{}, fmt.Errorf("unknown agent %q", agentName)
	}
	sum := sha256.Sum256([]byte(sourceID + "\n" + agentName))
	recipe := AgentRecipe{
		ID:           "plugin-" + hex.EncodeToString(sum[:8]),
		Name:         found.Name,
		Instructions: found.Body,
		Mode:         "auto",
	}
	if recipe.Instructions == "" {
		recipe.Instructions = found.Description
	}
	if pluginName != "" {
		recipe.Instructions = "Plugin " + pluginName + ".\n\n" + recipe.Instructions
	}
	if err := writeAgentRecipe(recipe); err != nil {
		return AgentInstanceMeta{}, err
	}
	return createAgentInstance(recipe, workspaceID, found.Name)
}
