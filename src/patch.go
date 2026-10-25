package main

import (
	"bytes"
	"fmt"
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

// gitPreamblePrefixes are lines a "git diff" adds before the first file
// section. They are skipped so a pasted git diff still applies.
var gitPreamblePrefixes = []string{
	"diff --git ", "index ", "new file mode ", "deleted file mode ",
	"old mode ", "new mode ", "similarity index ", "dissimilarity index ",
	"rename from ", "rename to ", "copy from ", "copy to ",
	"Binary files ", "GIT binary patch",
}

type patchFile struct {
	path          string
	before, after []byte
	create        bool
	delete        bool
	mode          os.FileMode // permissions to restore when a deletion is rolled back
}

type patchOp struct {
	kind      byte // ' ' context, '-' remove, '+' add
	text      string
	noNewline bool
}

type patchHunk struct {
	hint       int
	bare       bool
	hasCount   bool
	oldCount   int
	unprefixed int
	ops        []patchOp
}

type patchSection struct {
	oldName, newName string
	path             string
	create, delete   bool
	hunks            []patchHunk
	notes            []string
}

// patchTarget is one file a patch would touch, with its create/delete intent.
type patchTarget struct {
	path           string
	create, delete bool
}

func applyStrictPatch(src string) error {
	_, err := applyPatchText(src)
	return err
}

// patchTargets returns the files a patch would touch without reading or
// writing anything. It is used for previews and approval checks; callers fall
// back to a generic label when parsing fails.
func patchTargets(src string) ([]patchTarget, error) {
	if strings.TrimSpace(src) == "" {
		return nil, nil
	}
	sections, err := parsePatch(src)
	if err != nil {
		return nil, err
	}
	targets := make([]patchTarget, 0, len(sections))
	for _, sec := range sections {
		targets = append(targets, patchTarget{path: sec.path, create: sec.create, delete: sec.delete})
	}
	return targets, nil
}

// applyPatchText applies a patch and returns notes for hunks that only matched
// after normalising whitespace, so callers can pass the hint back to the model.
func applyPatchText(src string) (string, error) {
	sections, err := parsePatch(src)
	if err != nil {
		return "", err
	}
	var changes []patchFile
	var notes []string
	seen := make(map[string]bool, len(sections))
	for _, sec := range sections {
		if seen[sec.path] {
			return "", fmt.Errorf("duplicate patch target %q", sec.path)
		}
		seen[sec.path] = true
		notes = append(notes, sec.notes...)
		change := patchFile{path: sec.path, create: sec.create, delete: sec.delete}
		if sec.create {
			var b strings.Builder
			for _, h := range sec.hunks {
				for _, op := range h.ops {
					b.WriteString(op.text)
					if !op.noNewline {
						b.WriteString("\n")
					}
				}
			}
			change.after = []byte(b.String())
			changes = append(changes, change)
			continue
		}
		if st, err := os.Lstat(sec.path); err == nil && !st.Mode().IsRegular() {
			return "", fmt.Errorf("nonregular target %s", sec.path)
		}
		original, err := os.ReadFile(sec.path)
		if err != nil {
			return "", fmt.Errorf("cannot patch %s: %w; check the path with ls before patching", sec.path, err)
		}
		if len(original) > 8<<20 {
			return "", fmt.Errorf("file too large: %s", sec.path)
		}
		change.before = original
		oldLines := fileLines(original)
		lineEnding := detectLineEnding(original)
		cursor := 0
		var result bytes.Buffer
		for _, h := range sec.hunks {
			var oldPart [][]byte
			added := 0
			for _, op := range h.ops {
				if op.kind == '+' {
					added++
					continue
				}
				line := []byte(op.text)
				if !op.noNewline {
					line = append(line, '\n')
				}
				oldPart = append(oldPart, line)
			}
			hint, radius := h.hint, patchSearchRadius
			if h.bare {
				hint, radius = cursor+1, -1
			}
			var best, tier int
			switch {
			case len(oldPart) == 0 && sec.delete:
				return "", fmt.Errorf("deletion hunk in %s has no context or removed lines", sec.path)
			case len(oldPart) == 0:
				best = hint - 1
				if best < cursor {
					best = cursor
				}
				if best > len(oldLines) {
					best = len(oldLines)
				}
				tier = matchExact
				notes = append(notes, fmt.Sprintf("%s: inserted %d line(s) before line %d using an empty context; include unchanged context lines next time", sec.path, added, best+1))
			case sec.delete:
				// A deletion must quote existing lines exactly; a relaxed match is unsafe.
				p, ties := locateExactHunk(oldLines, oldPart, cursor, hint)
				if p < 0 {
					return "", patchContextError(sec.path, hint, oldPart, oldLines, cursor)
				}
				if ties > 1 {
					return "", fmt.Errorf("ambiguous deletion near line %d in %s: %d equally close matches; quote more lines or use a closer hint", hint, sec.path, ties)
				}
				cursor = p + len(oldPart)
				continue
			default:
				p, ties, t, matches := locateHunk(oldLines, oldPart, cursor, hint, radius)
				if p < 0 {
					return "", patchContextError(sec.path, hint, oldPart, oldLines, cursor)
				}
				if ties > 1 {
					return "", fmt.Errorf("ambiguous context near line %d in %s: %d equally close matches; add more distinguishing context lines or a closer line hint", hint, sec.path, ties)
				}
				best, tier = p, t
				if matches > 1 {
					notes = append(notes, fmt.Sprintf("%s: hunk near line %d matched %d places; patched the nearest one", sec.path, hint, matches))
				}
				switch tier {
				case matchTrailingSpace:
					notes = append(notes, fmt.Sprintf("%s: hunk near line %d matched only after ignoring trailing whitespace", sec.path, best+1))
				case matchIndent:
					notes = append(notes, fmt.Sprintf("%s: hunk near line %d matched only after ignoring indentation; check the result", sec.path, best+1))
				}
			}
			for _, line := range oldLines[cursor:best] {
				result.Write(line)
			}
			// Context lines are copied from the file so existing indentation and
			// trailing whitespace survive even a relaxed match.
			source := best
			for _, op := range h.ops {
				switch op.kind {
				case '+':
					result.WriteString(op.text)
					if !op.noNewline {
						result.WriteString(lineEnding)
					}
				case ' ':
					if source < len(oldLines) {
						result.Write(contextLine(oldLines[source], op.noNewline))
					} else {
						result.WriteString(op.text)
						if !op.noNewline {
							result.WriteString(lineEnding)
						}
					}
					source++
				default: // '-': drop the line without writing it
					source++
				}
			}
			cursor = best + len(oldPart)
		}
		if !sec.delete {
			for _, line := range oldLines[cursor:] {
				result.Write(line)
			}
			change.after = result.Bytes()
		}
		changes = append(changes, change)
	}
	// Preflight every target before changing any file; reject symlinks and nonregular files.
	for _, c := range changes {
		st, statErr := os.Lstat(c.path)
		exists := statErr == nil
		if statErr != nil && !os.IsNotExist(statErr) {
			return "", statErr
		}
		if exists && !st.Mode().IsRegular() {
			return "", fmt.Errorf("nonregular target %s", c.path)
		}
		if c.create {
			if exists {
				return "", fmt.Errorf("cannot create %s: file already exists; use \"--- a/%s\" to modify it instead", c.path, c.path)
			}
			continue
		}
		if !exists {
			return "", fmt.Errorf("cannot patch %s: file does not exist; check the path with ls before patching", c.path)
		}
		c.mode = st.Mode().Perm()
		now, err := os.ReadFile(c.path)
		if err != nil {
			return "", err
		}
		if !bytes.Equal(now, c.before) {
			return "", fmt.Errorf("concurrent modification: %s changed while the patch was prepared; read_file it again", c.path)
		}
	}
	// A multi-file filesystem rename cannot be fully atomic. Roll back completed writes on error.
	applied := 0
	for _, c := range changes {
		var err error
		if c.delete {
			err = os.Remove(c.path)
		} else {
			err = atomicWrite(c.path, c.after)
		}
		if err != nil {
			for j := applied - 1; j >= 0; j-- {
				done := &changes[j]
				if done.create {
					_ = os.Remove(done.path)
					continue
				}
				if err := atomicWrite(done.path, done.before); err != nil {
					continue
				}
				if done.delete {
					_ = os.Chmod(done.path, done.mode)
				}
			}
			return "", err
		}
		applied++
	}
	return strings.Join(notes, "\n"), nil
}

// parsePatch turns patch text into file sections. It never reads or writes
// files; applyPatchText and patchTargets both build on it.
func parsePatch(src string) ([]patchSection, error) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	lines := strings.Split(src, "\n")
	var sections []patchSection
	for i := 0; i < len(lines); {
		line := lines[i]
		if strings.TrimSpace(line) == "" || isGitPreamble(line) {
			i++
			continue
		}
		if !strictSectionStart(lines, i) {
			return nil, fmt.Errorf("line %d (%q): expected a file section: %q immediately followed by %q; git diff headers are ignored and each section needs at least one hunk", i+1, line, "--- a/path", "+++ b/path")
		}
		sec, next, err := parseSection(lines, i)
		if err != nil {
			return nil, err
		}
		sections = append(sections, sec)
		i = next
	}
	if len(sections) == 0 {
		return nil, fmt.Errorf("empty patch: no --- / +++ file sections found")
	}
	return sections, nil
}

