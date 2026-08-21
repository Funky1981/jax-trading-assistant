package canonical

import (
	"bytes"
	"errors"
	"math"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestCanonicalContractsValidateAndRoundTrip(t *testing.T) {
	fixtures := validContracts()
	tests := []struct {
		name        string
		value       Contract
		destination func() Contract
	}{
		{"issuer", fixtures.issuer, func() Contract { return &Issuer{} }},
		{"instrument", fixtures.instrument, func() Contract { return &Instrument{} }},
		{"event", fixtures.event, func() Contract { return &Event{} }},
		{"evidence", fixtures.evidence, func() Contract { return &Evidence{} }},
		{"observation", fixtures.observation, func() Contract { return &Observation{} }},
		{"research_run", fixtures.run, func() Contract { return &ResearchRun{} }},
		{"quant_result", fixtures.quant, func() Contract { return &QuantResult{} }},
		{"recommendation", fixtures.recommendation, func() Contract { return &Recommendation{} }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.value.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			first, err := EncodeJSON(test.value)
			if err != nil {
				t.Fatalf("EncodeJSON() error = %v", err)
			}
			second, err := EncodeJSON(test.value)
			if err != nil {
				t.Fatalf("second EncodeJSON() error = %v", err)
			}
			if string(first) != string(second) {
				t.Fatalf("encoding is not deterministic:\n%s\n%s", first, second)
			}

			destination := test.destination()
			if err := DecodeJSON(first, destination); err != nil {
				t.Fatalf("DecodeJSON() error = %v", err)
			}
			decoded := reflect.ValueOf(destination).Elem().Interface()
			if !reflect.DeepEqual(test.value, decoded) {
				t.Fatalf("round trip mismatch:\nwant: %#v\n got: %#v", test.value, decoded)
			}
		})
	}
}

func TestCanonicalContractsRejectUnsupportedVersions(t *testing.T) {
	fixtures := validContracts()
	tests := []struct {
		name     string
		contract Contract
	}{
		{"issuer", withIssuerVersion(fixtures.issuer, "jax.issuer/v2")},
		{"instrument", withInstrumentVersion(fixtures.instrument, "jax.instrument/v2")},
		{"event", withEventVersion(fixtures.event, "jax.event/v2")},
		{"evidence", withEvidenceVersion(fixtures.evidence, "jax.evidence/v2")},
		{"observation", withObservationVersion(fixtures.observation, "jax.observation/v2")},
		{"research_run", withResearchRunVersion(fixtures.run, "jax.research_run/v2")},
		{"quant_result", withQuantResultVersion(fixtures.quant, "jax.quant_result/v2")},
		{"recommendation", withRecommendationVersion(fixtures.recommendation, "jax.recommendation/v2")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertValidationField(t, test.contract.Validate(), "contract_version")
			if _, err := EncodeJSON(test.contract); err == nil {
				t.Fatal("EncodeJSON() accepted an unsupported version")
			}
		})
	}
}

func TestIssuerValidationRejectsSelfParent(t *testing.T) {
	issuer := validContracts().issuer
	issuer.ParentIssuerID = &issuer.ID
	assertValidationField(t, issuer.Validate(), "parent_issuer_id")
}

func TestInstrumentValidationRequiresBoundaryIdentity(t *testing.T) {
	instrument := validContracts().instrument
	instrument.ExternalIDs = nil
	assertValidationField(t, instrument.Validate(), "external_ids")
}

func TestEventValidationRequiresDomainTime(t *testing.T) {
	event := validContracts().event
	event.OccurredAt = nil
	event.EffectiveAt = nil
	assertValidationField(t, event.Validate(), "occurred_at")
}

func TestEvidenceValidationRejectsCollectionBeforePublication(t *testing.T) {
	evidence := validContracts().evidence
	tooEarly := evidence.PublishedAt.Add(-time.Second)
	evidence.CollectedAt = tooEarly
	assertValidationField(t, evidence.Validate(), "collected_at")
}

func TestObservationValidationRejectsAmbiguousValue(t *testing.T) {
	observation := validContracts().observation
	text := "also set"
	observation.Value.Text = &text
	assertValidationField(t, observation.Validate(), "value")
}

func TestResearchRunValidationRejectsImpossibleSucceededState(t *testing.T) {
	run := validContracts().run
	run.CompletedAt = nil
	assertValidationField(t, run.Validate(), "status")
}

func TestQuantResultValidationRejectsNonFiniteValue(t *testing.T) {
	result := validContracts().quant
	result.Values[0].Value = math.NaN()
	assertValidationField(t, result.Validate(), "values[0].value")
}

func TestRecommendationValidationRejectsExecutionAuthority(t *testing.T) {
	recommendation := validContracts().recommendation
	recommendation.ExecutionAuthority = "ORDER"
	assertValidationField(t, recommendation.Validate(), "execution_authority")
}

