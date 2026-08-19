package aishadow

import (
	"errors"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
)

// analyseC1FEvent is intentionally not routed from a C1F3 profile in C1F2.
// It freezes the bounded one-correction behavior for later activation only
// after C1F2A supplies immutable typed-label and rubric bindings.
func analyseC1FEvent(config Config, provider Provider, resolver assetresolution.Resolver, runID, manifestVersion, eventID, inputFingerprint string, input EventInput, proxyExposures []string) (EventResult, []Attempt, []ProviderTrace, error) {
	request, err := V6InitialRequest(input, proxyExposures)
	if err != nil {
		return EventResult{}, nil, nil, err
	}
	attempts := []Attempt{}
	traces := []ProviderTrace{}
	var firstRequest time.Time
	var totalDuration time.Duration
	var final Attempt
	var parsed *V5StructuredResult
	var attribution *CausalAttributionDecision
	var resolution *PolicyResolution
	for number := 1; number <= 2; number++ {
		request.EventID = eventID
		request.AttemptNumber = number
		requested := time.Now().UTC()
		if number == 1 {
			firstRequest = requested
		}
		response, providerErr := provider.Complete(request)
		responded := time.Now().UTC()
		duration := responded.Sub(requested)
		totalDuration += duration
		validationErrors := []string{}
		failureReason := ""
		if providerErr != nil {
			failureReason = providerErr.Error()
			validationErrors = []string{"provider request failed"}
		} else {
			parsed, attribution, resolution, validationErrors = ParseValidateAndApplyC1F(response.Content, input, resolver)
		}
		trace := ProviderTrace{AttemptNumber: number, Content: response.Content, ModelIdentifier: response.ModelIdentifier, RequestID: response.RequestID, ResponseID: response.ResponseID, Status: response.Status, SystemFingerprint: response.SystemFingerprint, FinishReason: response.FinishReason, Usage: response.Usage}
		if providerErr != nil {
			trace.ProviderError = providerErr.Error()
		}
		traces = append(traces, trace)
		status := "accepted"
		if len(validationErrors) > 0 {
			status = "rejected"
		}
		attempt := Attempt{RunID: runID, EventID: eventID, AttemptNumber: number, InputFingerprint: inputFingerprint, Provider: config.Provider, Model: config.Model, ModelReportedIdentifier: response.ModelIdentifier, PromptVersion: V6PromptVersion, SchemaVersion: V5SchemaVersion, Seed: config.Seed, Temperature: config.Temperature, RequestTimestamp: requested, ResponseTimestamp: responded, Duration: duration, RawResponseHash: rawHash(response.Content), ValidationStatus: status, ValidationErrors: validationErrors, FailureReason: failureReason}
		attempts = append(attempts, attempt)
		final = attempt
		if providerErr != nil {
			var fatal interface{ FatalExperimentStop() bool }
			if errors.As(providerErr, &fatal) && fatal.FatalExperimentStop() {
				return EventResult{}, attempts, traces, providerErr
			}
		}
		if status == "accepted" || providerErr != nil || number == 2 {
			break
		}
		request, err = V6CorrectiveRequest(validationErrors, response.Content, proxyExposures)
		if err != nil {
			return EventResult{}, attempts, traces, err
		}
	}
	final.RequestTimestamp = firstRequest
	final.Duration = totalDuration
	return EventResult{Attempt: final, ManifestVersion: manifestVersion, RetryCount: len(attempts) - 1, V5Parsed: parsed, CausalAttribution: attribution, Resolution: resolution}, attempts, traces, nil
}
