package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/instruments"
)

const worldMonitorFreshnessWindow = 24 * time.Hour

type worldMonitorResearchTrigger struct {
	Source                string         `json:"source"`
	SourceEventID         string         `json:"source_event_id"`
	EventType             string         `json:"event_type"`
	Headline              string         `json:"headline"`
	Summary               string         `json:"summary"`
	SourceURLs            []string       `json:"source_urls"`
	SourceCount           int            `json:"source_count"`
	TimestampUTC          time.Time      `json:"timestamp_utc"`
	Region                string         `json:"region"`
	PossibleAffectedETFs  []string       `json:"possible_affected_etfs"`
	AssetThemes           []string       `json:"asset_themes"`
	Severity              string         `json:"severity"`
	SourceTier            string         `json:"source_tier"`
	Confidence            float64        `json:"confidence"`
	ConfidenceReasons     []string       `json:"confidence_reasons"`
	Reason                string         `json:"reason"`
	RawPayload            map[string]any `json:"raw_payload"`
	IsSynthetic           *bool          `json:"is_synthetic,omitempty"`
	CollectionTimestamp   *time.Time     `json:"collection_timestamp_utc,omitempty"`
	DiscoveryMethod       string         `json:"discovery_method,omitempty"`
	AnalysisProvider      string         `json:"analysis_provider,omitempty"`
	AnalysisModel         string         `json:"analysis_model,omitempty"`
	DeterministicAnalysis string         `json:"deterministic_analysis,omitempty"`
	SyntheticReason       string         `json:"-"`
	AllowStalePublication bool           `json:"-"`
}

type worldMonitorResearchReceipt struct {
	InboxID         string `json:"inbox_id,omitempty"`
	EventID         string `json:"event_id,omitempty"`
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	Duplicate       bool   `json:"duplicate"`
}

type worldMonitorValidationResult struct {
	Valid  bool
	Reason string
}

