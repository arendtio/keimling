package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// auxiliary_model is the model for the side tasks that used to be pinned to
// their own flag defaults: approval review, output filtering, autopilot
// supervision, the fast delegate, and the external web search. Empty means
// each of those tasks uses the chat model of its run.
//
// Precedence, highest first:
//  1. an explicit flag (--review-model, --autopilot-model, --delegate-fast-model),
//  2. auxiliary_model from config.json (--auxiliary-model),
//  3. the chat model of this run or thread.
// Every task passes its own marker; auxiliarySetting is the one table that
// maps a task to the setting it reads, so a task can get its own field later.
//
// A saved auxiliary model that no configured provider offers is ignored: the
// task falls back to the chat model and says so once.

// auxiliaryTask names one side task. The marker keeps the "which model does
// this task read" decision out of the call sites.
type auxiliaryTask int

const (
	auxReview auxiliaryTask = iota
	auxFilter
	auxPilot
	auxDelegate
	auxSearch
)

// auxiliarySetting maps one task to its own flag and to the saved setting it
// follows. This switch is the only place that decides it: giving the approval
// review its own model later is one case here plus one settings field, and no
// call site changes.
func (c Config) auxiliarySetting(task auxiliaryTask) (flagValue, saved string) {
	switch task {
	case auxReview:
		return c.ReviewModel, c.AuxiliaryModel
	case auxPilot:
		return c.PilotModel, c.AuxiliaryModel
	case auxDelegate:
		return c.DelegateFast, c.AuxiliaryModel
	case auxFilter, auxSearch:
		return "", c.AuxiliaryModel
	default:
		return "", c.AuxiliaryModel
	}
}

// auxiliaryModel is the model one side task uses. Precedence, highest first:
//  1. the task's own flag (--review-model, --autopilot-model, --delegate-fast-model),
//  2. auxiliary_model from config.json (--auxiliary-model),
//  3. the chat model of this run or thread.
//
// A saved model that no configured provider offers is ignored: the task falls
// back to the chat model and says so once.
func (a *Agent) auxiliaryModel(task auxiliaryTask) string {
	if a == nil {
		return ""
	}
	flagValue, configured := a.Cfg.auxiliarySetting(task)
	if m := strings.TrimSpace(flagValue); m != "" {
		return m
	}
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return a.auxiliaryChatModel()
	}
	if !auxiliaryModelSaved(a.Cfg.ConfigPath, configured) {
		fallback := a.auxiliaryChatModel()
		a.noteAuxiliaryFallback(configured, fallback)
		return fallback
	}
	return configured
}

// auxiliaryChatModel is the normal chat model: the one frozen for the run in
// progress, else the Settings chat model.
func (a *Agent) auxiliaryChatModel() string {
	if a == nil {
		return ""
	}
	if a.runChat != nil {
		if m := strings.TrimSpace(a.runChat.Model); m != "" {
			return m
		}
	}
	return strings.TrimSpace(a.Cfg.Model)
}

// auxiliaryModelSaved reports whether a configured provider offers the model.
// Without saved providers there is nothing to compare, so the value stands.
func auxiliaryModelSaved(configPath, model string) bool {
	configPath = strings.TrimSpace(configPath)
	model = strings.TrimSpace(model)
	if configPath == "" || model == "" {
		return true
	}
	fc, err := readConfiguration(configPath)
	if err != nil || len(fc.ModelProviders) == 0 {
		return true
	}
	fc.ensureModelProviders()
	p, m := findProviderModel(fc.ModelProviders, model, fc.ActiveModelProviderID)
	return p != nil && m != nil
}

// auxiliaryWarned keeps the fallback notice to one line per model and process.
var auxiliaryWarned sync.Map

func (a *Agent) noteAuxiliaryFallback(configured, fallback string) {
	key := strings.TrimSpace(a.Cfg.ConfigPath) + "\x00" + configured
	if _, seen := auxiliaryWarned.LoadOrStore(key, true); seen {
		return
	}
	if fallback == "" {
		fallback = "the chat model"
	}
	a.println(fmt.Sprintf("\u25cc Auxiliary model %q is not a saved provider model; using %q for side tasks. Change it in Settings.", configured, fallback))
}

// approvalReview is the automatic approval review on the side-task model.
func (a *Agent) approvalReview(ctx context.Context, tool, args, cwd, purpose string, sus *SoftSuspicion) (Verdict, error) {
	return a.Client.Review(ctx, a.auxiliaryModel(auxReview), a.Cfg.ApprovalPolicy, tool, args, cwd, purpose, sus, a.Cfg.ApprovalPolicyReplace)
}

// pilotReview is one autopilot supervision call on the side-task model.
// The worker keeps the same instruction and history it always had.
func (a *Agent) pilotReview(ctx context.Context, message, report string) (PilotDecision, error) {
	return a.Client.Pilot(ctx, a.auxiliaryModel(auxPilot), a.Cfg.PilotReasoning, message, report, a.autopilotPrompt(), a.modelInput())
}
