// Package fred provides a bounded, raw-first adapter for the official FRED
// API. ALFRED historical semantics are represented as an explicit information
// state on the same logical provider identity.
package fred

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

type Provider struct {
	config Config
	client *http.Client
}

func NewProvider(config Config, client *http.Client) (*Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Provider{config: config, client: client}, nil
}

func LoadConfigFromEnv() (Config, error) {
	config := Config{
		BaseURL:          strings.TrimRight(strings.TrimSpace(os.Getenv("FRED_BASE_URL")), "/"),
		APIKey:           strings.TrimSpace(os.Getenv(APIKeyEnvironment)),
		MaxResponseBytes: DefaultMaxResponse,
		MaxPages:         DefaultMaxPages,
	}
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	if config.APIKey == "" {
		return Config{}, fmt.Errorf("%s is required for FRED API access", APIKeyEnvironment)
	}
	return config, config.Validate()
}

func RegisterFREDProvider(registry *providercontract.Registry) error {
	if registry == nil {
		return errors.New("FRED provider registry is required")
	}
	return registry.Register(FREDProviderDefinition())
}

func FREDProviderDefinition() providercontract.ProviderDefinition {
	raw := fredRawRepresentation()
	return providercontract.ProviderDefinition{
		ContractVersion:    providercontract.ProviderDefinitionV1,
		Identity:           ProviderIdentity,
		DisplayName:        "Federal Reserve Bank of St. Louis FRED/ALFRED",
		AdapterVersion:     canonical.VersionIdentity{Namespace: "jax.fred.adapter", Value: AdapterVersion},
		ProviderAPIVersion: &canonical.VersionIdentity{Namespace: "fred.api.documentation", Value: "v1"},
		Capabilities: []providercontract.Capability{{
			ContractVersion: providercontract.CapabilityContractV1,
			ID:              providercontract.CapabilityMacroObservation,
			Category:        providercontract.DataCategoryMacroeconomicData,
			Support:         providercontract.SupportSupported,
			Raw:             raw,
			Authentication:  providercontract.AuthenticationRequirement{Class: providercontract.AuthenticationAPIKey},
			Operational: providercontract.OperationalSemantics{
				DeliveryModes:      []providercontract.DeliveryMode{providercontract.DeliverySnapshot, providercontract.DeliveryHistorical},
				FreshnessModes:     []providercontract.FreshnessMode{providercontract.FreshnessOnDemand},
				QualityRequirement: providercontract.QualityCanonicalValidationRequired,
			},
			ProviderNeutralOutputs: []providercontract.ProviderNeutralOutput{{
				ContractVersion: providercontract.ProviderNeutralOutputV1,
				Schema:          canonical.VersionIdentity{Namespace: "jax.macroevidence", Value: "macro_observation/v1"},
			}},
		}},
	}
}

func fredRawRepresentation() providercontract.RawRepresentation {
	return providercontract.RawRepresentation{
		Boundary:  providercontract.RawBoundaryProvider,
		Format:    providercontract.RawFormatJSONDocument,
		Schema:    canonical.VersionIdentity{Namespace: RawSchemaNamespace, Value: RawSchemaValue},
		MediaType: "application/json",
	}
}

func (provider *Provider) AcquireSeriesMetadata(ctx context.Context, deps Dependencies, request SeriesRequest) (SeriesResult, error) {
	if err := validateSeriesRequest(request); err != nil {
		return SeriesResult{}, err
	}
	if err := provider.validateDependencies(deps); err != nil {
		return SeriesResult{}, err
	}
	endpoint, err := provider.endpoint("series")
	if err != nil {
		return SeriesResult{}, err
	}
	query, err := seriesQuery(request)
	if err != nil {
		return SeriesResult{}, err
	}
	execution, err := provider.acquire(ctx, deps, providercontract.OperationMetadataFetch, endpoint, query)
	if err != nil {
		return SeriesResult{Execution: execution}, err
	}
	raw, err := provider.persist(ctx, deps, execution, request.PayloadID, request.Retention)
	if err != nil {
		return SeriesResult{Execution: execution}, err
	}
	payload, err := providercontract.RetrieveRawPayload(ctx, deps.Store, raw.Ref)
	if err != nil {
		return SeriesResult{Execution: execution, Raw: raw}, err
	}
	series, err := normalizeSeries(payload, raw.Ref, request.SeriesID, metadataInformationState(request.InformationState))
	if err != nil {
		return SeriesResult{Execution: execution, Raw: raw}, err
	}
	return SeriesResult{Execution: execution, Raw: raw, Series: series}, nil
}

