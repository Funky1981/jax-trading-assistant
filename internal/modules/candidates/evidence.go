package candidates

import (
	"math"
	"time"

	"github.com/google/uuid"
)

type EvidenceStatus string

const (
	EvidenceStatusMissing    EvidenceStatus = "missing"
	EvidenceStatusWeak       EvidenceStatus = "weak"
	EvidenceStatusMixed      EvidenceStatus = "mixed"
	EvidenceStatusSufficient EvidenceStatus = "sufficient"
	EvidenceStatusStale      EvidenceStatus = "stale"
	EvidenceStatusBlocked    EvidenceStatus = "blocked"
)

type FreshnessStatus string

const (
	FreshnessStatusFresh         FreshnessStatus = "fresh"
	FreshnessStatusStale         FreshnessStatus = "stale"
	FreshnessStatusCriticalStale FreshnessStatus = "critical_stale"
)

type EvidenceItem struct {
	EvidenceID           uuid.UUID       `json:"evidenceId"`
	CandidateID          uuid.UUID       `json:"candidateId"`
	SourceType           string          `json:"sourceType"`
	SourceRef            string          `json:"sourceRef"`
	ObservedAt           time.Time       `json:"observedAt"`
	Summary              string          `json:"summary"`
	EvidenceKind         string          `json:"evidenceKind"`
	SupportsCandidate    bool            `json:"supportsCandidate"`
	ContradictsCandidate bool            `json:"contradictsCandidate"`
	Confidence           float64         `json:"confidence"`
	ImpactScore          float64         `json:"impactScore"`
	QualityScore         float64         `json:"qualityScore"`
	FreshnessStatus      FreshnessStatus `json:"freshnessStatus"`
	Notes                *string         `json:"notes,omitempty"`
}

type EvidenceScoreSummary struct {
	CandidateID                 uuid.UUID      `json:"candidateId"`
	SupportScore                float64        `json:"supportScore"`
	ContradictionScore          float64        `json:"contradictionScore"`
	QualityScore                float64        `json:"qualityScore"`
	FreshnessScore              float64        `json:"freshnessScore"`
	OverallEvidenceScore        float64        `json:"overallEvidenceScore"`
	EvidenceItemCount           int            `json:"evidenceItemCount"`
	SupportingItemCount         int            `json:"supportingItemCount"`
	ContradictoryItemCount      int            `json:"contradictoryItemCount"`
	StaleItemCount              int            `json:"staleItemCount"`
	EvidenceStatus              EvidenceStatus `json:"evidenceStatus"`
	EvidenceReady               bool           `json:"evidenceReady"`
	EvidenceGateReady           bool           `json:"evidenceGateReady"`
	ApprovalGranted             bool           `json:"approvalGranted"`
	BrokerExecutionAllowed      bool           `json:"brokerExecutionAllowed"`
	ExecutionInstructionCreated bool           `json:"executionInstructionCreated"`
}

func ScoreEvidenceForCandidate(candidate Candidate, items []EvidenceItem, now time.Time) EvidenceScoreSummary {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	summary := EvidenceScoreSummary{
		CandidateID:                 candidate.ID,
		EvidenceStatus:              EvidenceStatusMissing,
		ApprovalGranted:             false,
		BrokerExecutionAllowed:      false,
		ExecutionInstructionCreated: false,
	}

	structural := ValidateStructuralCompleteness(candidate)
	if !structural.StructurallyComplete {
		summary.EvidenceStatus = EvidenceStatusBlocked
		return summary
	}
	if len(items) == 0 {
		return summary
	}

	var supportTotal, contradictionTotal, qualityTotal, freshnessTotal float64
	for _, item := range items {
		itemScore := bounded01(item.Confidence) * bounded01(item.ImpactScore) * bounded01(item.QualityScore)
		freshness := evidenceFreshnessScore(item, now)

		summary.EvidenceItemCount++
		qualityTotal += bounded01(item.QualityScore)
		freshnessTotal += freshness

		if item.SupportsCandidate {
			summary.SupportingItemCount++
			supportTotal += itemScore * freshness
		}
		if item.ContradictsCandidate {
			summary.ContradictoryItemCount++
			contradictionTotal += itemScore * freshness
		}
		if item.FreshnessStatus == FreshnessStatusStale || item.FreshnessStatus == FreshnessStatusCriticalStale || freshness < 0.50 {
			summary.StaleItemCount++
		}
	}

	count := float64(summary.EvidenceItemCount)
	summary.SupportScore = round3(supportTotal)
	summary.ContradictionScore = round3(contradictionTotal)
	summary.QualityScore = round3(qualityTotal / count)
	summary.FreshnessScore = round3(freshnessTotal / count)
	summary.OverallEvidenceScore = round3(clamp01(supportTotal-contradictionTotal*0.75) * summary.QualityScore * summary.FreshnessScore)
	summary.EvidenceStatus = classifyEvidenceStatus(summary)
	summary.EvidenceReady = summary.EvidenceStatus == EvidenceStatusSufficient
	summary.EvidenceGateReady = summary.EvidenceReady
	return summary
}

func classifyEvidenceStatus(summary EvidenceScoreSummary) EvidenceStatus {
	if summary.EvidenceItemCount == 0 {
		return EvidenceStatusMissing
	}
	if summary.StaleItemCount > 0 {
		return EvidenceStatusStale
	}
	if summary.ContradictoryItemCount > 0 && summary.SupportingItemCount == 0 {
		return EvidenceStatusBlocked
	}
	if summary.ContradictoryItemCount > 0 {
		return EvidenceStatusMixed
	}
	if summary.OverallEvidenceScore >= 0.60 && summary.QualityScore >= 0.70 && summary.FreshnessScore >= 0.70 {
		return EvidenceStatusSufficient
	}
	return EvidenceStatusWeak
}

func evidenceFreshnessScore(item EvidenceItem, now time.Time) float64 {
	switch item.FreshnessStatus {
	case FreshnessStatusCriticalStale:
		return 0
	case FreshnessStatusStale:
		return 0.25
	case FreshnessStatusFresh:
		return 1
	}
	if item.ObservedAt.IsZero() {
		return 0.50
	}
	age := now.Sub(item.ObservedAt)
	if age < 0 {
		age = 0
	}
	switch {
	case age <= 24*time.Hour:
		return 1
	case age <= 48*time.Hour:
		return 0.65
	default:
		return 0.25
	}
}

func bounded01(value float64) float64 {
	return clamp01(value)
}

func clamp01(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}
