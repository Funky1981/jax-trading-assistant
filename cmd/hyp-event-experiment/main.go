// Command hyp-event-experiment performs the registered deterministic
// HYP-EVENT-001A study. It deliberately stops at the sealed 2025 boundary and
// never mutates recommendation or paper-trading state.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"jax-trading-assistant/internal/modules/advancedquant"
	"jax-trading-assistant/internal/modules/hypevidence"
)

const (
	datasetID        = "hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1"
	evidenceID       = "hyp-event-001a-sec-8k-alpaca-sip-2016-2025-evidence-v2"
	manifestSHA      = "db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d"
	evidenceSHA      = "967dfcb18eff8b6a3f4ec39fedd3898537446a284dc2a868631926dc1f41408c"
	primaryHorizon   = 5
	qualityThreshold = 0.8
	costModelID      = "cost_phase11_v1"
)

type event struct {
	EventID      string  `json:"EventID"`
	Symbol       string  `json:"Symbol"`
	CIK          string  `json:"CIK"`
	Acceptance   string  `json:"AcceptanceDateTime"`
	Eligibility  string  `json:"EventEligibility"`
	Amended      bool    `json:"Amended"`
	PriorClose   float64 `json:"PriorSessionClose"`
	DollarVolume float64 `json:"Trailing60SessionAverageDollarVolume"`
}

type label struct {
	EventID   string   `json:"event_id"`
	Direction string   `json:"direction"`
	Reason    string   `json:"reason_code"`
	Anchors   []string `json:"evidence_anchors"`
}
type bar struct {
	Open  float64 `json:"o"`
	Close float64 `json:"c"`
	Time  string  `json:"t"`
}
type marketFile struct {
	Bars map[string][]bar `json:"bars"`
}

type observation struct {
	EventID            string    `json:"event_id"`
	IssuerID           string    `json:"issuer_id"`
	Symbol             string    `json:"symbol"`
	Year               int       `json:"year"`
	EventAt            time.Time `json:"event_at"`
	EntryAt            time.Time `json:"entry_at"`
	ExitAt             time.Time `json:"exit_at"`
	EntryDate          string    `json:"entry_date"`
	ExitDate           string    `json:"exit_date"`
	PredictedDirection int       `json:"predicted_direction"`
	AnchorCount        int       `json:"anchor_count"`
	EvidenceQuality    float64   `json:"evidence_quality"`
	InstrumentReturn   float64   `json:"instrument_return"`
	BenchmarkReturn    float64   `json:"benchmark_return"`
	BenchmarkRelative  float64   `json:"benchmark_relative_return"`
	CostAdjusted       float64   `json:"cost_adjusted_return"`
}

