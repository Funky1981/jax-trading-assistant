package fred

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

type seriesEnvelope struct {
	RealtimeStart string         `json:"realtime_start"`
	RealtimeEnd   string         `json:"realtime_end"`
	Series        []seriesRecord `json:"seriess"`
}

type seriesRecord struct {
	ID                 string `json:"id"`
	RealtimeStart      string `json:"realtime_start"`
	RealtimeEnd        string `json:"realtime_end"`
	Title              string `json:"title"`
	ObservationStart   string `json:"observation_start"`
	ObservationEnd     string `json:"observation_end"`
	Frequency          string `json:"frequency"`
	FrequencyCode      string `json:"frequency_short"`
	Units              string `json:"units"`
	UnitsShort         string `json:"units_short"`
	SeasonalAdjustment string `json:"seasonal_adjustment"`
	SeasonalCode       string `json:"seasonal_adjustment_short"`
	LastUpdated        string `json:"last_updated"`
	Notes              string `json:"notes"`
}

type vintageEnvelope struct {
	RealtimeStart string   `json:"realtime_start"`
	RealtimeEnd   string   `json:"realtime_end"`
	OrderBy       string   `json:"order_by"`
	SortOrder     string   `json:"sort_order"`
	Count         int      `json:"count"`
	Offset        int      `json:"offset"`
	Limit         int      `json:"limit"`
	Dates         []string `json:"vintage_dates"`
}

type observationsEnvelope struct {
	RealtimeStart    string           `json:"realtime_start"`
	RealtimeEnd      string           `json:"realtime_end"`
	ObservationStart string           `json:"observation_start"`
	ObservationEnd   string           `json:"observation_end"`
	Units            string           `json:"units"`
	OutputType       int              `json:"output_type"`
	Frequency        string           `json:"frequency"`
	Aggregation      string           `json:"aggregation_method"`
	OrderBy          string           `json:"order_by"`
	SortOrder        string           `json:"sort_order"`
	Count            int              `json:"count"`
	Offset           int              `json:"offset"`
	Limit            int              `json:"limit"`
	Observations     []observationRow `json:"observations"`
}

type observationRow struct {
	RealtimeStart string          `json:"realtime_start"`
	RealtimeEnd   string          `json:"realtime_end"`
	Date          string          `json:"date"`
	Value         json.RawMessage `json:"value"`
}

type parsedObservations struct {
	Page                   PageInfo
	ProviderRealtimePeriod *RealtimePeriod
	Observations           []MacroObservation
}

type parsedVintageDates struct {
	Page                   PageInfo
	ProviderRealtimePeriod *RealtimePeriod
	Dates                  []Date
}

type parsedObservation struct {
	Date           Date
	Value          MacroValue
	RealtimePeriod *RealtimePeriod
}

func decodeOneDocument(raw []byte, destination any) error {
	if len(raw) == 0 || !json.Valid(raw) {
		return fmt.Errorf("FRED response is not valid JSON")
	}
	if err := rejectDuplicateJSONProperties(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return fmt.Errorf("trailing JSON: %w", err)
	}
	return nil
}

