package aishadow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) SafetyCounts() (SafetyCounts, error) {
	if s.pool == nil {
		return SafetyCounts{}, fmt.Errorf("AI shadow store requires a database")
	}
	var counts SafetyCounts
	err := s.pool.QueryRow(context.Background(), `SELECT
		(SELECT COUNT(*) FROM trade_approvals),
		(SELECT COUNT(*) FROM candidate_approvals),
		(SELECT COUNT(*) FROM candidate_paper_tickets),
		(SELECT COUNT(*) FROM execution_instructions),
		(SELECT COUNT(*) FROM order_intents),
		(SELECT COUNT(*) FROM execution_instructions WHERE NULLIF(BTRIM(broker_order_id),'') IS NOT NULL),
		(SELECT COUNT(*) FROM trades),
		(SELECT COUNT(*) FROM fills)`).Scan(
		&counts.Approvals, &counts.CandidateApprovals, &counts.PaperTickets,
		&counts.ExecutionInstructions, &counts.OrderIntents, &counts.BrokerOrders,
		&counts.Trades, &counts.Fills,
	)
	if err != nil {
		return SafetyCounts{}, fmt.Errorf("read AI shadow safety counts: %w", err)
	}
	return counts, nil
}

func (s *PGStore) StartRun(run RunRecord) error {
	safety, _ := json.Marshal(run.SafetyBefore)
	_, err := s.pool.Exec(context.Background(), `INSERT INTO ai_shadow_benchmark_runs (
		id,manifest_version,manifest_fingerprint,provider,model,prompt_version,schema_version,
		seed,temperature,event_limit,started_at,status,safety_before
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'running',$12::jsonb)`,
		run.ID, run.ManifestVersion, run.ManifestFingerprint, run.Provider, run.Model,
		run.PromptVersion, run.SchemaVersion, run.Seed, run.Temperature, run.EventLimit,
		run.StartedAt, string(safety))
	if err != nil {
		return fmt.Errorf("start AI shadow run: %w", err)
	}
	return nil
}

func (s *PGStore) SaveAttempt(attempt Attempt) error {
	_, err := s.pool.Exec(context.Background(), `INSERT INTO ai_shadow_benchmark_attempts (
		run_id,event_id,attempt_number,input_fingerprint,provider,model,model_reported_identifier,
		prompt_version,schema_version,seed,temperature,request_timestamp,response_timestamp,duration_ms,
		raw_response_hash,validation_status,validation_errors,failure_reason
	) VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NULLIF($18,''))`,
		attempt.RunID, attempt.EventID, attempt.AttemptNumber, attempt.InputFingerprint,
		attempt.Provider, attempt.Model, attempt.ModelReportedIdentifier, attempt.PromptVersion,
		attempt.SchemaVersion, attempt.Seed, attempt.Temperature, attempt.RequestTimestamp,
		attempt.ResponseTimestamp, attempt.Duration.Milliseconds(), attempt.RawResponseHash,
		attempt.ValidationStatus, nonNilStrings(attempt.ValidationErrors), attempt.FailureReason)
	if err != nil {
		return fmt.Errorf("persist AI shadow attempt: %w", err)
	}
	return nil
}

func (s *PGStore) SaveResult(result EventResult) error {
	var parsed any
	if result.Parsed != nil {
		raw, err := json.Marshal(result.Parsed)
		if err != nil {
			return fmt.Errorf("marshal AI shadow result: %w", err)
		}
		parsed = string(raw)
	}
	_, err := s.pool.Exec(context.Background(), `INSERT INTO ai_shadow_benchmark_results (
		run_id,manifest_version,event_id,input_fingerprint,provider,model,model_reported_identifier,
		prompt_version,schema_version,seed,temperature,request_timestamp,response_timestamp,duration_ms,
		retry_count,raw_response_hash,parsed_result,validation_status,validation_errors,failure_reason
	) VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9,$10,$11,$12,$13,$14,$15,$16,$17::jsonb,$18,$19,NULLIF($20,''))`,
		result.RunID, result.ManifestVersion, result.EventID, result.InputFingerprint,
		result.Provider, result.Model, result.ModelReportedIdentifier, result.PromptVersion,
		result.SchemaVersion, result.Seed, result.Temperature, result.RequestTimestamp,
		result.ResponseTimestamp, result.Duration.Milliseconds(), result.RetryCount,
		result.RawResponseHash, parsed, result.ValidationStatus, nonNilStrings(result.ValidationErrors), result.FailureReason)
	if err != nil {
		return fmt.Errorf("persist AI shadow result: %w", err)
	}
	return nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func (s *PGStore) FinishRun(finish FinishRecord) error {
	safety, _ := json.Marshal(finish.SafetyAfter)
	paths, _ := json.Marshal(finish.ReportPaths)
	_, err := s.pool.Exec(context.Background(), `UPDATE ai_shadow_benchmark_runs
		SET completed_at=$2,status=$3,failure_reason=NULLIF($4,''),safety_after=$5::jsonb,report_paths=$6::jsonb
		WHERE id=$1 AND status='running'`, finish.RunID, finish.CompletedAt, finish.Status,
		finish.FailureReason, string(safety), string(paths))
	if err != nil {
		return fmt.Errorf("finish AI shadow run: %w", err)
	}
	return nil
}
