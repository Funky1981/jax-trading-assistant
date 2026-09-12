package marketdata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	AlpacaProviderID           = "pvd_alpaca_market_data"
	AlpacaHistoricalSourceID   = "src_alpaca_stock_bars"
	AlpacaBarsNormalizerID     = "cmp_alpaca_stock_bars"
	AlpacaBarsMaximumBytes     = 4 << 20
	AlpacaBarsMaximumPages     = 32
	AlpacaHistoricalRawVersion = "documented-2026-09/v1"
)

var (
	AlpacaProviderIdentity = canonical.ProviderIdentity{
		ID: AlpacaProviderID, Namespace: "alpaca.market_data_api",
		ExternalID: &canonical.ExternalID{Namespace: "provider.slug", Value: "alpaca"},
	}
	AlpacaHistoricalSource = canonical.SourceIdentity{ID: AlpacaHistoricalSourceID, Kind: canonical.SourceKindDataset}
	AlpacaHistoricalRaw    = canonical.VersionIdentity{Namespace: "alpaca.stock_bars", Value: AlpacaHistoricalRawVersion}
)

type AlpacaBarsRequest struct {
	Instrument canonical.Instrument
	StartDate  time.Time
	EndDate    time.Time
	Interval   Timeframe
	Feed       MarketFeed
	Adjustment MarketAdjustmentState
	PayloadID  providercontract.RawPayloadID
	Retention  providercontract.RawPayloadRetentionPolicy
}

type AlpacaBarsDependencies struct {
	Registry *providercontract.Registry
	Executor *providercontract.OperationalExecutor
	Store    providercontract.RawPayloadStore
	Pipeline *providercontract.NormalizationPipeline
}

type AlpacaBarsResult struct {
	Executions    []providercontract.ExecutionResult
	RawPayloads   []providercontract.RawPayloadDescriptor
	Normalization providercontract.BatchNormalizationResult
	Bars          []CanonicalMarketBar
}

func AlpacaProviderDefinition() providercontract.ProviderDefinition {
	return providercontract.ProviderDefinition{
		ContractVersion:    providercontract.ProviderDefinitionV1,
		Identity:           cloneMarketProviderIdentity(AlpacaProviderIdentity),
		DisplayName:        "Alpaca Historical Stock Bars",
		AdapterVersion:     canonical.VersionIdentity{Namespace: "jax.marketdata.alpaca", Value: "1.0.0"},
		ProviderAPIVersion: &canonical.VersionIdentity{Namespace: "alpaca.data_api", Value: "v2"},
		Capabilities: []providercontract.Capability{{
			ContractVersion: providercontract.CapabilityContractV1,
			ID:              providercontract.CapabilityMarketBars,
			Category:        providercontract.DataCategoryMarketData,
			Support:         providercontract.SupportSupported,
			Raw: providercontract.RawRepresentation{
				Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument,
				Schema: AlpacaHistoricalRaw, MediaType: "application/json",
			},
			Authentication: providercontract.AuthenticationRequirement{Class: providercontract.AuthenticationAPIKeyPair},
			Operational: providercontract.OperationalSemantics{
				DeliveryModes:      []providercontract.DeliveryMode{providercontract.DeliveryHistorical},
				FreshnessModes:     []providercontract.FreshnessMode{providercontract.FreshnessEndOfDay},
				QualityRequirement: providercontract.QualityCanonicalValidationRequired,
			},
			CanonicalOutputs: []canonical.ContractSchemaRef{{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2}},
		}},
	}
}

