package sec

import (
	"context"
	"os"
	"testing"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

// This test is opt-in and uses the accepted SEC provider path. It fails when
// the operator has not supplied the required automated-client identity so a
// fixture cannot accidentally look like a real company-evidence pass.
func TestPhase03ExitGateLiveAAPLCompanyEvidence(t *testing.T) {
	if os.Getenv("JAX_RUN_LIVE_PHASE03_GATE") != "1" {
		t.Skip("set JAX_RUN_LIVE_PHASE03_GATE=1 to run the bounded live Phase-03 gate check")
	}
	config, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("accepted SEC live configuration unavailable: %v", err)
	}
	provider, err := NewProvider(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	issuer := testIssuer()
	resolver := StaticIdentityResolver{testCIK: issuer}
	registry, pipeline, err := NewProviderPipeline(resolver)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := providercontract.NewOperationalExecutor(registry, testPolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.AcquireSubmissions(context.Background(), Dependencies{Registry: registry, Executor: executor, Store: providercontract.NewMemoryRawPayloadStore(), Pipeline: pipeline, Resolver: resolver}, SubmissionsRequest{Identity: CIKIdentity{Issuer: canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(issuer.ID), ContractVersion: issuer.ContractVersion}, CIK: testCIK}, PayloadID: "rpa_phase03_gate_sec_live", Retention: testRetention()})
	if err != nil {
		t.Fatalf("accepted SEC live path failed: %v; execution=%+v", err, result.Execution)
	}
	if len(result.Filings) == 0 || len(result.RawPayloads) == 0 {
		t.Fatal("live SEC result did not contain normalized filing evidence and raw provenance")
	}
	t.Logf("LIVE_ACQUIRED_NOW provider=%s source=%s raw_payload=%s digest=%s normalized_filing=%s filing_date=%s acceptance=%v acquired_at=%s", ProviderID, SubmissionsSourceID, result.RawPayloads[0].Ref.ID, result.RawPayloads[0].Ref.Content.Digest.Value, result.Filings[0].Evidence.ID, result.Filings[0].Filing.Dates.FilingDate, result.Filings[0].Filing.Dates.AcceptanceDateTime, result.RawPayloads[0].Ref.ReceivedAt)
}
