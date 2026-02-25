package utcp

import (
	"fmt"
	"strings"

	"jax-trading-assistant/libs/runtimepolicy"
)

// ValidateRuntimeProviderPolicy applies runtime-mode safety checks to providers.json.
// In strict modes (paper/live), providers flagged synthetic are rejected at startup.
func ValidateRuntimeProviderPolicy(mode runtimepolicy.Mode, cfg ProvidersConfig) error {
	if !mode.EnforceStrictProviderPolicy() {
		return nil
	}

	synthetic := make([]string, 0, 4)
	for _, p := range cfg.Providers {
		dsType := strings.ToLower(strings.TrimSpace(p.DataSourceType))
		isSynthetic := p.IsSynthetic || dsType == "synthetic"
		if isSynthetic {
			synthetic = append(synthetic, p.ID)
		}
	}
	if len(synthetic) == 0 {
		return nil
	}
	return fmt.Errorf("runtime mode %s blocks synthetic providers: %s", mode, strings.Join(synthetic, ","))
}
