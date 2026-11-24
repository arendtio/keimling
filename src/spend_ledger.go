package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The spend ledger records every completed model response and every real
// provider budget fetch. It lives in the Keimling config directory, never in
// the workspace, so a running chat cannot dirty a Git tree. Files are JSONL,
// one file per month, appended under a process-wide mutex.

const (
	spendGranularityHour = "hour"
	spendGranularityDay  = "day"

	spendRecordPrefix  = "spend"
	budgetRecordPrefix = "budget"

	spendDefaultDays = 30
	spendMaxDays     = 366
	spendMaxHourDays = 7
)

// spendDir is the ledger directory beside config.json.
func spendDir() string {
	return filepath.Join(keimlingConfigDir(), "spend")
}

// spendConfigFile is the settings file used to resolve model prices. main sets
// it to the effective --config path; the default matches the process default.
var spendConfigFile = filepath.Join(keimlingConfigDir(), "config.json")

// spendRecord is one completed model response.
type spendRecord struct {
	Time         string   `json:"time"` // RFC3339 UTC
	Provider     string   `json:"provider"`
	Model        string   `json:"model"`
	Call         string   `json:"call,omitempty"` // chat | compaction | filter
	InputTokens  int64    `json:"input_tokens"`
	OutputTokens int64    `json:"output_tokens"`
	USD          *float64 `json:"usd,omitempty"`
	Priced       bool     `json:"priced"`
	// AgentID and ThreadID name who caused the call. Empty for the human's own
	// chat and for helper calls, so old records stay valid.
	AgentID  string `json:"agent_id,omitempty"`
	ThreadID string `json:"thread_id,omitempty"`
}

// spendBudgetRecord is one real provider budget fetch, not a cache hit.
type spendBudgetRecord struct {
	Time                string   `json:"time"`
	Provider            string   `json:"provider"`
	Kind                string   `json:"kind,omitempty"`
	RemainingUSD        *float64 `json:"remaining_usd,omitempty"`
	LimitUSD            *float64 `json:"limit_usd,omitempty"`
	UsageUSD            *float64 `json:"usage_usd,omitempty"`
	AccountRemainingUSD *float64 `json:"account_remaining_usd,omitempty"`
}

var spendLedgerMu sync.Mutex

// appendSpendJSONL appends one marshaled record to the monthly file for at.
func appendSpendJSONL(dir, prefix string, at time.Time, line []byte) error {
	spendLedgerMu.Lock()
	defer spendLedgerMu.Unlock()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	name := prefix + "-" + at.UTC().Format("2006-01") + ".jsonl"
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

// recordCloudSpend is the Client hook: it turns one response into one ledger
// line. A ledger write must never break a chat, so errors only log.
func recordCloudSpend(rec spendRecord) {
	recordModelSpend(spendDir(), rec)
}

// agentSpendHook stamps the caller on each record, so a limit can be enforced
// per agent later. A nil agent keeps the plain hook.
func agentSpendHook(a *Agent) func(spendRecord) {
	if a == nil {
		return recordCloudSpend
	}
	return func(rec spendRecord) {
		if rec.AgentID == "" {
			rec.AgentID = strings.TrimSpace(a.Cfg.AgentInstanceID)
		}
		if rec.ThreadID == "" {
			rec.ThreadID = strings.TrimSpace(a.Cfg.ThreadID)
		}
		recordCloudSpend(rec)
	}
}

// agentSpendToday sums the priced calls of one agent for the current UTC day.
// An unreadable or missing ledger means nothing has been spent.
func agentSpendToday(agentID string) float64 {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return 0
	}
	day := time.Now().UTC().Format("2006-01-02")
	path := filepath.Join(spendDir(), spendRecordPrefix+"-"+day[:7]+".jsonl")
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	total := 0.0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var rec spendRecord
		if json.Unmarshal(sc.Bytes(), &rec) != nil {
			continue
		}
		if rec.AgentID != agentID || rec.USD == nil {
			continue
		}
		if t, err := time.Parse(time.RFC3339, rec.Time); err == nil && t.UTC().Format("2006-01-02") != day {
			continue
		}
		total += *rec.USD
	}
	return total
}

