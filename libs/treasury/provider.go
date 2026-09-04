package treasury

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/macroevidence"
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

func RegisterProvider(registry *providercontract.Registry) error {
	if registry == nil {
		return errors.New("Treasury provider registry is required")
	}
	return registry.Register(ProviderDefinition())
}

func ProviderDefinition() providercontract.ProviderDefinition {
	return providercontract.ProviderDefinition{
		ContractVersion:    providercontract.ProviderDefinitionV1,
		Identity:           ProviderIdentity,
		DisplayName:        "U.S. Treasury Daily Treasury Par Yield Curve Rates",
		AdapterVersion:     canonical.VersionIdentity{Namespace: "jax.treasury.adapter", Value: AdapterVersion},
		ProviderAPIVersion: &canonical.VersionIdentity{Namespace: "treasury.interest_rate_xml", Value: "documented-v1"},
		Capabilities: []providercontract.Capability{{
			ContractVersion: providercontract.CapabilityContractV1,
			ID:              providercontract.CapabilityMacroObservation,
			Category:        providercontract.DataCategoryMacroeconomicData,
			Support:         providercontract.SupportSupported,
			Raw:             rawRepresentation(),
			Authentication:  providercontract.AuthenticationRequirement{Class: providercontract.AuthenticationNone},
			Operational: providercontract.OperationalSemantics{
				DeliveryModes:      []providercontract.DeliveryMode{providercontract.DeliverySnapshot, providercontract.DeliveryHistorical},
				FreshnessModes:     []providercontract.FreshnessMode{providercontract.FreshnessEndOfDay, providercontract.FreshnessOnDemand},
				QualityRequirement: providercontract.QualityCanonicalValidationRequired,
			},
			ProviderNeutralOutputs: []providercontract.ProviderNeutralOutput{{
				ContractVersion: providercontract.ProviderNeutralOutputV1,
				Schema:          canonical.VersionIdentity{Namespace: "jax.macroevidence", Value: "macro_observation/v1"},
			}},
		}},
	}
}

func (provider *Provider) AcquireYear(ctx context.Context, deps Dependencies, request YearRequest) (YearResult, error) {
	result := YearResult{RequestedYear: request.Year, Completeness: macroevidence.CompletenessIncomplete}
	if request.Year < 1990 || request.Year > 9999 {
		return result, errors.New("Treasury yield curve year is outside the documented coverage")
	}
	if strings.TrimSpace(string(request.PayloadID)) == "" {
		return result, errors.New("Treasury yield curve payload ID is required")
	}
	if err := request.Retention.Validate(); err != nil {
		return result, err
	}
	if deps.Registry == nil || deps.Executor == nil || deps.Store == nil {
		return result, errors.New("Treasury acquisition path is not fully configured")
	}
	operation := providercontract.Operation{ContractVersion: providercontract.OperationContractV1, Provider: ProviderIdentity, CapabilityID: providercontract.CapabilityMacroObservation, Kind: providercontract.OperationReadFetch, RetrySafety: providercontract.RetrySafetyRepeatable}
	execution, err := deps.Executor.Execute(ctx, operation, func(attemptCtx context.Context, _ providercontract.AttemptContext) providercontract.ProviderAttemptResult {
		return provider.fetch(attemptCtx, request.Year)
	})
	result.Execution = execution
	if err != nil {
		return result, err
	}
	revision := canonical.RevisionIdentity{Namespace: "treasury.daily_yield_curve.response_sha256", Value: canonical.DigestBytes(execution.RawBytes).Value}
	raw, err := providercontract.PersistRawPayload(ctx, deps.Registry, deps.Store, providercontract.RawPayloadPersistenceRequest{
		ID: request.PayloadID, Provider: ProviderIdentity, Capability: providercontract.CapabilityMacroObservation, Raw: rawRepresentation(),
		Capture: providercontract.RawPayloadCapture{ByteForm: providercontract.RawPayloadByteFormEntityBody, ContentCodingState: providercontract.ContentCodingIdentity, CharacterEncoding: "utf-8"},
		Source:  &SourceIdentity, Revision: &revision, ReceivedAt: execution.CompletedAt, Retention: request.Retention, Complete: true,
	}, execution.RawBytes)
	if err != nil {
		return result, err
	}
	result.Raw = raw
	payload, err := providercontract.RetrieveRawPayload(ctx, deps.Store, raw.Ref)
	if err != nil {
		return result, err
	}
	result.Observations, err = parseYieldCurve(payload, raw.Ref, request.Year)
	if err != nil {
		return result, err
	}
	result.Completeness = macroevidence.CompletenessComplete
	return result, nil
}

func (provider *Provider) fetch(ctx context.Context, year int) providercontract.ProviderAttemptResult {
	endpoint := strings.TrimRight(provider.config.BaseURL, "/") + "/resource-center/data-chart-center/interest-rates/pages/xml"
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: errors.New("Treasury endpoint is malformed")}}
	}
	query := parsed.Query()
	query.Set("data", "daily_treasury_yield_curve")
	query.Set("field_tdr_date_value", strconv.Itoa(year))
	parsed.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: errors.New("Treasury request could not be constructed")}}
	}
	request.Header.Set("Accept", "text/xml, application/xml")
	response, err := provider.client.Do(request)
	if err != nil {
		failure := providercontract.ClassifyTransportError(ctx, err)
		failure.Cause = errors.New("Treasury transport request failed")
		return providercontract.ProviderAttemptResult{Failure: &failure}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{HTTPStatus: response.StatusCode, RetryAfter: response.Header.Get("Retry-After")}}
	}
	mediaType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	if mediaType != "text/xml" && mediaType != "application/xml" {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("Treasury response media type is not XML")}}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, provider.config.MaxResponseBytes+1))
	if err != nil {
		failure := providercontract.ClassifyTransportError(ctx, err)
		failure.Cause = errors.New("Treasury response could not be read")
		return providercontract.ProviderAttemptResult{Failure: &failure}
	}
	if len(body) == 0 || int64(len(body)) > provider.config.MaxResponseBytes {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("Treasury response is empty or exceeds the bounded capture policy")}}
	}
	return providercontract.ProviderAttemptResult{RawBytes: body}
}
