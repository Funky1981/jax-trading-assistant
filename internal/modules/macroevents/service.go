package macroevents

import (
	"context"
	"time"
)

type eventStore interface {
	FindBySourceEventID(ctx context.Context, source, sourceEventID string) (StoredEvent, bool, error)
	Save(ctx context.Context, event StoredEvent) (StoredEvent, error)
}

type Service struct {
	store eventStore
	now   func() time.Time
}

func NewService(store eventStore) *Service {
	return &Service{
		store: store,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) Ingest(ctx context.Context, input EventInput) (Receipt, error) {
	input = NormalizeInput(input)
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}
	if existing, found, err := s.store.FindBySourceEventID(ctx, input.Source, input.SourceEventID); err != nil {
		return Receipt{}, err
	} else if found {
		return receiptFromStoredEvent(existing, true), nil
	}

	validation := Validate(input, s.now())
	event := StoredEvent{
		Input:           input,
		Status:          validation.Status,
		RejectionReason: validation.Reason,
	}

	if validation.Valid {
		mappings, err := ValidateAndNormalizeETFs(input.AffectedETFs)
		if err != nil {
			event.Status = StatusRejected
			event.RejectionReason = err.Error()
		} else {
			event.Mappings = mappings
		}
	}

	stored, err := s.store.Save(ctx, event)
	if err != nil {
		return Receipt{}, err
	}
	return receiptFromStoredEvent(stored, false), nil
}

func receiptFromStoredEvent(event StoredEvent, duplicate bool) Receipt {
	return Receipt{
		MacroEventID:    event.ID,
		Status:          event.Status,
		RejectionReason: event.RejectionReason,
		Duplicate:       duplicate,
		MappedETFs:      mappedSymbols(event.Mappings),
	}
}

func mappedSymbols(mappings []ETFMapping) []string {
	out := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		out = append(out, mapping.Symbol)
	}
	return out
}
