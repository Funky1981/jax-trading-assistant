package fred

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/macroevidence"
)

var fixtureAPIKey = strings.Repeat("a", 32)

func fixtureRetention() providercontract.RawPayloadRetentionPolicy {
	return providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionNotAuthorized}
}

func fixturePolicy() providercontract.OperationalPolicy {
	identity := func(id, name, namespace string) canonical.ComponentIdentity {
		return canonical.ComponentIdentity{ID: id, Kind: canonical.ComponentKindPolicy, Name: name, Version: canonical.VersionIdentity{Namespace: namespace, Value: "fixture/v1"}}
	}
	return providercontract.OperationalPolicy{
		ContractVersion: providercontract.OperationalPolicyContractV1,
		Classification:  providercontract.ClassificationPolicy{ContractVersion: providercontract.ClassificationPolicyContractV1, Identity: identity("cmp_fred_test_class", "FRED fixture classification", "jax.policy.failure_classification")},
		Retry:           providercontract.RetryPolicy{ContractVersion: providercontract.RetryPolicyContractV1, Identity: identity("cmp_fred_test_retry", "FRED fixture retry", "jax.policy.retry"), MaximumAttempts: 2, MaximumElapsed: time.Minute, PerAttemptTimeout: 10 * time.Second, RetryableFailures: []providercontract.FailureClass{providercontract.FailureTransportTransient, providercontract.FailureProviderServer, providercontract.FailureTemporaryUnavailable, providercontract.FailureRateLimited, providercontract.FailureAttemptDeadline}, Backoff: providercontract.BackoffPolicy{InitialDelay: time.Millisecond, Multiplier: 2, MaximumDelay: time.Second, Jitter: providercontract.JitterNone}},
		RateLimit:       providercontract.RateLimitPolicy{ContractVersion: providercontract.RateLimitPolicyContractV1, Identity: identity("cmp_fred_test_rate", "FRED fixture rate limit", "jax.policy.rate_limit"), RequestLimit: 100, Window: time.Minute, ConcurrencyLimit: 4, MaximumProviderDelay: time.Second},
		Health:          providercontract.HealthPolicy{ContractVersion: providercontract.HealthPolicyContractV1, Identity: identity("cmp_fred_test_health", "FRED fixture health", "jax.policy.health"), DegradedAfterFailures: 1, UnavailableAfterFailures: 3, RecoverySuccesses: 1, AssessmentHorizon: time.Hour},
		Component:       canonical.ComponentIdentity{ID: "cmp_fred_test_build", Kind: canonical.ComponentKindSoftwareBuild, Name: "FRED fixture executor", Version: canonical.VersionIdentity{Namespace: "git.commit", Value: "fred-fixture"}},
	}
}

