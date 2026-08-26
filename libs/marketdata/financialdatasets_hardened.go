package marketdata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

const (
	FinancialDatasetsProviderID             = "pvd_financial_datasets"
	FinancialDatasetsHistoricalSourceID     = "src_financial_datasets_historical_prices"
	FinancialDatasetsDailyBarsNormalizerID  = "cmp_financial_datasets_daily_bars"
	financialDatasetsMaximumResponseBytes   = 4 << 20
	financialDatasetsMaximumRequestCalendar = 366
)

var (
	FinancialDatasetsProviderIdentity = canonical.ProviderIdentity{
		ID: FinancialDatasetsProviderID, Namespace: "financialdatasets.api",
		ExternalID: &canonical.ExternalID{Namespace: "provider.slug", Value: "financial-datasets"},
	}
	FinancialDatasetsHistoricalSource = canonical.SourceIdentity{
		ID: FinancialDatasetsHistoricalSourceID, Kind: canonical.SourceKindDataset,
	}
	FinancialDatasetsHistoricalRawSchema = canonical.VersionIdentity{
		Namespace: "financialdatasets.prices.historical", Value: "documented-2026-08-26/v1",
	}
)

type MarketAdjustmentState string

const (
	MarketAdjustmentUnknown    MarketAdjustmentState = "UNKNOWN"
	MarketAdjustmentAdjusted   MarketAdjustmentState = "ADJUSTED"
	MarketAdjustmentUnadjusted MarketAdjustmentState = "UNADJUSTED"
)

type MarketSessionState string

const (
	MarketSessionProviderEODUnspecified MarketSessionState = "PROVIDER_EOD_SESSION_UNSPECIFIED"
)

type MarketTimestampSemantics string

const (
	MarketTimestampProviderDateIntervalEnd MarketTimestampSemantics = "PROVIDER_SESSION_DATE_INTERVAL_END"
)

type MarketTimestampAuthority string

const (
	MarketTimestampAuthorityIntervalBoundary MarketTimestampAuthority = "INTERVAL_BOUNDARY_ONLY"
)

const (
	MarketMetricOpen   = "market_bar_open"
	MarketMetricHigh   = "market_bar_high"
	MarketMetricLow    = "market_bar_low"
	MarketMetricClose  = "market_bar_close"
	MarketMetricVolume = "market_bar_volume"
)

// CanonicalMarketBar is a provider-neutral research-facing projection over
// five validated Observation V2 records. It remains data-platform owned and
// carries no vendor DTO, endpoint, credential, or provider name.
type CanonicalMarketBar struct {
	Instrument         canonical.ContractRef
	Interval           Timeframe
	Start              time.Time
	End                time.Time
	ProviderDate       string
	TimestampSemantics MarketTimestampSemantics
	TimestampAuthority MarketTimestampAuthority
	Session            MarketSessionState
	MarketTimezone     string
	Adjustment         MarketAdjustmentState
	Open               canonical.Observation
	High               canonical.Observation
	Low                canonical.Observation
	Close              canonical.Observation
	Volume             canonical.Observation
}

func (bar CanonicalMarketBar) Validate() error {
	if bar.Instrument.Kind != canonical.ContractKindInstrument || bar.Instrument.ID == "" || bar.Instrument.ContractVersion == "" {
		return errors.New("market bar requires a canonical instrument reference")
	}
	if bar.Interval != Timeframe1Day || bar.Start.IsZero() || bar.End.IsZero() || !bar.End.After(bar.Start) || bar.Start.Location() != time.UTC || bar.End.Location() != time.UTC {
		return errors.New("market bar has invalid daily UTC interval")
	}
	if bar.TimestampSemantics != MarketTimestampProviderDateIntervalEnd || bar.TimestampAuthority != MarketTimestampAuthorityIntervalBoundary || bar.Session != MarketSessionProviderEODUnspecified || bar.MarketTimezone == "" {
		return errors.New("market bar timestamp/session semantics are incomplete")
	}
	observations := []canonical.Observation{bar.Open, bar.High, bar.Low, bar.Close, bar.Volume}
	for _, observation := range observations {
		if err := observation.Validate(); err != nil {
			return err
		}
		if observation.Subject != bar.Instrument || observation.ObservedAt != bar.End {
			return errors.New("market bar observation identity or timestamp mismatch")
		}
		if observation.Value.Number == nil || math.IsNaN(*observation.Value.Number) || math.IsInf(*observation.Value.Number, 0) {
			return errors.New("market bar observation value is not finite")
		}
	}
	open, high, low, closeValue := *bar.Open.Value.Number, *bar.High.Value.Number, *bar.Low.Value.Number, *bar.Close.Value.Number
	if high < low || high < open || high < closeValue || low > open || low > closeValue {
		return errors.New("market bar OHLC invariants are violated")
	}
	if *bar.Volume.Value.Number < 0 {
		return errors.New("market bar volume is negative")
	}
	return nil
}