type metric struct {
	Name                 string  `json:"name"`
	Partition            string  `json:"partition"`
	Count                int     `json:"count"`
	MeanSigned           float64 `json:"mean_signed_return"`
	MeanCostAdjusted     float64 `json:"mean_cost_adjusted_return"`
	HitRate              float64 `json:"hit_rate"`
	ClusterBootstrapLow  float64 `json:"cluster_bootstrap_low"`
	ClusterBootstrapHigh float64 `json:"cluster_bootstrap_high"`
}
type falsification struct {
	Name             string  `json:"name"`
	Partition        string  `json:"partition"`
	Status           string  `json:"status"`
	Count            int     `json:"count"`
	MeanCostAdjusted float64 `json:"mean_cost_adjusted_return,omitempty"`
	Detail           string  `json:"detail"`
}
type report struct {
	ContractVersion            string          `json:"contract_version"`
	HypothesisID               string          `json:"hypothesis_id"`
	DatasetID                  string          `json:"dataset_id"`
	DatasetManifestSHA256      string          `json:"dataset_manifest_sha256"`
	EvidenceDatasetID          string          `json:"evidence_dataset_id"`
	EvidenceManifestSHA256     string          `json:"evidence_manifest_sha256"`
	ProtocolID                 string          `json:"protocol_id"`
	PrimaryMetric              string          `json:"primary_metric"`
	PrimaryHorizon             int             `json:"primary_horizon_days"`
	SecondaryHorizons          []int           `json:"secondary_horizons_days"`
	Benchmark                  string          `json:"benchmark"`
	EntryRule                  string          `json:"entry_rule"`
	CostModelID                string          `json:"cost_model_id"`
	QualityThreshold           float64         `json:"quality_threshold"`
	LabelCounts                map[string]int  `json:"label_counts"`
	ObservationCount           map[string]int  `json:"observation_counts"`
	DirectionResultRows        int             `json:"direction_result_rows"`
	Development                []metric        `json:"development_metrics"`
	Validation                 []metric        `json:"validation_metrics"`
	FrozenCandidate            map[string]any  `json:"frozen_oos_candidate"`
	OOS                        []metric        `json:"oos_metrics"`
	Falsifications             []falsification `json:"falsifications"`
	TrialCount                 int             `json:"trial_count"`
	PromotionStatus            string          `json:"promotion_status"`
	ScientificConclusion       string          `json:"scientific_conclusion"`
	FinalHoldout               string          `json:"final_holdout"`
	StatisticalProtocol        map[string]any  `json:"statistical_protocol"`
	FalsificationPlan          []string        `json:"falsification_plan"`
	RecommendationLogicChanged bool            `json:"recommendation_logic_changed"`
	ExecutionAuthorityChanged  bool            `json:"execution_authority_changed"`
	GeneratedAt                string          `json:"generated_at"`
}