func parseSeries(raw []byte, requestedSeriesID string, ref providercontract.RawPayloadRef) (MacroSeries, error) {
	var envelope seriesEnvelope
	if err := decodeOneDocument(raw, &envelope); err != nil {
		return MacroSeries{}, err
	}
	requestedSeriesID = strings.ToUpper(strings.TrimSpace(requestedSeriesID))
	if err := validateSeriesID(requestedSeriesID); err != nil {
		return MacroSeries{}, err
	}
	if len(envelope.Series) != 1 {
		return MacroSeries{}, fmt.Errorf("FRED series response must contain exactly one series")
	}
	item := envelope.Series[0]
	if strings.ToUpper(strings.TrimSpace(item.ID)) != requestedSeriesID {
		return MacroSeries{}, fmt.Errorf("FRED series ID does not match the requested series")
	}
	if strings.TrimSpace(item.Title) == "" {
		return MacroSeries{}, fmt.Errorf("FRED series title is required")
	}
	envelopePeriod, err := parseOptionalPeriod(envelope.RealtimeStart, envelope.RealtimeEnd)
	if err != nil {
		return MacroSeries{}, err
	}
	observationStart, err := parseDate(item.ObservationStart)
	if err != nil {
		return MacroSeries{}, fmt.Errorf("FRED observation start: %w", err)
	}
	observationEnd, err := parseDate(item.ObservationEnd)
	if err != nil {
		return MacroSeries{}, fmt.Errorf("FRED observation end: %w", err)
	}
	if observationEnd < observationStart {
		return MacroSeries{}, fmt.Errorf("FRED observation end precedes observation start")
	}
	period, err := parseOptionalPeriod(item.RealtimeStart, item.RealtimeEnd)
	if err != nil {
		return MacroSeries{}, err
	}
	if period == nil {
		period = envelopePeriod
	} else if envelopePeriod != nil && *period != *envelopePeriod {
		return MacroSeries{}, fmt.Errorf("FRED series realtime metadata disagrees with response metadata")
	}
	lastUpdated, err := parseOptionalLastUpdated(item.LastUpdated)
	if err != nil {
		return MacroSeries{}, err
	}
	seriesID := MacroSeriesID("mser_" + canonical.DigestBytes([]byte("fred-series\x00" + requestedSeriesID)).Value[:24])
	provenance, err := makeProvenance(ref, "fred-series\x00"+requestedSeriesID)
	if err != nil {
		return MacroSeries{}, err
	}
	series := MacroSeries{ID: seriesID, ProviderSeriesID: requestedSeriesID, Title: item.Title, ObservationStart: observationStart, ObservationEnd: observationEnd, Frequency: item.Frequency, FrequencyCode: item.FrequencyCode, Units: item.Units, UnitsShort: item.UnitsShort, SeasonalAdjustment: item.SeasonalAdjustment, SeasonalAdjustmentCode: item.SeasonalCode, LastUpdated: lastUpdated, Notes: item.Notes, ProviderRealtimePeriod: period, SourcePayload: ref, Provenance: provenance}
	if err := series.Validate(); err != nil {
		return MacroSeries{}, err
	}
	return series, nil
}

func parseVintageDates(raw []byte, requestedSeriesID string) (parsedVintageDates, error) {
	var envelope vintageEnvelope
	if err := decodeOneDocument(raw, &envelope); err != nil {
		return parsedVintageDates{}, err
	}
	if err := validateSeriesID(strings.ToUpper(strings.TrimSpace(requestedSeriesID))); err != nil {
		return parsedVintageDates{}, err
	}
	page := PageInfo{Count: envelope.Count, Offset: envelope.Offset, Limit: envelope.Limit}
	if err := page.Validate(10000); err != nil {
		return parsedVintageDates{}, err
	}
	period, err := parseOptionalPeriod(envelope.RealtimeStart, envelope.RealtimeEnd)
	if err != nil {
		return parsedVintageDates{}, err
	}
	if envelope.SortOrder != "asc" || envelope.OrderBy != "vintage_date" {
		return parsedVintageDates{}, fmt.Errorf("FRED vintage response ordering is not the requested ascending vintage date order")
	}
	if len(envelope.Dates) > envelope.Limit {
		return parsedVintageDates{}, fmt.Errorf("FRED vintage response exceeds its declared limit")
	}
	dates := make([]Date, 0, len(envelope.Dates))
	var previous Date
	for _, rawDate := range envelope.Dates {
		date, err := parseDate(rawDate)
		if err != nil {
			return parsedVintageDates{}, fmt.Errorf("FRED vintage date: %w", err)
		}
		if previous != "" && date <= previous {
			return parsedVintageDates{}, fmt.Errorf("FRED vintage dates are not strictly ascending")
		}
		previous = date
		dates = append(dates, date)
	}
	return parsedVintageDates{Page: page, ProviderRealtimePeriod: period, Dates: dates}, nil
}