// ExchangeCloseTime is intentionally unavailable for provider calendar-date
// bars. Callers must not reinterpret the neutral UTC interval boundary as an
// exchange close or market event timestamp.
func (bar CanonicalMarketBar) ExchangeCloseTime() (time.Time, error) {
	if bar.TimestampAuthority != MarketTimestampAuthorityIntervalBoundary {
		return time.Time{}, errors.New("market bar does not declare a supported exchange-close timestamp")
	}
	return time.Time{}, errors.New("provider calendar boundary is not an authoritative exchange-close timestamp")
}

type FinancialDatasetsBarsRequest struct {
	Instrument canonical.Instrument
	StartDate  time.Time
	EndDate    time.Time
	Interval   Timeframe
	PayloadID  providercontract.RawPayloadID
	Retention  providercontract.RawPayloadRetentionPolicy
}

type FinancialDatasetsBarsDependencies struct {
	Registry *providercontract.Registry
	Executor *providercontract.OperationalExecutor
	Store    providercontract.RawPayloadStore
	Pipeline *providercontract.NormalizationPipeline
}

type FinancialDatasetsBarsResult struct {
	Execution     providercontract.ExecutionResult
	Raw           providercontract.RawPayloadDescriptor
	Normalization providercontract.BatchNormalizationResult
	Bars          []CanonicalMarketBar
}

// FinancialDatasetsInstrumentResolver binds a provider ticker to a validated
// canonical Instrument. Ticker strings never become canonical identities.
type FinancialDatasetsInstrumentResolver interface {
	ResolveFinancialDatasetsTicker(string) (canonical.Instrument, error)
}

type FinancialDatasetsDailyBarsNormalizer struct {
	descriptor providercontract.NormalizerDescriptor
	resolver   FinancialDatasetsInstrumentResolver
}

func FinancialDatasetsProviderDefinition() providercontract.ProviderDefinition {
	identity := cloneMarketProviderIdentity(FinancialDatasetsProviderIdentity)
	return providercontract.ProviderDefinition{
		ContractVersion: providercontract.ProviderDefinitionV1,
		Identity:        identity,
		DisplayName:     "Financial Datasets",
		AdapterVersion:  canonical.VersionIdentity{Namespace: "jax.marketdata.financial_datasets", Value: "1.0.0"},
		Capabilities: []providercontract.Capability{{
			ContractVersion: providercontract.CapabilityContractV1,
			ID:              providercontract.CapabilityMarketBars,
			Category:        providercontract.DataCategoryMarketData,
			Support:         providercontract.SupportSupported,
			Raw: providercontract.RawRepresentation{
				Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument,
				Schema: FinancialDatasetsHistoricalRawSchema, MediaType: "application/json",
			},
			Authentication: providercontract.AuthenticationRequirement{Class: providercontract.AuthenticationAPIKey},
			Operational: providercontract.OperationalSemantics{
				DeliveryModes:      []providercontract.DeliveryMode{providercontract.DeliveryHistorical},
				FreshnessModes:     []providercontract.FreshnessMode{providercontract.FreshnessEndOfDay},
				QualityRequirement: providercontract.QualityCanonicalValidationRequired,
			},
			CanonicalOutputs: []canonical.ContractSchemaRef{{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2}},
		}},
	}
}

