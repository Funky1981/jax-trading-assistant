package utcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts"
)

type fakeMemoryStore struct {
	lastBank   string
	lastItem   contracts.MemoryItem
	lastQuery  contracts.MemoryQuery
	lastParams contracts.ReflectionParams
	retained   int
}

func (s *fakeMemoryStore) Ping(context.Context) error { return nil }

func (s *fakeMemoryStore) Retain(_ context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	s.lastBank = bank
	s.lastItem = item
	s.retained++
	return "mem_1", nil
}

func (s *fakeMemoryStore) Recall(_ context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	s.lastBank = bank
	s.lastQuery = query
	return []contracts.MemoryItem{queryResultItem()}, nil
}

func (s *fakeMemoryStore) Reflect(_ context.Context, bank string, params contracts.ReflectionParams) ([]contracts.MemoryItem, error) {
	s.lastBank = bank
	s.lastParams = params
	return []contracts.MemoryItem{queryResultItem()}, nil
}

func TestMemoryTools_RetainRecallReflect(t *testing.T) {
	store := &fakeMemoryStore{}
	registry := NewLocalRegistry()
	if err := RegisterMemoryTools(registry, store); err != nil {
		t.Fatalf("register: %v", err)
	}

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{{ID: MemoryProviderID, Transport: "local"}},
	}, WithLocalRegistry(registry))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	svc := NewMemoryService(client)
	retainOut, err := svc.Retain(context.Background(), contracts.MemoryRetainRequest{
		Bank: "trade_decisions",
		Item: contracts.MemoryItem{
			TS:      time.Now().UTC(),
			Type:    "decision",
			Summary: "test",
			Tags:    []string{"EARNINGS"},
			Data:    map[string]any{"confidence": 0.5},
			Source:  &contracts.MemorySource{System: "test"},
		},
	})
	if err != nil {
		t.Fatalf("retain: %v", err)
	}
	if retainOut.ID == "" {
		t.Fatalf("expected id")
	}
	if store.retained != 1 {
		t.Fatalf("expected retain to be called")
	}
	if len(store.lastItem.Tags) != 1 || store.lastItem.Tags[0] != "earnings" {
		t.Fatalf("expected tags normalized, got %#v", store.lastItem.Tags)
	}

	recallOut, err := svc.Recall(context.Background(), contracts.MemoryRecallRequest{
		Bank: "trade_decisions",
		Query: contracts.MemoryQuery{
			Q:     "gap",
			Tags:  []string{"Gap"},
			Limit: 5,
		},
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(recallOut.Items) != 1 {
		t.Fatalf("expected 1 item")
	}
	if len(store.lastQuery.Tags) != 1 || store.lastQuery.Tags[0] != "gap" {
		t.Fatalf("expected query tags normalized, got %#v", store.lastQuery.Tags)
	}

	reflectOut, err := svc.Reflect(context.Background(), contracts.MemoryReflectRequest{
		Bank: "trade_decisions",
		Params: contracts.ReflectionParams{
			Query:      "what worked",
			WindowDays: 7,
		},
	})
	if err != nil {
		t.Fatalf("reflect: %v", err)
	}
	if len(reflectOut.Items) != 1 {
		t.Fatalf("expected 1 item")
	}
}

func TestMemoryTools_InvalidPayloads(t *testing.T) {
	store := &fakeMemoryStore{}
	registry := NewLocalRegistry()
	if err := RegisterMemoryTools(registry, store); err != nil {
		t.Fatalf("register: %v", err)
	}

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{{ID: MemoryProviderID, Transport: "local"}},
	}, WithLocalRegistry(registry))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	svc := NewMemoryService(client)
	_, err = svc.Retain(context.Background(), contracts.MemoryRetainRequest{})
	if err == nil {
		t.Fatalf("expected error for missing bank/item")
	}

	_, err = svc.Recall(context.Background(), contracts.MemoryRecallRequest{})
	if err == nil {
		t.Fatalf("expected error for missing bank")
	}

	_, err = svc.Reflect(context.Background(), contracts.MemoryReflectRequest{Bank: "trade"})
	if err == nil {
		t.Fatalf("expected error for missing query")
	}
}

func queryResultItem() contracts.MemoryItem {
	return contracts.MemoryItem{
		TS:      time.Now().UTC(),
		Type:    "decision",
		Summary: "result",
		Data:    map[string]any{"ok": true},
		Source:  &contracts.MemorySource{System: "test"},
	}
}

func TestMemoryTools_StoreErrorPropagates(t *testing.T) {
	store := &errorMemoryStore{}
	registry := NewLocalRegistry()
	if err := RegisterMemoryTools(registry, store); err != nil {
		t.Fatalf("register: %v", err)
	}

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{{ID: MemoryProviderID, Transport: "local"}},
	}, WithLocalRegistry(registry))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	svc := NewMemoryService(client)
	_, err = svc.Retain(context.Background(), contracts.MemoryRetainRequest{
		Bank: "trade_decisions",
		Item: contracts.MemoryItem{
			TS:      time.Now().UTC(),
			Type:    "decision",
			Summary: "test",
			Data:    map[string]any{"a": 1},
			Source:  &contracts.MemorySource{System: "test"},
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

type errorMemoryStore struct{}

func (s *errorMemoryStore) Ping(context.Context) error { return nil }

func (s *errorMemoryStore) Retain(context.Context, string, contracts.MemoryItem) (contracts.MemoryID, error) {
	return "", errors.New("retain failed")
}

func (s *errorMemoryStore) Recall(context.Context, string, contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	return nil, errors.New("recall failed")
}

func (s *errorMemoryStore) Reflect(context.Context, string, contracts.ReflectionParams) ([]contracts.MemoryItem, error) {
	return nil, errors.New("reflect failed")
}
