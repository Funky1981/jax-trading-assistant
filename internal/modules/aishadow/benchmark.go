package aishadow

import (
	"fmt"
	"path/filepath"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"

	"github.com/google/uuid"
)

type Runner struct {
	Config        Config
	Provider      Provider
	Repository    Repository
	OutputRoot    string
	AssetResolver assetresolution.Resolver
}

func (r Runner) Run(manifest Manifest, events []BenchmarkEvent) (Report, ArtifactPaths, error) {
	if r.Provider == nil || r.Repository == nil {
		return Report{}, ArtifactPaths{}, fmt.Errorf("AI shadow provider and repository are required")
	}
	if len(events) != r.Config.MaxEvents {
		return Report{}, ArtifactPaths{}, fmt.Errorf("selected event count %d does not match configured maximum %d", len(events), r.Config.MaxEvents)
	}
	proxyExposures, err := r.AssetResolver.ProxyExposures()
	if err != nil {
		return Report{}, ArtifactPaths{}, fmt.Errorf("prepare bounded exposure policy: %w", err)
	}
	before, err := r.Repository.SafetyCounts()
	if err != nil {
		return Report{}, ArtifactPaths{}, err
	}
	runID := uuid.NewString()
	started := time.Now().UTC()
	run := RunRecord{
		ID: runID, ManifestVersion: manifest.Version, ManifestFingerprint: manifest.Fingerprint,
		Provider: r.Config.Provider, Model: r.Config.Model, PromptVersion: PromptVersion,
		SchemaVersion: SchemaVersion, Seed: r.Config.Seed, Temperature: r.Config.Temperature,
		EventLimit: len(events), StartedAt: started, SafetyBefore: before,
	}
	if err := r.Repository.StartRun(run); err != nil {
		return Report{}, ArtifactPaths{}, err
	}
	results := make([]EventResult, 0, len(events))
	failRun := func(cause error) (Report, ArtifactPaths, error) {
		after, _ := r.Repository.SafetyCounts()
		_ = r.Repository.FinishRun(FinishRecord{RunID: runID, CompletedAt: time.Now().UTC(), Status: "invalid", FailureReason: cause.Error(), SafetyAfter: after})
		return Report{}, ArtifactPaths{}, cause
	}
	for _, event := range events {
		result, attempts, err := r.analyse(runID, event, proxyExposures)
		if err != nil {
			return failRun(err)
		}
		for _, attempt := range attempts {
			if err := r.Repository.SaveAttempt(attempt); err != nil {
				return failRun(err)
			}
		}
		if err := r.Repository.SaveResult(result); err != nil {
			return failRun(err)
		}
		results = append(results, result)
	}
	after, err := r.Repository.SafetyCounts()
	if err != nil {
		return failRun(err)
	}
	if before != after {
		return failRun(fmt.Errorf("prohibited safety counts changed during AI shadow benchmark"))
	}
	report := BuildReport(run, manifest, events, results, before, after)
	paths, err := WriteArtifacts(filepath.Join(r.OutputRoot, runID), report, manifest)
	if err != nil {
		return failRun(err)
	}
	if err := r.Repository.FinishRun(FinishRecord{RunID: runID, CompletedAt: time.Now().UTC(), Status: "completed", SafetyAfter: after, ReportPaths: paths}); err != nil {
		return Report{}, ArtifactPaths{}, err
	}
	return report, paths, nil
}

func (r Runner) analyse(runID string, event BenchmarkEvent, proxyExposures []string) (EventResult, []Attempt, error) {
	result, attempts, _, err := analyseEvent(
		r.Config, r.Provider, r.AssetResolver, runID, ManifestVersion,
		event.ID, event.InputFingerprint, event.Input, proxyExposures,
	)
	return result, attempts, err
}

func analyseEvent(config Config, provider Provider, resolver assetresolution.Resolver, runID, manifestVersion, eventID, inputFingerprint string, input EventInput, proxyExposures []string) (EventResult, []Attempt, []ProviderTrace, error) {
	request, err := InitialRequest(input, proxyExposures)
	if err != nil {
		return EventResult{}, nil, nil, err
	}
	attempts := []Attempt{}
	traces := []ProviderTrace{}
	var firstRequest time.Time
	var totalDuration time.Duration
	var final Attempt
	var parsed *StructuredResult
	var resolution *PolicyResolution
	for number := 1; number <= 2; number++ {
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
			parsed, resolution, validationErrors = ParseAndValidate(response.Content, input, resolver)
		}
		trace := ProviderTrace{AttemptNumber: number, Content: response.Content, ModelIdentifier: response.ModelIdentifier}
		if providerErr != nil {
			trace.ProviderError = providerErr.Error()
		}
		traces = append(traces, trace)
		status := "accepted"
		if len(validationErrors) > 0 {
			status = "rejected"
		}
		attempt := Attempt{
			RunID: runID, EventID: eventID, AttemptNumber: number, InputFingerprint: inputFingerprint,
			Provider: config.Provider, Model: config.Model, ModelReportedIdentifier: response.ModelIdentifier,
			PromptVersion: PromptVersion, SchemaVersion: SchemaVersion, Seed: config.Seed,
			Temperature: config.Temperature, RequestTimestamp: requested, ResponseTimestamp: responded,
			Duration: duration, RawResponseHash: rawHash(response.Content), ValidationStatus: status,
			ValidationErrors: validationErrors, FailureReason: failureReason,
		}
		attempts = append(attempts, attempt)
		final = attempt
		if status == "accepted" || providerErr != nil || number == 2 {
			break
		}
		request, err = CorrectiveRequest(validationErrors, response.Content, proxyExposures)
		if err != nil {
			return EventResult{}, attempts, traces, err
		}
	}
	final.RequestTimestamp = firstRequest
	final.Duration = totalDuration
	return EventResult{Attempt: final, ManifestVersion: manifestVersion, RetryCount: len(attempts) - 1, Parsed: parsed, Resolution: resolution}, attempts, traces, nil
}
