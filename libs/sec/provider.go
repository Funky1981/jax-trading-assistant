package sec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
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

func RegisterSECProvider(registry *providercontract.Registry) error {
	if registry == nil {
		return errors.New("SEC provider registry is required")
	}
	return registry.Register(SECProviderDefinition())
}

func RegisterSECNormalizers(registry *providercontract.NormalizerRegistry, resolver SECIdentityResolver) error {
	if registry == nil || resolver == nil {
		return errors.New("SEC normalizer registration requires registry and identity resolver")
	}
	for _, normalizer := range []providercontract.Normalizer{newSubmissionsNormalizer(resolver), newCompanyFactsNormalizer(resolver)} {
		if err := registry.Register(normalizer); err != nil {
			return err
		}
	}
	return nil
}

// NewProviderPipeline composes the accepted Phase 02 registry and pipeline.
func NewProviderPipeline(resolver SECIdentityResolver) (*providercontract.Registry, *providercontract.NormalizationPipeline, error) {
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		return nil, nil, err
	}
	if err := RegisterSECProvider(registry); err != nil {
		return nil, nil, err
	}
	normalizers, err := providercontract.NewNormalizerRegistry(registry)
	if err != nil {
		return nil, nil, err
	}
	if err := RegisterSECNormalizers(normalizers, resolver); err != nil {
		return nil, nil, err
	}
	pipeline, err := providercontract.NewNormalizationPipeline(registry, normalizers)
	if err != nil {
		return nil, nil, err
	}
	return registry, pipeline, nil
}

func (provider *Provider) fetch(ctx context.Context, endpoint string) providercontract.ProviderAttemptResult {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: err}}
	}
	userAgent, err := provider.config.Identity.userAgent()
	if err != nil {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: err}}
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "application/json")
	response, err := provider.client.Do(request)
	if err != nil {
		return providercontract.ProviderAttemptResult{Failure: ptrFailure(providercontract.ClassifyTransportError(ctx, err))}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{HTTPStatus: response.StatusCode}}
	}
	mediaType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	if mediaType != "application/json" {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureMalformedRequest, Cause: fmt.Errorf("SEC response media type is %q", mediaType)}}
	}
	limited := io.LimitReader(response.Body, provider.config.MaxResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return providercontract.ProviderAttemptResult{Failure: ptrFailure(providercontract.ClassifyTransportError(ctx, err))}
	}
	if int64(len(body)) > provider.config.MaxResponseBytes {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailurePermanentRejection, Cause: errors.New("SEC response exceeds bounded capture policy")}}
	}
	if len(body) == 0 {
		return providercontract.ProviderAttemptResult{Failure: &providercontract.ProviderFailure{Class: providercontract.FailureProviderPayloadParse, Cause: errors.New("SEC response body is empty")}}
	}
	return providercontract.ProviderAttemptResult{RawBytes: body}
}

func ptrFailure(value providercontract.ProviderFailure) *providercontract.ProviderFailure {
	return &value
}
func (provider *Provider) endpoint(parts ...string) (string, error) {
	base := strings.TrimRight(provider.config.BaseURL, "/")
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join(parsed.Path, path.Join(parts...))
	return parsed.String(), nil
}

func (provider *Provider) acquire(ctx context.Context, deps Dependencies, operation providercontract.Operation, endpoint string) (providercontract.ExecutionResult, error) {
	if provider == nil || deps.Registry == nil || deps.Executor == nil || deps.Store == nil || deps.Pipeline == nil {
		return providercontract.ExecutionResult{}, errors.New("SEC acquisition path is not fully configured")
	}
	return deps.Executor.Execute(ctx, operation, func(attemptCtx context.Context, _ providercontract.AttemptContext) providercontract.ProviderAttemptResult {
		return provider.fetch(attemptCtx, endpoint)
	})
}

func persist(ctx context.Context, registry *providercontract.Registry, store providercontract.RawPayloadStore, execution providercontract.ExecutionResult, payloadID providercontract.RawPayloadID, capability providercontract.CapabilityID, raw providercontract.RawRepresentation, source canonical.SourceIdentity, retention providercontract.RawPayloadRetentionPolicy) (providercontract.RawPayloadDescriptor, error) {
	revision := canonical.RevisionIdentity{Namespace: "sec.response.sha256", Value: canonical.DigestBytes(execution.RawBytes).Value}
	return providercontract.PersistRawPayload(ctx, registry, store, providercontract.RawPayloadPersistenceRequest{ID: payloadID, Provider: ProviderIdentity, Capability: capability, Raw: raw, Capture: providercontract.RawPayloadCapture{ByteForm: providercontract.RawPayloadByteFormEntityBody, ContentCodingState: providercontract.ContentCodingState("IDENTITY"), CharacterEncoding: "utf-8"}, Source: &source, Revision: &revision, ReceivedAt: execution.CompletedAt, Retention: retention, Complete: true}, execution.RawBytes)
}