func (p *AlpacaProvider) AcquireAndNormalizeDailyBars(ctx context.Context, dependencies AlpacaBarsDependencies, request AlpacaBarsRequest) (AlpacaBarsResult, error) {
	result := AlpacaBarsResult{}
	if err := validateAlpacaBarsRequest(request); err != nil {
		return result, err
	}
	if p == nil || dependencies.Registry == nil || dependencies.Executor == nil || dependencies.Store == nil || dependencies.Pipeline == nil {
		return result, errors.New("alpaca hardened bars path is not fully configured")
	}
	if p.rawClient == nil {
		p.rawClient = &http.Client{Timeout: 30 * time.Second}
	}
	if strings.TrimSpace(p.baseURL) == "" {
		return result, errors.New("alpaca market-data base URL is required")
	}
	definition := AlpacaProviderDefinition()
	if _, err := dependencies.Registry.Lookup(definition.Identity); err != nil {
		return result, fmt.Errorf("alpaca provider is not registered: %w", err)
	}
	feed := request.Feed
	adjustment := request.Adjustment
	normalizer, err := newAlpacaBarsNormalizer(request.Instrument, feed, adjustment)
	if err != nil {
		return result, err
	}
	operation := providercontract.Operation{ContractVersion: providercontract.OperationContractV1, Provider: cloneMarketProviderIdentity(AlpacaProviderIdentity), CapabilityID: providercontract.CapabilityMarketBars, Kind: providercontract.OperationReadFetch, RetrySafety: providercontract.RetrySafetyRepeatable}

	seenTokens := map[string]struct{}{"": {}}
	pageToken := ""
	allRecords := make([]providercontract.NormalizationResult, 0)
	var firstRaw providercontract.RawPayloadRef
	for page := 0; page < AlpacaBarsMaximumPages; page++ {
		endpoint, err := p.historicalBarsURL(request.Instrument, request.StartDate, request.EndDate, feed, adjustment, pageToken)
		if err != nil {
			return result, err
		}
		var contentType string
		execution, err := dependencies.Executor.Execute(ctx, operation, func(attemptCtx context.Context, _ providercontract.AttemptContext) providercontract.ProviderAttemptResult {
			attempt, mediaType := p.fetchAlpacaBarsAttempt(attemptCtx, endpoint)
			if attempt.Failure == nil {
				contentType = mediaType
			}
			return attempt
		})
		result.Executions = append(result.Executions, execution)
		if err != nil {
			return result, err
		}
		pageID := request.PayloadID
		if page > 0 {
			pageID = providercontract.RawPayloadID("rpa_" + canonical.DigestBytes([]byte(string(request.PayloadID) + "\x00" + pageToken)).Value[:24])
		}
		source := alpacaSource(feed, adjustment)
		revision := canonical.RevisionIdentity{Namespace: "alpaca.stock_bars.response_sha256", Value: canonical.DigestBytes(execution.RawBytes).Value}
		raw, err := providercontract.PersistRawPayload(ctx, dependencies.Registry, dependencies.Store, providercontract.RawPayloadPersistenceRequest{
			ID: pageID, Provider: definition.Identity, Capability: providercontract.CapabilityMarketBars,
			Raw:     providercontract.RawRepresentation{Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument, Schema: AlpacaHistoricalRaw, MediaType: contentType},
			Capture: providercontract.RawPayloadCapture{ByteForm: providercontract.RawPayloadByteFormEntityBody, ContentCodingState: providercontract.ContentCodingIdentity, CharacterEncoding: "utf-8"},
			Source:  &source, Revision: &revision, ReceivedAt: execution.CompletedAt, Retention: request.Retention, Complete: true,
		}, execution.RawBytes)
		if err != nil {
			return result, err
		}
		result.RawPayloads = append(result.RawPayloads, raw)
		if page == 0 {
			firstRaw = raw.Ref
		}
		normalized, err := dependencies.Pipeline.NormalizeBatchStored(ctx, dependencies.Store, providercontract.StoredNormalizationRequest{RawRef: raw.Ref, Target: normalizer.descriptor.Target, Normalizer: normalizer.descriptor.Component})
		if err != nil {
			return result, err
		}
		allRecords = append(allRecords, normalized.Records...)
		payload, err := providercontract.RetrieveRawPayload(ctx, dependencies.Store, raw.Ref)
		if err != nil {
			return result, err
		}
		parsed, err := parseAlpacaBarsPayload(payload)
		if err != nil {
			return result, err
		}
		next := strings.TrimSpace(parsed.NextPageToken)
		if next == "" {
			break
		}
		if _, exists := seenTokens[next]; exists {
			return result, errors.New("alpaca pagination token did not progress")
		}
		seenTokens[next] = struct{}{}
		pageToken = next
		if page == AlpacaBarsMaximumPages-1 {
			return result, errors.New("alpaca pagination exceeded bounded page limit")
		}
	}
	if len(allRecords) == 0 || firstRaw.ID == "" {
		return result, errors.New("alpaca response contained no accepted bars")
	}
	result.Normalization = providercontract.BatchNormalizationResult{RawRef: firstRaw, Normalizer: normalizer.descriptor.Component, Target: normalizer.descriptor.Target, Records: allRecords}
	result.Bars, err = projectAlpacaMarketBars(result.Normalization, request.Instrument, feed, adjustment)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (p *AlpacaProvider) historicalBarsURL(instrument canonical.Instrument, start, end time.Time, feed MarketFeed, adjustment MarketAdjustmentState, pageToken string) (string, error) {
	ticker, err := alpacaTicker(instrument)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(strings.TrimRight(p.baseURL, "/") + "/v2/stocks/bars")
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("symbols", ticker)
	query.Set("timeframe", "1Day")
	query.Set("start", start.Format("2006-01-02"))
	query.Set("end", end.Format("2006-01-02"))
	query.Set("limit", "1000")
	query.Set("adjustment", alpacaAdjustmentQuery(adjustment))
	query.Set("feed", string(feed))
	query.Set("sort", "asc")
	if strings.TrimSpace(pageToken) != "" {
		query.Set("page_token", pageToken)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (p *AlpacaProvider) fetchAlpacaBarsAttempt(ctx context.Context, endpoint string) (providercontract.ProviderAttemptResult, string) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: err}}, ""
	}
	request.Header.Set("APCA-API-KEY-ID", p.config.APIKey)
	request.Header.Set("APCA-API-SECRET-KEY", p.config.APISecret)
	request.Header.Set("Accept", "application/json")
	response, err := p.rawClient.Do(request)
	if err != nil {
		failure := providercontract.ClassifyTransportError(ctx, err)
		return providercontract.ProviderAttemptResult{Failure: &failure}, ""
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{HTTPStatus: response.StatusCode, RetryAfter: response.Header.Get("Retry-After")}}, ""
	}
	contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
	baseType, _, parseErr := mime.ParseMediaType(contentType)
	if parseErr != nil || baseType != "application/json" {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("alpaca response media type is not application/json")}}, ""
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, AlpacaBarsMaximumBytes+1))
	if err != nil {
		failure := providercontract.ClassifyTransportError(ctx, err)
		return providercontract.ProviderAttemptResult{Failure: &failure}, ""
	}
	if len(body) == 0 || len(body) > AlpacaBarsMaximumBytes {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("alpaca response body is empty or exceeds the bounded capture limit")}}, ""
	}
	return providercontract.ProviderAttemptResult{RawBytes: body}, baseType
}