// recordModelSpend prices and appends one model response. Without a known
// model price the record is kept with priced:false and no invented amount.
func recordModelSpend(dir string, rec spendRecord) {
	if rec.InputTokens <= 0 && rec.OutputTokens <= 0 {
		return
	}
	if strings.TrimSpace(rec.Time) == "" {
		rec.Time = time.Now().UTC().Format(time.RFC3339)
	}
	rec.Model = strings.TrimSpace(rec.Model)
	if pm, kind := spendPriceFor(rec.Model); pm != nil {
		if usd, ok := estimateSessionUSD(rec.InputTokens, rec.OutputTokens, pm); ok {
			rec.USD = usd
			rec.Priced = true
		}
		if kind != "" {
			rec.Provider = kind
		}
	}
	if strings.TrimSpace(rec.Provider) == "" {
		rec.Provider = "unknown"
	}
	if rec.Model == "" {
		rec.Model = "unknown"
	}
	at, err := time.Parse(time.RFC3339, rec.Time)
	if err != nil {
		at = time.Now().UTC()
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return
	}
	if err := appendSpendJSONL(dir, spendRecordPrefix, at, line); err != nil {
		fmt.Printf("spend ledger: %v\n", err)
	}
}

// recordBudgetSnapshot stores one real provider budget fetch. Providers without
// a budget endpoint or a failed fetch leave nothing behind.
func recordBudgetSnapshot(rec spendBudgetRecord) {
	recordBudgetSnapshotIn(spendDir(), rec)
}

func recordBudgetSnapshotIn(dir string, rec spendBudgetRecord) {
	if rec.RemainingUSD == nil && rec.LimitUSD == nil && rec.UsageUSD == nil && rec.AccountRemainingUSD == nil {
		return
	}
	if strings.TrimSpace(rec.Time) == "" {
		rec.Time = time.Now().UTC().Format(time.RFC3339)
	}
	at, err := time.Parse(time.RFC3339, rec.Time)
	if err != nil {
		at = time.Now().UTC()
	}
	if strings.TrimSpace(rec.Provider) == "" {
		rec.Provider = "unknown"
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return
	}
	if err := appendSpendJSONL(dir, budgetRecordPrefix, at, line); err != nil {
		fmt.Printf("spend ledger: %v\n", err)
	}
}

// spendPriceFor resolves the configured per-token prices of a model id.
func spendPriceFor(model string) (*providerModel, string) {
	return spendPriceIn(spendConfigFile, model)
}

// spendPriceIn reads model prices from the settings file at path. The first
// priced match wins; a provider kind travels with it for grouping.
func spendPriceIn(path, model string) (*providerModel, string) {
	model = strings.TrimSpace(model)
	path = strings.TrimSpace(path)
	if model == "" || path == "" {
		return nil, ""
	}
	fc, err := readConfiguration(path)
	if err != nil {
		return nil, ""
	}
	fc.ensureModelProviders()
	var fallback *providerModel
	fallbackKind := ""
	for i := range fc.ModelProviders {
		p := fc.ModelProviders[i]
		kind := detectProviderKind(p.BaseURL, p.Kind)
		for j := range p.Models {
			m := p.Models[j]
			if strings.TrimSpace(m.ID) != model {
				continue
			}
			if strings.TrimSpace(m.PromptPrice) != "" || strings.TrimSpace(m.CompletionPrice) != "" {
				match := m
				return &match, kind
			}
			if fallback == nil {
				match := m
				fallback, fallbackKind = &match, kind
			}
		}
	}
	return fallback, fallbackKind
}

// ledgerFiles lists the monthly ledger files for one prefix in name order.
func ledgerFiles(dir, prefix string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, prefix+"-") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	sort.Strings(out)
	return out
}

// parseSpendLines keeps every readable record and drops blank or broken lines.
func parseSpendLines(data []byte) []spendRecord {
	var out []spendRecord
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec spendRecord
		if json.Unmarshal([]byte(line), &rec) != nil {
			continue
		}
		if _, err := time.Parse(time.RFC3339, strings.TrimSpace(rec.Time)); err != nil {
			continue
		}
		out = append(out, rec)
	}
	return out
}

