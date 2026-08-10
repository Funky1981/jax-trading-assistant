package aishadow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DeepSeekDiagnosticProvider            = "deepseek"
	DeepSeekDiagnosticModel               = "deepseek-v4-pro"
	DeepSeekDiagnosticAPIKeyEnv           = "JAX_DEEPSEEK_EXPERIMENT_API_KEY"
	DeepSeekDiagnosticEndpoint            = "https://api.deepseek.com/chat/completions"
	DeepSeekDiagnosticReasoningEffort     = "none"
	DeepSeekDiagnosticThinkingModeEnv     = "JAX_AI_THINKING_MODE"
	DeepSeekDiagnosticThinkingMode        = "disabled"
	DeepSeekDiagnosticStructuredOutput    = "v4-prompt-contract-plain-text"
	DeepSeekDiagnosticEvidenceNamespace   = "deepseek-hosted-a1-v1"
	DeepSeekDiagnosticExperimentID        = "A1"
	DeepSeekDiagnosticMaximumBudgetMicros = int64(250_000)
)

type DeepSeekDiagnosticConfig struct {
	Runtime                        Config
	APIKey                         APISecret
	ExperimentID                   string
	ReasoningEffort                string
	ThinkingMode                   string
	MaxOutputTokens                int
	BudgetCeilingMicros            int64
	CacheMissPriceMicrosPerMillion int64
	CacheHitPriceMicrosPerMillion  int64
	OutputPriceMicrosPerMillion    int64
	InferenceExplicitlyAuthorized  bool
}

