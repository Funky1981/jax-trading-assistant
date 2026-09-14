package main

import (
	"fmt"
	"sort"
)

const (
	topFiveRuleVersion          = "CEIL_5_PERCENT_V1"
	slicePolicyVersion          = "CALENDAR_YEAR_X_SPY_SMA200_REGIME_V1"
	signPermutationAlgorithmVer = "SHA256_INDEPENDENT_SIGN_ASSIGNMENT_V1"
)

// FalsificationStatus is deliberately closed: an unclassified registered
// diagnostic cannot be treated as a pass.
type FalsificationStatus string

const (
	FalsificationPass          FalsificationStatus = "PASS"
	FalsificationFail          FalsificationStatus = "FAIL"
	FalsificationInsufficient  FalsificationStatus = "INSUFFICIENT"
	FalsificationInformational FalsificationStatus = "INFORMATIONAL"
	FalsificationNotApplicable FalsificationStatus = "NOT_APPLICABLE"
)

type FalsificationDisposition struct {
	Name     string              `json:"name"`
	Status   FalsificationStatus `json:"status"`
	Blocking bool                `json:"blocking"`
	Reason   string              `json:"reason,omitempty"`
}

type activeInterval struct {
	Instrument string
	Start      int
	End        int
}

func newActiveInterval(instrument string, signalIndex, exitIndex int) activeInterval {
	return activeInterval{Instrument: instrument, Start: signalIndex, End: exitIndex}
}

func (a activeInterval) contains(index int) bool {
	return index >= a.Start && index <= a.End
}

func dateInActiveIntervals(index int, intervals []activeInterval) bool {
	for _, interval := range intervals {
		if interval.contains(index) {
			return true
		}
	}
	return false
}

func validStructuralWindow(d *instrumentData, start, length int) bool {
	if start < 0 || start+length >= len(d.Dates) {
		return false
	}
	for i := start; i <= start+length; i++ {
		if d.Boundaries[d.Dates[i]] {
			return false
		}
	}
	return true
}

func regimeAtDate(data map[string]*instrumentData, date string, cfg FrozenExperimentConfig) string {
	d := data["SPY"]
	if d == nil {
		return "UNKNOWN"
	}
	i := indexOf(d.Dates, date)
	if i < cfg.SMASlow-1 {
		return "UNKNOWN"
	}
	close := d.Split[date].Close
	slow := avgWindow(d, i, cfg.SMASlow)
	if close > slow {
		return "SPY_ABOVE_SMA200"
	}
	return "SPY_BELOW_SMA200"
}

func themeForInstrument(instrument string) string {
	switch instrument {
	case "SPY", "QQQ", "IWM", "DIA":
		return "broad_equity"
	case "XLK", "XLF", "XLE":
		return "sector_equity"
	case "TLT", "GLD":
		return "rates_precious_metal"
	default:
		return "UNKNOWN"
	}
}

func actualSignalDate(d *instrumentData, index int, cfg FrozenExperimentConfig) bool {
	if index < cfg.SMASlow-1 || index >= len(d.Dates) {
		return false
	}
	sig, kind := buildSignal(d, index, cfg.SMAFast, cfg.SMAMedium, cfg.SMASlow, cfg)
	return kind != "HOLD" && actionableByConfig(sig.Confidence, cfg)
}

func placeboKey(instrument, date string) string { return instrument + "|" + date }

func matchedPlaceboDate(data map[string]*instrumentData, ep episode, partitionStart, partitionEnd string, used map[string]bool, intervals []activeInterval, cfg FrozenExperimentConfig, manifestHash string) string {
	d := data[ep.Instrument]
	if d == nil {
		return ""
	}
	signalIndex := indexOf(d.Dates, ep.SignalDate)
	if signalIndex < 0 {
		return ""
	}
	wantRegime := regimeAtDate(data, ep.SignalDate, cfg)
	type candidate struct {
		date, score string
		distance    int
	}
	choices := []candidate{}
	for idx, date := range d.Dates {
		if date < partitionStart || date > partitionEnd || date < ep.SignalDate[:4]+"-01-01" || date > ep.SignalDate[:4]+"-12-31" || idx < cfg.SMASlow-1 || idx+cfg.HoldingPeriod >= len(d.Dates) {
			continue
		}
		if idx == signalIndex || used[placeboKey(ep.Instrument, date)] || actualSignalDate(d, idx, cfg) || dateInActiveIntervals(idx, intervals) {
			continue
		}
		if regimeAtDate(data, date, cfg) != wantRegime || !validStructuralWindow(d, idx, cfg.HoldingPeriod) {
			continue
		}
		distance := absInt(idx - signalIndex)
		if distance > 60 {
			continue
		}
		choices = append(choices, candidate{date: date, distance: distance, score: matchedPlaceboScore(manifestHash, ep.Instrument, ep.SignalDate, date)})
	}
	sort.Slice(choices, func(i, j int) bool {
		if choices[i].distance != choices[j].distance {
			return choices[i].distance < choices[j].distance
		}
		return choices[i].score < choices[j].score
	})
	if len(choices) == 0 {
		return ""
	}
	used[placeboKey(ep.Instrument, choices[0].date)] = true
	return choices[0].date
}