func NewFinancialDatasetsDailyBarsNormalizer(resolver FinancialDatasetsInstrumentResolver) (*FinancialDatasetsDailyBarsNormalizer, error) {
	if resolver == nil {
		return nil, errors.New("financial-datasets daily-bars normalizer requires an instrument resolver")
	}
	providerIdentity := cloneMarketProviderIdentity(FinancialDatasetsProviderIdentity)
	mappingContent := canonical.RawContentIdentity([]byte("jax financial-datasets daily EOD OHLCV mapping/v1"))
	descriptor := providercontract.NormalizerDescriptor{
		ContractVersion: providercontract.NormalizerDescriptorV1,
		Provider:        providerIdentity,
		CapabilityID:    providercontract.CapabilityMarketBars,
		Raw: providercontract.RawRepresentation{
			Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument,
			Schema: FinancialDatasetsHistoricalRawSchema, MediaType: "application/json",
		},
		Component: canonical.ComponentIdentity{
			ID: FinancialDatasetsDailyBarsNormalizerID, Kind: canonical.ComponentKindNormalizer,
			Name:     "Financial Datasets documented daily EOD OHLCV normalizer",
			Version:  canonical.VersionIdentity{Namespace: "jax.normalizer.market_bars", Value: "1.0.0"},
			Provider: &providerIdentity, Content: &mappingContent,
		},
		Target: canonical.ContractSchemaRef{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2},
	}
	if err := descriptor.Validate(); err != nil {
		return nil, err
	}
	return &FinancialDatasetsDailyBarsNormalizer{descriptor: descriptor, resolver: resolver}, nil
}

func (normalizer *FinancialDatasetsDailyBarsNormalizer) Descriptor() providercontract.NormalizerDescriptor {
	return normalizer.descriptor
}

func (normalizer *FinancialDatasetsDailyBarsNormalizer) Normalize(context.Context, providercontract.NormalizationInput) (providercontract.NormalizationCandidate, error) {
	return providercontract.NormalizationCandidate{}, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorAmbiguousProviderValue, "historical prices response contains a canonical observation batch", nil)
}

func (normalizer *FinancialDatasetsDailyBarsNormalizer) NormalizeBatch(_ context.Context, input providercontract.NormalizationInput) ([]providercontract.NormalizationCandidate, error) {
	payload, err := parseFinancialDatasetsHistoricalPayload(input.Bytes)
	if err != nil {
		return nil, err
	}
	if input.RawRef.Source == nil || *input.RawRef.Source != FinancialDatasetsHistoricalSource || input.RawRef.Revision == nil {
		return nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorIdentityResolution, "raw source and immutable revision are required", nil)
	}
	ticker, rows, err := validateFinancialDatasetsHistoricalPayload(payload, input.RawRef.ReceivedAt)
	if err != nil {
		return nil, err
	}
	instrument, err := normalizer.resolver.ResolveFinancialDatasetsTicker(ticker)
	if err != nil {
		return nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorIdentityResolution, "provider ticker is not bound to a canonical instrument", err)
	}
	if err := instrument.Validate(); err != nil {
		return nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorIdentityResolution, "resolved canonical instrument is invalid", err)
	}
	resolvedTicker, err := financialDatasetsTicker(instrument)
	if err != nil || resolvedTicker != ticker {
		return nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorIdentityResolution, "resolved canonical instrument does not match provider ticker", err)
	}

	candidates := make([]providercontract.NormalizationCandidate, 0, len(rows)*5)
	for _, row := range rows {
		barCandidates, err := normalizer.mapRow(input.RawRef, instrument, row)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, barCandidates...)
	}
	return candidates, nil
}

