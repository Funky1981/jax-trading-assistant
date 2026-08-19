package aishadow

import "fmt"

const C1FScoringVersion = "ai-shadow-causal-attribution-scoring-c1f3-v1"

func ValidateC1FContractRoute(prompt, output, validator, policy, identity, scoring string) error {
	want := map[string]string{
		"prompt": V6PromptVersion, "output": V5SchemaVersion, "validator": C1FValidatorVersion,
		"policy": CausalAttributionPolicyVersion, "identity": IssuerSemanticIdentityVersion, "scoring": C1FScoringVersion,
	}
	got := map[string]string{"prompt": prompt, "output": output, "validator": validator, "policy": policy, "identity": identity, "scoring": scoring}
	for key, expected := range want {
		if got[key] != expected {
			return fmt.Errorf("C1F route requires %s %q", key, expected)
		}
	}
	return nil
}
