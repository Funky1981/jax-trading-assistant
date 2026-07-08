package candidates

import (
	"strings"
	"time"
)

const (
	RiskStatusPending = "pending"

	ApprovalStatusNotReady = "not_ready"

	GateStatusNotEvaluated = "not_evaluated"
	GateStatusBlocked      = "blocked"
	GateStatusReady        = "ready"
)

type StructuredCandidateFields struct {
	Source                       string
	InstrumentType               string
	Venue                        *string
	Currency                     *string
	SetupType                    string
	Direction                    string
	TimeHorizon                  string
	StrategyFamily               *string
	CandidateReasonSummary       *string
	CatalystType                 *string
	CatalystSummary              string
	CatalystSource               *string
	CatalystTimestamp            *time.Time
	CatalystConfidence           *float64
	SupportingEvidenceSummary    *string
	ContradictoryEvidenceSummary *string
	EvidenceSourceCount          *int
	HasContradictoryEvidence     bool
	InvalidationReason           string
	ExpectedRewardRiskRatio      *float64
	SlippageAllowance            *float64
	MaxNormalLoss                *float64
	MaxSlippageAdjustedLoss      *float64
	PositionSize                 *float64
	RiskStatus                   string
	ApprovalStatus               string
	RejectReasons                []string
	GateStatus                   string
	ModelVersion                 *string
	GeneratorVersion             *string
	RawSourceRef                 *string
	SourcePayloadRef             *string
	DecisionLogRef               *string
}

type StructuralValidation struct {
	StructurallyComplete bool     `json:"structurallyComplete"`
	GateReady            bool     `json:"gateReady"`
	RiskStatus           string   `json:"riskStatus"`
	GateStatus           string   `json:"gateStatus"`
	MissingFields        []string `json:"missingFields,omitempty"`
	RejectReasons        []string `json:"rejectReasons,omitempty"`

	// BrokerExecutionAllowed is intentionally false in this phase. Candidate
	// structure can prepare review data, but must not authorize order placement.
	BrokerExecutionAllowed bool `json:"brokerExecutionAllowed"`
}

func applyStructuredProposalFields(c *Candidate, fields StructuredCandidateFields) {
	c.Source = firstNonEmpty(fields.Source, c.DataProvenance)
	c.InstrumentType = fields.InstrumentType
	c.Venue = fields.Venue
	c.Currency = fields.Currency
	c.SetupType = fields.SetupType
	c.Direction = firstNonEmpty(fields.Direction, directionFromSignalType(c.SignalType))
	c.TimeHorizon = fields.TimeHorizon
	c.StrategyFamily = fields.StrategyFamily
	c.CandidateReasonSummary = firstStringPtr(fields.CandidateReasonSummary, c.Reasoning)
	c.CatalystType = fields.CatalystType
	c.CatalystSummary = fields.CatalystSummary
	c.CatalystSource = fields.CatalystSource
	c.CatalystTimestamp = fields.CatalystTimestamp
	c.CatalystConfidence = fields.CatalystConfidence
	c.SupportingEvidenceSummary = fields.SupportingEvidenceSummary
	c.ContradictoryEvidenceSummary = fields.ContradictoryEvidenceSummary
	c.EvidenceSourceCount = fields.EvidenceSourceCount
	c.HasContradictoryEvidence = fields.HasContradictoryEvidence
	c.InvalidationReason = fields.InvalidationReason
	c.ExpectedRewardRiskRatio = fields.ExpectedRewardRiskRatio
	c.SlippageAllowance = fields.SlippageAllowance
	c.MaxNormalLoss = fields.MaxNormalLoss
	c.MaxSlippageAdjustedLoss = fields.MaxSlippageAdjustedLoss
	c.PositionSize = fields.PositionSize
	c.RiskStatus = normalizedRiskStatus(fields.RiskStatus)
	c.HumanApprovalRequired = true
	c.ApprovalStatus = firstNonEmpty(fields.ApprovalStatus, ApprovalStatusNotReady)
	c.RejectReasons = fields.RejectReasons
	c.GateStatus = firstNonEmpty(fields.GateStatus, GateStatusNotEvaluated)
	c.ModelVersion = fields.ModelVersion
	c.GeneratorVersion = fields.GeneratorVersion
	c.RawSourceRef = fields.RawSourceRef
	c.SourcePayloadRef = fields.SourcePayloadRef
	c.DecisionLogRef = fields.DecisionLogRef
}

func ValidateStructuralCompleteness(c Candidate) StructuralValidation {
	result := StructuralValidation{
		StructurallyComplete:   true,
		GateReady:              true,
		RiskStatus:             normalizedRiskStatus(c.RiskStatus),
		GateStatus:             GateStatusReady,
		BrokerExecutionAllowed: false,
	}

	required := []struct {
		name    string
		missing bool
	}{
		{name: "symbol", missing: strings.TrimSpace(c.Symbol) == ""},
		{name: "setup_type", missing: strings.TrimSpace(c.SetupType) == ""},
		{name: "direction", missing: strings.TrimSpace(c.Direction) == ""},
		{name: "catalyst_summary", missing: strings.TrimSpace(c.CatalystSummary) == ""},
		{name: "proposed_entry_price", missing: c.EntryPrice == nil || *c.EntryPrice <= 0},
		{name: "stop_loss_price", missing: c.StopLoss == nil || *c.StopLoss <= 0},
		{name: "invalidation_reason", missing: strings.TrimSpace(c.InvalidationReason) == ""},
	}
	for _, field := range required {
		if field.missing {
			result.MissingFields = append(result.MissingFields, field.name)
		}
	}
	if len(result.MissingFields) > 0 {
		result.StructurallyComplete = false
		result.GateReady = false
		result.GateStatus = GateStatusBlocked
		result.RejectReasons = append(result.RejectReasons, "structural_fields_missing")
	}

	if c.HasContradictoryEvidence {
		result.GateReady = false
		result.GateStatus = GateStatusBlocked
		result.RejectReasons = append(result.RejectReasons, "contradictory_evidence_present")
	}

	return result
}

func normalizedRiskStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return RiskStatusPending
	}
	return status
}

func directionFromSignalType(signalType string) string {
	switch strings.ToUpper(strings.TrimSpace(signalType)) {
	case "BUY", "LONG":
		return "long"
	case "SELL", "SHORT":
		return "short"
	default:
		return strings.ToLower(strings.TrimSpace(signalType))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstStringPtr(values ...*string) *string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			return value
		}
	}
	return nil
}
