package orchestration

import (
	"testing"

	"jax-trading-assistant/internal/modules/planner"
)

func lookupValues(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func TestNewConfiguredPlannerDefaultsToDeterministic(t *testing.T) {
	service, err := NewConfiguredPlanner(lookupValues(nil))
	if err != nil {
		t.Fatal(err)
	}
	identity := service.Identity()
	if identity.Provider != "jax-planner" || identity.Model != planner.ContractVersion {
		t.Fatalf("unexpected default identity: %+v", identity)
	}
}

func TestNewConfiguredPlannerSelectsExistingLiteLLMTransport(t *testing.T) {
	service, err := NewConfiguredPlanner(lookupValues(map[string]string{
		"JAX_PLANNER_PROVIDER": "litellm",
		"AI_GATEWAY_BASE_URL":  "http://gateway.test",
		"AI_GATEWAY_API_KEY":   "virtual-test-key",
		"AI_DEFAULT_MODEL":     "local-small",
	}))
	if err != nil {
		t.Fatal(err)
	}
	identity := service.Identity()
	if identity.Provider != "litellm" || identity.Model != "local-small" {
		t.Fatalf("unexpected configured identity: %+v", identity)
	}
}

func TestNewConfiguredPlannerRejectsUnknownProvider(t *testing.T) {
	if _, err := NewConfiguredPlanner(lookupValues(map[string]string{"JAX_PLANNER_PROVIDER": "unknown"})); err == nil {
		t.Fatal("unknown planner provider was accepted")
	}
}

func TestNewConfiguredPlannerRejectsIncompleteModelConfiguration(t *testing.T) {
	if _, err := NewConfiguredPlanner(lookupValues(map[string]string{"JAX_PLANNER_PROVIDER": "litellm"})); err == nil {
		t.Fatal("incomplete model configuration was accepted")
	}
}