func fixtureDependencies(t *testing.T, baseURL string, clients ...*http.Client) (*Provider, Dependencies) {
	t.Helper()
	var client *http.Client
	if len(clients) > 0 {
		client = clients[0]
	}
	provider, err := NewProvider(Config{BaseURL: baseURL, APIKey: fixtureAPIKey, MaxResponseBytes: 1 << 20, MaxPages: 4}, client)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := RegisterFREDProvider(registry); err != nil {
		t.Fatal(err)
	}
	executor, err := providercontract.NewOperationalExecutor(registry, fixturePolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return provider, Dependencies{Registry: registry, Executor: executor, Store: providercontract.NewMemoryRawPayloadStore()}
}

func seriesFixture() string {
	return seriesFixtureAt("2026-08-01", "2026-08-01")
}

func seriesFixtureAt(realtimeStart, realtimeEnd string) string {
	return `{"realtime_start":"` + realtimeStart + `","realtime_end":"` + realtimeEnd + `","seriess":[{"id":"GDP","realtime_start":"` + realtimeStart + `","realtime_end":"` + realtimeEnd + `","title":"Gross Domestic Product","observation_start":"1947-01-01","observation_end":"2026-07-01","frequency":"Quarterly","frequency_short":"Q","units":"Billions of Dollars","units_short":"Bil. of $","seasonal_adjustment":"Seasonally Adjusted","seasonal_adjustment_short":"SA","last_updated":"2026-08-01 09:30:00-05","notes":"Fixture note"}]}`
}

func vintageFixture() string {
	return `{"realtime_start":"1776-07-04","realtime_end":"9999-12-31","order_by":"vintage_date","sort_order":"asc","count":2,"offset":0,"limit":1000,"vintage_dates":["2025-02-15","2025-03-27"]}`
}

func observationsFixture(value string, realtimeStart, realtimeEnd string, outputType int) string {
	return `{"realtime_start":"` + realtimeStart + `","realtime_end":"` + realtimeEnd + `","observation_start":"2025-01-01","observation_end":"2025-01-01","units":"lin","output_type":` + string(rune('0'+outputType)) + `,"order_by":"observation_date","sort_order":"asc","count":1,"offset":0,"limit":1000,"observations":[{"realtime_start":"` + realtimeStart + `","realtime_end":"` + realtimeEnd + `","date":"2025-01-01","value":"` + value + `"}]}`
}

func TestFREDProviderDefinitionUsesOneLogicalIdentityAndMacroCapability(t *testing.T) {
	definition := FREDProviderDefinition()
	if err := definition.Validate(); err != nil {
		t.Fatal(err)
	}
	if definition.Identity.ID != ProviderID || definition.Identity.ExternalID == nil || definition.Identity.ExternalID.Value != "fred-alfred" {
		t.Fatalf("provider identity = %+v", definition.Identity)
	}
	if len(definition.Capabilities) != 1 || definition.Capabilities[0].ID != providercontract.CapabilityMacroObservation {
		t.Fatalf("capabilities = %+v", definition.Capabilities)
	}
	capability := definition.Capabilities[0]
	if len(capability.CanonicalOutputs) != 0 || len(capability.ProviderNeutralOutputs) != 1 || capability.ProviderNeutralOutputs[0].Schema.Namespace != "jax.macroevidence" || capability.ProviderNeutralOutputs[0].Schema.Value != "macro_observation/v1" {
		t.Fatalf("macro output contract = %+v", capability)
	}
	if definition.Capabilities[0].Authentication.Class != providercontract.AuthenticationAPIKey {
		t.Fatalf("authentication = %+v", definition.Capabilities[0].Authentication)
	}
}

func TestMacroOutputIsOwnedByProviderNeutralPackage(t *testing.T) {
	var output macroevidence.MacroObservation = MacroObservation{}
	if got := fmt.Sprintf("%T", output); !strings.Contains(got, "macroevidence.MacroObservation") {
		t.Fatalf("macro output type = %s; FRED DTO package leaked into output", got)
	}
}

func TestFREDAcquirePersistsExactBytesAndPreservesSeriesObservationsAndVintages(t *testing.T) {
	seriesBody := seriesFixture()
	vintageBody := vintageFixture()
	observationBody := observationsFixture("100.00000000000001", "2025-02-15", "2025-02-15", 1)
	var mu sync.Mutex
	seenQueries := make([]string, 0, 3)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("api_key") != fixtureAPIKey {
			t.Error("FRED API key was not supplied to the test endpoint")
		}
		mu.Lock()
		seenQueries = append(seenQueries, request.URL.RawQuery)
		mu.Unlock()
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		switch request.URL.Path {
		case "/series":
			if request.URL.Query().Get("realtime_start") == "2025-02-15" {
				_, _ = writer.Write([]byte(seriesFixtureAt("2025-02-15", "2025-02-15")))
			} else {
				_, _ = writer.Write([]byte(seriesBody))
			}
		case "/series/vintagedates":
			_, _ = writer.Write([]byte(vintageBody))
		case "/series/observations":
			_, _ = writer.Write([]byte(observationBody))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	provider, deps := fixtureDependencies(t, server.URL, server.Client())
	result, err := provider.Acquire(context.Background(), deps, AcquisitionRequest{SeriesID: "GDP", SeriesPayloadID: "rpa_fred_series_fixture", ObservationPayloadID: "rpa_fred_observation_fixture", VintagePayloadID: "rpa_fred_vintage_fixture", ObservationStart: Date("2025-01-01"), ObservationEnd: Date("2025-01-01"), InformationState: InformationState{Mode: InformationStateAsOf, Date: datePointer(Date("2025-02-15"))}, Retention: fixtureRetention(), IncludeVintageDates: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RawPayloads) != 3 || result.Observations.Completeness != CompletenessComplete || result.VintageDates.Completeness != CompletenessComplete {
		t.Fatalf("acquisition completeness/raw payloads = %q/%q/%d", result.Observations.Completeness, result.VintageDates.Completeness, len(result.RawPayloads))
	}
	if result.Series.ProviderSeriesID != "GDP" || result.Series.FrequencyCode != "Q" || result.Series.Units != "Billions of Dollars" || result.Series.SeasonalAdjustmentCode != "SA" || result.Series.ObservationStart != Date("1947-01-01") {
		t.Fatalf("series metadata = %+v", result.Series)
	}
	if result.Series.ProviderRealtimePeriod == nil || result.Series.ProviderRealtimePeriod.Start != Date("2025-02-15") || result.Series.RequestedInformation.Mode != InformationStateAsOf || result.Series.RequestedInformation.Date == nil || *result.Series.RequestedInformation.Date != Date("2025-02-15") || result.Series.LastUpdated == nil || result.Series.LastUpdated.Location() != time.UTC {
		t.Fatalf("series realtime/last-updated = %+v/%v", result.Series.ProviderRealtimePeriod, result.Series.LastUpdated)
	}
	if len(result.VintageDates.VintageDates) != 2 || result.VintageDates.VintageDates[0] != Date("2025-02-15") {
		t.Fatalf("vintage dates = %+v", result.VintageDates.VintageDates)
	}
	if len(result.Observations.Observations) != 1 {
		t.Fatalf("observations = %+v", result.Observations.Observations)
	}
	observation := result.Observations.Observations[0]
	if !observation.Value.Present || observation.Value.SourceValue != "100.00000000000001" || observation.Value.Number == nil || observation.ObservationDate != Date("2025-01-01") {
		t.Fatalf("observation value/date = %+v/%s", observation.Value, observation.ObservationDate)
	}
	if observation.RealtimePeriod == nil || observation.RealtimePeriod.Start != Date("2025-02-15") || observation.AcquiredAt.IsZero() || observation.AcquiredAt.Location() != time.UTC {
		t.Fatalf("observation information state = %+v acquired=%v", observation.RealtimePeriod, observation.AcquiredAt)
	}
	if strings.Contains(string(mustJSON(t, result.RawPayloads[0].Ref)), fixtureAPIKey) || strings.Contains(observation.Provenance.Inputs[0].Evidence.Provider.ExternalID.Value, fixtureAPIKey) {
		t.Fatal("API key leaked into raw reference or provenance")
	}
	if got, err := deps.Store.Get(context.Background(), result.RawPayloads[2].Ref); err != nil || string(got) != observationBody {
		t.Fatalf("stored exact observation bytes = %q/%v", got, err)
	}
	if result.RawPayloads[2].Ref.Content.Digest != canonical.DigestBytes([]byte(observationBody)) {
		t.Fatalf("observation digest = %+v", result.RawPayloads[2].Ref.Content.Digest)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seenQueries) != 3 || !strings.Contains(seenQueries[0], "realtime_start=2025-02-15") || !strings.Contains(seenQueries[0], "realtime_end=2025-02-15") || !strings.Contains(seenQueries[2], "units=lin") || !strings.Contains(seenQueries[2], "realtime_start=2025-02-15") || !strings.Contains(seenQueries[2], "realtime_end=2025-02-15") {
		t.Fatalf("request queries = %v", seenQueries)
	}
}

func TestFREDMissingValueRemainsExplicitlyMissing(t *testing.T) {
	body := observationsFixture(".", "2025-02-15", "2025-02-15", 1)
	server := fixtureServer(t, map[string]string{"/series": seriesFixture(), "/series/observations": body})
	defer server.Close()
	provider, deps := fixtureDependencies(t, server.URL, server.Client())
	series, err := provider.AcquireSeriesMetadata(context.Background(), deps, SeriesRequest{SeriesID: "GDP", InformationState: InformationState{Mode: InformationStateCurrent}, PayloadID: "rpa_fred_missing_series", Retention: fixtureRetention()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.AcquireObservations(context.Background(), deps, ObservationsRequest{Series: series.Series, ObservationStart: Date("2025-01-01"), ObservationEnd: Date("2025-01-01"), InformationState: InformationState{Mode: InformationStateAsOf, Date: datePointer(Date("2025-02-15"))}, PayloadID: "rpa_fred_missing_observation", Retention: fixtureRetention()})
	if err != nil || len(result.Observations) != 1 {
		t.Fatalf("missing observation result = %+v/%v", result, err)
	}
	if result.Observations[0].Value.Present || result.Observations[0].Value.Number != nil || result.Observations[0].Value.SourceValue != "." {
		t.Fatalf("missing value was not preserved = %+v", result.Observations[0].Value)
	}
}

func TestFREDAsOfAndCurrentViewsCannotBeConfused(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/series":
			_, _ = writer.Write([]byte(seriesFixture()))
		case "/series/observations":
			if request.URL.Query().Get("realtime_start") == "2025-02-15" {
				_, _ = writer.Write([]byte(observationsFixture("100", "2025-02-15", "2025-02-15", 1)))
				return
			}
			_, _ = writer.Write([]byte(observationsFixture("105", "2026-09-02", "2026-09-02", 1)))
		}
	}))
	defer server.Close()
	provider, deps := fixtureDependencies(t, server.URL, server.Client())
	series, err := provider.AcquireSeriesMetadata(context.Background(), deps, SeriesRequest{SeriesID: "GDP", InformationState: InformationState{Mode: InformationStateCurrent}, PayloadID: "rpa_fred_view_series", Retention: fixtureRetention()})
	if err != nil {
		t.Fatal(err)
	}
	base := ObservationsRequest{Series: series.Series, ObservationStart: Date("2025-01-01"), ObservationEnd: Date("2025-01-01"), Retention: fixtureRetention()}
	asOf := base
	asOf.PayloadID = "rpa_fred_asof_observation"
	asOf.InformationState = InformationState{Mode: InformationStateAsOf, Date: datePointer(Date("2025-02-15"))}
	historical, err := provider.AcquireObservations(context.Background(), deps, asOf)
	if err != nil {
		t.Fatal(err)
	}
	current := base
	current.PayloadID = "rpa_fred_current_observation"
	current.InformationState = InformationState{Mode: InformationStateCurrent}
	latest, err := provider.AcquireObservations(context.Background(), deps, current)
	if err != nil {
		t.Fatal(err)
	}
	if historical.Observations[0].Value.SourceValue != "100" || latest.Observations[0].Value.SourceValue != "105" || historical.Observations[0].RequestedInformation.Mode == latest.Observations[0].RequestedInformation.Mode {
		t.Fatalf("current/as-of views were conflated: historical=%+v latest=%+v", historical.Observations[0], latest.Observations[0])
	}
}

func TestFREDInitialReleaseIsExplicitAndDoesNotMasqueradeAsCurrent(t *testing.T) {
	var observationQuery url.Values
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/series":
			_, _ = writer.Write([]byte(seriesFixture()))
		case "/series/observations":
			observationQuery = request.URL.Query()
			if observationQuery.Get("output_type") != "4" {
				t.Error("initial-release request did not use output_type=4")
			}
			_, _ = writer.Write([]byte(observationsFixture("100", "2025-02-15", "2025-02-15", 4)))
		}
	}))
	defer server.Close()
	provider, deps := fixtureDependencies(t, server.URL, server.Client())
	result, err := provider.Acquire(context.Background(), deps, AcquisitionRequest{SeriesID: "GDP", SeriesPayloadID: "rpa_fred_initial_series", ObservationPayloadID: "rpa_fred_initial_observation", ObservationStart: Date("2025-01-01"), ObservationEnd: Date("2025-01-01"), InformationState: InformationState{Mode: InformationStateInitialRelease}, Retention: fixtureRetention()})
	if err != nil {
		t.Fatal(err)
	}
	if observationQuery.Get("output_type") != "4" || observationQuery.Get("vintage_dates") != "" || observationQuery.Get("realtime_start") != "" || observationQuery.Get("realtime_end") != "" {
		t.Fatalf("initial-release query semantics = %v", observationQuery)
	}
	if result.Series.RequestedInformation.Mode != InformationStateCurrent || len(result.Observations.Observations) != 1 {
		t.Fatalf("initial-release metadata/observations = %+v/%+v", result.Series.RequestedInformation, result.Observations.Observations)
	}
	observation := result.Observations.Observations[0]
	if observation.Value.SourceValue != "100" || observation.RequestedInformation.Mode != InformationStateInitialRelease || observation.AcquiredAt.IsZero() {
		t.Fatalf("initial-release normalized result = %+v", observation)
	}
}

