package evidencequality

import (
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/macroevidence"
)

var qualityTestTime = time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)

func testRawRef(t *testing.T, id string, provider canonical.ProviderIdentity, source canonical.SourceIdentity, body []byte) providercontract.RawPayloadRef {
	t.Helper()
	ref := providercontract.RawPayloadRef{
		ContractVersion: providercontract.RawPayloadRefContractV1,
		ID:              providercontract.RawPayloadID(id),
		Content:         canonical.RawContentIdentity(body),
		Provider:        provider,
		CapabilityID:    providercontract.CapabilityMacroObservation,
		Raw: providercontract.RawRepresentation{
			Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatTabular,
			Schema: canonical.VersionIdentity{Namespace: "test.raw", Value: "v1"}, MediaType: "text/csv",
		},
		Capture:    providercontract.RawPayloadCapture{ByteForm: providercontract.RawPayloadByteFormEntityBody, ContentCodingState: providercontract.ContentCodingIdentity},
		Source:     &source,
		Revision:   &canonical.RevisionIdentity{Namespace: "test.revision", Value: id},
		ReceivedAt: qualityTestTime,
		SizeBytes:  int64(len(body)),
		Retention:  providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "test.retention", Value: "v1"}, Redistribution: providercontract.RawPayloadRedistributionRestricted},
	}
	if err := ref.Validate(); err != nil {
		t.Fatalf("test raw reference invalid: %v", err)
	}
	return ref
}

func testProvenance(t *testing.T, ref providercontract.RawPayloadRef, salt string) canonical.Provenance {
	t.Helper()
	evidenceID := canonical.EvidenceID("evd_" + canonical.DigestBytes([]byte("quality-evidence\x00" + salt)).Value[:24])
	evidence, err := ref.AsEvidenceRef(canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(evidenceID), ContractVersion: canonical.EvidenceContractV2})
	if err != nil {
		t.Fatalf("AsEvidenceRef() error: %v", err)
	}
	inputs := []canonical.LineageInput{{Kind: canonical.LineageInputKindEvidence, Evidence: &evidence}}
	fingerprint, err := canonical.ComputeInputFingerprint(inputs)
	if err != nil {
		t.Fatalf("ComputeInputFingerprint() error: %v", err)
	}
	result := canonical.Provenance{
		ContractVersion:  canonical.ProvenanceContractV1,
		ID:               "pvn_" + canonical.DigestBytes([]byte("quality-provenance\x00" + salt)).Value[:24],
		Inputs:           inputs,
		InputFingerprint: fingerprint,
		Producer:         canonical.ComponentIdentity{ID: "cmp_quality_test_normalizer", Kind: canonical.ComponentKindNormalizer, Name: "quality test normalizer", Version: canonical.VersionIdentity{Namespace: "test.normalizer", Value: "v1"}},
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("test provenance invalid: %v", err)
	}
	return result
}

func testInput(t *testing.T, side, value string, present bool) ObservationInput {
	t.Helper()
	mapping := ApprovedVIXMapping()
	rawValue := strings.ToLower(strings.NewReplacer("/", "_", " ", "_").Replace(value))
	if side == "left" {
		return makeInput(t, "mobs_"+side, mapping.LeftSeries, mapping.LeftSemantics, value, present, "rpa_quality_left_"+rawValue)
	}
	return makeInput(t, "mobs_"+side, mapping.RightSeries, mapping.RightSemantics, value, present, "rpa_quality_right_"+rawValue)
}

func makeInput(t *testing.T, id string, series SourceSeriesIdentity, semantics SeriesSemantics, value string, present bool, rawID string) ObservationInput {
	t.Helper()
	body := []byte(id + "\x00" + rawID)
	ref := testRawRef(t, rawID, series.Provider, series.Source, body)
	return ObservationInput{ID: id, Series: series, Semantics: semantics, ObservationDate: "2026-08-21", Value: SourceValue{Present: present, SourceValue: value}, InformationState: InformationState{Mode: string("CURRENT")}, AcquiredAt: qualityTestTime, SourcePayload: ref, Provenance: testProvenance(t, ref, id)}
}