func submissionRaw() providercontract.RawRepresentation {
	return providercontract.RawRepresentation{Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument, Schema: canonical.VersionIdentity{Namespace: SubmissionsRawSchema, Value: "documented/v1"}, MediaType: "application/json"}
}
func factsRaw() providercontract.RawRepresentation {
	return providercontract.RawRepresentation{Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument, Schema: canonical.VersionIdentity{Namespace: CompanyFactsRawSchema, Value: "documented/v1"}, MediaType: "application/json"}
}

func (provider *Provider) AcquireSubmissions(ctx context.Context, deps Dependencies, request SubmissionsRequest) (SubmissionsResult, error) {
	if err := request.Identity.Validate(); err != nil {
		return SubmissionsResult{}, err
	}
	if deps.Resolver == nil {
		return SubmissionsResult{}, errors.New("SEC canonical issuer resolver is required")
	}
	if request.PayloadID == "" {
		return SubmissionsResult{}, errors.New("SEC submissions payload ID is required")
	}
	if request.MaxHistoricalFiles < 0 {
		return SubmissionsResult{}, errors.New("SEC historical file limit must not be negative")
	}
	if request.MaxHistoricalFiles > maximumHistoricalFiles {
		return SubmissionsResult{}, fmt.Errorf("SEC historical file limit must not exceed %d", maximumHistoricalFiles)
	}
	endpoint, err := provider.endpoint("submissions", request.Identity.CIK+".json")
	if err != nil {
		return SubmissionsResult{}, err
	}
	operation := providercontract.Operation{ContractVersion: providercontract.OperationContractV1, Provider: ProviderIdentity, CapabilityID: providercontract.CapabilityCorporateFiling, Kind: providercontract.OperationReadFetch, RetrySafety: providercontract.RetrySafetyRepeatable}
	execution, err := provider.acquire(ctx, deps, operation, endpoint)
	if err != nil {
		return SubmissionsResult{Execution: execution}, err
	}
	raw, err := persist(ctx, deps.Registry, deps.Store, execution, request.PayloadID, providercontract.CapabilityCorporateFiling, submissionRaw(), SubmissionsSource, request.Retention)
	if err != nil {
		return SubmissionsResult{Execution: execution}, err
	}
	refs := []providercontract.RawPayloadDescriptor{raw}
	normalizer := newSubmissionsNormalizer(staticRequestResolver{identity: request.Identity})
	// The pipeline's registered normalizer resolves the CIK through the caller's
	// resolver. The local parser below only rehydrates source semantics after the
	// stored bytes have been verified; it never parses the HTTP response first.
	batch, err := deps.Pipeline.NormalizeBatchStored(ctx, deps.Store, providercontract.StoredNormalizationRequest{RawRef: raw.Ref, Target: canonical.ContractSchemaRef{Kind: canonical.ContractKindEvidence, Version: canonical.EvidenceContractV2}, Normalizer: normalizer.Descriptor().Component})
	if err != nil {
		return SubmissionsResult{Execution: execution, RawPayloads: refs}, err
	}
	payload, err := providercontract.RetrieveRawPayload(ctx, deps.Store, raw.Ref)
	if err != nil {
		return SubmissionsResult{Execution: execution, RawPayloads: refs}, err
	}
	parsed, err := parseSubmissions(payload, raw.Ref)
	if err != nil {
		return SubmissionsResult{Execution: execution, RawPayloads: refs}, err
	}
	filings, err := filingsForIdentities(parsed.Identities, raw.Ref, deps.Resolver)
	if err != nil {
		return SubmissionsResult{Execution: execution, RawPayloads: refs}, err
	}
	completeness := CompletenessComplete
	if len(parsed.AdditionalFiles) > 0 {
		completeness = CompletenessAdditionalFilesAvailable
		if request.IncludeHistorical {
			limit := request.MaxHistoricalFiles
			if limit == 0 {
				return SubmissionsResult{Execution: execution, RawPayloads: refs, Completeness: completeness}, errors.New("SEC submissions acquisition incomplete: historical files require an explicit bounded limit")
			}
			if limit < len(parsed.AdditionalFiles) {
				return SubmissionsResult{Execution: execution, RawPayloads: refs, Completeness: completeness}, fmt.Errorf("SEC submissions acquisition incomplete: %d historical files available, limit is %d", len(parsed.AdditionalFiles), limit)
			}
			for _, file := range parsed.AdditionalFiles {
				fileEndpoint, endpointErr := provider.endpoint("submissions", file.Name)
				if endpointErr != nil {
					return SubmissionsResult{Execution: execution, RawPayloads: refs, Completeness: completeness}, endpointErr
				}
				pageOperation := operation
				pageOperation.Kind = providercontract.OperationPaginatedRead
				pageExecution, fetchErr := provider.acquire(ctx, deps, pageOperation, fileEndpoint)
				if fetchErr != nil {
					return SubmissionsResult{Execution: execution, RawPayloads: refs, Completeness: completeness}, fetchErr
				}
				pageID := providercontract.RawPayloadID("rpa_" + canonical.DigestBytes([]byte(string(request.PayloadID) + "\x00" + file.Name)).Value[:24])
				pageRaw, persistErr := persist(ctx, deps.Registry, deps.Store, pageExecution, pageID, providercontract.CapabilityCorporateFiling, submissionRaw(), SubmissionsSource, request.Retention)
				if persistErr != nil {
					return SubmissionsResult{Execution: execution, RawPayloads: refs, Completeness: completeness}, persistErr
				}
				refs = append(refs, pageRaw)
				pageBatch, normalizeErr := deps.Pipeline.NormalizeBatchStored(ctx, deps.Store, providercontract.StoredNormalizationRequest{RawRef: pageRaw.Ref, Target: canonical.ContractSchemaRef{Kind: canonical.ContractKindEvidence, Version: canonical.EvidenceContractV2}, Normalizer: normalizer.Descriptor().Component})
				if normalizeErr != nil {
					return SubmissionsResult{Execution: execution, RawPayloads: refs, Completeness: completeness}, normalizeErr
				}
				pagePayload, retrieveErr := providercontract.RetrieveRawPayload(ctx, deps.Store, pageRaw.Ref)
				if retrieveErr != nil {
					return SubmissionsResult{Execution: execution, RawPayloads: refs, Completeness: completeness}, retrieveErr
				}
				pageParsed, parseErr := parseSubmissions(pagePayload, pageRaw.Ref)
				if parseErr != nil {
					return SubmissionsResult{Execution: execution, RawPayloads: refs, Completeness: completeness}, parseErr
				}
				pageFilings, parseErr := filingsForIdentities(pageParsed.Identities, pageRaw.Ref, deps.Resolver)
				if parseErr != nil {
					return SubmissionsResult{Execution: execution, RawPayloads: refs, Completeness: completeness}, parseErr
				}
				filings = append(filings, pageFilings...)
				_ = batch
				_ = pageBatch
			}
			completeness = CompletenessComplete
		}
	}
	return SubmissionsResult{Execution: execution, RawPayloads: refs, Filings: filings, Completeness: completeness}, nil
}

