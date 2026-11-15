package main

import (
	"fmt"
	"os"
	"strings"
)

var colorOn bool

func termColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func paint(code, text string) string {
	if !colorOn || text == "" {
		return text
	}
	return "\033[" + code + "m" + text + "\033[0m"
}

func verdictColor(word string) string {
	switch word {
	case "better":
		return "32"
	case "worse":
		return "31"
	case "mixed":
		return "33"
	default:
		return "2"
	}
}

func costDelta(n int64) string {
	s := fmt.Sprintf("%+d", n)
	switch {
	case n < 0:
		return paint("32", s)
	case n > 0:
		return paint("31", s)
	default:
		return paint("2", s)
	}
}

func fmtDur(ms int64) string {
	if ms < 0 {
		ms = -ms
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func signedDur(ms int64) string {
	sign := "+"
	if ms < 0 {
		sign = "-"
	}
	body := sign + fmtDur(ms)
	return costDeltaText(ms, body)
}

func costDeltaText(n int64, body string) string {
	switch {
	case n < 0:
		return paint("32", body)
	case n > 0:
		return paint("31", body)
	default:
		return paint("2", body)
	}
}

func toolList(names []string) string {
	if len(names) == 0 {
		return "—"
	}
	var parts []string
	for i := 0; i < len(names); {
		j := i + 1
		for j < len(names) && names[j] == names[i] {
			j++
		}
		if j-i == 1 {
			parts = append(parts, names[i])
		} else {
			parts = append(parts, fmt.Sprintf("%s ×%d", names[i], j-i))
		}
		i = j
	}
	return strings.Join(parts, ", ")
}

func printVersion(commit string) {
	fmt.Println(paint("2", "version "+commit))
}

func printScenario(id string) {
	fmt.Printf("\n%s\n", paint("1;36", id))
}

func printRun(n, runs int, card scenarioCard) {
	mark := paint("32", "pass")
	if !card.Passed {
		mark = paint("31", "fail")
	}
	fmt.Printf("  %d/%d  %s  %d steps  %d tok  %s  %s\n",
		n, runs, mark, card.Steps, card.InputTokens+card.OutputTokens, fmtDur(card.DurationMS), paint("2", toolList(card.Tools)))
	if len(card.Failures) > 0 {
		fmt.Printf("       %s\n", paint("31", strings.Join(card.Failures, "; ")))
	}
}

func printBaseline() {
	fmt.Printf("  %s\n", paint("2", "no earlier baseline"))
}

func printCompare(baseCommit string, cur, base runStats) {
	word := judge(cur, base)
	fmt.Printf("  %s  %s\n", paint("1;"+verdictColor(word), word), paint("2", "against "+baseCommit))
	fmt.Printf("  steps %d (%s)  tokens %d (%s)  time %s (%s)  passed %d/%d (was %d/%d)\n",
		cur.Steps, costDelta(int64(cur.Steps-base.Steps)),
		cur.Tokens, shownDelta(cur.Tokens-base.Tokens, withinNoise(cur.Tokens-base.Tokens, base.Tokens, cur.TokenSpan, base.TokenSpan)),
		fmtDur(cur.Duration), shownDur(cur.Duration-base.Duration, withinNoise(cur.Duration-base.Duration, base.Duration, cur.DurSpan, base.DurSpan)),
		cur.Passed, cur.N, base.Passed, base.N)
}

func shownDelta(n int64, quiet bool) string {
	s := fmt.Sprintf("%+d", n)
	if quiet {
		return paint("2", s)
	}
	return costDelta(n)
}

func shownDur(ms int64, quiet bool) string {
	sign := "+"
	if ms < 0 {
		sign = "-"
	}
	body := sign + fmtDur(ms)
	if quiet {
		return paint("2", body)
	}
	return costDeltaText(ms, body)
}

func printSum(verdict string) {
	fmt.Printf("\n%s  %s\n", paint("1", "sum"), paint("1;"+verdictColor(verdict), verdict))
}