func (provider *Provider) AcquireVintageDates(ctx context.Context, deps Dependencies, request VintageDatesRequest) (VintageDatesResult, error) {
	if err := validateVintageDatesRequest(request); err != nil {
		return VintageDatesResult{Completeness: CompletenessIncomplete}, err
	}
	if err := provider.validateDependencies(deps); err != nil {
		return VintageDatesResult{Completeness: CompletenessIncomplete}, err
	}
	pageSize := request.PageSize
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}
	maxPages := request.MaxPages
	if maxPages == 0 {
		maxPages = provider.config.MaxPages
	}
	result := VintageDatesResult{Completeness: CompletenessIncomplete}
	for page := 0; page < maxPages; page++ {
		offset := page * pageSize
		query := url.Values{"series_id": []string{request.SeriesID}, "file_type": []string{"json"}, "sort_order": []string{"asc"}, "limit": []string{strconv.Itoa(pageSize)}, "offset": []string{strconv.Itoa(offset)}}
		if request.RealtimeStart != nil {
			query.Set("realtime_start", string(*request.RealtimeStart))
		}
		if request.RealtimeEnd != nil {
			query.Set("realtime_end", string(*request.RealtimeEnd))
		}
		endpoint, err := provider.endpoint("series", "vintagedates")
		if err != nil {
			return result, err
		}
		execution, err := provider.acquire(ctx, deps, providercontract.OperationPaginatedRead, endpoint, query)
		result.Executions = append(result.Executions, execution)
		if err != nil {
			return result, err
		}
		payloadID := request.PayloadID
		if page > 0 {
			payloadID = pagePayloadID(request.PayloadID, page)
		}
		raw, err := provider.persist(ctx, deps, execution, payloadID, request.Retention)
		if err != nil {
			return result, err
		}
		result.RawPayloads = append(result.RawPayloads, raw)
		payload, err := providercontract.RetrieveRawPayload(ctx, deps.Store, raw.Ref)
		if err != nil {
			return result, err
		}
		parsed, err := parseVintageDates(payload, request.SeriesID)
		if err != nil {
			return result, err
		}
		if parsed.Page.Offset != offset || parsed.Page.Limit != pageSize {
			return result, fmt.Errorf("FRED vintage response pagination does not match the requested page")
		}
		if result.Page.Count == 0 {
			result.Page = parsed.Page
		} else if result.Page.Count != parsed.Page.Count {
			return result, fmt.Errorf("FRED vintage response count changed across pages")
		}
		if result.ProviderRealtimePeriod == nil {
			result.ProviderRealtimePeriod = parsed.ProviderRealtimePeriod
		} else if parsed.ProviderRealtimePeriod != nil && *result.ProviderRealtimePeriod != *parsed.ProviderRealtimePeriod {
			return result, fmt.Errorf("FRED vintage realtime metadata changed across pages")
		}
		result.VintageDates = append(result.VintageDates, parsed.Dates...)
		if len(result.VintageDates) > parsed.Page.Count {
			return result, fmt.Errorf("FRED vintage response contains more dates than count")
		}
		if len(result.VintageDates) > len(parsed.Dates) {
			previous := result.VintageDates[len(result.VintageDates)-len(parsed.Dates)-1]
			if len(parsed.Dates) > 0 && parsed.Dates[0] <= previous {
				return result, fmt.Errorf("FRED vintage pages are not globally ascending or contain duplicates")
			}
		}
		if len(result.VintageDates) == parsed.Page.Count {
			result.Completeness = CompletenessComplete
			return result, nil
		}
	}
	return result, fmt.Errorf("FRED vintage acquisition incomplete: bounded page limit reached")
}

