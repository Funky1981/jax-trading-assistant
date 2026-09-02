// Package macroevidence contains provider-neutral macro series and
// observation contracts. Provider adapters project into these types without
// importing one another's response DTOs.
package macroevidence

import (
	"fmt"
	"math"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

type Date string

func (date Date) Validate() error {
	value := string(date)
	if len(value) != len("2006-01-02") {
		return fmt.Errorf("macro date must use YYYY-MM-DD")
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return fmt.Errorf("macro date %q is invalid", value)
	}
	return nil
}

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

type MacroSeriesID string

type MacroSeries struct {
	ID                     MacroSeriesID                  `json:"id"`
	ProviderSeriesID       string                         `json:"provider_series_id"`
	Title                  string                         `json:"title"`
	ObservationStart       Date                           `json:"observation_start"`
	ObservationEnd         Date                           `json:"observation_end"`
	Frequency              string                         `json:"frequency,omitempty"`
	FrequencyCode          string                         `json:"frequency_code,omitempty"`
	Units                  string                         `json:"units,omitempty"`
	UnitsShort             string                         `json:"units_short,omitempty"`
	SeasonalAdjustment     string                         `json:"seasonal_adjustment,omitempty"`
	SeasonalAdjustmentCode string                         `json:"seasonal_adjustment_code,omitempty"`
	LastUpdated            *time.Time                     `json:"last_updated,omitempty"`
	Notes                  string                         `json:"notes,omitempty"`
	RequestedInformation   InformationState               `json:"requested_information"`
	ProviderRealtimePeriod *RealtimePeriod                `json:"provider_realtime_period,omitempty"`
	SourcePayload          providercontract.RawPayloadRef `json:"source_payload"`
	Provenance             canonical.Provenance           `json:"provenance"`
}

func (series MacroSeries) Validate() error {
	if !strings.HasPrefix(string(series.ID), "mser_") {
		return fmt.Errorf("macro series ID must use mser_ prefix")
	}
	if strings.TrimSpace(series.ProviderSeriesID) == "" || len(series.ProviderSeriesID) > 128 {
		return fmt.Errorf("provider series identity is required")
	}
	if strings.TrimSpace(series.Title) == "" {
		return fmt.Errorf("macro series title is required")
	}
	if err := series.ObservationStart.Validate(); err != nil {
		return fmt.Errorf("observation start: %w", err)
	}
	if err := series.ObservationEnd.Validate(); err != nil {
		return fmt.Errorf("observation end: %w", err)
	}
	if series.ObservationEnd < series.ObservationStart {
		return fmt.Errorf("observation end precedes observation start")
	}
	if err := series.RequestedInformation.Validate(); err != nil {
		return fmt.Errorf("series information state: %w", err)
	}
	if series.ProviderRealtimePeriod != nil {
		if err := series.ProviderRealtimePeriod.Validate(); err != nil {
			return err
		}
	}
	if err := series.SourcePayload.Validate(); err != nil {
		return fmt.Errorf("series source payload: %w", err)
	}
	if err := series.Provenance.Validate(); err != nil {
		return fmt.Errorf("series provenance: %w", err)
	}
	return nil
}

type InformationStateMode string

const (
	InformationStateCurrent        InformationStateMode = "CURRENT"
	InformationStateAsOf           InformationStateMode = "AS_OF_REALTIME_DATE"
	InformationStateVintage        InformationStateMode = "VINTAGE_DATE"
	InformationStateInitialRelease InformationStateMode = "INITIAL_RELEASE"
)

type InformationState struct {
	Mode InformationStateMode `json:"mode"`
	Date *Date                `json:"date,omitempty"`
}

func (state InformationState) Validate() error {
	switch state.Mode {
	case InformationStateCurrent, InformationStateInitialRelease:
		if state.Date != nil {
			return fmt.Errorf("this information state mode must not carry a date")
		}
	case InformationStateAsOf, InformationStateVintage:
		if state.Date == nil {
			return fmt.Errorf("historical information state requires an explicit date")
		}
		if err := state.Date.Validate(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported information state mode")
	}
	return nil
}

type MacroValue struct {
	Present     bool     `json:"present"`
	SourceValue string   `json:"source_value"`
	Number      *float64 `json:"number,omitempty"`
}

func (value MacroValue) Validate() error {
	if strings.TrimSpace(value.SourceValue) == "" {
		return fmt.Errorf("source value is required")
	}
	if !value.Present {
		if value.SourceValue != "." || value.Number != nil {
			return fmt.Errorf("missing macro value must preserve its source marker")
		}
		return nil
	}
	if value.SourceValue == "." || value.Number == nil {
		return fmt.Errorf("present macro value requires a normalized number")
	}
	if math.IsNaN(*value.Number) || math.IsInf(*value.Number, 0) {
		return fmt.Errorf("present macro value requires a finite normalized number")
	}
	return nil
}

type MacroObservation struct {
	ID                   string                         `json:"id"`
	Series               MacroSeriesID                  `json:"series"`
	ProviderSeriesID     string                         `json:"provider_series_id"`
	ObservationDate      Date                           `json:"observation_date"`
	Value                MacroValue                     `json:"value"`
	RealtimePeriod       *RealtimePeriod                `json:"realtime_period,omitempty"`
	RequestedInformation InformationState               `json:"requested_information"`
	AcquiredAt           time.Time                      `json:"acquired_at"`
	SourcePayload        providercontract.RawPayloadRef `json:"source_payload"`
	Provenance           canonical.Provenance           `json:"provenance"`
}

func (observation MacroObservation) Validate() error {
	if !strings.HasPrefix(observation.ID, "mobs_") {
		return fmt.Errorf("macro observation ID must use mobs_ prefix")
	}
	if !strings.HasPrefix(string(observation.Series), "mser_") {
		return fmt.Errorf("macro observation series reference is required")
	}
	if strings.TrimSpace(observation.ProviderSeriesID) == "" || len(observation.ProviderSeriesID) > 128 {
		return fmt.Errorf("provider series identity is required")
	}
	if err := observation.ObservationDate.Validate(); err != nil {
		return fmt.Errorf("observation date: %w", err)
	}
	if err := observation.Value.Validate(); err != nil {
		return err
	}
	if observation.RealtimePeriod != nil {
		if err := observation.RealtimePeriod.Validate(); err != nil {
			return err
		}
	}
	if err := observation.RequestedInformation.Validate(); err != nil {
		return err
	}
	if err := validateUTC("acquired_at", observation.AcquiredAt); err != nil {
		return err
	}
	if err := observation.SourcePayload.Validate(); err != nil {
		return fmt.Errorf("observation source payload: %w", err)
	}
	if err := observation.Provenance.Validate(); err != nil {
		return fmt.Errorf("observation provenance: %w", err)
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

type CompletenessState string

const (
	CompletenessComplete   CompletenessState = "COMPLETE"
	CompletenessIncomplete CompletenessState = "INCOMPLETE"
)

type PageInfo struct {
	Count  int `json:"count"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

func (page PageInfo) Validate(maxLimit int) error {
	if page.Count < 0 || page.Offset < 0 || page.Offset > page.Count || page.Limit < 1 || page.Limit > maxLimit {
		return fmt.Errorf("macro pagination metadata is invalid")
	}
	return nil
}
