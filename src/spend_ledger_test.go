package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	at, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse %s: %v", s, err)
	}
	return at.UTC()
}

func usdPtr(v float64) *float64 { return &v }

func TestParseSpendLines(t *testing.T) {
	input := `{"time":"2026-01-02T03:04:05Z","provider":"openrouter","model":"vendor/a","input_tokens":10,"output_tokens":5,"usd":0.001,"priced":true}
not json at all
{"time":"","provider":"openrouter","model":"vendor/a"}
{"time":"2026-01-02T03:04:05Z","provider":"litellm","model":"b","input_tokens":1,"output_tokens":2,"priced":false}

`
	recs := parseSpendLines([]byte(input))
	if len(recs) != 2 {
		t.Fatalf("want 2 records, got %d: %+v", len(recs), recs)
	}
	if recs[0].Model != "vendor/a" || !recs[0].Priced || recs[0].USD == nil || *recs[0].USD != 0.001 {
		t.Fatalf("first record wrong: %+v", recs[0])
	}
	if recs[1].Priced || recs[1].USD != nil || recs[1].Provider != "litellm" {
		t.Fatalf("second record should stay unpriced: %+v", recs[1])
	}
}

func TestParseBudgetLines(t *testing.T) {
	input := `{"time":"2026-02-01T00:00:00Z","provider":"p1","kind":"openrouter","remaining_usd":4.5,"limit_usd":10,"usage_usd":5.5,"account_remaining_usd":42}
{"time":"bogus","provider":"p1"}
{"provider":"p1","remaining_usd":1}
`
	recs := parseBudgetLines([]byte(input))
	if len(recs) != 1 {
		t.Fatalf("want 1 snapshot, got %d: %+v", len(recs), recs)
	}
	if recs[0].RemainingUSD == nil || *recs[0].RemainingUSD != 4.5 || recs[0].AccountRemainingUSD == nil || *recs[0].AccountRemainingUSD != 42 {
		t.Fatalf("snapshot values wrong: %+v", recs[0])
	}
}

func TestAggregateSpendBuckets(t *testing.T) {
	now := mustTime(t, "2026-03-10T12:30:00Z")
	records := []spendRecord{
		{Time: "2026-03-09T01:00:00Z", Provider: "openrouter", Model: "vendor/a", InputTokens: 100, OutputTokens: 50, USD: usdPtr(0.01), Priced: true},
		{Time: "2026-03-09T23:59:00Z", Provider: "openrouter", Model: "vendor/a", InputTokens: 20, OutputTokens: 10, USD: usdPtr(0.002), Priced: true},
		{Time: "2026-03-10T02:00:00Z", Provider: "litellm", Model: "vendor/b", InputTokens: 7, OutputTokens: 3},
		{Time: "2026-03-10T02:30:00Z", Provider: "litellm", Model: "vendor/b", InputTokens: 7, OutputTokens: 3, USD: usdPtr(0.5), Priced: true},
	}
	agg := aggregateSpend(records, spendGranularityDay, 3, now)
	if agg.Granularity != spendGranularityDay {
		t.Fatalf("granularity = %q", agg.Granularity)
	}
	if got, want := agg.Since, "2026-03-08T00:00:00Z"; got != want {
		t.Fatalf("since = %q, want %q", got, want)
	}
	if len(agg.Buckets) != 3 {
		t.Fatalf("want 3 day buckets, got %d", len(agg.Buckets))
	}
	if got := agg.Buckets[0]; got.Start != "2026-03-08T00:00:00Z" || got.Calls != 0 || got.USD != 0 {
		t.Fatalf("empty bucket not zero-filled: %+v", got)
	}
	if got := agg.Buckets[1]; got.Start != "2026-03-09T00:00:00Z" || got.Calls != 2 || got.InputTokens != 120 || got.USD < 0.0119 || got.USD > 0.0121 {
		t.Fatalf("day 2026-03-09 wrong: %+v", got)
	}
	if got := agg.Buckets[2]; got.Start != "2026-03-10T00:00:00Z" || got.Calls != 2 || got.Unpriced != 1 || got.USD != 0.5 {
		t.Fatalf("day 2026-03-10 wrong: %+v", got)
	}
	if agg.Totals.Calls != 4 || agg.Totals.Unpriced != 1 {
		t.Fatalf("totals wrong: %+v", agg.Totals)
	}
	if len(agg.ByModel) != 2 {
		t.Fatalf("want 2 model rows, got %+v", agg.ByModel)
	}
	if agg.ByModel[0].Model != "vendor/b" || agg.ByModel[0].Provider != "litellm" || agg.ByModel[0].Calls != 2 || agg.ByModel[0].Unpriced != 1 {
		t.Fatalf("priced model should sort first: %+v", agg.ByModel[0])
	}
	if agg.ByModel[1].Model != "vendor/a" || agg.ByModel[1].InputTokens != 120 {
		t.Fatalf("second model row wrong: %+v", agg.ByModel[1])
	}
}

