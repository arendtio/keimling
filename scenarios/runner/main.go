package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	repo := flag.String("repo", ".", "repository root")
	binary := flag.String("binary", "./keimling", "keimling binary to run")
	scenario := flag.String("scenario", "", "scenario id (empty runs every scenario)")
	runs := flag.Int("runs", 3, "how many times to run each scenario")
	list := flag.Bool("list", false, "print scenario ids and exit")
	checkpoint := flag.Bool("checkpoint", false, "freeze the latest run of each scenario as the baseline")
	flag.Parse()
	root, err := filepath.Abs(*repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *list {
		ids, err := scenarioIDs(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, id := range ids {
			fmt.Println(id)
		}
		return
	}
	if *checkpoint {
		if err := writeCheckpoint(root); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("checkpoint written to", checkpointPath(root))
		return
	}
	if *runs < 1 {
		*runs = 1
	}
	bin, err := filepath.Abs(*binary)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if st, err := os.Stat(bin); err != nil || st.IsDir() {
		fmt.Fprintf(os.Stderr, "binary %s is missing; build keimling first\n", bin)
		os.Exit(1)
	}
	ids := []string{strings.TrimSpace(*scenario)}
	if ids[0] == "" {
		ids, err = scenarioIDs(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	commit := binaryCommit(bin)
	fmt.Println("version", commit)
	var words []string
	for _, id := range ids {
		word, err := runScenario(root, bin, id, commit, *runs)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if word != "" {
			words = append(words, word)
		}
	}
	if len(words) == 0 {
		return
	}
	verdict := sumVerdict(words)
	fmt.Println("sum:", verdict)
	if sumFails(verdict) {
		os.Exit(1)
	}
}

func runScenario(repo, bin, id, commit string, runs int) (string, error) {
	spec, err := loadSpec(filepath.Join(repo, "scenarios", id, "scenario.json"))
	if err != nil {
		return "", err
	}
	for n := 1; n <= runs; n++ {
		card, err := oneRun(repo, bin, id, spec, commit, n)
		if err != nil {
			return "", err
		}
		fmt.Printf("scenario %s run %d/%d passed=%v steps=%d tools=%s tokens=%d duration_ms=%d\n", id, n, runs, card.Passed, card.Steps, strings.Join(card.Tools, ","), card.InputTokens+card.OutputTokens, card.DurationMS)
		if len(card.Failures) > 0 {
			fmt.Println(strings.Join(card.Failures, "; "))
		}
	}
	cards, err := loadCards(filepath.Join(repo, "scenarios", "cards"), id)
	if err != nil {
		return "", err
	}
	model := cardsModel(cards, commit)
	cur := statsOf(lastN(cards, model, commit, runs))
	cp, err := loadCheckpoint(repo)
	if err != nil {
		return "", err
	}
	var base runStats
	baseName := ""
	if m, ok := cp.Scenarios[id]; ok && m.N > 0 && (m.Model == "" || m.Model == model) {
		base = m.stats()
		baseName = m.Commit
	} else {
		baseName = baselineCommit(cards, model, nil)
		if baseName == commit {
			base = statsOf(allButLastN(cards, model, commit, runs))
		} else if baseName != "" {
			base = statsOf(cardsFor(cards, model, baseName))
		}
	}
	if base.N == 0 || baseName == "" {
		fmt.Printf("%s: no earlier baseline\n", id)
		return "", nil
	}
	fmt.Println(compareLine(id, baseName, cur, base))
	return judge(cur, base), nil
}

func lastN(cards []scenarioCard, model, commit string, n int) []scenarioCard {
	match := cardsFor(cards, model, commit)
	if n > 0 && len(match) > n {
		return match[len(match)-n:]
	}
	return match
}

func allButLastN(cards []scenarioCard, model, commit string, n int) []scenarioCard {
	match := cardsFor(cards, model, commit)
	if n > 0 && len(match) > n {
		return match[:len(match)-n]
	}
	return nil
}

func cardsModel(cards []scenarioCard, commit string) string {
	for i := len(cards) - 1; i >= 0; i-- {
		if cards[i].Commit == commit && cards[i].Model != "" {
			return cards[i].Model
		}
	}
	return ""
}

func scenarioIDs(repo string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(repo, "scenarios"))
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(repo, "scenarios", e.Name(), "scenario.json")); err == nil {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}

func oneRun(repo, bin, id string, spec scenarioSpec, commit string, n int) (scenarioCard, error) {
	prep, err := prepare(id, spec.Setup)
	if err != nil {
		return scenarioCard{}, err
	}
	defer os.RemoveAll(prep.DataHome)
	args := []string{"--cli", "--oneshot", "--mode", "yolo", "--web=false", "--autopilot=false", "--resume=false", "--check-model=false"}
	if prep.PriorResult != "" {
		args = append(args, "--prior-result", prep.PriorResult)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = prep.Root
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+prep.DataHome)
	cmd.Stdin = strings.NewReader(spec.Task + "\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	out := stdout.String()
	session := sessionPath(out)
	model := modelName(out)
	inTok, outTok, dur := finishedLine(out)
	var items []json.RawMessage
	var results []byte
	if session != "" {
		raw, err := os.ReadFile(session)
		if err == nil {
			items = journalItems(raw)
		}
		results, _ = os.ReadFile(session + ".results.json")
	}
	card := scoreRun(id, spec, items, results, unchanged(prep.Watch), model, commit, inTok, outTok, dur)
	if runErr != nil && card.OutcomeOK {
		card.Passed = false
		card.OutcomeOK = false
		card.Failures = append(card.Failures, strings.TrimSpace(stderr.String()+" "+runErr.Error()))
	}
	if err := keepRun(repo, commit, id, n, session); err != nil {
		return card, err
	}
	if err := appendCard(filepath.Join(repo, "scenarios", "cards"), id, card); err != nil {
		return card, err
	}
	if runErr != nil && len(card.Failures) == 0 {
		return card, fmt.Errorf("%v\n%s", runErr, stderr.String())
	}
	return card, nil
}

func keepRun(repo, version, id string, n int, session string) error {
	if session == "" {
		return nil
	}
	dir := filepath.Join(repo, "scenarios", "runs", version, id, strconv.Itoa(n))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := copyFile(session, filepath.Join(dir, "session.jsonl")); err != nil {
		return err
	}
	results := session + ".results.json"
	if _, err := os.Stat(results); err == nil {
		return copyFile(results, filepath.Join(dir, "session.jsonl.results.json"))
	}
	return nil
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o600)
}

func sessionPath(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "Session: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Session: "))
		}
	}
	return ""
}

func modelName(out string) string {
	line := out
	if i := strings.IndexByte(out, '\n'); i >= 0 {
		line = out[:i]
	}
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func finishedLine(out string) (int64, int64, int64) {
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "run finished ") {
			continue
		}
		var inTok, outTok, dur int64
		for _, field := range strings.Fields(line) {
			k, v, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				continue
			}
			switch k {
			case "input_tokens":
				inTok = n
			case "output_tokens":
				outTok = n
			case "duration_ms":
				dur = n
			}
		}
		return inTok, outTok, dur
	}
	return 0, 0, 0
}

func binaryCommit(bin string) string {
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return "dev"
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return "dev"
	}
	return fields[len(fields)-1]
}
