package macroevents

import (
	"context"
	"encoding/json"
	"strings"
)

type EvidenceVerdict string

const (
	EvidenceVerdictCandidateAllowed     EvidenceVerdict = "candidate_allowed"
	EvidenceVerdictCandidateBlocked     EvidenceVerdict = "candidate_blocked"
	EvidenceVerdictWatchOnly            EvidenceVerdict = "watch_only"
	EvidenceVerdictInsufficientEvidence EvidenceVerdict = "insufficient_evidence"
)

type EvidenceInput struct {
	MacroEvent           EventInput
	Scenario             ScenarioEvaluation
	Reaction             ReactionSnapshot
	PricedIn             PricedInScore
	Confounders          []Confounder
	HistoricalComparison string
	RiskGuardrail        string
	EntryStopTarget      string
	Symbol               string
}

type EvidenceBundle struct {
	ID              string
	MacroEventID    string
	Symbol          string
	Status          string
	Verdict         EvidenceVerdict
	Summary         string
	Evidence        map[string]any
	MissingEvidence []string
	WalkawayReasons []string
}

func BuildEvidenceBundle(input EvidenceInput) EvidenceBundle {
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	if symbol == "" {
		symbol = strings.ToUpper(strings.TrimSpace(input.Reaction.Symbol))
	}
	bundle := EvidenceBundle{
		MacroEventID: input.MacroEvent.MacroEventID,
		Symbol:       symbol,
		Status:       "ready",
		Verdict:      EvidenceVerdictCandidateAllowed,
		Evidence: map[string]any{
			"what_happened":         input.MacroEvent.Headline,
			"structured_macro_data": input.MacroEvent,
			"why_this_etf":          input.Scenario.PrimarySymbols,
			"expected_reaction":     input.Scenario.ExpectedReactions,
			"actual_chart_reaction": input.Reaction,
			"priced_in_verdict":     input.PricedIn,
			"confounders":           input.Confounders,
			"historical_comparison": input.HistoricalComparison,
			"risk_guardrail_result": input.RiskGuardrail,
			"entry_stop_target":     input.EntryStopTarget,
			"beginner_summary":      beginnerSummary(input),
		},
	}

	if input.MacroEvent.MacroEventID == "" {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "macro event")
	}
	if input.Scenario.Result != ScenarioResultEligibleForReactionCheck {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "eligible scenario")
	}
	if input.Reaction.Status != ReactionStatusAvailable {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "chart reaction")
	}
	if input.PricedIn.Verdict == "" {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "priced-in verdict")
	}
	if strings.TrimSpace(input.HistoricalComparison) == "" {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "historical comparison")
	}
	if strings.TrimSpace(input.RiskGuardrail) == "" {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "risk guardrail")
	}
	if strings.TrimSpace(input.EntryStopTarget) == "" {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "entry/stop/target proposal")
	}

	if input.Reaction.TooExtended {
		bundle.Verdict = EvidenceVerdictWatchOnly
		bundle.WalkawayReasons = append(bundle.WalkawayReasons, "reaction is too extended for immediate candidate")
	}
	if input.PricedIn.BlocksCandidate {
		bundle.Verdict = EvidenceVerdictCandidateBlocked
		bundle.WalkawayReasons = append(bundle.WalkawayReasons, "priced-in verdict blocks candidate")
	}
	for _, confounder := range input.Confounders {
		if confounder.BlocksCandidate {
			bundle.Verdict = EvidenceVerdictCandidateBlocked
			bundle.WalkawayReasons = append(bundle.WalkawayReasons, "high-severity confounder blocks candidate")
			break
		}
	}
	if input.Reaction.Status == ReactionStatusAvailable && !input.Reaction.ConfirmsEvent && !input.Reaction.TooExtended {
		bundle.Verdict = EvidenceVerdictCandidateBlocked
		bundle.WalkawayReasons = append(bundle.WalkawayReasons, "chart reaction does not confirm scenario")
	}
	if hasCriticalMissingEvidence(bundle.MissingEvidence) {
		bundle.Verdict = EvidenceVerdictInsufficientEvidence
		bundle.Status = "incomplete"
	}

	bundle.Summary = evidenceSummary(bundle)
	return bundle
}

func hasCriticalMissingEvidence(missing []string) bool {
	for _, item := range missing {
		switch item {
		case "macro event", "eligible scenario", "chart reaction", "priced-in verdict", "risk guardrail", "entry/stop/target proposal":
			return true
		}
	}
	return false
}

func beginnerSummary(input EvidenceInput) string {
	return "Jax checked the macro event, ETF mapping, chart reaction, priced-in risk, and confounders before allowing any paper-only candidate."
}

func evidenceSummary(bundle EvidenceBundle) string {
	switch bundle.Verdict {
	case EvidenceVerdictCandidateAllowed:
		return "Macro evidence supports a paper-only candidate subject to human approval."
	case EvidenceVerdictWatchOnly:
		return "Macro evidence supports watch-only handling because the reaction is extended."
	case EvidenceVerdictCandidateBlocked:
		return "Macro evidence blocks candidate creation."
	default:
		return "Macro evidence is incomplete."
	}
}

type evidenceStore interface {
	SaveEvidenceBundle(ctx context.Context, bundle EvidenceBundle) (EvidenceBundle, error)
}

type EvidenceService struct {
	store evidenceStore
}

func NewEvidenceService(store evidenceStore) *EvidenceService {
	return &EvidenceService{store: store}
}

func (s *EvidenceService) BuildAndSave(ctx context.Context, input EvidenceInput) (EvidenceBundle, error) {
	bundle := BuildEvidenceBundle(input)
	if s.store == nil {
		return bundle, nil
	}
	return s.store.SaveEvidenceBundle(ctx, bundle)
}

func MarshalEvidence(evidence map[string]any) ([]byte, error) {
	if evidence == nil {
		evidence = map[string]any{}
	}
	return json.Marshal(evidence)
}
