package runtimepolicy

import (
	"fmt"
	"os"
	"strings"
)

type Mode string

const (
	ModeDev      Mode = "dev"
	ModeTest     Mode = "test"
	ModeResearch Mode = "research"
	ModePaper    Mode = "paper"
	ModeLive     Mode = "live"
)

func ParseMode(raw string) (Mode, error) {
	v := strings.TrimSpace(strings.ToLower(raw))
	switch v {
	case "", "dev":
		return ModeDev, nil
	case "test":
		return ModeTest, nil
	case "research":
		return ModeResearch, nil
	case "paper":
		return ModePaper, nil
	case "live", "prod", "production":
		return ModeLive, nil
	default:
		return "", fmt.Errorf("invalid runtime mode %q (valid: dev,test,research,paper,live)", raw)
	}
}

func CurrentMode() Mode {
	for _, key := range []string{"JAX_RUNTIME_MODE", "APP_RUNTIME_MODE", "APP_ENV", "ENV"} {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			mode, err := ParseMode(raw)
			if err == nil {
				return mode
			}
		}
	}
	return ModeDev
}

func (m Mode) String() string { return string(m) }

func (m Mode) AllowsSyntheticBacktest() bool {
	return m == ModeDev || m == ModeTest
}

func (m Mode) EnforceStrictProviderPolicy() bool {
	return m == ModePaper || m == ModeLive
}

func (m Mode) EnforceNoSyntheticTruthPaths() bool {
	return m == ModeResearch || m == ModePaper || m == ModeLive
}