func parseObservationEnvelope(raw []byte, request ObservationsRequest) (PageInfo, []parsedObservation, error) {
	var envelope observationsEnvelope
	if err := decodeOneDocument(raw, &envelope); err != nil {
		return PageInfo{}, nil, err
	}
	page := PageInfo{Count: envelope.Count, Offset: envelope.Offset, Limit: envelope.Limit}
	if err := page.Validate(100000); err != nil {
		return PageInfo{}, nil, err
	}
	envelopePeriod, err := parseOptionalPeriod(envelope.RealtimeStart, envelope.RealtimeEnd)
	if err != nil {
		return PageInfo{}, nil, err
	}
	if envelope.Units != "lin" || envelope.OutputType != 1 && request.InformationState.Mode != InformationStateInitialRelease || envelope.Frequency != "" || envelope.Aggregation != "" || envelope.SortOrder != "asc" || envelope.OrderBy != "observation_date" {
		return PageInfo{}, nil, fmt.Errorf("FRED response does not describe native ascending observations")
	}
	if request.InformationState.Mode == InformationStateInitialRelease && envelope.OutputType != 4 {
		return PageInfo{}, nil, fmt.Errorf("FRED initial-release response has the wrong output type")
	}
	if len(envelope.Observations) > envelope.Limit {
		return PageInfo{}, nil, fmt.Errorf("FRED observation response exceeds its declared limit")
	}
	result := make([]parsedObservation, 0, len(envelope.Observations))
	var previousKey string
	for _, row := range envelope.Observations {
		date, err := parseDate(row.Date)
		if err != nil {
			return PageInfo{}, nil, fmt.Errorf("FRED observation date: %w", err)
		}
		if date < request.ObservationStart || date > request.ObservationEnd {
			return PageInfo{}, nil, fmt.Errorf("FRED observation is outside the requested bounded date range")
		}
		value, err := parseValue(row.Value)
		if err != nil {
			return PageInfo{}, nil, err
		}
		period, err := parseOptionalPeriod(row.RealtimeStart, row.RealtimeEnd)
		if err != nil {
			return PageInfo{}, nil, err
		}
		if period == nil {
			period = envelopePeriod
		}
		key := string(date) + "\x00"
		if period != nil {
			key += string(period.Start) + "\x00" + string(period.End)
		}
		if previousKey != "" && key <= previousKey {
			return PageInfo{}, nil, fmt.Errorf("FRED observations are not strictly ordered or contain duplicates")
		}
		previousKey = key
		result = append(result, parsedObservation{Date: date, Value: value, RealtimePeriod: period})
	}
	return page, result, nil
}

func normalizeSeries(raw []byte, ref providercontract.RawPayloadRef, requestedSeriesID string) (MacroSeries, error) {
	return parseSeries(raw, requestedSeriesID, ref)
}

func normalizeObservations(raw []byte, ref providercontract.RawPayloadRef, request ObservationsRequest) (parsedObservations, error) {
	page, rows, err := parseObservationEnvelope(raw, request)
	if err != nil {
		return parsedObservations{}, err
	}
	var providerPeriod *RealtimePeriod
	if len(rows) > 0 {
		providerPeriod = rows[0].RealtimePeriod
	}
	result := parsedObservations{Page: page, ProviderRealtimePeriod: providerPeriod, Observations: make([]MacroObservation, 0, len(rows))}
	for _, row := range rows {
		observation, err := buildObservation(row, request.Series, request.InformationState, ref)
		if err != nil {
			return parsedObservations{}, err
		}
		result.Observations = append(result.Observations, observation)
	}
	return result, nil
}

func buildObservation(parsed parsedObservation, series MacroSeries, state InformationState, ref providercontract.RawPayloadRef) (MacroObservation, error) {
	periodValue := ""
	if parsed.RealtimePeriod != nil {
		periodValue = string(parsed.RealtimePeriod.Start) + "\x00" + string(parsed.RealtimePeriod.End)
	}
	observationID := "mobs_" + canonical.DigestBytes([]byte(strings.Join([]string{"fred-observation", series.ProviderSeriesID, string(parsed.Date), periodValue, parsed.Value.SourceValue, string(state.Mode), dateValue(state.Date)}, "\x00"))).Value[:24]
	provenance, err := makeProvenanceForRefs([]providercontract.RawPayloadRef{series.SourcePayload, ref}, "fred-observation\x00"+observationID)
	if err != nil {
		return MacroObservation{}, err
	}
	observation := MacroObservation{ID: observationID, Series: series.ID, ProviderSeriesID: series.ProviderSeriesID, ObservationDate: parsed.Date, Value: parsed.Value, RealtimePeriod: parsed.RealtimePeriod, RequestedInformation: state, AcquiredAt: ref.ReceivedAt, SourcePayload: ref, Provenance: provenance}
	if err := observation.Validate(); err != nil {
		return MacroObservation{}, err
	}
	return observation, nil
}

