package bls

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/releaseevidence"
)

func blsRetention() providercontract.RawPayloadRetentionPolicy {
	return providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionNotAuthorized}
}

func blsPolicy() providercontract.OperationalPolicy {
	identity := func(id, name, namespace string) canonical.ComponentIdentity {
		return canonical.ComponentIdentity{ID: id, Kind: canonical.ComponentKindPolicy, Name: name, Version: canonical.VersionIdentity{Namespace: namespace, Value: "fixture/v1"}}
	}
	return providercontract.OperationalPolicy{
		ContractVersion: providercontract.OperationalPolicyContractV1,
		Classification:  providercontract.ClassificationPolicy{ContractVersion: providercontract.ClassificationPolicyContractV1, Identity: identity("cmp_bls_class", "BLS fixture classification", "jax.policy.failure_classification")},
		Retry:           providercontract.RetryPolicy{ContractVersion: providercontract.RetryPolicyContractV1, Identity: identity("cmp_bls_retry", "BLS fixture retry", "jax.policy.retry"), MaximumAttempts: 2, MaximumElapsed: time.Minute, PerAttemptTimeout: 10 * time.Second, RetryableFailures: []providercontract.FailureClass{providercontract.FailureTransportTransient, providercontract.FailureProviderServer, providercontract.FailureTemporaryUnavailable, providercontract.FailureRateLimited, providercontract.FailureAttemptDeadline}, Backoff: providercontract.BackoffPolicy{InitialDelay: time.Millisecond, Multiplier: 2, MaximumDelay: time.Second, Jitter: providercontract.JitterNone}},
		RateLimit:       providercontract.RateLimitPolicy{ContractVersion: providercontract.RateLimitPolicyContractV1, Identity: identity("cmp_bls_rate", "BLS fixture rate limit", "jax.policy.rate_limit"), RequestLimit: 100, Window: time.Minute, ConcurrencyLimit: 4, MaximumProviderDelay: time.Second},
		Health:          providercontract.HealthPolicy{ContractVersion: providercontract.HealthPolicyContractV1, Identity: identity("cmp_bls_health", "BLS fixture health", "jax.policy.health"), DegradedAfterFailures: 1, UnavailableAfterFailures: 3, RecoverySuccesses: 1, AssessmentHorizon: time.Hour},
		Component:       canonical.ComponentIdentity{ID: "cmp_bls_build", Kind: canonical.ComponentKindSoftwareBuild, Name: "BLS fixture executor", Version: canonical.VersionIdentity{Namespace: "git.commit", Value: "bls-fixture"}},
	}
}

