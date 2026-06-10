package macroevents

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceIngestPersistsValidEventOnce(t *testing.T) {
	now := fixedServiceNow()
	store := newFakeStore()
	service := NewService(store)
	service.now = func() time.Time { return now }

	actual := 172000.0
	expected := 85000.0
	receipt, err := service.Ingest(context.Background(), validServiceEvent(now, EventTypeUSNonfarmPayrolls, actual, &expected))
	if err != nil {
		t.Fatalf("Ingest returned error: %v", err)
	}

	if receipt.Status != StatusAccepted {
		t.Fatalf("status = %q, want %q", receipt.Status, StatusAccepted)
	}
	if receipt.MacroEventID == "" {
		t.Fatal("expected macro event id")
	}
	if len(receipt.MappedETFs) != 2 {
		t.Fatalf("mapped etfs = %#v, want 2 symbols", receipt.MappedETFs)
	}
	if store.saveCalls != 1 {
		t.Fatalf("saveCalls = %d, want 1", store.saveCalls)
	}
	if len(store.saved[0].Mappings) != 2 {
		t.Fatalf("saved mappings = %#v, want two mappings", store.saved[0].Mappings)
	}
}

func TestServiceIngestReturnsDuplicateReceipt(t *testing.T) {
	now := fixedServiceNow()
	store := newFakeStore()
	store.existing = &StoredEvent{ID: "macro-existing", Status: StatusAccepted, Mappings: []ETFMapping{{Symbol: "QQQ"}}}
	service := NewService(store)
	service.now = func() time.Time { return now }

	actual := 172000.0
	expected := 85000.0
	receipt, err := service.Ingest(context.Background(), validServiceEvent(now, EventTypeUSNonfarmPayrolls, actual, &expected))
	if err != nil {
		t.Fatalf("Ingest returned error: %v", err)
	}

	if !receipt.Duplicate {
		t.Fatal("expected duplicate receipt")
	}
	if receipt.MacroEventID != "macro-existing" {
		t.Fatalf("macro event id = %q, want macro-existing", receipt.MacroEventID)
	}
	if store.saveCalls != 0 {
		t.Fatalf("saveCalls = %d, want 0", store.saveCalls)
	}
}

func TestServiceIngestStoresLowConfidenceAsQuarantinedWithoutETFMap(t *testing.T) {
	now := fixedServiceNow()
	store := newFakeStore()
	service := NewService(store)
	service.now = func() time.Time { return now }

	actual := 1.0
	expected := 2.0
	input := validServiceEvent(now, EventTypeUSCPIHeadline, actual, &expected)
	input.Confidence = 0.49

	receipt, err := service.Ingest(context.Background(), input)
	if err != nil {
		t.Fatalf("Ingest returned error: %v", err)
	}

	if receipt.Status != StatusQuarantined {
		t.Fatalf("status = %q, want %q", receipt.Status, StatusQuarantined)
	}
	if len(receipt.MappedETFs) != 0 {
		t.Fatalf("mapped etfs = %#v, want none", receipt.MappedETFs)
	}
	if len(store.saved[0].Mappings) != 0 {
		t.Fatalf("stored mappings = %#v, want none", store.saved[0].Mappings)
	}
}

func TestServiceIngestStoresInvalidEventAsRejected(t *testing.T) {
	now := fixedServiceNow()
	store := newFakeStore()
	service := NewService(store)
	service.now = func() time.Time { return now }

	actual := 1.0
	expected := 2.0
	input := validServiceEvent(now, EventType("UK_CPI"), actual, &expected)

	receipt, err := service.Ingest(context.Background(), input)
	if err != nil {
		t.Fatalf("Ingest returned error: %v", err)
	}

	if receipt.Status != StatusRejected {
		t.Fatalf("status = %q, want %q", receipt.Status, StatusRejected)
	}
	if receipt.RejectionReason == "" {
		t.Fatal("expected rejection reason")
	}
	if len(store.saved[0].Mappings) != 0 {
		t.Fatalf("stored mappings = %#v, want none", store.saved[0].Mappings)
	}
}

func TestServiceIngestDoesNotExposeTradingHooks(t *testing.T) {
	storeType := newFakeStore()
	var _ eventStore = storeType
}

func fixedServiceNow() time.Time {
	return time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
}

func validServiceEvent(now time.Time, eventType EventType, actual float64, expected *float64) EventInput {
	return EventInput{
		Source:        "calendar",
		SourceEventID: "macro-1",
		EventType:     eventType,
		Region:        "US",
		EventTimeUTC:  now.Add(-5 * time.Minute),
		Headline:      "Macro event",
		Summary:       "Macro event summary",
		ActualValue:   &actual,
		ExpectedValue: expected,
		Unit:          "jobs",
		Direction:     DirectionHawkishRates,
		Confidence:    0.9,
		AffectedETFs:  []string{"QQQ", "SPY"},
		RawPayload:    map[string]any{"provider": "calendar"},
	}
}

type fakeStore struct {
	existing  *StoredEvent
	saved     []StoredEvent
	saveCalls int
}

func newFakeStore() *fakeStore {
	return &fakeStore{}
}

func (s *fakeStore) FindBySourceEventID(_ context.Context, _, _ string) (StoredEvent, bool, error) {
	if s.existing == nil {
		return StoredEvent{}, false, nil
	}
	return *s.existing, true, nil
}

func (s *fakeStore) Save(_ context.Context, event StoredEvent) (StoredEvent, error) {
	s.saveCalls++
	if event.Input.SourceEventID == "store-error" {
		return StoredEvent{}, errors.New("store failed")
	}
	event.ID = "macro-new"
	s.saved = append(s.saved, event)
	return event, nil
}
