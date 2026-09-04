package fred

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/releaseevidence"
)

func TestFREDReleaseCalendarPreservesDateOnlyAndFutureNoDataSemantics(t *testing.T) {
	metadataBody := `{"realtime_start":"2026-01-01","realtime_end":"9999-12-31","order_by":"release_id","sort_order":"asc","count":2,"offset":0,"limit":2,"releases":[{"id":10,"realtime_start":"2026-01-01","realtime_end":"9999-12-31","name":"Consumer Price Index","press_release":true,"link":"https://www.bls.gov/cpi/"},{"id":20,"realtime_start":"2026-01-01","realtime_end":"9999-12-31","name":"Employment Situation","press_release":true,"link":"https://www.bls.gov/news.release/empsit.htm"}]}`
	datesBody := `{"realtime_start":"2026-01-01","realtime_end":"9999-12-31","order_by":"release_date","sort_order":"asc","count":2,"offset":0,"limit":2,"release_dates":[{"release_id":10,"release_name":"Consumer Price Index","date":"2026-09-10"},{"release_id":20,"release_name":"Employment Situation","date":"2026-10-02"}]}`
	var seenKey string
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("api_key") != fixtureAPIKey {
			t.Errorf("API key missing from request")
		}
		seenKey = request.URL.Query().Get("api_key")
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/releases" {
			_, _ = writer.Write([]byte(metadataBody))
			return
		}
		_, _ = writer.Write([]byte(datesBody))
	}))
	defer server.Close()
	provider, deps := fixtureDependencies(t, server.URL, server.Client())
	result, err := provider.AcquireReleaseCalendar(context.Background(), deps, ReleaseCalendarRequest{MetadataPayloadID: "rpa_fred_release_metadata", DatesPayloadID: "rpa_fred_release_dates", Retention: fixtureRetention(), IncludeReleaseDatesWithNoData: true, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if seenKey != fixtureAPIKey || len(result.Releases) != 2 || result.Completeness != CompletenessComplete || len(result.RawPayloads) != 2 {
		t.Fatalf("result = %+v", result)
	}
	for _, release := range result.Releases {
		if release.ScheduledDate == nil || release.ScheduledLocalTime != nil || release.ScheduledTimezone != nil || release.ScheduledInstant != nil || release.ActualReleaseInstant != nil {
			t.Fatalf("FRED date-only timing was widened: %+v", release)
		}
		if release.TimingAuthority != releaseevidence.TimingDateOnlySchedule || release.Status != releaseevidence.ReleaseStatusUnknown {
			t.Fatalf("FRED timing/status = %+v", release)
		}
		if release.AcquiredAt.Before(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("acquisition time is not distinct evidence: %v", release.AcquiredAt)
		}
		encoded, _ := json.Marshal(release)
		if strings.Contains(string(encoded), fixtureAPIKey) {
			t.Fatal("FRED API key leaked into normalized release evidence")
		}
	}
	stored, err := deps.Store.Get(context.Background(), result.RawPayloads[0].Ref)
	if err != nil || string(stored) != metadataBody {
		t.Fatalf("metadata raw bytes were not exact: %v", err)
	}
}

func TestFREDReleaseCalendarIncludesNoDataFutureDatesAndFailsClosedOnInconsistentPages(t *testing.T) {
	metadataBody := `{"realtime_start":"2026-01-01","realtime_end":"9999-12-31","order_by":"release_id","sort_order":"asc","count":1,"offset":0,"limit":1,"releases":[{"id":10,"realtime_start":"2026-01-01","realtime_end":"9999-12-31","name":"Consumer Price Index","press_release":true}]}`
	datePage := func(offset, count int) string {
		return `{"realtime_start":"2026-01-01","realtime_end":"9999-12-31","order_by":"release_date","sort_order":"asc","count":` + string(rune('0'+count)) + `,"offset":` + string(rune('0'+offset)) + `,"limit":1,"release_dates":[{"release_id":10,"release_name":"Consumer Price Index","date":"2026-09-10"}]}`
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/releases" {
			_, _ = writer.Write([]byte(metadataBody))
			return
		}
		// The second page deliberately repeats offset zero while reporting a
		// multi-page result. The adapter must not call this complete.
		if request.URL.Query().Get("offset") == "0" {
			_, _ = writer.Write([]byte(datePage(0, 2)))
			return
		}
		_, _ = writer.Write([]byte(datePage(0, 2)))
	}))
	defer server.Close()
	provider, deps := fixtureDependencies(t, server.URL, server.Client())
	result, err := provider.AcquireReleaseCalendar(context.Background(), deps, ReleaseCalendarRequest{MetadataPayloadID: "rpa_fred_release_bad_metadata", DatesPayloadID: "rpa_fred_release_bad_dates", Retention: fixtureRetention(), IncludeReleaseDatesWithNoData: true, PageSize: 1, MaxPages: 3})
	if err == nil || result.Completeness == CompletenessComplete {
		t.Fatalf("inconsistent pagination was accepted: %+v", result)
	}

	goodServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/releases" {
			_, _ = writer.Write([]byte(metadataBody))
			return
		}
		good := `{"realtime_start":"2026-01-01","realtime_end":"9999-12-31","order_by":"release_date","sort_order":"asc","count":1,"offset":0,"limit":1,"release_dates":[{"release_id":10,"release_name":"Consumer Price Index","date":"2026-09-10"}]}`
		_, _ = writer.Write([]byte(good))
	}))
	defer goodServer.Close()
	provider, deps = fixtureDependencies(t, goodServer.URL, goodServer.Client())
	result, err = provider.AcquireReleaseCalendar(context.Background(), deps, ReleaseCalendarRequest{MetadataPayloadID: "rpa_fred_release_good_metadata", DatesPayloadID: "rpa_fred_release_good_dates", Retention: fixtureRetention(), IncludeReleaseDatesWithNoData: true, PageSize: 1})
	if err != nil || len(result.Releases) != 1 || result.Releases[0].DataAvailability != releaseevidence.DataAvailabilityNoData {
		t.Fatalf("future/no-data semantics = %+v, %v", result, err)
	}
}

func TestFREDCalendarProviderNeutralTypeIsNotFREDDTO(t *testing.T) {
	typeOfRelease := reflect.TypeOf(releaseevidence.EconomicRelease{})
	if typeOfRelease.PkgPath() != "jax-trading-assistant/libs/releaseevidence" {
		t.Fatalf("release type package = %q", typeOfRelease.PkgPath())
	}
	for index := 0; index < typeOfRelease.NumField(); index++ {
		field := typeOfRelease.Field(index)
		if strings.Contains(field.Type.PkgPath(), "/fred") || strings.Contains(field.Type.PkgPath(), "golang-ical") {
			t.Fatalf("provider-specific field leaked into normalized contract: %s %s", field.Name, field.Type)
		}
	}
}
