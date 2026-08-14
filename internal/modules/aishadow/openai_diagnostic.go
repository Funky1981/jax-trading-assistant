package aishadow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	OpenAIDiagnosticProvider                   = "openai"
	OpenAIDiagnosticSolModel                   = "gpt-5.6-sol"
	OpenAIDiagnosticLunaModel                  = "gpt-5.6-luna"
	OpenAIDiagnosticModel                      = OpenAIDiagnosticSolModel
	OpenAIDiagnosticAPIKeyEnv                  = "JAX_OPENAI_EXPERIMENT_API_KEY"
	OpenAIDiagnosticEndpoint                   = "https://api.openai.com/v1/responses"
	OpenAIDiagnosticReasoningEffort            = "none"
	OpenAIDiagnosticStructuredOutput           = "v4-prompt-contract-plain-text"
	OpenAIDiagnosticEvidenceNamespace          = "openai-hosted-a1-v1"
	OpenAIDiagnosticInferenceAuthEnv           = "JAX_AI_HOSTED_INFERENCE_AUTHORIZED"
	OpenAIDiagnosticContractModeEnv            = "JAX_AI_OUTPUT_CONTRACT_MODE"
	OpenAIDiagnosticExperimentID               = "A1"
	OpenAIStructuredOutputsExperimentID        = "WP-00.03C1B"
	OpenAIStructuredOutputsEvidenceNamespace   = "openai-hosted-c1b-structured-outputs-v1"
	OpenAIStructuredOutputsMode                = "openai-responses-json-schema-strict"
	OpenAIStructuredOutputsSchemaName          = "jax_ai_shadow_output_v4_issuer_resolution"
	OpenAIStructuredOutputsServiceTier         = "default"
	OpenAIDiagnosticMaximumBudgetMicros        = int64(1_000_000)
	OpenAIDiagnosticLunaMaximumBudgetMicros    = int64(120_000)
	OpenAIStructuredOutputsMaximumBudgetMicros = int64(200_000)
	OpenAIDiagnosticPricingSource              = "https://developers.openai.com/api/docs/pricing; execution-time configuration; re-verify immediately before paid execution"
)

type OpenAIOutputContractMode string

const (
	OpenAIOutputContractPromptOnly       OpenAIOutputContractMode = "prompt-only"
	OpenAIOutputContractStrictJSONSchema OpenAIOutputContractMode = OpenAIStructuredOutputsMode
)

var experimentIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// APISecret deliberately has no exported value. Formatting and JSON encoding
// cannot reveal the credential, while the provider can still build its header.
type APISecret struct {
	value string
}

func (APISecret) String() string { return "<redacted>" }

func (APISecret) GoString() string { return "<redacted>" }

func (APISecret) MarshalJSON() ([]byte, error) { return []byte(`"<redacted>"`), nil }

func (s APISecret) authorizationHeader() string { return "Bearer " + s.value }

func (s APISecret) present() bool { return s.value != "" }

type OpenAIDiagnosticConfig struct {
	Runtime                          Config
	APIKey                           APISecret
	ExperimentID                     string
	ReasoningEffort                  string
	MaxOutputTokens                  int
	BudgetCeilingMicros              int64
	InputPriceMicrosPerMillion       int64
	CachedInputPriceMicrosPerMillion int64
	CacheWritePriceMicrosPerMillion  int64
	OutputPriceMicrosPerMillion      int64
	InferenceExplicitlyAuthorized    bool
	OutputContractMode               OpenAIOutputContractMode
}

func LoadOpenAIDiagnosticConfig(lookup func(string) (string, bool)) (OpenAIDiagnosticConfig, error) {
	profile, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileOriginal)
	return LoadOpenAIDiagnosticConfigForProfile(lookup, profile, true)
}

