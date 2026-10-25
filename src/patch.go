package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
var hintHeader = regexp.MustCompile(`^@@ ~([0-9]+)(?: @@)?$`)
var bareHeader = regexp.MustCompile(`^@@$`)

const patchSearchRadius = 50

const (
	matchExact = iota + 1
	matchTrailingSpace
	matchIndent
)

type patchFile struct {
	path          string
	before, after []byte
}

type patchOp struct {
	kind byte
	data []byte
	// noNewline is set for the last line of a side whose hunk carried a
	// "\ No newline at end of file" marker.
	noNewline bool
}

func applyStrictPatch(src string) error {
	_, err := applyPatchText(src)
	return err
}

// applyPatchText applies a patch and returns notes for hunks that only matched
// after normalising whitespace, so callers can pass the hint back to the model.
func applyPatchText(src string) (string, error) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	lines := strings.Split(src, "\n")
	var changes []patchFile
	var names []string
	var notes []string
	for i := 0; i < len(lines); {
		if strings.TrimSpace(lines[i]) == "" {
			i++
			continue
		}
		if !strings.HasPrefix(lines[i], "--- ") || i+1 >= len(lines) || !strings.HasPrefix(lines[i+1], "+++ ") {
			return "", fmt.Errorf("line %d (%q): expected a \"--- a/path\" line followed by \"+++ b/path\"; one patch per file, then at least one @@ hunk", i+1, lines[i])
		}
		oldName := strings.TrimSpace(strings.TrimPrefix(lines[i], "--- "))
		newName := strings.TrimSpace(strings.TrimPrefix(lines[i+1], "+++ "))
		i += 2
		if !strings.HasPrefix(newName, "b/") {
			return "", fmt.Errorf("+++ path %q must start with b/", newName)
		}
		name := strings.TrimPrefix(newName, "b/")
		if oldName != "/dev/null" && oldName != "a/"+name {
			return "", fmt.Errorf("rename/delete unsupported: --- %s does not match +++ %s", oldName, newName)
		}
		if filepath.IsAbs(name) || name == "" || name == ".." || strings.HasPrefix(filepath.Clean(name), ".."+string(os.PathSeparator)) || privatePath(name) {
			return "", fmt.Errorf("unsafe path %q", name)
		}
		for _, n := range names {
			if n == name {
				return "", fmt.Errorf("duplicate patch target %q", name)
			}
		}
		names = append(names, name)
		var original []byte
		if oldName != "/dev/null" {
			var err error
			original, err = os.ReadFile(name)
			if err != nil {
				return "", fmt.Errorf("cannot patch %s: %w; check the path with ls before patching", name, err)
			}
			if len(original) > 8<<20 {
				return "", fmt.Errorf("file too large: %s", name)
			}
		}
		oldLines := fileLines(original)
		cursor := 0
		var result bytes.Buffer
		hunks := 0
		for i < len(lines) && strings.HasPrefix(lines[i], "@@") {
			header := strings.TrimSpace(lines[i])
			i++
			hunks++
			hint, radius := 1, patchSearchRadius
			zeroLength := false
			if m := hintHeader.FindStringSubmatch(header); m != nil {
				hint, _ = strconv.Atoi(m[1])
			} else if m := hunkHeader.FindStringSubmatch(header); m != nil {
				hint, _ = strconv.Atoi(m[1])
				zeroLength = m[2] == "0"
			} else if bareHeader.MatchString(header) {
				// Without a line hint, continue searching from the previous hunk.
				hint, radius = cursor+1, -1
			} else {
				return "", fmt.Errorf("malformed hunk header %q in %s: use \"@@ -12,3 +12,4 @@\", \"@@ ~12\" or \"@@\"", header, name)
			}
			if zeroLength {
				hint++
			}
			var ops []patchOp
			var oldPart [][]byte
			unprefixed := 0
			for i < len(lines) && lines[i] != "" {
				line := lines[i]
				if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "@@") {
					break
				}
				i++
				kind := line[0]
				if kind == '\\' { // "\ No newline at end of file": applies to the previous line
					if len(ops) > 0 {
						ops[len(ops)-1].noNewline = true
						if ops[len(ops)-1].kind != '+' && len(oldPart) > 0 {
							last := len(oldPart) - 1
							oldPart[last] = bytes.TrimSuffix(oldPart[last], []byte("\n"))
						}
					}
					continue
				}
				data := []byte(line[1:] + "\n")
				switch kind {
				case '+', '-', ' ':
				default:
					// Models often forget the leading space on context lines.
					kind, data, unprefixed = ' ', []byte(line+"\n"), unprefixed+1
				}
				ops = append(ops, patchOp{kind: kind, data: data})
				if kind != '+' {
					oldPart = append(oldPart, data)
				}
			}
			if unprefixed > 0 {
				notes = append(notes, fmt.Sprintf("%s: %d context line(s) were missing the leading space and were treated as context", name, unprefixed))
			}
			// Hunk counts are ignored; the actual context and removed lines decide.
			var best, tier int
			if len(oldPart) == 0 {
				// Pure insertion: place it at the hinted line, never before the previous hunk.
				best = hint - 1
				if best < cursor {
					best = cursor
				}
				if best > len(oldLines) {
					best = len(oldLines)
				}
				tier = matchExact
				added := 0
				for _, op := range ops {
					if op.kind == '+' {
						added++
					}
				}
				notes = append(notes, fmt.Sprintf("%s: inserted %d line(s) before line %d using an empty context; include unchanged context lines next time", name, added, best+1))
			} else if oldName == "/dev/null" {
				best, tier = 0, matchExact
			} else {
				var ties int
				best, ties, tier = locateHunk(oldLines, oldPart, cursor, hint, radius)
				if best < 0 {
					return "", patchContextError(name, hint, oldPart, oldLines, cursor)
				}
				if ties > 1 {
					return "", fmt.Errorf("ambiguous context near line %d in %s: %d identical matches; add more distinguishing context lines or a closer line hint", hint, name, ties)
				}
				switch tier {
				case matchTrailingSpace:
					notes = append(notes, fmt.Sprintf("%s: hunk near line %d matched only after ignoring trailing whitespace", name, best+1))
				case matchIndent:
					notes = append(notes, fmt.Sprintf("%s: hunk near line %d matched only after ignoring indentation; check the result", name, best+1))
				}
			}
			for _, line := range oldLines[cursor:best] {
				result.Write(line)
			}
			// Context lines are copied from the file so existing indentation and
			// trailing whitespace survive even a relaxed match.
			source := best
			for _, op := range ops {
				switch op.kind {
				case '+':
					result.Write(contextLine(op.data, op.noNewline))
				case ' ':
					if source < len(oldLines) {
						result.Write(contextLine(oldLines[source], op.noNewline))
					} else {
						result.Write(op.data)
					}
					source++
				default: // '-': drop the line without writing it
					source++
				}
			}
			cursor = best + len(oldPart)
		}
		if hunks == 0 {
			return "", fmt.Errorf("no @@ hunk after the headers for %s", name)
		}
		for _, line := range oldLines[cursor:] {
			result.Write(line)
		}
		changes = append(changes, patchFile{name, original, result.Bytes()})
	}
	if len(changes) == 0 {
		return "", fmt.Errorf("empty patch: no --- / +++ file sections found")
	}
	// Preflight every target before changing any file; reject symlinks and nonregular files.
	for _, c := range changes {
		if st, err := os.Lstat(c.path); err == nil {
			if !st.Mode().IsRegular() {
				return "", fmt.Errorf("nonregular target %s", c.path)
			}
		} else if !os.IsNotExist(err) {
			return "", err
		}
		now, err := os.ReadFile(c.path)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if !bytes.Equal(now, c.before) {
			return "", fmt.Errorf("concurrent modification %s", c.path)
		}
	}
	// A multi-file filesystem rename cannot be fully atomic. Roll back completed writes on error.
	applied := 0
	for _, c := range changes {
		if err := atomicWrite(c.path, c.after); err != nil {
			for j := applied - 1; j >= 0; j-- {
				if changes[j].before == nil {
					_ = os.Remove(changes[j].path)
				} else {
					_ = atomicWrite(changes[j].path, changes[j].before)
				}
			}
			return "", err
		}
		applied++
	}
	return strings.Join(notes, "\n"), nil
}