func TestFREDSeriesMetadataQueryUsesCompatibleInformationState(t *testing.T) {
	date := Date("2025-02-15")
	for _, testCase := range []struct {
		name           string
		state          InformationState
		expectRealtime bool
	}{
		{name: "current", state: InformationState{Mode: InformationStateCurrent}},
		{name: "as_of", state: InformationState{Mode: InformationStateAsOf, Date: &date}, expectRealtime: true},
		{name: "vintage", state: InformationState{Mode: InformationStateVintage, Date: &date}, expectRealtime: true},
		{name: "initial_release", state: InformationState{Mode: InformationStateInitialRelease}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			query, err := seriesQuery(SeriesRequest{SeriesID: "GDP", InformationState: testCase.state})
			if err != nil {
				t.Fatal(err)
			}
			if query.Get("series_id") != "GDP" || query.Get("file_type") != "json" {
				t.Fatalf("base metadata query = %v", query)
			}
			if testCase.expectRealtime {
				if query.Get("realtime_start") != string(date) || query.Get("realtime_end") != string(date) {
					t.Fatalf("historical metadata query = %v", query)
				}
			} else if query.Get("realtime_start") != "" || query.Get("realtime_end") != "" {
				t.Fatalf("non-historical metadata query = %v", query)
			}
		})
	}
}

