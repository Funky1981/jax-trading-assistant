package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

type AIDecisionRecord struct {
	RunID       string
	FlowID      string
	Role        string
	Provider    string
	Model       string
	Prompt      map[string]any
	Response    map[string]any
	SchemaValid bool
	Decision    string
	Reasoning   string
	RuleTrace   map[string]any
}

func (s *Service) LogAuditEvent(ctx context.Context, correlationID, category, action, outcome, message string, metadata map[string]any) error {
	if s == nil || s.db == nil {
		return nil
	}
	meta, _ := json.Marshal(metadata)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_events (id, correlation_id, category, action, outcome, message, metadata, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, NOW())
	`, uuid.NewString(), correlationID, category, action, outcome, message, string(meta))
	return err
}

func (s *Service) LogAIDecision(ctx context.Context, rec AIDecisionRecord) (string, error) {
	if s == nil || s.db == nil {
		return "", nil
	}
	id := uuid.NewString()
	prompt, _ := json.Marshal(rec.Prompt)
	response, _ := json.Marshal(rec.Response)
	trace, _ := json.Marshal(rec.RuleTrace)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_decisions (
			id, run_id, flow_id, role, provider, model, prompt, response,
			schema_valid, decision, reasoning, rule_trace, created_at
		) VALUES (
			$1::uuid, NULLIF($2,'')::uuid, $3, $4, $5, $6, $7::jsonb, $8::jsonb,
			$9, $10, $11, $12::jsonb, NOW()
		)
	`, id, rec.RunID, rec.FlowID, rec.Role, rec.Provider, rec.Model, string(prompt), string(response),
		rec.SchemaValid, rec.Decision, rec.Reasoning, string(trace))
	return id, err
}

func (s *Service) LogAIAcceptance(ctx context.Context, decisionID string, accepted bool, acceptedBy, reason string, ruleTrace map[string]any) error {
	if s == nil || s.db == nil || decisionID == "" {
		return nil
	}
	trace, _ := json.Marshal(ruleTrace)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_decision_acceptance (id, decision_id, accepted, accepted_by, reason, rule_trace, created_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::jsonb, NOW())
	`, uuid.NewString(), decisionID, accepted, acceptedBy, reason, string(trace))
	return err
}

func ValidatePlanShape(summary, action string, confidence float64, steps []string) (bool, map[string]any) {
	trace := map[string]any{
		"summary_present":  summary != "",
		"action_present":   action != "",
		"confidence_range": confidence >= 0 && confidence <= 1,
		"steps_non_empty":  len(steps) > 0,
		"validated_at_utc": time.Now().UTC().Format(time.RFC3339),
	}
	valid := trace["summary_present"].(bool) &&
		trace["action_present"].(bool) &&
		trace["confidence_range"].(bool) &&
		trace["steps_non_empty"].(bool)
	return valid, trace
}

func (s *Service) MustAvailable() error {
	if s == nil || s.db == nil {
		return fmt.Errorf("audit service unavailable")
	}
	return nil
}
