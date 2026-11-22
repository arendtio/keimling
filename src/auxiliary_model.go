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
//
// A saved auxiliary model that no configured provider offers is ignored: the
// task falls back to the chat model and says so once.

// auxiliaryModel is the model one side task uses. override is that task's own
// flag value; empty means the flag was not set or the task has no flag.
func (a *Agent) auxiliaryModel(override string) string {
	if a == nil {
		return ""
	}
	if m := strings.TrimSpace(override); m != "" {
		return m
	}
	configured := strings.TrimSpace(a.Cfg.AuxiliaryModel)
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
	return a.Client.Review(ctx, a.auxiliaryModel(a.Cfg.ReviewModel), a.Cfg.ApprovalPolicy, tool, args, cwd, purpose, sus, a.Cfg.ApprovalPolicyReplace)
}

// pilotReview is one autopilot supervision call on the side-task model.
// The worker keeps the same instruction and history it always had.
func (a *Agent) pilotReview(ctx context.Context, message, report string) (PilotDecision, error) {
	return a.Client.Pilot(ctx, a.auxiliaryModel(a.Cfg.PilotModel), a.Cfg.PilotReasoning, message, report, a.autopilotPrompt(), a.modelInput())
}