func macroInput(t *testing.T, side, value string) ObservationInput {
	t.Helper()
	mapping := ApprovedVIXMapping()
	selector, semantics := mapping.LeftSeries, mapping.LeftSemantics
	if side == "right" {
		selector, semantics = mapping.RightSeries, mapping.RightSemantics
	}
	ref := testRawRef(t, "rpa_macro_bridge_"+strings.ToLower(side), selector.Provider, selector.Source, []byte("macro-"+side))
	state := macroevidence.InformationState{Mode: macroevidence.InformationStateCurrent}
	series := macroevidence.MacroSeries{ID: macroevidence.MacroSeriesID("mser_macro_" + side), ProviderSeriesID: selector.ProviderSeriesID, Title: "fixture", ObservationStart: "2026-08-21", ObservationEnd: "2026-08-21", Frequency: "Daily", FrequencyCode: "D", Units: semantics.Units, RequestedInformation: state, SourcePayload: ref, Provenance: testProvenance(t, ref, "macro-series-"+side)}
	number := 14.5
	observation := macroevidence.MacroObservation{ID: "mobs_macro_" + side, Series: series.ID, ProviderSeriesID: selector.ProviderSeriesID, ObservationDate: "2026-08-21", Value: macroevidence.MacroValue{Present: true, SourceValue: value, Number: &number}, RequestedInformation: state, AcquiredAt: qualityTestTime, SourcePayload: ref, Provenance: testProvenance(t, ref, "macro-observation-"+side)}
	input, err := FromMacroObservation(series, observation, VIXMeasurement, VIXObservationSemantics)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestProviderNeutralMacroObservationBridge(t *testing.T) {
	registry, err := NewApprovedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	left, right := macroInput(t, "left", "14.50"), macroInput(t, "right", "14.5")
	check, err := Check(registry, left, right, qualityTestTime.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if check.Outcome != OutcomeAgree || check.Lineage != LineageSameOriginRedistributed {
		t.Fatalf("provider-neutral bridge result = %s/%s", check.Outcome, check.Lineage)
	}
}

func TestApprovedVIXDistributionMappingAgreesWithoutIndependentCorroboration(t *testing.T) {
	registry, err := NewApprovedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	left := testInput(t, "left", "14.50", true)
	right := testInput(t, "right", "14.5", true)
	check, err := Check(registry, left, right, qualityTestTime.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if check.Outcome != OutcomeAgree || check.ReasonCode != ReasonValuesEqual {
		t.Fatalf("outcome = %s/%s, want AGREE/VALUES_EQUAL", check.Outcome, check.ReasonCode)
	}
	if check.Lineage != LineageSameOriginRedistributed || check.IndependentSourceCount != 1 {
		t.Fatalf("lineage = %s, independent count = %d", check.Lineage, check.IndependentSourceCount)
	}
	if check.LeftValue != "14.50" || check.RightValue != "14.5" || check.AbsoluteDifference != "0" {
		t.Fatalf("lexical values/difference were not preserved: %#v", check)
	}
	if len(check.LeftEvidence) == 0 || len(check.RightEvidence) == 0 || len(check.Provenance.Inputs) != 2 {
		t.Fatalf("provenance did not preserve both source acquisitions")
	}
}

func TestCrossSourceOutcomeMatrix(t *testing.T) {
	cases := []struct {
		name   string
		left   SourceValue
		right  SourceValue
		want   ComparisonOutcome
		reason ReasonCode
	}{
		{"diverge", SourceValue{Present: true, SourceValue: "14.50"}, SourceValue{Present: true, SourceValue: "14.51"}, OutcomeDiverge, ReasonValuesDiffer},
		{"left_missing", SourceValue{Present: false, SourceValue: "."}, SourceValue{Present: true, SourceValue: "14.5"}, OutcomeLeftMissing, ReasonLeftMissing},
		{"right_missing", SourceValue{Present: true, SourceValue: "14.5"}, SourceValue{Present: false, SourceValue: "N/A"}, OutcomeRightMissing, ReasonRightMissing},
		{"both_missing", SourceValue{Present: false, SourceValue: "."}, SourceValue{Present: false, SourceValue: "N/A"}, OutcomeBothMissing, ReasonBothMissing},
	}
	registry, err := NewApprovedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			left := testInput(t, "left", test.left.SourceValue, test.left.Present)
			right := testInput(t, "right", test.right.SourceValue, test.right.Present)
			check, err := Check(registry, left, right, qualityTestTime.Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			if check.Outcome != test.want || check.ReasonCode != test.reason {
				t.Fatalf("got %s/%s, want %s/%s", check.Outcome, check.ReasonCode, test.want, test.reason)
			}
		})
	}
}

func TestCrossSourceNegativeSemanticControls(t *testing.T) {
	registry, err := NewApprovedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	baseLeft := testInput(t, "left", "14.5", true)
	baseRight := testInput(t, "right", "14.5", true)
	cases := []struct {
		name   string
		edit   func(*ObservationInput, *ObservationInput)
		reason ReasonCode
	}{
		{"unknown_mapping_similar_title", func(left, right *ObservationInput) { right.Series.ProviderSeriesID = "VIX_LIKE_TITLE" }, ReasonMappingNotApproved},
		{"unit_mismatch", func(left, right *ObservationInput) { right.Semantics.Units = "percent" }, ReasonUnitMismatch},
		{"frequency_mismatch", func(left, right *ObservationInput) { right.Semantics.FrequencyCode = "M" }, ReasonFrequencyMismatch},
		{"release_semantics_mismatch", func(left, right *ObservationInput) { right.Semantics.ReleaseSemantics = "PUBLICATION_INSTANT_REQUIRED" }, ReasonReleaseSemanticsMismatch},
		{"information_state_mismatch", func(left, right *ObservationInput) {
			right.InformationState.Mode = "VINTAGE_DATE"
			right.InformationState.Date = "2026-08-20"
		}, ReasonInformationStateMismatch},
		{"date_mismatch", func(left, right *ObservationInput) { right.ObservationDate = "2026-08-20" }, ReasonObservationDateMismatch},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			left, right := baseLeft, baseRight
			test.edit(&left, &right)
			check, err := Check(registry, left, right, qualityTestTime.Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			if check.Outcome != OutcomeNotComparable || check.ReasonCode != test.reason {
				t.Fatalf("got %s/%s, want NOT_COMPARABLE/%s", check.Outcome, check.ReasonCode, test.reason)
			}
		})
	}
}