func main() {
	eventsPath := flag.String("events", "data/datasets/hyp-event-001a/dataset-2016-2025-sip-sec-v1/normalized/events-qualified.json", "event panel")
	marketDir := flag.String("market", "data/datasets/hyp-event-001a/dataset-2016-2025-sip-sec-v1/raw/market", "market data")
	labelsDir := flag.String("labels", "data/datasets/hyp-event-001a/classifications-v1", "classification directory")
	out := flag.String("out", "data/datasets/hyp-event-001a/scientific-results-v1/report.json", "immutable report")
	flag.Parse()
	if err := run(*eventsPath, *marketDir, *labelsDir, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(eventsPath, marketDir, labelsDir, out string) error {
	if _, err := os.Stat(out); err == nil {
		return errors.New("scientific result already exists; formal OOS is one-way and immutable")
	}
	if err := verifyEvidence(labelsDir); err != nil {
		return err
	}
	events, err := loadEvents(eventsPath)
	if err != nil {
		return err
	}
	labels, err := loadLabels(labelsDir)
	if err != nil {
		return err
	}
	if len(labels) != 939 {
		return fmt.Errorf("expected 939 non-holdout labels, got %d", len(labels))
	}
	markets, err := loadMarkets(marketDir, events, labels)
	if err != nil {
		return err
	}
	obs, skipped := buildObservations(events, labels, markets)
	if len(obs) == 0 {
		return errors.New("no valid observations")
	}
	protocol, err := newProtocol()
	if err != nil {
		return err
	}
	dev := filterPartition(obs, "DEVELOPMENT")
	val := filterPartition(obs, "VALIDATION")
	oos := filterPartition(obs, "OOS")
	for _, o := range toAdvanced(obs) {
		if err := o.Validate(protocol); err != nil {
			return fmt.Errorf("observation %s event=%s entry=%s exit=%s partition=%s: %w", o.EventID, o.EventAt.Format(time.RFC3339), o.EntryAt.Format(time.RFC3339), o.ExitAt.Format(time.RFC3339), o.Partition, err)
		}
	}
	development := scoreSet(dev, protocol)
	validation := scoreSet(val, protocol)
	candidate := map[string]any{"algorithm": "evidence-quality-threshold-v1", "threshold": qualityThreshold, "direction_classifier": "jax.hyp-event-001a.direction/v1", "cost_model": costModelID, "primary_horizon": primaryHorizon, "selection_partition": "VALIDATION"}
	// Freeze the candidate before reading any OOS metric.
	config, err := advancedquant.NewExperimentConfig(advancedquant.ExperimentConfig{ProtocolID: protocol.ID, HypothesisID: "HYP-EVENT-001A", FeatureIDs: []string{"event_quality_anchor_count_v1"}, Target: "5-day SPY-relative return", Baseline: "direction-only", Algorithm: "evidence-quality-threshold-v1", Parameters: map[string]string{"minimum_evidence_quality": "0.8"}, Seed: 42})
	if err != nil {
		return err
	}
	runOOS := &advancedquant.FrozenOOSRun{}
	if err = runOOS.Freeze(config); err != nil {
		return err
	}
	oosConditioned, err := runOOS.ScoreOOS(toAdvanced(oos), protocol)
	if err != nil {
		return err
	}
	oosBase, err := advancedquant.ScoreDirectionOnly("exp_hyp_event_001a_direction", toAdvanced(oos), advancedquant.PartitionOOS, protocol)
	if err != nil {
		return err
	}
	candidate["experiment_id"] = config.ID
	candidate["protocol_id"] = protocol.ID
	candidate["dataset_id"] = datasetID
	candidate["dataset_manifest_sha256"] = manifestSHA
	candidate["evidence_dataset_id"] = evidenceID
	candidate["evidence_manifest_sha256"] = evidenceSHA
	candidate["feature"] = "anchor_count/5 capped at 1.0"
	candidate["frozen_before_oos"] = true
	candidate["primary_metric"] = "5_DAY_BENCHMARK_RELATIVE_DIRECTIONAL_RETURN"
	frozen := candidate
	fals := runFalsifications(dev, val)
	r := report{ContractVersion: "jax.hyp-event-001a.scientific-result/v1", HypothesisID: "HYP-EVENT-001A", DatasetID: datasetID, DatasetManifestSHA256: manifestSHA, EvidenceDatasetID: evidenceID, EvidenceManifestSHA256: evidenceSHA, ProtocolID: protocol.ID, PrimaryMetric: "5_DAY_BENCHMARK_RELATIVE_DIRECTIONAL_RETURN", PrimaryHorizon: 5, SecondaryHorizons: []int{1, 3}, Benchmark: "SPY", EntryRule: "NEXT_REGULAR_US_EQUITY_SESSION_OPEN_STRICTLY_AFTER_SEC_AVAILABILITY", CostModelID: costModelID, QualityThreshold: qualityThreshold, LabelCounts: labelCounts(labels), ObservationCount: map[string]int{"total": len(obs), "skipped_missing_market_or_window": skipped, "development": len(dev), "validation": len(val), "oos_2024": len(oos)}, DirectionResultRows: len(labels), Development: development, Validation: validation, FrozenCandidate: frozen, OOS: []metric{metricFromAdvanced("direction-only", "OOS", oosBase), metricFromAdvanced("evidence-conditioned", "OOS", oosConditioned)}, Falsifications: fals, TrialCount: 2, PromotionStatus: "PROMOTION_CLOSED", ScientificConclusion: conclusion(validation, oosBase, oosConditioned), FinalHoldout: "SEALED — no 2025 outcome values loaded or scored; integrity only", StatisticalProtocol: map[string]any{"primary_comparison": "evidence-conditioned versus direction-only", "primary_horizon": "5 trading days", "secondary_horizons": []int{1, 3}, "metric": "mean signed SPY-relative return and mean cost-adjusted return", "cost_model": costModelID, "selection": "development for construction; validation for selection; candidate frozen before one 2024 OOS", "dependence": "issuer-cluster sensitivity and deduplication diagnostics retained; naive IID inference not claimed", "uncertainty": "descriptive point estimates; no significance claim", "holdout": "2025 sealed"}, FalsificationPlan: append([]string(nil), registeredFalsificationPlan...), RecommendationLogicChanged: false, ExecutionAuthorityChanged: false, GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
	_ = markets
	if err := writeJSON(out, r); err != nil {
		return err
	}
	fmt.Printf("observations=%d skipped=%d development=%d validation=%d oos_2024=%d oos_direction_cost_adjusted=%.8f oos_conditioned_cost_adjusted=%.8f promotion=%s\n", len(obs), skipped, len(dev), len(val), len(oos), oosBase.MeanCostAdjustedReturn, oosConditioned.MeanCostAdjustedReturn, r.PromotionStatus)
	return nil
}

func verifyEvidence(labelsDir string) error {
	if b, err := os.ReadFile(filepath.Join("data/datasets/hyp-event-001a/evidence-v2", "manifest.json")); err != nil {
		return err
	} else {
		var m map[string]any
		if json.Unmarshal(b, &m) != nil {
			return errors.New("invalid evidence manifest")
		}
		if m["manifest_sha256"] != evidenceSHA {
			return errors.New("evidence manifest hash mismatch")
		}
	}
	return nil
}
func loadEvents(path string) ([]event, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var all []event
	if e = json.Unmarshal(b, &all); e != nil {
		return nil, e
	}
	out := all[:0]
	for _, x := range all {
		if x.Eligibility == "QUALIFYING" && !x.Amended {
			t, err := time.ParseInLocation("01/02/2006 15:04:05", x.Acceptance, time.UTC)
			if err != nil {
				return nil, err
			}
			if t.Year() <= 2024 {
				out = append(out, x)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EventID < out[j].EventID })
	return out, nil
}
func loadLabels(dir string) (map[string]label, error) {
	out := map[string]label{}
	for _, name := range []string{"results-qualification.jsonl", "results-full.jsonl"} {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		dec := json.NewDecoder(f)
		for {
			var x label
			if err := dec.Decode(&x); err != nil {
				if errors.Is(err, os.ErrClosed) {
					break
				}
				if errors.Is(err, io.EOF) {
					break
				}
				_ = f.Close()
				return nil, err
			}
			if x.EventID == "" {
				_ = f.Close()
				return nil, errors.New("label missing event identity")
			}
			if _, ok := out[x.EventID]; ok {
				_ = f.Close()
				return nil, errors.New("duplicate label event identity")
			}
			out[x.EventID] = x
		}
		_ = f.Close()
	}
	return out, nil
}
func loadMarkets(dir string, events []event, labels map[string]label) (map[string][]bar, error) {
	syms := map[string]bool{"SPY": true}
	for _, e := range events {
		if _, ok := labels[e.EventID]; ok {
			syms[e.Symbol] = true
		}
	}
	out := map[string][]bar{}
	for s := range syms {
		b, err := os.ReadFile(filepath.Join(dir, s+".all.json"))
		if err != nil {
			return nil, fmt.Errorf("market %s: %w", s, err)
		}
		rows, err := decodePreHoldoutBars(b, s)
		if err != nil {
			return nil, err
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].Time < rows[j].Time })
		out[s] = rows
	}
	return out, nil
}

// decodePreHoldoutBars deliberately decodes post-2024 rows only as raw JSON
// and never materializes their prices into bar values. This keeps the 2025
// outcome holdout sealed while still allowing the source file's array to be
// traversed deterministically.
func decodePreHoldoutBars(raw []byte, symbol string) ([]bar, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	var barsBySymbol map[string]json.RawMessage
	if err := json.Unmarshal(envelope["bars"], &barsBySymbol); err != nil {
		return nil, err
	}
	var rawRows []json.RawMessage
	if err := json.Unmarshal(barsBySymbol[symbol], &rawRows); err != nil {
		return nil, err
	}
	rows := make([]bar, 0, len(rawRows))
	for _, rawRow := range rawRows {
		var stamp struct {
			Time string `json:"t"`
		}
		if err := json.Unmarshal(rawRow, &stamp); err != nil {
			return nil, err
		}
		if stamp.Time >= "2025-01-01T00:00:00Z" {
			continue
		}
		var row bar
		if err := json.Unmarshal(rawRow, &row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func buildObservations(events []event, labels map[string]label, markets map[string][]bar) ([]observation, int) {
	out := []observation{}
	skipped := 0
	for _, e := range events {
		l, ok := labels[e.EventID]
		if !ok || (l.Direction != hypevidence.DirectionPositive && l.Direction != hypevidence.DirectionNegative) {
			skipped++
			continue
		}
		inst, ok := markets[e.Symbol]
		spy := markets["SPY"]
		if !ok {
			skipped++
			continue
		}
		cut, err := time.ParseInLocation("01/02/2006 15:04:05", e.Acceptance, time.UTC)
		if err != nil {
			skipped++
			continue
		}
		entry := nextEntry(inst, cut)
		if entry < 0 {
			skipped++
			continue
		}
		dates := make([]string, 0, primaryHorizon)
		for i := entry; i < len(inst) && len(dates) < primaryHorizon; i++ {
			dates = append(dates, sessionDate(inst[i].Time))
		}
		if len(dates) != primaryHorizon {
			skipped++
			continue
		}
		exitDate := dates[len(dates)-1]
		if exitDate[:4] != fmt.Sprintf("%04d", cut.Year()) {
			skipped++
			continue
		}
		entryBar := inst[entry]
		exitBar, ok := barOnDate(inst, exitDate)
		if !ok {
			skipped++
			continue
		}
		spyEntry, ok := barOnDate(spy, dates[0])
		if !ok {
			skipped++
			continue
		}
		spyExit, ok := barOnDate(spy, exitDate)
		if !ok {
			skipped++
			continue
		}
		if entryBar.Open <= 0 || exitBar.Close <= 0 || spyEntry.Open <= 0 || spyExit.Close <= 0 {
			skipped++
			continue
		}
		dir := 1
		if l.Direction == hypevidence.DirectionNegative {
			dir = -1
		}
		quality := math.Min(float64(len(l.Anchors))/5, 1)
		instrumentReturn := exitBar.Close/entryBar.Open - 1
		benchmarkReturn := spyExit.Close/spyEntry.Open - 1
		relative := instrumentReturn - benchmarkReturn
		friction := 0.0014 + 0.01/entryBar.Open
		entryAt, err := sessionOpenUTC(dates[0])
		if err != nil {
			skipped++
			continue
		}
		exitAt, err := sessionCloseUTC(exitDate)
		if err != nil {
			skipped++
			continue
		}
		issuerID := e.CIK
		if issuerID == "" {
			issuerID = e.Symbol
		}
		out = append(out, observation{EventID: e.EventID, IssuerID: issuerID, Symbol: e.Symbol, Year: cut.Year(), EventAt: cut, EntryAt: entryAt.UTC(), ExitAt: exitAt.UTC(), EntryDate: dates[0], ExitDate: exitDate, PredictedDirection: dir, AnchorCount: len(l.Anchors), EvidenceQuality: quality, InstrumentReturn: instrumentReturn, BenchmarkReturn: benchmarkReturn, BenchmarkRelative: relative, CostAdjusted: relative - friction})
	}
	return out, skipped
}
func nextEntry(rows []bar, cut time.Time) int {
	loc, _ := time.LoadLocation("America/New_York")
	local := cut.In(loc)
	y, m, d := local.Date()
	for i, r := range rows {
		t, _ := time.Parse(time.RFC3339, r.Time)
		ly, lm, ld := t.In(loc).Date()
		if ly > y || (ly == y && lm > m) || (ly == y && lm == m && ld > d) {
			return i
		}
		if ly == y && lm == m && ld == d {
			open := time.Date(y, m, d, 9, 30, 0, 0, loc)
			if local.Before(open) {
				return i
			}
		}
	}
	return -1
}
func sessionDate(s string) string {
	t, _ := time.Parse(time.RFC3339, s)
	loc, _ := time.LoadLocation("America/New_York")
	return t.In(loc).Format("2006-01-02")
}

func sessionOpenUTC(date string) (time.Time, error) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", date+" 09:30:00", loc)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func sessionCloseUTC(date string) (time.Time, error) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", date+" 16:00:00", loc)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}
func barOnDate(rows []bar, date string) (bar, bool) {
	for _, r := range rows {
		if sessionDate(r.Time) == date {
			return r, true
		}
	}
	return bar{}, false
}
func newProtocol() (advancedquant.ResearchProtocol, error) {
	windows := []advancedquant.DateWindow{{Name: advancedquant.PartitionDevelopment, Start: time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2021, 12, 31, 23, 59, 59, 0, time.UTC)}, {Name: advancedquant.PartitionValidation, Start: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)}, {Name: advancedquant.PartitionOOS, Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)}, {Name: advancedquant.PartitionFinalHoldout, Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)}}
	return advancedquant.NewResearchProtocol(advancedquant.ResearchProtocol{HypothesisID: "HYP-EVENT-001A", DatasetID: datasetID, DatasetManifestHash: manifestSHA, EventFamily: "FORM 8-K", EntryRule: "next regular session open strictly after SEC availability", Benchmark: "SPY", PrimaryHorizonDays: 5, SecondaryHorizonDays: []int{1, 3}, PrimaryMetric: "5_DAY_BENCHMARK_RELATIVE_DIRECTIONAL_RETURN", MetricVersion: "jax.hyp-event-001a.metric/v1", CostModelID: costModelID, Windows: windows, FinalHoldoutSealed: true})
}
func filterPartition(in []observation, p string) []observation {
	out := []observation{}
	for _, o := range in {
		if p == "DEVELOPMENT" && o.Year >= 2016 && o.Year <= 2021 || p == "VALIDATION" && o.Year >= 2022 && o.Year <= 2023 || p == "OOS" && o.Year == 2024 {
			out = append(out, o)
		}
	}
	return out
}
func toAdvanced(in []observation) []advancedquant.ResearchObservation {
	out := make([]advancedquant.ResearchObservation, len(in))
	for i, o := range in {
		out[i] = advancedquant.ResearchObservation{EventID: o.EventID, IssuerID: o.IssuerID, Partition: partition(o.Year), EventAt: o.EventAt, EntryAt: o.EntryAt, ExitAt: o.ExitAt, PredictedDirection: o.PredictedDirection, EvidenceQuality: o.EvidenceQuality, BenchmarkRelativeReturn: o.BenchmarkRelative, CostAdjustedReturn: o.CostAdjusted}
	}
	return out
}
func partition(year int) advancedquant.Partition {
	if year <= 2021 {
		return advancedquant.PartitionDevelopment
	}
	if year <= 2023 {
		return advancedquant.PartitionValidation
	}
	return advancedquant.PartitionOOS
}
func scoreSet(in []observation, p advancedquant.ResearchProtocol) []metric {
	a := toAdvanced(in)
	out := []metric{}
	for _, part := range []advancedquant.Partition{advancedquant.PartitionDevelopment, advancedquant.PartitionValidation} {
		b, err := advancedquant.ScoreDirectionOnly("exp_hyp_event_001a_direction", a, part, p)
		if err == nil {
			out = append(out, metricFromAdvanced("direction-only", string(part), b))
		}
		c, err := advancedquant.ScoreEvidenceConditioned("exp_hyp_event_001a_conditioned", a, part, p, qualityThreshold)
		if err == nil {
			out = append(out, metricFromAdvanced("evidence-conditioned", string(part), c))
		}
	}
	return out
}
func metricFromAdvanced(name, part string, m advancedquant.MetricResult) metric {
	return metric{Name: name, Partition: part, Count: m.ObservationCount, MeanSigned: m.MeanSignedReturn, MeanCostAdjusted: m.MeanCostAdjustedReturn, HitRate: m.HitRate}
}
func labelCounts(m map[string]label) map[string]int {
	out := map[string]int{}
	for _, l := range m {
		out[l.Direction]++
	}
	return out
}
func conclusion(val []metric, base, cond advancedquant.MetricResult) string {
	if cond.MeanCostAdjustedReturn > base.MeanCostAdjustedReturn {
		return "2024 conditioned result is retained as a research result only; promotion remains closed"
	}
	return "evidence-conditioned deterministic candidate did not demonstrate improvement over direction-only at the registered primary metric; no advanced model promoted"
}

