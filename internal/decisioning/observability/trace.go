package observability

import "time"

type Trace struct {
	TraceID        string    `json:"trace_id"`
	PipelineID     string    `json:"pipeline_id"`
	EventID        string    `json:"event_id"`
	DecisionID     string    `json:"decision_id"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
	ModulesVisited []string  `json:"modules_visited"`
	FinalStatus    string    `json:"final_status"`
	FinalDecision  string    `json:"final_decision"`
	Warnings       []string  `json:"warnings"`
	Errors         []string  `json:"errors"`
}

type TraceInput struct {
	TraceID        string
	PipelineID     string
	EventID        string
	DecisionID     string
	StartedAt      time.Time
	CompletedAt    time.Time
	ModulesVisited []string
	FinalStatus    string
	FinalDecision  string
	Warnings       []string
	Errors         []string
}

func NewTrace(input TraceInput) Trace {
	return Trace{
		TraceID:        input.TraceID,
		PipelineID:     input.PipelineID,
		EventID:        input.EventID,
		DecisionID:     input.DecisionID,
		StartedAt:      input.StartedAt,
		CompletedAt:    input.CompletedAt,
		ModulesVisited: append([]string(nil), input.ModulesVisited...),
		FinalStatus:    input.FinalStatus,
		FinalDecision:  input.FinalDecision,
		Warnings:       append([]string(nil), input.Warnings...),
		Errors:         append([]string(nil), input.Errors...),
	}
}
