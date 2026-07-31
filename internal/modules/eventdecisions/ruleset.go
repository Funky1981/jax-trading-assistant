package eventdecisions

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func LoadRuleset(path string) (Ruleset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Ruleset{}, fmt.Errorf("read decision ruleset: %w", err)
	}
	var rules Ruleset
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&rules); err != nil {
		return Ruleset{}, fmt.Errorf("decode decision ruleset: %w", err)
	}
	if err := rules.Validate(); err != nil {
		return Ruleset{}, err
	}
	return rules, nil
}

func (r Ruleset) Validate() error {
	if strings.TrimSpace(r.Version) == "" || strings.TrimSpace(r.ProcessorIdentity) == "" {
		return fmt.Errorf("ruleset version and processor identity are required")
	}
	if r.WatchConfidenceMinimum <= 0 || r.WatchConfidenceMinimum > 1 {
		return fmt.Errorf("watch confidence minimum must be in (0,1]")
	}
	if r.CandidateEvidenceMinimum <= 0 || r.CandidateEvidenceMinimum > 1 {
		return fmt.Errorf("candidate evidence minimum must be in (0,1]")
	}
	if strings.TrimSpace(r.SubjectRulesetVersion) == "" {
		return fmt.Errorf("subject ruleset version is required")
	}
	if r.SubjectCandidateIndependentMin < 2 {
		return fmt.Errorf("subject candidate independent minimum must be at least 2")
	}
	if r.SubjectFreshnessHours < 1 || r.SubjectFreshnessHours > 168 {
		return fmt.Errorf("subject freshness hours must be between 1 and 168")
	}
	if r.MaximumLeverage <= 0 || r.MaximumLeverage > 1 {
		return fmt.Errorf("maximum leverage must be in (0,1]")
	}
	if strings.TrimSpace(r.AllowedCandidateInstrumentType) == "" || len(r.MaterialSeverities) == 0 {
		return fmt.Errorf("candidate instrument type and material severities are required")
	}
	return nil
}
