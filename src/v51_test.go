package main

import (
	"encoding/json"
	"testing"
)

func TestContextUsageCalibration(t *testing.T) {
	history := []json.RawMessage{json.RawMessage(`{"role":"user","content":"hello"}`)}
	raw := contextEstimate(history)
	a := &Agent{History: history, Client: NewClient("")}
	if got := a.contextOccupancyEstimate(); got != raw {
		t.Fatalf("unmeasured: got %d want %d", got, raw)
	}
	a.Client.LastMainUsage = MainUsage{Available: true, InputTokens: 26, EstimatedBefore: raw}
	if got := a.contextOccupancyEstimate(); got != 30 {
		t.Fatalf("calibrated: got %d want 30", got)
	}
	a.History = append(a.History, json.RawMessage(`{"role":"user","content":"followup"}`))
	growth := contextEstimate(a.History) - raw
	want := 26 + growth + (26+growth+6)/7
	if got := a.contextOccupancyEstimate(); got != want {
		t.Fatalf("growth got %d want %d", got, want)
	}
	a.History = nil
	if a.contextUsageCalibrated() {
		t.Fatal("stale calibration should not apply after structural reduction")
	}
}