func (provider *Provider) AcquireObservations(ctx context.Context, deps Dependencies, request ObservationsRequest) (ObservationsResult, error) {
	if err := validateObservationsRequest(request); err != nil {
		return ObservationsResult{Completeness: CompletenessIncomplete}, err
	}
	if err := provider.validateDependencies(deps); err != nil {
		return ObservationsResult{Completeness: CompletenessIncomplete}, err
	}
	pageSize := request.PageSize
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}
	maxPages := request.MaxPages
	if maxPages == 0 {
		maxPages = provider.config.MaxPages
	}
	result := ObservationsResult{Completeness: CompletenessIncomplete}
	for page := 0; page < maxPages; page++ {
		offset := page * pageSize
		query, err := observationQuery(request, pageSize, offset)
		if err != nil {
			return result, err
		}
		endpoint, err := provider.endpoint("series", "observations")
		if err != nil {
			return result, err
		}
		execution, err := provider.acquire(ctx, deps, providercontract.OperationPaginatedRead, endpoint, query)
		result.Executions = append(result.Executions, execution)
		if err != nil {
			return result, err
		}
		payloadID := request.PayloadID
		if page > 0 {
			payloadID = pagePayloadID(request.PayloadID, page)
		}
		raw, err := provider.persist(ctx, deps, execution, payloadID, request.Retention)
		if err != nil {
			return result, err
		}
		result.RawPayloads = append(result.RawPayloads, raw)
		payload, err := providercontract.RetrieveRawPayload(ctx, deps.Store, raw.Ref)
		if err != nil {
			return result, err
		}
		parsed, err := normalizeObservations(payload, raw.Ref, request)
		if err != nil {
			return result, err
		}
		if parsed.Page.Offset != offset || parsed.Page.Limit != pageSize {
			return result, fmt.Errorf("FRED observation response pagination does not match the requested page")
		}
		if result.Page.Count == 0 {
			result.Page = parsed.Page
		} else if result.Page.Count != parsed.Page.Count {
			return result, fmt.Errorf("FRED observation response count changed across pages")
		}
		if result.ProviderRealtimePeriod == nil {
			result.ProviderRealtimePeriod = parsed.ProviderRealtimePeriod
		} else if parsed.ProviderRealtimePeriod != nil && *result.ProviderRealtimePeriod != *parsed.ProviderRealtimePeriod {
			return result, fmt.Errorf("FRED observation realtime metadata changed across pages")
		}
		result.Observations = append(result.Observations, parsed.Observations...)
		if len(result.Observations) > parsed.Page.Count {
			return result, fmt.Errorf("FRED observation response contains more rows than count")
		}
		if len(result.Observations) > len(parsed.Observations) {
			previous := result.Observations[len(result.Observations)-len(parsed.Observations)-1]
			if len(parsed.Observations) > 0 && parsed.Observations[0].ObservationDate <= previous.ObservationDate {
				return result, fmt.Errorf("FRED observation pages are not globally ascending or contain duplicates")
			}
		}
		if len(result.Observations) == parsed.Page.Count {
			result.Completeness = CompletenessComplete
			return result, nil
		}
	}
	return result, fmt.Errorf("FRED observation acquisition incomplete: bounded page limit reached")
}

func (provider *Provider) Acquire(ctx context.Context, deps Dependencies, request AcquisitionRequest) (AcquisitionResult, error) {
	if err := validateAcquisitionRequest(request); err != nil {
		return AcquisitionResult{}, err
	}
	seriesResult, err := provider.AcquireSeriesMetadata(ctx, deps, SeriesRequest{SeriesID: request.SeriesID, InformationState: metadataInformationState(request.InformationState), PayloadID: request.SeriesPayloadID, Retention: request.Retention})
	if err != nil {
		return AcquisitionResult{}, err
	}
	result := AcquisitionResult{Series: seriesResult.Series, RawPayloads: []providercontract.RawPayloadDescriptor{seriesResult.Raw}}
	if request.IncludeVintageDates {
		vintages, err := provider.AcquireVintageDates(ctx, deps, VintageDatesRequest{SeriesID: request.SeriesID, PayloadID: request.VintagePayloadID, Retention: request.Retention, PageSize: request.PageSize, MaxPages: request.MaxPages})
		result.VintageDates = vintages
		result.RawPayloads = append(result.RawPayloads, vintages.RawPayloads...)
		if err != nil {
			return result, err
		}
	}
	observations, err := provider.AcquireObservations(ctx, deps, ObservationsRequest{Series: seriesResult.Series, ObservationStart: request.ObservationStart, ObservationEnd: request.ObservationEnd, InformationState: request.InformationState, PayloadID: request.ObservationPayloadID, Retention: request.Retention, PageSize: request.PageSize, MaxPages: request.MaxPages})
	result.Observations = observations
	result.RawPayloads = append(result.RawPayloads, observations.RawPayloads...)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (provider *Provider) validateDependencies(deps Dependencies) error {
	if provider == nil || deps.Registry == nil || deps.Executor == nil || deps.Store == nil {
		return errors.New("FRED acquisition path is not fully configured")
	}
	return nil
}

func (provider *Provider) acquire(ctx context.Context, deps Dependencies, kind providercontract.OperationKind, endpoint string, query url.Values) (providercontract.ExecutionResult, error) {
	operation := providercontract.Operation{ContractVersion: providercontract.OperationContractV1, Provider: ProviderIdentity, CapabilityID: providercontract.CapabilityMacroObservation, Kind: kind, RetrySafety: providercontract.RetrySafetyRepeatable}
	return deps.Executor.Execute(ctx, operation, func(attemptCtx context.Context, _ providercontract.AttemptContext) providercontract.ProviderAttemptResult {
		return provider.fetch(attemptCtx, endpoint, query)
	})
}

func (provider *Provider) fetch(ctx context.Context, endpoint string, query url.Values) providercontract.ProviderAttemptResult {
	requestURL, err := url.Parse(endpoint)
	if err != nil {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: errors.New("FRED endpoint is malformed")}}
	}
	requestQuery := make(url.Values, len(query)+1)
	for key, values := range query {
		requestQuery[key] = append([]string(nil), values...)
	}
	requestQuery.Set("api_key", provider.config.APIKey)
	requestURL.RawQuery = requestQuery.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: errors.New("FRED request could not be constructed")}}
	}
	request.Header.Set("Accept", "application/json")
	response, err := provider.client.Do(request)
	if err != nil {
		classified := providercontract.ClassifyTransportError(ctx, err)
		classified.Cause = errors.New("FRED transport request failed")
		return providercontract.ProviderAttemptResult{Failure: &classified}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{HTTPStatus: response.StatusCode, RetryAfter: response.Header.Get("Retry-After")}}
	}
	mediaType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	if mediaType != "application/json" {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("FRED response media type is not application/json")}}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, provider.config.MaxResponseBytes+1))
	if err != nil {
		classified := providercontract.ClassifyTransportError(ctx, err)
		classified.Cause = errors.New("FRED response could not be read")
		return providercontract.ProviderAttemptResult{Failure: &classified}
	}
	if len(body) == 0 || int64(len(body)) > provider.config.MaxResponseBytes {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("FRED response is empty or exceeds the bounded capture policy")}}
	}
	return providercontract.ProviderAttemptResult{RawBytes: body}
}

