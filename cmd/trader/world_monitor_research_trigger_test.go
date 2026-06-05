package main

import (
	"strings"
	"testing"
	"time"
)

func validWorldMonitorResearchTrigger(now time.Time) worldMonitorResearchTrigger {
	return worldMonitorResearchTrigger{
		Source:               "world-monitor",
		SourceEventID:        "wm-20260605-001",
		EventType:            "macro_rates",
		Headline:             "Fed signals possible rate cuts after weaker inflation print",
		Summary:              "Multiple sources report softer inflation data and falling yields.",
		SourceURLs:           []string{"https://example.com/source-1", "https://example.com/source-2"},
		SourceCount:          2,
		TimestampUTC:         now.Add(-20 * time.Minute),
		Region:               "US",
		PossibleAffectedETFs: []string{"QQQ", "SPY", "TLT"},
		AssetThemes:          []string{"rates", "growth_equities", "bonds"},
		Severity:             "high",
		SourceTier:           "tier2",
		Confidence:           0.72,
		ConfidenceReasons: []string{
			"2 independent sources",
			"event is macro/rates related",
			"likely affects QQQ, TLT and SPY",
		},
		Reason: "Rates-sensitive headline likely relevant to QQQ, SPY and TLT.",
		RawPayload: map[string]any{
			"cluster_id": "fed-rates-20260605",
		},
	}
}

func TestWorldMonitorResearchTrigger_ValidPayloadPasses(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	result := validateWorldMonitorResearchTrigger(validWorldMonitorResearchTrigger(now), now)
	if !result.Valid {
		t.Fatalf("expected valid trigger, got rejection %q", result.Reason)
	}
}

func TestWorldMonitorResearchTrigger_RejectsMissingTimestamp(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.TimestampUTC = time.Time{}

	result := validateWorldMonitorResearchTrigger(trigger, now)
	requireWorldMonitorRejection(t, result, "timestamp_utc")
}

func TestWorldMonitorResearchTrigger_RejectsMissingSourceURLs(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.SourceURLs = nil

	result := validateWorldMonitorResearchTrigger(trigger, now)
	requireWorldMonitorRejection(t, result, "source_urls")
}

func TestWorldMonitorResearchTrigger_RejectsLowSourceCountUnlessTier1(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.SourceCount = 1
	trigger.SourceTier = "tier2"

	result := validateWorldMonitorResearchTrigger(trigger, now)
	requireWorldMonitorRejection(t, result, "source_count")

	trigger.SourceTier = "tier1"
	result = validateWorldMonitorResearchTrigger(trigger, now)
	if !result.Valid {
		t.Fatalf("expected tier1 single-source trigger to pass, got %q", result.Reason)
	}
}

func TestWorldMonitorResearchTrigger_RejectsStaleEvent(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.TimestampUTC = now.Add(-25 * time.Hour)

	result := validateWorldMonitorResearchTrigger(trigger, now)
	requireWorldMonitorRejection(t, result, "stale")
}

func TestWorldMonitorResearchTrigger_RejectsTradeInstructionLanguage(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.Headline = "Buy QQQ now"

	result := validateWorldMonitorResearchTrigger(trigger, now)
	requireWorldMonitorRejection(t, result, "trade instruction")
}

func TestWorldMonitorResearchTrigger_RejectsUnknownLowConfidenceEvent(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.EventType = "unknown"
	trigger.Confidence = 0.39

	result := validateWorldMonitorResearchTrigger(trigger, now)
	requireWorldMonitorRejection(t, result, "unknown")
}

func TestWorldMonitorResearchTrigger_RejectsNonETFMapping(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.PossibleAffectedETFs = []string{"AAPL"}

	result := validateWorldMonitorResearchTrigger(trigger, now)
	requireWorldMonitorRejection(t, result, "ETF")
}

func TestWorldMonitorResearchTrigger_RejectsConfidenceWithoutReasons(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.ConfidenceReasons = nil

	result := validateWorldMonitorResearchTrigger(trigger, now)
	requireWorldMonitorRejection(t, result, "confidence_reasons")
}

func TestWorldMonitorResearchTrigger_RejectsRuntimeOverridePayload(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.RawPayload["runtime_mode"] = "live"

	result := validateWorldMonitorResearchTrigger(trigger, now)
	requireWorldMonitorRejection(t, result, "runtime")
}

func requireWorldMonitorRejection(t *testing.T, result worldMonitorValidationResult, want string) {
	t.Helper()
	if result.Valid {
		t.Fatalf("expected trigger rejection containing %q", want)
	}
	if !strings.Contains(strings.ToLower(result.Reason), strings.ToLower(want)) {
		t.Fatalf("expected rejection containing %q, got %q", want, result.Reason)
	}
}
