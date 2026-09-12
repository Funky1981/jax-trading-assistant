package cboe

import (
	"context"
	"errors"
	"io"
	"net/http"
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
		return errors.New("cboe provider registry is required")
	}
	return registry.Register(ProviderDefinition())
}

func ProviderDefinition() providercontract.ProviderDefinition {
	return providercontract.ProviderDefinition{
		ContractVersion: providercontract.ProviderDefinitionV1,
		Identity:        ProviderIdentity,
		DisplayName:     "Cboe VIX Index historical daily values",
		AdapterVersion:  canonical.VersionIdentity{Namespace: "jax.cboe.adapter", Value: AdapterVersion},
		Capabilities: []providercontract.Capability{{
			ContractVersion: providercontract.CapabilityContractV1,
			ID:              providercontract.CapabilityMacroObservation,
			Category:        providercontract.DataCategoryMacroeconomicData,
			Support:         providercontract.SupportSupported,
			Raw:             rawRepresentation(),
			Authentication:  providercontract.AuthenticationRequirement{Class: providercontract.AuthenticationNone},
			Operational: providercontract.OperationalSemantics{
				DeliveryModes:      []providercontract.DeliveryMode{providercontract.DeliveryHistorical},
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

func (provider *Provider) AcquireHistory(ctx context.Context, deps Dependencies, request HistoryRequest) (HistoryResult, error) {
	result := HistoryResult{Completeness: macroevidence.CompletenessIncomplete}
	if strings.TrimSpace(string(request.PayloadID)) == "" {
		return result, errors.New("cboe VIX history payload ID is required")
	}
	if err := request.Retention.Validate(); err != nil {
		return result, err
	}
	if deps.Registry == nil || deps.Executor == nil || deps.Store == nil {
		return result, errors.New("cboe acquisition path is not fully configured")
	}
	operation := providercontract.Operation{ContractVersion: providercontract.OperationContractV1, Provider: ProviderIdentity, CapabilityID: providercontract.CapabilityMacroObservation, Kind: providercontract.OperationReadFetch, RetrySafety: providercontract.RetrySafetyRepeatable}
	execution, err := deps.Executor.Execute(ctx, operation, func(attemptCtx context.Context, _ providercontract.AttemptContext) providercontract.ProviderAttemptResult {
		return provider.fetch(attemptCtx)
	})
	result.Execution = execution
	if err != nil {
		return result, err
	}
	revision := canonical.RevisionIdentity{Namespace: "cboe.vix_history.response_sha256", Value: canonical.DigestBytes(execution.RawBytes).Value}
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
	result.Series, result.Observations, err = parseHistory(payload, raw.Ref)
	if err != nil {
		return result, err
	}
	result.Completeness = macroevidence.CompletenessComplete
	return result, nil
}

func (provider *Provider) fetch(ctx context.Context) providercontract.ProviderAttemptResult {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.config.HistoryURL, nil)
	if err != nil {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: errors.New("cboe VIX request could not be constructed")}}
	}
	request.Header.Set("Accept", "text/csv")
	response, err := provider.client.Do(request)
	if err != nil {
		failure := providercontract.ClassifyTransportError(ctx, err)
		failure.Cause = errors.New("cboe VIX transport request failed")
		return providercontract.ProviderAttemptResult{Failure: &failure}
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{HTTPStatus: response.StatusCode, RetryAfter: response.Header.Get("Retry-After")}}
	}
	mediaType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	if mediaType != "text/csv" && mediaType != "application/csv" {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("cboe VIX response media type is not CSV")}}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, provider.config.MaxResponseBytes+1))
	if err != nil {
		failure := providercontract.ClassifyTransportError(ctx, err)
		failure.Cause = errors.New("cboe VIX response could not be read")
		return providercontract.ProviderAttemptResult{Failure: &failure}
	}
	if len(body) == 0 || int64(len(body)) > provider.config.MaxResponseBytes {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("cboe VIX response is empty or exceeds the bounded capture policy")}}
	}
	return providercontract.ProviderAttemptResult{RawBytes: body}
}