type alpacaBarsNormalizer struct {
	descriptor canonicalNormalizerDescriptor
	instrument canonical.Instrument
	feed       MarketFeed
	adjustment MarketAdjustmentState
}

type canonicalNormalizerDescriptor = providercontract.NormalizerDescriptor

func newAlpacaBarsNormalizer(instrument canonical.Instrument, feed MarketFeed, adjustment MarketAdjustmentState) (*alpacaBarsNormalizer, error) {
	if err := instrument.Validate(); err != nil {
		return nil, err
	}
	if err := validateAlpacaFeed(feed); err != nil {
		return nil, err
	}
	if adjustment != MarketAdjustmentUnadjusted {
		return nil, errors.New("alpaca hardened bars path currently requires raw/unadjusted bars")
	}
	providerIdentity := cloneMarketProviderIdentity(AlpacaProviderIdentity)
	mappingContent := canonical.RawContentIdentity([]byte("jax Alpaca daily stock bars mapping/v1"))
	descriptor := providercontract.NormalizerDescriptor{ContractVersion: providercontract.NormalizerDescriptorV1, Provider: providerIdentity, CapabilityID: providercontract.CapabilityMarketBars, Raw: providercontract.RawRepresentation{Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument, Schema: AlpacaHistoricalRaw, MediaType: "application/json"}, Component: canonical.ComponentIdentity{ID: AlpacaBarsNormalizerID, Kind: canonical.ComponentKindNormalizer, Name: "Alpaca documented daily stock bars normalizer", Version: canonical.VersionIdentity{Namespace: "jax.normalizer.market_bars", Value: "alpaca-v1"}, Provider: &providerIdentity, Content: &mappingContent}, Target: canonical.ContractSchemaRef{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2}}
	if err := descriptor.Validate(); err != nil {
		return nil, err
	}
	return &alpacaBarsNormalizer{descriptor: descriptor, instrument: instrument, feed: feed, adjustment: adjustment}, nil
}