func LoadDeepSeekDiagnosticConfig(lookup func(string) (string, bool)) (DeepSeekDiagnosticConfig, error) {
	required := []string{
		"JAX_AI_SHADOW_ENABLED", "JAX_AI_PROVIDER", "JAX_AI_MODEL", "JAX_AI_TIMEOUT_SECONDS", "JAX_AI_MAX_EVENTS",
		DeepSeekDiagnosticAPIKeyEnv, "JAX_AI_EXPERIMENT_ID", "JAX_AI_REASONING_EFFORT", DeepSeekDiagnosticThinkingModeEnv,
		"JAX_AI_MAX_OUTPUT_TOKENS", "JAX_AI_EXPERIMENT_BUDGET_USD", "JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS",
		"JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS", "JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS",
	}
	values := make(map[string]string, len(required))
	for _, key := range required {
		value, ok := lookup(key)
		if !ok || strings.TrimSpace(value) == "" {
			return DeepSeekDiagnosticConfig{}, fmt.Errorf("missing required DeepSeek diagnostic configuration: %s", key)
		}
		values[key] = strings.TrimSpace(value)
	}
	enabled, err := strconv.ParseBool(values["JAX_AI_SHADOW_ENABLED"])
	if err != nil || !enabled {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("JAX_AI_SHADOW_ENABLED must explicitly be true")
	}
	if provider := strings.ToLower(values["JAX_AI_PROVIDER"]); provider != DeepSeekDiagnosticProvider {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("DeepSeek issuer diagnostic requires JAX_AI_PROVIDER=%s", DeepSeekDiagnosticProvider)
	}
	if values["JAX_AI_MODEL"] != DeepSeekDiagnosticModel {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("DeepSeek issuer diagnostic requires JAX_AI_MODEL=%s", DeepSeekDiagnosticModel)
	}
	if values["JAX_AI_EXPERIMENT_ID"] != DeepSeekDiagnosticExperimentID || !experimentIDPattern.MatchString(values["JAX_AI_EXPERIMENT_ID"]) {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("DeepSeek issuer diagnostic requires JAX_AI_EXPERIMENT_ID=%s", DeepSeekDiagnosticExperimentID)
	}
	if values["JAX_AI_REASONING_EFFORT"] != DeepSeekDiagnosticReasoningEffort {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("DeepSeek A1 requires JAX_AI_REASONING_EFFORT=%s", DeepSeekDiagnosticReasoningEffort)
	}
	if values[DeepSeekDiagnosticThinkingModeEnv] != DeepSeekDiagnosticThinkingMode {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("DeepSeek A1 requires %s=%s", DeepSeekDiagnosticThinkingModeEnv, DeepSeekDiagnosticThinkingMode)
	}
	timeoutSeconds, err := strconv.Atoi(values["JAX_AI_TIMEOUT_SECONDS"])
	if err != nil || timeoutSeconds < 1 || timeoutSeconds > 600 {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("JAX_AI_TIMEOUT_SECONDS must be between 1 and 600")
	}
	maxEvents, err := strconv.Atoi(values["JAX_AI_MAX_EVENTS"])
	if err != nil || maxEvents != diagnosticEventCount {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("DeepSeek issuer diagnostic requires JAX_AI_MAX_EVENTS=%d", diagnosticEventCount)
	}
	maxOutputTokens, err := strconv.Atoi(values["JAX_AI_MAX_OUTPUT_TOKENS"])
	if err != nil || maxOutputTokens < 1 || maxOutputTokens > 4096 {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("JAX_AI_MAX_OUTPUT_TOKENS must be between 1 and 4096")
	}
	budget, err := parseUSDMicros(values["JAX_AI_EXPERIMENT_BUDGET_USD"])
	if err != nil || budget <= 0 || budget > DeepSeekDiagnosticMaximumBudgetMicros {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("JAX_AI_EXPERIMENT_BUDGET_USD must be positive and no greater than 0.25")
	}
	cacheMissPrice, err := parseUSDMicros(values["JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS"])
	if err != nil || cacheMissPrice <= 0 {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS must be a positive decimal")
	}
	cacheHitPrice, err := parseUSDMicros(values["JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS"])
	if err != nil || cacheHitPrice <= 0 || cacheHitPrice > cacheMissPrice {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS must be positive and no greater than the cache-miss input price")
	}
	outputPrice, err := parseUSDMicros(values["JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS"])
	if err != nil || outputPrice <= 0 {
		return DeepSeekDiagnosticConfig{}, fmt.Errorf("JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS must be a positive decimal")
	}
	authorized := false
	if value, ok := lookup(OpenAIDiagnosticInferenceAuthEnv); ok && strings.TrimSpace(value) != "" {
		authorized, err = strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return DeepSeekDiagnosticConfig{}, fmt.Errorf("%s must be true or false", OpenAIDiagnosticInferenceAuthEnv)
		}
	}
	return DeepSeekDiagnosticConfig{
		Runtime: Config{Enabled: true, Provider: DeepSeekDiagnosticProvider, Model: DeepSeekDiagnosticModel,
			BaseURL: "https://api.deepseek.com", Timeout: time.Duration(timeoutSeconds) * time.Second, MaxEvents: maxEvents},
		APIKey: APISecret{value: values[DeepSeekDiagnosticAPIKeyEnv]}, ExperimentID: values["JAX_AI_EXPERIMENT_ID"],
		ReasoningEffort: values["JAX_AI_REASONING_EFFORT"], ThinkingMode: values[DeepSeekDiagnosticThinkingModeEnv],
		MaxOutputTokens: maxOutputTokens, BudgetCeilingMicros: budget,
		CacheMissPriceMicrosPerMillion: cacheMissPrice, CacheHitPriceMicrosPerMillion: cacheHitPrice,
		OutputPriceMicrosPerMillion: outputPrice, InferenceExplicitlyAuthorized: authorized,
	}, nil
}

type DeepSeekDiagnosticClient struct {
	config        DeepSeekDiagnosticConfig
	http          HTTPDoer
	budget        *experimentBudget
	mu            sync.Mutex
	returned      map[string]bool
	fingerprints  map[string]bool
	finishReasons map[string]bool
	failures      []HostedProviderFailure
	requests      int
	retries       int
	timeouts      int
	rejections    int
	stopReason    string
}

func NewDeepSeekDiagnosticClient(config DeepSeekDiagnosticConfig, transport HTTPDoer) *DeepSeekDiagnosticClient {
	if transport == nil {
		transport = &http.Client{Timeout: config.Runtime.Timeout}
	}
	return &DeepSeekDiagnosticClient{
		config: config, http: transport, returned: map[string]bool{}, fingerprints: map[string]bool{}, finishReasons: map[string]bool{},
		failures: []HostedProviderFailure{},
		budget: newExperimentBudget(config.BudgetCeilingMicros, config.CacheMissPriceMicrosPerMillion,
			config.CacheHitPriceMicrosPerMillion, config.CacheMissPriceMicrosPerMillion, config.OutputPriceMicrosPerMillion),
	}
}

