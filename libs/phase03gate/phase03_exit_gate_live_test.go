package phase03gate

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/marketdata"
	"jax-trading-assistant/libs/sec"
	"jax-trading-assistant/libs/treasury"
)

const phase03GateCIK = "0000320193"

func TestPhase03ExitGateLiveAAPLPacket(t *testing.T) {
	if os.Getenv("JAX_RUN_LIVE_PHASE03_GATE") != "1" {
		t.Skip("set JAX_RUN_LIVE_PHASE03_GATE=1 to run the bounded live Phase-03 gate")
	}
	if strings.TrimSpace(os.Getenv("ALPACA_API_KEY")) == "" || strings.TrimSpace(os.Getenv("ALPACA_API_SECRET")) == "" {
		t.Skip("ALPACA_API_KEY and ALPACA_API_SECRET are not configured")
	}

	ctx := context.Background()
	instrument := phase03GateInstrument()
	issuer := phase03GateIssuer()
	store := providercontract.NewMemoryRawPayloadStore()

	marketResult := acquirePhase03Market(t, ctx, store, instrument)
	companyResult := acquirePhase03Company(t, ctx, store, issuer)
	macroResult := acquirePhase03Macro(t, ctx, store)

	marketItem := phase03MarketItem(t, marketResult)
	companyItem := phase03CompanyItem(t, companyResult)
	macroItem := phase03MacroItem(t, macroResult)
	packet, err := NewPacket(instrument, issuer, []EvidenceItem{marketItem, companyItem, macroItem}, []string{"eqc_vix_fred_fixture_v1"}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := packet.AssertRealExitCondition(); err != nil {
		t.Fatalf("real Phase-03 exit assertion failed: %v", err)
	}
	lastBar := marketResult.Bars[len(marketResult.Bars)-1]
	t.Logf("PHASE 03 EXIT CONDITION DEMONSTRATED packet=%s market_raw=%s market_digest=%s market_observation=%s market_date=%s market_open=%v market_high=%v market_low=%v market_close=%v market_volume=%v company_raw=%s company_digest=%s company_evidence=%s macro_raw=%s macro_digest=%s macro_evidence=%s", packet.ID, marketItem.RawPayload.ID, marketItem.RawPayload.Content.Digest.Value, marketItem.NormalizedRef.ID, lastBar.ProviderDate, *lastBar.Open.Value.Number, *lastBar.High.Value.Number, *lastBar.Low.Value.Number, *lastBar.Close.Value.Number, *lastBar.Volume.Value.Number, companyItem.RawPayload.ID, companyItem.RawPayload.Content.Digest.Value, companyItem.NormalizedRef.ID, macroItem.RawPayload.ID, macroItem.RawPayload.Content.Digest.Value, macroItem.NormalizedRef.ID)
}

func phase03GateInstrument() canonical.Instrument {
	return canonical.Instrument{ContractVersion: canonical.InstrumentContractV1, ID: "ins_aapl_common", Type: canonical.InstrumentTypeEquity, Name: "Apple Inc. common stock", Currency: "USD", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ExternalIDs: []canonical.ExternalID{{Namespace: "ticker.us", Value: "AAPL"}}, Issuers: []canonical.InstrumentIssuer{{IssuerID: "iss_apple", Role: canonical.InstrumentIssuerRoleIssuer}}}
}

func phase03GateIssuer() canonical.Issuer {
	return canonical.Issuer{ContractVersion: canonical.IssuerContractV1, ID: "iss_apple", Type: canonical.IssuerTypeCorporate, Name: "Apple Inc.", Jurisdiction: "US", ExternalIDs: []canonical.ExternalID{{Namespace: "sec.cik", Value: phase03GateCIK}}, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func phase03GateRetention() providercontract.RawPayloadRetentionPolicy {
	return providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionRestricted}
}

func phase03GatePolicy() providercontract.OperationalPolicy {
	identity := func(id, namespace string, kind canonical.ComponentKind) canonical.ComponentIdentity {
		return canonical.ComponentIdentity{ID: id, Kind: kind, Name: id, Version: canonical.VersionIdentity{Namespace: namespace, Value: "phase03-gate/v1"}}
	}
	return providercontract.OperationalPolicy{
		ContractVersion: providercontract.OperationalPolicyContractV1,
		Classification:  providercontract.ClassificationPolicy{ContractVersion: providercontract.ClassificationPolicyContractV1, Identity: identity("cmp_phase03_classification", "jax.policy.failure_classification", canonical.ComponentKindPolicy)},
		Retry:           providercontract.RetryPolicy{ContractVersion: providercontract.RetryPolicyContractV1, Identity: identity("cmp_phase03_retry", "jax.policy.retry", canonical.ComponentKindPolicy), MaximumAttempts: 2, MaximumElapsed: 2 * time.Minute, PerAttemptTimeout: 60 * time.Second, RetryableFailures: []providercontract.FailureClass{providercontract.FailureTransportTransient, providercontract.FailureProviderServer, providercontract.FailureTemporaryUnavailable, providercontract.FailureRateLimited, providercontract.FailureAttemptDeadline}, Backoff: providercontract.BackoffPolicy{InitialDelay: time.Millisecond, Multiplier: 2, MaximumDelay: time.Second, Jitter: providercontract.JitterNone}},
		RateLimit:       providercontract.RateLimitPolicy{ContractVersion: providercontract.RateLimitPolicyContractV1, Identity: identity("cmp_phase03_rate_limit", "jax.policy.rate_limit", canonical.ComponentKindPolicy), RequestLimit: 100, Window: time.Minute, ConcurrencyLimit: 2, MaximumProviderDelay: time.Second},
		Health:          providercontract.HealthPolicy{ContractVersion: providercontract.HealthPolicyContractV1, Identity: identity("cmp_phase03_health", "jax.policy.health", canonical.ComponentKindPolicy), DegradedAfterFailures: 1, UnavailableAfterFailures: 3, RecoverySuccesses: 1, AssessmentHorizon: time.Hour},
		Component:       identity("cmp_phase03_operational", "git.commit", canonical.ComponentKindSoftwareBuild),
	}
}

func acquirePhase03Market(t *testing.T, ctx context.Context, store providercontract.RawPayloadStore, instrument canonical.Instrument) marketdata.AlpacaBarsResult {
	t.Helper()
	feed := marketdata.MarketFeed(strings.ToLower(strings.TrimSpace(os.Getenv("ALPACA_DATA_FEED"))))
	if feed == "" {
		feed = marketdata.MarketFeedSIP
	}
	provider, err := marketdata.NewAlpacaProvider(marketdata.ProviderConfig{Name: marketdata.ProviderAlpaca, APIKey: os.Getenv("ALPACA_API_KEY"), APISecret: os.Getenv("ALPACA_API_SECRET"), Tier: "free", Feed: feed.String(), Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(marketdata.AlpacaProviderDefinition()); err != nil {
		t.Fatal(err)
	}
	normalizers, err := providercontract.NewNormalizerRegistry(registry)
	if err != nil {
		t.Fatal(err)
	}
	normalizer, err := marketdata.NewAlpacaBarsNormalizerForGate(instrument, feed)
	if err != nil {
		t.Fatal(err)
	}
	if err := normalizers.Register(normalizer); err != nil {
		t.Fatal(err)
	}
	pipeline, err := providercontract.NewNormalizationPipeline(registry, normalizers)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := providercontract.NewOperationalExecutor(registry, phase03GatePolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	end := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -2)
	result, err := provider.AcquireAndNormalizeDailyBars(ctx, marketdata.AlpacaBarsDependencies{Registry: registry, Executor: executor, Store: store, Pipeline: pipeline}, marketdata.AlpacaBarsRequest{Instrument: instrument, StartDate: end.AddDate(0, 0, -7), EndDate: end, Interval: marketdata.Timeframe1Day, Feed: feed, Adjustment: marketdata.MarketAdjustmentUnadjusted, PayloadID: "rpa_phase03_gate_market_alpaca_live", Retention: phase03GateRetention()})
	if err != nil {
		t.Fatalf("Alpaca real market acquisition failed: %v", err)
	}
	return result
}

func acquirePhase03Company(t *testing.T, ctx context.Context, store providercontract.RawPayloadStore, issuer canonical.Issuer) sec.SubmissionsResult {
	t.Helper()
	config, err := sec.LoadConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := sec.NewProvider(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolver := sec.StaticIdentityResolver{phase03GateCIK: issuer}
	registry, pipeline, err := sec.NewProviderPipeline(resolver)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := providercontract.NewOperationalExecutor(registry, phase03GatePolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.AcquireSubmissions(ctx, sec.Dependencies{Registry: registry, Executor: executor, Store: store, Pipeline: pipeline, Resolver: resolver}, sec.SubmissionsRequest{Identity: sec.CIKIdentity{Issuer: canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(issuer.ID), ContractVersion: issuer.ContractVersion}, CIK: phase03GateCIK}, PayloadID: "rpa_phase03_gate_sec_live", Retention: phase03GateRetention()})
	if err != nil {
		t.Fatalf("SEC real company acquisition failed: %v", err)
	}
	return result
}

func acquirePhase03Macro(t *testing.T, ctx context.Context, store providercontract.RawPayloadStore) treasury.YearResult {
	t.Helper()
	provider, err := treasury.NewProvider(treasury.DefaultConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := treasury.RegisterProvider(registry); err != nil {
		t.Fatal(err)
	}
	executor, err := providercontract.NewOperationalExecutor(registry, phase03GatePolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.AcquireYear(ctx, treasury.Dependencies{Registry: registry, Executor: executor, Store: store}, treasury.YearRequest{Year: time.Now().UTC().Year(), PayloadID: "rpa_phase03_gate_treasury_live", Retention: phase03GateRetention()})
	if err != nil {
		t.Fatalf("Treasury real macro acquisition failed: %v", err)
	}
	return result
}

func phase03MarketItem(t *testing.T, result marketdata.AlpacaBarsResult) EvidenceItem {
	t.Helper()
	bar := result.Bars[len(result.Bars)-1]
	for _, accepted := range result.Normalization.Records {
		observation, ok := accepted.Record.(canonical.Observation)
		if ok && observation.ID != "" && observation.Metric == marketdata.MarketMetricClose && observation.ObservedAt.Equal(bar.Close.ObservedAt) {
			return EvidenceItem{Family: FamilyMarket, Origin: OriginLiveAcquiredNow, NormalizedRef: canonical.ContractRef{Kind: canonical.ContractKindObservation, ID: string(observation.ID), ContractVersion: observation.ContractVersion}, RawPayload: accepted.RawRef, Provenance: *observation.Provenance, Temporal: TemporalMetadata{ObservationDate: bar.ProviderDate, AcquiredAt: accepted.RawRef.ReceivedAt}}
		}
	}
	t.Fatal("Alpaca result did not contain a close observation for the representative bar")
	return EvidenceItem{}
}

func phase03CompanyItem(t *testing.T, result sec.SubmissionsResult) EvidenceItem {
	t.Helper()
	if len(result.Filings) == 0 {
		t.Fatal("SEC result did not contain a filing")
	}
	filing := result.Filings[0]
	if filing.Evidence.ImmutableRef == nil {
		t.Fatal("SEC filing evidence lacks its immutable raw reference")
	}
	filingDate := string(filing.Filing.Dates.FilingDate)
	item := EvidenceItem{Family: FamilyCompany, Origin: OriginLiveAcquiredNow, NormalizedRef: canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(filing.Evidence.ID), ContractVersion: filing.Evidence.ContractVersion}, RawPayload: filing.Filing.SourcePayload, Provenance: phase03RawProvenance(t, filing.Filing.SourcePayload, "sec-filing"), Temporal: TemporalMetadata{FilingDate: filingDate, AcquiredAt: filing.Filing.Dates.AcquiredAt, AcceptanceDateTime: filing.Filing.Dates.AcceptanceDateTime}}
	if filing.Filing.Dates.ReportDate != nil {
		item.Temporal.ReportDate = string(*filing.Filing.Dates.ReportDate)
	}
	return item
}

func phase03MacroItem(t *testing.T, result treasury.YearResult) EvidenceItem {
	t.Helper()
	for _, candidate := range result.Observations {
		if candidate.Tenor != "10 Yr" || !candidate.Observation.Value.Present {
			continue
		}
		if len(candidate.Observation.Provenance.Inputs) == 0 || candidate.Observation.Provenance.Inputs[0].Evidence == nil {
			t.Fatal("Treasury observation lacks canonical evidence lineage")
		}
		evidence := candidate.Observation.Provenance.Inputs[0].Evidence
		return EvidenceItem{Family: FamilyMacroContext, Origin: OriginLiveAcquiredNow, NormalizedRef: evidence.Evidence, RawPayload: candidate.Observation.SourcePayload, Provenance: candidate.Observation.Provenance, Temporal: TemporalMetadata{ObservationDate: string(candidate.Observation.ObservationDate), AcquiredAt: candidate.Observation.AcquiredAt}}
	}
	t.Fatal("Treasury result did not contain a present 10 Yr observation")
	return EvidenceItem{}
}

func phase03RawProvenance(t *testing.T, raw providercontract.RawPayloadRef, label string) canonical.Provenance {
	t.Helper()
	evidenceID := canonical.EvidenceID("evd_" + canonical.DigestBytes([]byte("phase03-gate\x00" + label + "\x00" + string(raw.ID))).Value[:24])
	evidenceRef, err := raw.AsEvidenceRef(canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(evidenceID), ContractVersion: canonical.EvidenceContractV2})
	if err != nil {
		t.Fatal(err)
	}
	inputs := []canonical.LineageInput{{Kind: canonical.LineageInputKindEvidence, Evidence: &evidenceRef}}
	fingerprint, err := canonical.ComputeInputFingerprint(inputs)
	if err != nil {
		t.Fatal(err)
	}
	provider := raw.Provider
	return canonical.Provenance{ContractVersion: canonical.ProvenanceContractV1, ID: "pvn_" + canonical.DigestBytes([]byte("phase03-provenance\x00" + label + "\x00" + string(raw.ID))).Value[:24], Inputs: inputs, InputFingerprint: fingerprint, Producer: canonical.ComponentIdentity{ID: "cmp_phase03_gate_builder", Kind: canonical.ComponentKindSoftwareBuild, Name: "Phase 03 gate packet builder", Version: canonical.VersionIdentity{Namespace: "jax.phase03_gate", Value: "v1"}, Provider: &provider}}
}