// NewAlpacaBarsNormalizerForGate exposes the accepted deterministic normalizer
// to the bounded Phase-03 packet harness without exposing provider DTOs.
func NewAlpacaBarsNormalizerForGate(instrument canonical.Instrument, feed MarketFeed) (providercontract.Normalizer, error) {
	return newAlpacaBarsNormalizer(instrument, feed, MarketAdjustmentUnadjusted)
}

func (normalizer *alpacaBarsNormalizer) Descriptor() providercontract.NormalizerDescriptor {
	return normalizer.descriptor
}

func (normalizer *alpacaBarsNormalizer) Normalize(context.Context, providercontract.NormalizationInput) (providercontract.NormalizationCandidate, error) {
	return providercontract.NormalizationCandidate{}, errors.New("alpaca stock bars response contains a bounded observation batch")
}

func (normalizer *alpacaBarsNormalizer) NormalizeBatch(_ context.Context, input providercontract.NormalizationInput) ([]providercontract.NormalizationCandidate, error) {
	payload, err := parseAlpacaBarsPayload(input.Bytes)
	if err != nil {
		return nil, err
	}
	ticker, err := alpacaTicker(normalizer.instrument)
	if err != nil {
		return nil, err
	}
	rows, ok := payload.Bars[ticker]
	if !ok || len(rows) == 0 {
		return nil, errors.New("alpaca response contains no bars for the requested canonical symbol")
	}
	if len(payload.Bars) != 1 {
		return nil, errors.New("alpaca response contains an unexpected symbol")
	}
	result := make([]providercontract.NormalizationCandidate, 0, len(rows)*5)
	seen := map[time.Time]struct{}{}
	var previous time.Time
	for _, row := range rows {
		observedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(row.Timestamp))
		if err != nil || observedAt.Location() != time.UTC {
			return nil, errors.New("alpaca daily bar timestamp must be an RFC3339 UTC timestamp")
		}
		observedAt = observedAt.UTC()
		if observedAt.After(input.RawRef.ReceivedAt) {
			return nil, errors.New("alpaca bar timestamp is later than acquisition time")
		}
		if !previous.IsZero() && !observedAt.After(previous) {
			return nil, errors.New("alpaca bars are not in strict chronological order")
		}
		if _, exists := seen[observedAt]; exists {
			return nil, errors.New("alpaca response contains duplicate bar timestamps")
		}
		seen[observedAt] = struct{}{}
		previous = observedAt
		values := []struct {
			metric string
			typeID canonical.ObservationType
			number json.Number
			unit   string
			field  string
		}{
			{MarketMetricOpen, canonical.ObservationTypePrice, row.Open, normalizer.instrument.Currency, "open"},
			{MarketMetricHigh, canonical.ObservationTypePrice, row.High, normalizer.instrument.Currency, "high"},
			{MarketMetricLow, canonical.ObservationTypePrice, row.Low, normalizer.instrument.Currency, "low"},
			{MarketMetricClose, canonical.ObservationTypePrice, row.Close, normalizer.instrument.Currency, "close"},
			{MarketMetricVolume, canonical.ObservationTypeVolume, row.Volume, "shares", "volume"},
		}
		parsed := make([]float64, 0, len(values))
		for _, value := range values {
			var parsedValue float64
			var parseErr error
			if value.metric == MarketMetricVolume {
				parsedValue, parseErr = parseAlpacaVolume(value.number)
			} else {
				parsedValue, parseErr = parseAlpacaPrice(value.number)
			}
			if parseErr != nil {
				return nil, fmt.Errorf("alpaca %s is invalid: %w", value.field, parseErr)
			}
			parsed = append(parsed, parsedValue)
		}
		if parsed[1] < parsed[2] || parsed[1] < parsed[0] || parsed[1] < parsed[3] || parsed[2] > parsed[0] || parsed[2] > parsed[3] || parsed[4] < 0 {
			return nil, errors.New("alpaca OHLCV invariants are violated")
		}
		for index, value := range values {
			seed := strings.Join([]string{string(input.RawRef.ID), input.RawRef.Content.Digest.Value, normalizer.descriptor.Component.ID, normalizer.descriptor.Component.Version.Value, string(normalizer.instrument.ID), normalizer.feed.String(), string(normalizer.adjustment), observedAt.Format(time.RFC3339Nano), value.metric}, "\x00")
			digest := canonical.DigestBytes([]byte(seed)).Value
			evidenceID := canonical.EvidenceID("evd_" + digest[24:48])
			observationID := canonical.ObservationID("obs_" + digest[:24])
			evidenceRef, refErr := input.RawRef.AsEvidenceRef(canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(evidenceID), ContractVersion: canonical.EvidenceContractV2})
			if refErr != nil {
				return nil, refErr
			}
			lineage := canonical.LineageInput{Kind: canonical.LineageInputKindEvidence, Evidence: &evidenceRef}
			fingerprint, fpErr := canonical.ComputeInputFingerprint([]canonical.LineageInput{lineage})
			if fpErr != nil {
				return nil, fpErr
			}
			number := parsed[index]
			observation := canonical.Observation{ContractVersion: canonical.ObservationContractV2, ID: observationID, Type: value.typeID, Subject: canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: string(normalizer.instrument.ID), ContractVersion: normalizer.instrument.ContractVersion}, Metric: value.metric, Value: canonical.ObservedValue{Type: canonical.ObservedValueTypeNumber, Number: &number, Unit: value.unit}, Source: canonical.SourceReference{ID: alpacaSourceID(normalizer.feed, normalizer.adjustment), Kind: canonical.SourceKindDataset, ExternalID: &canonical.ExternalID{Namespace: "alpaca.feed.adjustment", Value: normalizer.feed.String() + "/" + string(normalizer.adjustment)}, URI: "https://data.alpaca.markets/v2/stocks/bars?feed=" + normalizer.feed.String() + "&adjustment=" + alpacaAdjustmentQuery(normalizer.adjustment)}, EvidenceIDs: []canonical.EvidenceID{evidenceID}, ObservedAt: observedAt, CollectedAt: input.RawRef.ReceivedAt, CreatedAt: input.RawRef.ReceivedAt, Provenance: &canonical.Provenance{ContractVersion: canonical.ProvenanceContractV1, ID: "pvn_" + digest[:24], Inputs: []canonical.LineageInput{lineage}, InputFingerprint: fingerprint, Producer: normalizer.descriptor.Component}}
			result = append(result, providercontract.NormalizationCandidate{Record: observation, Revision: canonical.RevisionIdentity{Namespace: "jax.normalized.alpaca_market_bar", Value: "v1/" + digest}, Dispositions: []providercontract.FieldDisposition{{ProviderField: "symbol", Status: providercontract.FieldDispositionRepresented, CanonicalField: "subject"}, {ProviderField: "t", Status: providercontract.FieldDispositionRepresented, CanonicalField: "observed_at"}, {ProviderField: value.field, Status: providercontract.FieldDispositionRepresented, CanonicalField: "value.number"}, {ProviderField: "feed", Status: providercontract.FieldDispositionRepresented, CanonicalField: "source.external_id"}, {ProviderField: "adjustment", Status: providercontract.FieldDispositionRepresented, CanonicalField: "source.external_id"}}})
		}
	}
	return result, nil
}