func LoadOpenAIDiagnosticConfigForProfile(lookup func(string) (string, bool), profile DiagnosticEvaluationProfile, requireCredential bool) (OpenAIDiagnosticConfig, error) {
	required := []string{
		"JAX_AI_SHADOW_ENABLED", "JAX_AI_PROVIDER", "JAX_AI_MODEL", "JAX_AI_TIMEOUT_SECONDS", "JAX_AI_MAX_EVENTS",
		"JAX_AI_EXPERIMENT_ID", "JAX_AI_REASONING_EFFORT", "JAX_AI_MAX_OUTPUT_TOKENS",
		"JAX_AI_EXPERIMENT_BUDGET_USD", "JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS", "JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS",
		"JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS", "JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS",
	}
	if requireCredential {
		required = append(required, OpenAIDiagnosticAPIKeyEnv)
	}
	values := make(map[string]string, len(required))
	for _, key := range required {
		value, ok := lookup(key)
		if !ok || strings.TrimSpace(value) == "" {
			return OpenAIDiagnosticConfig{}, fmt.Errorf("missing required hosted diagnostic configuration: %s", key)
		}
		values[key] = strings.TrimSpace(value)
	}
	enabled, err := strconv.ParseBool(values["JAX_AI_SHADOW_ENABLED"])
	if err != nil || !enabled {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("JAX_AI_SHADOW_ENABLED must explicitly be true")
	}
	if provider := strings.ToLower(values["JAX_AI_PROVIDER"]); provider != OpenAIDiagnosticProvider {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("hosted issuer diagnostic requires JAX_AI_PROVIDER=%s", OpenAIDiagnosticProvider)
	}
	model := values["JAX_AI_MODEL"]
	if !supportedOpenAIDiagnosticModel(model) {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("hosted issuer diagnostic requires JAX_AI_MODEL=%s or %s", OpenAIDiagnosticSolModel, OpenAIDiagnosticLunaModel)
	}
	experimentID := values["JAX_AI_EXPERIMENT_ID"]
	if !experimentIDPattern.MatchString(experimentID) {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("hosted issuer diagnostic experiment identity is invalid")
	}
	contractMode := OpenAIOutputContractPromptOnly
	if raw, ok := lookup(OpenAIDiagnosticContractModeEnv); ok && strings.TrimSpace(raw) != "" {
		contractMode = OpenAIOutputContractMode(strings.TrimSpace(raw))
	}
	if err := validateOpenAIExperimentCell(experimentID, model, contractMode); err != nil {
		return OpenAIDiagnosticConfig{}, err
	}
	if profile.RequiredProvider != "" && (profile.RequiredProvider != OpenAIDiagnosticProvider || profile.RequiredModel != model ||
		profile.RequiredExperimentID != experimentID || profile.RequiredOutputContractMode != contractMode) {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("frozen profile %s requires provider=%s model=%s experiment=%s contract=%s", profile.Identity, profile.RequiredProvider, profile.RequiredModel, profile.RequiredExperimentID, profile.RequiredOutputContractMode)
	}
	if !requireCredential && !profile.CredentiallessPreflightAllowed {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("frozen profile %s does not allow credentialless preflight", profile.Identity)
	}
	if values["JAX_AI_REASONING_EFFORT"] != OpenAIDiagnosticReasoningEffort {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("OpenAI issuer diagnostic requires JAX_AI_REASONING_EFFORT=%s", OpenAIDiagnosticReasoningEffort)
	}
	timeoutSeconds, err := strconv.Atoi(values["JAX_AI_TIMEOUT_SECONDS"])
	if err != nil || timeoutSeconds < 1 || timeoutSeconds > 600 {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("JAX_AI_TIMEOUT_SECONDS must be between 1 and 600")
	}
	maxEvents, err := strconv.Atoi(values["JAX_AI_MAX_EVENTS"])
	if err != nil || maxEvents != profile.CaseCount {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("hosted issuer diagnostic profile %s requires JAX_AI_MAX_EVENTS=%d", profile.Identity, profile.CaseCount)
	}
	maxOutputTokens, err := strconv.Atoi(values["JAX_AI_MAX_OUTPUT_TOKENS"])
	if err != nil || maxOutputTokens < 1 || maxOutputTokens > 4096 {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("JAX_AI_MAX_OUTPUT_TOKENS must be between 1 and 4096")
	}
	budget, err := parseUSDMicros(values["JAX_AI_EXPERIMENT_BUDGET_USD"])
	maximumBudget := openAIDiagnosticMaximumBudgetMicros(model, experimentID)
	if profile.MaximumBudgetMicros > 0 {
		maximumBudget = profile.MaximumBudgetMicros
	}
	if err != nil || budget <= 0 || budget > maximumBudget {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("JAX_AI_EXPERIMENT_BUDGET_USD must be positive and no greater than %s for %s", formatUSDMicros(maximumBudget), model)
	}
	inputPrice, err := parseUSDMicros(values["JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS"])
	if err != nil || inputPrice <= 0 {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS must be a positive decimal")
	}
	outputPrice, err := parseUSDMicros(values["JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS"])
	if err != nil || outputPrice <= 0 {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS must be a positive decimal")
	}
	cachedInputPrice, err := parseUSDMicros(values["JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS"])
	if err != nil || cachedInputPrice <= 0 || cachedInputPrice > inputPrice {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS must be positive and no greater than the input price")
	}
	cacheWritePrice, err := parseUSDMicros(values["JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS"])
	if err != nil || cacheWritePrice <= 0 {
		return OpenAIDiagnosticConfig{}, fmt.Errorf("JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS must be a positive decimal")
	}
	authorized := false
	if value, ok := lookup(OpenAIDiagnosticInferenceAuthEnv); ok && strings.TrimSpace(value) != "" {
		authorized, err = strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return OpenAIDiagnosticConfig{}, fmt.Errorf("%s must be true or false", OpenAIDiagnosticInferenceAuthEnv)
		}
	}
	return OpenAIDiagnosticConfig{
		Runtime: Config{
			Enabled: true, Provider: OpenAIDiagnosticProvider, Model: model,
			BaseURL: "https://api.openai.com", Timeout: time.Duration(timeoutSeconds) * time.Second, MaxEvents: maxEvents,
		},
		APIKey: APISecret{value: values[OpenAIDiagnosticAPIKeyEnv]}, ExperimentID: values["JAX_AI_EXPERIMENT_ID"],
		ReasoningEffort: values["JAX_AI_REASONING_EFFORT"], MaxOutputTokens: maxOutputTokens,
		BudgetCeilingMicros: budget, InputPriceMicrosPerMillion: inputPrice,
		CachedInputPriceMicrosPerMillion: cachedInputPrice, CacheWritePriceMicrosPerMillion: cacheWritePrice,
		OutputPriceMicrosPerMillion:   outputPrice,
		InferenceExplicitlyAuthorized: authorized,
		OutputContractMode:            contractMode,
	}, nil
}

