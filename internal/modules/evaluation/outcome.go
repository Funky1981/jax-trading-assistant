package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/researchrecommendation"
)

const OutcomeTrackingContractV1 = "jax.recommendation_outcome/v1"

type OutcomeHorizon string

const (
	OutcomeNextObservation    OutcomeHorizon = "NEXT_OBSERVATION"
	OutcomeOneDay             OutcomeHorizon = "ONE_DAY"
	OutcomeFiveObservations   OutcomeHorizon = "FIVE_OBSERVATIONS"
	OutcomeTwentyObservations OutcomeHorizon = "TWENTY_OBSERVATIONS"
	OutcomeInvalidation       OutcomeHorizon = "INVALIDATION_REACHED"
)

type OutcomeObservation struct {
	At             time.Time `json:"at"`
	Price          float64   `json:"price"`
	BenchmarkPrice *float64  `json:"benchmark_price,omitempty"`
}

func (observation OutcomeObservation) Validate() error {
	if observation.At.IsZero() || observation.At.Location() != time.UTC || !finite(observation.Price) || observation.Price <= 0 {
		return fmt.Errorf("outcome observation requires a positive price and UTC timestamp")
	}
	if observation.BenchmarkPrice != nil && (!finite(*observation.BenchmarkPrice) || *observation.BenchmarkPrice <= 0) {
		return fmt.Errorf("benchmark observation requires a positive finite price")
	}
	return nil
}

type OutcomeWindowResult struct {
	Horizon                    OutcomeHorizon `json:"horizon"`
	StartAt                    time.Time      `json:"start_at"`
	EndAt                      time.Time      `json:"end_at"`
	Complete                   bool           `json:"complete"`
	MissingReason              string         `json:"missing_reason,omitempty"`
	StartPrice                 float64        `json:"start_price"`
	EndPrice                   float64        `json:"end_price"`
	Return                     *float64       `json:"return,omitempty"`
	BenchmarkRelativeReturn    *float64       `json:"benchmark_relative_return,omitempty"`
	MaximumFavourableExcursion *float64       `json:"maximum_favourable_excursion,omitempty"`
	MaximumAdverseExcursion    *float64       `json:"maximum_adverse_excursion,omitempty"`
	InvalidationReached        bool           `json:"invalidation_reached"`
	DirectionalOutcome         string         `json:"directional_outcome"`
}

func (window OutcomeWindowResult) Validate(decisionAt time.Time) error {
	if window.Horizon != OutcomeNextObservation && window.Horizon != OutcomeOneDay && window.Horizon != OutcomeFiveObservations && window.Horizon != OutcomeTwentyObservations && window.Horizon != OutcomeInvalidation {
		return fmt.Errorf("unsupported outcome horizon %q", window.Horizon)
	}
	if window.DirectionalOutcome != "NOT_APPLICABLE" {
		return fmt.Errorf("outcome direction must remain NOT_APPLICABLE without an explicit strategy")
	}
	if !window.Complete {
		if strings.TrimSpace(window.MissingReason) == "" || window.Return != nil {
			return fmt.Errorf("incomplete outcome window requires a reason and no synthetic return")
		}
		return nil
	}
	if window.StartAt.IsZero() || window.EndAt.IsZero() || window.StartAt.Location() != time.UTC || window.EndAt.Location() != time.UTC || !window.EndAt.After(window.StartAt) || window.StartAt.Before(decisionAt) || !finite(window.StartPrice) || !finite(window.EndPrice) || window.StartPrice <= 0 || window.EndPrice <= 0 || window.Return == nil || !finite(*window.Return) {
		return fmt.Errorf("complete outcome window has invalid timestamps, prices or return")
	}
	return nil
}

type RecommendationOutcome struct {
	ContractVersion         string                `json:"contract_version"`
	ID                      string                `json:"id"`
	CaseID                  string                `json:"case_id"`
	RecommendationID        string                `json:"recommendation_id"`
	ResearchOutputID        string                `json:"research_output_id"`
	ContextPlanID           string                `json:"context_plan_id"`
	DecisionAt              time.Time             `json:"decision_at"`
	Disposition             string                `json:"disposition"`
	OriginalThesis          string                `json:"original_thesis"`
	OriginalInvalidationIDs []string              `json:"original_invalidation_evidence_ids"`
	Windows                 []OutcomeWindowResult `json:"windows"`
	RecordedAt              time.Time             `json:"recorded_at"`
	OutcomeSource           string                `json:"outcome_source"`
}

