package macroevents

import "context"

type fundamentalStore interface {
	SaveFundamentalAnalysisSnapshot(ctx context.Context, snapshot FundamentalSnapshot) (FundamentalSnapshot, error)
}

type FundamentalService struct {
	store fundamentalStore
}

func NewFundamentalService(store fundamentalStore) *FundamentalService {
	return &FundamentalService{store: store}
}

func (s *FundamentalService) EvaluateAndSave(ctx context.Context, input FundamentalInput) (FundamentalSnapshot, error) {
	snapshot := EvaluateFundamentalSnapshot(input)
	if s.store == nil {
		return snapshot, nil
	}
	return s.store.SaveFundamentalAnalysisSnapshot(ctx, snapshot)
}
