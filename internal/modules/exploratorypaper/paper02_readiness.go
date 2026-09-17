package exploratorypaper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

type ProspectiveEventSourceReadiness struct {
	Ready          bool   `json:"ready"`
	SourceIdentity string `json:"sourceIdentity"`
	Endpoint       string `json:"endpoint,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type RiskPolicyIdentity struct {
	Version                string  `json:"version"`
	Hash                   string  `json:"hash"`
	MaxRiskPerTrade        float64 `json:"maxRiskPerTrade"`
	MaxPositionPercentage  float64 `json:"maxPositionPercentage"`
	MaxLeverage            float64 `json:"maxLeverage"`
	MaxConcurrentPositions int     `json:"maxConcurrentPositions"`
	MaxPositionValue       float64 `json:"maxPositionValue"`
	MaxAggregateRisk       float64 `json:"maxAggregateRisk"`
	MaxConcentration       float64 `json:"maxConcentration"`
	MaxCorrelatedExposure  float64 `json:"maxCorrelatedExposure"`
	RiskKillBehavior       string  `json:"riskKillBehavior"`
}

func (p RiskPolicyIdentity) ContentHash() string {
	copy := p
	copy.Hash = ""
	data, _ := json.Marshal(copy)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

type EntryPolicyIdentity struct {
	Version                string   `json:"version"`
	Hash                   string   `json:"hash"`
	Mode                   string   `json:"mode"`
	CandidateRequirement   string   `json:"candidateRequirement"`
	EvidencePolicy         string   `json:"evidencePolicy"`
	RiskAcceptance         string   `json:"riskAcceptance"`
	HumanApproval          string   `json:"humanApproval"`
	PaperAcknowledgement   string   `json:"paperAcknowledgement"`
	ThesisContract         string   `json:"thesisContract"`
	SimulatedVenue         string   `json:"simulatedVenue"`
	ExecutionAuthority     string   `json:"executionAuthority"`
	BrokerExecutionAllowed bool     `json:"brokerExecutionAllowed"`
	MaximumLeverage        float64  `json:"maximumLeverage"`
	BoundFields            []string `json:"boundFields"`
}

type Paper02ReadinessInput struct {
	Calendar               *VersionedSessionCalendar
	CalendarError          error
	EventSource            ProspectiveEventSourceReadiness
	Universe               *EligibleUniverseManifest
	UniverseError          error
	RiskPolicy             *RiskPolicyIdentity
	RiskPolicyError        error
	EntryPolicy            *EntryPolicyIdentity
	EntryPolicyError       error
	RuntimeMode            string
	ExecutionAuthority     string
	BrokerExecutionAllowed bool
	MaximumLeverage        float64
}

type Paper02Readiness struct {
	CalendarReady               bool                      `json:"calendarReady"`
	CalendarVersion             string                    `json:"calendarVersion,omitempty"`
	CalendarHash                string                    `json:"calendarHash,omitempty"`
	ProspectiveEventIntakeReady bool                      `json:"prospectiveEventIntakeReady"`
	EventSourceIdentity         string                    `json:"eventSourceIdentity,omitempty"`
	EligibleUniverseReady       bool                      `json:"eligibleUniverseReady"`
	EligibleUniverseVersion     string                    `json:"eligibleUniverseVersion,omitempty"`
	EligibleUniverseHash        string                    `json:"eligibleUniverseHash,omitempty"`
	RiskPolicyReady             bool                      `json:"riskPolicyReady"`
	RiskPolicyVersion           string                    `json:"riskPolicyVersion,omitempty"`
	RiskPolicyHash              string                    `json:"riskPolicyHash,omitempty"`
	EntryPolicyReady            bool                      `json:"entryPolicyReady"`
	EntryPolicyVersion          string                    `json:"entryPolicyVersion,omitempty"`
	EntryPolicyHash             string                    `json:"entryPolicyHash,omitempty"`
	RuntimeMode                 string                    `json:"runtimeMode"`
	ExecutionAuthority          string                    `json:"executionAuthority"`
	BrokerExecutionAllowed      bool                      `json:"brokerExecutionAllowed"`
	MaximumLeverage             float64                   `json:"maximumLeverage"`
	OverallReady                bool                      `json:"overallReady"`
	BlockingReasons             []string                  `json:"blockingReasons"`
	RiskPolicy                  *RiskPolicyIdentity       `json:"riskPolicy,omitempty"`
	EntryPolicy                 *EntryPolicyIdentity      `json:"entryPolicy,omitempty"`
	EligibleUniverse            *EligibleUniverseManifest `json:"eligibleUniverse,omitempty"`
}

func AssessPaper02Readiness(input Paper02ReadinessInput) Paper02Readiness {
	result := Paper02Readiness{
		RuntimeMode:            input.RuntimeMode,
		ExecutionAuthority:     input.ExecutionAuthority,
		BrokerExecutionAllowed: input.BrokerExecutionAllowed,
		MaximumLeverage:        input.MaximumLeverage,
		BlockingReasons:        []string{},
		RiskPolicy:             input.RiskPolicy,
		EntryPolicy:            input.EntryPolicy,
		EligibleUniverse:       input.Universe,
	}
	if input.CalendarError != nil || input.Calendar == nil {
		result.BlockingReasons = append(result.BlockingReasons, "calendar is missing or invalid")
	} else if err := input.Calendar.Validate(); err != nil {
		result.BlockingReasons = append(result.BlockingReasons, "calendar is invalid: "+err.Error())
	} else {
		result.CalendarReady = true
		result.CalendarVersion = input.Calendar.Version
		result.CalendarHash = input.Calendar.ContentHash()
	}
	result.EventSourceIdentity = input.EventSource.SourceIdentity
	if !input.EventSource.Ready {
		reason := input.EventSource.Reason
		if reason == "" {
			reason = "prospective event intake is unavailable"
		}
		result.BlockingReasons = append(result.BlockingReasons, reason)
	} else {
		result.ProspectiveEventIntakeReady = true
	}
	if input.UniverseError != nil || input.Universe == nil {
		result.BlockingReasons = append(result.BlockingReasons, "eligible universe is missing or invalid")
	} else if err := input.Universe.Validate(); err != nil {
		result.BlockingReasons = append(result.BlockingReasons, "eligible universe is invalid: "+err.Error())
	} else if input.Calendar != nil && input.Universe.Market != input.Calendar.Market {
		result.BlockingReasons = append(result.BlockingReasons, "eligible universe is incompatible with the session calendar")
	} else {
		result.EligibleUniverseReady = true
		result.EligibleUniverseVersion = input.Universe.Version
		result.EligibleUniverseHash = input.Universe.ContentHash()
	}
	if input.RiskPolicyError != nil || input.RiskPolicy == nil {
		result.BlockingReasons = append(result.BlockingReasons, "risk policy identity is missing or invalid")
	} else if err := validateRiskPolicyIdentity(*input.RiskPolicy); err != nil {
		result.BlockingReasons = append(result.BlockingReasons, "risk policy is invalid: "+err.Error())
	} else {
		result.RiskPolicyReady = true
		result.RiskPolicyVersion, result.RiskPolicyHash = input.RiskPolicy.Version, input.RiskPolicy.Hash
	}
	if input.EntryPolicyError != nil || input.EntryPolicy == nil {
		result.BlockingReasons = append(result.BlockingReasons, "entry policy identity is missing or invalid")
	} else if err := input.EntryPolicy.Validate(); err != nil {
		result.BlockingReasons = append(result.BlockingReasons, "entry policy is invalid: "+err.Error())
	} else {
		result.EntryPolicyReady = true
		result.EntryPolicyVersion, result.EntryPolicyHash = input.EntryPolicy.Version, input.EntryPolicy.Hash
	}
	if input.RuntimeMode != "PAPER" {
		result.BlockingReasons = append(result.BlockingReasons, "runtime mode must be PAPER")
	}
	if input.ExecutionAuthority != "NONE" {
		result.BlockingReasons = append(result.BlockingReasons, "execution authority must be NONE")
	}
	if input.BrokerExecutionAllowed {
		result.BlockingReasons = append(result.BlockingReasons, "broker execution must be disabled")
	}
	if math.IsNaN(input.MaximumLeverage) || math.IsInf(input.MaximumLeverage, 0) || input.MaximumLeverage <= 0 || input.MaximumLeverage > 1 {
		result.BlockingReasons = append(result.BlockingReasons, "maximum leverage must be in (0,1]")
	}
	result.OverallReady = len(result.BlockingReasons) == 0
	return result
}

func validateRiskPolicyIdentity(policy RiskPolicyIdentity) error {
	if policy.Version == "" || !strings.HasPrefix(policy.Hash, "sha256:") || policy.MaxRiskPerTrade <= 0 || policy.MaxPositionPercentage <= 0 || policy.MaxLeverage <= 0 || policy.MaxLeverage > 1 || policy.MaxConcurrentPositions <= 0 || policy.MaxPositionValue <= 0 || policy.MaxAggregateRisk <= 0 || policy.MaxConcentration <= 0 || policy.MaxCorrelatedExposure <= 0 || policy.RiskKillBehavior == "" {
		return fmt.Errorf("risk policy fields are incomplete")
	}
	return nil
}

func (p EntryPolicyIdentity) Validate() error {
	if p.Version == "" || !strings.HasPrefix(p.Hash, "sha256:") || p.Mode != "PAPER" || p.CandidateRequirement != "CANDIDATE" || p.EvidencePolicy == "" || p.RiskAcceptance == "" || p.HumanApproval == "" || p.PaperAcknowledgement == "" || p.ThesisContract != ContractVersion || p.SimulatedVenue != "jax.paper.venue/v1" || p.ExecutionAuthority != "NONE" || p.BrokerExecutionAllowed || p.MaximumLeverage <= 0 || p.MaximumLeverage > 1 || len(p.BoundFields) == 0 {
		return fmt.Errorf("entry policy fields are incomplete or unsafe")
	}
	return nil
}

func HashJSONFile(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return "", "", err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256(canonical)
	hash := "sha256:" + hex.EncodeToString(digest[:])
	return hash, "content-" + hex.EncodeToString(digest[:8]), nil
}
