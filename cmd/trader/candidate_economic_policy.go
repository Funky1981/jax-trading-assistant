package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	candidatesmod "jax-trading-assistant/internal/modules/candidates"
)

const candidateEconomicPolicyEnv = "JAX_CANDIDATE_ECONOMIC_POLICY_FILE"

type candidateEconomicPolicy struct {
	PolicyVersion       string                               `json:"policy_version"`
	IdentityPolicy      string                               `json:"identity_policy_version"`
	IdentitySource      string                               `json:"identity_source"`
	SizingPolicyID      string                               `json:"sizing_policy_id"`
	SizingPolicyVersion string                               `json:"sizing_policy_version"`
	RiskAllocation      float64                              `json:"risk_allocation"`
	RequestedLeverage   float64                              `json:"requested_leverage"`
	SlippageAllowance   *float64                             `json:"slippage_allowance"`
	Instruments         map[string]candidateEconomicIdentity `json:"instruments"`
}

type candidateEconomicIdentity struct {
	InstrumentID string `json:"instrument_id"`
	IssuerID     string `json:"issuer_id"`
}

// instrumentSymbol resolves canonical economic identity only through the
// explicit instrument map. Both canonical handoff and PAPER execution quote
// lookup use this resolver so symbols are never inferred from IDs.
func (p candidateEconomicPolicy) instrumentSymbol(instrumentID string) (string, error) {
	instrumentID = strings.TrimSpace(instrumentID)
	if instrumentID == "" {
		return "", errors.New("canonical_economic_inputs_unavailable: instrument identity is empty")
	}
	var matches []string
	for symbol, identity := range p.Instruments {
		if strings.TrimSpace(identity.InstrumentID) == instrumentID {
			matches = append(matches, strings.ToUpper(strings.TrimSpace(symbol)))
		}
	}
	sort.Strings(matches)
	if len(matches) != 1 || matches[0] == "" {
		return "", fmt.Errorf("canonical_economic_inputs_unavailable: instrument %q does not have one explicit symbol mapping", instrumentID)
	}
	return matches[0], nil
}

func (p candidateEconomicPolicy) frozenInstrumentSymbol(instrumentID, frozenSymbol string) (string, error) {
	symbol, err := p.instrumentSymbol(instrumentID)
	if err != nil {
		return "", err
	}
	frozenSymbol = strings.ToUpper(strings.TrimSpace(frozenSymbol))
	if frozenSymbol == "" || frozenSymbol != symbol {
		return "", fmt.Errorf("canonical_economic_inputs_unavailable: current symbol mapping %q differs from frozen market symbol %q", symbol, frozenSymbol)
	}
	return symbol, nil
}

func loadCandidateEconomicPolicy() (candidateEconomicPolicy, error) {
	path := strings.TrimSpace(os.Getenv(candidateEconomicPolicyEnv))
	if path == "" {
		return candidateEconomicPolicy{}, errors.New("canonical_economic_inputs_unavailable: explicit policy file is not configured")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return candidateEconomicPolicy{}, fmt.Errorf("canonical_economic_inputs_unavailable: read policy: %w", err)
	}
	var policy candidateEconomicPolicy
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return candidateEconomicPolicy{}, fmt.Errorf("canonical_economic_inputs_unavailable: decode policy: %w", err)
	}
	if strings.TrimSpace(policy.PolicyVersion) == "" || strings.TrimSpace(policy.IdentityPolicy) == "" ||
		strings.TrimSpace(policy.IdentitySource) == "" || strings.TrimSpace(policy.SizingPolicyID) == "" ||
		strings.TrimSpace(policy.SizingPolicyVersion) == "" || len(policy.Instruments) == 0 || policy.SlippageAllowance == nil || !finiteNonNegative(*policy.SlippageAllowance) {
		return candidateEconomicPolicy{}, errors.New("canonical_economic_inputs_unavailable: policy provenance and explicit instrument mappings are required")
	}
	for symbol, identity := range policy.Instruments {
		if strings.TrimSpace(symbol) == "" || strings.TrimSpace(identity.InstrumentID) == "" || strings.TrimSpace(identity.IssuerID) == "" {
			return candidateEconomicPolicy{}, errors.New("canonical_economic_inputs_unavailable: every instrument mapping must include symbol, instrument ID, and issuer ID")
		}
	}
	return policy, nil
}

func (p candidateEconomicPolicy) build(symbol string, candidateID uuid.UUID, now time.Time) (candidatesmod.CandidateEconomicInput, error) {
	identity, ok := p.Instruments[strings.ToUpper(strings.TrimSpace(symbol))]
	if !ok || strings.TrimSpace(identity.InstrumentID) == "" || strings.TrimSpace(identity.IssuerID) == "" {
		return candidatesmod.CandidateEconomicInput{}, errors.New("canonical_economic_inputs_unavailable: explicit instrument and issuer identity mapping is missing")
	}
	if p.RiskAllocation <= 0 || p.RequestedLeverage <= 0 || strings.TrimSpace(p.PolicyVersion) == "" || p.SlippageAllowance == nil || !finiteNonNegative(*p.SlippageAllowance) {
		return candidatesmod.CandidateEconomicInput{}, errors.New("canonical_economic_inputs_unavailable: explicit sizing request is missing")
	}
	return candidatesmod.BuildCandidateEconomicInput(candidatesmod.CandidateEconomicInput{
		CandidateID: candidateID, InstrumentID: identity.InstrumentID, IssuerID: identity.IssuerID,
		IdentitySource: p.IdentitySource, IdentityPolicyVersion: p.IdentityPolicy,
		RiskAllocation: p.RiskAllocation, RequestedLeverage: p.RequestedLeverage,
		SlippageAllowance: p.SlippageAllowance,
		SizingPolicyID:    p.SizingPolicyID, SizingPolicyVersion: p.SizingPolicyVersion,
	}, now)
}
