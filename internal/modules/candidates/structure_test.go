package candidates

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStructuralCompletenessRequiresCoreReviewFields(t *testing.T) {
	cases := []struct {
		name         string
		mutate       func(*Candidate)
		missingField string
	}{
		{
			name: "symbol",
			mutate: func(c *Candidate) {
				c.Symbol = ""
			},
			missingField: "symbol",
		},
		{
			name: "setup type",
			mutate: func(c *Candidate) {
				c.SetupType = ""
			},
			missingField: "setup_type",
		},
		{
			name: "direction",
			mutate: func(c *Candidate) {
				c.Direction = ""
			},
			missingField: "direction",
		},
		{
			name: "catalyst summary",
			mutate: func(c *Candidate) {
				c.CatalystSummary = ""
			},
			missingField: "catalyst_summary",
		},
		{
			name: "proposed entry price",
			mutate: func(c *Candidate) {
				c.EntryPrice = nil
			},
			missingField: "proposed_entry_price",
		},
		{
			name: "stop loss",
			mutate: func(c *Candidate) {
				c.StopLoss = nil
			},
			missingField: "stop_loss_price",
		},
		{
			name: "invalidation reason",
			mutate: func(c *Candidate) {
				c.InvalidationReason = ""
			},
			missingField: "invalidation_reason",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := completeStructuredCandidate()
			tc.mutate(&candidate)

			result := ValidateStructuralCompleteness(candidate)

			if result.StructurallyComplete {
				t.Fatalf("expected candidate missing %s to be structurally incomplete", tc.missingField)
			}
			if !containsString(result.MissingFields, tc.missingField) {
				t.Fatalf("missing fields = %#v, want %q", result.MissingFields, tc.missingField)
			}
			if result.GateReady {
				t.Fatal("structurally incomplete candidate must not be gate-ready")
			}
		})
	}
}

func TestStructuralCompletenessMissingStopLossIsIncomplete(t *testing.T) {
	candidate := completeStructuredCandidate()
	candidate.StopLoss = nil

	result := ValidateStructuralCompleteness(candidate)

	if result.StructurallyComplete {
		t.Fatal("candidate without stop loss must be structurally incomplete")
	}
	if !containsString(result.MissingFields, "stop_loss_price") {
		t.Fatalf("missing fields = %#v, want stop_loss_price", result.MissingFields)
	}
}

func TestStructuralCompletenessMissingCatalystIsIncomplete(t *testing.T) {
	candidate := completeStructuredCandidate()
	candidate.CatalystSummary = ""

	result := ValidateStructuralCompleteness(candidate)

	if result.StructurallyComplete {
		t.Fatal("candidate without catalyst summary must be structurally incomplete")
	}
	if !containsString(result.MissingFields, "catalyst_summary") {
		t.Fatalf("missing fields = %#v, want catalyst_summary", result.MissingFields)
	}
}

func TestStructuralCompletenessContradictoryEvidenceIsNotGateReady(t *testing.T) {
	candidate := completeStructuredCandidate()
	candidate.HasContradictoryEvidence = true
	candidate.ContradictoryEvidenceSummary = stringPtr("earnings guide-down conflicts with momentum setup")

	result := ValidateStructuralCompleteness(candidate)

	if !result.StructurallyComplete {
		t.Fatalf("contradictory evidence should not make core structure incomplete: %#v", result.MissingFields)
	}
	if result.GateReady {
		t.Fatal("candidate with contradictory evidence must not be gate-ready")
	}
	if !containsString(result.RejectReasons, "contradictory_evidence_present") {
		t.Fatalf("reject reasons = %#v, want contradictory_evidence_present", result.RejectReasons)
	}
}

func TestStructuralCompletenessRiskFieldsMayBePending(t *testing.T) {
	candidate := completeStructuredCandidate()
	candidate.RiskStatus = ""
	candidate.SlippageAllowance = nil
	candidate.MaxNormalLoss = nil
	candidate.MaxSlippageAdjustedLoss = nil
	candidate.PositionSize = nil

	result := ValidateStructuralCompleteness(candidate)

	if !result.StructurallyComplete {
		t.Fatalf("unset risk placeholders should not break structural completeness: %#v", result.MissingFields)
	}
	if result.RiskStatus != RiskStatusPending {
		t.Fatalf("risk status = %q, want %q", result.RiskStatus, RiskStatusPending)
	}
}

func TestStructuralCompletenessDoesNotCreateBrokerExecution(t *testing.T) {
	candidate := completeStructuredCandidate()

	result := ValidateStructuralCompleteness(candidate)

	if result.BrokerExecutionAllowed {
		t.Fatal("structured candidate validation must not allow broker execution")
	}
	if candidate.ExecutionInstructionID != nil || candidate.TradeID != nil {
		t.Fatal("structured candidate validation must not create execution or trade identifiers")
	}
}

func completeStructuredCandidate() Candidate {
	entry := 101.25
	stop := 98.75
	target := 106.00
	confidence := 0.72
	catalystConfidence := 0.68
	sourceCount := 2
	now := time.Date(2026, 7, 8, 9, 30, 0, 0, time.UTC)

	return Candidate{
		ID:                        uuid.New(),
		StrategyInstanceID:        uuid.New(),
		Symbol:                    "SPY",
		SignalType:                "BUY",
		Source:                    "unit-test",
		InstrumentType:            "ETF",
		SetupType:                 "pullback_continuation",
		Direction:                 "long",
		TimeHorizon:               "swing",
		CandidateReasonSummary:    stringPtr("SPY reclaimed prior support after a shallow pullback."),
		CatalystType:              stringPtr("market_structure"),
		CatalystSummary:           "Broad-market ETF holding above key support after lower CPI reaction.",
		CatalystSource:            stringPtr("unit-test-feed"),
		CatalystTimestamp:         &now,
		CatalystConfidence:        &catalystConfidence,
		SupportingEvidenceSummary: stringPtr("Price reclaimed VWAP and breadth improved."),
		EvidenceSourceCount:       &sourceCount,
		EntryPrice:                &entry,
		StopLoss:                  &stop,
		TakeProfit:                &target,
		Confidence:                &confidence,
		InvalidationReason:        "Break and close below support invalidates the setup.",
		HumanApprovalRequired:     true,
		ApprovalStatus:            ApprovalStatusNotReady,
		GateStatus:                GateStatusNotEvaluated,
		RiskStatus:                RiskStatusPending,
		DataProvenance:            "unit-test",
		Status:                    StatusDetected,
		SessionDate:               "2026-07-08",
		DetectedAt:                now,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func stringPtr(value string) *string {
	return &value
}
