package main

import (
	"testing"

	"jax-trading-assistant/libs/runtimepolicy"
)

func TestRequireApprovedStrategiesFailsInPaperAndLiveWhenEmpty(t *testing.T) {
	t.Parallel()

	modes := []runtimepolicy.Mode{
		runtimepolicy.ModePaper,
		runtimepolicy.ModeLive,
	}
	for _, mode := range modes {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()

			err := requireApprovedStrategies(mode, 0)
			if err == nil {
				t.Fatalf("expected empty approved strategy set to fail in %s mode", mode)
			}
		})
	}
}

func TestRequireApprovedStrategiesAllowsEmptyRegistryOutsidePaperAndLive(t *testing.T) {
	t.Parallel()

	modes := []runtimepolicy.Mode{
		runtimepolicy.ModeDev,
		runtimepolicy.ModeTest,
		runtimepolicy.ModeResearch,
	}
	for _, mode := range modes {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()

			if err := requireApprovedStrategies(mode, 0); err != nil {
				t.Fatalf("expected empty approved strategy set to be allowed in %s mode: %v", mode, err)
			}
		})
	}
}

func TestRequireEventProvidersFailsInPaperModeWhenCredentialsMissing(t *testing.T) {
	t.Parallel()

	err := requireEventProviders(runtimepolicy.ModePaper, false, false)
	if err == nil {
		t.Fatal("expected paper mode to fail when event providers are missing")
	}
}

func TestRequireEventProvidersAllowsMissingCredentialsOutsideStrictModes(t *testing.T) {
	t.Parallel()

	modes := []runtimepolicy.Mode{
		runtimepolicy.ModeDev,
		runtimepolicy.ModeTest,
		runtimepolicy.ModeResearch,
	}
	for _, mode := range modes {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()

			if err := requireEventProviders(mode, false, false); err != nil {
				t.Fatalf("expected missing event providers to be tolerated in %s mode: %v", mode, err)
			}
		})
	}
}