func fileSectionStart(lines []string, i int) bool {
	return i+1 < len(lines) && strings.HasPrefix(lines[i], "--- ") && strings.HasPrefix(lines[i+1], "+++ ")
}

// strictSectionStart additionally requires git-style paths, so a removed line
// "-- x" followed by an added line "++ x" is not mistaken for a new section.
func strictSectionStart(lines []string, i int) bool {
	if !fileSectionStart(lines, i) {
		return false
	}
	oldName := strings.TrimSpace(strings.TrimPrefix(lines[i], "--- "))
	newName := strings.TrimSpace(strings.TrimPrefix(lines[i+1], "+++ "))
	oldOK := oldName == "/dev/null" || strings.HasPrefix(oldName, "a/")
	newOK := newName == "/dev/null" || strings.HasPrefix(newName, "b/")
	return oldOK && newOK
}

func isGitPreamble(line string) bool {
	for _, p := range gitPreamblePrefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}

func parseSection(lines []string, i int) (patchSection, int, error) {
	var sec patchSection
	sec.oldName = strings.TrimSpace(strings.TrimPrefix(lines[i], "--- "))
	sec.newName = strings.TrimSpace(strings.TrimPrefix(lines[i+1], "+++ "))
	i += 2
	switch {
	case sec.oldName == "/dev/null" && sec.newName == "/dev/null":
		return sec, i, fmt.Errorf("invalid section: both --- and +++ are /dev/null")
	case sec.oldName == "/dev/null":
		sec.create = true
		if !strings.HasPrefix(sec.newName, "b/") {
			return sec, i, fmt.Errorf("+++ path %q must start with b/ (use \"+++ b/path\" or \"+++ /dev/null\")", sec.newName)
		}
		sec.path = strings.TrimPrefix(sec.newName, "b/")
	case sec.newName == "/dev/null":
		sec.delete = true
		if !strings.HasPrefix(sec.oldName, "a/") {
			return sec, i, fmt.Errorf("--- path %q must start with a/ when deleting (use \"--- a/path\")", sec.oldName)
		}
		sec.path = strings.TrimPrefix(sec.oldName, "a/")
	default:
		if !strings.HasPrefix(sec.newName, "b/") {
			return sec, i, fmt.Errorf("+++ path %q must start with b/", sec.newName)
		}
		sec.path = strings.TrimPrefix(sec.newName, "b/")
		if sec.oldName != "a/"+sec.path {
			return sec, i, fmt.Errorf("rename unsupported: --- %s does not match +++ %s", sec.oldName, sec.newName)
		}
	}
	if err := checkPatchPath(sec.path); err != nil {
		return sec, i, err
	}
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "" {
			i++
			continue
		}
		if fileSectionStart(lines, i) {
			break
		}
		if !strings.HasPrefix(lines[i], "@@") {
			return sec, i, fmt.Errorf("line %d (%q) in %s: expected a hunk header (%q, %q or %q) or a new file section", i+1, lines[i], sec.path, "@@ -12,3 +12,4 @@", "@@ ~12", "@@")
		}
		hunk, next, err := parseHunk(lines, i, sec.path)
		if err != nil {
			return sec, next, err
		}
		if hunk.unprefixed > 0 {
			sec.notes = append(sec.notes, fmt.Sprintf("%s: %d context line(s) were missing the leading space and were treated as context", sec.path, hunk.unprefixed))
		}
		sec.hunks = append(sec.hunks, hunk)
		i = next
	}
	if len(sec.hunks) == 0 {
		return sec, i, fmt.Errorf("no @@ hunk after the headers for %s", sec.path)
	}
	if sec.create {
		for _, h := range sec.hunks {
			for _, op := range h.ops {
				if op.kind != '+' {
					return sec, i, fmt.Errorf("new file creation for %s cannot include context or removed lines", sec.path)
				}
			}
		}
	}
	return sec, i, nil
}

