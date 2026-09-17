package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"jax-trading-assistant/internal/modules/exploratorypaper"
	"jax-trading-assistant/libs/risk"
	"jax-trading-assistant/libs/runtimepolicy"
)

const (
	paper02CalendarPath = "config/paper-02/session-calendar-us-equities-2026-v1.json"
	paper02UniversePath = "config/paper-02/eligible-universe-us-equities-2026-v1.json"
	paper02EntryPath    = "config/paper-02/entry-policy-v1.json"
	riskPolicyPath      = "config/risk-constraints.json"
)

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func resolveRuntimePath(configured string) string {
	if _, err := os.Stat(configured); err == nil {
		return configured
	}
	if _, err := os.Stat("../../" + configured); err == nil {
		return "../../" + configured
	}
	return configured
}

func parseBoolDefault(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	return strconv.ParseBool(raw)
}

func parseFloatDefault(key string, fallback float64) (float64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(raw, 64)
}

// loadPaper02RuntimeReadiness is a read-only projection. It never creates a
// pilot, admits an opportunity, invokes a venue, or contacts an event source.
func loadPaper02RuntimeReadiness() exploratorypaper.Paper02Readiness {
	input := exploratorypaper.Paper02ReadinessInput{
		RuntimeMode:        strings.ToUpper(runtimepolicy.CurrentMode().String()),
		ExecutionAuthority: "NONE",
	}

	calendarPath := resolveRuntimePath(envOrDefault("PAPER_SESSION_CALENDAR_FILE", paper02CalendarPath))
	calendar, err := exploratorypaper.LoadVersionedSessionCalendar(calendarPath)
	if err != nil {
		input.CalendarError = err
	} else {
		input.Calendar = &calendar
	}

	configuredSource, err := loadWorldMonitorPullConfig(os.LookupEnv)
	if err != nil {
		input.EventSource = exploratorypaper.ProspectiveEventSourceReadiness{SourceIdentity: worldMonitorPullConsumer, Reason: err.Error()}
	} else if !configuredSource.Enabled {
		input.EventSource = exploratorypaper.ProspectiveEventSourceReadiness{SourceIdentity: worldMonitorPullConsumer, Reason: "genuine prospective event intake is disabled"}
	} else {
		input.EventSource = exploratorypaper.ProspectiveEventSourceReadiness{Ready: true, SourceIdentity: worldMonitorPullConsumer, Endpoint: configuredSource.Endpoint}
	}

	universePath := resolveRuntimePath(envOrDefault("PAPER02_ELIGIBLE_UNIVERSE_FILE", paper02UniversePath))
	universe, err := exploratorypaper.LoadEligibleUniverse(universePath)
	if err != nil {
		input.UniverseError = err
	} else {
		input.Universe = &universe
	}

	resolvedRiskPolicyPath := resolveRuntimePath(riskPolicyPath)
	policy, err := risk.LoadPolicy(resolvedRiskPolicyPath)
	if err != nil {
		input.RiskPolicyError = err
	} else {
		maxPositionPct, pctErr := parseFloatDefault("MAX_POSITION_PCT", 0.20)
		if pctErr != nil {
			input.RiskPolicyError = fmt.Errorf("MAX_POSITION_PCT is invalid: %w", pctErr)
		} else {
			identity := &exploratorypaper.RiskPolicyIdentity{
				Version:                policy.Version,
				MaxRiskPerTrade:        policy.Position.MaxRiskPerTrade,
				MaxPositionPercentage:  maxPositionPct,
				MaxLeverage:            policy.Position.MaxLeverage,
				MaxConcurrentPositions: policy.Portfolio.MaxPositions,
				MaxPositionValue:       policy.Portfolio.MaxPositionSize,
				MaxAggregateRisk:       policy.Portfolio.MaxPortfolioRisk,
				MaxConcentration:       policy.Portfolio.MaxSectorExposure,
				MaxCorrelatedExposure:  policy.Portfolio.MaxCorrelatedExposure,
				RiskKillBehavior:       "portfolio-risk-and-drawdown-fail-closed",
			}
			identity.Hash = identity.ContentHash()
			input.RiskPolicy = identity
		}
	}

	entryPath := resolveRuntimePath(envOrDefault("PAPER02_ENTRY_POLICY_FILE", paper02EntryPath))
	entryData, err := os.ReadFile(entryPath)
	if err != nil {
		input.EntryPolicyError = err
	} else {
		var entry exploratorypaper.EntryPolicyIdentity
		if err := json.Unmarshal(entryData, &entry); err != nil {
			input.EntryPolicyError = err
		} else if hash, _, hashErr := exploratorypaper.HashJSONFile(entryPath); hashErr != nil {
			input.EntryPolicyError = hashErr
		} else {
			entry.Hash = hash
			input.EntryPolicy = &entry
		}
	}

	brokerAllowed, brokerErr := parseBoolDefault("BROKER_EXECUTION_ALLOWED", false)
	if brokerErr != nil {
		brokerAllowed = true
	}
	input.BrokerExecutionAllowed = brokerAllowed
	maxLeverage, leverageErr := parseFloatDefault("MAX_LEVERAGE", 1)
	if leverageErr != nil {
		maxLeverage = 2
	}
	input.MaximumLeverage = maxLeverage
	return exploratorypaper.AssessPaper02Readiness(input)
}