func (c *DeepSeekDiagnosticClient) Complete(request ProviderRequest) (ProviderResponse, error) {
	estimatedInputTokens := len([]byte(request.System)) + len([]byte(request.User)) + 1024
	estimatedCost := c.budget.estimateCost(estimatedInputTokens, c.config.MaxOutputTokens)
	requestNumber, err := c.reserve(request, estimatedCost)
	if err != nil {
		return ProviderResponse{}, err
	}
	payload := struct {
		Model     string                 `json:"model"`
		Messages  []deepSeekInputMessage `json:"messages"`
		Thinking  map[string]string      `json:"thinking"`
		MaxTokens int                    `json:"max_tokens"`
	}{
		Model:    c.config.Runtime.Model,
		Messages: []deepSeekInputMessage{{Role: "system", Content: request.System}, {Role: "user", Content: request.User}},
		Thinking: map[string]string{"type": c.config.ThinkingMode}, MaxTokens: c.config.MaxOutputTokens,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		c.budget.releaseReservation(estimatedCost)
		return ProviderResponse{}, fmt.Errorf("marshal DeepSeek diagnostic request")
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Runtime.Timeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, DeepSeekDiagnosticEndpoint, bytes.NewReader(raw))
	if err != nil {
		c.budget.releaseReservation(estimatedCost)
		return ProviderResponse{}, fmt.Errorf("create DeepSeek diagnostic request")
	}
	httpRequest.Header.Set("Authorization", c.config.APIKey.authorizationHeader())
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(httpRequest)
	if err != nil {
		c.budget.finishFailure(estimatedCost, true)
		timedOut := errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded)
		c.recordFailure(HostedProviderFailure{RequestNumber: requestNumber, EventID: request.EventID, AttemptNumber: request.AttemptNumber, Kind: "transport", Timeout: timedOut, AmbiguousSpend: true})
		return ProviderResponse{}, deepSeekSafeError{kind: "transport", timeout: timedOut, ambiguous: true, fatal: true}
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	requestID := c.sanitize(response.Header.Get("x-request-id"))
	if readErr != nil {
		c.budget.finishFailure(estimatedCost, true)
		c.recordFailure(HostedProviderFailure{RequestNumber: requestNumber, EventID: request.EventID, AttemptNumber: request.AttemptNumber, Kind: "response_read", HTTPStatus: response.StatusCode, RequestID: requestID, AmbiguousSpend: true})
		return ProviderResponse{}, deepSeekSafeError{kind: "response_read", status: response.StatusCode, requestID: requestID, ambiguous: true, fatal: true}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		failure := decodeDeepSeekError(body)
		failure.RequestNumber, failure.EventID, failure.AttemptNumber = requestNumber, request.EventID, request.AttemptNumber
		failure.HTTPStatus, failure.RequestID, failure.Kind = response.StatusCode, requestID, "http"
		failure.ProviderType, failure.ProviderCode = c.sanitize(failure.ProviderType), c.sanitize(failure.ProviderCode)
		failure.AmbiguousSpend = response.StatusCode == http.StatusRequestTimeout || response.StatusCode >= 500
		c.budget.finishFailure(estimatedCost, failure.AmbiguousSpend)
		c.recordFailure(failure)
		return ProviderResponse{}, deepSeekSafeError{kind: "http", status: response.StatusCode, providerType: failure.ProviderType, providerCode: failure.ProviderCode, requestID: requestID, ambiguous: failure.AmbiguousSpend, fatal: true}
	}
	var decoded deepSeekResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		c.failWithAmbiguousUsage(estimatedCost, requestNumber, request, response.StatusCode, requestID, "decode")
		return ProviderResponse{}, deepSeekSafeError{kind: "decode", status: response.StatusCode, requestID: requestID, ambiguous: true, fatal: true}
	}
	usage, usageErr := decoded.Usage.providerUsage()
	if usageErr != nil || !validDeepSeekProviderUsage(usage) {
		c.failWithAmbiguousUsage(estimatedCost, requestNumber, request, response.StatusCode, requestID, "usage")
		return ProviderResponse{}, deepSeekSafeError{kind: "usage", status: response.StatusCode, requestID: requestID, ambiguous: true, fatal: true}
	}
	if err := c.budget.finishSuccess(estimatedCost, usage); err != nil {
		c.setStopReason("budget_exceeded_after_reported_usage")
		return ProviderResponse{}, err
	}
	decoded.Model = c.sanitize(decoded.Model)
	decoded.SystemFingerprint = c.sanitize(decoded.SystemFingerprint)
	if decoded.Model != "" {
		c.mu.Lock()
		c.returned[decoded.Model] = true
		c.mu.Unlock()
	}
	if decoded.Model != c.config.Runtime.Model {
		return ProviderResponse{}, c.identityFailure(requestNumber, request, response.StatusCode, requestID, "model_identity")
	}
	if strings.TrimSpace(decoded.SystemFingerprint) == "" {
		return ProviderResponse{}, c.identityFailure(requestNumber, request, response.StatusCode, requestID, "system_fingerprint")
	}
	if usage.ReasoningTokens != 0 {
		c.setStopReason("non_thinking_reasoning_tokens_nonzero")
		return ProviderResponse{}, c.identityFailure(requestNumber, request, response.StatusCode, requestID, "reasoning_tokens")
	}
	if decoded.Object != "chat.completion" || strings.TrimSpace(decoded.ID) == "" || len(decoded.Choices) != 1 || decoded.Choices[0].Index != 0 {
		return ProviderResponse{}, c.identityFailure(requestNumber, request, response.StatusCode, requestID, "response_shape")
	}
	choice := decoded.Choices[0]
	choice.FinishReason = c.sanitize(choice.FinishReason)
	if choice.Message.ReasoningContent != nil && strings.TrimSpace(*choice.Message.ReasoningContent) != "" {
		c.setStopReason("non_thinking_reasoning_content_present")
		return ProviderResponse{}, c.identityFailure(requestNumber, request, response.StatusCode, requestID, "reasoning_content")
	}
	if strings.TrimSpace(choice.FinishReason) == "" || choice.Message.Content == nil || strings.TrimSpace(*choice.Message.Content) == "" {
		return ProviderResponse{}, c.identityFailure(requestNumber, request, response.StatusCode, requestID, "response_content")
	}
	content := c.sanitize(*choice.Message.Content)
	c.mu.Lock()
	c.fingerprints[decoded.SystemFingerprint] = true
	c.finishReasons[choice.FinishReason] = true
	c.mu.Unlock()
	return ProviderResponse{
		Content: content, ModelIdentifier: decoded.Model, RequestID: requestID, ResponseID: c.sanitize(decoded.ID),
		Status: "completed", SystemFingerprint: decoded.SystemFingerprint, FinishReason: choice.FinishReason, Usage: usage,
	}, nil
}