// parseBudgetLines keeps every readable budget snapshot.
func parseBudgetLines(data []byte) []spendBudgetRecord {
	var out []spendBudgetRecord
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec spendBudgetRecord
		if json.Unmarshal([]byte(line), &rec) != nil {
			continue
		}
		if _, err := time.Parse(time.RFC3339, strings.TrimSpace(rec.Time)); err != nil {
			continue
		}
		out = append(out, rec)
	}
	return out
}

// readSpendRecords returns every ledger record on disk.
func readSpendRecords(dir string) []spendRecord {
	var out []spendRecord
	for _, path := range ledgerFiles(dir, spendRecordPrefix) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		out = append(out, parseSpendLines(data)...)
	}
	return out
}

// readBudgetRecords returns every budget snapshot on disk.
func readBudgetRecords(dir string) []spendBudgetRecord {
	var out []spendBudgetRecord
	for _, path := range ledgerFiles(dir, budgetRecordPrefix) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		out = append(out, parseBudgetLines(data)...)
	}
	return out
}

// spendTotals is one aggregation cell.
type spendTotals struct {
	USD          float64 `json:"usd"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Calls        int64   `json:"calls"`
	Unpriced     int64   `json:"unpriced_calls"`
}

func (t *spendTotals) add(rec spendRecord) {
	t.Calls++
	t.InputTokens += rec.InputTokens
	t.OutputTokens += rec.OutputTokens
	if rec.Priced && rec.USD != nil {
		t.USD += *rec.USD
	} else {
		t.Unpriced++
	}
}

// spendBucket is one hour or day of the cost chart.
type spendBucket struct {
	Start string `json:"start"`
	spendTotals
}

// spendModelRow is one provider/model line of the cost table.
type spendModelRow struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	spendTotals
}

// spendOverview is the response of GET /api/spend.
type spendOverview struct {
	Granularity string              `json:"granularity"`
	Since       string              `json:"since"`
	Until       string              `json:"until"`
	Totals      spendTotals         `json:"totals"`
	Buckets     []spendBucket       `json:"buckets"`
	ByModel     []spendModelRow     `json:"by_model"`
	Budget      []spendBudgetRecord `json:"budget"`
}

// normalizeSpendGranularity accepts hour/daily spellings and defaults to day.
func normalizeSpendGranularity(g string) string {
	switch strings.ToLower(strings.TrimSpace(g)) {
	case spendGranularityHour, "hourly", "h":
		return spendGranularityHour
	default:
		return spendGranularityDay
	}
}

// spendDays clamps the requested window to what a chart can render.
func spendDays(gran string, days int) int {
	if days <= 0 {
		if gran == spendGranularityHour {
			return 3
		}
		return spendDefaultDays
	}
	max := spendMaxDays
	if gran == spendGranularityHour {
		max = spendMaxHourDays
	}
	if days > max {
		return max
	}
	return days
}

func spendStep(gran string) time.Duration {
	if gran == spendGranularityHour {
		return time.Hour
	}
	return 24 * time.Hour
}

// spendBucketStart truncates a UTC timestamp to its hour or day.
func spendBucketStart(at time.Time, gran string) time.Time {
	at = at.UTC()
	if gran == spendGranularityHour {
		return at.Truncate(time.Hour)
	}
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
}

// spendWindowStart is the first bucket of the requested window.
func spendWindowStart(gran string, days int, now time.Time) time.Time {
	last := spendBucketStart(now, gran)
	if gran == spendGranularityHour {
		return last.Add(-time.Duration(days*24-1) * time.Hour)
	}
	return last.AddDate(0, 0, -(days - 1))
}

// aggregateSpend buckets records by hour or day and groups them by model.
// Records outside the window (older than days, or in the future) are skipped.
func aggregateSpend(records []spendRecord, gran string, days int, now time.Time) spendOverview {
	gran = normalizeSpendGranularity(gran)
	days = spendDays(gran, days)
	step := spendStep(gran)
	first := spendWindowStart(gran, days, now)
	last := spendBucketStart(now, gran)
	count := int(last.Sub(first)/step) + 1
	if count < 1 {
		count = 1
	}
	buckets := make([]spendBucket, count)
	for i := range buckets {
		buckets[i].Start = first.Add(time.Duration(i) * step).Format(time.RFC3339)
	}
	var totals spendTotals
	rows := map[string]*spendModelRow{}
	end := last.Add(step)
	for _, rec := range records {
		at, err := time.Parse(time.RFC3339, strings.TrimSpace(rec.Time))
		if err != nil {
			continue
		}
		at = at.UTC()
		if at.Before(first) || !at.Before(end) {
			continue
		}
		idx := int(at.Sub(first) / step)
		if idx < 0 || idx >= count {
			continue
		}
		buckets[idx].spendTotals.add(rec)
		totals.add(rec)
		provider := strings.TrimSpace(rec.Provider)
		if provider == "" {
			provider = "unknown"
		}
		model := strings.TrimSpace(rec.Model)
		if model == "" {
			model = "unknown"
		}
		key := provider + "\x00" + model
		row := rows[key]
		if row == nil {
			row = &spendModelRow{Provider: provider, Model: model}
			rows[key] = row
		}
		row.spendTotals.add(rec)
	}
	byModel := make([]spendModelRow, 0, len(rows))
	for _, row := range rows {
		byModel = append(byModel, *row)
	}
	sort.Slice(byModel, func(i, j int) bool {
		if byModel[i].USD != byModel[j].USD {
			return byModel[i].USD > byModel[j].USD
		}
		ti := byModel[i].InputTokens + byModel[i].OutputTokens
		tj := byModel[j].InputTokens + byModel[j].OutputTokens
		if ti != tj {
			return ti > tj
		}
		if byModel[i].Model != byModel[j].Model {
			return byModel[i].Model < byModel[j].Model
		}
		return byModel[i].Provider < byModel[j].Provider
	})
	return spendOverview{
		Granularity: gran,
		Since:       first.Format(time.RFC3339),
		Until:       now.UTC().Format(time.RFC3339),
		Totals:      totals,
		Buckets:     buckets,
		ByModel:     byModel,
	}
}

// spendOverviewFor reads the ledger and returns the aggregated window plus the
// budget history inside that window.
func spendOverviewFor(dir, gran string, days int, now time.Time) spendOverview {
	gran = normalizeSpendGranularity(gran)
	days = spendDays(gran, days)
	out := aggregateSpend(readSpendRecords(dir), gran, days, now)
	first, err := time.Parse(time.RFC3339, out.Since)
	if err != nil {
		first = now.UTC().AddDate(0, 0, -days)
	}
	for _, rec := range readBudgetRecords(dir) {
		at, err := time.Parse(time.RFC3339, strings.TrimSpace(rec.Time))
		if err != nil {
			continue
		}
		if at.UTC().Before(first) {
			continue
		}
		out.Budget = append(out.Budget, rec)
	}
	sort.Slice(out.Budget, func(i, j int) bool { return out.Budget[i].Time < out.Budget[j].Time })
	return out
}

// parseSpendDays reads the optional days query value.
func parseSpendDays(gran string, raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return spendDays(gran, 0)
	}
	return spendDays(gran, n)
}

// agentSpendLimitError refuses the next turn once an instance has used up the
// daily cap of its definition. A chat or an instance without a cap is never
// refused.
func agentSpendLimitError(a *Agent) error {
	if a == nil {
		return nil
	}
	id := strings.TrimSpace(a.Cfg.AgentInstanceID)
	if id == "" {
		return nil
	}
	m, err := readAgentInstanceMeta(id)
	if err != nil {
		return nil
	}
	rec, err := readAgentRecipe(m.RecipeID)
	if err != nil || rec.SpendLimitUSD == nil {
		return nil
	}
	limit := *rec.SpendLimitUSD
	if limit <= 0 {
		return nil
	}
	spent := agentSpendToday(id)
	if spent < limit {
		return nil
	}
	return fmt.Errorf("spend limit reached: $%.2f of $%.2f used today. Raise the limit in the definition or wait for tomorrow", spent, limit)
}
