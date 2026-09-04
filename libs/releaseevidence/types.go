// Package releaseevidence contains provider-neutral economic release schedule
// evidence. It deliberately does not model observations, forecasts, surprise,
// market reaction, or execution events.
package releaseevidence

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

const EconomicReleaseContractV1 canonical.ContractVersion = "jax.economic_release/v1"

// Date is a source-supplied calendar date. It is intentionally not a time.Time:
// a date-only release must not become a fabricated midnight instant.
type Date string

func (date Date) Validate() error {
	value := string(date)
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return fmt.Errorf("release date %q must use YYYY-MM-DD", value)
	}
	return nil
}

// RealtimePeriod records a provider information-state boundary. It is not a
// release time and is not an acquisition timestamp.
type RealtimePeriod struct {
	Start Date `json:"start"`
	End   Date `json:"end"`
}

func (period RealtimePeriod) Validate() error {
	if err := period.Start.Validate(); err != nil {
		return fmt.Errorf("realtime start: %w", err)
	}
	if err := period.End.Validate(); err != nil {
		return fmt.Errorf("realtime end: %w", err)
	}
	if period.End < period.Start {
		return fmt.Errorf("realtime end precedes realtime start")
	}
	return nil
}

type TimingAuthority string

const (
	TimingUnknown            TimingAuthority = "UNKNOWN"
	TimingDateOnlySchedule   TimingAuthority = "DATE_ONLY_SCHEDULE"
	TimingScheduledLocalTime TimingAuthority = "SCHEDULED_LOCAL_TIME"
	TimingScheduledInstant   TimingAuthority = "SCHEDULED_INSTANT"
	TimingActualReleaseTime  TimingAuthority = "ACTUAL_RELEASE_TIME"
)

type ReleaseStatus string

const (
	ReleaseStatusUnknown     ReleaseStatus = "UNKNOWN"
	ReleaseStatusScheduled   ReleaseStatus = "SCHEDULED"
	ReleaseStatusRescheduled ReleaseStatus = "RESCHEDULED"
	ReleaseStatusCancelled   ReleaseStatus = "CANCELLED"
	ReleaseStatusReleased    ReleaseStatus = "RELEASED"
)

// DataAvailability is a conservative statement about what the FRED date
// endpoint indicated. It is never a macro observation value.
type DataAvailability string

const (
	DataAvailabilityUnknown   DataAvailability = "UNKNOWN"
	DataAvailabilityIndicated DataAvailability = "DATA_AVAILABLE_INDICATED"
	DataAvailabilityNoData    DataAvailability = "NO_DATA_INDICATED"
)

// SourceRevision preserves generic source revision/metadata semantics without
// exposing an iCalendar library type or a provider response DTO.
type SourceRevision struct {
	Sequence            *int       `json:"sequence,omitempty"`
	CalendarObjectStamp *time.Time `json:"calendar_object_stamp,omitempty"`
	LastModifiedAt      *time.Time `json:"last_modified_at,omitempty"`
}

