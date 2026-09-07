package portfoliorisk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

const RiskPolicyContractVersion = "jax.portfolio.risk_policy/v1"

// RiskPolicy is an explicit, immutable policy input. A nil optional limit
// means that the policy does not define that dimension; decision code must not
// invent a value. MaximumLeverage is mandatory and cannot exceed the Phase-09
// safety ceiling of 1x.
type RiskPolicy struct {
	PolicyID              string   `json:"policy_id"`
	ContractVersion       string   `json:"contract_version"`
	Version               string   `json:"version"`
	Currency              string   `json:"currency"`
	MaximumPositionValue  *float64 `json:"maximum_position_value,omitempty"`
	MaximumConcentration  *float64 `json:"maximum_concentration,omitempty"`
	MaximumGrossExposure  *float64 `json:"maximum_gross_exposure,omitempty"`
	MaximumNetExposure    *float64 `json:"maximum_net_exposure,omitempty"`
	MaximumRiskAllocation *float64 `json:"maximum_risk_allocation,omitempty"`
	MinimumCash           *float64 `json:"minimum_cash,omitempty"`
	MaximumLeverage       *float64 `json:"maximum_leverage"`
	MaximumStressLoss     *float64 `json:"maximum_stress_loss,omitempty"`
}

func Limit(value float64) *float64 { return &value }

func BuildRiskPolicy(input RiskPolicy) (RiskPolicy, error) {
	claimed := strings.TrimSpace(input.PolicyID)
	input.PolicyID = ""
	if input.ContractVersion == "" {
		input.ContractVersion = RiskPolicyContractVersion
	}
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Version = strings.TrimSpace(input.Version)
	if err := input.Validate(); err != nil {
		return RiskPolicy{}, err
	}
	input.PolicyID = policyIdentity(input)
	if claimed != "" && claimed != input.PolicyID {
		return RiskPolicy{}, fmt.Errorf("%w: claimed %q calculated %q", ErrPolicyIdentity, claimed, input.PolicyID)
	}
	return input, nil
}

var ErrPolicyIdentity = fmt.Errorf("risk policy identity does not match content")

func (p RiskPolicy) Validate() error {
	if p.ContractVersion != "" && p.ContractVersion != RiskPolicyContractVersion {
		return fmt.Errorf("invalid risk policy: unsupported contract version %q", p.ContractVersion)
	}
	if strings.TrimSpace(p.Version) == "" {
		return fmt.Errorf("invalid risk policy: version is required")
	}
	if !SupportedCurrency(p.Currency) {
		return fmt.Errorf("invalid risk policy: unsupported currency %q", p.Currency)
	}
	if p.MaximumLeverage == nil || !finiteLimit(p.MaximumLeverage) || *p.MaximumLeverage <= 0 || *p.MaximumLeverage > 1 {
		return fmt.Errorf("invalid risk policy: maximum leverage must be in (0,1]")
	}
	limits := []struct {
		name    string
		value   *float64
		maximum float64
	}{
		{"maximum position value", p.MaximumPositionValue, math.Inf(1)},
		{"maximum concentration", p.MaximumConcentration, 1},
		{"maximum gross exposure", p.MaximumGrossExposure, math.Inf(1)},
		{"maximum net exposure", p.MaximumNetExposure, math.Inf(1)},
		{"maximum risk allocation", p.MaximumRiskAllocation, 1},
		{"minimum cash", p.MinimumCash, math.Inf(1)},
		{"maximum stress loss", p.MaximumStressLoss, 1},
	}
	for _, limit := range limits {
		if limit.value != nil && (!finiteLimit(limit.value) || *limit.value < 0 || *limit.value > limit.maximum) {
			return fmt.Errorf("invalid risk policy: %s is outside its permitted range", limit.name)
		}
	}
	return nil
}

func finiteLimit(value *float64) bool { return value != nil && finite(*value) }

func policyIdentity(policy RiskPolicy) string {
	b, _ := json.Marshal(policy)
	digest := sha256.Sum256(b)
	return "rpol_" + hex.EncodeToString(digest[:])
}
