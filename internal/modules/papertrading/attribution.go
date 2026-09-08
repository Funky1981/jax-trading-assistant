package papertrading

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
)

type AttributionInput struct {
	RecommendationID    string    `json:"recommendation_id"`
	RiskDecisionID      string    `json:"risk_decision_id"`
	PaperIntentID       string    `json:"paper_intent_id"`
	OrderID             string    `json:"order_id"`
	FillID              string    `json:"fill_id"`
	Direction           string    `json:"direction"`
	RecommendationEntry float64   `json:"recommendation_entry"`
	RiskEntry           float64   `json:"risk_entry"`
	ActivationReference float64   `json:"activation_reference"`
	RecommendationQty   float64   `json:"recommendation_quantity"`
	RiskQty             float64   `json:"risk_quantity"`
	ExitPrice           float64   `json:"exit_price"`
	Fill                PaperFill `json:"fill"`
}

type AttributionResult struct {
	Identity             string  `json:"identity"`
	ContractVersion      string  `json:"contract_version"`
	RecommendationID     string  `json:"recommendation_id"`
	RiskDecisionID       string  `json:"risk_decision_id"`
	PaperIntentID        string  `json:"paper_intent_id"`
	OrderID              string  `json:"order_id"`
	FillID               string  `json:"fill_id"`
	GrossThesisPnL       float64 `json:"gross_thesis_pnl"`
	RiskSizingImpact     float64 `json:"risk_sizing_impact"`
	LatencyImpact        float64 `json:"latency_impact"`
	ExecutionPriceImpact float64 `json:"execution_price_impact"`
	SpreadImpact         float64 `json:"spread_impact"`
	SlippageImpact       float64 `json:"slippage_impact"`
	FeeImpact            float64 `json:"fee_impact"`
	NetPaperPnL          float64 `json:"net_paper_pnl"`
}

func Attribute(input AttributionInput) (AttributionResult, error) {
	if input.RecommendationID == "" || input.RiskDecisionID == "" || input.PaperIntentID == "" || input.OrderID == "" || input.FillID == "" || input.Direction != input.Fill.Direction || (input.Direction != "LONG" && input.Direction != "SHORT") || !finitePositive(input.RecommendationEntry) || !finitePositive(input.RiskEntry) || !finitePositive(input.ActivationReference) || !finitePositive(input.RecommendationQty) || !finitePositive(input.RiskQty) || !finitePositive(input.ExitPrice) {
		return AttributionResult{}, ErrInvalidArtifact
	}
	if err := input.Fill.Validate(); err != nil || input.Fill.FillID != input.FillID || input.Fill.OrderID != input.OrderID || input.Fill.PaperIntentID != input.PaperIntentID {
		return AttributionResult{}, ErrInvalidArtifact
	}
	multiplier := directionMultiplier(input.Direction)
	result := AttributionResult{ContractVersion: AttributionVersion, RecommendationID: input.RecommendationID, RiskDecisionID: input.RiskDecisionID, PaperIntentID: input.PaperIntentID, OrderID: input.OrderID, FillID: input.FillID}
	result.GrossThesisPnL = (input.ExitPrice - input.RecommendationEntry) * input.RecommendationQty * multiplier
	result.RiskSizingImpact = (input.ExitPrice - input.RecommendationEntry) * (input.RiskQty - input.RecommendationQty) * multiplier
	result.LatencyImpact = (input.ActivationReference - input.RiskEntry) * input.RiskQty * multiplier
	result.ExecutionPriceImpact = (input.Fill.Price - input.ActivationReference) * input.Fill.Quantity * multiplier
	result.SpreadImpact = -input.Fill.Costs.SpreadCost
	result.SlippageImpact = -input.Fill.Costs.SlippageCost
	result.FeeImpact = -input.Fill.Costs.Commission
	result.NetPaperPnL = (input.ExitPrice-input.Fill.Price)*input.Fill.Quantity*multiplier - input.Fill.Costs.Commission
	for _, value := range []float64{result.GrossThesisPnL, result.RiskSizingImpact, result.LatencyImpact, result.ExecutionPriceImpact, result.SpreadImpact, result.SlippageImpact, result.FeeImpact, result.NetPaperPnL} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return AttributionResult{}, fmt.Errorf("%w: attribution is not finite", ErrInvalidArtifact)
		}
	}
	result.Identity = attributionIdentity(result)
	return result, nil
}

func attributionIdentity(result AttributionResult) string {
	copyResult := result
	copyResult.Identity = ""
	data, _ := json.Marshal(copyResult)
	digest := sha256.Sum256(data)
	return "pat_" + hex.EncodeToString(digest[:])
}
