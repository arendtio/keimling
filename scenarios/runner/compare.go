package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type checkpointFile struct {
	Scenarios map[string]checkpointMark `json:"scenarios"`
}

type checkpointMark struct {
	Model    string `json:"model"`
	Commit   string `json:"commit"`
	Steps    int    `json:"steps"`
	Tokens   int64  `json:"tokens"`
	Duration int64  `json:"duration_ms"`
	Passed   int    `json:"passed"`
	N        int    `json:"n"`
}

func (m checkpointMark) stats() runStats {
	return runStats{Steps: m.Steps, Tokens: m.Tokens, Duration: m.Duration, Passed: m.Passed, N: m.N}
}

type runStats struct {
	Steps    int
	Tokens   int64
	Duration int64
	Passed   int
	N        int
}

func checkpointPath(repo string) string {
	return filepath.Join(repo, "scenarios", "checkpoint.json")
}

func loadCheckpoint(repo string) (checkpointFile, error) {
	var doc checkpointFile
	b, err := os.ReadFile(checkpointPath(repo))
	if err != nil {
		if os.IsNotExist(err) {
			return doc, nil
		}
		return doc, err
	}
	err = json.Unmarshal(b, &doc)
	return doc, err
}

func writeCheckpoint(repo string) error {
	ids, err := scenarioIDs(repo)
	if err != nil {
		return err
	}
	doc := checkpointFile{Scenarios: map[string]checkpointMark{}}
	for _, id := range ids {
		cards, err := loadCards(filepath.Join(repo, "scenarios", "cards"), id)
		if err != nil {
			return err
		}
		if len(cards) == 0 {
			continue
		}
		last := cards[len(cards)-1]
		st := statsOf(trailing(cards))
		doc.Scenarios[id] = checkpointMark{Model: last.Model, Commit: last.Commit, Steps: st.Steps, Tokens: st.Tokens, Duration: st.Duration, Passed: st.Passed, N: st.N}
	}
	if len(doc.Scenarios) == 0 {
		return fmt.Errorf("no scenario cards to freeze")
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(checkpointPath(repo), append(b, '\n'), 0o600)
}

func trailing(cards []scenarioCard) []scenarioCard {
	if len(cards) == 0 {
		return nil
	}
	last := cards[len(cards)-1]
	i := len(cards) - 1
	for i > 0 && cards[i-1].Model == last.Model && cards[i-1].Commit == last.Commit {
		i--
	}
	return cards[i:]
}

func cardsFor(cards []scenarioCard, model, commit string) []scenarioCard {
	var out []scenarioCard
	for _, c := range cards {
		if c.Model == model && c.Commit == commit {
			out = append(out, c)
		}
	}
	return out
}

// baselineCommit is the checkpoint when one is set for this scenario and model.
// Otherwise it is the latest card of that model whose stamp has no plus sign.
func baselineCommit(cards []scenarioCard, model string, mark *checkpointMark) string {
	if mark != nil && mark.Commit != "" && (mark.Model == "" || mark.Model == model) {
		return mark.Commit
	}
	found := ""
	for _, c := range cards {
		if c.Model == model && !strings.Contains(c.Commit, "+") {
			found = c.Commit
		}
	}
	return found
}

func statsOf(cards []scenarioCard) runStats {
	var steps []int
	var tokens, durs []int64
	st := runStats{N: len(cards)}
	for _, c := range cards {
		steps = append(steps, c.Steps)
		tokens = append(tokens, c.InputTokens+c.OutputTokens)
		durs = append(durs, c.DurationMS)
		if c.Passed {
			st.Passed++
		}
	}
	st.Steps = medianInt(steps)
	st.Tokens = medianInt64(tokens)
	st.Duration = medianInt64(durs)
	return st
}

// judge names the change from base to cur. A lost pass outweighs a shorter path.
// Time is reported by the caller and does not flip this word on its own.
func judge(cur, base runStats) string {
	if cur.Passed < base.Passed {
		return "worse"
	}
	if cur.Passed > base.Passed {
		return "better"
	}
	stepUp := cur.Steps > base.Steps
	stepDown := cur.Steps < base.Steps
	tokUp := cur.Tokens > base.Tokens
	tokDown := cur.Tokens < base.Tokens
	if (stepUp || tokUp) && (stepDown || tokDown) {
		return "mixed"
	}
	if stepUp || tokUp {
		return "worse"
	}
	if stepDown || tokDown {
		return "better"
	}
	return "same"
}

func sumFails(verdict string) bool {
	return verdict == "worse"
}

func sumVerdict(words []string) string {
	better, worse := false, false
	for _, w := range words {
		switch w {
		case "better":
			better = true
		case "worse":
			worse = true
		case "mixed":
			better, worse = true, true
		}
	}
	switch {
	case better && worse:
		return "mixed"
	case worse:
		return "worse"
	case better:
		return "better"
	default:
		return "same"
	}
}

func compareLine(id, baseCommit string, cur, base runStats) string {
	return fmt.Sprintf("%s: %s against %s, steps %d (%+d), tokens %d (%+d), time %dms (%+dms), passed %d/%d (was %d/%d)",
		id, judge(cur, base), baseCommit,
		cur.Steps, cur.Steps-base.Steps,
		cur.Tokens, cur.Tokens-base.Tokens,
		cur.Duration, cur.Duration-base.Duration,
		cur.Passed, cur.N, base.Passed, base.N)
}