func TestFREDRejectsCrossPageObservationAndVintageIntegrityViolations(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		firstDate  string
		secondDate string
	}{
		{name: "duplicate", firstDate: "2025-02-01", secondDate: "2025-02-01"},
		{name: "out_of_order", firstDate: "2025-02-02", secondDate: "2025-02-01"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				date := testCase.firstDate
				if request.URL.Query().Get("offset") == "1" {
					date = testCase.secondDate
				}
				switch request.URL.Path {
				case "/series":
					_, _ = writer.Write([]byte(seriesFixtureAt("2025-02-15", "2025-02-15")))
				case "/series/vintagedates":
					_, _ = writer.Write([]byte(`{"realtime_start":"1776-07-04","realtime_end":"9999-12-31","order_by":"vintage_date","sort_order":"asc","count":2,"offset":` + request.URL.Query().Get("offset") + `,"limit":1,"vintage_dates":["` + date + `"]}`))
				case "/series/observations":
					_, _ = writer.Write([]byte(`{"realtime_start":"2025-02-15","realtime_end":"2025-02-15","observation_start":"2025-01-01","observation_end":"2025-03-01","units":"lin","output_type":1,"order_by":"observation_date","sort_order":"asc","count":2,"offset":` + request.URL.Query().Get("offset") + `,"limit":1,"observations":[{"realtime_start":"2025-02-15","realtime_end":"2025-02-15","date":"` + date + `","value":"100"}]}`))
				}
			}))
			defer server.Close()
			provider, deps := fixtureDependencies(t, server.URL, server.Client())
			series, err := provider.AcquireSeriesMetadata(context.Background(), deps, SeriesRequest{SeriesID: "GDP", InformationState: InformationState{Mode: InformationStateAsOf, Date: datePointer(Date("2025-02-15"))}, PayloadID: providercontract.RawPayloadID("rpa_fred_integrity_series_" + testCase.name), Retention: fixtureRetention()})
			if err != nil {
				t.Fatal(err)
			}
			vintages, err := provider.AcquireVintageDates(context.Background(), deps, VintageDatesRequest{SeriesID: "GDP", PayloadID: providercontract.RawPayloadID("rpa_fred_integrity_vintage_" + testCase.name), Retention: fixtureRetention(), PageSize: 1, MaxPages: 2})
			if err == nil || vintages.Completeness == CompletenessComplete {
				t.Fatalf("invalid vintage pages accepted = %+v/%v", vintages, err)
			}
			observations, err := provider.AcquireObservations(context.Background(), deps, ObservationsRequest{Series: series.Series, ObservationStart: Date("2025-01-01"), ObservationEnd: Date("2025-03-01"), InformationState: InformationState{Mode: InformationStateAsOf, Date: datePointer(Date("2025-02-15"))}, PayloadID: providercontract.RawPayloadID("rpa_fred_integrity_observation_" + testCase.name), Retention: fixtureRetention(), PageSize: 1, MaxPages: 2})
			if err == nil || observations.Completeness == CompletenessComplete {
				t.Fatalf("invalid observation pages accepted = %+v/%v", observations, err)
			}
		})
	}
}