func TestDecodeJSONRejectsUnknownFieldsAndTrailingValues(t *testing.T) {
	unknown := []byte(`{"contract_version":"jax.event/v1","id":"evt_x","type":"news","assertion":"asserted","title":"x","occurred_at":"2026-08-21T10:00:00Z","created_at":"2026-08-21T10:00:00Z","unexpected":true}`)
	if err := DecodeJSON(unknown, &Event{}); err == nil {
		t.Fatal("DecodeJSON() accepted an unknown field")
	}

	valid, err := EncodeJSON(validContracts().event)
	if err != nil {
		t.Fatalf("EncodeJSON() error = %v", err)
	}
	valid = append(valid, []byte(` {}`)...)
	if err := DecodeJSON(valid, &Event{}); err == nil {
		t.Fatal("DecodeJSON() accepted a trailing JSON value")
	}
}

func TestRecommendationJSONFixture(t *testing.T) {
	encoded, err := EncodeJSON(validContracts().recommendation)
	if err != nil {
		t.Fatalf("EncodeJSON() error = %v", err)
	}
	want, err := os.ReadFile("testdata/recommendation_v1.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if !bytes.Equal(encoded, bytes.TrimSpace(want)) {
		t.Fatalf("fixture mismatch:\nwant: %s\n got: %s", bytes.TrimSpace(want), encoded)
	}
	var decoded Recommendation
	if err := DecodeJSON(want, &decoded); err != nil {
		t.Fatalf("DecodeJSON(fixture) error = %v", err)
	}
}

func TestValidationRequiresUTC(t *testing.T) {
	event := validContracts().event
	nonUTC := time.Date(2026, 8, 21, 11, 0, 0, 0, time.FixedZone("BST", 3600))
	event.CreatedAt = nonUTC
	assertValidationField(t, event.Validate(), "created_at")
}

func assertValidationField(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Validate() accepted invalid %s", field)
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error type = %T, want *ValidationError (%v)", err, err)
	}
	if validationErr.Field != field {
		t.Fatalf("validation field = %q, want %q (%v)", validationErr.Field, field, err)
	}
}

type canonicalFixtures struct {
	issuer         Issuer
	instrument     Instrument
	event          Event
	evidence       Evidence
	observation    Observation
	run            ResearchRun
	quant          QuantResult
	recommendation Recommendation
}