// parseHunk reads one hunk body. Counts in @@ headers only disambiguate empty
// context lines; they are not validated strictly so legacy or approximate
// counts keep working. A "--- x" line immediately followed by "+++ y" always
// starts a new file section, even inside a hunk.
func parseHunk(lines []string, i int, path string) (patchHunk, int, error) {
	header := strings.TrimSpace(lines[i])
	i++
	h := patchHunk{hint: 1}
	switch {
	case hintHeader.MatchString(header):
		m := hintHeader.FindStringSubmatch(header)
		h.hint, _ = strconv.Atoi(m[1])
	case hunkHeader.MatchString(header):
		m := hunkHeader.FindStringSubmatch(header)
		h.hint, _ = strconv.Atoi(m[1])
		if m[2] != "" {
			h.hasCount = true
			h.oldCount, _ = strconv.Atoi(m[2])
		}
		if h.hint == 0 {
			h.hint = 1
		}
	case bareHeader.MatchString(header):
		h.bare = true
	default:
		return h, i, fmt.Errorf("malformed hunk header %q in %s: use \"@@ -12,3 +12,4 @@\", \"@@ ~12\" or \"@@\"", header, path)
	}
	oldConsumed := 0
	for i < len(lines) {
		line := lines[i]
		if strings.HasPrefix(line, "@@") {
			break
		}
		if strictSectionStart(lines, i) {
			break
		}
		if line == "" {
			if i == len(lines)-1 {
				// strings.Split leaves one empty string after the final newline.
				i++
				break
			}
			if h.hasCount && oldConsumed < h.oldCount {
				h.ops = append(h.ops, patchOp{kind: ' ', text: ""})
				oldConsumed++
				i++
				continue
			}
			// A raw empty line ends the hunk unless real content follows.
			j := i + 1
			for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
				j++
			}
			if j >= len(lines) || strings.HasPrefix(lines[j], "@@") || strictSectionStart(lines, j) {
				i = j
				break
			}
			h.ops = append(h.ops, patchOp{kind: ' ', text: ""})
			oldConsumed++
			i++
			continue
		}
		if line[0] == '\\' { // "\ No newline at end of file": applies to the previous line
			if len(h.ops) > 0 {
				h.ops[len(h.ops)-1].noNewline = true
			}
			i++
			continue
		}
		kind := line[0]
		text := ""
		switch kind {
		case '+', '-', ' ':
			text = line[1:]
		default:
			// Models often forget the leading space on context lines.
			kind = ' '
			text = line
			h.unprefixed++
		}
		h.ops = append(h.ops, patchOp{kind: kind, text: text})
		if kind != '+' {
			oldConsumed++
		}
		i++
	}
	if len(h.ops) == 0 {
		return h, i, fmt.Errorf("hunk %q in %s has no content lines", header, path)
	}
	return h, i, nil
}

