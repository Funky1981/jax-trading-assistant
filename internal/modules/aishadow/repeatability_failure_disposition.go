package aishadow

import (
	"fmt"
	"os"

	"github.com/google/uuid"
)

const (
	C1F3RepeatabilityR2FailureDispositionVersion = "ai-shadow-c1f3-repeatability-r2-failure-disposition-v1"
	C1F3RepeatabilityR2FailureDispositionPath    = "config/ai-shadow-c1f3-repeatability-r2-failure-disposition-v1.json"
	C1F3RepeatabilityR2FailureDispositionSHA256  = "92837a3cb779da846d4da20793e88e5abbecabe17ca05dad69f71b2fd38ef121"
	C1F3RepeatabilityR2FailedRunID               = "59f74c48-bb28-479e-aa7f-16f213b49a15"
	C1F3RepeatabilityR2PreflightRunID            = "9ec27c4c-b53b-4b85-9167-d17f76c940e3"
	C1F3RepeatabilityR2PreflightSHA256           = "d552298a912e0e9dc56aeaabd9958bc3225c06c7131e1fc15f9f12086b233e18"
)

type C1F3RepeatabilityR2FailureDisposition struct {
	Version     string `json:"version"`
	FailedRunID string `json:"failed_run_id"`
	Preflight   struct {
		RunID  string `json:"run_id"`
		SHA256 string `json:"sha256"`
	} `json:"preflight"`
	Profile struct {
		Identity    string `json:"identity"`
		Fingerprint string `json:"fingerprint"`
	} `json:"profile"`
	FailureClassification         string `json:"failure_classification"`
	FirstProviderActivityOccurred bool   `json:"first_provider_activity_occurred"`
	ProviderRequestCount          struct {
		ExactKnown     bool  `json:"exact_known"`
		PossibleValues []int `json:"possible_values"`
	} `json:"provider_request_count"`
	ScoreableArtifactsPresent  bool `json:"scoreable_artifacts_present"`
	ArtifactIndexPresent       bool `json:"artifact_index_present"`
	SemanticResultPresent      bool `json:"semantic_result_present"`
	CellConsumed               bool `json:"cell_consumed"`
	RerunAllowed               bool `json:"rerun_allowed"`
	ResponseSemanticsInspected bool `json:"response_semantics_inspected"`
	RepeatabilityMeasured      bool `json:"repeatability_measured"`
}

func LoadC1F3RepeatabilityR2FailureDisposition(path string) (C1F3RepeatabilityR2FailureDisposition, error) {
	if got, err := hashOpaqueFile(path); err != nil {
		return C1F3RepeatabilityR2FailureDisposition{}, err
	} else if got != C1F3RepeatabilityR2FailureDispositionSHA256 {
		return C1F3RepeatabilityR2FailureDisposition{}, fmt.Errorf("C1F3 repeatability r2 failure disposition hash changed")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return C1F3RepeatabilityR2FailureDisposition{}, err
	}
	var disposition C1F3RepeatabilityR2FailureDisposition
	if err := decodeStrictC1F3ControlPlane(raw, &disposition); err != nil {
		return C1F3RepeatabilityR2FailureDisposition{}, err
	}
	if disposition.Version != C1F3RepeatabilityR2FailureDispositionVersion || disposition.FailedRunID != C1F3RepeatabilityR2FailedRunID ||
		disposition.Preflight.RunID != C1F3RepeatabilityR2PreflightRunID || disposition.Preflight.SHA256 != C1F3RepeatabilityR2PreflightSHA256 ||
		disposition.Profile.Identity != C1F3RepeatabilityProfileIdentity || disposition.Profile.Fingerprint != C1F3RepeatabilityProfileFingerprint ||
		disposition.FailureClassification != "TECHNICAL_INTEGRITY_FAIL_NO_REPEATABILITY_RESULT" || !disposition.FirstProviderActivityOccurred ||
		disposition.ProviderRequestCount.ExactKnown || len(disposition.ProviderRequestCount.PossibleValues) != 2 ||
		disposition.ProviderRequestCount.PossibleValues[0] != 1 || disposition.ProviderRequestCount.PossibleValues[1] != 2 ||
		disposition.ScoreableArtifactsPresent || disposition.ArtifactIndexPresent || disposition.SemanticResultPresent ||
		!disposition.CellConsumed || disposition.RerunAllowed || disposition.ResponseSemanticsInspected || disposition.RepeatabilityMeasured {
		return C1F3RepeatabilityR2FailureDisposition{}, fmt.Errorf("C1F3 repeatability r2 failure disposition changed")
	}
	if _, err := uuid.Parse(disposition.FailedRunID); err != nil {
		return C1F3RepeatabilityR2FailureDisposition{}, fmt.Errorf("C1F3 repeatability r2 failed run ID is invalid")
	}
	return disposition, nil
}
