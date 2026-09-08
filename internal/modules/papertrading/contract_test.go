package papertrading

import (
	"errors"
	"testing"
)

func TestPaperCapabilityContractIsExplicitAndVersioned(t *testing.T) {
	contract := DefaultPaperCapabilityContract()
	if err := contract.Validate(); err != nil {
		t.Fatal(err)
	}
	if capabilityIdentity(contract) == "" || contract.Environment != EnvironmentPaper || contract.ExecutionAuthority != ExecutionAuthorityNone {
		t.Fatalf("incomplete paper contract: %#v", contract)
	}
	if !contract.Supports(OrderMarket) || !contract.Supports(OrderLimit) || contract.Supports(OrderType("STOP")) {
		t.Fatal("order capability declaration is not explicit")
	}
}

func TestPaperCapabilityContractRejectsUnknownOrLiveConfiguration(t *testing.T) {
	unknown := DefaultPaperCapabilityContract()
	unknown.Environment = EnvironmentUnknown
	if !errors.Is(unknown.Validate(), ErrInvalidEnvironment) {
		t.Fatal("unknown environment did not fail closed")
	}
	live := DefaultPaperCapabilityContract()
	live.Environment = EnvironmentLive
	if !errors.Is(live.Validate(), ErrLiveExecutionDisabled) {
		t.Fatal("live environment did not remain disabled")
	}
	unsupported := DefaultPaperCapabilityContract()
	unsupported.SupportsLimit = false
	if err := unsupported.Validate(); err != nil {
		t.Fatal(err)
	}
	if unsupported.Supports(OrderLimit) {
		t.Fatal("unsupported limit capability reported as supported")
	}
}