func TrackRecommendationOutcome(caseFile HistoricalCase, observations []OutcomeObservation, invalidationAt *time.Time, outcomeSource string, recordedAt time.Time) (RecommendationOutcome, error) {
	if err := caseFile.Validate(); err != nil {
		return RecommendationOutcome{}, err
	}
	if strings.TrimSpace(outcomeSource) == "" || recordedAt.IsZero() || recordedAt.Location() != time.UTC {
		return RecommendationOutcome{}, fmt.Errorf("outcome source and UTC recorded_at are required")
	}
	if err := validateOutcomeObservations(caseFile.DecisionAt, observations); err != nil {
		return RecommendationOutcome{}, err
	}
	if recordedAt.Before(caseFile.DecisionAt) || (len(observations) > 0 && recordedAt.Before(observations[len(observations)-1].At)) {
		return RecommendationOutcome{}, fmt.Errorf("outcome recorded_at must follow the decision and observations")
	}
	if invalidationAt != nil && (invalidationAt.IsZero() || invalidationAt.Location() != time.UTC || !invalidationAt.After(caseFile.DecisionAt) || invalidationAt.After(recordedAt)) {
		return RecommendationOutcome{}, fmt.Errorf("invalidation time must be UTC, later than decision and no later than recording")
	}
	windows := buildOutcomeWindows(caseFile.DecisionAt, observations, invalidationAt)
	outcome := RecommendationOutcome{ContractVersion: OutcomeTrackingContractV1, CaseID: caseFile.ID, RecommendationID: caseFile.Recommendation.ID, ResearchOutputID: caseFile.Research.ID, ContextPlanID: caseFile.ContextPlan.ID, DecisionAt: caseFile.DecisionAt, Disposition: string(caseFile.Recommendation.Disposition), OriginalThesis: caseFile.Recommendation.Thesis, OriginalInvalidationIDs: collectInvalidationIDs(caseFile.Recommendation), Windows: windows, RecordedAt: recordedAt, OutcomeSource: outcomeSource}
	outcome.ID = deriveOutcomeID(outcome)
	if err := outcome.Validate(caseFile); err != nil {
		return RecommendationOutcome{}, err
	}
	return outcome, nil
}

func (outcome RecommendationOutcome) Validate(caseFile HistoricalCase) error {
	if err := caseFile.Validate(); err != nil {
		return err
	}
	if outcome.ContractVersion != OutcomeTrackingContractV1 || !validIdentity("outcome_", outcome.ID) || outcome.CaseID != caseFile.ID || outcome.RecommendationID != caseFile.Recommendation.ID || outcome.ResearchOutputID != caseFile.Research.ID || outcome.ContextPlanID != caseFile.ContextPlan.ID || outcome.DecisionAt != caseFile.DecisionAt || outcome.Disposition != string(caseFile.Recommendation.Disposition) || outcome.OriginalThesis != caseFile.Recommendation.Thesis || strings.TrimSpace(outcome.OutcomeSource) == "" || outcome.RecordedAt.IsZero() || outcome.RecordedAt.Location() != time.UTC || len(outcome.Windows) == 0 {
		return fmt.Errorf("outcome does not preserve the original recommendation identity")
	}
	if !slices.Equal(outcome.OriginalInvalidationIDs, collectInvalidationIDs(caseFile.Recommendation)) {
		return fmt.Errorf("outcome does not preserve original invalidation evidence identity")
	}
	for _, window := range outcome.Windows {
		if err := window.Validate(outcome.DecisionAt); err != nil {
			return err
		}
	}
	if outcome.ID != deriveOutcomeID(outcome) {
		return fmt.Errorf("outcome ID does not match its immutable contents")
	}
	return nil
}

func validateOutcomeObservations(decisionAt time.Time, observations []OutcomeObservation) error {
	if len(observations) == 0 {
		return fmt.Errorf("outcome observations are required; missing data cannot become zero returns")
	}
	for index, observation := range observations {
		if err := observation.Validate(); err != nil {
			return err
		}
		if !observation.At.After(decisionAt) || index > 0 && !observation.At.After(observations[index-1].At) {
			return fmt.Errorf("outcome observations must be strictly after the decision and ordered")
		}
	}
	return nil
}

