package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// A continuation points at journal lines and charter revisions. It does not
// copy the transcript or keep every branch's memory in RAM.
type branchState struct {
	Active     string                `json:"active,omitempty"`
	Next       int                   `json:"next,omitempty"`
	CharterSeq int                   `json:"charter_seq,omitempty"`
	Marks      map[string]userMark   `json:"marks,omitempty"`
	Forks      map[string]*forkGroup `json:"forks,omitempty"`
	Tips       map[string]string     `json:"tips,omitempty"`
}

type userMark struct {
	CharterBefore string `json:"charter_before,omitempty"`
	Checkpoint    int    `json:"checkpoint,omitempty"`
	After         []int  `json:"after,omitempty"`
}

type forkGroup struct {
	Source   int      `json:"source"`
	IDs      []string `json:"ids"`
	UserLine []int    `json:"user_line"`
	Active   int      `json:"active"`
}

type tailSnap struct {
	Checkpoint int
	After      []int
}

type journalRow struct {
	Line   int
	Type   string
	Branch string
	Item   json.RawMessage
}

var (
	errEmptyBranchText = errors.New("write a message first")
	errNotUserMessage  = errors.New("only a message you sent can start a continuation")
	errNoContinuation  = errors.New("that continuation is gone")
)

func branchIndexPath(session string) string { return session + ".branches.json" }
func charterLogPath(session string) string  { return session + ".charters.jsonl" }

func (st *branchState) init() {
	if st.Marks == nil {
		st.Marks = map[string]userMark{}
	}
	if st.Forks == nil {
		st.Forks = map[string]*forkGroup{}
	}
	if st.Tips == nil {
		st.Tips = map[string]string{}
	}
}

func lineKey(line int) string { return strconv.Itoa(line) }

func copyInts(in []int) []int {
	if len(in) == 0 {
		return nil
	}
	out := make([]int, len(in))
	copy(out, in)
	return out
}

func loadBranchState(session string) (*branchState, error) {
	b, err := os.ReadFile(branchIndexPath(session))
	if err != nil {
		if os.IsNotExist(err) {
			return &branchState{}, nil
		}
		return nil, err
	}
	st := &branchState{}
	if len(bytes.TrimSpace(b)) == 0 {
		return st, nil
	}
	if err := json.Unmarshal(b, st); err != nil {
		return nil, err
	}
	return st, nil
}