func TestFREDStrictParsingAndCompletenessFailures(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "trailing json", body: seriesFixture() + " {}"},
		{name: "duplicate property", body: `{"realtime_start":"2026-08-01","realtime_start":"2026-08-02","seriess":[]}`},
		{name: "wrong series", body: strings.Replace(seriesFixture(), `"id":"GDP"`, `"id":"CPIAUCSL"`, 1)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := fixtureServer(t, map[string]string{"/series": testCase.body})
			defer server.Close()
			provider, deps := fixtureDependencies(t, server.URL, server.Client())
			if _, err := provider.AcquireSeriesMetadata(context.Background(), deps, SeriesRequest{SeriesID: "GDP", InformationState: InformationState{Mode: InformationStateCurrent}, PayloadID: providercontract.RawPayloadID("rpa_fred_strict_" + strings.ReplaceAll(testCase.name, " ", "_")), Retention: fixtureRetention()}); err == nil {
				t.Fatal("malformed response was accepted")
			}
		})
	}
	server := fixtureServer(t, map[string]string{"/series/vintagedates": `{"realtime_start":"1776-07-04","realtime_end":"9999-12-31","order_by":"vintage_date","sort_order":"asc","count":3,"offset":0,"limit":1,"vintage_dates":["2025-02-15"]}`})
	defer server.Close()
	provider, deps := fixtureDependencies(t, server.URL, server.Client())
	result, err := provider.AcquireVintageDates(context.Background(), deps, VintageDatesRequest{SeriesID: "GDP", PayloadID: "rpa_fred_incomplete_vintages", Retention: fixtureRetention(), PageSize: 1, MaxPages: 1})
	if err == nil || result.Completeness != CompletenessIncomplete {
		t.Fatalf("incomplete vintage result = %+v/%v", result, err)
	}
}

