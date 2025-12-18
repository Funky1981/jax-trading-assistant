package utcp

import (
	"context"
	"fmt"
	"math"
)

const (
	RiskProviderID       = "risk"
	ToolRiskPositionSize = "risk.position_size"
	ToolRiskRMultiple    = "risk.r_multiple"
)

func RegisterRiskTools(registry *LocalRegistry) error {
	if registry == nil {
		return fmt.Errorf("register risk tools: registry is nil")
	}

	if err := registry.Register(RiskProviderID, ToolRiskPositionSize, riskPositionSizeTool); err != nil {
		return err
	}
	if err := registry.Register(RiskProviderID, ToolRiskRMultiple, riskRMultipleTool); err != nil {
		return err
	}

	return nil
}

func riskPositionSizeTool(_ context.Context, input any, output any) error {
	var in PositionSizeInput
	if err := decodeJSONLike(input, &in); err != nil {
		return fmt.Errorf("risk.position_size: %w", err)
	}
	out, err := computePositionSize(in)
	if err != nil {
		return err
	}
	if output == nil {
		return nil
	}
	typed, ok := output.(*PositionSizeOutput)
	if !ok {
		return fmt.Errorf("risk.position_size: output must be *utcp.PositionSizeOutput")
	}
	*typed = out
	return nil
}

func riskRMultipleTool(_ context.Context, input any, output any) error {
	var in RMultipleInput
	if err := decodeJSONLike(input, &in); err != nil {
		return fmt.Errorf("risk.r_multiple: %w", err)
	}
	out, err := computeRMultiple(in)
	if err != nil {
		return err
	}
	if output == nil {
		return nil
	}
	typed, ok := output.(*RMultipleOutput)
	if !ok {
		return fmt.Errorf("risk.r_multiple: output must be *utcp.RMultipleOutput")
	}
	*typed = out
	return nil
}

func computePositionSize(in PositionSizeInput) (PositionSizeOutput, error) {
	if in.AccountSize <= 0 {
		return PositionSizeOutput{}, fmt.Errorf("risk.position_size: accountSize must be > 0")
	}
	if in.RiskPercent <= 0 {
		return PositionSizeOutput{}, fmt.Errorf("risk.position_size: riskPercent must be > 0")
	}

	riskPerUnit := math.Abs(in.Entry - in.Stop)
	if riskPerUnit <= 0 {
		return PositionSizeOutput{}, fmt.Errorf("risk.position_size: entry and stop must differ")
	}

	maxRisk := in.AccountSize * (in.RiskPercent / 100)
	if maxRisk <= 0 {
		return PositionSizeOutput{}, fmt.Errorf("risk.position_size: computed max risk must be > 0")
	}

	positionSize := int(math.Floor(maxRisk / riskPerUnit))
	if positionSize < 0 {
		positionSize = 0
	}

	totalRisk := float64(positionSize) * riskPerUnit
	return PositionSizeOutput{
		PositionSize: positionSize,
		RiskPerUnit:  riskPerUnit,
		TotalRisk:    totalRisk,
	}, nil
}

func computeRMultiple(in RMultipleInput) (RMultipleOutput, error) {
	riskPerUnit := math.Abs(in.Entry - in.Stop)
	if riskPerUnit <= 0 {
		return RMultipleOutput{}, fmt.Errorf("risk.r_multiple: entry and stop must differ")
	}
	reward := math.Abs(in.Target - in.Entry)
	return RMultipleOutput{RMultiple: reward / riskPerUnit}, nil
}