func validContracts() canonicalFixtures {
	base := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	published := base.Add(time.Minute)
	collected := base.Add(2 * time.Minute)
	created := base.Add(3 * time.Minute)
	started := base.Add(4 * time.Minute)
	completed := base.Add(5 * time.Minute)
	validUntil := base.Add(24 * time.Hour)
	confidence := 0.82
	price := 226.40
	parentIssuer := IssuerID("iss_parent")

	issuer := Issuer{
		ContractVersion: IssuerContractV1,
		ID:              "iss_apple",
		Type:            IssuerTypeCorporate,
		Name:            "Apple Inc.",
		Jurisdiction:    "US",
		ExternalIDs:     []ExternalID{{Namespace: "lei", Value: "HWUPKR0MPOU8FGXBT394"}},
		ParentIssuerID:  &parentIssuer,
		Effective:       EffectivePeriod{From: &base},
		CreatedAt:       created,
	}
	instrument := Instrument{
		ContractVersion: InstrumentContractV1,
		ID:              "ins_aapl_common",
		Type:            InstrumentTypeEquity,
		Name:            "Apple Inc. common stock",
		Currency:        "USD",
		ExternalIDs: []ExternalID{
			{Namespace: "ticker.xnas", Value: "AAPL"},
			{Namespace: "isin", Value: "US0378331005"},
		},
		Issuers:   []InstrumentIssuer{{IssuerID: issuer.ID, Role: InstrumentIssuerRoleIssuer}},
		Effective: EffectivePeriod{From: &base},
		CreatedAt: created,
	}
	instrumentRef := ContractRef{Kind: ContractKindInstrument, ID: string(instrument.ID), ContractVersion: InstrumentContractV1}
	issuerRef := ContractRef{Kind: ContractKindIssuer, ID: string(issuer.ID), ContractVersion: IssuerContractV1}
	event := Event{
		ContractVersion: EventContractV1,
		ID:              "evt_apple_filing",
		Type:            EventTypeRegulatory,
		Assertion:       EventAssertionConfirmed,
		Title:           "Apple files quarterly report",
		Summary:         "A confirmed regulatory filing became available.",
		Subjects:        []ContractRef{instrumentRef, issuerRef},
		ExternalIDs:     []ExternalID{{Namespace: "sec.accession", Value: "0000320193-26-000001"}},
		OccurredAt:      &published,
		CreatedAt:       created,
	}
	eventRef := ContractRef{Kind: ContractKindEvent, ID: string(event.ID), ContractVersion: EventContractV1}
	evidence := Evidence{
		ContractVersion: EvidenceContractV1,
		ID:              "evd_apple_10q",
		Type:            EvidenceTypeFiling,
		Title:           "Apple quarterly filing",
		Source: SourceReference{
			ID:         "src_sec_edgar",
			Kind:       SourceKindRegulator,
			ExternalID: &ExternalID{Namespace: "sec.accession", Value: "0000320193-26-000001"},
			URI:        "https://www.sec.gov/Archives/example",
		},
		Links:       []EvidenceLink{{Target: eventRef, Relationship: EvidenceRelationshipSupports}},
		PublishedAt: &published,
		CollectedAt: collected,
		CreatedAt:   created,
	}
	observation := Observation{
		ContractVersion: ObservationContractV1,
		ID:              "obs_aapl_close",
		Type:            ObservationTypePrice,
		Subject:         instrumentRef,
		Metric:          "close_price",
		Value:           ObservedValue{Type: ObservedValueTypeNumber, Number: &price, Unit: "USD"},
		Source: SourceReference{
			ID:         "src_market_data",
			Kind:       SourceKindProvider,
			ExternalID: &ExternalID{Namespace: "provider.symbol", Value: "AAPL"},
		},
		EvidenceIDs: []EvidenceID{evidence.ID},
		ObservedAt:  base,
		PublishedAt: &published,
		CollectedAt: collected,
		CreatedAt:   created,
	}
	observationRef := ContractRef{Kind: ContractKindObservation, ID: string(observation.ID), ContractVersion: ObservationContractV1}
	quantRef := ContractRef{Kind: ContractKindQuantResult, ID: "qnt_aapl_event_return", ContractVersion: QuantResultContractV1}
	recommendationRef := ContractRef{Kind: ContractKindRecommendation, ID: "rec_aapl_watch", ContractVersion: RecommendationContractV1}
	run := ResearchRun{
		ContractVersion: ResearchRunContractV1,
		ID:              "run_aapl_event_study",
		Type:            ResearchRunTypeAnalysis,
		Status:          ResearchRunStatusSucceeded,
		Method:          ComponentRef{Kind: ComponentKindMethod, Name: "event study", Version: "1.0.0"},
		InputRefs:       []ContractRef{eventRef, observationRef},
		OutputRefs:      []ContractRef{quantRef, recommendationRef},
		CreatedAt:       created,
		StartedAt:       &started,
		CompletedAt:     &completed,
	}
	quant := QuantResult{
		ContractVersion: QuantResultContractV1,
		ID:              QuantResultID(quantRef.ID),
		Type:            QuantResultTypeScalar,
		ResearchRunID:   run.ID,
		Method:          ComponentRef{Kind: ComponentKindAlgorithm, Name: "event return", Version: "1.0.0"},
		InputRefs:       []ContractRef{observationRef},
		Values: []QuantValue{{
			Metric: "event_return",
			Value:  0.031,
			Unit:   "ratio",
			Dimensions: []QuantDimension{
				{Name: "window", Value: "1d"},
			},
		}},
		CalculatedAt: completed,
	}
	evidenceRef := ContractRef{Kind: ContractKindEvidence, ID: string(evidence.ID), ContractVersion: EvidenceContractV1}
	recommendation := Recommendation{
		ContractVersion:    RecommendationContractV1,
		ID:                 RecommendationID(recommendationRef.ID),
		Disposition:        RecommendationDispositionWatch,
		ResearchRunID:      run.ID,
		Subjects:           []ContractRef{instrumentRef, eventRef},
		Basis:              []ContractRef{eventRef, evidenceRef, observationRef, quantRef},
		Reasons:            []RecommendationReason{{Code: "evidence_requires_review", Summary: "The filing and measured reaction support continued research."}},
		Confidence:         &confidence,
		Authority:          RecommendationAuthorityResearchDecisionSupport,
		ExecutionAuthority: ExecutionAuthorityNone,
		CreatedAt:          completed,
		ValidUntil:         &validUntil,
	}
	return canonicalFixtures{issuer, instrument, event, evidence, observation, run, quant, recommendation}
}

func withIssuerVersion(value Issuer, version ContractVersion) Issuer {
	value.ContractVersion = version
	return value
}

func withInstrumentVersion(value Instrument, version ContractVersion) Instrument {
	value.ContractVersion = version
	return value
}

func withEventVersion(value Event, version ContractVersion) Event {
	value.ContractVersion = version
	return value
}

func withEvidenceVersion(value Evidence, version ContractVersion) Evidence {
	value.ContractVersion = version
	return value
}

func withObservationVersion(value Observation, version ContractVersion) Observation {
	value.ContractVersion = version
	return value
}

func withResearchRunVersion(value ResearchRun, version ContractVersion) ResearchRun {
	value.ContractVersion = version
	return value
}

func withQuantResultVersion(value QuantResult, version ContractVersion) QuantResult {
	value.ContractVersion = version
	return value
}

func withRecommendationVersion(value Recommendation, version ContractVersion) Recommendation {
	value.ContractVersion = version
	return value
}