func (normalizer *FinancialDatasetsDailyBarsNormalizer) mapRow(ref providercontract.RawPayloadRef, instrument canonical.Instrument, row validatedFinancialDatasetsRow) ([]providercontract.NormalizationCandidate, error) {
	values := []struct {
		metric string
		typeID canonical.ObservationType
		value  float64
		unit   string
		field  string
	}{
		{MarketMetricOpen, canonical.ObservationTypePrice, row.Open, instrument.Currency, "open"},
		{MarketMetricHigh, canonical.ObservationTypePrice, row.High, instrument.Currency, "high"},
		{MarketMetricLow, canonical.ObservationTypePrice, row.Low, instrument.Currency, "low"},
		{MarketMetricClose, canonical.ObservationTypePrice, row.Close, instrument.Currency, "close"},
		{MarketMetricVolume, canonical.ObservationTypeVolume, float64(row.Volume), "shares", "volume"},
	}
	if instrument.Currency == "" {
		return nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorUnmappableProviderValue, "instrument currency is required for price observations", nil)
	}

	result := make([]providercontract.NormalizationCandidate, 0, len(values))
	for _, item := range values {
		seed := strings.Join([]string{string(ref.ID), ref.Content.Digest.Value, normalizer.descriptor.Component.ID, normalizer.descriptor.Component.Version.Value, string(instrument.ID), row.ProviderDate, item.metric}, "\x00")
		digest := canonical.DigestBytes([]byte(seed)).Value
		observationID := canonical.ObservationID("obs_" + digest[:24])
		evidenceID := canonical.EvidenceID("evd_" + digest[24:48])
		evidenceRef, err := ref.AsEvidenceRef(canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(evidenceID), ContractVersion: canonical.EvidenceContractV2})
		if err != nil {
			return nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorProvenanceValidation, "raw evidence reference could not be constructed", err)
		}
		evidenceRef.ObservedAt = &row.End
		lineage := canonical.LineageInput{Kind: canonical.LineageInputKindEvidence, Evidence: &evidenceRef}
		fingerprint, err := canonical.ComputeInputFingerprint([]canonical.LineageInput{lineage})
		if err != nil {
			return nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorProvenanceValidation, "raw lineage fingerprint could not be constructed", err)
		}
		value := item.value
		observation := canonical.Observation{
			ContractVersion: canonical.ObservationContractV2,
			ID:              observationID, Type: item.typeID,
			Subject: canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: string(instrument.ID), ContractVersion: instrument.ContractVersion},
			Metric:  item.metric,
			Value:   canonical.ObservedValue{Type: canonical.ObservedValueTypeNumber, Number: &value, Unit: item.unit},
			Source: canonical.SourceReference{
				ID: FinancialDatasetsHistoricalSourceID, Kind: canonical.SourceKindDataset,
				ExternalID: &canonical.ExternalID{Namespace: "financialdatasets.dataset", Value: "historical-prices"},
				URI:        "https://docs.financialdatasets.ai/api/prices/historical",
			},
			EvidenceIDs: []canonical.EvidenceID{evidenceID}, ObservedAt: row.End,
			CollectedAt: ref.ReceivedAt, CreatedAt: ref.ReceivedAt,
			Provenance: &canonical.Provenance{
				ContractVersion: canonical.ProvenanceContractV1, ID: "pvn_" + digest[:24],
				Inputs: []canonical.LineageInput{lineage}, InputFingerprint: fingerprint,
				Producer: normalizer.descriptor.Component,
			},
		}
		result = append(result, providercontract.NormalizationCandidate{
			Record:   observation,
			Revision: canonical.RevisionIdentity{Namespace: "jax.normalized.market_bar_observation", Value: "v1/" + digest},
			Dispositions: []providercontract.FieldDisposition{
				{ProviderField: "ticker", Status: providercontract.FieldDispositionRepresented, CanonicalField: "subject"},
				{ProviderField: "time", Status: providercontract.FieldDispositionRepresented, CanonicalField: "observed_at"},
				{ProviderField: item.field, Status: providercontract.FieldDispositionRepresented, CanonicalField: "value.number"},
				{ProviderField: "interval", Status: providercontract.FieldDispositionRepresented, CanonicalField: "metric"},
			},
		})
	}
	return result, nil
}