func validateWorldMonitorResearchTrigger(trigger worldMonitorResearchTrigger, now time.Time) worldMonitorValidationResult {
	if strings.TrimSpace(trigger.Source) == "" {
		return rejectWorldMonitorTrigger("source is required")
	}
	if strings.TrimSpace(trigger.SourceEventID) == "" {
		return rejectWorldMonitorTrigger("source_event_id is required")
	}
	if !allowedWorldMonitorEventType(trigger.EventType) {
		return rejectWorldMonitorTrigger("event_type is not allowed")
	}
	if strings.TrimSpace(trigger.Headline) == "" {
		return rejectWorldMonitorTrigger("headline is required")
	}
	if len(nonEmptyWorldMonitorStrings(trigger.SourceURLs)) == 0 {
		return rejectWorldMonitorTrigger("source_urls are required")
	}
	for _, rawURL := range nonEmptyWorldMonitorStrings(trigger.SourceURLs) {
		parsed, err := url.ParseRequestURI(rawURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return rejectWorldMonitorTrigger("source_urls must contain absolute HTTP(S) URLs")
		}
	}
	if trigger.TimestampUTC.IsZero() {
		return rejectWorldMonitorTrigger("timestamp_utc is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !trigger.AllowStalePublication && trigger.TimestampUTC.UTC().Before(now.UTC().Add(-worldMonitorFreshnessWindow)) {
		return rejectWorldMonitorTrigger("stale event exceeds freshness window")
	}
	if strings.TrimSpace(trigger.Reason) == "" {
		return rejectWorldMonitorTrigger("reason is required")
	}
	if trigger.Confidence <= 0 || trigger.Confidence > 1 {
		return rejectWorldMonitorTrigger("confidence must be between 0 and 1")
	}
	if len(nonEmptyWorldMonitorStrings(trigger.ConfidenceReasons)) == 0 {
		return rejectWorldMonitorTrigger("confidence_reasons are required")
	}
	if !allowedWorldMonitorSeverity(trigger.Severity) {
		return rejectWorldMonitorTrigger("severity is not allowed")
	}
	if !allowedWorldMonitorSourceTier(trigger.SourceTier) {
		return rejectWorldMonitorTrigger("source_tier is not allowed")
	}
	if trigger.SourceCount < 2 && !strings.EqualFold(strings.TrimSpace(trigger.SourceTier), "tier1") {
		return rejectWorldMonitorTrigger("source_count is below threshold")
	}
	if strings.EqualFold(strings.TrimSpace(trigger.EventType), "unknown") && trigger.Confidence < 0.5 {
		return rejectWorldMonitorTrigger("unknown event_type requires higher confidence")
	}
	if containsWorldMonitorTradeInstruction(trigger) {
		return rejectWorldMonitorTrigger("payload contains trade instruction language")
	}
	if containsWorldMonitorRuntimeOverride(trigger.RawPayload) {
		return rejectWorldMonitorTrigger("payload contains runtime override fields")
	}
	if err := validateWorldMonitorETFMapping(trigger.PossibleAffectedETFs); err != nil {
		return rejectWorldMonitorTrigger(err.Error())
	}
	return worldMonitorValidationResult{Valid: true}
}

func normalizeWorldMonitorResearchTrigger(trigger worldMonitorResearchTrigger) worldMonitorResearchTrigger {
	trigger.IsSynthetic, trigger.SyntheticReason = classifyWorldMonitorSyntheticProvenance(trigger)
	if strings.TrimSpace(trigger.Severity) == "" {
		trigger.Severity = rawWorldMonitorString(trigger.RawPayload, "threat_level")
	}
	if strings.TrimSpace(trigger.Severity) == "" {
		trigger.Severity = "medium"
	}

	if strings.TrimSpace(trigger.SourceTier) == "" {
		if trigger.SourceCount <= 1 {
			trigger.SourceTier = "tier1"
		} else {
			trigger.SourceTier = "tier2"
		}
	}

	if len(nonEmptyWorldMonitorStrings(trigger.ConfidenceReasons)) == 0 {
		reasons := []string{}
		if reason := strings.TrimSpace(trigger.Reason); reason != "" {
			reasons = append(reasons, reason)
		}
		if trigger.SourceCount > 0 {
			reasons = append(reasons, fmt.Sprintf("%d source%s reported by World Monitor", trigger.SourceCount, pluralSuffix(trigger.SourceCount)))
		}
		if severity := strings.TrimSpace(trigger.Severity); severity != "" {
			reasons = append(reasons, "World Monitor severity: "+strings.ToLower(severity))
		}
		trigger.ConfidenceReasons = reasons
	}

	return trigger
}

func classifyWorldMonitorSyntheticProvenance(trigger worldMonitorResearchTrigger) (*bool, string) {
	if trigger.IsSynthetic != nil {
		value := *trigger.IsSynthetic
		if value {
			return &value, "explicit World Monitor API is_synthetic=true"
		}
		return &value, "explicit World Monitor API is_synthetic=false"
	}
	if value, ok := trigger.RawPayload["localProof"].(bool); ok && value {
		return boolPointer(true), "trusted raw_payload.localProof=true"
	}
	if strings.EqualFold(strings.TrimSpace(trigger.Source), "world-monitor-local-proof") {
		return boolPointer(true), "exact local-proof source world-monitor-local-proof"
	}
	return boolPointer(false), "ordinary World Monitor input"
}

func boolPointer(value bool) *bool {
	return &value
}

func rawWorldMonitorString(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}
	value, ok := payload[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func rejectWorldMonitorTrigger(reason string) worldMonitorValidationResult {
	return worldMonitorValidationResult{Valid: false, Reason: reason}
}

func allowedWorldMonitorEventType(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "macro_rates", "inflation", "central_bank", "geopolitical", "energy_oil",
		"semiconductor_ai", "financial_credit", "commodity_shock", "cyber_outage",
		"market_panic", "supply_chain", "unknown":
		return true
	default:
		return false
	}
}

func allowedWorldMonitorSeverity(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func allowedWorldMonitorSourceTier(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "tier1", "tier2", "tier3", "unknown":
		return true
	default:
		return false
	}
}

func containsWorldMonitorTradeInstruction(trigger worldMonitorResearchTrigger) bool {
	text := strings.ToLower(strings.Join([]string{
		trigger.Headline,
		trigger.Summary,
		trigger.Reason,
		rawPayloadText(trigger.RawPayload),
	}, " "))
	for _, phrase := range []string{"buy ", "sell ", "short ", "go long", "execute", "broker_order", "trade_instruction"} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

func containsWorldMonitorRuntimeOverride(payload map[string]any) bool {
	for _, key := range flattenedPayloadKeys(payload, "") {
		for _, segment := range strings.Split(strings.ToLower(key), ".") {
			switch segment {
			case "runtime_mode", "allow_live_trading", "execution_enabled", "execution_worker_enabled",
				"approval", "approval_instruction", "broker", "broker_order", "execution", "execution_instruction",
				"order", "order_intent", "position_size", "risk_override":
				return true
			}
		}
	}
	return false
}

func flattenedPayloadKeys(payload map[string]any, prefix string) []string {
	keys := make([]string, 0, len(payload))
	for key, value := range payload {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if prefix != "" {
			normalized = prefix + "." + normalized
		}
		keys = append(keys, normalized)
		if nested, ok := value.(map[string]any); ok {
			keys = append(keys, flattenedPayloadKeys(nested, normalized)...)
		}
	}
	return keys
}

func validateWorldMonitorETFMapping(symbols []string) error {
	catalog, err := instruments.LoadDefaultCatalog()
	if err != nil {
		return fmt.Errorf("ETF catalog unavailable: %w", err)
	}
	for _, symbol := range nonEmptyWorldMonitorStrings(symbols) {
		if !catalog.IsKnownETF(symbol) {
			return fmt.Errorf("ETF mapping rejected: %s is not an allowed ETF", strings.ToUpper(strings.TrimSpace(symbol)))
		}
		result := catalog.Evaluate(symbol, "paper")
		if !result.Allowed {
			return fmt.Errorf("ETF mapping rejected: %s: %s", strings.ToUpper(strings.TrimSpace(symbol)), result.ReasonCode)
		}
	}
	return nil
}

func rawPayloadText(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(raw)
}

func nonEmptyWorldMonitorStrings(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
