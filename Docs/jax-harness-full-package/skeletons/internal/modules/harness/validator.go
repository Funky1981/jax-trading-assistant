package harness

import "fmt"

type ValidationResult struct {
    OK      bool
    Reasons []string
}

type Validator struct{}

func NewValidator() *Validator { return &Validator{} }

func (v *Validator) ValidateAnswer(policy Policy, bundle EvidenceBundle, answer string) ValidationResult {
    var reasons []string

    if err := policy.CheckAnswerAllowed(answer); err != nil {
        reasons = append(reasons, err.Error())
    }

    if bundle.IsWeak() {
        if !(containsInsensitive(answer, "not enough evidence") ||
             containsInsensitive(answer, "cannot verify") ||
             containsInsensitive(answer, "appears likely") ||
             containsInsensitive(answer, "based on the data currently in jax")) {
            reasons = append(reasons, "weak evidence answer missing uncertainty language")
        }
    }

    return ValidationResult{
        OK: len(reasons) == 0,
        Reasons: reasons,
    }
}

func (v *Validator) Must(result ValidationResult) error {
    if result.OK {
        return nil
    }
    return fmt.Errorf("validation failed: %v", result.Reasons)
}