// AcquireAndNormalizeDailyBars drives the existing Financial Datasets HTTP
// client through the accepted Phase 02 operational, raw-persistence, and
// normalization boundaries. Freshness and qualification remain separate
// caller-owned evaluations over the returned accepted records.
func (p *FinancialDatasetsProvider) AcquireAndNormalizeDailyBars(ctx context.Context, dependencies FinancialDatasetsBarsDependencies, request FinancialDatasetsBarsRequest) (FinancialDatasetsBarsResult, error) {
	if err := validateFinancialDatasetsBarsRequest(request); err != nil {
		return FinancialDatasetsBarsResult{}, err
	}
	if p == nil || dependencies.Registry == nil || dependencies.Executor == nil || dependencies.Store == nil || dependencies.Pipeline == nil {
		return FinancialDatasetsBarsResult{}, errors.New("financial-datasets hardened bars path is not fully configured")
	}
	ticker, err := financialDatasetsTicker(request.Instrument)
	if err != nil {
		return FinancialDatasetsBarsResult{}, err
	}
	operation := providercontract.Operation{
		ContractVersion: providercontract.OperationContractV1,
		Provider:        cloneMarketProviderIdentity(FinancialDatasetsProviderIdentity), CapabilityID: providercontract.CapabilityMarketBars,
		Kind: providercontract.OperationReadFetch, RetrySafety: providercontract.RetrySafetyRepeatable,
	}
	var mediaType string
	execution, err := dependencies.Executor.Execute(ctx, operation, func(attemptCtx context.Context, _ providercontract.AttemptContext) providercontract.ProviderAttemptResult {
		result, contentType := p.fetchHistoricalBarsAttempt(attemptCtx, ticker, request.StartDate, request.EndDate)
		if result.Failure == nil {
			mediaType = contentType
		}
		return result
	})
	if err != nil {
		return FinancialDatasetsBarsResult{Execution: execution}, err
	}
	revision := canonical.RevisionIdentity{Namespace: "financialdatasets.prices.response_sha256", Value: canonical.DigestBytes(execution.RawBytes).Value}
	definition := FinancialDatasetsProviderDefinition()
	raw, err := providercontract.PersistRawPayload(ctx, dependencies.Registry, dependencies.Store, providercontract.RawPayloadPersistenceRequest{
		ID: request.PayloadID, Provider: definition.Identity, Capability: providercontract.CapabilityMarketBars,
		Raw: providercontract.RawRepresentation{
			Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument,
			Schema: FinancialDatasetsHistoricalRawSchema, MediaType: mediaType,
		},
		Capture: providercontract.RawPayloadCapture{
			ByteForm: providercontract.RawPayloadByteFormEntityBody, ContentCodingState: providercontract.ContentCodingIdentity,
			CharacterEncoding: "utf-8",
		},
		Source: &FinancialDatasetsHistoricalSource, Revision: &revision, ReceivedAt: execution.CompletedAt,
		Retention: request.Retention, Complete: true,
	}, execution.RawBytes)
	if err != nil {
		return FinancialDatasetsBarsResult{Execution: execution}, err
	}
	normalizer, err := NewFinancialDatasetsDailyBarsNormalizer(singleInstrumentResolver{instrument: request.Instrument})
	if err != nil {
		return FinancialDatasetsBarsResult{Execution: execution, Raw: raw}, err
	}
	normalized, err := dependencies.Pipeline.NormalizeBatchStored(ctx, dependencies.Store, providercontract.StoredNormalizationRequest{
		RawRef: raw.Ref, Target: normalizer.descriptor.Target, Normalizer: normalizer.descriptor.Component,
	})
	if err != nil {
		return FinancialDatasetsBarsResult{Execution: execution, Raw: raw}, err
	}
	bars, err := ProjectCanonicalMarketBars(normalized, request.Instrument)
	if err != nil {
		return FinancialDatasetsBarsResult{Execution: execution, Raw: raw, Normalization: normalized}, err
	}
	return FinancialDatasetsBarsResult{Execution: execution, Raw: raw, Normalization: normalized, Bars: bars}, nil
}

func (p *FinancialDatasetsProvider) fetchHistoricalBarsAttempt(ctx context.Context, ticker string, start, end time.Time) (providercontract.ProviderAttemptResult, string) {
	endpoint, err := p.historicalPricesURL(ticker, start, end)
	if err != nil {
		failure := providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: err}
		return providercontract.ProviderAttemptResult{Failure: &failure}, ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		failure := providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: err}
		return providercontract.ProviderAttemptResult{Failure: &failure}, ""
	}
	req.Header.Set("X-API-KEY", p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		failure := providercontract.ClassifyTransportError(ctx, err)
		return providercontract.ProviderAttemptResult{Failure: &failure}, ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		failure := providercontract.ProviderFailure{HTTPStatus: resp.StatusCode, RetryAfter: resp.Header.Get("Retry-After")}
		return providercontract.ProviderAttemptResult{Failure: &failure}, ""
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	baseType, _, parseErr := mime.ParseMediaType(contentType)
	if parseErr != nil || baseType != "application/json" {
		failure := providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("provider response media type is not application/json")}
		return providercontract.ProviderAttemptResult{Failure: &failure}, ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, financialDatasetsMaximumResponseBytes+1))
	if err != nil {
		failure := providercontract.ClassifyTransportError(ctx, err)
		return providercontract.ProviderAttemptResult{Failure: &failure}, ""
	}
	if len(body) == 0 || len(body) > financialDatasetsMaximumResponseBytes {
		failure := providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("provider response body is empty or exceeds the bounded capture limit")}
		return providercontract.ProviderAttemptResult{Failure: &failure}, ""
	}
	return providercontract.ProviderAttemptResult{RawBytes: body}, contentType
}