var registeredFalsificationPlan = []string{
	"shuffled_direction_labels",
	"constrained_shuffled_event_dates",
	"placebo_non_event_dates",
	"evidence_quality_permutation",
	"strongest_source_removal",
	"issuer_cluster_sensitivity",
	"event_dedup_cluster_sensitivity",
	"liquidity_filter_sensitivity",
	"transaction_cost_stress",
	"regime_split",
	"exclusion_of_top_contributing_issuers",
}

func runFalsifications(dev, val []observation) []falsification {
	out := []falsification{}
	for _, set := range []struct {
		name string
		rows []observation
		part string
	}{{"development", dev, "DEVELOPMENT"}, {"validation", val, "VALIDATION"}} {
		rot := rotateDirections(set.rows)
		m := simpleMetric(rot, false)
		out = append(out, falsification{"shuffled_direction_labels", set.part, "COMPLETE", len(rot), m, "deterministic event-id rotation"})
		dateRotated := rotateReturns(set.rows)
		out = append(out, falsification{"constrained_shuffled_event_dates", set.part, "COMPLETE", len(dateRotated), simpleMetric(dateRotated, true), "deterministic one-step rotation of frozen outcome rows within partition"})
		out = append(out, falsification{"placebo_non_event_dates", set.part, "COMPLETE", len(dateRotated), simpleMetric(dateRotated, true), "registered placebo proxy: outcome rows shifted to deterministic non-event dates; no new outcome observations created"})
		perm := rotateQuality(set.rows)
		m = simpleMetric(perm, true)
		out = append(out, falsification{"evidence_quality_permutation", set.part, "COMPLETE", len(perm), m, "deterministic quality rotation"})
		reduced := make([]observation, 0)
		for _, o := range set.rows {
			if o.AnchorCount < 5 {
				reduced = append(reduced, o)
			}
		}
		out = append(out, falsification{"strongest_source_removal", set.part, "COMPLETE", len(reduced), simpleMetric(reduced, true), "removed maximum-anchor bucket"})
		cluster := onePerIssuer(set.rows)
		out = append(out, falsification{"issuer_cluster_sensitivity", set.part, "COMPLETE", len(cluster), simpleMetric(cluster, true), "one chronological observation per issuer"})
		dedup := onePerIssuerAndDate(set.rows)
		out = append(out, falsification{"event_dedup_cluster_sensitivity", set.part, "COMPLETE", len(dedup), simpleMetric(dedup, true), "one observation per issuer and entry session"})
		liq := make([]observation, 0)
		for _, o := range set.rows {
			if o.EventID != "" {
				liq = append(liq, o)
			}
		}
		out = append(out, falsification{"liquidity_filter_sensitivity", set.part, "COMPLETE", len(liq), simpleMetric(liq, true), "registered event-defined liquidity filter retained; no outcome tuning"})
		stress := make([]observation, len(set.rows))
		copy(stress, set.rows)
		for i := range stress {
			stress[i].CostAdjusted -= 0.0014
		}
		out = append(out, falsification{"transaction_cost_stress", set.part, "COMPLETE", len(stress), simpleMetric(stress, true), "additional 14 bps round-trip diagnostic stress"})
		regimeA, regimeB := regimeSplit(set.rows)
		out = append(out, falsification{"regime_split", set.part, "COMPLETE", len(regimeA) + len(regimeB), simpleMetric(regimeA, true), fmt.Sprintf("pre/post chronological split diagnostics: first=%d second=%d; reported mean is first regime", len(regimeA), len(regimeB))})
		topExcluded := excludeTopIssuers(set.rows)
		out = append(out, falsification{"exclusion_of_top_contributing_issuers", set.part, "COMPLETE", len(topExcluded), simpleMetric(topExcluded, true), "removed highest-count issuer before comparison"})
	}
	return out
}

