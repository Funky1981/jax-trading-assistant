package macroevents

import "context"

type pricedInStore interface {
	SavePricedInScore(ctx context.Context, score PricedInScore) (PricedInScore, error)
	SaveConfounders(ctx context.Context, confounders []Confounder) ([]Confounder, error)
}

type PricedInService struct {
	store pricedInStore
}

func NewPricedInService(store pricedInStore) *PricedInService {
	return &PricedInService{store: store}
}

func (s *PricedInService) ScoreAndSave(ctx context.Context, input PricedInInput) (PricedInScore, error) {
	score := ScorePricedIn(input)
	if s.store == nil {
		return score, nil
	}
	return s.store.SavePricedInScore(ctx, score)
}

func (s *PricedInService) DetectAndSaveConfounders(ctx context.Context, macroEventID string, inputs []ConfounderInput) ([]Confounder, error) {
	confounders := DetectConfounders(inputs)
	for i := range confounders {
		confounders[i].MacroEventID = macroEventID
	}
	if s.store == nil {
		return confounders, nil
	}
	return s.store.SaveConfounders(ctx, confounders)
}