func (p *FinancialDatasetsProvider) historicalPricesURL(ticker string, start, end time.Time) (string, error) {
	if p == nil || strings.TrimSpace(p.baseURL) == "" {
		return "", errors.New("financial-datasets base URL is required")
	}
	u, err := url.Parse(strings.TrimRight(p.baseURL, "/") + "/prices/")
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Set("ticker", ticker)
	query.Set("interval", "day")
	query.Set("start_date", start.Format("2006-01-02"))
	query.Set("end_date", end.Format("2006-01-02"))
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func ProjectCanonicalMarketBars(batch providercontract.BatchNormalizationResult, instrument canonical.Instrument) ([]CanonicalMarketBar, error) {
	if err := batch.Validate(); err != nil {
		return nil, err
	}
	if err := instrument.Validate(); err != nil {
		return nil, err
	}
	type group struct {
		end          time.Time
		observations map[string]canonical.Observation
	}
	groups := make(map[time.Time]*group)
	for _, accepted := range batch.Records {
		observation, ok := accepted.Record.(canonical.Observation)
		if !ok {
			return nil, fmt.Errorf("market bar projection requires canonical Observation values, got %T", accepted.Record)
		}
		if observation.Subject.Kind != canonical.ContractKindInstrument || observation.Subject.ID != string(instrument.ID) {
			return nil, errors.New("market observation instrument identity mismatch")
		}
		item := groups[observation.ObservedAt]
		if item == nil {
			item = &group{end: observation.ObservedAt, observations: make(map[string]canonical.Observation)}
			groups[observation.ObservedAt] = item
		}
		if _, exists := item.observations[observation.Metric]; exists {
			return nil, errors.New("duplicate market metric for bar interval")
		}
		item.observations[observation.Metric] = observation
	}
	ends := make([]time.Time, 0, len(groups))
	for end := range groups {
		ends = append(ends, end)
	}
	sort.Slice(ends, func(i, j int) bool { return ends[i].Before(ends[j]) })
	bars := make([]CanonicalMarketBar, 0, len(ends))
	for _, end := range ends {
		item := groups[end]
		for _, metric := range []string{MarketMetricOpen, MarketMetricHigh, MarketMetricLow, MarketMetricClose, MarketMetricVolume} {
			if _, ok := item.observations[metric]; !ok {
				return nil, fmt.Errorf("market bar is missing canonical metric %q", metric)
			}
		}
		start := end.AddDate(0, 0, -1)
		bars = append(bars, CanonicalMarketBar{
			Instrument: canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: string(instrument.ID), ContractVersion: instrument.ContractVersion},
			Interval:   Timeframe1Day, Start: start, End: end, ProviderDate: start.Format("2006-01-02"),
			TimestampSemantics: MarketTimestampProviderDateIntervalEnd, TimestampAuthority: MarketTimestampAuthorityIntervalBoundary,
			Session: MarketSessionProviderEODUnspecified, MarketTimezone: "UNKNOWN", Adjustment: MarketAdjustmentUnknown,
			Open: item.observations[MarketMetricOpen], High: item.observations[MarketMetricHigh], Low: item.observations[MarketMetricLow],
			Close: item.observations[MarketMetricClose], Volume: item.observations[MarketMetricVolume],
		})
		if err := bars[len(bars)-1].Validate(); err != nil {
			return nil, err
		}
	}
	return bars, nil
}

type financialDatasetsHistoricalPayload struct {
	Ticker      string                           `json:"ticker"`
	Prices      []financialDatasetsHistoricalRow `json:"prices"`
	NextPageURL string                           `json:"next_page_url,omitempty"`
}

type financialDatasetsHistoricalRow struct {
	Ticker string      `json:"ticker,omitempty"`
	Open   json.Number `json:"open"`
	Close  json.Number `json:"close"`
	High   json.Number `json:"high"`
	Low    json.Number `json:"low"`
	Volume json.Number `json:"volume"`
	Time   string      `json:"time"`
}

