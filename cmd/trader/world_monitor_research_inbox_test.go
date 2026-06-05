package main

import (
	"testing"
	"time"
)

func TestWorldMonitorResearchInbox_DedupeKeyStable(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)

	first := worldMonitorDedupeKey(trigger)
	second := worldMonitorDedupeKey(trigger)

	if first == "" {
		t.Fatal("expected non-empty dedupe key")
	}
	if first != second {
		t.Fatalf("expected stable dedupe key, got %q and %q", first, second)
	}
}

func TestWorldMonitorResearchInbox_MapsAcceptedTriggerToEventInput(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	service := newWorldMonitorResearchInboxService(nil)

	input := service.toPersistEventInput(trigger)

	if input.SourceID != "world-monitor" {
		t.Fatalf("SourceID = %q, want world-monitor", input.SourceID)
	}
	if input.EventKind != "research_trigger" {
		t.Fatalf("EventKind = %q, want research_trigger", input.EventKind)
	}
	if input.PrimarySymbol != "QQQ" {
		t.Fatalf("PrimarySymbol = %q, want QQQ", input.PrimarySymbol)
	}
	for _, symbol := range []string{"QQQ", "SPY", "TLT"} {
		if !containsString(input.Symbols, symbol) {
			t.Fatalf("Symbols missing %s: %v", symbol, input.Symbols)
		}
	}
	for _, key := range []string{"eventType", "sourceUrls", "sourceCount", "assetThemes", "confidenceReasons", "worldMonitorEventId", "mappingReason"} {
		if _, ok := input.Attributes[key]; !ok {
			t.Fatalf("Attributes missing %q: %#v", key, input.Attributes)
		}
	}
}

func TestWorldMonitorResearchInbox_LowSeverityIsIgnoredAndNotPersisted(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.Severity = "low"
	service := newWorldMonitorResearchInboxService(nil)
	service.now = func() time.Time { return now }

	status := service.statusForAcceptedTrigger(trigger)

	if status != "ignored" {
		t.Fatalf("status = %q, want ignored", status)
	}
	if service.shouldPersistAcceptedTrigger(trigger) {
		t.Fatal("low severity trigger must not be persisted to event_normalized by default")
	}
}

func TestWorldMonitorResearchInbox_InvalidTriggerReturnsRejectedReceipt(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.SourceURLs = nil
	service := newWorldMonitorResearchInboxService(nil)
	service.now = func() time.Time { return now }

	receipt := service.rejectedReceipt(trigger)

	if receipt.Status != "rejected" {
		t.Fatalf("Status = %q, want rejected", receipt.Status)
	}
	if receipt.RejectionReason == "" {
		t.Fatal("expected rejection reason")
	}
}