// EconomicRelease is a provider-neutral schedule/calendar evidence record.
// ScheduledInstant is a scheduled claim, even when derived from a local time;
// it is never an actual publication timestamp.
type EconomicRelease struct {
	ContractVersion      canonical.ContractVersion        `json:"contract_version"`
	ID                   string                           `json:"id"`
	Provider             canonical.ProviderIdentity       `json:"provider"`
	Source               canonical.SourceIdentity         `json:"source"`
	SourceReleaseID      string                           `json:"source_release_id"`
	Name                 string                           `json:"name"`
	Description          string                           `json:"description,omitempty"`
	ReferencePeriod      string                           `json:"reference_period,omitempty"`
	ScheduledDate        *Date                            `json:"scheduled_date,omitempty"`
	ScheduledLocalTime   *string                          `json:"scheduled_local_time,omitempty"`
	ScheduledTimezone    *string                          `json:"scheduled_timezone,omitempty"`
	ScheduledInstant     *time.Time                       `json:"scheduled_instant,omitempty"`
	ScheduledEndInstant  *time.Time                       `json:"scheduled_end_instant,omitempty"`
	ActualReleaseInstant *time.Time                       `json:"actual_release_instant,omitempty"`
	TimingAuthority      TimingAuthority                  `json:"timing_authority"`
	Status               ReleaseStatus                    `json:"status"`
	DataAvailability     DataAvailability                 `json:"data_availability"`
	RealtimePeriod       *RealtimePeriod                  `json:"realtime_period,omitempty"`
	DataLastUpdatedDate  *Date                            `json:"data_last_updated_date,omitempty"`
	PressRelease         *bool                            `json:"press_release,omitempty"`
	SourceLink           string                           `json:"source_link,omitempty"`
	Revision             *SourceRevision                  `json:"revision,omitempty"`
	AcquiredAt           time.Time                        `json:"acquired_at"`
	SourcePayloads       []providercontract.RawPayloadRef `json:"source_payloads"`
	Provenance           canonical.Provenance             `json:"provenance"`
}

var localTimePattern = regexp.MustCompile(`^(?:[01]\d|2[0-3]):[0-5]\d(?::[0-5]\d)?$`)

