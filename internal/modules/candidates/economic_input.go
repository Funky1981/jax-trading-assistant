package candidates

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

const CandidateEconomicInputContractVersion = "candidate-economic-input-v1"

// CandidateEconomicInput records the exposure request and its explicit
// identity/policy provenance. It deliberately contains no account quantity.
type CandidateEconomicInput struct {
	CandidateID           uuid.UUID `json:"candidate_id"`
	ContractVersion       string    `json:"contract_version"`
	InstrumentID          string    `json:"instrument_id"`
	IssuerID              string    `json:"issuer_id"`
	IdentitySource        string    `json:"identity_source"`
	IdentityPolicyVersion string    `json:"identity_policy_version"`
	RiskAllocation        float64   `json:"risk_allocation"`
	RequestedLeverage     float64   `json:"requested_leverage"`
	SlippageAllowance     *float64  `json:"slippage_allowance,omitempty"`
	SizingPolicyID        string    `json:"sizing_policy_id"`
	SizingPolicyVersion   string    `json:"sizing_policy_version"`
	ContentIdentity       string    `json:"content_identity"`
	CreatedAt             time.Time `json:"created_at"`
}

func BuildCandidateEconomicInput(input CandidateEconomicInput, now time.Time) (CandidateEconomicInput, error) {
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	if input.ContractVersion == "" {
		input.ContractVersion = CandidateEconomicInputContractVersion
	}
	input.InstrumentID = strings.TrimSpace(input.InstrumentID)
	input.IssuerID = strings.TrimSpace(input.IssuerID)
	input.IdentitySource = strings.TrimSpace(input.IdentitySource)
	input.IdentityPolicyVersion = strings.TrimSpace(input.IdentityPolicyVersion)
	input.SizingPolicyID = strings.TrimSpace(input.SizingPolicyID)
	input.SizingPolicyVersion = strings.TrimSpace(input.SizingPolicyVersion)
	if input.CandidateID == uuid.Nil || input.ContractVersion != CandidateEconomicInputContractVersion ||
		input.InstrumentID == "" || input.IssuerID == "" || input.IdentitySource == "" || input.IdentityPolicyVersion == "" ||
		input.SizingPolicyID == "" || input.SizingPolicyVersion == "" ||
		math.IsNaN(input.RiskAllocation) || math.IsInf(input.RiskAllocation, 0) || input.RiskAllocation <= 0 || input.RiskAllocation > 1 ||
		math.IsNaN(input.RequestedLeverage) || math.IsInf(input.RequestedLeverage, 0) || input.RequestedLeverage <= 0 || input.RequestedLeverage > 1 {
		return CandidateEconomicInput{}, errors.New("candidate economic input is incomplete or outside policy bounds")
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = now.UTC()
	}
	if input.CreatedAt.IsZero() {
		return CandidateEconomicInput{}, errors.New("candidate economic input created_at is required")
	}
	input.CreatedAt = input.CreatedAt.UTC()
	identityFields := struct {
		CandidateID           uuid.UUID `json:"candidate_id"`
		ContractVersion       string    `json:"contract_version"`
		InstrumentID          string    `json:"instrument_id"`
		IssuerID              string    `json:"issuer_id"`
		IdentitySource        string    `json:"identity_source"`
		IdentityPolicyVersion string    `json:"identity_policy_version"`
		RiskAllocation        float64   `json:"risk_allocation"`
		RequestedLeverage     float64   `json:"requested_leverage"`
		SizingPolicyID        string    `json:"sizing_policy_id"`
		SizingPolicyVersion   string    `json:"sizing_policy_version"`
		SlippageAllowance     *float64  `json:"slippage_allowance,omitempty"`
	}{input.CandidateID, input.ContractVersion, input.InstrumentID, input.IssuerID, input.IdentitySource,
		input.IdentityPolicyVersion, input.RiskAllocation, input.RequestedLeverage, input.SizingPolicyID, input.SizingPolicyVersion, input.SlippageAllowance}
	if input.SlippageAllowance != nil && (math.IsNaN(*input.SlippageAllowance) || math.IsInf(*input.SlippageAllowance, 0) || *input.SlippageAllowance < 0) {
		return CandidateEconomicInput{}, errors.New("candidate economic input slippage allowance must be finite and non-negative")
	}
	canonical, err := json.Marshal(identityFields)
	if err != nil {
		return CandidateEconomicInput{}, fmt.Errorf("marshal candidate economic identity: %w", err)
	}
	hash := sha256.Sum256(canonical)
	computed := "sha256:" + hex.EncodeToString(hash[:])
	if input.ContentIdentity != "" && input.ContentIdentity != computed {
		return CandidateEconomicInput{}, errors.New("candidate economic input content identity mismatch")
	}
	input.ContentIdentity = computed
	return input, nil
}

func (input CandidateEconomicInput) SameEconomicRequest(other CandidateEconomicInput) bool {
	return input.ContentIdentity != "" && input.ContentIdentity == other.ContentIdentity
}