func (provider *Provider) endpoint(parts ...string) (string, error) {
	if provider == nil || strings.TrimSpace(provider.config.BaseURL) == "" {
		return "", errors.New("FRED base URL is required")
	}
	parsed, err := url.Parse(strings.TrimRight(provider.config.BaseURL, "/"))
	if err != nil {
		return "", errors.New("FRED base URL is malformed")
	}
	parsed.Path = path.Join(parsed.Path, path.Join(parts...))
	return parsed.String(), nil
}

func (provider *Provider) persist(ctx context.Context, deps Dependencies, execution providercontract.ExecutionResult, payloadID providercontract.RawPayloadID, retention providercontract.RawPayloadRetentionPolicy) (providercontract.RawPayloadDescriptor, error) {
	revision := canonical.RevisionIdentity{Namespace: "fred.response.sha256", Value: canonical.DigestBytes(execution.RawBytes).Value}
	return providercontract.PersistRawPayload(ctx, deps.Registry, deps.Store, providercontract.RawPayloadPersistenceRequest{
		ID: payloadID, Provider: ProviderIdentity, Capability: providercontract.CapabilityMacroObservation,
		Raw: fredRawRepresentation(), Capture: providercontract.RawPayloadCapture{ByteForm: providercontract.RawPayloadByteFormEntityBody, ContentCodingState: providercontract.ContentCodingIdentity, CharacterEncoding: "utf-8"},
		Source: &SourceIdentity, Revision: &revision, ReceivedAt: execution.CompletedAt, Retention: retention, Complete: true,
	}, execution.RawBytes)
}

func observationQuery(request ObservationsRequest, pageSize, offset int) (url.Values, error) {
	query := url.Values{
		"series_id": []string{request.Series.ProviderSeriesID}, "file_type": []string{"json"},
		"units": []string{"lin"}, "output_type": []string{"1"}, "sort_order": []string{"asc"},
		"observation_start": []string{string(request.ObservationStart)}, "observation_end": []string{string(request.ObservationEnd)},
		"limit": []string{strconv.Itoa(pageSize)}, "offset": []string{strconv.Itoa(offset)},
	}
	switch request.InformationState.Mode {
	case InformationStateCurrent:
	case InformationStateAsOf:
		query.Set("realtime_start", string(*request.InformationState.Date))
		query.Set("realtime_end", string(*request.InformationState.Date))
	case InformationStateVintage:
		query.Set("vintage_dates", string(*request.InformationState.Date))
	case InformationStateInitialRelease:
		query.Set("output_type", "4")
	default:
		return nil, errors.New("unsupported FRED information state")
	}
	return query, nil
}

