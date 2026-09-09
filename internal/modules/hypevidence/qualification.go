package hypevidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const QualificationContractV1 = "jax.hyp-event-001a.qualification/v1"

type QualificationDecision string

const (
	QualificationPass QualificationDecision = "QUALIFICATION_PASS"
	QualificationFail QualificationDecision = "QUALIFICATION_FAIL"
)

// QualificationAttempt is deliberately provider-neutral but retains every
// billed attempt, including a response later rejected by semantic validation.
type QualificationAttempt struct {
	EventID              string `json:"event_id"`
	AttemptNumber        int    `json:"attempt_number"`
	RequestID            string `json:"request_id"`
	ResponseID           string `json:"response_id"`
	UsageArtifactID      string `json:"usage_artifact_id"`
	RawResponseSHA256    string `json:"raw_response_sha256"`
	InputTokens          int    `json:"input_tokens"`
	CachedInputTokens    int    `json:"cached_input_tokens"`
	CacheWriteTokens     int    `json:"cache_write_tokens"`
	OutputTokens         int    `json:"output_tokens"`
	TotalTokens          int    `json:"total_tokens"`
	StructuredValid      bool   `json:"structured_valid"`
	EvidenceAnchorsValid bool   `json:"evidence_anchors_valid"`
	FutureInfoCheck      bool   `json:"future_information_check"`
	InjectionCheck       bool   `json:"injection_check"`
	AbstentionCheck      bool   `json:"abstention_check"`
}

type QualificationArtifact struct {
	ContractVersion       string                 `json:"contract_version"`
	ID                    string                 `json:"id"`
	SampleIdentity        string                 `json:"sample_identity"`
	SampleSelectionPolicy string                 `json:"sample_selection_policy"`
	EventIDs              []string               `json:"event_ids"`
	ClassifierVersion     string                 `json:"classifier_version"`
	PromptVersion         string                 `json:"prompt_version"`
	Provider              string                 `json:"provider"`
	Model                 string                 `json:"model"`
	Attempts              []QualificationAttempt `json:"attempts"`
	SemanticReviewMethod  string                 `json:"semantic_review_method"`
	StructuredValidCount  int                    `json:"structured_valid_count"`
	GroundedCount         int                    `json:"grounded_count"`
	FutureInfoPassCount   int                    `json:"future_information_pass_count"`
	InjectionPassCount    int                    `json:"injection_pass_count"`
	AbstentionPassCount   int                    `json:"abstention_pass_count"`
	Metrics               map[string]float64     `json:"metrics"`
	Decision              QualificationDecision  `json:"decision"`
	DecisionID            string                 `json:"decision_id"`
	DecisionAt            time.Time              `json:"decision_at"`
}

func NewQualificationArtifact(artifact QualificationArtifact) (QualificationArtifact, error) {
	artifact.ContractVersion = QualificationContractV1
	artifact.ID = qualificationArtifactID(artifact)
	if err := artifact.Validate(); err != nil {
		return QualificationArtifact{}, err
	}
	return artifact, nil
}