type alpacaBarsPayload struct {
	Bars          map[string][]alpacaBarRow `json:"bars"`
	NextPageToken string                    `json:"next_page_token,omitempty"`
}

type alpacaBarRow struct {
	Timestamp  string      `json:"t"`
	Open       json.Number `json:"o"`
	High       json.Number `json:"h"`
	Low        json.Number `json:"l"`
	Close      json.Number `json:"c"`
	Volume     json.Number `json:"v"`
	TradeCount json.Number `json:"n"`
	VWAP       json.Number `json:"vw"`
}

func parseAlpacaBarsPayload(raw []byte) (alpacaBarsPayload, error) {
	if !json.Valid(raw) {
		return alpacaBarsPayload{}, errors.New("alpaca response JSON is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var payload alpacaBarsPayload
	if err := decoder.Decode(&payload); err != nil {
		return alpacaBarsPayload{}, fmt.Errorf("alpaca response does not match the documented bars schema: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return alpacaBarsPayload{}, errors.New("alpaca response contains trailing data")
	}
	if payload.Bars == nil {
		return alpacaBarsPayload{}, errors.New("alpaca response bars object is required")
	}
	return payload, nil
}

func parseAlpacaPrice(value json.Number) (float64, error) {
	text := strings.TrimSpace(string(value))
	if text == "" {
		return 0, errors.New("number is required")
	}
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil || parsed < 0 || strings.ContainsAny(strings.ToLower(text), "ninf") {
		return 0, errors.New("number must be finite and in range")
	}
	return parsed, nil
}

func parseAlpacaVolume(value json.Number) (float64, error) {
	text := strings.TrimSpace(string(value))
	if text == "" {
		return 0, errors.New("number is required")
	}
	parsed, err := strconv.ParseUint(text, 10, 63)
	if err != nil {
		return 0, errors.New("volume must be a non-negative integer")
	}
	return float64(parsed), nil
}

func alpacaTicker(instrument canonical.Instrument) (string, error) {
	var ticker string
	for _, external := range instrument.ExternalIDs {
		if external.Namespace != "alpaca.symbol" && external.Namespace != "ticker.us" && external.Namespace != "ticker.xnas" {
			continue
		}
		candidate := strings.ToUpper(strings.TrimSpace(external.Value))
		if candidate == "" || strings.ContainsAny(candidate, " /\\?&#") {
			return "", errors.New("canonical instrument contains an invalid Alpaca symbol identity")
		}
		if ticker != "" && ticker != candidate {
			return "", errors.New("canonical instrument contains conflicting Alpaca symbol identities")
		}
		ticker = candidate
	}
	if ticker == "" {
		return "", errors.New("canonical instrument lacks an Alpaca symbol identity")
	}
	return ticker, nil
}

func validateAlpacaFeed(feed MarketFeed) error {
	if feed != MarketFeedSIP && feed != MarketFeedIEX {
		return fmt.Errorf("alpaca feed must be explicit and one of %q or %q", MarketFeedSIP, MarketFeedIEX)
	}
	return nil
}

func alpacaAdjustmentQuery(adjustment MarketAdjustmentState) string {
	if adjustment == MarketAdjustmentUnadjusted {
		return "raw"
	}
	return strings.ToLower(string(adjustment))
}

func (feed MarketFeed) String() string { return string(feed) }

func alpacaSource(feed MarketFeed, adjustment MarketAdjustmentState) canonical.SourceIdentity {
	return canonical.SourceIdentity{ID: alpacaSourceID(feed, adjustment), Kind: canonical.SourceKindDataset}
}

func alpacaSourceID(feed MarketFeed, adjustment MarketAdjustmentState) string {
	return AlpacaHistoricalSourceID + "_" + feed.String() + "_" + strings.ToLower(string(adjustment))
}

func validateAlpacaBarsRequest(request AlpacaBarsRequest) error {
	if err := request.Instrument.Validate(); err != nil {
		return fmt.Errorf("invalid canonical instrument: %w", err)
	}
	if request.Interval != Timeframe1Day {
		return fmt.Errorf("%w: Alpaca hardened path supports daily bars only", ErrInvalidTimeframe)
	}
	if !isUTCDate(request.StartDate) || !isUTCDate(request.EndDate) || request.EndDate.Before(request.StartDate) {
		return errors.New("alpaca bar range must use ordered UTC calendar dates")
	}
	if days := int(request.EndDate.Sub(request.StartDate).Hours()/24) + 1; days <= 0 || days > 366 {
		return errors.New("alpaca bar range exceeds the bounded calendar window")
	}
	if err := validateAlpacaFeed(request.Feed); err != nil {
		return err
	}
	if request.Adjustment != MarketAdjustmentUnadjusted {
		return errors.New("alpaca hardened path requires explicit raw/unadjusted adjustment")
	}
	if strings.TrimSpace(string(request.PayloadID)) == "" {
		return errors.New("raw payload acquisition identity is required")
	}
	return request.Retention.Validate()
}

func projectAlpacaMarketBars(batch providercontract.BatchNormalizationResult, instrument canonical.Instrument, feed MarketFeed, adjustment MarketAdjustmentState) ([]CanonicalMarketBar, error) {
	if err := batch.RawRef.Validate(); err != nil {
		return nil, fmt.Errorf("alpaca projection received an invalid first raw reference: %w", err)
	}
	if err := batch.Normalizer.Validate(); err != nil {
		return nil, fmt.Errorf("alpaca projection received an invalid normalizer: %w", err)
	}
	if err := batch.Target.Validate(); err != nil {
		return nil, fmt.Errorf("alpaca projection received an invalid target: %w", err)
	}
	if len(batch.Records) == 0 {
		return nil, errors.New("alpaca projection received no normalized records")
	}
	if err := instrument.Validate(); err != nil {
		return nil, err
	}
	if err := validateAlpacaFeed(feed); err != nil {
		return nil, err
	}
	if adjustment != MarketAdjustmentUnadjusted {
		return nil, errors.New("alpaca projection requires raw/unadjusted adjustment")
	}
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, err
	}
	type group struct {
		at           time.Time
		observations map[string]canonical.Observation
	}
	groups := map[time.Time]*group{}
	for _, accepted := range batch.Records {
		if accepted.Status != providercontract.NormalizationStatusAccepted || accepted.Quality != providercontract.NormalizationQualityValidated {
			return nil, errors.New("alpaca projection received a non-accepted normalization result")
		}
		if err := accepted.RawRef.Validate(); err != nil {
			return nil, fmt.Errorf("alpaca projection received an invalid raw reference: %w", err)
		}
		if err := accepted.Output.Validate(); err != nil {
			return nil, fmt.Errorf("alpaca projection received an invalid canonical output reference: %w", err)
		}
		observation, ok := accepted.Record.(canonical.Observation)
		if !ok || observation.Subject.ID != string(instrument.ID) {
			return nil, errors.New("alpaca projection received an invalid canonical observation")
		}
		if err := observation.Validate(); err != nil {
			return nil, fmt.Errorf("alpaca projection received an invalid canonical observation: %w", err)
		}
		item := groups[observation.ObservedAt]
		if item == nil {
			item = &group{at: observation.ObservedAt, observations: map[string]canonical.Observation{}}
			groups[observation.ObservedAt] = item
		}
		if _, exists := item.observations[observation.Metric]; exists {
			return nil, errors.New("duplicate Alpaca market metric for bar interval")
		}
		item.observations[observation.Metric] = observation
	}
	starts := make([]time.Time, 0, len(groups))
	for start := range groups {
		starts = append(starts, start)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i].Before(starts[j]) })
	bars := make([]CanonicalMarketBar, 0, len(starts))
	coverage := MarketFeedCoverageSIP
	if feed == MarketFeedIEX {
		coverage = MarketFeedCoverageIEX
	}
	for _, observedAt := range starts {
		local := observedAt.In(loc)
		start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc).UTC()
		if !start.Equal(observedAt) {
			return nil, errors.New("alpaca daily bar timestamp is not the documented New York date boundary")
		}
		nextLocal := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, loc).UTC()
		item := groups[observedAt]
		for _, metric := range []string{MarketMetricOpen, MarketMetricHigh, MarketMetricLow, MarketMetricClose, MarketMetricVolume} {
			if _, ok := item.observations[metric]; !ok {
				return nil, fmt.Errorf("alpaca market bar is missing canonical metric %q", metric)
			}
		}
		bar := CanonicalMarketBar{Instrument: canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: string(instrument.ID), ContractVersion: instrument.ContractVersion}, Interval: Timeframe1Day, Start: start, End: nextLocal, ProviderDate: local.Format("2006-01-02"), TimestampSemantics: MarketTimestampProviderBarTimestamp, TimestampAuthority: MarketTimestampAuthorityProviderTimestamp, Session: MarketSessionProviderEODUnspecified, MarketTimezone: "America/New_York", Feed: feed, FeedCoverage: coverage, Adjustment: adjustment, Open: item.observations[MarketMetricOpen], High: item.observations[MarketMetricHigh], Low: item.observations[MarketMetricLow], Close: item.observations[MarketMetricClose], Volume: item.observations[MarketMetricVolume]}
		if err := bar.Validate(); err != nil {
			return nil, err
		}
		bars = append(bars, bar)
	}
	return bars, nil
}
