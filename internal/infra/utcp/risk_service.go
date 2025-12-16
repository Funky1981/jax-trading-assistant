package utcp

import "context"

type RiskService struct {
	client Client
}

func NewRiskService(c Client) *RiskService {
	return &RiskService{client: c}
}

func (s *RiskService) PositionSize(ctx context.Context, in PositionSizeInput) (PositionSizeOutput, error) {
	var out PositionSizeOutput
	if err := s.client.CallTool(ctx, RiskProviderID, ToolRiskPositionSize, in, &out); err != nil {
		return PositionSizeOutput{}, err
	}
	return out, nil
}

func (s *RiskService) RMultiple(ctx context.Context, in RMultipleInput) (RMultipleOutput, error) {
	var out RMultipleOutput
	if err := s.client.CallTool(ctx, RiskProviderID, ToolRiskRMultiple, in, &out); err != nil {
		return RMultipleOutput{}, err
	}
	return out, nil
}