func TestFREDAPIKeyConfigurationIsExplicitAndNeverLoggedInError(t *testing.T) {
	if _, err := NewProvider(Config{BaseURL: DefaultBaseURL, APIKey: "secret"}, nil); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("invalid credential error = %v", err)
	}
	if _, err := LoadConfigFromEnv(); err == nil && strings.Contains(err.Error(), fixtureAPIKey) {
		t.Fatal("environment key leaked")
	}
	if err := (MacroValue{Present: true, SourceValue: "NaN"}).Validate(); err == nil {
		t.Fatal("invalid macro value passed validation")
	}
	if err := (MacroValue{Present: true, SourceValue: "1", Number: floatPointer(math.NaN())}).Validate(); err == nil {
		t.Fatal("NaN macro value passed validation")
	}
	if err := (MacroValue{Present: true, SourceValue: "1", Number: floatPointer(math.Inf(1))}).Validate(); err == nil {
		t.Fatal("infinite macro value passed validation")
	}
	if err := (MacroValue{Present: false, SourceValue: ".", Number: floatPointer(0)}).Validate(); err == nil {
		t.Fatal("missing macro value exposed a numeric convenience value")
	}
}

func TestFREDPaginationCompletenessIsProvenForVintageAndObservationPages(t *testing.T) {
	seriesBody := seriesFixture()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/series":
			_, _ = writer.Write([]byte(seriesBody))
		case "/series/vintagedates":
			if request.URL.Query().Get("offset") == "0" {
				_, _ = writer.Write([]byte(`{"realtime_start":"1776-07-04","realtime_end":"9999-12-31","order_by":"vintage_date","sort_order":"asc","count":3,"offset":0,"limit":2,"vintage_dates":["2025-02-15","2025-03-27"]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"realtime_start":"1776-07-04","realtime_end":"9999-12-31","order_by":"vintage_date","sort_order":"asc","count":3,"offset":2,"limit":2,"vintage_dates":["2025-04-30"]}`))
		case "/series/observations":
			if request.URL.Query().Get("offset") == "0" {
				_, _ = writer.Write([]byte(`{"realtime_start":"2025-02-15","realtime_end":"2025-02-15","observation_start":"2025-01-01","observation_end":"2025-03-01","units":"lin","output_type":1,"order_by":"observation_date","sort_order":"asc","count":3,"offset":0,"limit":2,"observations":[{"realtime_start":"2025-02-15","realtime_end":"2025-02-15","date":"2025-01-01","value":"100"},{"realtime_start":"2025-02-15","realtime_end":"2025-02-15","date":"2025-02-01","value":"101"}]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"realtime_start":"2025-02-15","realtime_end":"2025-02-15","observation_start":"2025-01-01","observation_end":"2025-03-01","units":"lin","output_type":1,"order_by":"observation_date","sort_order":"asc","count":3,"offset":2,"limit":2,"observations":[{"realtime_start":"2025-02-15","realtime_end":"2025-02-15","date":"2025-03-01","value":"102"}]}`))
		}
	}))
	defer server.Close()
	provider, deps := fixtureDependencies(t, server.URL, server.Client())
	series, err := provider.AcquireSeriesMetadata(context.Background(), deps, SeriesRequest{SeriesID: "GDP", InformationState: InformationState{Mode: InformationStateCurrent}, PayloadID: "rpa_fred_page_series", Retention: fixtureRetention()})
	if err != nil {
		t.Fatal(err)
	}
	vintages, err := provider.AcquireVintageDates(context.Background(), deps, VintageDatesRequest{SeriesID: "GDP", PayloadID: "rpa_fred_page_vintage", Retention: fixtureRetention(), PageSize: 2})
	if err != nil || vintages.Completeness != CompletenessComplete || len(vintages.RawPayloads) != 2 || len(vintages.VintageDates) != 3 {
		t.Fatalf("vintage pagination = %+v/%v", vintages, err)
	}
	observations, err := provider.AcquireObservations(context.Background(), deps, ObservationsRequest{Series: series.Series, ObservationStart: Date("2025-01-01"), ObservationEnd: Date("2025-03-01"), InformationState: InformationState{Mode: InformationStateAsOf, Date: datePointer(Date("2025-02-15"))}, PayloadID: "rpa_fred_page_observation", Retention: fixtureRetention(), PageSize: 2})
	if err != nil || observations.Completeness != CompletenessComplete || len(observations.RawPayloads) != 2 || len(observations.Observations) != 3 {
		t.Fatalf("observation pagination = %+v/%v", observations, err)
	}
}

