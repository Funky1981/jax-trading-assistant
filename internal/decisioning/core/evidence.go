package core

import "time"

const VersionDecisionCoreV1 = "decision-core-v1"

type EvidenceBundle struct {
	EvidenceID            string         `json:"evidence_id"`
	InputEvent            Event          `json:"input_event"`
	MarketContext         map[string]any `json:"market_context"`
	ReasoningTraceSummary string         `json:"reasoning_trace_summary"`
	Scores                Scores         `json:"scores"`
	FinalDecision         Decision       `json:"final_decision"`
	GeneratedAt           time.Time      `json:"generated_at"`
	Version               string         `json:"version"`
}
