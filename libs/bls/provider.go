package bls

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
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

func DefaultConfig() Config {
	return Config{CalendarURL: DefaultCalendarURL, MaxResponseBytes: DefaultMaxResponse}
}

func RegisterProvider(registry *providercontract.Registry) error {
	if registry == nil {
		return errors.New("BLS provider registry is required")
	}
	return registry.Register(ProviderDefinition())
}

func ProviderDefinition() providercontract.ProviderDefinition {
	return providercontract.ProviderDefinition{
		ContractVersion:    providercontract.ProviderDefinitionV1,
		Identity:           ProviderIdentity,
		DisplayName:        "U.S. Bureau of Labor Statistics release calendar",
		AdapterVersion:     canonical.VersionIdentity{Namespace: "jax.bls.adapter", Value: AdapterVersion},
		ProviderAPIVersion: nil,
		Capabilities: []providercontract.Capability{{
			ContractVersion: providercontract.CapabilityContractV1,
			ID:              providercontract.CapabilityEconomicCalendar,
			Category:        providercontract.DataCategoryEconomicCalendar,
			Support:         providercontract.SupportSupported,
			Raw:             rawRepresentation(),
			Authentication:  providercontract.AuthenticationRequirement{Class: providercontract.AuthenticationNone},
			Operational: providercontract.OperationalSemantics{
				DeliveryModes:      []providercontract.DeliveryMode{providercontract.DeliverySnapshot, providercontract.DeliveryHistorical},
				FreshnessModes:     []providercontract.FreshnessMode{providercontract.FreshnessOnDemand, providercontract.FreshnessPeriodic},
				QualityRequirement: providercontract.QualityCanonicalValidationRequired,
			},
			ProviderNeutralOutputs: []providercontract.ProviderNeutralOutput{{
				ContractVersion: providercontract.ProviderNeutralOutputV1,
				Schema:          canonical.VersionIdentity{Namespace: "jax.releaseevidence", Value: "economic_release/v1"},
			}},
		}},
	}
}

func rawRepresentation() providercontract.RawRepresentation {
	return providercontract.RawRepresentation{Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatStructuredMessage, Schema: canonical.VersionIdentity{Namespace: RawSchemaNamespace, Value: RawSchemaValue}, MediaType: "text/calendar"}
}

func (provider *Provider) AcquireCalendar(ctx context.Context, deps Dependencies, request CalendarRequest) (CalendarResult, error) {
	result := CalendarResult{}
	if strings.TrimSpace(string(request.PayloadID)) == "" {
		return result, errors.New("BLS calendar payload ID is required")
	}
	if err := request.Retention.Validate(); err != nil {
		return result, err
	}
	if err := validateDependencies(deps); err != nil {
		return result, err
	}
	operation := providercontract.Operation{ContractVersion: providercontract.OperationContractV1, Provider: ProviderIdentity, CapabilityID: providercontract.CapabilityEconomicCalendar, Kind: providercontract.OperationMetadataFetch, RetrySafety: providercontract.RetrySafetyRepeatable}
	execution, err := deps.Executor.Execute(ctx, operation, func(attemptCtx context.Context, _ providercontract.AttemptContext) providercontract.ProviderAttemptResult {
		return provider.fetch(attemptCtx)
	})
	result.Execution = execution
	if err != nil {
		return result, err
	}
	revision := canonical.RevisionIdentity{Namespace: "bls.calendar.response.sha256", Value: canonical.DigestBytes(execution.RawBytes).Value}
	raw, err := providercontract.PersistRawPayload(ctx, deps.Registry, deps.Store, providercontract.RawPayloadPersistenceRequest{
		ID: request.PayloadID, Provider: ProviderIdentity, Capability: providercontract.CapabilityEconomicCalendar, Raw: rawRepresentation(),
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
	result.Releases, err = parseCalendar(payload, raw.Ref)
	return result, err
}

func (provider *Provider) fetch(ctx context.Context) providercontract.ProviderAttemptResult {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.config.CalendarURL, nil)
	if err != nil {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: errors.New("BLS calendar request could not be constructed")}}
	}
	request.Header.Set("Accept", "text/calendar")
	response, err := provider.client.Do(request)
	if err != nil {
		classified := providercontract.ClassifyTransportError(ctx, err)
		classified.Cause = errors.New("BLS calendar transport request failed")
		return providercontract.ProviderAttemptResult{Failure: &classified}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{HTTPStatus: response.StatusCode, RetryAfter: response.Header.Get("Retry-After")}}
	}
	mediaType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	if mediaType != "text/calendar" {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("BLS calendar response media type is not text/calendar")}}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, provider.config.MaxResponseBytes+1))
	if err != nil {
		classified := providercontract.ClassifyTransportError(ctx, err)
		classified.Cause = errors.New("BLS calendar response could not be read")
		return providercontract.ProviderAttemptResult{Failure: &classified}
	}
	if len(body) == 0 || int64(len(body)) > provider.config.MaxResponseBytes {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("BLS calendar is empty or exceeds the bounded capture policy")}}
	}
	return providercontract.ProviderAttemptResult{RawBytes: body}
}

func validateDependencies(deps Dependencies) error {
	if deps.Registry == nil || deps.Executor == nil || deps.Store == nil {
		return fmt.Errorf("BLS calendar acquisition path is not fully configured")
	}
	return nil
}