type validatedFinancialDatasetsRow struct {
	ProviderDate           string
	Start, End             time.Time
	Open, High, Low, Close float64
	Volume                 int64
}

func parseFinancialDatasetsHistoricalPayload(raw []byte) (financialDatasetsHistoricalPayload, error) {
	if !json.Valid(raw) {
		return financialDatasetsHistoricalPayload{}, marketNormalizationError(providercontract.NormalizationStageParsing, providercontract.NormalizationErrorParserFailure, "provider JSON is invalid", nil)
	}
	if err := rejectDuplicateJSONProperties(raw); err != nil {
		return financialDatasetsHistoricalPayload{}, marketNormalizationError(providercontract.NormalizationStageParsing, providercontract.NormalizationErrorParserFailure, "provider JSON contains duplicate object properties", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var payload financialDatasetsHistoricalPayload
	if err := decoder.Decode(&payload); err != nil {
		return financialDatasetsHistoricalPayload{}, marketNormalizationError(providercontract.NormalizationStageParsing, providercontract.NormalizationErrorParserFailure, "provider JSON does not match the documented historical-prices schema", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return financialDatasetsHistoricalPayload{}, marketNormalizationError(providercontract.NormalizationStageParsing, providercontract.NormalizationErrorParserFailure, "provider JSON contains trailing data", err)
	}
	return payload, nil
}

func validateFinancialDatasetsHistoricalPayload(payload financialDatasetsHistoricalPayload, receivedAt time.Time) (string, []validatedFinancialDatasetsRow, error) {
	if len(payload.Prices) == 0 {
		return "", nil, marketNormalizationError(providercontract.NormalizationStageParsing, providercontract.NormalizationErrorRequiredFieldMissing, "prices are required", nil)
	}
	if strings.TrimSpace(payload.NextPageURL) != "" {
		return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorAmbiguousProviderValue, "paginated historical response is incomplete for this bounded acquisition", nil)
	}
	ticker := strings.ToUpper(strings.TrimSpace(payload.Ticker))
	rows := make([]validatedFinancialDatasetsRow, 0, len(payload.Prices))
	seen := make(map[string]struct{}, len(payload.Prices))
	var previous time.Time
	for _, raw := range payload.Prices {
		rowTicker := strings.ToUpper(strings.TrimSpace(raw.Ticker))
		if ticker == "" {
			ticker = rowTicker
		}
		if ticker == "" || (rowTicker != "" && rowTicker != ticker) {
			return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorIdentityResolution, "response ticker is missing or inconsistent", nil)
		}
		date, err := time.Parse("2006-01-02", strings.TrimSpace(raw.Time))
		if err != nil {
			return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorAmbiguousProviderValue, "bar time must be an unambiguous provider session date", err)
		}
		start := date.UTC()
		end := start.AddDate(0, 0, 1)
		if end.After(receivedAt) {
			return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorAmbiguousProviderValue, "EOD bar interval has not completed by acquisition time", nil)
		}
		if _, duplicate := seen[raw.Time]; duplicate {
			return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorInvalidProviderValue, "duplicate or conflicting bar interval", nil)
		}
		if !previous.IsZero() && !start.After(previous) {
			return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorInvalidProviderValue, "bars are not in strict chronological order", nil)
		}
		seen[raw.Time] = struct{}{}
		previous = start
		open, err := parseFinitePositivePrice(raw.Open)
		if err != nil {
			return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorInvalidProviderValue, "open price is invalid", err)
		}
		high, err := parseFinitePositivePrice(raw.High)
		if err != nil {
			return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorInvalidProviderValue, "high price is invalid", err)
		}
		low, err := parseFinitePositivePrice(raw.Low)
		if err != nil {
			return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorInvalidProviderValue, "low price is invalid", err)
		}
		closeValue, err := parseFinitePositivePrice(raw.Close)
		if err != nil {
			return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorInvalidProviderValue, "close price is invalid", err)
		}
		volume, err := strconv.ParseInt(string(raw.Volume), 10, 64)
		if err != nil || volume < 0 {
			return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorInvalidProviderValue, "volume must be a non-negative integer", err)
		}
		if high < low || high < open || high < closeValue || low > open || low > closeValue {
			return "", nil, marketNormalizationError(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorInvalidProviderValue, "OHLC invariants are violated", nil)
		}
		rows = append(rows, validatedFinancialDatasetsRow{ProviderDate: raw.Time, Start: start, End: end, Open: open, High: high, Low: low, Close: closeValue, Volume: volume})
	}
	return ticker, rows, nil
}

