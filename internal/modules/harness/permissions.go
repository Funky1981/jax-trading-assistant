package harness

import (
	"fmt"
	"strings"
)

const ResearchPermissionPolicyContractV1 = "jax.research_permission_policy/v1"

type ResearchPermissionPolicy struct {
	ContractVersion       string   `json:"contract_version"`
	Version               string   `json:"version"`
	AllowedTiers          []string `json:"allowed_tiers"`
	ResearchOnly          bool     `json:"research_only"`
	ExternalDataAllowed   bool     `json:"external_data_allowed"`
	ExecutionAuthority    string   `json:"execution_authority"`
	ForbiddenCapabilities []string `json:"forbidden_capabilities"`
}

func DefaultResearchPermissionPolicy() ResearchPermissionPolicy {
	return ResearchPermissionPolicy{ContractVersion: ResearchPermissionPolicyContractV1, Version: "v1", AllowedTiers: []string{ToolPermissionDerivedRead, ToolPermissionEvidenceRead, ToolPermissionMemoryRead}, ResearchOnly: true, ExternalDataAllowed: false, ExecutionAuthority: "NONE", ForbiddenCapabilities: []string{"approval", "arbitrary_filesystem", "arbitrary_http", "arbitrary_sql", "broker", "candidate_creation", "execution_instruction", "fill", "order", "portfolio_mutation", "process_execution", "safety_configuration", "trade"}}
}

func (policy ResearchPermissionPolicy) Validate() error {
	if policy.ContractVersion != ResearchPermissionPolicyContractV1 || strings.TrimSpace(policy.Version) == "" || !policy.ResearchOnly || policy.ExecutionAuthority != "NONE" || !schemaStringList(policy.AllowedTiers) || !schemaStringList(policy.ForbiddenCapabilities) {
		return fmt.Errorf("research permission policy must be versioned, research-only and execution-disabled")
	}
	for _, tier := range policy.AllowedTiers {
		if tier != ToolPermissionEvidenceRead && tier != ToolPermissionDerivedRead && tier != ToolPermissionMemoryRead {
			return fmt.Errorf("unsupported research permission tier %q", tier)
		}
	}
	return nil
}

func (policy ResearchPermissionPolicy) Allows(spec ControlledToolSpec) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	if !spec.ReadOnly || spec.PermissionTier == "" || !containsExact(policy.AllowedTiers, spec.PermissionTier) || spec.ExternalData && !policy.ExternalDataAllowed {
		return fmt.Errorf("permission denied for controlled tool %s", spec.ID)
	}
	return nil
}

func (policy ResearchPermissionPolicy) DeniesCapability(capability string) bool {
	for _, denied := range policy.ForbiddenCapabilities {
		if denied == capability {
			return true
		}
	}
	return false
}
