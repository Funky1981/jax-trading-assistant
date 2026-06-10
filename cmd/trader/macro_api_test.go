package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func insertMacroAPITestEvent(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	eventID := uuid.NewString()
	sourceEventID := "macro-api-test-" + uuid.NewString()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO macro_events (
			id, source, source_event_id, event_type, region, event_time_utc,
			headline, summary, actual_value, expected_value, previous_value, unit,
			surprise_value, surprise_percent, direction, confidence, raw_payload, status
		) VALUES (
			$1::uuid, 'test', $2, 'cpi', 'US', $3,
			'CPI cooler than expected', 'Inflation cooled versus forecast', 3.1, 3.3, 3.4, 'pct',
			-0.2, -6.06, 'inflation_cool', 0.88, '{"fixture":true}'::jsonb, 'accepted'
		)
	`, eventID, sourceEventID, time.Now().UTC())
	if err != nil {
		t.Fatalf("insert macro event: %v", err)
	}
	_, err = pool.Exec(t.Context(), `
		INSERT INTO macro_event_etf_map (macro_event_id, symbol, theme, mapping_reason, confidence)
		VALUES ($1::uuid, 'TLT', 'rates', 'cool CPI can support long duration', 0.8)
	`, eventID)
	if err != nil {
		t.Fatalf("insert macro etf map: %v", err)
	}
	_, err = pool.Exec(t.Context(), `
		INSERT INTO macro_reaction_snapshots (
			macro_event_id, symbol, timeframe, pre_price, post_price, change_abs, change_percent,
			high_after, low_after, volume_ratio, atr_ratio, direction, confirms_event, too_extended,
			noisy, reason, raw_candles
		) VALUES (
			$1::uuid, 'TLT', '15m', 91.10, 92.20, 1.10, 1.21,
			92.40, 90.95, 1.8, 1.2, 'up', true, false, false,
			'Tlt confirmed the rates reaction', '[]'::jsonb
		)
	`, eventID)
	if err != nil {
		t.Fatalf("insert macro reaction: %v", err)
	}
	_, err = pool.Exec(t.Context(), `
		INSERT INTO macro_scenario_results (
			macro_event_id, scenario_key, candidate_bias, primary_symbols, secondary_symbols,
			required_confirmations, expected_reactions, result, reason
		) VALUES (
			$1::uuid, 'cool_cpi_duration_bid', 'long_duration', ARRAY['TLT'], ARRAY['SPY'],
			ARRAY['TLT up'], '{"TLT":"up"}'::jsonb, 'eligible_for_reaction_check',
			'Scenario has a direct ETF mapping'
		)
	`, eventID)
	if err != nil {
		t.Fatalf("insert macro scenario: %v", err)
	}
	_, err = pool.Exec(t.Context(), `
		INSERT INTO macro_priced_in_scores (macro_event_id, symbol, verdict, score, reasons)
		VALUES ($1::uuid, 'TLT', 'not_priced_in', 0.72, ARRAY['surprise larger than forecast'])
	`, eventID)
	if err != nil {
		t.Fatalf("insert priced in score: %v", err)
	}
	_, err = pool.Exec(t.Context(), `
		INSERT INTO macro_confounders (macro_event_id, confounder_type, headline, source, severity, reason)
		VALUES ($1::uuid, 'fed_speaker', 'Fed speaker due soon', 'calendar', 'medium', 'Could reverse rate reaction')
	`, eventID)
	if err != nil {
		t.Fatalf("insert confounder: %v", err)
	}
	var evidenceID string
	err = pool.QueryRow(t.Context(), `
		INSERT INTO macro_evidence_bundles (
			macro_event_id, symbol, status, verdict, summary, evidence, missing_evidence, walkaway_reasons
		) VALUES (
			$1::uuid, 'TLT', 'complete', 'candidate_allowed', 'Reaction and evidence support a paper candidate',
			'{"reaction":"confirmed"}'::jsonb, ARRAY[]::text[], ARRAY['Fed speaker invalidates setup']
		)
		RETURNING id::text
	`, eventID).Scan(&evidenceID)
	if err != nil {
		t.Fatalf("insert evidence bundle: %v", err)
	}
	_, err = pool.Exec(t.Context(), `
		INSERT INTO macro_candidate_trades (
			macro_event_id, evidence_bundle_id, symbol, side, bias, entry_type, entry_reference_price,
			stop_reference_price, target_reference_price, risk_percent, time_limit, status, created_reason,
			walkaway_reasons
		) VALUES (
			$1::uuid, $2::uuid, 'TLT', 'long', 'duration_bid', 'pullback_retest', 92.20,
			91.50, 94.00, 0.01, 'same_session', 'awaiting_human_approval',
			'Paper candidate only after macro evidence bundle', ARRAY['Fed speaker invalidates setup']
		)
	`, eventID, evidenceID)
	if err != nil {
		t.Fatalf("insert macro candidate: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(t.Context(), `DELETE FROM macro_events WHERE id=$1::uuid`, eventID)
	})
	return eventID
}

func TestMacroEventsHandlerListsEvents(t *testing.T) {
	pool := testFrontendAPIPool(t)
	eventID := insertMacroAPITestEvent(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/macro/events?limit=10", nil)
	rec := httptest.NewRecorder()
	macroEventsHandler(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Events []macroEventDTO `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode macro list: %v", err)
	}
	for _, event := range payload.Events {
		if event.ID == eventID {
			if event.CandidateCount != 1 || event.EvidenceCount != 1 {
				t.Fatalf("counts = candidates %d evidence %d, want 1/1", event.CandidateCount, event.EvidenceCount)
			}
			if len(event.ETFMappings) != 1 || event.ETFMappings[0].Symbol != "TLT" {
				t.Fatalf("etf mappings = %#v, want TLT", event.ETFMappings)
			}
			return
		}
	}
	t.Fatalf("macro event %s not found in list", eventID)
}