func (c *DeepSeekDiagnosticClient) sanitize(value string) string {
	if c.config.APIKey.value == "" {
		return value
	}
	return strings.ReplaceAll(value, c.config.APIKey.value, "<redacted>")
}

func (c *DeepSeekDiagnosticClient) reserve(request ProviderRequest, estimatedCost int64) (int, error) {
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

func (c *DeepSeekDiagnosticClient) failWithAmbiguousUsage(reserved int64, requestNumber int, request ProviderRequest, status int, requestID, kind string) {
	c.budget.finishFailure(reserved, true)
	c.recordFailure(HostedProviderFailure{RequestNumber: requestNumber, EventID: request.EventID, AttemptNumber: request.AttemptNumber, Kind: kind, HTTPStatus: status, RequestID: requestID, AmbiguousSpend: true})
}

func (c *DeepSeekDiagnosticClient) identityFailure(requestNumber int, request ProviderRequest, status int, requestID, kind string) error {
	c.recordFailure(HostedProviderFailure{RequestNumber: requestNumber, EventID: request.EventID, AttemptNumber: request.AttemptNumber, Kind: kind, HTTPStatus: status, RequestID: requestID})
	return deepSeekSafeError{kind: kind, status: status, requestID: requestID, fatal: true}
}

func (c *DeepSeekDiagnosticClient) recordFailure(failure HostedProviderFailure) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = append(c.failures, failure)
	if failure.Timeout {
		c.timeouts++
	}
}

func (c *DeepSeekDiagnosticClient) setStopReason(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopReason = reason
}

