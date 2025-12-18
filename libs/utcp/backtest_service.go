package utcp

import "context"

type BacktestService struct {
	client Client
}

func NewBacktestService(c Client) *BacktestService {
	return &BacktestService{client: c}
}

func (s *BacktestService) RunStrategy(ctx context.Context, in RunStrategyInput) (RunStrategyOutput, error) {
	var out RunStrategyOutput
	if err := s.client.CallTool(ctx, BacktestProviderID, ToolBacktestRunStrategy, in, &out); err != nil {
		return RunStrategyOutput{}, err
	}
	return out, nil
}

func (s *BacktestService) GetRun(ctx context.Context, runID string) (GetRunOutput, error) {
	var out GetRunOutput
	if err := s.client.CallTool(ctx, BacktestProviderID, ToolBacktestGetRun, GetRunInput{RunID: runID}, &out); err != nil {
		return GetRunOutput{}, err
	}
	return out, nil
}
