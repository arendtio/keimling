package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleSpend serves the aggregated spend ledger for the settings view.
// Query: granularity=day|hour, days=N. Records stay raw in the ledger; this
// endpoint returns hourly/daily buckets grouped by provider and model.
func (s *appServer) handleSpend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	gran := normalizeSpendGranularity(r.URL.Query().Get("granularity"))
	days := parseSpendDays(gran, r.URL.Query().Get("days"))
	overview := spendOverviewFor(spendDir(), gran, days, time.Now())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(overview)
}