func validateOpenAIExperimentCell(experimentID, model string, mode OpenAIOutputContractMode) error {
	switch experimentID {
	case OpenAIDiagnosticExperimentID:
		if mode != OpenAIOutputContractPromptOnly {
			return fmt.Errorf("existing OpenAI A1 cell requires %s=%s", OpenAIDiagnosticContractModeEnv, OpenAIOutputContractPromptOnly)
		}
	case OpenAIStructuredOutputsExperimentID, OpenAIGeneralizationExperimentID, OpenAIBoundaryExperimentID:
		if model != OpenAIDiagnosticLunaModel || mode != OpenAIOutputContractStrictJSONSchema {
			return fmt.Errorf("%s requires model=%s and %s=%s", OpenAIStructuredOutputsExperimentID, OpenAIDiagnosticLunaModel, OpenAIDiagnosticContractModeEnv, OpenAIOutputContractStrictJSONSchema)
		}
	default:
		return fmt.Errorf("unsupported OpenAI diagnostic experiment %q", experimentID)
	}
	return nil
}

func supportedOpenAIDiagnosticModel(model string) bool {
	return model == OpenAIDiagnosticSolModel || model == OpenAIDiagnosticLunaModel
}

func openAIDiagnosticMaximumBudgetMicros(model, experimentID string) int64 {
	if experimentID == OpenAIBoundaryExperimentID {
		return 100_000
	}
	if experimentID == OpenAIStructuredOutputsExperimentID || experimentID == OpenAIGeneralizationExperimentID {
		return OpenAIStructuredOutputsMaximumBudgetMicros
	}
	if model == OpenAIDiagnosticLunaModel {
		return OpenAIDiagnosticLunaMaximumBudgetMicros
	}
	return OpenAIDiagnosticMaximumBudgetMicros
}

func (c OpenAIDiagnosticConfig) EvidenceNamespace() string {
	switch c.ExperimentID {
	case OpenAIStructuredOutputsExperimentID:
		return OpenAIStructuredOutputsEvidenceNamespace
	case OpenAIGeneralizationExperimentID:
		return OpenAIGeneralizationEvidenceNamespace
	case OpenAIBoundaryExperimentID:
		return OpenAIBoundaryEvidenceNamespace
	}
	return OpenAIDiagnosticEvidenceNamespace
}

func (c OpenAIDiagnosticConfig) StructuredOutputMode() string {
	if c.OutputContractMode == OpenAIOutputContractStrictJSONSchema {
		return OpenAIStructuredOutputsMode
	}
	return OpenAIDiagnosticStructuredOutput
}

func (c OpenAIDiagnosticConfig) StructuredOutputsEnabled() bool {
	return c.OutputContractMode == OpenAIOutputContractStrictJSONSchema
}

func (c OpenAIDiagnosticConfig) ServiceTier() string {
	if (c.ExperimentID == OpenAIStructuredOutputsExperimentID || c.ExperimentID == OpenAIGeneralizationExperimentID || c.ExperimentID == OpenAIBoundaryExperimentID) && c.StructuredOutputsEnabled() {
		return OpenAIStructuredOutputsServiceTier
	}
	return ""
}