func blsFixtureDependencies(t *testing.T, calendarBody func() string) (*Provider, Dependencies) {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		_, _ = writer.Write([]byte(calendarBody()))
	}))
	t.Cleanup(server.Close)
	provider, err := NewProvider(Config{CalendarURL: server.URL + "/bls.ics", MaxResponseBytes: 1 << 20}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := RegisterProvider(registry); err != nil {
		t.Fatal(err)
	}
	executor, err := providercontract.NewOperationalExecutor(registry, blsPolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return provider, Dependencies{Registry: registry, Executor: executor, Store: providercontract.NewMemoryRawPayloadStore()}
}

func calendarFixture() string {
	return "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//BLS//Calendar//EN\r\nX-BLS-TEST:ignored\r\n" +
		"BEGIN:VEVENT\r\nUID:employment@example.bls.gov\r\nDTSTAMP:20260801T120000Z\r\nLAST-MODIFIED:20260801T110000Z\r\nSEQUENCE:2\r\nDTSTART;TZID=Eastern Standard Time:20260115T083000\r\nDTEND;TZID=Eastern Standard Time:20260115T083100\r\nSUMMARY:Employment Situation for December 2025\r\nDESCRIPTION:Official scheduled release\r\nSTATUS:CONFIRMED\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:cpi@example.bls.gov\r\nDTSTAMP:20260801T120100Z\r\nDTSTART;TZID=America/New_York:20260715T083000\r\nDTEND;TZID=America/New_York:20260715T083100\r\nSUMMARY:Consumer Price Index for June 2026\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:date-only@example.bls.gov\r\nDTSTAMP:20260801T120200Z\r\nDTSTART;VALUE=DATE:20260910\r\nSUMMARY:Calendar maintenance day\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
}

func TestBLSCalendarPreservesScheduleSemanticsAndExactRawBytes(t *testing.T) {
	provider, deps := blsFixtureDependencies(t, calendarFixture)
	result, err := provider.AcquireCalendar(context.Background(), deps, CalendarRequest{PayloadID: "rpa_bls_fixture", Retention: blsRetention()})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Releases) != 3 || result.Raw.Ref.SizeBytes != int64(len(calendarFixture())) {
		t.Fatalf("result = %+v", result)
	}
	stored, err := deps.Store.Get(context.Background(), result.Raw.Ref)
	if err != nil || string(stored) != calendarFixture() {
		t.Fatalf("exact raw bytes were not retained: %v", err)
	}
	byUID := make(map[string]releaseevidence.EconomicRelease, len(result.Releases))
	for _, release := range result.Releases {
		byUID[release.SourceReleaseID] = release
		if release.ActualReleaseInstant != nil || release.TimingAuthority == releaseevidence.TimingActualReleaseTime {
			t.Fatalf("calendar fabricated actual timing: %+v", release)
		}
		if release.AcquiredAt.Year() != 2026 {
			t.Fatalf("acquisition time was replaced: %v", release.AcquiredAt)
		}
	}
	employment := byUID["employment@example.bls.gov"]
	if employment.ScheduledLocalTime == nil || *employment.ScheduledLocalTime != "08:30:00" || employment.ScheduledTimezone == nil || *employment.ScheduledTimezone != BLSReleaseTimezone {
		t.Fatalf("local schedule = %+v", employment)
	}
	if got := employment.ScheduledInstant.UTC(); got.Hour() != 13 || got.Minute() != 30 || got.Location() != time.UTC {
		t.Fatalf("standard-time UTC schedule = %v", got)
	}
	if employment.ScheduledEndInstant == nil || employment.ScheduledEndInstant.Hour() != 13 || employment.ScheduledEndInstant.Minute() != 31 || employment.ScheduledEndInstant.Location() != time.UTC || employment.ScheduledEndInstant.Before(*employment.ScheduledInstant) {
		t.Fatalf("standard-time UTC end schedule = %v", employment.ScheduledEndInstant)
	}
	if employment.Revision == nil || employment.Revision.Sequence == nil || *employment.Revision.Sequence != 2 || employment.Revision.CalendarObjectStamp.Equal(*employment.ScheduledInstant) || employment.Revision.LastModifiedAt.Equal(*employment.ScheduledInstant) {
		t.Fatalf("calendar metadata was conflated with schedule = %+v", employment.Revision)
	}
	cpi := byUID["cpi@example.bls.gov"]
	if cpi.ScheduledInstant == nil || cpi.ScheduledInstant.Hour() != 12 || cpi.ScheduledEndInstant == nil || cpi.ScheduledEndInstant.Hour() != 12 || cpi.ScheduledEndInstant.Minute() != 31 || cpi.TimingAuthority != releaseevidence.TimingScheduledLocalTime {
		t.Fatalf("daylight-time UTC schedule = %+v", cpi)
	}
	dateOnly := byUID["date-only@example.bls.gov"]
	if dateOnly.ScheduledDate == nil || *dateOnly.ScheduledDate != "2026-09-10" || dateOnly.ScheduledLocalTime != nil || dateOnly.ScheduledInstant != nil || dateOnly.TimingAuthority != releaseevidence.TimingDateOnlySchedule {
		t.Fatalf("date-only schedule = %+v", dateOnly)
	}
	encoded, _ := json.Marshal(result.Releases)
	if strings.Contains(string(encoded), "github.com/arran4") {
		t.Fatal("ICS library type leaked into provider-neutral output")
	}
}

