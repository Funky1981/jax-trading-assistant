package macroevents

import "context"

type ScenarioService struct {
	store scenarioStore
}

func NewScenarioService(store scenarioStore) *ScenarioService {
	return &ScenarioService{store: store}
}

func (s *ScenarioService) EvaluateAndSave(ctx context.Context, input EventInput) (ScenarioEvaluation, error) {
	result := EvaluateScenario(input)
	if s.store == nil {
		return result, nil
	}
	return s.store.SaveScenarioResult(ctx, result)
}
