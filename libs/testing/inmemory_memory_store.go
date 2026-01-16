package testing

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"jax-trading-assistant/libs/contracts"
)

type InMemoryMemoryStore struct {
	mu    sync.RWMutex
	banks map[string][]contracts.MemoryItem
}

func NewInMemoryMemoryStore() *InMemoryMemoryStore {
	return &InMemoryMemoryStore{banks: make(map[string][]contracts.MemoryItem)}
}

func (s *InMemoryMemoryStore) Ping(_ context.Context) error {
	return nil
}

func (s *InMemoryMemoryStore) Retain(_ context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	bank = strings.TrimSpace(bank)
	if bank == "" {
		return "", errors.New("retain: bank is required")
	}
	if strings.TrimSpace(item.Type) == "" {
		return "", errors.New("retain: item.type is required")
	}
	if strings.TrimSpace(item.Summary) == "" {
		return "", errors.New("retain: item.summary is required")
	}
	if item.TS.IsZero() {
		item.TS = time.Now().UTC()
	}
	if item.ID == "" {
		item.ID = "mem_" + item.TS.UTC().Format("20060102T150405Z")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.banks[bank] = append(s.banks[bank], item)
	return contracts.MemoryID(item.ID), nil
}

func (s *InMemoryMemoryStore) Recall(_ context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	bank = strings.TrimSpace(bank)
	if bank == "" {
		return nil, errors.New("recall: bank is required")
	}

	s.mu.RLock()
	items := append([]contracts.MemoryItem(nil), s.banks[bank]...)
	s.mu.RUnlock()

	out := make([]contracts.MemoryItem, 0, len(items))
	for _, item := range items {
		if query.Symbol != "" && !strings.EqualFold(item.Symbol, query.Symbol) {
			continue
		}
		if len(query.Types) > 0 && !stringInSlice(item.Type, query.Types) {
			continue
		}
		if len(query.Tags) > 0 && !containsAllTags(item.Tags, query.Tags) {
			continue
		}
		if query.From != nil && item.TS.Before(query.From.UTC()) {
			continue
		}
		if query.To != nil && item.TS.After(query.To.UTC()) {
			continue
		}
		if query.Q != "" && !strings.Contains(strings.ToLower(item.Summary), strings.ToLower(query.Q)) {
			continue
		}
		out = append(out, item)
		if query.Limit > 0 && len(out) >= query.Limit {
			break
		}
	}

	return out, nil
}

func (s *InMemoryMemoryStore) Reflect(ctx context.Context, bank string, params contracts.ReflectionParams) ([]contracts.MemoryItem, error) {
	if strings.TrimSpace(params.Query) == "" {
		return nil, errors.New("reflect: params.query is required")
	}

	recalled, err := s.Recall(ctx, bank, contracts.MemoryQuery{Q: params.Query, Limit: 10})
	if err != nil {
		return nil, err
	}

	summary := "No prior memories matched."
	if len(recalled) > 0 {
		summary = "Found prior related memories."
	}

	return []contracts.MemoryItem{
		{
			TS:      time.Now().UTC(),
			Type:    "belief",
			Summary: summary,
			Data: map[string]any{
				"query": params.Query,
				"count": len(recalled),
			},
			Source: &contracts.MemorySource{System: "inmemory"},
		},
	}, nil
}

func stringInSlice(v string, list []string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func containsAllTags(have []string, want []string) bool {
	haveSet := make(map[string]struct{}, len(have))
	for _, t := range have {
		haveSet[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
	}
	for _, t := range want {
		if _, ok := haveSet[strings.ToLower(strings.TrimSpace(t))]; !ok {
			return false
		}
	}
	return true
}