func buildOutcomeWindows(decisionAt time.Time, observations []OutcomeObservation, invalidationAt *time.Time) []OutcomeWindowResult {
	windows := []OutcomeWindowResult{
		buildObservationWindow(OutcomeNextObservation, decisionAt, observations, 1),
		buildTimeWindow(OutcomeOneDay, decisionAt, observations, 24*time.Hour),
		buildObservationWindow(OutcomeFiveObservations, decisionAt, observations, 5),
		buildObservationWindow(OutcomeTwentyObservations, decisionAt, observations, 20),
	}
	if invalidationAt != nil {
		window := OutcomeWindowResult{Horizon: OutcomeInvalidation, Complete: true, InvalidationReached: true, DirectionalOutcome: "NOT_APPLICABLE"}
		for _, observation := range observations {
			if !observation.At.After(*invalidationAt) {
				window.EndPrice = observation.Price
			}
		}
		if window.EndPrice > 0 {
			window.StartAt = observations[0].At
			for _, observation := range observations {
				if !observation.At.After(*invalidationAt) {
					window.EndAt = observation.At
				}
			}
			window.StartPrice = observations[0].Price
			value := window.EndPrice/window.StartPrice - 1
			window.Return = &value
			window.MaximumFavourableExcursion, window.MaximumAdverseExcursion = excursion(observations, *window.Return, observations[0].Price)
		} else {
			window.Complete = false
			window.MissingReason = "no observation available by invalidation time"
		}
		windows = append(windows, window)
	}
	return windows
}

func buildObservationWindow(horizon OutcomeHorizon, decisionAt time.Time, observations []OutcomeObservation, count int) OutcomeWindowResult {
	window := OutcomeWindowResult{Horizon: horizon, StartAt: decisionAt, DirectionalOutcome: "NOT_APPLICABLE"}
	if len(observations) < count {
		window.MissingReason = fmt.Sprintf("requires %d post-decision observations", count)
		return window
	}
	return completeWindow(window, observations, observations[count-1], observations[:count])
}

func buildTimeWindow(horizon OutcomeHorizon, decisionAt time.Time, observations []OutcomeObservation, duration time.Duration) OutcomeWindowResult {
	window := OutcomeWindowResult{Horizon: horizon, StartAt: decisionAt, DirectionalOutcome: "NOT_APPLICABLE"}
	for index, observation := range observations {
		if !observation.At.Before(decisionAt.Add(duration)) {
			return completeWindow(window, observations, observation, observations[:index+1])
		}
	}
	window.MissingReason = fmt.Sprintf("requires an observation at least %s after the decision", duration)
	return window
}

func completeWindow(window OutcomeWindowResult, observations []OutcomeObservation, end OutcomeObservation, rangeObservations []OutcomeObservation) OutcomeWindowResult {
	window.EndAt = end.At
	if len(rangeObservations) > 1 {
		window.StartAt = observations[0].At
	}
	window.StartPrice = observations[0].Price
	window.EndPrice = end.Price
	value := end.Price/window.StartPrice - 1
	window.Return = &value
	window.MaximumFavourableExcursion, window.MaximumAdverseExcursion = excursion(rangeObservations, value, window.StartPrice)
	var benchmarkStart, benchmarkEnd float64
	if observations[0].BenchmarkPrice != nil && end.BenchmarkPrice != nil {
		benchmarkStart, benchmarkEnd = *observations[0].BenchmarkPrice, *end.BenchmarkPrice
		relative := (end.Price/window.StartPrice - 1) - (benchmarkEnd/benchmarkStart - 1)
		window.BenchmarkRelativeReturn = &relative
	}
	window.Complete = true
	return window
}

func excursion(observations []OutcomeObservation, _ float64, startPrice float64) (*float64, *float64) {
	maximum := math.Inf(-1)
	minimum := math.Inf(1)
	for _, observation := range observations {
		change := observation.Price/startPrice - 1
		maximum = math.Max(maximum, change)
		minimum = math.Min(minimum, change)
	}
	return &maximum, &minimum
}

func collectInvalidationIDs(recommendation researchrecommendation.ResearchRecommendation) []string {
	ids := []string{}
	for _, condition := range recommendation.InvalidationConditions {
		ids = append(ids, condition.EvidenceIDs...)
	}
	sort.Strings(ids)
	return ids
}

func deriveOutcomeID(outcome RecommendationOutcome) string {
	outcome.ID = ""
	seed, _ := json.Marshal(outcome)
	digest := sha256.Sum256(seed)
	return "outcome_" + hex.EncodeToString(digest[:])
}