func seriesQuery(request SeriesRequest) (url.Values, error) {
	query := url.Values{"series_id": []string{request.SeriesID}, "file_type": []string{"json"}}
	switch request.InformationState.Mode {
	case InformationStateCurrent, InformationStateInitialRelease:
		// FRED has no series-level initial-release output. Initial-release
		// acquisitions therefore bind explicitly to current metadata.
	case InformationStateAsOf, InformationStateVintage:
		query.Set("realtime_start", string(*request.InformationState.Date))
		query.Set("realtime_end", string(*request.InformationState.Date))
	default:
		return nil, errors.New("unsupported FRED series information state")
	}
	return query, nil
}

func metadataInformationState(state InformationState) InformationState {
	if state.Mode == InformationStateInitialRelease {
		return InformationState{Mode: InformationStateCurrent}
	}
	return state
}

func pagePayloadID(base providercontract.RawPayloadID, page int) providercontract.RawPayloadID {
	return providercontract.RawPayloadID("rpa_" + canonical.DigestBytes([]byte(string(base) + "\x00page\x00" + strconv.Itoa(page))).Value[:24])
}

func validateSeriesRequest(request SeriesRequest) error {
	if err := validateSeriesID(strings.TrimSpace(request.SeriesID)); err != nil {
		return err
	}
	if strings.TrimSpace(string(request.PayloadID)) == "" {
		return errors.New("FRED series payload ID is required")
	}
	if err := request.InformationState.Validate(); err != nil {
		return err
	}
	return request.Retention.Validate()
}

func validateVintageDatesRequest(request VintageDatesRequest) error {
	if err := validateSeriesID(strings.TrimSpace(request.SeriesID)); err != nil {
		return err
	}
	if strings.TrimSpace(string(request.PayloadID)) == "" {
		return errors.New("FRED vintage payload ID is required")
	}
	if request.PageSize < 0 || request.PageSize > 10000 {
		return errors.New("FRED vintage page size is outside the documented bound")
	}
	if request.MaxPages < 0 {
		return errors.New("FRED maximum page count must not be negative")
	}
	if err := validateOptionalDate(request.RealtimeStart); err != nil {
		return err
	}
	if err := validateOptionalDate(request.RealtimeEnd); err != nil {
		return err
	}
	if request.RealtimeStart != nil && request.RealtimeEnd != nil && *request.RealtimeEnd < *request.RealtimeStart {
		return errors.New("FRED realtime end precedes realtime start")
	}
	return request.Retention.Validate()
}

func validateObservationsRequest(request ObservationsRequest) error {
	if err := request.Series.Validate(); err != nil {
		return fmt.Errorf("invalid FRED series metadata: %w", err)
	}
	if request.Series.ProviderSeriesID != strings.TrimSpace(request.Series.ProviderSeriesID) {
		return errors.New("FRED provider series ID must not contain surrounding whitespace")
	}
	if err := request.ObservationStart.Validate(); err != nil {
		return err
	}
	if err := request.ObservationEnd.Validate(); err != nil {
		return err
	}
	if request.ObservationEnd < request.ObservationStart {
		return errors.New("FRED observation end precedes observation start")
	}
	if err := request.InformationState.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(string(request.PayloadID)) == "" {
		return errors.New("FRED observation payload ID is required")
	}
	if request.PageSize < 0 || request.PageSize > 100000 {
		return errors.New("FRED observation page size is outside the documented bound")
	}
	if request.MaxPages < 0 {
		return errors.New("FRED maximum page count must not be negative")
	}
	return request.Retention.Validate()
}

func validateAcquisitionRequest(request AcquisitionRequest) error {
	if err := validateSeriesID(strings.TrimSpace(request.SeriesID)); err != nil {
		return err
	}
	if strings.TrimSpace(string(request.SeriesPayloadID)) == "" || strings.TrimSpace(string(request.ObservationPayloadID)) == "" {
		return errors.New("FRED acquisition requires series and observation payload IDs")
	}
	if request.IncludeVintageDates && strings.TrimSpace(string(request.VintagePayloadID)) == "" {
		return errors.New("FRED vintage acquisition requires a vintage payload ID")
	}
	if err := request.ObservationStart.Validate(); err != nil {
		return err
	}
	if err := request.ObservationEnd.Validate(); err != nil {
		return err
	}
	if request.ObservationEnd < request.ObservationStart {
		return errors.New("FRED observation end precedes observation start")
	}
	return request.InformationState.Validate()
}

func validateOptionalDate(value *Date) error {
	if value != nil {
		return value.Validate()
	}
	return nil
}