func validateFinancialDatasetsBarsRequest(request FinancialDatasetsBarsRequest) error {
	if err := request.Instrument.Validate(); err != nil {
		return fmt.Errorf("invalid canonical instrument: %w", err)
	}
	if request.Interval != Timeframe1Day {
		return fmt.Errorf("%w: hardened financial-datasets bars path supports documented daily EOD bars only", ErrInvalidTimeframe)
	}
	if !isUTCDate(request.StartDate) || !isUTCDate(request.EndDate) || request.EndDate.Before(request.StartDate) {
		return errors.New("financial-datasets bar range must use ordered UTC calendar dates")
	}
	if days := int(request.EndDate.Sub(request.StartDate).Hours()/24) + 1; days <= 0 || days > financialDatasetsMaximumRequestCalendar {
		return errors.New("financial-datasets bar range exceeds the bounded calendar window")
	}
	if err := request.Retention.Validate(); err != nil {
		return fmt.Errorf("invalid raw retention policy: %w", err)
	}
	if strings.TrimSpace(string(request.PayloadID)) == "" {
		return errors.New("raw payload acquisition identity is required")
	}
	_, err := financialDatasetsTicker(request.Instrument)
	return err
}

func financialDatasetsTicker(instrument canonical.Instrument) (string, error) {
	var ticker string
	for _, external := range instrument.ExternalIDs {
		if external.Namespace != "financialdatasets.ticker" && external.Namespace != "ticker.us" {
			continue
		}
		candidate := strings.ToUpper(strings.TrimSpace(external.Value))
		if candidate == "" || strings.ContainsAny(candidate, " /\\?&#") {
			return "", errors.New("canonical instrument contains an invalid Financial Datasets ticker identity")
		}
		if ticker != "" && ticker != candidate {
			return "", errors.New("canonical instrument contains conflicting ticker identities")
		}
		ticker = candidate
	}
	if ticker == "" {
		return "", errors.New("canonical instrument lacks a Financial Datasets or US ticker external identity")
	}
	return ticker, nil
}

func parseFinitePositivePrice(number json.Number) (float64, error) {
	if strings.TrimSpace(string(number)) == "" {
		return 0, errors.New("number is required")
	}
	value, err := strconv.ParseFloat(string(number), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0, errors.New("price must be finite and positive")
	}
	return value, nil
}

func isUTCDate(value time.Time) bool {
	_, offset := value.Zone()
	return !value.IsZero() && offset == 0 && value.Hour() == 0 && value.Minute() == 0 && value.Second() == 0 && value.Nanosecond() == 0
}

func marketNormalizationError(stage providercontract.NormalizationStage, code providercontract.NormalizationErrorCode, detail string, cause error) error {
	return &providercontract.NormalizationError{Stage: stage, Code: code, Detail: detail, Cause: cause}
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
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("object key is not a string")
				}
				if _, duplicate := seen[key]; duplicate {
					return fmt.Errorf("duplicate JSON property %q", key)
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
			return errors.New("unexpected JSON delimiter")
		}
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("JSON contains trailing data")
	}
	return nil
}

type singleInstrumentResolver struct{ instrument canonical.Instrument }

func (resolver singleInstrumentResolver) ResolveFinancialDatasetsTicker(ticker string) (canonical.Instrument, error) {
	resolved, err := financialDatasetsTicker(resolver.instrument)
	if err != nil || resolved != strings.ToUpper(strings.TrimSpace(ticker)) {
		return canonical.Instrument{}, errors.New("ticker is not mapped to the requested canonical instrument")
	}
	return resolver.instrument, nil
}

func cloneMarketProviderIdentity(identity canonical.ProviderIdentity) canonical.ProviderIdentity {
	copyIdentity := identity
	if identity.ExternalID != nil {
		external := *identity.ExternalID
		copyIdentity.ExternalID = &external
	}
	return copyIdentity
}