func (release EconomicRelease) Validate() error {
	if release.ContractVersion != EconomicReleaseContractV1 {
		return fmt.Errorf("economic release contract version must be %q", EconomicReleaseContractV1)
	}
	if !strings.HasPrefix(release.ID, "erl_") {
		return fmt.Errorf("economic release ID must use erl_ prefix")
	}
	if err := release.Provider.Validate(); err != nil {
		return fmt.Errorf("provider: %w", err)
	}
	if err := release.Source.Validate(); err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if strings.TrimSpace(release.SourceReleaseID) == "" || len(release.SourceReleaseID) > 512 {
		return fmt.Errorf("source release identity is required")
	}
	if strings.TrimSpace(release.Name) == "" {
		return fmt.Errorf("release name is required")
	}
	if release.ScheduledDate != nil {
		if err := release.ScheduledDate.Validate(); err != nil {
			return err
		}
	}
	if release.ScheduledLocalTime != nil {
		if !localTimePattern.MatchString(*release.ScheduledLocalTime) {
			return fmt.Errorf("scheduled local time %q is invalid", *release.ScheduledLocalTime)
		}
		if release.ScheduledTimezone == nil || strings.TrimSpace(*release.ScheduledTimezone) == "" {
			return fmt.Errorf("scheduled timezone is required with a local time")
		}
		if _, err := time.LoadLocation(*release.ScheduledTimezone); err != nil {
			return fmt.Errorf("scheduled timezone %q is invalid: %w", *release.ScheduledTimezone, err)
		}
	}
	if release.ScheduledTimezone != nil && release.ScheduledLocalTime == nil {
		return fmt.Errorf("scheduled timezone requires a scheduled local time")
	}
	if release.ScheduledInstant != nil {
		if err := validateUTC("scheduled instant", *release.ScheduledInstant); err != nil {
			return err
		}
	}
	if release.ScheduledEndInstant != nil {
		if err := validateUTC("scheduled end instant", *release.ScheduledEndInstant); err != nil {
			return err
		}
		if release.ScheduledInstant == nil || release.ScheduledEndInstant.Before(*release.ScheduledInstant) {
			return fmt.Errorf("scheduled end instant must not precede scheduled instant")
		}
	}
	if release.ActualReleaseInstant != nil {
		if err := validateUTC("actual release instant", *release.ActualReleaseInstant); err != nil {
			return err
		}
		if release.TimingAuthority != TimingActualReleaseTime {
			return fmt.Errorf("actual release instant requires actual-release timing authority")
		}
	}
	switch release.TimingAuthority {
	case TimingUnknown:
		if release.ScheduledLocalTime != nil || release.ScheduledInstant != nil {
			return fmt.Errorf("scheduled timing fields require an explicit timing authority")
		}
	case TimingDateOnlySchedule:
		if release.ScheduledDate == nil || release.ScheduledLocalTime != nil || release.ScheduledInstant != nil {
			return fmt.Errorf("date-only timing requires only a scheduled date")
		}
	case TimingScheduledLocalTime:
		if release.ScheduledDate == nil || release.ScheduledLocalTime == nil || release.ScheduledTimezone == nil || release.ScheduledInstant == nil {
			return fmt.Errorf("scheduled local timing requires date, local time, timezone, and derived instant")
		}
	case TimingScheduledInstant:
		if release.ScheduledInstant == nil {
			return fmt.Errorf("scheduled-instant timing requires a scheduled instant")
		}
	case TimingActualReleaseTime:
		if release.ActualReleaseInstant == nil {
			return fmt.Errorf("actual-release timing requires an authoritative actual instant")
		}
	default:
		return fmt.Errorf("unsupported timing authority %q", release.TimingAuthority)
	}
	if release.ScheduledLocalTime != nil && release.TimingAuthority != TimingScheduledLocalTime {
		return fmt.Errorf("scheduled local time requires scheduled-local timing authority")
	}
	if release.ScheduledInstant != nil && release.TimingAuthority != TimingScheduledLocalTime && release.TimingAuthority != TimingScheduledInstant && release.TimingAuthority != TimingActualReleaseTime {
		return fmt.Errorf("scheduled instant requires an explicit scheduled timing authority")
	}
	switch release.Status {
	case ReleaseStatusUnknown, ReleaseStatusScheduled, ReleaseStatusRescheduled, ReleaseStatusCancelled, ReleaseStatusReleased:
	default:
		return fmt.Errorf("unsupported release status %q", release.Status)
	}
	switch release.DataAvailability {
	case DataAvailabilityUnknown, DataAvailabilityIndicated, DataAvailabilityNoData:
	default:
		return fmt.Errorf("unsupported data availability %q", release.DataAvailability)
	}
	if release.RealtimePeriod != nil {
		if err := release.RealtimePeriod.Validate(); err != nil {
			return err
		}
	}
	if release.DataLastUpdatedDate != nil {
		if err := release.DataLastUpdatedDate.Validate(); err != nil {
			return fmt.Errorf("data last updated date: %w", err)
		}
	}
	if release.Revision != nil {
		if release.Revision.Sequence != nil && *release.Revision.Sequence < 0 {
			return fmt.Errorf("source revision sequence must not be negative")
		}
		if release.Revision.CalendarObjectStamp != nil {
			if err := validateUTC("calendar object stamp", *release.Revision.CalendarObjectStamp); err != nil {
				return err
			}
		}
		if release.Revision.LastModifiedAt != nil {
			if err := validateUTC("last modified at", *release.Revision.LastModifiedAt); err != nil {
				return err
			}
		}
	}
	if err := validateUTC("acquired at", release.AcquiredAt); err != nil {
		return err
	}
	if len(release.SourcePayloads) == 0 {
		return fmt.Errorf("at least one source payload is required")
	}
	seen := make(map[providercontract.RawPayloadID]struct{}, len(release.SourcePayloads))
	for _, payload := range release.SourcePayloads {
		if err := payload.Validate(); err != nil {
			return fmt.Errorf("source payload: %w", err)
		}
		if _, ok := seen[payload.ID]; ok {
			return fmt.Errorf("duplicate source payload %q", payload.ID)
		}
		seen[payload.ID] = struct{}{}
	}
	if err := release.Provenance.Validate(); err != nil {
		return fmt.Errorf("provenance: %w", err)
	}
	return nil
}

func validateUTC(field string, value time.Time) error {
	_, offset := value.Zone()
	if value.IsZero() || offset != 0 || value.Year() < 0 || value.Year() > 9999 {
		return fmt.Errorf("%s must be a non-zero UTC timestamp", field)
	}
	return nil
}