type timestampPlaceboSelection struct {
	Replicate  int    `json:"replicate"`
	Instrument string `json:"instrument"`
	Year       int    `json:"year"`
	Date       string `json:"date"`
}

func timestampPlaceboSelections(data map[string]*instrumentData, start, end string, manifestHash string, cfg FrozenExperimentConfig, intervals map[string][]activeInterval) []timestampPlaceboSelection {
	result := make([]timestampPlaceboSelection, 0, cfg.BootstrapN*len(symbols))
	ordered := append([]string(nil), symbols...)
	sort.Strings(ordered)
	years := map[int]bool{}
	for _, instrument := range ordered {
		if d := data[instrument]; d != nil {
			for _, date := range d.Dates {
				if date >= start && date <= end {
					years[yearOf(date)] = true
				}
			}
		}
	}
	orderedYears := make([]int, 0, len(years))
	for year := range years {
		orderedYears = append(orderedYears, year)
	}
	sort.Ints(orderedYears)
	for replicate := 0; replicate < cfg.BootstrapN; replicate++ {
		for _, instrument := range ordered {
			d := data[instrument]
			if d == nil {
				continue
			}
			for _, year := range orderedYears {
				candidates := []string{}
				for idx, date := range d.Dates {
					if date < start || date > end || yearOf(date) != year || idx < cfg.SMASlow-1 || idx+cfg.HoldingPeriod >= len(d.Dates) || actualSignalDate(d, idx, cfg) || dateInActiveIntervals(idx, intervals[instrument]) || !validStructuralWindow(d, idx, cfg.HoldingPeriod) {
						continue
					}
					candidates = append(candidates, date)
				}
				sort.Slice(candidates, func(i, j int) bool {
					a := sha256Hex([]byte(fmt.Sprintf("%d|%d|%s|%s", seedFor(manifestHash, "|timestamp-placebo-v1|"), replicate, instrument, candidates[i])))
					b := sha256Hex([]byte(fmt.Sprintf("%d|%d|%s|%s", seedFor(manifestHash, "|timestamp-placebo-v1|"), replicate, instrument, candidates[j])))
					return a < b
				})
				if len(candidates) > 0 {
					result = append(result, timestampPlaceboSelection{Replicate: replicate, Instrument: instrument, Year: year, Date: candidates[0]})
				}
			}
		}
	}
	return result
}

type directionalObservation struct {
	Instrument string
	Year       int
	Direction  string
	Magnitude  float64
}

type secondaryDirectionalDiagnostics struct {
	Observations []directionalObservation
}

type signPermutationSummary struct {
	Status      FalsificationStatus `json:"status"`
	Replicates  int                 `json:"replicates"`
	Seed        uint64              `json:"seed"`
	PValue      float64             `json:"p_value,omitempty"`
	MixedStrata int                 `json:"mixed_strata"`
	Algorithm   string              `json:"algorithm"`
	Strata      []string            `json:"strata,omitempty"`
}

func secondarySignPermutation(diagnostics secondaryDirectionalDiagnostics, manifestHash string, cfg FrozenExperimentConfig) signPermutationSummary {
	return signPermutation(diagnostics.Observations, manifestHash, cfg)
}