func TestAggregateSpendHourly(t *testing.T) {
	now := mustTime(t, "2026-03-10T12:30:00Z")
	records := []spendRecord{
		{Time: "2026-03-10T11:05:00Z", Provider: "openrouter", Model: "m", InputTokens: 10, OutputTokens: 5, USD: usdPtr(0.1), Priced: true},
		{Time: "2026-03-10T11:55:00Z", Provider: "openrouter", Model: "m", InputTokens: 10, OutputTokens: 5, USD: usdPtr(0.2), Priced: true},
		{Time: "2026-03-10T12:01:00Z", Provider: "openrouter", Model: "m", InputTokens: 10, OutputTokens: 5, USD: usdPtr(0.3), Priced: true},
	}
	agg := aggregateSpend(records, spendGranularityHour, 1, now)
	if len(agg.Buckets) != 24 {
		t.Fatalf("want 24 hour buckets, got %d", len(agg.Buckets))
	}
	if got, want := agg.Since, "2026-03-09T13:00:00Z"; got != want {
		t.Fatalf("since = %q, want %q", got, want)
	}
	if got := agg.Buckets[22]; got.Start != "2026-03-10T11:00:00Z" || got.Calls != 2 || got.USD < 0.2999 || got.USD > 0.3001 {
		t.Fatalf("hour 11 bucket wrong: %+v", got)
	}
	if got := agg.Buckets[23]; got.Start != "2026-03-10T12:00:00Z" || got.Calls != 1 {
		t.Fatalf("hour 12 bucket wrong: %+v", got)
	}
}

func TestAggregateSpendMonthBoundary(t *testing.T) {
	now := mustTime(t, "2026-03-01T04:00:00Z")
	records := []spendRecord{
		{Time: "2026-02-28T23:00:00Z", Provider: "openrouter", Model: "m", InputTokens: 1, OutputTokens: 1, USD: usdPtr(0.25), Priced: true},
		{Time: "2026-03-01T00:30:00Z", Provider: "openrouter", Model: "m", InputTokens: 1, OutputTokens: 1, USD: usdPtr(0.75), Priced: true},
	}
	agg := aggregateSpend(records, spendGranularityDay, 2, now)
	if len(agg.Buckets) != 2 {
		t.Fatalf("want 2 buckets, got %d", len(agg.Buckets))
	}
	if agg.Buckets[0].Start != "2026-02-28T00:00:00Z" || agg.Buckets[0].USD != 0.25 {
		t.Fatalf("february bucket wrong: %+v", agg.Buckets[0])
	}
	if agg.Buckets[1].Start != "2026-03-01T00:00:00Z" || agg.Buckets[1].USD != 0.75 {
		t.Fatalf("march bucket wrong: %+v", agg.Buckets[1])
	}
}

func TestAggregateSpendSkipsOutsideWindow(t *testing.T) {
	now := mustTime(t, "2026-03-10T12:00:00Z")
	records := []spendRecord{
		{Time: "2026-02-01T00:00:00Z", Provider: "openrouter", Model: "old", InputTokens: 1, OutputTokens: 1, USD: usdPtr(9), Priced: true},
		{Time: "2026-03-11T00:00:00Z", Provider: "openrouter", Model: "future", InputTokens: 1, OutputTokens: 1, USD: usdPtr(9), Priced: true},
		{Time: "invalid", Provider: "openrouter", Model: "bad", InputTokens: 1, OutputTokens: 1},
	}
	agg := aggregateSpend(records, spendGranularityDay, 2, now)
	if agg.Totals.Calls != 0 || agg.Totals.USD != 0 || len(agg.ByModel) != 0 {
		t.Fatalf("records outside the window must be ignored: %+v", agg)
	}
}