func TestBLSCalendarRejectsDuplicateUIDAndInvalidTimezone(t *testing.T) {
	duplicate := func() string {
		return strings.Replace(calendarFixture(), "UID:cpi@example.bls.gov", "UID:employment@example.bls.gov", 1)
	}
	provider, deps := blsFixtureDependencies(t, duplicate)
	if _, err := provider.AcquireCalendar(context.Background(), deps, CalendarRequest{PayloadID: "rpa_bls_duplicate", Retention: blsRetention()}); err == nil {
		t.Fatal("duplicate source UID was accepted")
	}
	invalid := func() string {
		return strings.Replace(calendarFixture(), "TZID=Eastern Standard Time", "TZID=Not/AZone", 1)
	}
	provider, deps = blsFixtureDependencies(t, invalid)
	if _, err := provider.AcquireCalendar(context.Background(), deps, CalendarRequest{PayloadID: "rpa_bls_timezone", Retention: blsRetention()}); err == nil {
		t.Fatal("invalid timezone was accepted")
	}
	endBeforeStart := func() string {
		return strings.Replace(calendarFixture(), "DTEND;TZID=Eastern Standard Time:20260115T083100", "DTEND;TZID=Eastern Standard Time:20260115T082900", 1)
	}
	provider, deps = blsFixtureDependencies(t, endBeforeStart)
	if _, err := provider.AcquireCalendar(context.Background(), deps, CalendarRequest{PayloadID: "rpa_bls_end_before_start", Retention: blsRetention()}); err == nil {
		t.Fatal("DTEND preceding DTSTART was accepted")
	}
}