func checkPatchPath(name string) error {
	if name == "" {
		return fmt.Errorf("empty patch path")
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("unsafe path %q: use a path relative to the working directory", name)
	}
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("unsafe path %q: escapes the working directory", name)
	}
	if privatePath(name) {
		return fmt.Errorf("unsafe path %q: protected path", name)
	}
	return nil
}

// locateHunk returns the best match position, the number of equally close
// matches at the same tier (ambiguity), the relaxation tier and the total
// number of matches found. The nearest candidate wins; a better tier only
// breaks an equal-distance tie. radius < 0 searches from the cursor and takes
// the first match.
func locateHunk(oldLines, oldPart [][]byte, cursor, hint, radius int) (pos, ties, tier, matches int) {
	if len(oldPart) == 0 {
		return -1, 0, 0, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if radius < 0 {
		for p := cursor; p+len(oldPart) <= len(oldLines); p++ {
			if t := hunkTier(oldLines, p, oldPart); t > 0 {
				return p, 1, t, 1
			}
		}
		return -1, 0, 0, 0
	}
	bestPos, bestDistance, bestTier := -1, 0, 0
	for p := cursor; p+len(oldPart) <= len(oldLines); p++ {
		t := hunkTier(oldLines, p, oldPart)
		if t == 0 {
			continue
		}
		d := p + 1 - hint
		if d < 0 {
			d = -d
		}
		if d > radius {
			continue
		}
		matches++
		if bestPos < 0 || d < bestDistance || (d == bestDistance && t < bestTier) {
			bestPos, bestDistance, bestTier = p, d, t
		}
	}
	if bestPos < 0 {
		return -1, 0, 0, matches
	}
	for p := cursor; p+len(oldPart) <= len(oldLines); p++ {
		d := p + 1 - hint
		if d < 0 {
			d = -d
		}
		if d != bestDistance {
			continue
		}
		if hunkTier(oldLines, p, oldPart) == bestTier {
			ties++
		}
	}
	return bestPos, ties, bestTier, matches
}

// locateExactHunk finds the nearest exact match only; relaxed matching is never
// acceptable for a deletion.
func locateExactHunk(oldLines, oldPart [][]byte, cursor, hint int) (int, int) {
	best, bestDistance, ties := -1, 0, 0
	for p := cursor; p+len(oldPart) <= len(oldLines); p++ {
		if hunkTier(oldLines, p, oldPart) != matchExact {
			continue
		}
		d := p + 1 - hint
		if d < 0 {
			d = -d
		}
		switch {
		case best < 0 || d < bestDistance:
			best, bestDistance, ties = p, d, 1
		case d == bestDistance:
			ties++
		}
	}
	return best, ties
}

func hunkTier(oldLines [][]byte, pos int, oldPart [][]byte) int {
	if pos < 0 || pos+len(oldPart) > len(oldLines) {
		return 0
	}
	for tier := matchExact; tier <= matchIndent; tier++ {
		ok := true
		for j, want := range oldPart {
			if !lineMatches(oldLines[pos+j], want, tier) {
				ok = false
				break
			}
		}
		if ok {
			return tier
		}
	}
	return 0
}

// contextLine keeps the file's own bytes for a context line, except that a
// missing final newline in the hunk is honoured.
func contextLine(line []byte, noNewline bool) []byte {
	if !noNewline {
		return line
	}
	if bytes.HasSuffix(line, []byte("\r\n")) {
		return line[:len(line)-2]
	}
	return bytes.TrimSuffix(line, []byte("\n"))
}

// detectLineEnding reports "\r\n" only when every non-empty line of the file
// uses CRLF; mixed or LF files keep "\n" for newly added lines.
func detectLineEnding(b []byte) string {
	seen := false
	for _, line := range fileLines(b) {
		if len(bytes.TrimRight(line, "\r\n")) == 0 {
			continue
		}
		if !bytes.HasSuffix(line, []byte("\r\n")) {
			return "\n"
		}
		seen = true
	}
	if seen {
		return "\r\n"
	}
	return "\n"
}

func lineMatches(got, want []byte, tier int) bool {
	got, want = trimCRLF(got), trimCRLF(want)
	switch tier {
	case matchTrailingSpace:
		return bytes.Equal(bytes.TrimRight(got, " \t"), bytes.TrimRight(want, " \t"))
	case matchIndent:
		return bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want))
	default:
		return bytes.Equal(got, want)
	}
}