func (provider *Provider) AcquireCompanyFacts(ctx context.Context, deps Dependencies, request CompanyFactsRequest) (CompanyFactsResult, error) {
	if err := request.Identity.Validate(); err != nil {
		return CompanyFactsResult{}, err
	}
	if deps.Resolver == nil {
		return CompanyFactsResult{}, errors.New("SEC canonical issuer resolver is required")
	}
	if request.PayloadID == "" {
		return CompanyFactsResult{}, errors.New("SEC company facts payload ID is required")
	}
	endpoint, err := provider.endpoint("api", "xbrl", "companyfacts", request.Identity.CIK+".json")
	if err != nil {
		return CompanyFactsResult{}, err
	}
	operation := providercontract.Operation{ContractVersion: providercontract.OperationContractV1, Provider: ProviderIdentity, CapabilityID: providercontract.CapabilityFundamentalObservation, Kind: providercontract.OperationReadFetch, RetrySafety: providercontract.RetrySafetyRepeatable}
	execution, err := provider.acquire(ctx, deps, operation, endpoint)
	if err != nil {
		return CompanyFactsResult{Execution: execution}, err
	}
	raw, err := persist(ctx, deps.Registry, deps.Store, execution, request.PayloadID, providercontract.CapabilityFundamentalObservation, factsRaw(), CompanyFactsSource, request.Retention)
	if err != nil {
		return CompanyFactsResult{Execution: execution}, err
	}
	normalizer := newCompanyFactsNormalizer(staticRequestResolver{identity: request.Identity})
	batch, err := deps.Pipeline.NormalizeBatchStored(ctx, deps.Store, providercontract.StoredNormalizationRequest{RawRef: raw.Ref, Target: canonical.ContractSchemaRef{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2}, Normalizer: normalizer.Descriptor().Component})
	if err != nil {
		return CompanyFactsResult{Execution: execution, Raw: raw}, err
	}
	payload, err := providercontract.RetrieveRawPayload(ctx, deps.Store, raw.Ref)
	if err != nil {
		return CompanyFactsResult{Execution: execution, Raw: raw}, err
	}
	facts, err := parseCompanyFacts(payload, raw.Ref, request.Identity, deps.Resolver)
	if err != nil {
		return CompanyFactsResult{Execution: execution, Raw: raw}, err
	}
	if len(batch.Records) != len(facts) {
		return CompanyFactsResult{Execution: execution, Raw: raw}, fmt.Errorf("SEC normalizer output count does not match preserved fact semantics")
	}
	return CompanyFactsResult{Execution: execution, Raw: raw, Facts: facts}, nil
}