func (c *DeepSeekDiagnosticClient) ExperimentSnapshot() HostedExperimentSnapshot {
	budget := c.budget.snapshot()
	c.mu.Lock()
	defer c.mu.Unlock()
	models, fingerprints, reasons := sortedKeys(c.returned), sortedKeys(c.fingerprints), sortedKeys(c.finishReasons)
	return HostedExperimentSnapshot{
		ExperimentID: c.config.ExperimentID, Provider: DeepSeekDiagnosticProvider, RequestedModel: c.config.Runtime.Model,
		ReturnedModels: models, SystemFingerprints: fingerprints, FinishReasons: reasons,
		ReasoningEffort: c.config.ReasoningEffort, ThinkingMode: c.config.ThinkingMode,
		StructuredOutputMode: DeepSeekDiagnosticStructuredOutput, MaxOutputTokensPerRequest: c.config.MaxOutputTokens,
		Pricing: HostedPricingPlan{
			InputUSDPerMillionTokens:          formatUSDMicros(c.config.CacheMissPriceMicrosPerMillion),
			CachedInputUSDPerMillionTokens:    formatUSDMicros(c.config.CacheHitPriceMicrosPerMillion),
			CacheMissInputUSDPerMillionTokens: formatUSDMicros(c.config.CacheMissPriceMicrosPerMillion),
			OutputUSDPerMillionTokens:         formatUSDMicros(c.config.OutputPriceMicrosPerMillion),
			Source:                            "execution-time configuration; re-verify before paid execution",
		},
		BudgetCeilingUSD: formatUSDMicros(c.config.BudgetCeilingMicros), ActualCalculableCostUSD: formatUSDMicros(budget.actualMicros),
		CostByCategory: budget.costs, AmbiguousLiabilityUSD: formatUSDMicros(budget.ambiguousMicros),
		AccountedCostUSD: formatUSDMicros(budget.actualMicros + budget.ambiguousMicros), RemainingBudgetUSD: formatUSDMicros(budget.remainingMicros),
		RequestCount: c.requests, RetryCount: c.retries, Usage: budget.usage, ProviderErrorCount: len(c.failures),
		TimeoutCount: c.timeouts, BudgetRejectionCount: c.rejections, StopReason: c.stopReason,
		Failures: append([]HostedProviderFailure(nil), c.failures...),
	}
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	sort.Strings(keys)
	return keys
}

type deepSeekInputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepSeekResponse struct {
	ID                string                   `json:"id"`
	Model             string                   `json:"model"`
	Object            string                   `json:"object"`
	SystemFingerprint string                   `json:"system_fingerprint"`
	Choices           []deepSeekResponseChoice `json:"choices"`
	Usage             deepSeekResponseUsage    `json:"usage"`
}

type deepSeekResponseChoice struct {
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason"`
	Message      struct {
		Content          *string `json:"content"`
		ReasoningContent *string `json:"reasoning_content"`
		Role             string  `json:"role"`
	} `json:"message"`
}

type deepSeekResponseUsage struct {
	PromptTokens          *int `json:"prompt_tokens"`
	PromptCacheHitTokens  *int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens *int `json:"prompt_cache_miss_tokens"`
	CompletionTokens      *int `json:"completion_tokens"`
	TotalTokens           *int `json:"total_tokens"`
	CompletionDetails     *struct {
		ReasoningTokens *int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
}

func (u deepSeekResponseUsage) providerUsage() (ProviderUsage, error) {
	if u.PromptTokens == nil || u.PromptCacheHitTokens == nil || u.PromptCacheMissTokens == nil ||
		u.CompletionTokens == nil || u.TotalTokens == nil || u.CompletionDetails == nil || u.CompletionDetails.ReasoningTokens == nil {
		return ProviderUsage{}, fmt.Errorf("DeepSeek usage fields are incomplete")
	}
	return ProviderUsage{
		InputTokens: *u.PromptTokens, CachedTokens: *u.PromptCacheHitTokens, CacheMissTokens: *u.PromptCacheMissTokens,
		OutputTokens: *u.CompletionTokens, ReasoningTokens: *u.CompletionDetails.ReasoningTokens, TotalTokens: *u.TotalTokens,
	}, nil
}

func validDeepSeekProviderUsage(usage ProviderUsage) bool {
	return usage.InputTokens > 0 && usage.OutputTokens > 0 && usage.TotalTokens == usage.InputTokens+usage.OutputTokens &&
		usage.CachedTokens >= 0 && usage.CacheMissTokens >= 0 && usage.CachedTokens+usage.CacheMissTokens == usage.InputTokens &&
		usage.ReasoningTokens >= 0 && usage.CacheWriteTokens == 0
}

func decodeDeepSeekError(raw []byte) HostedProviderFailure {
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

type deepSeekSafeError struct {
	kind         string
	status       int
	providerType string
	providerCode string
	requestID    string
	timeout      bool
	ambiguous    bool
	fatal        bool
}

func (e deepSeekSafeError) FatalExperimentStop() bool { return e.fatal }

func (e deepSeekSafeError) Error() string {
	parts := []string{"DeepSeek diagnostic request failed", "kind=" + e.kind}
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
