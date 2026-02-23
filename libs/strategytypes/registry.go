package strategytypes

import (
	"fmt"
	"slices"
)

type Registry struct {
	types map[string]StrategyType
	order []string
}

func NewRegistry() *Registry {
	return &Registry{
		types: make(map[string]StrategyType),
		order: make([]string, 0, 8),
	}
}

func DefaultRegistry() *Registry {
	r := NewRegistry()
	_ = r.Register(NewSameDayEarningsDrift())
	_ = r.Register(NewSameDayNewsRepricing())
	_ = r.Register(NewOpeningRangeToClose())
	_ = r.Register(NewPanicReversion())
	_ = r.Register(NewIndexFlow())
	return r
}

func (r *Registry) Register(t StrategyType) error {
	if t == nil {
		return fmt.Errorf("strategy type is nil")
	}
	meta := t.Metadata()
	if meta.StrategyID == "" {
		return fmt.Errorf("strategy type id is required")
	}
	if _, ok := r.types[meta.StrategyID]; ok {
		return fmt.Errorf("strategy type already registered: %s", meta.StrategyID)
	}
	r.types[meta.StrategyID] = t
	r.order = append(r.order, meta.StrategyID)
	slices.Sort(r.order)
	return nil
}

func (r *Registry) Get(id string) (StrategyType, bool) {
	t, ok := r.types[id]
	return t, ok
}

func (r *Registry) ListMetadata() []StrategyMetadata {
	out := make([]StrategyMetadata, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.types[id].Metadata())
	}
	return out
}

func (r *Registry) MustGet(id string) StrategyType {
	t, ok := r.Get(id)
	if !ok {
		panic(fmt.Sprintf("unknown strategy type: %s", id))
	}
	return t
}
