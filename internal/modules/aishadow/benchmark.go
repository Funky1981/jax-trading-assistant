package aishadow

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type Runner struct {
	Config       Config
	Provider     Provider
	Repository   Repository
	OutputRoot   string
	KnownTickers map[string]bool
}

func (r Runner) Run(manifest Manifest, events []BenchmarkEvent) (Report, ArtifactPaths, error) {
	if r.Provider == nil || r.Repository == nil {
		return Report{}, ArtifactPaths{}, fmt.Errorf("AI shadow provider and repository are required")
	}
	if len(events) != r.Config.MaxEvents {
		return Report{}, ArtifactPaths{}, fmt.Errorf("selected event count %d does not match configured maximum %d", len(events), r.Config.MaxEvents)
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
		result, attempts, err := r.analyse(runID, event)
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
	report := BuildReport(run, manifest, events, results, before, after, r.KnownTickers)
	paths, err := WriteArtifacts(filepath.Join(r.OutputRoot, runID), report, manifest)
	if err != nil {
		return failRun(err)
	}
	if err := r.Repository.FinishRun(FinishRecord{RunID: runID, CompletedAt: time.Now().UTC(), Status: "completed", SafetyAfter: after, ReportPaths: paths}); err != nil {
		return Report{}, ArtifactPaths{}, err
	}
	return report, paths, nil
}

func (r Runner) analyse(runID string, event BenchmarkEvent) (EventResult, []Attempt, error) {
	request, err := InitialRequest(event.Input)
	if err != nil {
		return EventResult{}, nil, err
	}
	attempts := []Attempt{}
	var firstRequest time.Time
	var totalDuration time.Duration
	var final Attempt
	var parsed *StructuredResult
	for number := 1; number <= 2; number++ {
		requested := time.Now().UTC()
		if number == 1 {
			firstRequest = requested
		}
		response, providerErr := r.Provider.Complete(request)
		responded := time.Now().UTC()
		duration := responded.Sub(requested)
		totalDuration += duration
		validationErrors := []string{}
		failureReason := ""
		if providerErr != nil {
			failureReason = providerErr.Error()
			validationErrors = []string{"provider request failed"}
		} else {
			parsed, validationErrors = ParseAndValidate(response.Content)
		}
		status := "accepted"
		if len(validationErrors) > 0 {
			status = "rejected"
		}
		attempt := Attempt{
			RunID: runID, EventID: event.ID, AttemptNumber: number, InputFingerprint: event.InputFingerprint,
			Provider: r.Config.Provider, Model: r.Config.Model, ModelReportedIdentifier: response.ModelIdentifier,
			PromptVersion: PromptVersion, SchemaVersion: SchemaVersion, Seed: r.Config.Seed,
			Temperature: r.Config.Temperature, RequestTimestamp: requested, ResponseTimestamp: responded,
			Duration: duration, RawResponseHash: rawHash(response.Content), ValidationStatus: status,
			ValidationErrors: validationErrors, FailureReason: failureReason,
		}
		attempts = append(attempts, attempt)
		final = attempt
		if status == "accepted" || providerErr != nil || number == 2 {
			break
		}
		request, err = CorrectiveRequest(validationErrors, response.Content)
		if err != nil {
			return EventResult{}, attempts, err
		}
	}
	final.RequestTimestamp = firstRequest
	final.Duration = totalDuration
	return EventResult{Attempt: final, ManifestVersion: ManifestVersion, RetryCount: len(attempts) - 1, Parsed: parsed}, attempts, nil
}