func openAIReturnedModelMatchesRequested(requested, returned string) bool {
	switch requested {
	case OpenAIDiagnosticLunaModel:
		return returned == requested
	case OpenAIDiagnosticSolModel:
		return returned == requested || strings.HasPrefix(returned, requested+"-")
	default:
		return false
	}
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type HostedProviderFailure struct {
	RequestNumber  int    `json:"request_number"`
	EventID        string `json:"event_id,omitempty"`
	AttemptNumber  int    `json:"attempt_number,omitempty"`
	Kind           string `json:"kind"`
	HTTPStatus     int    `json:"http_status,omitempty"`
	ProviderType   string `json:"provider_type,omitempty"`
	ProviderCode   string `json:"provider_code,omitempty"`
	RequestID      string `json:"request_id,omitempty"`
	Timeout        bool   `json:"timeout"`
	AmbiguousSpend bool   `json:"ambiguous_spend"`
}

type HostedExperimentSnapshot struct {
	ExperimentID              string                  `json:"experiment_id"`
	Provider                  string                  `json:"provider"`
	RequestedModel            string                  `json:"requested_model"`
	ReturnedModels            []string                `json:"returned_models"`
	SystemFingerprints        []string                `json:"system_fingerprints,omitempty"`
	FinishReasons             []string                `json:"finish_reasons,omitempty"`
	ReasoningEffort           string                  `json:"reasoning_effort"`
	ServiceTier               string                  `json:"service_tier,omitempty"`
	ThinkingMode              string                  `json:"thinking_mode,omitempty"`
	StructuredOutputMode      string                  `json:"structured_output_mode"`
	MaxOutputTokensPerRequest int                     `json:"max_output_tokens_per_request"`
	Pricing                   HostedPricingPlan       `json:"pricing"`
	BudgetCeilingUSD          string                  `json:"budget_ceiling_usd"`
	ActualCalculableCostUSD   string                  `json:"actual_calculable_cost_usd"`
	CostByCategory            HostedCostBreakdown     `json:"cost_by_category"`
	AmbiguousLiabilityUSD     string                  `json:"ambiguous_liability_usd"`
	AccountedCostUSD          string                  `json:"accounted_cost_usd"`
	RemainingBudgetUSD        string                  `json:"remaining_experiment_budget_usd"`
	RequestCount              int                     `json:"request_count"`
	RetryCount                int                     `json:"retry_count"`
	Usage                     ProviderUsage           `json:"usage"`
	ProviderErrorCount        int                     `json:"provider_error_count"`
	TimeoutCount              int                     `json:"timeout_count"`
	BudgetRejectionCount      int                     `json:"budget_rejection_count"`
	StopReason                string                  `json:"stop_reason,omitempty"`
	Failures                  []HostedProviderFailure `json:"failures"`
}

type HostedCostBreakdown struct {
	UncachedInputUSD string `json:"uncached_input_usd"`
	CachedInputUSD   string `json:"cached_input_usd"`
	CacheWriteUSD    string `json:"cache_write_usd"`
	OutputUSD        string `json:"output_usd"`
	TotalUSD         string `json:"total_usd"`
}

type HostedPricingPlan struct {
	InputUSDPerMillionTokens          string `json:"input_usd_per_million_tokens"`
	CachedInputUSDPerMillionTokens    string `json:"cached_input_usd_per_million_tokens"`
	CacheMissInputUSDPerMillionTokens string `json:"cache_miss_input_usd_per_million_tokens,omitempty"`
	CacheWriteUSDPerMillionTokens     string `json:"cache_write_usd_per_million_tokens"`
	OutputUSDPerMillionTokens         string `json:"output_usd_per_million_tokens"`
	Source                            string `json:"source"`
}

type hostedExperimentRecorder interface {
	ExperimentSnapshot() HostedExperimentSnapshot
}

type OpenAIDiagnosticClient struct {
	config       OpenAIDiagnosticConfig
	http         HTTPDoer
	budget       *experimentBudget
	mu           sync.Mutex
	returned     map[string]bool
	fingerprints map[string]bool
	failures     []HostedProviderFailure
	requests     int
	retries      int
	timeouts     int
	rejections   int
	stopReason   string
}

func NewOpenAIDiagnosticClient(config OpenAIDiagnosticConfig, transport HTTPDoer) *OpenAIDiagnosticClient {
	if transport == nil {
		transport = &http.Client{Timeout: config.Runtime.Timeout}
	}
	return &OpenAIDiagnosticClient{
		config: config, http: transport, returned: map[string]bool{}, fingerprints: map[string]bool{}, failures: []HostedProviderFailure{},
		budget: newExperimentBudget(config.BudgetCeilingMicros, config.InputPriceMicrosPerMillion, config.CachedInputPriceMicrosPerMillion, config.CacheWritePriceMicrosPerMillion, config.OutputPriceMicrosPerMillion),
	}
}

func (c *OpenAIDiagnosticClient) Complete(request ProviderRequest) (ProviderResponse, error) {
	if !supportedOpenAIDiagnosticModel(c.config.Runtime.Model) {
		return ProviderResponse{}, providerSafeError{kind: "configured_model", fatal: true}
	}
	if err := validateOpenAIExperimentCell(c.config.ExperimentID, c.config.Runtime.Model, c.config.OutputContractMode); err != nil {
		return ProviderResponse{}, providerSafeError{kind: "experiment_cell", fatal: true}
	}
	raw, err := marshalOpenAIDiagnosticRequest(c.config, request)
	if err != nil {
		return ProviderResponse{}, providerSafeError{kind: "output_contract", fatal: true}
	}
	estimatedInputTokens := estimatedOpenAIInputTokens(c.config, request, len(raw))
	estimatedCost := c.budget.estimateCost(estimatedInputTokens, c.config.MaxOutputTokens)
	requestNumber, err := c.reserve(request, estimatedCost)
	if err != nil {
		return ProviderResponse{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Runtime.Timeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, OpenAIDiagnosticEndpoint, bytes.NewReader(raw))
	if err != nil {
		c.budget.releaseReservation(estimatedCost)
		return ProviderResponse{}, fmt.Errorf("create hosted diagnostic request")
	}
	httpRequest.Header.Set("Authorization", c.config.APIKey.authorizationHeader())
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(httpRequest)
	if err != nil {
		ambiguous := true
		c.budget.finishFailure(estimatedCost, ambiguous)
		timedOut := errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded)
		failure := HostedProviderFailure{RequestNumber: requestNumber, EventID: request.EventID, AttemptNumber: request.AttemptNumber, Kind: "transport", Timeout: timedOut, AmbiguousSpend: true}
		c.recordFailure(failure)
		return ProviderResponse{}, providerSafeError{kind: "transport", timeout: timedOut, ambiguous: true, fatal: true}
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	requestID := response.Header.Get("x-request-id")
	requestID = c.sanitize(requestID)
	if readErr != nil {
		c.budget.finishFailure(estimatedCost, true)
		c.recordFailure(HostedProviderFailure{RequestNumber: requestNumber, EventID: request.EventID, AttemptNumber: request.AttemptNumber, Kind: "response_read", HTTPStatus: response.StatusCode, RequestID: requestID, AmbiguousSpend: true})
		return ProviderResponse{}, providerSafeError{kind: "response_read", status: response.StatusCode, requestID: requestID, ambiguous: true, fatal: true}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		failure := decodeOpenAIError(body)
		failure.RequestNumber, failure.EventID, failure.AttemptNumber = requestNumber, request.EventID, request.AttemptNumber
		failure.HTTPStatus, failure.RequestID = response.StatusCode, requestID
		failure.Kind = "http"
		failure.ProviderType = c.sanitize(failure.ProviderType)
		failure.ProviderCode = c.sanitize(failure.ProviderCode)
		failure.AmbiguousSpend = response.StatusCode == http.StatusRequestTimeout || response.StatusCode >= 500
		c.budget.finishFailure(estimatedCost, failure.AmbiguousSpend)
		c.recordFailure(failure)
		return ProviderResponse{}, providerSafeError{kind: "http", status: response.StatusCode, providerType: failure.ProviderType, providerCode: failure.ProviderCode, requestID: requestID, ambiguous: failure.AmbiguousSpend, fatal: true}
	}
	var decoded openAIResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		c.budget.finishFailure(estimatedCost, true)
		c.recordFailure(HostedProviderFailure{RequestNumber: requestNumber, EventID: request.EventID, AttemptNumber: request.AttemptNumber, Kind: "decode", HTTPStatus: response.StatusCode, RequestID: requestID, AmbiguousSpend: true})
		return ProviderResponse{}, providerSafeError{kind: "decode", status: response.StatusCode, requestID: requestID, ambiguous: true, fatal: true}
	}
	usage := decoded.Usage.providerUsage()
	if !validProviderUsage(usage) {
		c.budget.finishFailure(estimatedCost, true)
		c.recordFailure(HostedProviderFailure{RequestNumber: requestNumber, EventID: request.EventID, AttemptNumber: request.AttemptNumber, Kind: "usage", HTTPStatus: response.StatusCode, RequestID: requestID, AmbiguousSpend: true})
		return ProviderResponse{}, providerSafeError{kind: "usage", status: response.StatusCode, requestID: requestID, ambiguous: true, fatal: true}
	}
	if err := c.budget.finishSuccess(estimatedCost, usage); err != nil {
		c.setStopReason("budget_exceeded_after_reported_usage")
		return ProviderResponse{}, err
	}
	decoded.Model = c.sanitize(decoded.Model)
	if decoded.Model != "" {
		c.mu.Lock()
		c.returned[decoded.Model] = true
		c.mu.Unlock()
	}
	if !openAIReturnedModelMatchesRequested(c.config.Runtime.Model, decoded.Model) {
		c.recordFailure(HostedProviderFailure{RequestNumber: requestNumber, EventID: request.EventID, AttemptNumber: request.AttemptNumber, Kind: "model_identity", HTTPStatus: response.StatusCode, RequestID: requestID})
		return ProviderResponse{}, providerSafeError{kind: "model_identity", status: response.StatusCode, requestID: requestID, fatal: true}
	}
	decoded.SystemFingerprint = c.sanitize(decoded.SystemFingerprint)
	if decoded.SystemFingerprint != "" {
		c.mu.Lock()
		c.fingerprints[decoded.SystemFingerprint] = true
		c.mu.Unlock()
	}
	content, contentErr := decoded.outputText()
	content = c.sanitize(content)
	if contentErr != nil || decoded.Status != "completed" {
		kind := "response_status"
		if contentErr != nil {
			kind = "response_content"
		}
		c.recordFailure(HostedProviderFailure{RequestNumber: requestNumber, EventID: request.EventID, AttemptNumber: request.AttemptNumber, Kind: kind, HTTPStatus: response.StatusCode, RequestID: requestID})
		return ProviderResponse{}, providerSafeError{kind: kind, status: response.StatusCode, requestID: requestID, fatal: true}
	}
	return ProviderResponse{Content: content, ModelIdentifier: decoded.Model, RequestID: requestID, ResponseID: decoded.ID, Status: decoded.Status, SystemFingerprint: decoded.SystemFingerprint, Usage: usage}, nil
}

func (c *OpenAIDiagnosticClient) sanitize(value string) string {
	if c.config.APIKey.value == "" {
		return value
	}
	return strings.ReplaceAll(value, c.config.APIKey.value, "<redacted>")
}

func (c *OpenAIDiagnosticClient) reserve(request ProviderRequest, estimatedCost int64) (int, error) {
	if err := c.budget.reserve(estimatedCost); err != nil {
		c.mu.Lock()
		c.rejections++
		c.stopReason = "budget_guard_rejected_request"
		c.mu.Unlock()
		return 0, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests++
	if request.RequestKind == "corrective" {
		c.retries++
	}
	return c.requests, nil
}

func (c *OpenAIDiagnosticClient) recordFailure(failure HostedProviderFailure) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = append(c.failures, failure)
	if failure.Timeout {
		c.timeouts++
	}
}

func (c *OpenAIDiagnosticClient) setStopReason(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopReason = reason
}

func (c *OpenAIDiagnosticClient) ExperimentSnapshot() HostedExperimentSnapshot {
	budget := c.budget.snapshot()
	c.mu.Lock()
	defer c.mu.Unlock()
	models := make([]string, 0, len(c.returned))
	for model := range c.returned {
		models = append(models, model)
	}
	sort.Strings(models)
	fingerprints := sortedKeys(c.fingerprints)
	return HostedExperimentSnapshot{
		ExperimentID: c.config.ExperimentID, Provider: OpenAIDiagnosticProvider, RequestedModel: c.config.Runtime.Model,
		ReturnedModels: models, SystemFingerprints: fingerprints, ReasoningEffort: c.config.ReasoningEffort, ServiceTier: c.config.ServiceTier(), StructuredOutputMode: c.config.StructuredOutputMode(),
		MaxOutputTokensPerRequest: c.config.MaxOutputTokens,
		Pricing: HostedPricingPlan{
			InputUSDPerMillionTokens:       formatUSDMicros(c.config.InputPriceMicrosPerMillion),
			CachedInputUSDPerMillionTokens: formatUSDMicros(c.config.CachedInputPriceMicrosPerMillion),
			CacheWriteUSDPerMillionTokens:  formatUSDMicros(c.config.CacheWritePriceMicrosPerMillion),
			OutputUSDPerMillionTokens:      formatUSDMicros(c.config.OutputPriceMicrosPerMillion),
			Source:                         OpenAIDiagnosticPricingSource,
		},
		BudgetCeilingUSD: formatUSDMicros(c.config.BudgetCeilingMicros), ActualCalculableCostUSD: formatUSDMicros(budget.actualMicros),
		CostByCategory: budget.costs, AmbiguousLiabilityUSD: formatUSDMicros(budget.ambiguousMicros),
		AccountedCostUSD: formatUSDMicros(budget.actualMicros + budget.ambiguousMicros), RemainingBudgetUSD: formatUSDMicros(budget.remainingMicros),
		RequestCount: c.requests, RetryCount: c.retries, Usage: budget.usage, ProviderErrorCount: len(c.failures), TimeoutCount: c.timeouts,
		BudgetRejectionCount: c.rejections, StopReason: c.stopReason, Failures: append([]HostedProviderFailure(nil), c.failures...),
	}
}

type openAIInputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type openAITextConfiguration struct {
	Format openAIResponseFormat `json:"format"`
}

type openAIDiagnosticRequest struct {
	Model           string                   `json:"model"`
	Input           []openAIInputMessage     `json:"input"`
	Reasoning       map[string]string        `json:"reasoning"`
	MaxOutputTokens int                      `json:"max_output_tokens"`
	Store           bool                     `json:"store"`
	Text            *openAITextConfiguration `json:"text,omitempty"`
	ServiceTier     string                   `json:"service_tier,omitempty"`
}

func marshalOpenAIDiagnosticRequest(config OpenAIDiagnosticConfig, request ProviderRequest) ([]byte, error) {
	payload := openAIDiagnosticRequest{
		Model:     config.Runtime.Model,
		Input:     []openAIInputMessage{{Role: "system", Content: request.System}, {Role: "user", Content: request.User}},
		Reasoning: map[string]string{"effort": config.ReasoningEffort}, MaxOutputTokens: config.MaxOutputTokens, Store: false,
	}
	if config.StructuredOutputsEnabled() {
		if err := validateOpenAIStructuredOutputSchema(request.Schema); err != nil {
			return nil, err
		}
		payload.Text = &openAITextConfiguration{Format: openAIResponseFormat{
			Type: "json_schema", Name: OpenAIStructuredOutputsSchemaName, Strict: true, Schema: request.Schema,
		}}
		payload.ServiceTier = config.ServiceTier()
	}
	return json.Marshal(payload)
}

func estimatedOpenAIInputTokens(config OpenAIDiagnosticConfig, request ProviderRequest, wireBytes int) int {
	if config.StructuredOutputsEnabled() {
		return wireBytes + 1024
	}
	return len([]byte(request.System)) + len([]byte(request.User)) + 1024
}

func validateOpenAIStructuredOutputSchema(schema map[string]any) error {
	if schema == nil {
		return fmt.Errorf("Structured Outputs require ProviderRequest.Schema")
	}
	for keyword := range schema {
		if !map[string]bool{"type": true, "properties": true, "required": true, "additionalProperties": true}[keyword] {
			return fmt.Errorf("Structured Outputs root contains unsupported schema keyword %q", keyword)
		}
	}
	if schema["type"] != "object" {
		return fmt.Errorf("Structured Outputs require a root object schema")
	}
	additional, ok := schema["additionalProperties"].(bool)
	if !ok || additional {
		return fmt.Errorf("Structured Outputs require additionalProperties=false")
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok || len(properties) == 0 {
		return fmt.Errorf("Structured Outputs require object properties")
	}
	required, ok := schema["required"].([]string)
	if !ok || len(required) != len(properties) {
		return fmt.Errorf("Structured Outputs require every property")
	}
	requiredSet := make(map[string]bool, len(required))
	for _, name := range required {
		requiredSet[name] = true
	}
	for name, property := range properties {
		if !requiredSet[name] {
			return fmt.Errorf("Structured Outputs property %q is not required", name)
		}
		if err := validateOpenAISchemaNode(property); err != nil {
			return fmt.Errorf("Structured Outputs property %q: %w", name, err)
		}
	}
	return nil
}

func validateOpenAISchemaNode(value any) error {
	node, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("schema node must be an object")
	}
	allowed := map[string]bool{
		"type": true, "enum": true, "items": true, "minLength": true, "maxLength": true,
		"minItems": true, "maxItems": true, "properties": true, "required": true, "additionalProperties": true,
	}
	for keyword := range node {
		if !allowed[keyword] {
			return fmt.Errorf("unsupported schema keyword %q", keyword)
		}
	}
	switch node["type"] {
	case "string":
		return nil
	case "array":
		if node["items"] == nil {
			return fmt.Errorf("array schema is missing items")
		}
		return validateOpenAISchemaNode(node["items"])
	case "object":
		return validateOpenAIStructuredOutputSchema(node)
	default:
		return fmt.Errorf("unsupported schema type %q", node["type"])
	}
}

type openAIResponse struct {
	ID                string              `json:"id"`
	Model             string              `json:"model"`
	Status            string              `json:"status"`
	SystemFingerprint string              `json:"system_fingerprint"`
	Output            []openAIOutputItem  `json:"output"`
	Usage             openAIResponseUsage `json:"usage"`
}

type openAIOutputItem struct {
	Type    string                `json:"type"`
	Content []openAIOutputContent `json:"content"`
}

type openAIOutputContent struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal"`
}

type openAIResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
	InputDetails struct {
		CachedTokens     int `json:"cached_tokens"`
		CacheWriteTokens int `json:"cache_write_tokens"`
	} `json:"input_tokens_details"`
	OutputDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

func (u openAIResponseUsage) providerUsage() ProviderUsage {
	uncached := u.InputTokens - u.InputDetails.CachedTokens - u.InputDetails.CacheWriteTokens
	return ProviderUsage{InputTokens: u.InputTokens, CachedTokens: u.InputDetails.CachedTokens, CacheMissTokens: uncached, CacheWriteTokens: u.InputDetails.CacheWriteTokens, OutputTokens: u.OutputTokens, ReasoningTokens: u.OutputDetails.ReasoningTokens, TotalTokens: u.TotalTokens}
}

func validProviderUsage(usage ProviderUsage) bool {
	return usage.InputTokens > 0 && usage.OutputTokens > 0 && usage.TotalTokens >= usage.InputTokens+usage.OutputTokens &&
		usage.CachedTokens >= 0 && usage.CacheWriteTokens >= 0 && usage.ReasoningTokens >= 0 &&
		usage.CachedTokens+usage.CacheWriteTokens <= usage.InputTokens
}

func (r openAIResponse) outputText() (string, error) {
	var values []string
	for _, item := range r.Output {
		if item.Type != "message" {
			continue
		}
		for _, content := range item.Content {
			if content.Type == "refusal" || content.Refusal != "" {
				return "", fmt.Errorf("hosted diagnostic response was refused")
			}
			if content.Type == "output_text" {
				values = append(values, content.Text)
			}
		}
	}
	if len(values) == 0 {
		return "", fmt.Errorf("hosted diagnostic response contained no output text")
	}
	return strings.Join(values, ""), nil
}