func signPermutation(observations []directionalObservation, manifestHash string, cfg FrozenExperimentConfig) signPermutationSummary {
	strata := map[string][]directionalObservation{}
	for _, observation := range observations {
		strata[fmt.Sprintf("%s-%d", observation.Instrument, observation.Year)] = append(strata[fmt.Sprintf("%s-%d", observation.Instrument, observation.Year)], observation)
	}
	keys := make([]string, 0, len(strata))
	for key := range strata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := map[string][]directionalObservation{}
	mixed := 0
	for _, key := range keys {
		values := append([]directionalObservation(nil), strata[key]...)
		sort.Slice(values, func(i, j int) bool {
			if values[i].Direction != values[j].Direction {
				return values[i].Direction < values[j].Direction
			}
			return values[i].Magnitude < values[j].Magnitude
		})
		ordered[key] = values
		hasBuy, hasSell := false, false
		for _, value := range values {
			hasBuy = hasBuy || value.Direction == "BUY"
			hasSell = hasSell || value.Direction == "SELL"
		}
		if hasBuy && hasSell {
			mixed++
		}
	}
	seed := seedFor(manifestHash, "|secondary-sign-permutation-v1|")
	if mixed == 0 {
		return signPermutationSummary{Status: FalsificationNotApplicable, Replicates: cfg.BootstrapN, Seed: seed, MixedStrata: 0, Algorithm: signPermutationAlgorithmVer, Strata: keys}
	}
	canonical := []directionalObservation{}
	for _, key := range keys {
		canonical = append(canonical, ordered[key]...)
	}
	observed := signedMean(canonical)
	ge := 0
	for replicate := 0; replicate < cfg.BootstrapN; replicate++ {
		null := make([]directionalObservation, 0, len(canonical))
		for _, key := range keys {
			for index, value := range ordered[key] {
				assignment := sha256Hex([]byte(fmt.Sprintf("%d|%d|%s|%d", seed, replicate, key, index)))
				if assignment[0]%2 == 0 {
					value.Direction = "BUY"
				} else {
					value.Direction = "SELL"
				}
				null = append(null, value)
			}
		}
		if signedMean(null) >= observed {
			ge++
		}
	}
	return signPermutationSummary{Status: FalsificationInformational, Replicates: cfg.BootstrapN, Seed: seed, PValue: float64(1+ge) / float64(1+cfg.BootstrapN), MixedStrata: mixed, Algorithm: signPermutationAlgorithmVer, Strata: keys}
}

func signedMean(observations []directionalObservation) float64 {
	if len(observations) == 0 {
		return 0
	}
	total := 0.0
	for _, observation := range observations {
		if observation.Direction == "SELL" {
			total -= absFloat(observation.Magnitude)
		} else {
			total += absFloat(observation.Magnitude)
		}
	}
	return total / float64(len(observations))
}