func TestMacroEventDetailHandlerLoadsAnalysis(t *testing.T) {
	pool := testFrontendAPIPool(t)
	eventID := insertMacroAPITestEvent(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/macro/events/"+eventID, nil)
	rec := httptest.NewRecorder()
	macroEventDetailHandler(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("detail status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var payload macroEventDetailDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode macro detail: %v", err)
	}
	if payload.Event.ID != eventID {
		t.Fatalf("event id = %q, want %q", payload.Event.ID, eventID)
	}
	if len(payload.Reactions) != 1 || !payload.Reactions[0].ConfirmsEvent {
		t.Fatalf("reactions = %#v, want confirmed reaction", payload.Reactions)
	}
	if len(payload.EvidenceBundles) != 1 || payload.EvidenceBundles[0].Verdict != "candidate_allowed" {
		t.Fatalf("evidence = %#v, want candidate_allowed", payload.EvidenceBundles)
	}
	if len(payload.Candidates) != 1 || !payload.Candidates[0].HumanApprovalRequired {
		t.Fatalf("candidates = %#v, want human approval required", payload.Candidates)
	}
}

func TestMacroEventDetailRejectsInvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/macro/events/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	macroEventDetailHandler(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want database unavailable before id parsing with nil pool", rec.Code)
	}

	pool := testFrontendAPIPool(t)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/macro/events/not-a-uuid", nil)
	rec = httptest.NewRecorder()
	macroEventDetailHandler(pool).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %q, want 400", rec.Code, rec.Body.String())
	}
}

func TestMacroCandidateApproveIsPaperOnlyStatusUpdate(t *testing.T) {
	pool := testFrontendAPIPool(t)
	eventID := insertMacroAPITestEvent(t, pool)

	var candidateID string
	if err := pool.QueryRow(t.Context(), `
		SELECT id::text FROM macro_candidate_trades WHERE macro_event_id=$1::uuid
	`, eventID).Scan(&candidateID); err != nil {
		t.Fatalf("load candidate id: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/macro/candidates/"+candidateID+"/approve", strings.NewReader(`{"notes":"reviewed"}`))
	rec := httptest.NewRecorder()
	macroCandidateDetailHandler(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("approve status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var payload macroCandidateDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode candidate: %v", err)
	}
	if payload.Status != "watch_only" {
		t.Fatalf("status = %q, want watch_only", payload.Status)
	}
}