func TestSpendOverviewJSONFields(t *testing.T) {
	now := mustTime(t, "2026-03-10T12:00:00Z")
	agg := aggregateSpend([]spendRecord{
		{Time: "2026-03-10T10:00:00Z", Provider: "openrouter", Model: "m", InputTokens: 3, OutputTokens: 4, USD: usdPtr(0.02), Priced: true},
	}, spendGranularityDay, 1, now)
	data, err := json.Marshal(agg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded struct {
		Buckets []map[string]any `json:"buckets"`
		ByModel []map[string]any `json:"by_model"`
		Totals  map[string]any   `json:"totals"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Buckets) != 1 {
		t.Fatalf("buckets missing: %s", data)
	}
	for _, field := range []string{"start", "usd", "input_tokens", "output_tokens", "calls", "unpriced_calls"} {
		if _, ok := decoded.Buckets[0][field]; !ok {
			t.Fatalf("bucket field %q missing in %v", field, decoded.Buckets[0])
		}
	}
	if _, ok := decoded.ByModel[0]["provider"]; !ok {
		t.Fatalf("by_model provider missing: %v", decoded.ByModel[0])
	}
	if _, ok := decoded.Totals["usd"]; !ok {
		t.Fatalf("totals usd missing: %v", decoded.Totals)
	}
}

func TestRecordModelSpendMonthlyFiles(t *testing.T) {
	dir := t.TempDir()
	prev := spendConfigFile
	spendConfigFile = filepath.Join(dir, "config.json")
	defer func() { spendConfigFile = prev }()

	recordModelSpend(dir, spendRecord{Time: "2026-01-15T10:00:00Z", Provider: "openrouter", Model: "vendor/a", InputTokens: 10, OutputTokens: 2})
	recordModelSpend(dir, spendRecord{Time: "2026-02-01T10:00:00Z", Provider: "openrouter", Model: "vendor/a", InputTokens: 5, OutputTokens: 1})
	recordModelSpend(dir, spendRecord{Time: "2026-02-02T10:00:00Z", Provider: "openrouter", Model: "vendor/a"}) // no tokens: skipped

	for _, name := range []string{"spend-2026-01.jsonl", "spend-2026-02.jsonl"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("monthly file %s missing: %v", name, err)
		}
	}
	recs := readSpendRecords(dir)
	if len(recs) != 2 {
		t.Fatalf("want 2 records, got %d: %+v", len(recs), recs)
	}
	for _, rec := range recs {
		if rec.Priced || rec.USD != nil {
			t.Fatalf("unknown model must stay unpriced: %+v", rec)
		}
		if rec.Provider != "openrouter" || rec.Model != "vendor/a" {
			t.Fatalf("record identity lost: %+v", rec)
		}
	}
}

func TestRecordModelSpendPricesFromConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := `{"model_providers":[{"id":"p1","name":"OpenRouter","base_url":"https://openrouter.ai/api/v1","kind":"openrouter","models":[{"id":"vendor/priced","prompt_price":"0.000003","completion_price":"0.000015"}]}]}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	prev := spendConfigFile
	spendConfigFile = cfgPath
	defer func() { spendConfigFile = prev }()

	recordModelSpend(dir, spendRecord{Time: "2026-03-01T00:00:00Z", Model: "vendor/priced", InputTokens: 1000, OutputTokens: 500})
	recs := readSpendRecords(dir)
	if len(recs) != 1 {
		t.Fatalf("want 1 record, got %d", len(recs))
	}
	rec := recs[0]
	if !rec.Priced || rec.USD == nil {
		t.Fatalf("record should be priced: %+v", rec)
	}
	if *rec.USD < 0.01049 || *rec.USD > 0.01051 {
		t.Fatalf("usd = %v, want 0.0105", *rec.USD)
	}
	if rec.Provider != "openrouter" {
		t.Fatalf("provider = %q, want openrouter", rec.Provider)
	}
}

func TestSpendPriceIn(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := `{"model_providers":[
		{"id":"plain","base_url":"http://127.0.0.1:11434/v1","kind":"local","models":[{"id":"local-model"}]},
		{"id":"priced","base_url":"https://api.example.test/v1","kind":"litellm","models":[{"id":"m1","prompt_price":"0.000001","completion_price":"0.000002"}]}
	]}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cases := []struct {
		name       string
		model      string
		wantNil    bool
		wantPrompt string
		wantKind   string
		configPath string
	}{
		{name: "priced model", model: "m1", wantPrompt: "0.000001", wantKind: "litellm", configPath: cfgPath},
		{name: "model without prices", model: "local-model", wantPrompt: "", wantKind: "local", configPath: cfgPath},
		{name: "unknown model", model: "nope", wantNil: true, configPath: cfgPath},
		{name: "missing config file", model: "m1", wantNil: true, configPath: filepath.Join(dir, "absent.json")},
		{name: "empty model", model: "", wantNil: true, configPath: cfgPath},
	}
	for _, tc := range cases {
		got, kind := spendPriceIn(tc.configPath, tc.model)
		if tc.wantNil {
			if got != nil {
				t.Fatalf("%s: want no model, got %+v", tc.name, got)
			}
			continue
		}
		if got == nil {
			t.Fatalf("%s: want model, got nil", tc.name)
		}
		if got.PromptPrice != tc.wantPrompt {
			t.Fatalf("%s: prompt price = %q, want %q", tc.name, got.PromptPrice, tc.wantPrompt)
		}
		if kind != tc.wantKind {
			t.Fatalf("%s: kind = %q, want %q", tc.name, kind, tc.wantKind)
		}
	}
}

func TestRecordBudgetSnapshotIn(t *testing.T) {
	dir := t.TempDir()
	recordBudgetSnapshotIn(dir, spendBudgetRecord{Provider: "p1", Kind: "openai"})
	if files := ledgerFiles(dir, budgetRecordPrefix); len(files) != 0 {
		t.Fatalf("empty snapshot must not write a file: %v", files)
	}
	recordBudgetSnapshotIn(dir, spendBudgetRecord{Time: "2026-04-01T00:00:00Z", Provider: "p1", Kind: "openrouter", RemainingUSD: usdPtr(3.5), AccountRemainingUSD: usdPtr(20)})
	recs := readBudgetRecords(dir)
	if len(recs) != 1 || recs[0].RemainingUSD == nil || *recs[0].RemainingUSD != 3.5 || recs[0].AccountRemainingUSD == nil || *recs[0].AccountRemainingUSD != 20 {
		t.Fatalf("snapshot roundtrip failed: %+v", recs)
	}
}

func TestSpendOverviewForFiltersBudgetHistory(t *testing.T) {
	dir := t.TempDir()
	prev := spendConfigFile
	spendConfigFile = filepath.Join(dir, "config.json")
	defer func() { spendConfigFile = prev }()

	recordModelSpend(dir, spendRecord{Time: "2026-05-10T08:00:00Z", Provider: "openrouter", Model: "m", InputTokens: 10, OutputTokens: 2})
	recordBudgetSnapshotIn(dir, spendBudgetRecord{Time: "2026-01-01T00:00:00Z", Provider: "p1", RemainingUSD: usdPtr(1)})
	recordBudgetSnapshotIn(dir, spendBudgetRecord{Time: "2026-05-10T09:00:00Z", Provider: "p1", RemainingUSD: usdPtr(2)})

	now := mustTime(t, "2026-05-10T12:00:00Z")
	out := spendOverviewFor(dir, spendGranularityDay, 3, now)
	if len(out.Budget) != 1 {
		t.Fatalf("want only the in-window snapshot, got %+v", out.Budget)
	}
	if out.Budget[0].RemainingUSD == nil || *out.Budget[0].RemainingUSD != 2 {
		t.Fatalf("wrong snapshot kept: %+v", out.Budget[0])
	}
	if out.Totals.Calls != 1 || len(out.ByModel) != 1 {
		t.Fatalf("spend records not aggregated: %+v", out)
	}
}

func TestParseSpendDays(t *testing.T) {
	cases := []struct {
		gran string
		raw  string
		want int
	}{
		{gran: spendGranularityDay, raw: "", want: 30},
		{gran: spendGranularityDay, raw: "7", want: 7},
		{gran: spendGranularityDay, raw: "9999", want: spendMaxDays},
		{gran: spendGranularityDay, raw: "-4", want: 30},
		{gran: spendGranularityHour, raw: "", want: 3},
		{gran: spendGranularityHour, raw: "2", want: 2},
		{gran: spendGranularityHour, raw: "30", want: spendMaxHourDays},
		{gran: spendGranularityHour, raw: "abc", want: 3},
	}
	for _, tc := range cases {
		if got := parseSpendDays(tc.gran, tc.raw); got != tc.want {
			t.Fatalf("parseSpendDays(%q, %q) = %d, want %d", tc.gran, tc.raw, got, tc.want)
		}
	}
}