func saveBranchState(session string, st *branchState) error {
	if st == nil {
		st = &branchState{}
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(branchIndexPath(session), b)
}

func writeFileAtomic(path string, body []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readJournalRows(path string) ([]journalRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	var rows []journalRow
	n := 0
	for {
		b, err := r.ReadBytes('\n')
		if len(b) > 0 {
			n++
			trim := bytes.TrimSpace(b)
			if len(trim) > 0 {
				var rec struct {
					Type   string          `json:"type"`
					Branch string          `json:"branch"`
					Item   json.RawMessage `json:"item"`
				}
				if json.Unmarshal(trim, &rec) == nil {
					rows = append(rows, journalRow{Line: n, Type: rec.Type, Branch: rec.Branch, Item: rec.Item})
				}
			}
		}
		if err == io.EOF {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

func rowByLine(rows []journalRow, line int) (journalRow, bool) {
	for _, ln := range rows {
		if ln.Line == line {
			return ln, true
		}
	}
	return journalRow{}, false
}

func lastJournalLine(path string) (int, error) {
	rows, err := readJournalRows(path)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[len(rows)-1].Line, nil
}

func journalHasCheckpoint(rows []journalRow, tag string) bool {
	for _, ln := range rows {
		if ln.Branch == tag && ln.Type == "checkpoint" {
			return true
		}
	}
	return false
}

func fileTail(rows []journalRow, tag string) tailSnap {
	var snap tailSnap
	seen := false
	for _, ln := range rows {
		if ln.Branch != tag {
			continue
		}
		if ln.Type == "checkpoint" {
			snap.Checkpoint = ln.Line
			snap.After = nil
			seen = true
			continue
		}
		if ln.Type == "item" && seen {
			snap.After = append(snap.After, ln.Line)
		}
	}
	if seen {
		return snap
	}
	for _, ln := range rows {
		if ln.Branch == tag && ln.Type == "item" {
			snap.After = append(snap.After, ln.Line)
		}
	}
	return snap
}

func loadTail(rows []journalRow, snap tailSnap) []json.RawMessage {
	by := map[int]journalRow{}
	for _, ln := range rows {
		by[ln.Line] = ln
	}
	var out []json.RawMessage
	if snap.Checkpoint > 0 {
		if ln, ok := by[snap.Checkpoint]; ok {
			var items []json.RawMessage
			if json.Unmarshal(ln.Item, &items) == nil {
				out = append(out, items...)
			}
		}
	}
	for _, n := range snap.After {
		ln, ok := by[n]
		if !ok || ln.Type != "item" || len(ln.Item) == 0 {
			continue
		}
		out = append(out, ln.Item)
	}
	return out
}

func legacySnap(rows []journalRow, source int, tag string) tailSnap {
	var snap tailSnap
	for _, ln := range rows {
		if ln.Line >= source {
			break
		}
		if ln.Type == "item" && ln.Branch == tag {
			snap.After = append(snap.After, ln.Line)
		}
	}
	return snap
}

func (st *branchState) originLine(tag string) int {
	if st == nil || tag == "" {
		return 0
	}
	for _, g := range st.Forks {
		if g == nil {
			continue
		}
		for i, id := range g.IDs {
			if i > 0 && id == tag && i < len(g.UserLine) && g.UserLine[i] > 0 {
				return g.UserLine[i]
			}
		}
	}
	return 0
}

func modelItems(rows []journalRow, st *branchState, tag string) []json.RawMessage {
	if tag == "" || journalHasCheckpoint(rows, tag) {
		return loadTail(rows, fileTail(rows, tag))
	}
	origin := 0
	if st != nil {
		origin = st.originLine(tag)
	}
	var mark userMark
	ok := false
	if st != nil && origin > 0 {
		mark, ok = st.Marks[lineKey(origin)]
	}
	if !ok {
		return loadTail(rows, fileTail(rows, tag))
	}
	out := loadTail(rows, tailSnap{Checkpoint: mark.Checkpoint, After: mark.After})
	for _, ln := range rows {
		if ln.Type == "item" && ln.Branch == tag && len(ln.Item) > 0 {
			out = append(out, ln.Item)
		}
	}
	return out
}

func walkVisible(rows []journalRow, st *branchState, tag string, items *[]json.RawMessage, lines *[]int) {
	for _, ln := range rows {
		if ln.Type != "item" || ln.Branch != tag || len(ln.Item) == 0 {
			continue
		}
		if st != nil && st.Forks != nil {
			if g := st.Forks[lineKey(ln.Line)]; g != nil && g.Active > 0 && g.Active < len(g.IDs) {
				walkVisible(rows, st, g.IDs[g.Active], items, lines)
				return
			}
		}
		*items = append(*items, ln.Item)
		*lines = append(*lines, ln.Line)
	}
}

func visibleJournal(path string) ([]json.RawMessage, []int, error) {
	rows, err := readJournalRows(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	st, err := loadBranchState(path)
	if err != nil {
		return nil, nil, err
	}
	var items []json.RawMessage
	var lines []int
	walkVisible(rows, st, "", &items, &lines)
	return items, lines, nil
}

func (st *branchState) place(line int) (source, index, count int) {
	if st == nil {
		return line, 0, 1
	}
	if g := st.Forks[lineKey(line)]; g != nil && len(g.IDs) > 1 && line == g.Source {
		idx := g.Active
		if idx < 0 || idx >= len(g.IDs) {
			idx = 0
		}
		return g.Source, idx, len(g.IDs)
	}
	for _, g := range st.Forks {
		if g == nil {
			continue
		}
		for i, ul := range g.UserLine {
			if ul == line && len(g.IDs) > 1 {
				return g.Source, i, len(g.IDs)
			}
		}
	}
	return line, 0, 1
}

func branchDiverged(path string) bool {
	st, err := loadBranchState(path)
	if err != nil || st == nil {
		return false
	}
	if st.Active != "" {
		return true
	}
	for _, g := range st.Forks {
		if g != nil && g.Active > 0 {
			return true
		}
	}
	return false
}

func sessionChaptersFor(path string) []sessionChapter {
	if branchDiverged(path) {
		return nil
	}
	return sessionChapters(path)
}

func annotateBranchMessages(path string, msgs []sessionMessage, lines []int) {
	if len(lines) == 0 || len(msgs) == 0 {
		return
	}
	st, err := loadBranchState(path)
	if err != nil {
		return
	}
	for i := range msgs {
		if msgs[i].Role != "user" || msgs[i].Event <= 0 || msgs[i].Event > len(lines) {
			continue
		}
		line := lines[msgs[i].Event-1]
		if line <= 0 {
			continue
		}
		source, index, count := st.place(line)
		msgs[i].Line = line
		msgs[i].BranchLine = source
		msgs[i].BranchIndex = index
		msgs[i].BranchCount = count
	}
}

func editableUserText(item json.RawMessage) (string, bool) {
	var m struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}
	if json.Unmarshal(item, &m) != nil || m.Role != "user" {
		return "", false
	}
	s, ok := m.Content.(string)
	if !ok {
		return "", false
	}
	if _, watch := watchDisplayText(s); watch {
		return "", false
	}
	if _, _, peer := parsePeerUserText(s); peer {
		return "", false
	}
	t := strings.TrimSpace(s)
	if t == "" ||
		strings.HasPrefix(t, "[AUTOPILOT") ||
		strings.HasPrefix(t, "Continue the current plan") ||
		strings.HasPrefix(t, "<compacted_conversation_memory>") ||
		strings.HasPrefix(t, "[User steering") {
		return "", false
	}
	return s, true
}

func (a *Agent) branchTag() string {
	if a == nil || a.Journal == nil {
		return ""
	}
	return a.Journal.Branch
}

func (a *Agent) discardBranchFiles() {
	if a == nil || a.Cfg.SessionPath == "" {
		return
	}
	_ = os.Remove(branchIndexPath(a.Cfg.SessionPath))
	_ = os.Remove(charterLogPath(a.Cfg.SessionPath))
}

func (a *Agent) resetBranchMemory() {
	if a == nil {
		return
	}
	if a.Journal != nil {
		a.Journal.Branch = ""
	}
	a.branchHold = nil
	a.branchPending = 0
	a.branchUserReady = false
}

func findCharterRevision(session string, body []byte) string {
	f, err := os.Open(charterLogPath(session))
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 4096), 2<<20)
	for sc.Scan() {
		var rec struct {
			ID   string          `json:"id"`
			Body json.RawMessage `json:"body"`
		}
		if json.Unmarshal(sc.Bytes(), &rec) != nil || rec.ID == "" {
			continue
		}
		if bytes.Equal(rec.Body, body) {
			return rec.ID
		}
	}
	return ""
}

func appendCharterRevision(session, id string, body []byte) error {
	rec, err := json.Marshal(struct {
		ID   string          `json:"id"`
		Body json.RawMessage `json:"body"`
	}{ID: id, Body: body})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(charterLogPath(session), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintln(f, string(rec)); err != nil {
		return err
	}
	return f.Sync()
}

func readCharterRevision(session, id string) ([]byte, error) {
	f, err := os.Open(charterLogPath(session))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 4096), 2<<20)
	for sc.Scan() {
		var rec struct {
			ID   string          `json:"id"`
			Body json.RawMessage `json:"body"`
		}
		if json.Unmarshal(sc.Bytes(), &rec) != nil {
			continue
		}
		if rec.ID == id {
			return rec.Body, nil
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("unknown charter revision %s", id)
}

func (a *Agent) charterRevisionID() (string, error) {
	if err := saveCharter(a.Cfg.SessionPath, a.ensureCharter()); err != nil {
		return "", err
	}
	body, err := os.ReadFile(charterPath(a.Cfg.SessionPath))
	if err != nil {
		return "", err
	}
	if id := findCharterRevision(a.Cfg.SessionPath, body); id != "" {
		return id, nil
	}
	st, err := loadBranchState(a.Cfg.SessionPath)
	if err != nil {
		return "", err
	}
	st.init()
	st.CharterSeq++
	id := fmt.Sprintf("c%d", st.CharterSeq)
	if err := appendCharterRevision(a.Cfg.SessionPath, id, body); err != nil {
		return "", err
	}
	if err := saveBranchState(a.Cfg.SessionPath, st); err != nil {
		return "", err
	}
	return id, nil
}

func (a *Agent) recordCharterTip() error {
	id, err := a.charterRevisionID()
	if err != nil {
		return err
	}
	st, err := loadBranchState(a.Cfg.SessionPath)
	if err != nil {
		return err
	}
	st.init()
	tag := a.branchTag()
	if st.Tips[tag] == id {
		return nil
	}
	st.Tips[tag] = id
	return saveBranchState(a.Cfg.SessionPath, st)
}

func (a *Agent) applyCharterRevision(id string) error {
	if id == "" {
		a.Charter = &Charter{Version: charterVersion}
		return saveCharter(a.Cfg.SessionPath, a.Charter)
	}
	body, err := readCharterRevision(a.Cfg.SessionPath, id)
	if err != nil {
		return err
	}
	var c Charter
	if err := json.Unmarshal(body, &c); err != nil {
		return err
	}
	a.Charter = &c
	return writeFileAtomic(charterPath(a.Cfg.SessionPath), body)
}

func (a *Agent) ensureTip(tag string) error {
	st, err := loadBranchState(a.Cfg.SessionPath)
	if err != nil {
		return err
	}
	st.init()
	if _, ok := st.Tips[tag]; ok {
		return nil
	}
	id, err := a.charterRevisionID()
	if err != nil {
		return err
	}
	st, err = loadBranchState(a.Cfg.SessionPath)
	if err != nil {
		return err
	}
	st.init()
	if _, ok := st.Tips[tag]; ok {
		return nil
	}
	st.Tips[tag] = id
	return saveBranchState(a.Cfg.SessionPath, st)
}

func (a *Agent) captureTail() (tailSnap, error) {
	rows, err := readJournalRows(a.Cfg.SessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return tailSnap{}, nil
		}
		return tailSnap{}, err
	}
	tag := a.branchTag()
	if tag == "" || journalHasCheckpoint(rows, tag) {
		return fileTail(rows, tag), nil
	}
	st, err := loadBranchState(a.Cfg.SessionPath)
	if err != nil {
		return tailSnap{}, err
	}
	origin := st.originLine(tag)
	mark, ok := st.Marks[lineKey(origin)]
	if !ok {
		return fileTail(rows, tag), nil
	}
	snap := tailSnap{Checkpoint: mark.Checkpoint, After: copyInts(mark.After)}
	for _, ln := range rows {
		if ln.Type == "item" && ln.Branch == tag {
			snap.After = append(snap.After, ln.Line)
		}
	}
	return snap, nil
}

func (a *Agent) takeTail() (tailSnap, error) {
	if a.branchHold != nil {
		s := *a.branchHold
		s.After = copyInts(s.After)
		a.branchHold = nil
		return s, nil
	}
	return a.captureTail()
}

func (a *Agent) rememberUser(line int, snap tailSnap, rev string) error {
	st, err := loadBranchState(a.Cfg.SessionPath)
	if err != nil {
		return err
	}
	st.init()
	key := lineKey(line)
	if _, ok := st.Marks[key]; !ok {
		st.Marks[key] = userMark{CharterBefore: rev, Checkpoint: snap.Checkpoint, After: copyInts(snap.After)}
	}
	if a.branchPending != 0 {
		if g := st.Forks[lineKey(a.branchPending)]; g != nil && g.Active >= 0 && g.Active < len(g.UserLine) {
			g.UserLine[g.Active] = line
		}
		a.branchPending = 0
	}
	return saveBranchState(a.Cfg.SessionPath, st)
}

func (a *Agent) addBranchableUser(content string) error {
	snap, err := a.takeTail()
	if err != nil {
		return err
	}
	rev, err := a.charterRevisionID()
	if err != nil {
		return err
	}
	if err := a.add(map[string]any{"role": "user", "content": content}); err != nil {
		return err
	}
	line, err := lastJournalLine(a.Journal.File.Name())
	if err != nil {
		return err
	}
	return a.rememberUser(line, snap, rev)
}

func (a *Agent) dropPendingSibling(source int) {
	st, err := loadBranchState(a.Cfg.SessionPath)
	if err != nil || st == nil || st.Forks == nil {
		return
	}
	g := st.Forks[lineKey(source)]
	if g == nil || len(g.IDs) < 2 || len(g.UserLine) != len(g.IDs) {
		return
	}
	last := len(g.IDs) - 1
	if g.UserLine[last] != 0 {
		return
	}
	delete(st.Tips, g.IDs[last])
	g.IDs = g.IDs[:last]
	g.UserLine = g.UserLine[:last]
	g.Active = 0
	st.Active = g.IDs[0]
	if len(g.IDs) < 2 {
		delete(st.Forks, lineKey(source))
	}
	_ = saveBranchState(a.Cfg.SessionPath, st)
}

// prepareFork points the live session at a new continuation of a user message
// and records that message. Run then answers it without adding it again.
func (a *Agent) prepareFork(sourceLine int, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errEmptyBranchText
	}
	if a == nil || a.Journal == nil || a.Cfg.SessionPath == "" {
		return errors.New("no session")
	}
	path := a.Cfg.SessionPath
	rows, err := readJournalRows(path)
	if err != nil {
		return err
	}
	row, ok := rowByLine(rows, sourceLine)
	if !ok {
		return errNotUserMessage
	}
	if _, ok := editableUserText(row.Item); !ok {
		return errNotUserMessage
	}
	if row.Branch == a.branchTag() {
		if err := a.ensureTip(row.Branch); err != nil {
			return err
		}
	}
	st, err := loadBranchState(path)
	if err != nil {
		return err
	}
	st.init()
	mark, marked := st.Marks[lineKey(sourceLine)]
	snap := tailSnap{Checkpoint: mark.Checkpoint, After: copyInts(mark.After)}
	if !marked {
		snap = legacySnap(rows, sourceLine, row.Branch)
		mark = userMark{}
	}
	g := st.Forks[lineKey(sourceLine)]
	if g == nil {
		g = &forkGroup{Source: sourceLine, IDs: []string{row.Branch}, UserLine: []int{sourceLine}}
		st.Forks[lineKey(sourceLine)] = g
	}
	st.Next++
	id := "b" + strconv.Itoa(st.Next)
	g.IDs = append(g.IDs, id)
	g.UserLine = append(g.UserLine, 0)
	g.Active = len(g.IDs) - 1
	st.Active = id
	st.Tips[id] = mark.CharterBefore
	if err := saveBranchState(path, st); err != nil {
		return err
	}

	prevTag := a.Journal.Branch
	prevHist := append([]json.RawMessage(nil), a.History...)
	hist := loadTail(rows, snap)
	a.Journal.Branch = id
	a.Journal.History = hist
	a.History = hist
	held := snap
	held.After = copyInts(snap.After)
	a.branchHold = &held
	a.branchPending = sourceLine
	if err := a.applyCharterRevision(mark.CharterBefore); err != nil {
		a.Journal.Branch = prevTag
		a.Journal.History = prevHist
		a.History = prevHist
		a.branchHold = nil
		a.branchPending = 0
		a.dropPendingSibling(sourceLine)
		return err
	}
	if err := a.addBranchableUser(text); err != nil {
		a.Journal.Branch = prevTag
		a.Journal.History = prevHist
		a.History = prevHist
		a.branchHold = nil
		a.branchPending = 0
		a.dropPendingSibling(sourceLine)
		return err
	}
	a.branchUserReady = true
	return nil
}

func (a *Agent) selectContinuation(sourceLine, index int) error {
	if a == nil || a.Journal == nil || a.Cfg.SessionPath == "" {
		return errors.New("no session")
	}
	path := a.Cfg.SessionPath
	st, err := loadBranchState(path)
	if err != nil {
		return err
	}
	st.init()
	g := st.Forks[lineKey(sourceLine)]
	if g == nil || index < 0 || index >= len(g.IDs) {
		return errNoContinuation
	}
	g.Active = index
	tag := g.IDs[index]
	st.Active = tag
	if err := saveBranchState(path, st); err != nil {
		return err
	}
	rows, err := readJournalRows(path)
	if err != nil {
		return err
	}
	hist := modelItems(rows, st, tag)
	a.Journal.Branch = tag
	a.Journal.History = hist
	a.History = hist
	if id, ok := st.Tips[tag]; ok {
		return a.applyCharterRevision(id)
	}
	return nil
}

func (a *Agent) restoreActiveBranch() {
	if a == nil || a.Journal == nil || a.Cfg.SessionPath == "" {
		return
	}
	st, err := loadBranchState(a.Cfg.SessionPath)
	if err != nil || st.Active == "" {
		return
	}
	rows, err := readJournalRows(a.Cfg.SessionPath)
	if err != nil {
		return
	}
	hist := modelItems(rows, st, st.Active)
	a.Journal.Branch = st.Active
	a.Journal.History = hist
	a.History = hist
	if id, ok := st.Tips[st.Active]; ok {
		_ = a.applyCharterRevision(id)
	}
}
