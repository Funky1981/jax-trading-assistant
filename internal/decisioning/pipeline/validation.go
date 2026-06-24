package pipeline

import (
	"fmt"
	"time"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/research"
)

func validateInput(input Input) (time.Time, []string, []string) {
	var errors []string
	var warnings []string

	if input.Event.EventID == "" {
		errors = append(errors, "event_id is required")
	}
	if input.Event.Headline == "" && input.Event.Summary == "" {
		warnings = append(warnings, "event headline or summary is missing")
	}
	if len(input.MarketContext) == 0 {
		warnings = append(warnings, "market context is missing")
	}
	if input.PortfolioContext == nil {
		warnings = append(warnings, "portfolio context is missing")
	}
	if input.ResearchEvidence == nil {
		warnings = append(warnings, "research evidence is missing")
	}

	now := input.CurrentTime
	if now.IsZero() {
		now = input.Event.ReceivedAt
	}
	if now.IsZero() {
		now = time.Unix(0, 0).UTC()
	}

	return now, errors, warnings
}

func validateResearchEvidence(evidence *research.BacktestEvidence) *research.ValidationResult {
	if evidence == nil {
		return nil
	}
	result := research.ValidateBacktestEvidence(*evidence)
	return &result
}

func evidenceSufficient(result *research.ValidationResult) bool {
	if result == nil || !result.IsValid {
		return false
	}
	return result.PromotionDecision == research.PromotionBacktestedPromising ||
		result.PromotionDecision == research.PromotionPaperReady
}

func mandatoryForbiddenActions() []string {
	return []string{
		core.ActionExecuteTrade,
		core.ActionCreateLiveOrder,
		core.ActionAutoApprove,
	}
}

func normaliseForbiddenActions(actions ...[]string) []string {
	merged := appendUnique(nil, mandatoryForbiddenActions()...)
	for _, set := range actions {
		merged = appendUnique(merged, set...)
	}
	return merged
}

func safeAllowedActions(actions []string) []string {
	var allowed []string
	for _, action := range actions {
		if action == core.ActionExecuteTrade || action == core.ActionCreateLiveOrder || action == core.ActionAutoApprove {
			continue
		}
		allowed = appendUnique(allowed, action)
	}
	return allowed
}

func appendUnique(base []string, values ...string) []string {
	for _, value := range values {
		if value == "" {
			continue
		}
		found := false
		for _, existing := range base {
			if existing == value {
				found = true
				break
			}
		}
		if !found {
			base = append(base, value)
		}
	}
	return base
}

func pipelineID(eventID string) string {
	if eventID == "" {
		return "pipe_invalid"
	}
	return fmt.Sprintf("pipe_%s", eventID)
}