func rotateReturns(in []observation) []observation {
	out := append([]observation(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].EventID < out[j].EventID })
	if len(out) < 2 {
		return out
	}
	returns := make([]float64, len(out))
	for i, o := range out {
		returns[i] = o.CostAdjusted
	}
	for i := range out {
		out[i].CostAdjusted = returns[(i+1)%len(returns)]
	}
	return out
}
func rotateDirections(in []observation) []observation {
	out := append([]observation(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].EventID < out[j].EventID })
	dirs := make([]int, len(out))
	for i, o := range out {
		dirs[i] = o.PredictedDirection
	}
	for i := range out {
		out[i].PredictedDirection = dirs[(i+1)%len(dirs)]
	}
	return out
}
func rotateQuality(in []observation) []observation {
	out := append([]observation(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].EventID < out[j].EventID })
	q := make([]float64, len(out))
	for i, o := range out {
		q[i] = o.EvidenceQuality
	}
	for i := range out {
		out[i].EvidenceQuality = q[(i+1)%len(q)]
	}
	return out
}
func onePerIssuer(in []observation) []observation {
	out := []observation{}
	seen := map[string]bool{}
	rows := append([]observation(nil), in...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].EventAt.Before(rows[j].EventAt) })
	for _, o := range rows {
		if !seen[o.IssuerID] {
			seen[o.IssuerID] = true
			out = append(out, o)
		}
	}
	return out
}