func overlapFilter(episodes []episode, data map[string]*instrumentData, window int) []episode {
	byInstrument := map[string][]episode{}
	for _, ep := range episodes {
		byInstrument[ep.Instrument] = append(byInstrument[ep.Instrument], ep)
	}
	result := []episode{}
	for instrument, values := range byInstrument {
		dates := data[instrument].Dates
		sort.Slice(values, func(i, j int) bool { return indexOf(dates, values[i].EntryDate) < indexOf(dates, values[j].EntryDate) })
		last := -1
		for _, ep := range values {
			index := indexOf(dates, ep.EntryDate)
			if last < 0 || index-last >= window {
				result = append(result, ep)
				last = index
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func buyHoldReturn(entry, exit float64) float64 {
	if entry <= 0 {
		return 0
	}
	return exit/entry - 1
}

func topFivePercentExclusion(episodes []episode) (before, after float64, removed []string) {
	ordered := append([]episode(nil), episodes...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].NetReturn > ordered[j].NetReturn })
	remove := topFiveRemovalCount(len(ordered))
	before = mean(ordered, func(e episode) float64 { return e.NetReturn })
	for i := 0; i < remove && i < len(ordered); i++ {
		removed = append(removed, ordered[i].ID)
	}
	if remove < len(ordered) {
		after = mean(ordered[remove:], func(e episode) float64 { return e.NetReturn })
	}
	return before, after, removed
}

func topFiveRemovalCount(n int) int {
	if n <= 0 {
		return 0
	}
	remove := (n*5 + 99) / 100
	if remove < 1 {
		return 1
	}
	return remove
}

func topFiveFalsification(episodes []episode, cfg FrozenExperimentConfig) (map[string]any, FalsificationDisposition) {
	before, after, removed := topFivePercentExclusion(episodes)
	removedSet := map[string]bool{}
	for _, id := range removed {
		removedSet[id] = true
	}
	remaining := make([]episode, 0, len(episodes)-len(removed))
	pairs := 0
	difference := 0.0
	for _, ep := range episodes {
		if removedSet[ep.ID] {
			continue
		}
		remaining = append(remaining, ep)
		if ep.PlaceboNetReturn != nil {
			pairs++
			difference += ep.NetReturn - *ep.PlaceboNetReturn
		}
	}
	pairedMean := 0.0
	if pairs > 0 {
		pairedMean = difference / float64(pairs)
	}
	status := FalsificationPass
	reason := "remaining primary effect and paired effect remain positive"
	if pairs < cfg.PairedFloor {
		status = FalsificationInsufficient
		reason = "remaining matched evidence is below the registered pair floor"
	} else if mean(remaining, func(e episode) float64 { return e.NetReturn }) <= 0 || pairedMean <= 0 {
		status = FalsificationFail
		reason = "top-five removal eliminates the remaining primary or paired effect"
	}
	disposition := FalsificationDisposition{Name: "top_5_percent_exclusion", Status: status, Blocking: true, Reason: reason}
	return map[string]any{"rule_version": topFiveRuleVersion, "removed": removed, "removed_count": len(removed), "mean_before": before, "mean_after": after, "remaining_episode_count": len(remaining), "remaining_matched_pairs": pairs, "remaining_mean_paired_difference": pairedMean, "disposition": disposition}, disposition
}

func episodeIntervals(data map[string]*instrumentData, episodes []episode) map[string][]activeInterval {
	result := map[string][]activeInterval{}
	for _, ep := range episodes {
		d := data[ep.Instrument]
		result[ep.Instrument] = append(result[ep.Instrument], newActiveInterval(ep.Instrument, indexOf(d.Dates, ep.SignalDate), indexOf(d.Dates, ep.ExitDate)))
	}
	return result
}

func dispositionForPairs(pairs, floor int) FalsificationStatus {
	if pairs < floor {
		return FalsificationInsufficient
	}
	return FalsificationPass
}

func dispositionForSlices(slices, floor int) FalsificationStatus {
	if slices < floor {
		return FalsificationInsufficient
	}
	return FalsificationPass
}

func falsificationDispositions(result map[string]any) []FalsificationDisposition {
	raw, ok := result["dispositions"].([]FalsificationDisposition)
	if ok {
		return raw
	}
	return nil
}

func calendarRegimeCells(eps []episode) []string {
	cells := map[string]bool{}
	for _, e := range eps {
		if e.Regime == "" || e.Regime == "UNKNOWN" {
			continue
		}
		cells[fmt.Sprintf("%d|%s", e.Year, e.Regime)] = true
	}
	result := make([]string, 0, len(cells))
	for cell := range cells {
		result = append(result, cell)
	}
	sort.Strings(result)
	return result
}

func evaluateVariant(data map[string]*instrumentData, cfg FrozenExperimentConfig, fast, medium, slow int, atrScale float64) partitionResult {
	r := partitionResult{Partition: "falsification", Start: cfg.OOSStart, End: cfg.OOSEnd, Abstentions: map[string]int{}}
	for _, instrument := range cfg.Universe {
		d := data[instrument]
		activeUntil := -1
		for i, date := range d.Dates {
			if date < cfg.OOSStart || date > cfg.OOSEnd || i < slow-1 {
				continue
			}
			r.SignalObservations++
			sig, kind := buildSignal(d, i, fast, medium, slow, cfg)
			if kind != "BUY" {
				r.Abstentions[kind]++
				continue
			}
			if !actionableByConfig(sig.Confidence, cfg) {
				r.Abstentions["below-confidence"]++
				continue
			}
			if i <= activeUntil || i+1 >= len(d.Dates) {
				r.Abstentions["active-position suppression"]++
				continue
			}
			next := d.Split[d.Dates[i+1]]
			if !geometryValid(sig.Stop, next.Open, sig.Target) {
				r.Abstentions["PRE_ENTRY_INVALIDATED"]++
				continue
			}
			ep, exitIndex, ok := simulateLong(d, instrument, sig, i+1, atrScale, cfg)
			if !ok {
				r.Abstentions["structural_action_invalidated"]++
				continue
			}
			ep.Regime, ep.Theme = regimeAtDate(data, date, cfg), themeForInstrument(instrument)
			activeUntil = exitIndex
			r.Episodes = append(r.Episodes, ep)
		}
	}
	finishPartition(&r)
	return r
}

func descriptiveSlices(r partitionResult, key string) map[string]any {
	slices := map[string][]episode{}
	for _, ep := range r.Episodes {
		name := ep.Regime
		if key == "theme" {
			name = ep.Theme
		}
		if name == "" {
			name = "UNKNOWN"
		}
		slices[name] = append(slices[name], ep)
	}
	out := map[string]any{"status": "REPORTED_DESCRIPTIVELY", "slice_key": key, "slices": map[string]any{}}
	values := out["slices"].(map[string]any)
	for name, episodes := range slices {
		values[name] = map[string]any{"episode_count": len(episodes), "mean_net_return": mean(episodes, func(e episode) float64 { return e.NetReturn }), "contribution": mean(episodes, func(e episode) float64 { return e.NetReturn }) * float64(len(episodes))}
	}
	return out
}

func benchmarks(data map[string]*instrumentData, r partitionResult, cfg FrozenExperimentConfig) map[string]any {
	out := map[string]any{"zero_return": 0.0, "same_asset_buy_hold": []any{}, "spy_buy_hold": []any{}, "gross_vs_net": true}
	same := []any{}
	spy := []any{}
	for _, ep := range r.Episodes {
		d := data[ep.Instrument]
		entry := d.Raw[ep.EntryDate].Open
		exit := d.Raw[ep.ExitDate].Close
		same = append(same, map[string]any{"episode_id": ep.ID, "return": buyHoldReturn(entry, exit), "duration": ep.Duration})
		spyData := data["SPY"]
		si, ei := indexOf(spyData.Dates, ep.EntryDate), indexOf(spyData.Dates, ep.ExitDate)
		if si >= 0 && ei >= si {
			spy = append(spy, map[string]any{"episode_id": ep.ID, "return": buyHoldReturn(spyData.Raw[ep.EntryDate].Open, spyData.Raw[ep.ExitDate].Close), "duration": ei - si + 1})
		}
	}
	out["same_asset_buy_hold"] = same
	out["spy_buy_hold"] = spy
	_ = cfg
	return out
}

func classifyPromotion(oos partitionResult, dispositions []FalsificationDisposition, cfg FrozenExperimentConfig) string {
	if len(dispositions) == 0 {
		return "INSUFFICIENT_EVIDENCE"
	}
	for _, disposition := range dispositions {
		if disposition.Blocking && disposition.Status == FalsificationFail {
			return "FAILED_VALIDATION"
		}
		if disposition.Blocking && disposition.Status == FalsificationInsufficient {
			return "INSUFFICIENT_EVIDENCE"
		}
		if disposition.Blocking && disposition.Status != FalsificationPass {
			return "INSUFFICIENT_EVIDENCE"
		}
	}
	if oos.Episodes == nil || len(oos.Episodes) < cfg.OOSSampleFloor || oos.MatchedPairs < cfg.PairedFloor || oos.Blocks < cfg.EffectiveBlockFloor || oos.Instruments < cfg.InstrumentFloor || oos.MaxInstrumentShare > cfg.ConcentrationCeiling || oos.RegimeSlices < cfg.MinimumSliceFloor {
		return "INSUFFICIENT_EVIDENCE"
	}
	if oos.MeanNet > 0 && oos.BootstrapLow > 0 && oos.MeanPairedDifference > 0 && oos.PairedBootstrapLow > 0 {
		return "FORWARD_PAPER_ELIGIBLE"
	}
	return "FAILED_VALIDATION"
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func requiredFalsificationNames() []string {
	return []string{
		"matched_non_signal_placebo", "timestamp_placebo", "secondary_sign_permutation",
		"top_5_percent_exclusion", "leave_one_instrument_out", "regime_slices",
		"theme_slices", "sma_19_49_199", "sma_21_51_201", "atr_0_90", "atr_1_10",
		"stress_cost", "overlap_sensitivity", "zero_and_buy_hold_baselines",
	}
}
