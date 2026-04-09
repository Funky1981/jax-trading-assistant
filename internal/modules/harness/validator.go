package harness

import (
	"fmt"
	"regexp"
)

var priceTargetPattern = regexp.MustCompile(`(?i)(price target|target price|target of|to reach|will hit)\s*\$?\d+(\.\d+)?`)

type ValidationResult struct {
	OK      bool
	Reasons []string
}

type DefaultValidator struct{}

func NewValidator() *DefaultValidator { return &DefaultValidator{} }

func (v *DefaultValidator) ValidateAnswer(policy Policy, bundle EvidenceBundle, answer string) ValidationResult {
	var reasons []string

	if err := policy.CheckAnswerAllowed(answer); err != nil {
		reasons = append(reasons, err.Error())
	}

	if containsUnsupportedPriceTarget(answer) {
		reasons = append(reasons, "unsupported price target detected")
	}

	if bundle.IsWeak() {
		if !containsUncertaintyLanguage(answer) {
			reasons = append(reasons, "weak evidence answer missing uncertainty language")
		}
		if containsCertaintyLanguage(answer) {
			reasons = append(reasons, "weak evidence answer uses certainty language")
		}
	}

	return ValidationResult{
		OK:      len(reasons) == 0,
		Reasons: reasons,
	}
}

func (v *DefaultValidator) Must(result ValidationResult) error {
	if result.OK {
		return nil
	}
	return fmt.Errorf("validation failed: %v", result.Reasons)
}

func containsUncertaintyLanguage(answer string) bool {
	phrases := []string{
		"not enough evidence",
		"cannot verify",
		"can't verify",
		"appears likely",
		"based on the data currently in jax",
		"may",
		"might",
		"uncertain",
	}
	for _, phrase := range phrases {
		if containsInsensitive(answer, phrase) {
			return true
		}
	}
	return false
}

func containsCertaintyLanguage(answer string) bool {
	phrases := []string{
		"definitely",
		"certainly",
		"guaranteed",
		"no doubt",
		"will definitely",
		"clearly will",
		"must buy",
		"must sell",
	}
	for _, phrase := range phrases {
		if containsInsensitive(answer, phrase) {
			return true
		}
	}
	return false
}

func containsUnsupportedPriceTarget(answer string) bool {
	return priceTargetPattern.MatchString(answer)
}