type failingStore struct{}

func (failingStore) Put(context.Context, providercontract.RawPayloadRef, []byte) (providercontract.RawPayloadLocation, error) {
	return providercontract.RawPayloadLocation{}, context.Canceled
}

func (failingStore) Get(context.Context, providercontract.RawPayloadRef) ([]byte, error) {
	return nil, context.Canceled
}

func TestFREDRawPersistenceFailureFailsClosed(t *testing.T) {
	server := fixtureServer(t, map[string]string{"/series": seriesFixture()})
	defer server.Close()
	provider, deps := fixtureDependencies(t, server.URL, server.Client())
	deps.Store = failingStore{}
	result, err := provider.AcquireSeriesMetadata(context.Background(), deps, SeriesRequest{SeriesID: "GDP", InformationState: InformationState{Mode: InformationStateCurrent}, PayloadID: "rpa_fred_persist_failure", Retention: fixtureRetention()})
	if err == nil || result.Raw.Ref.ID != "" || result.Series.ProviderSeriesID != "" {
		t.Fatalf("persistence failure was accepted = %+v/%v", result, err)
	}
}

func fixtureServer(t *testing.T, bodies map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		body, ok := bodies[request.URL.Path]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(body))
	}))
}

func datePointer(value Date) *Date { return &value }

func floatPointer(value float64) *float64 { return &value }

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
