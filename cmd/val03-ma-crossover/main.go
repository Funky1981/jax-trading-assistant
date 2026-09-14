// Command val03-ma-crossover executes the frozen VAL-03 ma_crossover_v1
// historical validation from hash-bound, already-acquired bar evidence.
// It has no broker, order, fill, approval, or trading-state dependency.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"jax-trading-assistant/libs/marketdata"
)

const (
	manifestPath  = "Docs/validation/manifests/VAL-02-ma_crossover_v1-PREREGISTRATION.json"
	readyPath     = "Docs/validation/results/VAL-03C-DATASET-READINESS.json"
	barFamilyPath = "Docs/validation/results/VAL-03B-DATASET-READINESS.json"
	integrityPath = "Docs/validation/results/VAL-03-HYP-MA-001-INTEGRITY-REVIEW.json"
	rawRoot       = ".runtime/val03b/raw"
	dataStart     = "2016-01-01"
	dataEnd       = "2024-12-31"
	holdoutStart  = "2025-01-01"
	bootstrapN    = 10000
	minLookback   = 199
)

var symbols = []string{"SPY", "QQQ", "IWM", "DIA", "XLK", "XLF", "XLE", "TLT", "GLD"}

type bar struct {
	Date                           string `json:"date"`
	Open, High, Low, Close, Volume float64
}
type providerBar struct {
	Open      float64 `json:"o"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
	Close     float64 `json:"c"`
	Volume    float64 `json:"v"`
	Timestamp string  `json:"t"`
}
type providerPayload struct {
	Bars map[string][]providerBar `json:"bars"`
}
type familyRef struct {
	Instrument       string   `json:"instrument"`
	Adjustment       string   `json:"adjustment"`
	RawPayloadSHA256 []string `json:"raw_payload_sha256"`
}
type readinessInput struct {
	Families []familyRef `json:"families"`
}

type instrumentData struct {
	Raw, Split, Detector map[string]bar
	Dates                []string
	Factor               map[string]float64
	Boundaries           map[string]bool
}
type signal struct {
	Date, Type                                            string
	Confidence, SMA20, SMA50, SMA200, ATR14, Stop, Target float64
}
type episode struct {
	ID, Instrument, SignalDate, EntryDate, ExitDate string
	Duration                                        int
	GrossReturn, NetReturn, StressNetReturn         float64
	PlaceboNetReturn                                *float64
	Direction, Regime, Theme                        string
	Year                                            int
}
type partitionResult struct {
	Partition, Start, End                                                string
	Episodes                                                             []episode `json:"episodes"`
	SignalObservations, ActionableLong, BearishDiagnostics               int
	Abstentions                                                          map[string]int
	MeanGross, MeanNet, MeanStress, MeanPairedDifference                 float64
	MatchedPairs                                                         int
	BootstrapLow, BootstrapHigh, PairedBootstrapLow, PairedBootstrapHigh float64
	Blocks, Instruments                                                  int
	MaxInstrumentShare                                                   float64
	RegimeSlices                                                         int
	CalendarRegimeCells                                                  []string                        `json:"calendar_regime_cells,omitempty"`
	SecondaryDiagnostics                                                 secondaryDirectionalDiagnostics `json:"secondary_diagnostics,omitempty"`
	SecondaryAbstentions                                                 map[string]int                  `json:"secondary_abstentions,omitempty"`
}
type runOutput struct {
	ContractVersion, ManifestSHA256, DatasetReadinessSHA256 string
	Runner, ExecutionAuthority                              string
	CreatesFill                                             bool
	Provider, Feed, Timeframe                               string
	Universe                                                []string
	DataStart, DataEnd                                      string
	HoldoutAccessed                                         bool
	PerformanceRunCount                                     int
	Partitions                                              map[string]partitionResult
	Falsification                                           map[string]any
	AbstentionTotals                                        map[string]int
	DataQuality                                             map[string]any
	TerminalClassification                                  string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "VAL03=FAILED: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if recoveryContractAuditOnly() {
		return runRecoveryContractAudit()
	}
	if err := validateDateRange(dataStart, dataEnd); err != nil {
		return err
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	manifestHash := sha256Hex(manifestBytes)
	readyBytes, err := os.ReadFile(readyPath)
	if err != nil {
		return err
	}
	readyHash := sha256Hex(readyBytes)
	if manifestHash != "96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a" {
		return errors.New("manifest hash mismatch")
	}
	cfg, err := loadFrozenExperimentConfig(manifestBytes, manifestHash)
	if err != nil {
		return err
	}
	guard, err := contaminatedRunGuard(integrityPath)
	if err != nil {
		return fmt.Errorf("validate contaminated-run guard: %w", err)
	}
	if preflightOnly() {
		if !guard {
			return errors.New("contaminated-run guard is not active")
		}
		fmt.Printf("VAL03_PREFLIGHT=PASS candidate=%s threshold=%.2f holdout=%s no_rerun=true outcomes_calculated=false\n", cfg.CandidateID, cfg.ActionableConfidenceThreshold, cfg.HoldoutStart)
		return nil
	}
	if contractAuditOnly() {
		if !guard {
			return errors.New("contaminated-run guard is not active")
		}
		if err := auditManifestContract(manifestBytes, cfg); err != nil {
			return err
		}
		fmt.Printf("VAL03_CONTRACT_AUDIT=PASS candidate=%s falsifications=%d no_outcomes_loaded=true no_2025=true no_2026=true\n", cfg.CandidateID, len(requiredFalsificationNames()))
		return nil
	}
	if guard {
		return errors.New("contaminated VAL-03 execution identity is immutable; new preregistration required")
	}
	if readyHash != "b7081deb3bf55be5a23c1f1fcafbe44ff90c4fcee924de47f738cdcad86861ce" {
		return errors.New("readiness hash mismatch")
	}
	var ready readinessInput
	if err := json.Unmarshal(readyBytes, &ready); err != nil {
		return err
	}
	if len(ready.Families) == 0 {
		familyBytes, err := os.ReadFile(barFamilyPath)
		if err != nil {
			return fmt.Errorf("read bar-family evidence: %w", err)
		}
		familyHash := sha256Hex(familyBytes)
		if familyHash != "78c1fbeec2122eebba1a8dbe21c2edfc3c569377770561afe9e128cd633037e5" {
			return errors.New("bar-family evidence hash mismatch")
		}
		if err := json.Unmarshal(familyBytes, &ready); err != nil {
			return err
		}
	}
	data, err := loadData(ready)
	if err != nil {
		return err
	}
	if err := validateData(data); err != nil {
		return err
	}
	partitions := map[string][2]string{"development": {cfg.DevelopmentStart, cfg.DevelopmentEnd}, "validation": {cfg.ValidationStart, cfg.ValidationEnd}, "formal_oos": {cfg.OOSStart, cfg.OOSEnd}}
	results := map[string]partitionResult{}
	for name, dates := range partitions {
		r, err := evaluatePartition(data, name, dates[0], dates[1], cfg)
		if err != nil {
			return err
		}
		results[name] = r
	}
	for _, name := range []string{"development", "validation", "formal_oos"} {
		r := results[name]
		if err := attachPlacebos(data, &r, cfg, manifestHash); err != nil {
			return err
		}
		results[name] = r
	}
	for _, name := range []string{"development", "validation", "formal_oos"} {
		r := results[name]
		applyBootstrap(&r, manifestHash, cfg)
		results[name] = r
	}
	oos := results["formal_oos"]
	falsification := buildFalsification(data, oos, manifestHash, cfg, oos.SecondaryDiagnostics)
	classification := classifyPromotion(oos, falsificationDispositions(falsification), cfg)
	out := runOutput{ContractVersion: "jax.val-03.ma-crossover-result/v1", ManifestSHA256: manifestHash, DatasetReadinessSHA256: readyHash, Runner: "cmd/val03-ma-crossover", ExecutionAuthority: "NONE", CreatesFill: false, Provider: "Alpaca", Feed: "SIP", Timeframe: "1Day", Universe: append([]string(nil), symbols...), DataStart: dataStart, DataEnd: dataEnd, HoldoutAccessed: false, PerformanceRunCount: 1, Partitions: results, Falsification: falsification, AbstentionTotals: totalAbstentions(results), DataQuality: qualitySummary(data, true), TerminalClassification: classification}
	if err := writeJSON("Docs/validation/results/VAL-03-HYP-MA-001-RUN-MANIFEST.json", out); err != nil {
		return err
	}
	if err := writeJSON("Docs/validation/results/VAL-03-HYP-MA-001-DATASET.json", qualitySummary(data, false)); err != nil {
		return err
	}
	for name, result := range results {
		if err := writeJSON("Docs/validation/results/VAL-03-HYP-MA-001-"+partitionFile(name)+".json", result); err != nil {
			return err
		}
	}
	if err := writeJSON("Docs/validation/results/VAL-03-HYP-MA-001-FALSIFICATION.json", falsification); err != nil {
		return err
	}
	fmt.Printf("VAL03 classification=%s oos_episodes=%d oos_blocks=%d matched_pairs=%d\n", classification, len(oos.Episodes), oos.Blocks, oos.MatchedPairs)
	return nil
}

func validateDateRange(start, end string) error {
	if start > end {
		return errors.New("invalid descending date range")
	}
	if end >= holdoutStart {
		return fmt.Errorf("VAL-03 range reaches sealed holdout: %s", end)
	}
	return nil
}

func validateProviderDate(date string) error {
	if len(date) < 10 {
		return errors.New("provider timestamp too short")
	}
	if date[:10] >= holdoutStart {
		return fmt.Errorf("holdout row %s", date[:10])
	}
	return nil
}

func loadData(ready readinessInput) (map[string]*instrumentData, error) {
	data := map[string]*instrumentData{}
	for _, s := range symbols {
		data[s] = &instrumentData{Raw: map[string]bar{}, Split: map[string]bar{}, Detector: map[string]bar{}, Factor: map[string]float64{}, Boundaries: map[string]bool{}}
	}
	for _, family := range ready.Families {
		if family.Adjustment != "raw" && family.Adjustment != "split" && family.Adjustment != "split_spin_off" {
			continue
		}
		if data[family.Instrument] == nil {
			return nil, fmt.Errorf("unexpected instrument %s", family.Instrument)
		}
		for _, hash := range family.RawPayloadSHA256 {
			payload, err := os.ReadFile(filepath.Join(rawRoot, hash+".json"))
			if err != nil {
				return nil, fmt.Errorf("payload %s: %w", hash, err)
			}
			var p providerPayload
			if err := json.Unmarshal(payload, &p); err != nil {
				return nil, err
			}
			for _, item := range p.Bars[family.Instrument] {
				if err := validateProviderDate(item.Timestamp); err != nil {
					return nil, err
				}
				date := item.Timestamp[:10]
				b := bar{Date: date, Open: item.Open, High: item.High, Low: item.Low, Close: item.Close, Volume: item.Volume}
				switch family.Adjustment {
				case "raw":
					data[family.Instrument].Raw[date] = b
				case "split":
					data[family.Instrument].Split[date] = b
				case "split_spin_off":
					data[family.Instrument].Detector[date] = b
				}
			}
		}
	}
	for _, symbol := range symbols {
		d := data[symbol]
		if len(d.Raw) == 0 || len(d.Split) == 0 || len(d.Detector) == 0 {
			return nil, fmt.Errorf("missing bar family for %s", symbol)
		}
		for date := range d.Raw {
			if _, ok := d.Split[date]; !ok {
				return nil, fmt.Errorf("RAW/SPLIT unsynchronized %s %s", symbol, date)
			}
			if _, ok := d.Detector[date]; !ok {
				return nil, fmt.Errorf("SPLIT/SPIN_OFF unsynchronized %s %s", symbol, date)
			}
			d.Dates = append(d.Dates, date)
		}
		sort.Strings(d.Dates)
		if len(d.Dates) != 2264 || d.Dates[0] != "2016-01-04" || d.Dates[len(d.Dates)-1] != dataEnd {
			return nil, fmt.Errorf("unexpected range for %s", symbol)
		}
		previous := 0.0
		for _, date := range d.Dates {
			raw, split, detector := d.Raw[date], d.Split[date], d.Detector[date]
			if raw.Close <= 0 || split.Close <= 0 {
				return nil, fmt.Errorf("non-positive close %s %s", symbol, date)
			}
			factor := split.Close / raw.Close
			d.Factor[date] = factor
			if previous != 0 && math.Abs(factor-previous) > marketdata.VAL03BFactorTolerance*math.Max(1, math.Max(factor, previous)) {
				d.Boundaries[date] = true
			}
			if !sameBarScale(split, detector) {
				return nil, fmt.Errorf("SPLIT/SPIN_OFF structural difference %s %s", symbol, date)
			}
			previous = factor
		}
	}
	return data, nil
}
func sameBarScale(a, b bar) bool {
	return closeEnough(a.Open, b.Open) && closeEnough(a.High, b.High) && closeEnough(a.Low, b.Low) && closeEnough(a.Close, b.Close) && closeEnough(a.Volume, b.Volume)
}
func closeEnough(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}
func validateData(data map[string]*instrumentData) error {
	for _, s := range symbols {
		if len(data[s].Boundaries) > 0 {
			return fmt.Errorf("unresolved structural boundary for %s", s)
		}
	}
	return nil
}

func evaluatePartition(data map[string]*instrumentData, name, start, end string, cfg FrozenExperimentConfig) (partitionResult, error) {
	fast, medium, slow := cfg.SMAFast, cfg.SMAMedium, cfg.SMASlow
	r := partitionResult{Partition: name, Start: start, End: end, Abstentions: map[string]int{}, SecondaryDiagnostics: secondaryDirectionalDiagnostics{Policy: secondaryDiagnosticPolicyVer, Abstentions: map[string]int{}}, SecondaryAbstentions: map[string]int{}}
	for _, symbol := range symbols {
		d := data[symbol]
		activeUntil := -1
		secondaryActiveUntil := -1
		for i, date := range d.Dates {
			if date < start || date > end || i < slow-1 {
				continue
			}
			r.SignalObservations++
			sig, kind := buildSignal(d, i, fast, medium, slow, cfg)
			if kind == "HOLD" {
				r.Abstentions["HOLD"]++
				continue
			}
			if !actionableByConfig(sig.Confidence, cfg) {
				r.Abstentions["below-confidence"]++
				if kind == "SELL" {
					r.BearishDiagnostics++
				}
				continue
			}
			if i > secondaryActiveUntil {
				if diagnostic, exitIndex, ok := simulateDirectional(d, symbol, sig, i+1, kind, cfg); ok {
					r.SecondaryDiagnostics.Observations = append(r.SecondaryDiagnostics.Observations, diagnostic)
					secondaryActiveUntil = exitIndex
				} else {
					r.SecondaryDiagnostics.Abstentions["invalid_or_incomplete_diagnostic"]++
					r.SecondaryAbstentions["invalid_or_incomplete_diagnostic"]++
				}
			} else {
				r.SecondaryDiagnostics.Abstentions["active-position suppression"]++
				r.SecondaryAbstentions["active-position suppression"]++
			}
			if kind == "SELL" {
				r.BearishDiagnostics++
				continue
			}
			if i <= activeUntil {
				r.Abstentions["active-position suppression"]++
				continue
			}
			if entryIndexForSignal(i) >= len(d.Dates) {
				r.Abstentions["incomplete data"]++
				continue
			}
			next := d.Split[d.Dates[entryIndexForSignal(i)]]
			if !geometryValid(sig.Stop, next.Open, sig.Target) {
				r.Abstentions["PRE_ENTRY_INVALIDATED"]++
				continue
			}
			ep, exitIndex, ok := simulateLong(d, symbol, sig, entryIndexForSignal(i), 1, cfg)
			if !ok {
				r.Abstentions["structural_action_invalidated"]++
				continue
			}
			activeUntil = exitIndex
			ep.Regime = regimeAtDate(data, sig.Date, cfg)
			ep.Theme = themeForInstrument(symbol)
			r.ActionableLong++
			r.Episodes = append(r.Episodes, ep)
		}
	}
	finishPartition(&r)
	return r, nil
}

func geometryValid(stop, open, target float64) bool { return stop < open && open < target }

func entryIndexForSignal(signalIndex int) int { return signalIndex + 1 }

func openingCrossExit(open, stop, target float64) (float64, bool) {
	if open <= stop {
		return open, true
	}
	if open >= target {
		return open, true
	}
	return 0, false
}

func intrabarExit(low, high, stop, target float64) (float64, bool) {
	if low <= stop {
		return stop, true
	}
	if high >= target {
		return target, true
	}
	return 0, false
}

func matchedPlaceboScore(manifestHash, instrument, signalDate, candidateDate string) string {
	return sha256Hex([]byte(manifestHash + "|matched-placebo-v1|" + instrument + "|" + signalDate + "|" + candidateDate))
}

func quantityForCapital(capital, open float64) int {
	if open <= 0 {
		return 0
	}
	return int(math.Floor(capital / open))
}
func buildSignal(d *instrumentData, i, fast, medium, slow int, cfg FrozenExperimentConfig) (signal, string) {
	if i < slow-1 {
		return signal{}, "HOLD"
	}
	sf, sm, ss := avgWindow(d, i, fast), avgWindow(d, i, medium), avgWindow(d, i, slow)
	vol, atr := avgVolume(d, i, cfg.AvgVolumePeriod), atr14(d, i, cfg.ATRPeriod)
	price := d.Split[d.Dates[i]].Close
	s := signal{Date: d.Dates[i], Confidence: 0.65, SMA20: sf, SMA50: sm, SMA200: ss, ATR14: atr}
	if sf > sm && sm > ss && price > sf {
		s.Type = "BUY"
		s.Confidence += 0.12
		if d.Split[d.Dates[i]].Volume > vol {
			s.Confidence += 0.08
		}
		if (sf-ss)/ss > 0.05 {
			s.Confidence += 0.10
		}
		if s.Confidence > 1 {
			s.Confidence = 1
		}
		s.Stop = sm - atr
		s.Target = price + 3*atr
		return s, "BUY"
	}
	if sf < sm && sm < ss && price < sf {
		s.Type = "SELL"
		s.Confidence += 0.12
		if d.Split[d.Dates[i]].Volume > vol {
			s.Confidence += 0.08
		}
		if (ss-sf)/ss > 0.05 {
			s.Confidence += 0.10
		}
		if s.Confidence > 1 {
			s.Confidence = 1
		}
		s.Stop = sm + atr
		s.Target = price - 3*atr
		return s, "SELL"
	}
	return s, "HOLD"
}
func avgWindow(d *instrumentData, i, n int) float64 {
	v := make([]float64, n)
	for j := range v {
		v[j] = d.Split[d.Dates[i-n+1+j]].Close
	}
	return avg(v)
}
func avgVolume(d *instrumentData, i, n int) float64 {
	v := make([]float64, n)
	for j := range v {
		v[j] = d.Split[d.Dates[i-n+1+j]].Volume
	}
	return avg(v)
}

func atr14(d *instrumentData, i, period int) float64 {
	total := 0.0
	for j := i - period + 1; j <= i; j++ {
		b, p := d.Split[d.Dates[j]], d.Split[d.Dates[j-1]].Close
		total += math.Max(b.High-b.Low, math.Max(math.Abs(b.High-p), math.Abs(b.Low-p)))
	}
	return total / float64(period)
}
func simulateLong(d *instrumentData, symbol string, sig signal, entryIndex int, atrScale float64, cfg FrozenExperimentConfig) (episode, int, bool) {
	if entryIndex >= len(d.Dates) {
		return episode{}, entryIndex, false
	}
	entryDate := d.Dates[entryIndex]
	rawEntry := d.Raw[entryDate].Open
	qty := quantityForCapital(10000, rawEntry)
	if qty < 1 {
		return episode{}, entryIndex, false
	}
	stop := sig.SMA50 - sig.ATR14*cfg.StopATR*atrScale
	target := d.Split[sig.Date].Close + cfg.TargetATR*sig.ATR14*atrScale
	exitIndex, exitSplit := -1, 0.0
	for offset := 0; offset < cfg.HoldingPeriod; offset++ {
		idx := entryIndex + offset
		if idx >= len(d.Dates) {
			return episode{}, entryIndex, false
		}
		date := d.Dates[idx]
		if d.Boundaries[date] {
			return episode{}, idx, false
		}
		b := d.Split[date]
		if offset > 0 {
			if opening, crossed := openingCrossExit(b.Open, stop, target); crossed {
				exitIndex, exitSplit = idx, opening
				break
			}
		}
		if touched, crossed := intrabarExit(b.Low, b.High, stop, target); crossed {
			exitIndex, exitSplit = idx, touched
			break
		}
		if offset == cfg.HoldingPeriod-1 {
			exitIndex, exitSplit = idx, b.Close
			break
		}
	}
	if exitIndex < 0 {
		return episode{}, entryIndex, false
	}
	rawExit := exitSplit / d.Factor[d.Dates[exitIndex]]
	base := netReturn(rawEntry, rawExit, qty, cfg.BaseSpreadBPS, cfg.BaseSlippageBPS, cfg.BaseImpactBPS)
	stress := netReturnFixed(rawEntry, rawExit, qty, cfg.StressSpreadBPS, cfg.StressSlippageBPS, cfg.StressImpactBPS, 0.5)
	ep := episode{ID: fmt.Sprintf("%s-%s", symbol, sig.Date), Instrument: symbol, SignalDate: sig.Date, EntryDate: entryDate, ExitDate: d.Dates[exitIndex], Duration: exitIndex - entryIndex + 1, GrossReturn: rawExit/rawEntry - 1, NetReturn: base, StressNetReturn: stress, Year: yearOf(sig.Date)}
	return ep, exitIndex, true
}

func simulateDirectional(d *instrumentData, symbol string, sig signal, entryIndex int, direction string, cfg FrozenExperimentConfig) (directionalObservation, int, bool) {
	if entryIndex >= len(d.Dates) {
		return directionalObservation{}, entryIndex, false
	}
	entryDate := d.Dates[entryIndex]
	if d.Boundaries[sig.Date] {
		return directionalObservation{}, entryIndex, false
	}
	next := d.Split[entryDate]
	switch direction {
	case "BUY":
		if !geometryValid(sig.Stop, next.Open, sig.Target) {
			return directionalObservation{}, entryIndex, false
		}
	case "SELL":
		if !(sig.Target < next.Open && next.Open < sig.Stop) {
			return directionalObservation{}, entryIndex, false
		}
	default:
		return directionalObservation{}, entryIndex, false
	}
	exitIndex, exitSplit := -1, 0.0
	for offset := 0; offset < cfg.HoldingPeriod; offset++ {
		idx := entryIndex + offset
		if idx >= len(d.Dates) {
			return directionalObservation{}, entryIndex, false
		}
		date := d.Dates[idx]
		if d.Boundaries[date] {
			return directionalObservation{}, idx, false
		}
		b := d.Split[date]
		if offset > 0 {
			if direction == "BUY" {
				if opening, crossed := openingCrossExit(b.Open, sig.Stop, sig.Target); crossed {
					exitIndex, exitSplit = idx, opening
					break
				}
			} else {
				if b.Open >= sig.Stop || b.Open <= sig.Target {
					exitIndex, exitSplit = idx, b.Open
					break
				}
			}
		}
		if direction == "BUY" {
			if touched, crossed := intrabarExit(b.Low, b.High, sig.Stop, sig.Target); crossed {
				exitIndex, exitSplit = idx, touched
				break
			}
		} else {
			if b.High >= sig.Stop {
				exitIndex, exitSplit = idx, sig.Stop
				break
			}
			if b.Low <= sig.Target {
				exitIndex, exitSplit = idx, sig.Target
				break
			}
		}
		if offset == cfg.HoldingPeriod-1 {
			exitIndex, exitSplit = idx, b.Close
			break
		}
	}
	if exitIndex < 0 {
		return directionalObservation{}, entryIndex, false
	}
	rawEntry := d.Raw[entryDate].Open
	rawExit := exitSplit / d.Factor[d.Dates[exitIndex]]
	underlying := rawExit/rawEntry - 1
	effect := underlying
	if direction == "SELL" {
		effect = -underlying
	}
	return directionalObservation{Instrument: symbol, Year: yearOf(sig.Date), Direction: direction, Magnitude: absFloat(effect), SignalDate: sig.Date, EntryDate: entryDate, ExitDate: d.Dates[exitIndex], DirectionalEffect: effect, AbsoluteMagnitude: absFloat(effect), Stratum: canonicalDirectionalStratum(symbol, yearOf(sig.Date))}, exitIndex, true
}
func finishPartition(r *partitionResult) {
	r.Blocks, r.Instruments, r.MaxInstrumentShare, r.RegimeSlices = episodeBreadth(r.Episodes)
	r.CalendarRegimeCells = calendarRegimeCells(r.Episodes)
	r.MeanGross = mean(r.Episodes, func(e episode) float64 { return e.GrossReturn })
	r.MeanNet = mean(r.Episodes, func(e episode) float64 { return e.NetReturn })
	r.MeanStress = mean(r.Episodes, func(e episode) float64 { return e.StressNetReturn })
}
func episodeBreadth(eps []episode) (int, int, float64, int) {
	blocks, inst := map[string]bool{}, map[string]int{}
	for _, e := range eps {
		blocks[fmt.Sprintf("%s-%d", e.Instrument, e.Year)] = true
		inst[e.Instrument]++
	}
	maxShare := 0.0
	for _, n := range inst {
		maxShare = math.Max(maxShare, float64(n)/math.Max(1, float64(len(eps))))
	}
	return len(blocks), len(inst), maxShare, regimeBreadth(eps)
}
func regimeBreadth(eps []episode) int {
	return len(calendarRegimeCells(eps))
}
func attachPlacebos(data map[string]*instrumentData, r *partitionResult, cfg FrozenExperimentConfig, manifestHash string) error {
	if len(r.Episodes) == 0 {
		return nil
	}
	used := map[string]bool{}
	intervals := map[string][]activeInterval{}
	for _, e := range r.Episodes {
		d := data[e.Instrument]
		intervals[e.Instrument] = append(intervals[e.Instrument], newActiveInterval(e.Instrument, indexOf(d.Dates, e.SignalDate), indexOf(d.Dates, e.ExitDate)))
	}
	pairs, total := 0, 0.0
	for i := range r.Episodes {
		ep := &r.Episodes[i]
		d := data[ep.Instrument]
		best := matchedPlaceboDate(data, *ep, r.Start, r.End, used, intervals[ep.Instrument], cfg, manifestHash)
		if best == "" {
			r.Abstentions["no matched placebo"]++
			continue
		}
		p, ok := fixedHorizonReturn(d, best, cfg)
		if !ok {
			r.Abstentions["no matched placebo"]++
			continue
		}
		ep.PlaceboNetReturn = &p
		total += ep.NetReturn - p
		pairs++
	}
	r.MatchedPairs = pairs
	if pairs > 0 {
		r.MeanPairedDifference = total / float64(pairs)
	}
	return nil
}
func fixedHorizonReturn(d *instrumentData, date string, cfg FrozenExperimentConfig) (float64, bool) {
	i := indexOf(d.Dates, date)
	if i < 0 || i+cfg.HoldingPeriod >= len(d.Dates) {
		return 0, false
	}
	entry, exit := d.Dates[i+1], d.Dates[i+cfg.HoldingPeriod]
	rawEntry, rawExit := d.Raw[entry].Open, d.Raw[exit].Close
	qty := quantityForCapital(10000, rawEntry)
	if qty < 1 {
		return 0, false
	}
	return netReturn(rawEntry, rawExit, qty, cfg.BaseSpreadBPS, cfg.BaseSlippageBPS, cfg.BaseImpactBPS), true
}
func applyBootstrap(r *partitionResult, manifestHash string, cfg FrozenExperimentConfig) {
	blocks := map[string][]episode{}
	for _, e := range r.Episodes {
		id := fmt.Sprintf("%s-%d", e.Instrument, e.Year)
		blocks[id] = append(blocks[id], e)
	}
	ids := make([]string, 0, len(blocks))
	for id := range blocks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	r.Blocks = len(ids)
	if len(ids) == 0 {
		return
	}
	rng := seedFor(manifestHash, "|instrument-year-bootstrap-v2|")
	values, paired := make([]float64, cfg.BootstrapN), make([]float64, cfg.BootstrapN)
	for k := 0; k < cfg.BootstrapN; k++ {
		total, diff := 0.0, 0.0
		count, pairCount := 0, 0
		for j := 0; j < len(ids); j++ {
			rng = nextRand(rng)
			for _, e := range blocks[ids[int(rng%uint64(len(ids)))]] {
				total += e.NetReturn
				count++
				if e.PlaceboNetReturn != nil {
					diff += e.NetReturn - *e.PlaceboNetReturn
					pairCount++
				}
			}
		}
		if count > 0 {
			values[k] = total / float64(count)
		}
		if pairCount > 0 {
			paired[k] = diff / float64(pairCount)
		}
	}
	sort.Float64s(values)
	sort.Float64s(paired)
	r.BootstrapLow, r.BootstrapHigh = values[250], values[9749]
	if r.MatchedPairs > 0 {
		r.PairedBootstrapLow, r.PairedBootstrapHigh = paired[250], paired[9749]
	}
}
func buildFalsification(data map[string]*instrumentData, oos partitionResult, manifestHash string, cfg FrozenExperimentConfig, secondary secondaryDirectionalDiagnostics) map[string]any {
	intervals := episodeIntervals(data, oos.Episodes)
	permutation := secondarySignPermutation(secondary, manifestHash, cfg)
	topFive, topFiveDisposition := topFiveFalsification(oos.Episodes, cfg)
	dispositions := []FalsificationDisposition{
		{Name: "matched_non_signal_placebo", Status: dispositionForPairs(oos.MatchedPairs, cfg.PairedFloor), Blocking: true},
		{Name: "timestamp_placebo", Status: FalsificationInformational, Blocking: false},
		{Name: "secondary_sign_permutation", Status: permutation.Status, Blocking: false},
		topFiveDisposition,
		{Name: "leave_one_instrument_out", Status: FalsificationInformational, Blocking: false},
		{Name: "regime_slices", Status: dispositionForSlices(oos.RegimeSlices, cfg.MinimumSliceFloor), Blocking: true},
		{Name: "theme_slices", Status: FalsificationInformational, Blocking: false},
		{Name: "sma_19_49_199", Status: FalsificationInformational, Blocking: false},
		{Name: "sma_21_51_201", Status: FalsificationInformational, Blocking: false},
		{Name: "atr_0_90", Status: FalsificationInformational, Blocking: false},
		{Name: "atr_1_10", Status: FalsificationInformational, Blocking: false},
		{Name: "stress_cost", Status: FalsificationInformational, Blocking: false},
		{Name: "overlap_sensitivity", Status: FalsificationInformational, Blocking: false},
		{Name: "zero_and_buy_hold_baselines", Status: FalsificationInformational, Blocking: false},
	}
	return map[string]any{"matched_non_signal_placebo": map[string]any{"pairs": oos.MatchedPairs, "mean_difference": oos.MeanPairedDifference, "bootstrap_low": oos.PairedBootstrapLow}, "timestamp_placebo": timestampPlacebo(data, oos, manifestHash, cfg, intervals), "secondary_sign_permutation": permutation, "top_5_percent_exclusion": topFive, "leave_one_instrument_out": leaveOneOut(oos), "regime_slices": descriptiveSlices(oos, "regime"), "theme_slices": descriptiveSlices(oos, "theme"), "sma_19_49_199": variantSummary(data, cfg, 19, 49, 199, 1), "sma_21_51_201": variantSummary(data, cfg, 21, 51, 201, 1), "atr_0_90": variantSummary(data, cfg, 20, 50, 200, 0.90), "atr_1_10": variantSummary(data, cfg, 20, 50, 200, 1.10), "stress_cost": map[string]any{"mean_net_return": oos.MeanStress, "model": cfg.StressCostID}, "overlap_sensitivity": overlapSummary(oos, data, cfg), "zero_and_buy_hold_baselines": benchmarks(data, oos, cfg), "dispositions": dispositions, "seed_domain": manifestHash + "|falsification-v1|"}
}
func timestampPlacebo(data map[string]*instrumentData, oos partitionResult, manifestHash string, cfg FrozenExperimentConfig, intervals map[string][]activeInterval) map[string]any {
	selections := timestampPlaceboSelections(data, oos.Start, oos.End, manifestHash, cfg, intervals)
	b, _ := json.Marshal(selections)
	return map[string]any{"replicates": cfg.BootstrapN, "selection_count": len(selections), "selection_digest": sha256Hex(b), "seed": seedFor(manifestHash, "|timestamp-placebo-v1|"), "status": "EXECUTED_DETERMINISTIC_CONSTRAINED_SELECTIONS", "future_matching": false}
}
func leaveOneOut(r partitionResult) []map[string]any {
	out := []map[string]any{}
	for _, s := range symbols {
		eps := []episode{}
		for _, e := range r.Episodes {
			if e.Instrument != s {
				eps = append(eps, e)
			}
		}
		out = append(out, map[string]any{"excluded_instrument": s, "episode_count": len(eps), "mean_net_return": mean(eps, func(e episode) float64 { return e.NetReturn })})
	}
	return out
}
func variantSummary(data map[string]*instrumentData, cfg FrozenExperimentConfig, fast, medium, slow int, atrScale float64) map[string]any {
	result := evaluateVariant(data, cfg, fast, medium, slow, atrScale)
	return map[string]any{"fast": fast, "medium": medium, "slow": slow, "atr_scale": atrScale, "episodes": len(result.Episodes), "mean_net_return": result.MeanNet, "replacement_allowed": false, "status": "EXECUTED_FROZEN_FALSIFICATION_VARIANT"}
}
func overlapSummary(r partitionResult, data map[string]*instrumentData, cfg FrozenExperimentConfig) map[string]any {
	kept := overlapFilter(r.Episodes, data, cfg.OverlapWindow)
	return map[string]any{"window_sessions": cfg.OverlapWindow, "episode_count": len(kept), "mean_net_return": mean(kept, func(e episode) float64 { return e.NetReturn }), "status": "EXECUTED_FROZEN_EARLIEST_SIGNAL"}
}
func qualitySummary(data map[string]*instrumentData, performanceOutputGenerated bool) map[string]any {
	per := map[string]any{}
	for _, s := range symbols {
		d := data[s]
		per[s] = map[string]any{"raw_sessions": len(d.Raw), "split_sessions": len(d.Split), "split_spin_off_sessions": len(d.Detector), "first_session": d.Dates[0], "last_session": d.Dates[len(d.Dates)-1], "structural_boundaries": []string{}, "no_2025_rows": true}
	}
	return map[string]any{"provider": "Alpaca", "feed": "SIP", "timeframe": "1Day", "date_range": []string{dataStart, dataEnd}, "instruments": per, "performance_output_generated": performanceOutputGenerated}
}
func totalAbstentions(results map[string]partitionResult) map[string]int {
	out := map[string]int{}
	for _, r := range results {
		for k, v := range r.Abstentions {
			out[k] += v
		}
	}
	return out
}
func netReturn(entry, exit float64, qty int, spread, slippage, impact float64) float64 {
	return netReturnFixed(entry, exit, qty, spread, slippage, impact, 0)
}
func netReturnFixed(entry, exit float64, qty int, spread, slippage, impact, fixedCommission float64) float64 {
	rate := (spread + slippage + impact) / 10000
	en, xn := entry*float64(qty), exit*float64(qty)
	ec, xc := fixedCommission, fixedCommission
	if fixedCommission == 0 {
		ec = math.Min(math.Max(0.005*float64(qty), 1), 0.01*en)
		xc = math.Min(math.Max(0.005*float64(qty), 1), 0.01*xn)
	}
	return (xn*(1-rate) - xc - (en*(1+rate) + ec)) / en
}
func avg(v []float64) float64 {
	total := 0.0
	for _, x := range v {
		total += x
	}
	return total / float64(len(v))
}
func mean(v []episode, f func(episode) float64) float64 {
	if len(v) == 0 {
		return 0
	}
	total := 0.0
	for _, e := range v {
		total += f(e)
	}
	return total / float64(len(v))
}
func indexOf(v []string, x string) int {
	for i, y := range v {
		if y == x {
			return i
		}
	}
	return -1
}
func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func yearOf(s string) int {
	n := 0
	for _, r := range s[:4] {
		n = n*10 + int(r-'0')
	}
	return n
}
func sha256Hex(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func seedFor(s, domain string) uint64 {
	h := sha256.Sum256([]byte(s + domain))
	return binary.BigEndian.Uint64(h[:8])
}
func nextRand(x uint64) uint64 { x ^= x << 13; x ^= x >> 7; x ^= x << 17; return x }
func partitionFile(s string) string {
	if s == "formal_oos" {
		return "OOS"
	}
	if s == "development" {
		return "DEVELOPMENT"
	}
	return "VALIDATION"
}
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o600)
}
