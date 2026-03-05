package utcp

import (
	"errors"
	"fmt"
	"strings"

	"jax-trading-assistant/libs/runtimepolicy"
)

// ValidateRuntimeProviderPolicy applies runtime-mode safety checks to providers.json.
// In strict modes (paper/live), providers flagged synthetic are rejected at startup.
func ValidateRuntimeProviderPolicy(mode runtimepolicy.Mode, cfg ProvidersConfig) error {
	synthetic := make([]string, 0, 4)
	truthPathViolations := make([]string, 0, 4)
	for _, p := range cfg.Providers {
		dsType := strings.ToLower(strings.TrimSpace(p.DataSourceType))
		isSynthetic := p.IsSynthetic || dsType == "synthetic"
		if mode.EnforceStrictProviderPolicy() && isSynthetic {
			synthetic = append(synthetic, p.ID)
		}
		if mode.EnforceNoSyntheticTruthPaths() && isTruthPathProvider(p) {
			if dsType != "real" || isSynthetic {
				truthPathViolations = append(truthPathViolations, fmt.Sprintf("%s(%s)", p.ID, dsTypeOrUnknown(dsType)))
			}
		}
	}

	errs := make([]string, 0, 2)
	if len(synthetic) > 0 {
		errs = append(errs, fmt.Sprintf("runtime mode %s blocks synthetic providers: %s", mode, strings.Join(synthetic, ",")))
	}
	if len(truthPathViolations) > 0 {
		errs = append(errs, fmt.Sprintf("runtime mode %s requires real data_source_type for truth-path providers: %s", mode, strings.Join(truthPathViolations, ",")))
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

func isTruthPathProvider(p ProviderConfig) bool {
	id := strings.ToLower(strings.TrimSpace(p.ID))
	switch id {
	case "market-data", "memory", "dexter", "broker":
		return true
	default:
		return false
	}
}

func dsTypeOrUnknown(dsType string) string {
	dsType = strings.TrimSpace(strings.ToLower(dsType))
	if dsType == "" {
		return "unknown"
	}
	return dsType
}
