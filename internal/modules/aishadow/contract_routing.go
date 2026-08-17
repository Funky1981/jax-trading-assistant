package aishadow

import (
	"fmt"
	"reflect"
)

// ValidateContractRoute rejects mixed or unknown prompt/contract/policy cells
// before provider construction. Historical v4 and typed v5 routes are the
// only executable current-generation combinations.
func ValidateContractRoute(promptVersion, outputContract, causalPolicy string) error {
	switch outputContract {
	case SchemaVersion:
		if promptVersion != PromptVersion {
			return fmt.Errorf("v4 output contract requires prompt %q", PromptVersion)
		}
		if causalPolicy != CausalConsistencyPolicyVersion {
			return fmt.Errorf("v4 output contract requires historical policy %q", CausalConsistencyPolicyVersion)
		}
		return nil
	case V5SchemaVersion:
		if promptVersion != V5PromptVersion {
			return fmt.Errorf("v5 output contract requires prompt %q", V5PromptVersion)
		}
		if causalPolicy != CausalAttributionPolicyVersion {
			return fmt.Errorf("v5 output contract requires typed policy %q", CausalAttributionPolicyVersion)
		}
		return nil
	default:
		return fmt.Errorf("unsupported AI shadow output contract %q", outputContract)
	}
}

func ValidateV5ProviderRequestSchema(request ProviderRequest) error {
	if request.SchemaContract != V5SchemaVersion {
		return fmt.Errorf("v5 provider request requires schema contract %q", V5SchemaVersion)
	}
	properties, ok := request.Schema["properties"].(map[string]any)
	if !ok {
		return fmt.Errorf("v5 provider request schema has no properties")
	}
	candidates, ok := properties["principal_proxy_candidates"].(map[string]any)
	if !ok {
		return fmt.Errorf("v5 provider request schema has no principal_proxy_candidates")
	}
	items, ok := candidates["items"].(map[string]any)
	if !ok {
		return fmt.Errorf("v5 provider request proxy candidates have no items schema")
	}
	proxyExposures, ok := items["enum"].([]string)
	if !ok {
		return fmt.Errorf("v5 provider request proxy candidates have no bounded enum")
	}
	want := V5OutputSchema(proxyExposures)
	if !reflect.DeepEqual(request.Schema, want) {
		return fmt.Errorf("v5 provider request schema does not equal the canonical contract")
	}
	hash, err := fingerprint(request.Schema)
	if err != nil {
		return err
	}
	if request.SchemaSHA256 == "" || request.SchemaSHA256 != hash {
		return fmt.Errorf("v5 provider request schema hash mismatch")
	}
	return nil
}