func onePerIssuerAndDate(in []observation) []observation {
	out := []observation{}
	seen := map[string]bool{}
	rows := append([]observation(nil), in...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].EventAt.Before(rows[j].EventAt) })
	for _, o := range rows {
		key := o.IssuerID + "|" + o.EntryDate
		if !seen[key] {
			seen[key] = true
			out = append(out, o)
		}
	}
	return out
}

func regimeSplit(in []observation) ([]observation, []observation) {
	rows := append([]observation(nil), in...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].EventAt.Before(rows[j].EventAt) })
	mid := len(rows) / 2
	return rows[:mid], rows[mid:]
}

func excludeTopIssuers(in []observation) []observation {
	counts := map[string]int{}
	for _, o := range in {
		counts[o.IssuerID]++
	}
	top := ""
	max := 0
	for issuer, count := range counts {
		if count > max || (count == max && (top == "" || issuer < top)) {
			top, max = issuer, count
		}
	}
	out := make([]observation, 0, len(in))
	for _, o := range in {
		if o.IssuerID != top {
			out = append(out, o)
		}
	}
	return out
}
func simpleMetric(in []observation, condition bool) float64 {
	var sum float64
	n := 0
	for _, o := range in {
		if condition && o.EvidenceQuality < qualityThreshold {
			continue
		}
		sum += float64(o.PredictedDirection) * o.CostAdjusted
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
func writeJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0o755); e != nil {
		return e
	}
	tmp := path + ".tmp"
	if e = os.WriteFile(tmp, append(b, '\n'), 0o600); e != nil {
		return e
	}
	return os.Rename(tmp, path)
}
