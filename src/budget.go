package main

import (
	"fmt"
	"strings"
)

const budgetWarnPct = 70.0

// Approximate tokens for the injected <context_budget> developer block.
const budgetBlockOverhead = 180

// ContextBudget is the occupancy signal injected into every main model call.
type ContextBudget struct {
	Limit        int     `json:"limit"`
	Used         int     `json:"used"`
	Remaining    int     `json:"remaining"`
	UsedPct      float64 `json:"used_pct"`
	RemainingPct float64 `json:"remaining_pct"`
	Source       string  `json:"source"` // usage | calibrated | estimate
	Warn         bool    `json:"warn"`
}

func (a *Agent) contextBudget() ContextBudget {
	return a.contextBudgetFromUsed(a.contextOccupancyEstimate())
}

func (a *Agent) contextBudgetFromUsed(used int) ContextBudget {
	limit := 0
	source := "estimate"
	if a != nil {
		limit = a.Cfg.ContextLimit
		if a.contextUsageCalibrated() {
			source = "calibrated"
			if u := a.Client.LastMainUsage; u.Available && u.InputTokens > 0 {
				raw := a.rawModelContextEstimate()
				if raw <= u.EstimatedBefore+32 {
					used = int(u.InputTokens)
					source = "usage"
				}
			}
		}
	}
	remaining := 0
	usedPct := 0.0
	remPct := 100.0
	if limit > 0 {
		remaining = limit - used
		if remaining < 0 {
			remaining = 0
		}
		usedPct = 100 * float64(used) / float64(limit)
		if usedPct > 100 {
			usedPct = 100
		}
		remPct = 100 - usedPct
		if remPct < 0 {
			remPct = 0
		}
	}
	return ContextBudget{
		Limit:        limit,
		Used:         used,
		Remaining:    remaining,
		UsedPct:      usedPct,
		RemainingPct: remPct,
		Source:       source,
		Warn:         limit > 0 && usedPct >= budgetWarnPct,
	}
}

func (b ContextBudget) FormatBlock() string {
	var sb strings.Builder
	sb.WriteString("<context_budget>\n")
	fmt.Fprintf(&sb, "limit=%d used=%d remaining=%d used_pct=%.1f remaining_pct=%.1f source=%s\n",
		b.Limit, b.Used, b.Remaining, b.UsedPct, b.RemainingPct, b.Source)
	if b.Warn {
		sb.WriteString("WARNING: context occupancy is high (≥70%). Prefer terse tool use; avoid stuffing incidental detail into replies or tool args; rely on charter updates (findings/plan) instead of repeating raw dumps.\n")
	} else {
		sb.WriteString("Prefer updating the structured charter over carrying large raw transcripts forward.\n")
	}
	sb.WriteString("</context_budget>")
	return sb.String()
}