func TestBLSCalendarRescheduleRetainsRawAcquisitionAndChangesNormalizedRevision(t *testing.T) {
	var mu sync.Mutex
	call := 0
	oldBody := calendarFixture()
	newBody := strings.Replace(oldBody, "20260115T083000", "20260115T100000", 1)
	newBody = strings.Replace(newBody, "20260115T083100", "20260115T100100", 1)
	titleBody := strings.Replace(oldBody, "Employment Situation for December 2025", "Unrelated title", 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mu.Lock()
		call++
		body := oldBody
		if call == 4 {
			body = newBody
		} else if call > 4 {
			body = titleBody
		}
		mu.Unlock()
		writer.Header().Set("Content-Type", "text/calendar")
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	provider, err := NewProvider(Config{CalendarURL: server.URL + "/bls.ics", MaxResponseBytes: 1 << 20}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	registry, _ := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err := RegisterProvider(registry); err != nil {
		t.Fatal(err)
	}
	executor, _ := providercontract.NewOperationalExecutor(registry, blsPolicy(), providercontract.SystemTimeSource{}, nil, nil)
	deps := Dependencies{Registry: registry, Executor: executor, Store: providercontract.NewMemoryRawPayloadStore()}
	first, err := provider.AcquireCalendar(context.Background(), deps, CalendarRequest{PayloadID: "rpa_bls_reschedule_one", Retention: blsRetention()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := provider.AcquireCalendar(context.Background(), deps, CalendarRequest{PayloadID: "rpa_bls_reschedule_two", Retention: blsRetention()})
	if err != nil {
		t.Fatal(err)
	}
	same, err := provider.AcquireCalendar(context.Background(), deps, CalendarRequest{PayloadID: "rpa_bls_reschedule_same", Retention: blsRetention()})
	if err != nil {
		t.Fatal(err)
	}
	third, err := provider.AcquireCalendar(context.Background(), deps, CalendarRequest{PayloadID: "rpa_bls_reschedule_three", Retention: blsRetention()})
	if err != nil {
		t.Fatal(err)
	}
	titleVariant, err := provider.AcquireCalendar(context.Background(), deps, CalendarRequest{PayloadID: "rpa_bls_title_variant", Retention: blsRetention()})
	if err != nil {
		t.Fatal(err)
	}
	if first.Raw.Ref.ID == second.Raw.Ref.ID || first.Raw.Ref.ID == third.Raw.Ref.ID || first.Raw.Ref.Content.Digest == third.Raw.Ref.Content.Digest {
		t.Fatal("reschedule acquisitions were not distinct")
	}
	var firstEmployment, sameEmployment, thirdEmployment, titleEmployment releaseevidence.EconomicRelease
	for _, release := range first.Releases {
		if release.SourceReleaseID == "employment@example.bls.gov" {
			firstEmployment = release
		}
	}
	for _, release := range same.Releases {
		if release.SourceReleaseID == "employment@example.bls.gov" {
			sameEmployment = release
		}
	}
	for _, release := range third.Releases {
		if release.SourceReleaseID == "employment@example.bls.gov" {
			thirdEmployment = release
		}
	}
	for _, release := range titleVariant.Releases {
		if release.SourceReleaseID == "employment@example.bls.gov" {
			titleEmployment = release
		}
	}
	if firstEmployment.ID != sameEmployment.ID {
		t.Fatalf("same semantic schedule changed identity: %s/%s", firstEmployment.ID, sameEmployment.ID)
	}
	if firstEmployment.ID == thirdEmployment.ID || firstEmployment.ScheduledLocalTime == nil || thirdEmployment.ScheduledLocalTime == nil || *firstEmployment.ScheduledLocalTime == *thirdEmployment.ScheduledLocalTime {
		t.Fatalf("normalized same-date reschedule was not distinguishable: %s/%s %+v/%+v", firstEmployment.ID, thirdEmployment.ID, firstEmployment.ScheduledLocalTime, thirdEmployment.ScheduledLocalTime)
	}
	if firstEmployment.ID != titleEmployment.ID {
		t.Fatalf("title change altered source-event identity: %s/%s", firstEmployment.ID, titleEmployment.ID)
	}
	oldStored, err := deps.Store.Get(context.Background(), first.Raw.Ref)
	if err != nil || string(oldStored) != oldBody {
		t.Fatalf("old raw evidence was overwritten: %v", err)
	}
}

func TestBLSCalendarStatusChangeWithoutRevisionMetadataChangesIdentity(t *testing.T) {
	confirmed := calendarFixture()
	for _, property := range []string{
		"DTSTAMP:20260801T120000Z\r\n",
		"LAST-MODIFIED:20260801T110000Z\r\n",
		"SEQUENCE:2\r\n",
	} {
		confirmed = strings.Replace(confirmed, property, "", 1)
	}
	cancelled := strings.Replace(confirmed, "STATUS:CONFIRMED", "STATUS:CANCELLED", 1)
	provider, deps := blsFixtureDependencies(t, func() string { return confirmed })
	first, err := provider.AcquireCalendar(context.Background(), deps, CalendarRequest{PayloadID: "rpa_bls_status_confirmed", Retention: blsRetention()})
	if err != nil {
		t.Fatal(err)
	}
	provider, deps = blsFixtureDependencies(t, func() string { return cancelled })
	second, err := provider.AcquireCalendar(context.Background(), deps, CalendarRequest{PayloadID: "rpa_bls_status_cancelled", Retention: blsRetention()})
	if err != nil {
		t.Fatal(err)
	}
	var firstEmployment, secondEmployment releaseevidence.EconomicRelease
	for _, release := range first.Releases {
		if release.SourceReleaseID == "employment@example.bls.gov" {
			firstEmployment = release
		}
	}
	for _, release := range second.Releases {
		if release.SourceReleaseID == "employment@example.bls.gov" {
			secondEmployment = release
		}
	}
	if firstEmployment.ID == secondEmployment.ID || secondEmployment.Status != releaseevidence.ReleaseStatusCancelled {
		t.Fatalf("status-only revision was not distinguishable: %+v/%+v", firstEmployment, secondEmployment)
	}
}