// locateHunk returns the best match position, the number of equally close
// matches, and the relaxation tier that produced the match. Exact matches win;
// whitespace-only differences are only accepted when nothing better exists.
func locateHunk(oldLines, oldPart [][]byte, cursor, hint, radius int) (int, int, int) {
	for tier := matchExact; tier <= matchIndent; tier++ {
		best, bestDistance, ties := -1, math.MaxInt, 0
		for pos := cursor; pos+len(oldPart) <= len(oldLines); pos++ {
			if radius >= 0 {
				distance := pos + 1 - hint
				if distance < 0 {
					distance = -distance
				}
				if distance > radius {
					continue
				}
				if !hunkMatches(oldLines, pos, oldPart, tier) {
					continue
				}
				if distance < bestDistance {
					best, bestDistance, ties = pos, distance, 1
				} else if distance == bestDistance {
					ties++
				}
				continue
			}
			if !hunkMatches(oldLines, pos, oldPart, tier) {
				continue
			}
			// Unconstrained search: the first match at or after the cursor wins.
			return pos, 1, tier
		}
		if best >= 0 {
			return best, ties, tier
		}
	}
	return -1, 0, 0
}

// contextLine keeps the file's own bytes for a context line, except that a
// missing final newline in the hunk is honoured.
func contextLine(line []byte, noNewline bool) []byte {
	if noNewline {
		return bytes.TrimSuffix(line, []byte("\n"))
	}
	return line
}