func (a QualificationArtifact) Validate() error {
	if a.ContractVersion != QualificationContractV1 || !validQualificationID(a.ID, "qual_") || strings.TrimSpace(a.SampleIdentity) == "" || strings.TrimSpace(a.SampleSelectionPolicy) == "" || len(a.EventIDs) == 0 || strings.TrimSpace(a.ClassifierVersion) == "" || strings.TrimSpace(a.PromptVersion) == "" || strings.TrimSpace(a.Provider) == "" || strings.TrimSpace(a.Model) == "" || len(a.Attempts) == 0 || strings.TrimSpace(a.SemanticReviewMethod) == "" || a.DecisionAt.IsZero() || a.DecisionAt.Location() != time.UTC || (a.Decision != QualificationPass && a.Decision != QualificationFail) {
		return fmt.Errorf("qualification artifact is incomplete")
	}
	seenEvents := map[string]struct{}{}
	for _, eventID := range a.EventIDs {
		if strings.TrimSpace(eventID) == "" {
			return fmt.Errorf("qualification event identity is empty")
		}
		if _, ok := seenEvents[eventID]; ok {
			return fmt.Errorf("qualification event identities are duplicated")
		}
		seenEvents[eventID] = struct{}{}
	}
	seenAttempts := map[string]struct{}{}
	attemptsByEvent := map[string]int{}
	for _, attempt := range a.Attempts {
		if _, ok := seenEvents[attempt.EventID]; !ok || attempt.AttemptNumber <= 0 || strings.TrimSpace(attempt.RequestID) == "" || strings.TrimSpace(attempt.ResponseID) == "" || strings.TrimSpace(attempt.UsageArtifactID) == "" || !validSHA256(attempt.RawResponseSHA256) || attempt.InputTokens <= 0 || attempt.CachedInputTokens < 0 || attempt.CacheWriteTokens < 0 || attempt.CachedInputTokens+attempt.CacheWriteTokens > attempt.InputTokens || attempt.OutputTokens < 0 || attempt.TotalTokens < attempt.InputTokens+attempt.OutputTokens {
			return fmt.Errorf("qualification attempt is missing durable usage or identity")
		}
		key := attempt.EventID + "#" + fmt.Sprint(attempt.AttemptNumber)
		if _, ok := seenAttempts[key]; ok {
			return fmt.Errorf("qualification attempts are duplicated")
		}
		seenAttempts[key] = struct{}{}
		attemptsByEvent[attempt.EventID]++
	}
	for eventID := range seenEvents {
		if attemptsByEvent[eventID] == 0 {
			return fmt.Errorf("qualification event %q has no persisted attempt", eventID)
		}
	}
	structuredValid, grounded, futureInfo, injection, abstention := 0, 0, 0, 0, 0
	for _, attempt := range a.Attempts {
		if attempt.StructuredValid {
			structuredValid++
		}
		if attempt.EvidenceAnchorsValid {
			grounded++
		}
		if attempt.FutureInfoCheck {
			futureInfo++
		}
		if attempt.InjectionCheck {
			injection++
		}
		if attempt.AbstentionCheck {
			abstention++
		}
	}
	if a.StructuredValidCount != structuredValid || a.GroundedCount != grounded || a.FutureInfoPassCount != futureInfo || a.InjectionPassCount != injection || a.AbstentionPassCount != abstention {
		return fmt.Errorf("qualification metrics do not match retained attempts")
	}
	if a.StructuredValidCount < 0 || a.GroundedCount < 0 || a.FutureInfoPassCount < 0 || a.InjectionPassCount < 0 || a.AbstentionPassCount < 0 || a.StructuredValidCount > len(a.Attempts) || a.GroundedCount > a.StructuredValidCount || a.FutureInfoPassCount > len(a.Attempts) || a.InjectionPassCount > len(a.Attempts) || a.AbstentionPassCount > len(a.Attempts) {
		return fmt.Errorf("qualification metrics are invalid")
	}
	if a.ID != qualificationArtifactID(a) {
		return fmt.Errorf("qualification artifact identity does not match contents")
	}
	return nil
}

func (a QualificationArtifact) Compatible(classifierVersion, promptVersion, provider, model string) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.Decision != QualificationPass {
		return fmt.Errorf("qualification has not passed")
	}
	if a.ClassifierVersion != classifierVersion || a.PromptVersion != promptVersion || a.Provider != provider || a.Model != model {
		return fmt.Errorf("qualification identity does not match requested classifier")
	}
	return nil
}

func LoadQualificationArtifact(path string) (QualificationArtifact, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return QualificationArtifact{}, err
	}
	var artifact QualificationArtifact
	if err := json.Unmarshal(b, &artifact); err != nil {
		return QualificationArtifact{}, err
	}
	if err := artifact.Validate(); err != nil {
		return QualificationArtifact{}, err
	}
	return artifact, nil
}

// WriteQualificationArtifact is immutable: an existing path may only be
// reopened when it contains byte-equivalent validated content.
func WriteQualificationArtifact(path string, artifact QualificationArtifact) error {
	if err := artifact.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) != string(b) {
			return fmt.Errorf("qualification artifact is immutable")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func qualificationArtifactID(a QualificationArtifact) string {
	a.ID = ""
	b, _ := json.Marshal(a)
	d := sha256.Sum256(b)
	return "qual_" + hex.EncodeToString(d[:])
}

func validQualificationID(id, prefix string) bool {
	if !strings.HasPrefix(id, prefix) || len(id) != len(prefix)+64 {
		return false
	}
	return validSHA256(id[len(prefix):])
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}