func TestAcquisitionTimeCannotMasqueradeAsPublicationTime(t *testing.T) {
	input := testInput(t, "left", "14.5", true)
	publication := qualityTestTime.Add(time.Minute)
	input.PublicationTime = &publication
	if err := input.Validate(); err == nil {
		t.Fatal("accepted publication time after acquisition time")
	}
}

func TestTreasuryAndFREDDGS10AreNotImplicitlyEquivalent(t *testing.T) {
	registry, err := NewApprovedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	left := testInput(t, "left", "4.25", true)
	right := testInput(t, "right", "4.25", true)
	left.Series = SourceSeriesIdentity{Provider: canonical.ProviderIdentity{ID: "pvd_treasury", Namespace: "treasury.interest_rates"}, Source: canonical.SourceIdentity{ID: "src_treasury_daily_par_yield_curve", Kind: canonical.SourceKindPublisher}, ProviderSeriesID: "daily_treasury_par_yield_curve:BC_10YEAR"}
	left.SourcePayload.Provider = left.Series.Provider
	left.SourcePayload.Source = &left.Series.Source
	left.Semantics = SeriesSemantics{MeasurementType: "TREASURY_PAR_YIELD", Units: "percent", FrequencyCode: "D", ObservationSemantics: "DAILY_PAR_YIELD", ReleaseSemantics: "NO_RELEASE_INSTANT_ASSERTED"}
	right.Series = SourceSeriesIdentity{Provider: canonical.ProviderIdentity{ID: "pvd_fred", Namespace: "fred.alfred.data_api"}, Source: canonical.SourceIdentity{ID: "src_fred_economic_data", Kind: canonical.SourceKindDataset}, ProviderSeriesID: "DGS10"}
	right.SourcePayload.Provider = right.Series.Provider
	right.SourcePayload.Source = &right.Series.Source
	right.Semantics = SeriesSemantics{MeasurementType: "TREASURY_CONSTANT_MATURITY_YIELD", Units: "Percent", FrequencyCode: "D", ObservationSemantics: "DAILY_INVESTMENT_BASIS", ReleaseSemantics: "NO_RELEASE_INSTANT_ASSERTED"}
	// This test intentionally checks the registry boundary, not provider
	// acquisition. The changed series identities no longer match the only
	// approved mapping and therefore cannot be compared.
	check, err := Check(registry, left, right, qualityTestTime.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if check.Outcome != OutcomeNotComparable || check.ReasonCode != ReasonMappingNotApproved {
		t.Fatalf("got %s/%s", check.Outcome, check.ReasonCode)
	}
}

func TestIndependentMappingIsDowngradedForDuplicateAcquisition(t *testing.T) {
	mapping := ApprovedVIXMapping()
	mapping.Lineage = LineageIndependent
	mapping.RightSeries = mapping.LeftSeries
	mapping.RightSemantics = mapping.LeftSemantics
	registry, err := NewRegistry([]Mapping{mapping})
	if err != nil {
		t.Fatal(err)
	}
	left := testInput(t, "left", "14.5", true)
	right := testInput(t, "right", "14.5", true)
	right.Series = left.Series
	right.SourcePayload = left.SourcePayload
	check, err := Check(registry, left, right, qualityTestTime.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if check.IndependentSourceCount == 2 || check.Lineage != LineageSameOriginRedistributed {
		t.Fatalf("duplicate acquisition counted independently: %s/%d", check.Lineage, check.IndependentSourceCount)
	}
}

func TestStableIDsAndFailClosedValidation(t *testing.T) {
	registry, err := NewApprovedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	left := testInput(t, "left", "14.5", true)
	right := testInput(t, "right", "14.5", true)
	first, err := Check(registry, left, right, qualityTestTime.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Check(registry, left, right, qualityTestTime.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("same semantic input/policy changed result ID: %s != %s", first.ID, second.ID)
	}
	if err := first.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := ApprovedVIXMapping()
	bad.Policy.Tolerance = "0.01"
	if _, err := NewRegistry([]Mapping{bad}); err == nil {
		t.Fatal("accepted arbitrary non-zero exact-decimal tolerance")
	}
	badResult := first
	badResult.Outcome = ComparisonOutcome("WINNER_LEFT")
	if err := badResult.Validate(); err == nil {
		t.Fatal("accepted unsupported result outcome")
	}
	if _, err := parseDecimal("NaN"); err == nil {
		t.Fatal("accepted NaN")
	}
	if _, err := parseDecimal("Inf"); err == nil {
		t.Fatal("accepted Inf")
	}
}

func TestRangeCheckReportsMissingRowsAndCompleteness(t *testing.T) {
	registry, err := NewApprovedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	left := testInput(t, "left", "14.5", true)
	right := testInput(t, "right", "14.5", true)
	left.ObservationDate, right.ObservationDate = "2026-08-20", "2026-08-20"
	left2 := testInput(t, "left2", "15.0", true)
	left2.ObservationDate = "2026-08-21"
	right2 := testInput(t, "right2", "15.0", true)
	right2.ObservationDate = "2026-08-22"
	result, checks, err := CheckRange(registry, RangeRequest{MappingID: VIXMappingID, Start: "2026-08-20", End: "2026-08-22", Left: []ObservationInput{left, left2}, Right: []ObservationInput{right, right2}, LeftComplete: RangeCompletenessComplete, RightComplete: RangeCompletenessUnknown, EvaluatedAt: qualityTestTime.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 1 || result.AgreementCount != 1 || len(result.LeftOnly) != 1 || len(result.RightOnly) != 1 || result.OverlapStart != "2026-08-20" || result.OverlapEnd != "2026-08-20" || result.Completeness != RangeCompletenessIncomplete {
		t.Fatalf("unexpected range result: %#v", result)
	}
	if result.ID == "" || !strings.HasPrefix(result.ID, RangeIDPrefix) {
		t.Fatalf("invalid range ID %q", result.ID)
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("range result validation failed: %v", err)
	}
}