func decodeOpenAIError(raw []byte) HostedProviderFailure {
	var envelope struct {
		Error struct {
			Type string          `json:"type"`
			Code json.RawMessage `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &envelope)
	code := strings.Trim(string(envelope.Error.Code), `"`)
	if code == "null" {
		code = ""
	}
	return HostedProviderFailure{ProviderType: envelope.Error.Type, ProviderCode: code}
}

type providerSafeError struct {
	kind         string
	status       int
	providerType string
	providerCode string
	requestID    string
	timeout      bool
	ambiguous    bool
	fatal        bool
}

func (e providerSafeError) FatalExperimentStop() bool { return e.fatal }

func (e providerSafeError) Error() string {
	parts := []string{"OpenAI diagnostic request failed", "kind=" + e.kind}
	if e.status != 0 {
		parts = append(parts, fmt.Sprintf("http_status=%d", e.status))
	}
	if e.providerType != "" {
		parts = append(parts, "provider_type="+e.providerType)
	}
	if e.providerCode != "" {
		parts = append(parts, "provider_code="+e.providerCode)
	}
	if e.requestID != "" {
		parts = append(parts, "request_id="+e.requestID)
	}
	if e.timeout {
		parts = append(parts, "timeout=true")
	}
	if e.ambiguous {
		parts = append(parts, "spend_outcome=ambiguous")
	}
	return strings.Join(parts, " ")
}

type BudgetGuardError struct {
	RemainingMicros int64
	RequiredMicros  int64
}

func (e BudgetGuardError) Error() string {
	return fmt.Sprintf("hosted experiment budget rejected request: remaining_usd=%s required_max_usd=%s", formatUSDMicros(e.RemainingMicros), formatUSDMicros(e.RequiredMicros))
}

func (BudgetGuardError) FatalExperimentStop() bool { return true }

type experimentBudget struct {
	mu                     sync.Mutex
	ceilingMicros          int64
	inputPriceMicros       int64
	cachedInputPriceMicros int64
	cacheWritePriceMicros  int64
	outputPriceMicros      int64
	reservedMicros         int64
	actualMicros           int64
	ambiguousMicros        int64
	uncachedInputMicros    int64
	cachedInputMicros      int64
	cacheWriteMicros       int64
	outputMicros           int64
	usage                  ProviderUsage
}

type experimentBudgetSnapshot struct {
	actualMicros    int64
	ambiguousMicros int64
	remainingMicros int64
	usage           ProviderUsage
	costs           HostedCostBreakdown
}

func newExperimentBudget(ceiling, inputPrice, cachedInputPrice, cacheWritePrice, outputPrice int64) *experimentBudget {
	return &experimentBudget{
		ceilingMicros: ceiling, inputPriceMicros: inputPrice, cachedInputPriceMicros: cachedInputPrice,
		cacheWritePriceMicros: cacheWritePrice, outputPriceMicros: outputPrice,
	}
}

func (b *experimentBudget) estimateCost(inputTokens, outputTokens int) int64 {
	worstInputPrice := b.inputPriceMicros
	if b.cacheWritePriceMicros > worstInputPrice {
		worstInputPrice = b.cacheWritePriceMicros
	}
	return tokenCostMicros(inputTokens, worstInputPrice) + tokenCostMicros(outputTokens, b.outputPriceMicros)
}

func (b *experimentBudget) reserve(amount int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.ceilingMicros - b.actualMicros - b.ambiguousMicros - b.reservedMicros
	if amount > remaining {
		return BudgetGuardError{RemainingMicros: maxInt64(remaining, 0), RequiredMicros: amount}
	}
	b.reservedMicros += amount
	return nil
}

func (b *experimentBudget) releaseReservation(amount int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reservedMicros = maxInt64(b.reservedMicros-amount, 0)
}

func (b *experimentBudget) finishFailure(reserved int64, ambiguous bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reservedMicros = maxInt64(b.reservedMicros-reserved, 0)
	if ambiguous {
		b.ambiguousMicros += reserved
	}
}

func (b *experimentBudget) finishSuccess(reserved int64, usage ProviderUsage) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reservedMicros = maxInt64(b.reservedMicros-reserved, 0)
	baseInputTokens := usage.InputTokens - usage.CachedTokens - usage.CacheWriteTokens
	if usage.CacheMissTokens > 0 || usage.InputTokens == usage.CachedTokens+usage.CacheMissTokens {
		baseInputTokens = usage.CacheMissTokens
	}
	if baseInputTokens < 0 {
		baseInputTokens = 0
	}
	uncachedInputCost := tokenCostMicros(baseInputTokens, b.inputPriceMicros)
	cachedInputCost := tokenCostMicros(usage.CachedTokens, b.cachedInputPriceMicros)
	cacheWriteCost := tokenCostMicros(usage.CacheWriteTokens, b.cacheWritePriceMicros)
	outputCost := tokenCostMicros(usage.OutputTokens, b.outputPriceMicros)
	actual := uncachedInputCost + cachedInputCost + cacheWriteCost + outputCost
	b.actualMicros += actual
	b.uncachedInputMicros += uncachedInputCost
	b.cachedInputMicros += cachedInputCost
	b.cacheWriteMicros += cacheWriteCost
	b.outputMicros += outputCost
	b.usage.InputTokens += usage.InputTokens
	b.usage.CachedTokens += usage.CachedTokens
	b.usage.CacheMissTokens += usage.CacheMissTokens
	b.usage.CacheWriteTokens += usage.CacheWriteTokens
	b.usage.OutputTokens += usage.OutputTokens
	b.usage.ReasoningTokens += usage.ReasoningTokens
	b.usage.TotalTokens += usage.TotalTokens
	if b.actualMicros+b.ambiguousMicros > b.ceilingMicros {
		return BudgetGuardError{RemainingMicros: 0, RequiredMicros: actual}
	}
	return nil
}

func (b *experimentBudget) snapshot() experimentBudgetSnapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := maxInt64(b.ceilingMicros-b.actualMicros-b.ambiguousMicros-b.reservedMicros, 0)
	return experimentBudgetSnapshot{
		actualMicros: b.actualMicros, ambiguousMicros: b.ambiguousMicros, remainingMicros: remaining, usage: b.usage,
		costs: HostedCostBreakdown{
			UncachedInputUSD: formatUSDMicros(b.uncachedInputMicros), CachedInputUSD: formatUSDMicros(b.cachedInputMicros),
			CacheWriteUSD: formatUSDMicros(b.cacheWriteMicros), OutputUSD: formatUSDMicros(b.outputMicros), TotalUSD: formatUSDMicros(b.actualMicros),
		},
	}
}

