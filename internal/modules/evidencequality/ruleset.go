package evidencequality

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func LoadRuleset(path string) (Ruleset, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Ruleset{}, fmt.Errorf("read evidence quality ruleset: %w", err)
	}
	var rules Ruleset
	if err := json.Unmarshal(raw, &rules); err != nil {
		return Ruleset{}, fmt.Errorf("decode evidence quality ruleset: %w", err)
	}
	if strings.TrimSpace(rules.Version) == "" || rules.PrimaryAnchor != "receipt" {
		return Ruleset{}, fmt.Errorf("ruleset requires a version and receipt primary anchor")
	}
	if rules.MinimumComparisonGroupSize < 2 || rules.BootstrapIterations <= 0 || rules.PermutationIterations <= 0 {
		return Ruleset{}, fmt.Errorf("ruleset statistical parameters are invalid")
	}
	return rules, nil
}