func dateValue(value *Date) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func parseValue(raw json.RawMessage) (MacroValue, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return MacroValue{}, fmt.Errorf("FRED observation value must be a string")
	}
	value = strings.TrimSpace(value)
	if value == "." {
		return MacroValue{Present: false, SourceValue: value}, nil
	}
	if value == "" {
		return MacroValue{}, fmt.Errorf("FRED observation value is empty")
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return MacroValue{}, fmt.Errorf("FRED observation value is not a finite numeric value")
	}
	return MacroValue{Present: true, SourceValue: value, Number: &number}, nil
}

func parseOptionalPeriod(startRaw, endRaw string) (*RealtimePeriod, error) {
	startRaw, endRaw = strings.TrimSpace(startRaw), strings.TrimSpace(endRaw)
	if startRaw == "" && endRaw == "" {
		return nil, nil
	}
	if startRaw == "" || endRaw == "" {
		return nil, fmt.Errorf("FRED realtime period must provide both start and end")
	}
	start, err := parseDate(startRaw)
	if err != nil {
		return nil, fmt.Errorf("FRED realtime start: %w", err)
	}
	end, err := parseDate(endRaw)
	if err != nil {
		return nil, fmt.Errorf("FRED realtime end: %w", err)
	}
	period := &RealtimePeriod{Start: start, End: end}
	if err := period.Validate(); err != nil {
		return nil, err
	}
	return period, nil
}

func parseOptionalLastUpdated(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05-07", "2006-01-02 15:04:05-07:00", time.RFC3339Nano} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			value := parsed.UTC()
			return &value, nil
		}
	}
	return nil, fmt.Errorf("FRED last_updated timestamp is malformed")
}

func makeProvenance(ref providercontract.RawPayloadRef, salt string) (canonical.Provenance, error) {
	return makeProvenanceForRefs([]providercontract.RawPayloadRef{ref}, salt)
}

func makeProvenanceForRefs(refs []providercontract.RawPayloadRef, salt string) (canonical.Provenance, error) {
	if len(refs) == 0 {
		return canonical.Provenance{}, fmt.Errorf("FRED provenance requires at least one raw payload")
	}
	inputs := make([]canonical.LineageInput, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if _, exists := seen[string(ref.ID)]; exists {
			continue
		}
		seen[string(ref.ID)] = struct{}{}
		evidenceID := canonical.EvidenceID("evd_" + canonical.DigestBytes([]byte("fred-evidence\x00" + string(ref.ID))).Value[:24])
		evidenceRef, err := ref.AsEvidenceRef(canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(evidenceID), ContractVersion: canonical.EvidenceContractV2})
		if err != nil {
			return canonical.Provenance{}, err
		}
		inputs = append(inputs, canonical.LineageInput{Kind: canonical.LineageInputKindEvidence, Evidence: &evidenceRef})
	}
	fingerprint, err := canonical.ComputeInputFingerprint(inputs)
	if err != nil {
		return canonical.Provenance{}, err
	}
	provider := ProviderIdentity
	return canonical.Provenance{ContractVersion: canonical.ProvenanceContractV1, ID: "pvn_" + canonical.DigestBytes([]byte("fred-provenance\x00" + salt)).Value[:24], Inputs: inputs, InputFingerprint: fingerprint, Producer: canonical.ComponentIdentity{ID: NormalizerID, Kind: canonical.ComponentKindNormalizer, Name: "FRED macro deterministic normalizer", Version: canonical.VersionIdentity{Namespace: "jax.fred.normalizer", Value: NormalizerVersion}, Provider: &provider}}, nil
}

func rejectDuplicateJSONProperties(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]struct{}{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key := keyToken.(string)
				if _, exists := seen[key]; exists {
					return fmt.Errorf("FRED response contains duplicate JSON property")
				}
				seen[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("FRED response contains an unexpected JSON delimiter")
		}
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("FRED response contains trailing data")
	}
	return nil
}
