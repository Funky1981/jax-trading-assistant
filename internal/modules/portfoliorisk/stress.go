package portfoliorisk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const StressAlgorithmV1 = "jax.portfolio.stress_scenario/v1"

type StressScenario struct {
	ScenarioID   string             `json:"scenario_id"`
	Version      string             `json:"version"`
	Name         string             `json:"name"`
	DefaultShock *float64           `json:"default_shock,omitempty"`
	PriceShocks  map[string]float64 `json:"price_shocks,omitempty"`
}

func BuildStressScenario(input StressScenario) (StressScenario, error) {
	claimed := input.ScenarioID
	input.ScenarioID = ""
	input.Version = trim(input.Version)
	input.Name = trim(input.Name)
	if input.Version == "" || input.Name == "" || input.DefaultShock == nil || !finite(*input.DefaultShock) || *input.DefaultShock <= -1 || *input.DefaultShock > 1 {
		return StressScenario{}, fmt.Errorf("invalid stress scenario: version, name and default shock in (-1,1] are required")
	}
	input.PriceShocks = cloneShockMap(input.PriceShocks)
	for instrument, shock := range input.PriceShocks {
		if trim(instrument) == "" || !finite(shock) || shock <= -1 || shock > 1 {
			return StressScenario{}, fmt.Errorf("invalid stress shock for %q", instrument)
		}
	}
	input.ScenarioID = stressScenarioIdentity(input)
	if claimed != "" && claimed != input.ScenarioID {
		return StressScenario{}, fmt.Errorf("%w: stress scenario identity mismatch", ErrSnapshotIdentity)
	}
	return input, nil
}

type StressLine struct {
	InstrumentID  string  `json:"instrument_id"`
	BaseValue     float64 `json:"base_value"`
	Shock         float64 `json:"shock"`
	StressedValue float64 `json:"stressed_value"`
	PnL           float64 `json:"pnl"`
}

type StressResult struct {
	ResultID       string       `json:"result_id"`
	Algorithm      string       `json:"algorithm"`
	ScenarioID     string       `json:"scenario_id"`
	SnapshotID     string       `json:"snapshot_id"`
	Equity         float64      `json:"equity"`
	StressedEquity float64      `json:"stressed_equity"`
	PnL            float64      `json:"pnl"`
	LossFraction   float64      `json:"loss_fraction"`
	Lines          []StressLine `json:"lines"`
}

func CalculateStress(snapshot PortfolioSnapshot, analytics ExposureAnalytics, scenario StressScenario) (StressResult, error) {
	canonical, err := BuildSnapshot(snapshot)
	if err != nil {
		return StressResult{}, err
	}
	canonicalScenario, err := BuildStressScenario(scenario)
	if err != nil {
		return StressResult{}, err
	}
	if analytics.SnapshotID != canonical.SnapshotID || analytics.Validate() != nil || analytics.Equity != canonical.Equity.Value || !canonical.Equity.Known || canonical.Equity.Value <= 0 {
		return StressResult{}, fmt.Errorf("stress requires matching known portfolio analytics")
	}
	byInstrument := make(map[string]float64, len(analytics.Lines))
	for _, line := range analytics.Lines {
		byInstrument[line.InstrumentID] = line.MarketValue
	}
	lines := make([]StressLine, 0, len(byInstrument))
	pnl := 0.0
	for instrument, base := range byInstrument {
		shock := *canonicalScenario.DefaultShock
		if specific, ok := canonicalScenario.PriceShocks[instrument]; ok {
			shock = specific
		}
		stressed := base * (1 + shock)
		line := StressLine{InstrumentID: instrument, BaseValue: base, Shock: shock, StressedValue: stressed, PnL: stressed - base}
		lines = append(lines, line)
		pnl += line.PnL
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].InstrumentID < lines[j].InstrumentID })
	result := StressResult{Algorithm: StressAlgorithmV1, ScenarioID: canonicalScenario.ScenarioID, SnapshotID: canonical.SnapshotID, Equity: canonical.Equity.Value, StressedEquity: canonical.Equity.Value + pnl, PnL: pnl, LossFraction: -pnl / canonical.Equity.Value, Lines: lines}
	if !finite(result.StressedEquity) || !finite(result.LossFraction) {
		return StressResult{}, fmt.Errorf("stress produced non-finite result")
	}
	result.ResultID = stressResultIdentity(result)
	return result, nil
}

func trim(value string) string { return strings.TrimSpace(value) }
func cloneShockMap(input map[string]float64) map[string]float64 {
	output := make(map[string]float64, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
func stressScenarioIdentity(scenario StressScenario) string {
	b, _ := json.Marshal(scenario)
	digest := sha256.Sum256(b)
	return "scen_" + hex.EncodeToString(digest[:])
}
func stressResultIdentity(result StressResult) string {
	copyResult := result
	copyResult.ResultID = ""
	b, _ := json.Marshal(copyResult)
	digest := sha256.Sum256(b)
	return "stress_" + hex.EncodeToString(digest[:])
}
