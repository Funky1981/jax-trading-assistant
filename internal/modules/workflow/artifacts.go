package workflow

import "time"

// PaperIntent is descriptive and immutable. It is deliberately not an order,
// execution instruction, broker request, trade, or fill.
type PaperIntent struct {
	IntentID               string    `json:"intent_id"`
	ContractVersion        string    `json:"contract_version"`
	WorkflowID             string    `json:"workflow_id"`
	RecommendationID       string    `json:"recommendation_id"`
	RiskDecisionID         string    `json:"risk_decision_id"`
	InstrumentID           string    `json:"instrument_id"`
	Direction              string    `json:"direction"`
	DescriptiveValue       float64   `json:"descriptive_value"`
	ExecutionStatus        string    `json:"execution_status"`
	PaperOnly              bool      `json:"paper_only"`
	BrokerExecutionAllowed bool      `json:"broker_execution_allowed"`
	PortfolioMutation      bool      `json:"portfolio_mutation"`
	CreatedAt              time.Time `json:"created_at"`
}

// Breaker is defined here so the state store can own safety state atomically;
// its command semantics are implemented in breakers.go.
type Breaker struct {
	Name      string    `json:"name"`
	Tripped   bool      `json:"tripped"`
	Reason    string    `json:"reason,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}
