package bls

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/releaseevidence"
)

func parseCalendar(raw []byte, ref providercontract.RawPayloadRef) ([]releaseevidence.EconomicRelease, error) {
	calendar, err := ics.ParseCalendar(strings.NewReader(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("BLS iCalendar parse failed: %w", err)
	}
	events := calendar.Events()
	if len(events) == 0 {
		return nil, fmt.Errorf("BLS iCalendar contains no VEVENT entries")
	}
	seenUIDs := make(map[string]struct{}, len(events))
	out := make([]releaseevidence.EconomicRelease, 0, len(events))
	for _, event := range events {
		release, err := normalizeEvent(event, ref)
		if err != nil {
			return nil, err
		}
		if _, exists := seenUIDs[release.SourceReleaseID]; exists {
			return nil, fmt.Errorf("BLS iCalendar contains duplicate UID %q", release.SourceReleaseID)
		}
		seenUIDs[release.SourceReleaseID] = struct{}{}
		out = append(out, release)
	}
	return out, nil
}

func normalizeEvent(event *ics.VEvent, ref providercontract.RawPayloadRef) (releaseevidence.EconomicRelease, error) {
	uid := propertyValue(event, ics.ComponentPropertyUniqueId)
	if uid == "" {
		return releaseevidence.EconomicRelease{}, fmt.Errorf("BLS VEVENT UID is required")
	}
	name := propertyValue(event, ics.ComponentPropertySummary)
	if name == "" {
		return releaseevidence.EconomicRelease{}, fmt.Errorf("BLS VEVENT %q SUMMARY is required", uid)
	}
	start := event.GetProperty(ics.ComponentPropertyDtStart)
	if start == nil {
		return releaseevidence.EconomicRelease{}, fmt.Errorf("BLS VEVENT %q DTSTART is required", uid)
	}
	date, local, timezone, instant, endInstant, err := parseScheduleTime(start, nil)
	if err != nil {
		return releaseevidence.EconomicRelease{}, fmt.Errorf("BLS VEVENT %q DTSTART: %w", uid, err)
	}
	if end := event.GetProperty(ics.ComponentPropertyDtEnd); end != nil && instant != nil {
		_, _, _, _, endInstant, err = parseScheduleTime(end, timezone)
		if err != nil {
			return releaseevidence.EconomicRelease{}, fmt.Errorf("BLS VEVENT %q DTEND: %w", uid, err)
		}
	}

	revision, err := parseRevision(event)
	if err != nil {
		return releaseevidence.EconomicRelease{}, fmt.Errorf("BLS VEVENT %q revision metadata: %w", uid, err)
	}
	revisionKey := revisionKey(revision, date)
	provider := ProviderIdentity
	source := SourceIdentity
	id := releaseevidence.DeterministicID(provider.ID, source.ID, uid, stringValue(date), revisionKey)
	provenance, err := releaseevidence.MakeProvenance([]providercontract.RawPayloadRef{ref}, canonical.ComponentIdentity{
		ID: NormalizerID, Kind: canonical.ComponentKindNormalizer, Name: "BLS release deterministic normalizer",
		Version: canonical.VersionIdentity{Namespace: "jax.bls.normalizer", Value: NormalizerVersion}, Provider: &provider,
	}, id)
	if err != nil {
		return releaseevidence.EconomicRelease{}, err
	}
	status := releaseevidence.ReleaseStatusScheduled
	if value := strings.ToUpper(propertyValue(event, ics.ComponentPropertyStatus)); value == "CANCELLED" {
		status = releaseevidence.ReleaseStatusCancelled
	}
	release := releaseevidence.EconomicRelease{
		ContractVersion: releaseevidence.EconomicReleaseContractV1,
		ID:              id, Provider: provider, Source: source, SourceReleaseID: uid, Name: name,
		Description: propertyValue(event, ics.ComponentPropertyDescription), ScheduledDate: date,
		ScheduledLocalTime: local, ScheduledTimezone: timezone, ScheduledInstant: instant,
		ScheduledEndInstant: endInstant, ActualReleaseInstant: nil,
		TimingAuthority: releaseevidence.TimingUnknown, Status: status,
		DataAvailability: releaseevidence.DataAvailabilityUnknown, AcquiredAt: ref.ReceivedAt,
		SourcePayloads: []providercontract.RawPayloadRef{ref}, Revision: revision, Provenance: provenance,
	}
	if local != nil {
		release.TimingAuthority = releaseevidence.TimingScheduledLocalTime
	} else {
		release.TimingAuthority = releaseevidence.TimingDateOnlySchedule
	}
	if err := release.Validate(); err != nil {
		return releaseevidence.EconomicRelease{}, err
	}
	return release, nil
}

func parseScheduleTime(property *ics.IANAProperty, fallbackTimezone *string) (*releaseevidence.Date, *string, *string, *time.Time, *time.Time, error) {
	value := strings.TrimSpace(property.Value)
	if len(value) == 8 {
		if _, err := time.Parse("20060102", value); err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("date-only DTSTART is malformed")
		}
		date := releaseevidence.Date(value[:4] + "-" + value[4:6] + "-" + value[6:8])
		return &date, nil, nil, nil, nil, nil
	}
	if (len(value) != 15 && len(value) != 16) || value[8] != 'T' {
		return nil, nil, nil, nil, nil, fmt.Errorf("date-time value is malformed")
	}
	utcValue := strings.HasSuffix(value, "Z")
	base := strings.TrimSuffix(value, "Z")
	parsed, err := time.Parse("20060102T150405", base)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("date-time value is malformed")
	}
	date := releaseevidence.Date(base[:4] + "-" + base[4:6] + "-" + base[6:8])
	localTime := base[9:11] + ":" + base[11:13] + ":" + base[13:15]
	timezone := ""
	if values := property.ICalParameters["TZID"]; len(values) > 1 {
		return nil, nil, nil, nil, nil, fmt.Errorf("TZID must contain one value")
	} else if len(values) == 1 {
		timezone = canonicalBLSTimezone(values[0])
	} else if fallbackTimezone != nil {
		timezone = *fallbackTimezone
	} else if utcValue {
		timezone = "UTC"
	} else {
		// The BLS feed documents all calendar times as Eastern Time. This is an
		// adapter rule, not machine-local timezone inference.
		timezone = BLSReleaseTimezone
	}
	if utcValue && len(property.ICalParameters["TZID"]) > 0 {
		return nil, nil, nil, nil, nil, fmt.Errorf("UTC DTSTART conflicts with TZID")
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("invalid timezone %q", timezone)
	}
	if utcValue {
		parsed = parsed.UTC()
	} else {
		parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, location)
	}
	return &date, &localTime, &timezone, utc(parsed), nil, nil
}