// trimCRLF rewrites a trailing "\r\n" to "\n" so CRLF files compare equal to
// LF patch text without allocating in the common case.
func trimCRLF(b []byte) []byte {
	if !bytes.HasSuffix(b, []byte("\r\n")) {
		return b
	}
	out := make([]byte, len(b)-1)
	copy(out, b[:len(b)-2])
	out[len(out)-1] = '\n'
	return out
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
	// Point at the closest partial match so a wrong hint or a near-miss is obvious.
	if partial, matched := closestPartialMatch(oldLines, oldPart, cursor, hint); partial >= 0 {
		fmt.Fprintf(&b, "; closest match starts at line %d (%d of %d context lines agree)", partial+1, matched, len(oldPart))
		if matched*2 >= len(oldPart) {
			start = partial
		}
	}
	if start > len(oldLines) {
		start = len(oldLines)
	}
	if hint > len(oldLines) && len(oldLines) > 0 {
		fmt.Fprintf(&b, "; the hint is past the end of the file, whose last line is %d", len(oldLines))
	}
	end := start + 8
	if end > len(oldLines) {
		end = len(oldLines)
	}
	if start < end {
		fmt.Fprintf(&b, "; actual file at %d..%d:", start+1, end)
		for j := start; j < end; j++ {
			line := strings.TrimSuffix(strings.TrimRight(string(oldLines[j]), "\n"), "\r")
			fmt.Fprintf(&b, "\n%6d\t%s", j+1, line)
		}
	}
	b.WriteString("\nFix: read_file the range above, copy the context lines verbatim (including indentation), and retry with an accurate \"@@ ~LINE\" hint. A bare \"@@\" searches forward from the previous hunk, so a hunk far from that point fails.")
	return fmt.Errorf("%s", b.String())
}

// closestPartialMatch returns the position whose leading lines agree best with
// oldPart, preferring positions near the hint and never before the cursor.
func closestPartialMatch(oldLines, oldPart [][]byte, cursor, hint int) (int, int) {
	best, bestMatched, bestDistance := -1, 0, 0
	for pos := cursor; pos < len(oldLines); pos++ {
		matched := 0
		for matched < len(oldPart) && pos+matched < len(oldLines) {
			if !lineMatches(oldLines[pos+matched], oldPart[matched], matchExact) {
				break
			}
			matched++
		}
		if matched == 0 {
			continue
		}
		d := pos + 1 - hint
		if d < 0 {
			d = -d
		}
		if matched > bestMatched || (matched == bestMatched && d < bestDistance) {
			best, bestMatched, bestDistance = pos, matched, d
		}
	}
	return best, bestMatched
}