func tokenCostMicros(tokens int, priceMicrosPerMillion int64) int64 {
	if tokens <= 0 || priceMicrosPerMillion <= 0 {
		return 0
	}
	numerator := int64(tokens) * priceMicrosPerMillion
	return (numerator + 999_999) / 1_000_000
}

func parseUSDMicros(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return 0, fmt.Errorf("invalid USD decimal")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("invalid USD decimal")
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid USD decimal")
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > 6 {
		return 0, fmt.Errorf("USD decimal has more than six fractional digits")
	}
	fraction += strings.Repeat("0", 6-len(fraction))
	fractional := int64(0)
	if fraction != "" {
		fractional, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid USD decimal")
		}
	}
	if whole > (int64(^uint64(0)>>1)-fractional)/1_000_000 {
		return 0, fmt.Errorf("USD decimal is too large")
	}
	return whole*1_000_000 + fractional, nil
}

func formatUSDMicros(value int64) string {
	if value < 0 {
		value = 0
	}
	whole := value / 1_000_000
	fraction := fmt.Sprintf("%06d", value%1_000_000)
	fraction = strings.TrimRight(fraction, "0")
	if fraction == "" {
		fraction = "00"
	} else if len(fraction) == 1 {
		fraction += "0"
	}
	return fmt.Sprintf("%d.%s", whole, fraction)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