func parseRevision(event *ics.VEvent) (*releaseevidence.SourceRevision, error) {
	var revision releaseevidence.SourceRevision
	hasRevision := false
	if property := event.GetProperty(ics.ComponentPropertySequence); property != nil {
		sequence, err := strconv.Atoi(strings.TrimSpace(property.Value))
		if err != nil || sequence < 0 {
			return nil, fmt.Errorf("SEQUENCE is invalid")
		}
		revision.Sequence = &sequence
		hasRevision = true
	}
	if property := event.GetProperty(ics.ComponentPropertyDtstamp); property != nil {
		value, err := parseMetadataTime(property)
		if err != nil {
			return nil, fmt.Errorf("DTSTAMP: %w", err)
		}
		revision.CalendarObjectStamp = value
		hasRevision = true
	}
	if property := event.GetProperty(ics.ComponentPropertyLastModified); property != nil {
		value, err := parseMetadataTime(property)
		if err != nil {
			return nil, fmt.Errorf("LAST-MODIFIED: %w", err)
		}
		revision.LastModifiedAt = value
		hasRevision = true
	}
	if !hasRevision {
		return nil, nil
	}
	return &revision, nil
}

func parseMetadataTime(property *ics.IANAProperty) (*time.Time, error) {
	value := strings.TrimSpace(property.Value)
	if len(value) != 16 || value[8] != 'T' || !strings.HasSuffix(value, "Z") {
		return nil, fmt.Errorf("must be an explicit UTC timestamp")
	}
	parsed, err := time.Parse("20060102T150405Z", value)
	if err != nil {
		return nil, err
	}
	return utc(parsed), nil
}

func canonicalBLSTimezone(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "\"")
	switch value {
	case "Eastern Standard Time", "Eastern Time", "US/Eastern":
		return BLSReleaseTimezone
	default:
		return value
	}
}

func propertyValue(event *ics.VEvent, property ics.ComponentProperty) string {
	if value := event.GetProperty(property); value != nil {
		return strings.TrimSpace(value.Value)
	}
	return ""
}

func revisionKey(revision *releaseevidence.SourceRevision, date *releaseevidence.Date) string {
	parts := []string{stringValue(date)}
	if revision == nil {
		return strings.Join(parts, "|")
	}
	if revision.Sequence != nil {
		parts = append(parts, "sequence="+strconv.Itoa(*revision.Sequence))
	}
	if revision.LastModifiedAt != nil {
		parts = append(parts, "last_modified="+revision.LastModifiedAt.Format(time.RFC3339Nano))
	}
	return strings.Join(parts, "|")
}

func stringValue(value *releaseevidence.Date) string {
	if value == nil {
		return ""
	}
	return string(*value)
}