func hunkMatches(oldLines [][]byte, pos int, oldPart [][]byte, tier int) bool {
	for j, want := range oldPart {
		if !lineMatches(oldLines[pos+j], want, tier) {
			return false
		}
	}
	return true
}

func lineMatches(got, want []byte, tier int) bool {
	switch tier {
	case matchTrailingSpace:
		return bytes.Equal(bytes.TrimRight(got, " \t"), bytes.TrimRight(want, " \t"))
	case matchIndent:
		return bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want))
	default:
		return bytes.Equal(got, want)
	}
}

// patchContextError shows what the file actually contains so the next attempt can copy it verbatim.
func patchContextError(name string, hint int, oldPart, oldLines [][]byte, cursor int) error {
	var b strings.Builder
	fmt.Fprintf(&b, "no context match near line %d in %s (file has %d lines)", hint, name, len(oldLines))
	if len(oldPart) > 0 {
		fmt.Fprintf(&b, "; first expected line: %q", strings.TrimRight(string(oldPart[0]), "\n"))
	}
	start := hint - 1
	if start < cursor {
		start = cursor
	}
	if start > len(oldLines) {
		start = len(oldLines)
	}
	end := start + 6
	if end > len(oldLines) {
		end = len(oldLines)
	}
	if start < end {
		fmt.Fprintf(&b, "; actual file at %d..%d:", start+1, end)
		for j := start; j < end; j++ {
			fmt.Fprintf(&b, "\n%6d\t%s", j+1, strings.TrimRight(string(oldLines[j]), "\n"))
		}
	}
	b.WriteString("\nFix: read_file the range above, copy the context lines verbatim (including indentation), and retry with an accurate \"@@ ~LINE\" hint.")
	return fmt.Errorf("%s", b.String())
}
