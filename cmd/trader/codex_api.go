package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/strategytypes"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type strategyInstanceDTO struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	StrategyTypeID     string          `json:"strategyTypeId"`
	StrategyID         string          `json:"strategyId,omitempty"`
	Enabled            bool            `json:"enabled"`
	SessionTimezone    string          `json:"sessionTimezone"`
	FlattenByCloseTime string          `json:"flattenByCloseTime"`
	ConfigJSON         json.RawMessage `json:"configJson"`
	ConfigHash         string          `json:"configHash"`
	ArtifactID         *string         `json:"artifactId,omitempty"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
}

func registerCodexAPIRoutes(mux *http.ServeMux, protect func(http.HandlerFunc) http.HandlerFunc, pool *pgxpool.Pool, orchestratorURL string, strategyTypeReg *strategytypes.Registry) {
	mux.HandleFunc("/api/v1/events", protect(eventsHandler(pool)))
	mux.HandleFunc("/api/v1/events/classify", protect(eventsClassifyHandler(pool)))
	mux.HandleFunc("/api/v1/events/", protect(eventsDetailHandler(pool)))
	mux.HandleFunc("/api/v1/datasets", protect(datasetsListHandler(pool)))
	mux.HandleFunc("/api/v1/datasets/", protect(datasetsDetailHandler(pool)))

	mux.HandleFunc("/api/v1/instances", protect(instancesHandler(pool, strategyTypeReg)))
	mux.HandleFunc("/api/v1/instances/", protect(instancesDetailHandler(pool, strategyTypeReg)))

	mux.HandleFunc("/api/v1/backtests/run", protect(backtestRunHandler(pool, orchestratorURL)))
	mux.HandleFunc("/api/v1/backtests/runs", protect(backtestRunsHandler(pool)))
	mux.HandleFunc("/api/v1/backtests/runs/", protect(backtestRunDetailHandler(pool)))

	mux.HandleFunc("/api/v1/research/projects", protect(researchProjectsHandler(pool, orchestratorURL)))
	mux.HandleFunc("/api/v1/research/projects/", protect(researchProjectsDetailHandler(pool, orchestratorURL)))

	mux.HandleFunc("/api/v1/testing/status", protect(testingStatusHandler(pool)))
	mux.HandleFunc("/api/v1/testing/recon/data", protect(testingTriggerHandler(pool, "Gate1", "data_recon")))
	mux.HandleFunc("/api/v1/testing/recon/pnl", protect(testingTriggerHandler(pool, "Gate5", "pnl_recon")))
	mux.HandleFunc("/api/v1/testing/failure-suite", protect(testingTriggerHandler(pool, "Gate6", "failure_suite")))
	mux.HandleFunc("/api/v1/testing/failure-tests/run", protect(testingTriggerHandler(pool, "Gate6", "failure_suite")))
	mux.HandleFunc("/api/v1/testing/flatten-proof", protect(testingTriggerHandler(pool, "Gate7", "flatten_proof")))

	mux.HandleFunc("/api/v1/runs", protect(runsListHandler(pool)))
	mux.HandleFunc("/api/v1/runs/", protect(runsDetailHandler(pool)))

	mux.HandleFunc("/api/v1/ai-decisions", protect(aiDecisionsHandler(pool)))
	mux.HandleFunc("/api/v1/ai-decisions/", protect(aiDecisionDetailHandler(pool)))
	mux.HandleFunc("/api/v1/gates", protect(gatesHandler(pool)))
	mux.HandleFunc("/api/v1/test-runs", protect(testRunsHandler(pool)))
}

func instancesHandler(pool *pgxpool.Pool, strategyTypeReg *strategytypes.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rows, err := pool.Query(r.Context(), `
				SELECT id::text, name, strategy_type_id, COALESCE(strategy_id,''), enabled,
				       session_timezone, flatten_by_close_time, config::text, config_hash,
				       artifact_id::text, created_at, updated_at
				FROM strategy_instances
				ORDER BY updated_at DESC
			`)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			out := make([]strategyInstanceDTO, 0, 32)
			for rows.Next() {
				var (
					dto strategyInstanceDTO
					raw string
					aid sql.NullString
				)
				if err := rows.Scan(
					&dto.ID, &dto.Name, &dto.StrategyTypeID, &dto.StrategyID, &dto.Enabled,
					&dto.SessionTimezone, &dto.FlattenByCloseTime, &raw, &dto.ConfigHash,
					&aid, &dto.CreatedAt, &dto.UpdatedAt,
				); err == nil {
					dto.ConfigJSON = json.RawMessage(raw)
					if aid.Valid {
						dto.ArtifactID = &aid.String
					}
					out = append(out, dto)
				}
			}
			jsonOK(w, out)
		case http.MethodPost:
			var req strategyInstanceDTO
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			if req.Name == "" || req.StrategyTypeID == "" {
				http.Error(w, "name and strategyTypeId are required", http.StatusBadRequest)
				return
			}
			if req.ConfigJSON == nil {
				req.ConfigJSON = json.RawMessage(`{}`)
			}
			if err := validateStrategyInstance(strategyTypeReg, req.StrategyTypeID, req.ConfigJSON); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			hash := hashConfig(req.ConfigJSON)
			row := pool.QueryRow(r.Context(), `
				INSERT INTO strategy_instances (
					name, strategy_type_id, strategy_id, enabled, session_timezone,
					flatten_by_close_time, config, config_hash, artifact_id
				) VALUES (
					$1, $2, NULLIF($3,''), $4, COALESCE(NULLIF($5,''), 'America/New_York'),
					COALESCE(NULLIF($6,''), '15:55'), $7::jsonb, $8, NULLIF($9,'')::uuid
				)
				RETURNING id::text, created_at, updated_at
			`, req.Name, req.StrategyTypeID, req.StrategyID, req.Enabled,
				req.SessionTimezone, req.FlattenByCloseTime, string(req.ConfigJSON), hash, strOrEmpty(req.ArtifactID))
			if err := row.Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			req.ConfigHash = hash
			jsonOK(w, req)
		case http.MethodPut:
			var req strategyInstanceDTO
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			if req.ID == "" {
				http.Error(w, "id is required", http.StatusBadRequest)
				return
			}
			if req.ConfigJSON == nil {
				req.ConfigJSON = json.RawMessage(`{}`)
			}
			if req.StrategyTypeID != "" {
				if err := validateStrategyInstance(strategyTypeReg, req.StrategyTypeID, req.ConfigJSON); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			hash := hashConfig(req.ConfigJSON)
			_, err := pool.Exec(r.Context(), `
				UPDATE strategy_instances
				SET name = COALESCE(NULLIF($2,''), name),
				    strategy_type_id = COALESCE(NULLIF($3,''), strategy_type_id),
				    strategy_id = COALESCE(NULLIF($4,''), strategy_id),
				    enabled = $5,
				    session_timezone = COALESCE(NULLIF($6,''), session_timezone),
				    flatten_by_close_time = COALESCE(NULLIF($7,''), flatten_by_close_time),
				    config = $8::jsonb,
				    config_hash = $9,
				    artifact_id = NULLIF($10,'')::uuid
				WHERE id = $1::uuid
			`, req.ID, req.Name, req.StrategyTypeID, req.StrategyID, req.Enabled,
				req.SessionTimezone, req.FlattenByCloseTime, string(req.ConfigJSON), hash, strOrEmpty(req.ArtifactID))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			req.ConfigHash = hash
			jsonOK(w, map[string]any{"ok": true, "instance": req})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func instancesDetailHandler(pool *pgxpool.Pool, strategyTypeReg *strategytypes.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/instances/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 1 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		id := parts[0]
		action := ""
		if len(parts) > 1 {
			action = parts[1]
		}
		if r.Method == http.MethodPut && action == "" {
			var req struct {
				Enabled    *bool           `json:"enabled"`
				ConfigJSON json.RawMessage `json:"configJson"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			configRaw := ""
			hash := ""
			if req.ConfigJSON != nil {
				if len(req.ConfigJSON) == 0 {
					req.ConfigJSON = json.RawMessage(`{}`)
				}
				configRaw = string(req.ConfigJSON)
				hash = hashConfig(req.ConfigJSON)
				strategyTypeID := instanceStrategyTypeID(r.Context(), pool, id)
				if err := validateStrategyInstance(strategyTypeReg, strategyTypeID, req.ConfigJSON); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			var enabled sql.NullBool
			if req.Enabled != nil {
				enabled = sql.NullBool{Bool: *req.Enabled, Valid: true}
			}
			_, err := pool.Exec(r.Context(), `
				UPDATE strategy_instances
				SET enabled = COALESCE($2, enabled),
				    config = COALESCE(NULLIF($3,'')::jsonb, config),
				    config_hash = COALESCE(NULLIF($4,''), config_hash)
				WHERE id = $1::uuid
			`, id, enabled, configRaw, hash)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			jsonOK(w, map[string]any{"ok": true})
			return
		}
		if r.Method != http.MethodPost || (action != "enable" && action != "disable") {
			http.NotFound(w, r)
			return
		}
		enabled := action == "enable"
		_, err := pool.Exec(r.Context(), `UPDATE strategy_instances SET enabled = $2 WHERE id = $1::uuid`, id, enabled)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		jsonOK(w, map[string]any{"id": id, "enabled": enabled})
	}
}

type backtestRunRequest struct {
	InstanceID         string         `json:"instanceId"`
	StrategyID         string         `json:"strategyId"`
	StrategyConfigID   string         `json:"strategyConfigId"`
	From               string         `json:"from"`
	To                 string         `json:"to"`
	SymbolsOverride    []string       `json:"symbolsOverride"`
	DatasetID          string         `json:"datasetId"`
	Seed               int64          `json:"seed"`
	InitialCapital     float64        `json:"initialCapital"`
	RiskPerTrade       float64        `json:"riskPerTrade"`
	Parameters         map[string]any `json:"parameters,omitempty"`
	SessionTimezone    string         `json:"sessionTimezone,omitempty"`
	FlattenByCloseTime string         `json:"flattenByCloseTime,omitempty"`
}

func backtestRunHandler(pool *pgxpool.Pool, orchestratorURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req backtestRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		out, err := runBacktestAndPersist(r.Context(), pool, strings.TrimRight(orchestratorURL, "/"), req, "api")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		jsonOK(w, out)
	}
}

func backtestRunsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parseIntParam(r.URL.Query().Get("limit"), 50)
		instanceID := r.URL.Query().Get("instanceId")
		query := `
			SELECT id::text, external_run_id, COALESCE(instance_id::text,''), strategy_type_id, symbols, run_from, run_to,
			       status, stats::text, COALESCE(dataset_id,''), COALESCE(dataset_hash,''), data_source_type, COALESCE(source_provider,''),
			       is_synthetic, COALESCE(synthetic_reason,''), provenance_verified_at, started_at, completed_at, created_at
			FROM backtest_runs
			WHERE ($1 = '' OR instance_id::text = $1)
			ORDER BY started_at DESC
			LIMIT $2`
		rows, err := pool.Query(r.Context(), query, instanceID, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		out := make([]map[string]any, 0, limit)
		for rows.Next() {
			var id, runID, iid, sid, status, stats, datasetID, datasetHash, dataSourceType, sourceProvider, syntheticReason string
			var isSynthetic bool
			var symbols []string
			var from, to, started, created time.Time
			var completed, provenanceVerifiedAt *time.Time
			if err := rows.Scan(&id, &runID, &iid, &sid, &symbols, &from, &to, &status, &stats, &datasetID, &datasetHash,
				&dataSourceType, &sourceProvider, &isSynthetic, &syntheticReason, &provenanceVerifiedAt, &started, &completed, &created); err == nil {
				out = append(out, map[string]any{
					"id":          id,
					"runId":       runID,
					"instanceId":  iid,
					"strategyId":  sid,
					"symbols":     symbols,
					"from":        from,
					"to":          to,
					"status":      status,
					"stats":       json.RawMessage(stats),
					"datasetId":   datasetID,
					"datasetHash": datasetHash,
					"provenance": map[string]any{
						"dataSourceType":       dataSourceType,
						"sourceProvider":       sourceProvider,
						"isSynthetic":          isSynthetic,
						"syntheticReason":      syntheticReason,
						"provenanceVerifiedAt": provenanceVerifiedAt,
					},
					"startedAt":   started,
					"completedAt": completed,
					"createdAt":   created,
				})
			}
		}
		jsonOK(w, out)
	}
}

func backtestRunDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/backtests/runs/"), "/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		var (
			rowID, runID, iid, sid, status, stats, cfg, errText, datasetID, datasetHash, dataSourceType, sourceProvider, syntheticReason string
			symbols                                                                                                                      []string
			from, to, started, created                                                                                                   time.Time
			completed, provenanceVerifiedAt                                                                                              *time.Time
			isSynthetic                                                                                                                  bool
		)
		err := pool.QueryRow(r.Context(), `
			SELECT id::text, external_run_id, COALESCE(instance_id::text,''), strategy_type_id, symbols, run_from, run_to,
			       status, stats::text, config_snapshot::text, COALESCE(dataset_id,''), COALESCE(dataset_hash,''), data_source_type,
			       COALESCE(source_provider,''), is_synthetic, COALESCE(synthetic_reason,''), provenance_verified_at,
			       started_at, completed_at, COALESCE(error,''), created_at
			FROM backtest_runs
			WHERE external_run_id = $1 OR id::text = $1
		`, id).Scan(&rowID, &runID, &iid, &sid, &symbols, &from, &to, &status, &stats, &cfg, &datasetID, &datasetHash, &dataSourceType,
			&sourceProvider, &isSynthetic, &syntheticReason, &provenanceVerifiedAt, &started, &completed, &errText, &created)
		if err != nil {
			if err == pgx.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type symbolAgg struct {
			Symbol  string  `json:"symbol"`
			Trades  int     `json:"trades"`
			Wins    int     `json:"-"`
			WinRate float64 `json:"winRate"`
			PnL     float64 `json:"pnl"`
		}
		trades := make([]map[string]any, 0, 128)
		bySymbolMap := map[string]*symbolAgg{}
		trRows, trErr := pool.Query(r.Context(), `
			SELECT symbol, side, entry_price, exit_price, quantity, pnl, pnl_pct, opened_at, closed_at, metadata::text
			FROM backtest_trades
			WHERE run_id = $1::uuid
			ORDER BY COALESCE(opened_at, created_at) ASC
		`, rowID)
		if trErr == nil {
			defer trRows.Close()
			for trRows.Next() {
				var (
					symbol, side, metadata                  string
					entryPrice, exitPrice, qty, pnl, pnlPct sql.NullFloat64
					openedAt, closedAt                      *time.Time
				)
				if err := trRows.Scan(&symbol, &side, &entryPrice, &exitPrice, &qty, &pnl, &pnlPct, &openedAt, &closedAt, &metadata); err != nil {
					continue
				}
				trades = append(trades, map[string]any{
					"symbol":     symbol,
					"side":       side,
					"entryPrice": nullableFloat(entryPrice),
					"exitPrice":  nullableFloat(exitPrice),
					"quantity":   nullableFloat(qty),
					"pnl":        nullableFloat(pnl),
					"pnlPct":     nullableFloat(pnlPct),
					"openedAt":   openedAt,
					"closedAt":   closedAt,
					"metadata":   json.RawMessage(metadata),
				})
				agg := bySymbolMap[symbol]
				if agg == nil {
					agg = &symbolAgg{Symbol: symbol}
					bySymbolMap[symbol] = agg
				}
				agg.Trades++
				if pnl.Valid && pnl.Float64 > 0 {
					agg.Wins++
				}
				if pnl.Valid {
					agg.PnL += pnl.Float64
				}
			}
		}
		bySymbol := make([]map[string]any, 0, len(bySymbolMap))
		for _, agg := range bySymbolMap {
			if agg.Trades > 0 {
				agg.WinRate = float64(agg.Wins) / float64(agg.Trades)
			}
			bySymbol = append(bySymbol, map[string]any{
				"symbol":  agg.Symbol,
				"trades":  agg.Trades,
				"winRate": agg.WinRate,
				"pnl":     agg.PnL,
			})
		}
		var parentRunID string
		_ = pool.QueryRow(r.Context(), `
			SELECT id::text
			FROM runs
			WHERE backtest_run_id = $1::uuid
			ORDER BY started_at DESC
			LIMIT 1
		`, rowID).Scan(&parentRunID)

		jsonOK(w, map[string]any{
			"id":          rowID,
			"runId":       runID,
			"parentRunId": parentRunID,
			"instanceId":  iid,
			"strategyId":  sid,
			"symbols":     symbols,
			"from":        from,
			"to":          to,
			"status":      status,
			"stats":       json.RawMessage(stats),
			"config":      json.RawMessage(cfg),
			"datasetId":   datasetID,
			"datasetHash": datasetHash,
			"provenance": map[string]any{
				"dataSourceType":       dataSourceType,
				"sourceProvider":       sourceProvider,
				"isSynthetic":          isSynthetic,
				"syntheticReason":      syntheticReason,
				"provenanceVerifiedAt": provenanceVerifiedAt,
			},
			"trades":      trades,
			"bySymbol":    bySymbol,
			"startedAt":   started,
			"completedAt": completed,
			"createdAt":   created,
			"error":       errText,
		})
	}
}

func researchProjectsHandler(pool *pgxpool.Pool, orchestratorURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rows, err := pool.Query(r.Context(), `
				SELECT id::text, name, COALESCE(description,''), COALESCE(owner,''), status,
				       COALESCE(base_instance_id::text,''), parameter_grid::text, train_from, train_to, test_from, test_to,
				       created_at, updated_at
				FROM research_projects
				ORDER BY updated_at DESC
			`)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			out := make([]map[string]any, 0, 32)
			for rows.Next() {
				var id, name, desc, owner, status, baseID, params string
				var trainFrom, trainTo, testFrom, testTo *time.Time
				var created, updated time.Time
				if err := rows.Scan(&id, &name, &desc, &owner, &status, &baseID, &params,
					&trainFrom, &trainTo, &testFrom, &testTo, &created, &updated); err == nil {
					out = append(out, map[string]any{
						"id":             id,
						"name":           name,
						"description":    desc,
						"owner":          owner,
						"status":         status,
						"baseInstanceId": baseID,
						"parameterGrid":  json.RawMessage(params),
						"trainFrom":      trainFrom,
						"trainTo":        trainTo,
						"testFrom":       testFrom,
						"testTo":         testTo,
						"createdAt":      created,
						"updatedAt":      updated,
					})
				}
			}
			jsonOK(w, out)
		case http.MethodPost:
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			name, _ := req["name"].(string)
			if name == "" {
				http.Error(w, "name is required", http.StatusBadRequest)
				return
			}
			desc, _ := req["description"].(string)
			owner, _ := req["owner"].(string)
			baseInstanceID, _ := req["baseInstanceId"].(string)
			paramRaw, _ := json.Marshal(req["parameterGrid"])
			if len(paramRaw) == 0 || string(paramRaw) == "null" {
				paramRaw = []byte("{}")
			}
			trainFrom := parseOptionalDate(req["trainFrom"])
			trainTo := parseOptionalDate(req["trainTo"])
			testFrom := parseOptionalDate(req["testFrom"])
			testTo := parseOptionalDate(req["testTo"])
			var id string
			err := pool.QueryRow(r.Context(), `
				INSERT INTO research_projects (
					name, description, owner, base_instance_id, parameter_grid, status,
					train_from, train_to, test_from, test_to
				)
				VALUES (
					$1, $2, $3, NULLIF($4,'')::uuid, $5::jsonb, 'draft',
					$6, $7, $8, $9
				)
				RETURNING id::text
			`, name, desc, owner, baseInstanceID, string(paramRaw), trainFrom, trainTo, testFrom, testTo).Scan(&id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			jsonOK(w, map[string]any{"id": id, "name": name})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func researchProjectsDetailHandler(pool *pgxpool.Pool, orchestratorURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/research/projects/"), "/")
		parts := strings.Split(path, "/")
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		projectID := parts[0]
		action := ""
		if len(parts) > 1 {
			action = parts[1]
		}
		switch {
		case r.Method == http.MethodGet && action == "":
			var (
				id, name, desc, owner, status, params string
			)
			err := pool.QueryRow(r.Context(), `
				SELECT id::text, name, COALESCE(description,''), COALESCE(owner,''), status, parameter_grid::text
				FROM research_projects
				WHERE id = $1::uuid
			`, projectID).Scan(&id, &name, &desc, &owner, &status, &params)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			jsonOK(w, map[string]any{
				"id":            id,
				"name":          name,
				"description":   desc,
				"owner":         owner,
				"status":        status,
				"parameterGrid": json.RawMessage(params),
			})
		case r.Method == http.MethodPost && action == "run":
			var req backtestRunRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			project, err := loadResearchProject(r.Context(), pool, projectID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			baseReq := req
			if baseReq.InstanceID == "" {
				baseReq.InstanceID = project.BaseInstanceID
			}
			if baseReq.StrategyID == "" {
				baseReq.StrategyID = instanceStrategyID(r.Context(), pool, baseReq.InstanceID)
			}

			if err := setResearchProjectStatus(r.Context(), pool, projectID, "running"); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			flowID := observability.FlowIDFromContext(r.Context())
			var parentRunID string
			summaryStart := map[string]any{
				"projectId": projectID,
				"status":    "running",
			}
			_ = pool.QueryRow(r.Context(), `
				INSERT INTO runs (
					run_type, status, flow_id, source, instance_id, summary, started_at,
					data_source_type, source_provider, dataset_id, is_synthetic, provenance_verified_at
				)
				VALUES ('research', 'running', $1, 'api', NULLIF($2,'')::uuid, $3::jsonb, NOW(), 'real', 'research.project', NULLIF($4,''), FALSE, NOW())
				RETURNING id::text
			`, flowID, baseReq.InstanceID, toJSONString(summaryStart), baseReq.DatasetID).Scan(&parentRunID)

			orchestratorURL = strings.TrimRight(strings.TrimSpace(orchestratorURL), "/")
			if orchestratorURL == "" {
				orchestratorURL = strings.TrimRight(strings.TrimSpace(envStr("ORCHESTRATOR_URL", "http://localhost:8091")), "/")
			}
			remoteReq := map[string]any{
				"projectId":        projectID,
				"instanceId":       baseReq.InstanceID,
				"strategyId":       baseReq.StrategyID,
				"strategyConfigId": baseReq.StrategyConfigID,
				"datasetId":        baseReq.DatasetID,
				"symbolsOverride":  baseReq.SymbolsOverride,
				"from":             baseReq.From,
				"to":               baseReq.To,
				"trainFrom":        nullableTimeString(project.TrainFrom),
				"trainTo":          nullableTimeString(project.TrainTo),
				"testFrom":         nullableTimeString(project.TestFrom),
				"testTo":           nullableTimeString(project.TestTo),
				"parameterGrid":    project.ParameterGrid,
				"seed":             baseReq.Seed,
				"initialCapital":   baseReq.InitialCapital,
				"riskPerTrade":     baseReq.RiskPerTrade,
			}
			payload, _ := json.Marshal(remoteReq)
			remoteRaw, err := proxyPost(r.Context(), orchestratorURL+"/research/projects/run", payload)
			if err != nil {
				_ = setResearchProjectStatus(r.Context(), pool, projectID, "failed")
				_, _ = pool.Exec(r.Context(), `
					UPDATE runs
					SET status='failed', error=$2, completed_at=NOW()
					WHERE id = $1::uuid
				`, parentRunID, err.Error())
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			var remoteResp map[string]any
			if err := json.Unmarshal(remoteRaw, &remoteResp); err != nil {
				http.Error(w, "invalid research runtime response", http.StatusBadGateway)
				return
			}
			runRows := toSliceAny(remoteResp["runs"])
			runSummaries := make([]map[string]any, 0, len(runRows))
			failed := 0
			for _, row := range runRows {
				item, ok := row.(map[string]any)
				if !ok {
					continue
				}
				index := intFromAny(item["index"])
				combo, _ := item["combo"].(map[string]any)
				trainRunID, trainSummary := extractRunnerLeg(item["train"])
				testRunID, testSummary := extractRunnerLeg(item["test"])
				rank, _ := toFloat64(item["rankScore"])
				errText := strings.TrimSpace(toString(item["error"]))
				if errText != "" {
					failed++
				}
				metrics := map[string]any{
					"train": normalizeBacktestMetrics(trainSummary),
					"test":  normalizeBacktestMetrics(testSummary),
					"rank":  rank,
				}
				seed := deterministicSweepSeed(projectID, index, combo)
				params := map[string]any{
					"combo":    combo,
					"seed":     seed,
					"instance": baseReq.InstanceID,
				}
				lineage := map[string]any{
					"phase":       "walk_forward",
					"index":       index,
					"combo":       combo,
					"trainRunId":  trainRunID,
					"testRunId":   testRunID,
					"runnerJobId": toString(remoteResp["jobId"]),
				}
				paramsJSON, _ := json.Marshal(params)
				metricsJSON, _ := json.Marshal(metrics)
				lineageJSON, _ := json.Marshal(lineage)
				statusRow := "completed"
				if errText != "" {
					statusRow = "failed"
				}
				var projectRunID string
				_ = pool.QueryRow(r.Context(), `
					INSERT INTO research_project_runs (
						project_id, backtest_run_id, status, parameters, metrics, rank_score, lineage, error, started_at, completed_at
					)
					VALUES (
						$1::uuid,
						(SELECT id FROM backtest_runs WHERE external_run_id = NULLIF($2,'')),
						$3,
						$4::jsonb,
						$5::jsonb,
						$6,
						$7::jsonb,
						NULLIF($8,''),
						NOW(),
						NOW()
					)
					RETURNING id::text
				`, projectID, testRunID, statusRow, string(paramsJSON), string(metricsJSON), rank, string(lineageJSON), errText).Scan(&projectRunID)

				runSummaries = append(runSummaries, map[string]any{
					"projectRunId": projectRunID,
					"index":        index,
					"combo":        combo,
					"trainRunId":   trainRunID,
					"testRunId":    testRunID,
					"rankScore":    rank,
					"metrics":      metrics,
					"error":        errText,
				})
			}

			status := toString(remoteResp["status"])
			if status == "" {
				status = "completed"
				if failed > 0 {
					status = "degraded"
				}
				if failed == len(runSummaries) && len(runSummaries) > 0 {
					status = "failed"
				}
			}
			_ = setResearchProjectStatus(r.Context(), pool, projectID, status)
			finalSummary := map[string]any{
				"projectId":       projectID,
				"status":          status,
				"totalCombos":     intFromAny(remoteResp["totalCombos"]),
				"failedCombos":    intFromAny(remoteResp["failedCombos"]),
				"successful":      len(runSummaries) - failed,
				"projectRunIds":   runSummaries,
				"researchJobId":   toString(remoteResp["jobId"]),
				"researchRuntime": "cmd/research",
			}
			if parentRunID != "" {
				finalStatus := "completed"
				if status == "failed" {
					finalStatus = "failed"
				}
				_, _ = pool.Exec(r.Context(), `
					UPDATE runs
					SET status = $2, summary = $3::jsonb, completed_at = NOW(), error = CASE WHEN $2='failed' THEN 'research project run failed' ELSE NULL END
					WHERE id = $1::uuid
				`, parentRunID, finalStatus, toJSONString(finalSummary))
			}
			jsonOK(w, map[string]any{
				"projectId":    projectID,
				"status":       status,
				"parentRunId":  parentRunID,
				"totalCombos":  intFromAny(remoteResp["totalCombos"]),
				"failedCombos": intFromAny(remoteResp["failedCombos"]),
				"jobId":        toString(remoteResp["jobId"]),
				"runs":         runSummaries,
			})
		case r.Method == http.MethodGet && action == "runs":
			rows, err := pool.Query(r.Context(), `
				SELECT id::text, COALESCE(backtest_run_id::text,''), status, parameters::text, metrics::text, rank_score, lineage::text, COALESCE(error,''), started_at, completed_at
				FROM research_project_runs
				WHERE project_id = $1::uuid
				ORDER BY created_at DESC
			`, projectID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			out := make([]map[string]any, 0, 32)
			for rows.Next() {
				var id, btID, status, params, metrics, lineage, errorText string
				var rank float64
				var started, completed *time.Time
				if err := rows.Scan(&id, &btID, &status, &params, &metrics, &rank, &lineage, &errorText, &started, &completed); err == nil {
					out = append(out, map[string]any{
						"id":            id,
						"backtestRunId": btID,
						"status":        status,
						"parameters":    json.RawMessage(params),
						"metrics":       json.RawMessage(metrics),
						"rankScore":     rank,
						"lineage":       json.RawMessage(lineage),
						"error":         errorText,
						"startedAt":     started,
						"completedAt":   completed,
					})
				}
			}
			jsonOK(w, out)
		default:
			http.NotFound(w, r)
		}
	}
}

func testingStatusHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rows, err := pool.Query(r.Context(), `
			SELECT gate_name, status, COALESCE(last_run_id::text,''), details::text, last_run_at, updated_at
			FROM gate_status
			ORDER BY gate_name
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		statusByGate := map[string]map[string]any{}
		for rows.Next() {
			var gate, status, runID, details string
			var lastRunAt, updatedAt *time.Time
			if err := rows.Scan(&gate, &status, &runID, &details, &lastRunAt, &updatedAt); err == nil {
				statusByGate[gate] = map[string]any{
					"gate":      gate,
					"status":    status,
					"lastRunId": runID,
					"details":   json.RawMessage(details),
					"lastRunAt": lastRunAt,
					"updatedAt": updatedAt,
				}
			}
		}
		expected := []string{"Gate0", "Gate1", "Gate2", "Gate3", "Gate4", "Gate5", "Gate6", "Gate7"}
		out := make([]map[string]any, 0, len(expected))
		for _, gate := range expected {
			if existing, ok := statusByGate[gate]; ok {
				out = append(out, existing)
				continue
			}
			out = append(out, map[string]any{
				"gate":      gate,
				"status":    "not_started",
				"lastRunId": "",
				"details":   json.RawMessage(`{}`),
				"lastRunAt": nil,
				"updatedAt": nil,
			})
		}
		jsonOK(w, out)
	}
}

func testingTriggerHandler(pool *pgxpool.Pool, gate, testType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.EqualFold(os.Getenv("ALLOW_LIVE_TRADING"), "true") {
			http.Error(w, "testing endpoints are paper-only", http.StatusForbidden)
			return
		}
		started := time.Now().UTC()
		artifactPath, summary := runTrustGateJob(r.Context(), pool, gate, testType)
		statusValue := "completed"
		gateStatus := "passed"
		if strings.EqualFold(toString(summary["status"]), "failed") {
			statusValue = "failed"
			gateStatus = "failed"
		}
		summary["artifactUri"] = artifactPath
		summaryJSON, _ := json.Marshal(summary)
		var runID string
		if pool != nil {
			_ = pool.QueryRow(r.Context(), `
				INSERT INTO test_runs (test_name, status, summary, artifact_uri, started_at, completed_at)
				VALUES ($1, $2, $3::jsonb, $4, $5, NOW())
				RETURNING id::text
			`, testType, statusValue, string(summaryJSON), artifactPath, started).Scan(&runID)
			_, _ = pool.Exec(r.Context(), `
				INSERT INTO gate_status (gate_name, status, last_run_id, details, last_run_at)
				VALUES ($1, $2, $3::uuid, $4::jsonb, NOW())
				ON CONFLICT (gate_name)
				DO UPDATE SET
					status = EXCLUDED.status,
					last_run_id = EXCLUDED.last_run_id,
					details = EXCLUDED.details,
					last_run_at = NOW()
			`, gate, gateStatus, runID, string(summaryJSON))
		}
		jsonOK(w, map[string]any{
			"gate":        gate,
			"testRunId":   runID,
			"status":      statusValue,
			"artifactUri": artifactPath,
			"summary":     summary,
		})
	}
}

func runsListHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parseIntParam(r.URL.Query().Get("limit"), 100)
		rows, err := pool.Query(r.Context(), `
			SELECT id::text, run_type, status, COALESCE(flow_id,''), COALESCE(source,''), COALESCE(instance_id::text,''), summary::text,
			       COALESCE(dataset_id,''), COALESCE(dataset_hash,''), data_source_type, COALESCE(source_provider,''), is_synthetic,
			       COALESCE(synthetic_reason,''), provenance_verified_at, started_at, completed_at, COALESCE(error,'')
			FROM runs
			ORDER BY started_at DESC
			LIMIT $1
		`, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		out := make([]map[string]any, 0, limit)
		for rows.Next() {
			var id, runType, status, flowID, source, instanceID, summary, errText, datasetID, datasetHash, dataSourceType, sourceProvider, syntheticReason string
			var isSynthetic bool
			var started time.Time
			var completed, provenanceVerifiedAt *time.Time
			if err := rows.Scan(&id, &runType, &status, &flowID, &source, &instanceID, &summary, &datasetID, &datasetHash, &dataSourceType,
				&sourceProvider, &isSynthetic, &syntheticReason, &provenanceVerifiedAt, &started, &completed, &errText); err == nil {
				out = append(out, map[string]any{
					"id":          id,
					"runType":     runType,
					"status":      status,
					"flowId":      flowID,
					"source":      source,
					"instanceId":  instanceID,
					"summary":     json.RawMessage(summary),
					"datasetId":   datasetID,
					"datasetHash": datasetHash,
					"provenance": map[string]any{
						"dataSourceType":       dataSourceType,
						"sourceProvider":       sourceProvider,
						"isSynthetic":          isSynthetic,
						"syntheticReason":      syntheticReason,
						"provenanceVerifiedAt": provenanceVerifiedAt,
					},
					"startedAt":   started,
					"completedAt": completed,
					"error":       errText,
				})
			}
		}
		jsonOK(w, out)
	}
}

func runsDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/runs/"), "/")
		if path == "" {
			http.NotFound(w, r)
			return
		}
		parts := strings.Split(path, "/")
		runID := parts[0]
		if len(parts) > 1 && parts[1] == "timeline" {
			flowID := runFlowID(r.Context(), pool, runID)
			rows, _ := pool.Query(r.Context(), `
				SELECT id, category, action, outcome, message, metadata::text, timestamp
				FROM audit_events
				WHERE correlation_id = $1
				ORDER BY timestamp ASC
			`, flowID)
			timeline := make([]map[string]any, 0, 64)
			if rows != nil {
				defer rows.Close()
				for rows.Next() {
					var id, cat, action, outcome, message, meta string
					var ts time.Time
					if err := rows.Scan(&id, &cat, &action, &outcome, &message, &meta, &ts); err == nil {
						timeline = append(timeline, map[string]any{
							"id":       id,
							"type":     "audit",
							"category": cat,
							"action":   action,
							"outcome":  outcome,
							"message":  message,
							"metadata": json.RawMessage(meta),
							"ts":       ts,
						})
					}
				}
			}
			jsonOK(w, map[string]any{"runId": runID, "timeline": timeline})
			return
		}

		var (
			id, runType, status, flowID, source, instanceID, summary, errText, datasetID, datasetHash, dataSourceType, sourceProvider, syntheticReason string
			started                                                                                                                                    time.Time
			completed, provenanceVerifiedAt                                                                                                            *time.Time
			isSynthetic                                                                                                                                bool
		)
		err := pool.QueryRow(r.Context(), `
			SELECT id::text, run_type, status, COALESCE(flow_id,''), COALESCE(source,''), COALESCE(instance_id::text,''),
			       summary::text, COALESCE(dataset_id,''), COALESCE(dataset_hash,''), data_source_type, COALESCE(source_provider,''),
			       is_synthetic, COALESCE(synthetic_reason,''), provenance_verified_at, started_at, completed_at, COALESCE(error,'')
			FROM runs
			WHERE id = $1::uuid
		`, runID).Scan(&id, &runType, &status, &flowID, &source, &instanceID, &summary, &datasetID, &datasetHash, &dataSourceType, &sourceProvider,
			&isSynthetic, &syntheticReason, &provenanceVerifiedAt, &started, &completed, &errText)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		jsonOK(w, map[string]any{
			"id":          id,
			"runType":     runType,
			"status":      status,
			"flowId":      flowID,
			"source":      source,
			"instanceId":  instanceID,
			"summary":     json.RawMessage(summary),
			"datasetId":   datasetID,
			"datasetHash": datasetHash,
			"provenance": map[string]any{
				"dataSourceType":       dataSourceType,
				"sourceProvider":       sourceProvider,
				"isSynthetic":          isSynthetic,
				"syntheticReason":      syntheticReason,
				"provenanceVerifiedAt": provenanceVerifiedAt,
			},
			"startedAt":   started,
			"completedAt": completed,
			"error":       errText,
		})
	}
}

func aiDecisionsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parseIntParam(r.URL.Query().Get("limit"), 100)
		rows, err := pool.Query(r.Context(), `
			SELECT d.id::text, COALESCE(d.run_id::text,''), COALESCE(d.flow_id,''), d.role, COALESCE(d.provider,''), COALESCE(d.model,''),
			       d.schema_valid, COALESCE(d.decision,''), COALESCE(d.reasoning,''), d.prompt::text, d.response::text, d.rule_trace::text,
			       d.created_at
			FROM ai_decisions d
			ORDER BY d.created_at DESC
			LIMIT $1
		`, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		out := make([]map[string]any, 0, limit)
		for rows.Next() {
			var id, runID, flowID, role, provider, model, decision, reasoning, prompt, response, trace string
			var schemaValid bool
			var created time.Time
			if err := rows.Scan(&id, &runID, &flowID, &role, &provider, &model, &schemaValid, &decision, &reasoning,
				&prompt, &response, &trace, &created); err == nil {
				out = append(out, map[string]any{
					"id":          id,
					"runId":       runID,
					"flowId":      flowID,
					"role":        role,
					"provider":    provider,
					"model":       model,
					"schemaValid": schemaValid,
					"decision":    decision,
					"reasoning":   reasoning,
					"prompt":      json.RawMessage(prompt),
					"response":    json.RawMessage(response),
					"ruleTrace":   json.RawMessage(trace),
					"createdAt":   created,
				})
			}
		}
		jsonOK(w, out)
	}
}

func aiDecisionDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/ai-decisions/"), "/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		var (
			decisionID, runID, flowID, role, provider, model, decision, reasoning, prompt, response, trace string
			schemaValid                                                                                    bool
			created                                                                                        time.Time
		)
		err := pool.QueryRow(r.Context(), `
			SELECT id::text, COALESCE(run_id::text,''), COALESCE(flow_id,''), role, COALESCE(provider,''), COALESCE(model,''),
			       schema_valid, COALESCE(decision,''), COALESCE(reasoning,''), prompt::text, response::text, rule_trace::text, created_at
			FROM ai_decisions
			WHERE id = $1::uuid
		`, id).Scan(&decisionID, &runID, &flowID, &role, &provider, &model, &schemaValid, &decision, &reasoning, &prompt, &response, &trace, &created)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		acceptance := map[string]any{}
		var acceptedBy, reason, ruleTrace string
		var accepted bool
		var acceptedAt time.Time
		if err := pool.QueryRow(r.Context(), `
			SELECT accepted, COALESCE(accepted_by,''), COALESCE(reason,''), rule_trace::text, created_at
			FROM ai_decision_acceptance
			WHERE decision_id = $1::uuid
			ORDER BY created_at DESC
			LIMIT 1
		`, id).Scan(&accepted, &acceptedBy, &reason, &ruleTrace, &acceptedAt); err == nil {
			acceptance["accepted"] = accepted
			acceptance["acceptedBy"] = acceptedBy
			acceptance["reason"] = reason
			acceptance["ruleTrace"] = json.RawMessage(ruleTrace)
			acceptance["createdAt"] = acceptedAt
		}
		jsonOK(w, map[string]any{
			"id":          decisionID,
			"runId":       runID,
			"flowId":      flowID,
			"role":        role,
			"provider":    provider,
			"model":       model,
			"schemaValid": schemaValid,
			"decision":    decision,
			"reasoning":   reasoning,
			"prompt":      json.RawMessage(prompt),
			"response":    json.RawMessage(response),
			"ruleTrace":   json.RawMessage(trace),
			"acceptance":  acceptance,
			"createdAt":   created,
		})
	}
}

func gatesHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return testingStatusHandler(pool)
}

func testRunsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parseIntParam(r.URL.Query().Get("limit"), 100)
		rows, err := pool.Query(r.Context(), `
			SELECT id::text, COALESCE(run_id::text,''), test_name, status, summary::text, COALESCE(artifact_uri,''),
			       started_at, completed_at, created_at
			FROM test_runs
			ORDER BY created_at DESC
			LIMIT $1
		`, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		out := make([]map[string]any, 0, limit)
		for rows.Next() {
			var id, runID, name, status, summary, artifact string
			var started, completed, created *time.Time
			if err := rows.Scan(&id, &runID, &name, &status, &summary, &artifact, &started, &completed, &created); err == nil {
				out = append(out, map[string]any{
					"id":          id,
					"runId":       runID,
					"testName":    name,
					"status":      status,
					"summary":     json.RawMessage(summary),
					"artifactUri": artifact,
					"startedAt":   started,
					"completedAt": completed,
					"createdAt":   created,
				})
			}
		}
		jsonOK(w, out)
	}
}

type researchProjectRecord struct {
	ID             string
	BaseInstanceID string
	ParameterGrid  map[string]any
	TrainFrom      *time.Time
	TrainTo        *time.Time
	TestFrom       *time.Time
	TestTo         *time.Time
}

func loadResearchProject(ctx context.Context, pool *pgxpool.Pool, projectID string) (*researchProjectRecord, error) {
	var (
		id, baseInstanceID, paramRaw string
		trainFrom, trainTo           *time.Time
		testFrom, testTo             *time.Time
	)
	err := pool.QueryRow(ctx, `
		SELECT id::text, COALESCE(base_instance_id::text,''), parameter_grid::text, train_from, train_to, test_from, test_to
		FROM research_projects
		WHERE id = $1::uuid
	`, projectID).Scan(&id, &baseInstanceID, &paramRaw, &trainFrom, &trainTo, &testFrom, &testTo)
	if err != nil {
		return nil, fmt.Errorf("load project: %w", err)
	}
	grid := map[string]any{}
	if strings.TrimSpace(paramRaw) != "" {
		_ = json.Unmarshal([]byte(paramRaw), &grid)
	}
	return &researchProjectRecord{
		ID:             id,
		BaseInstanceID: baseInstanceID,
		ParameterGrid:  grid,
		TrainFrom:      trainFrom,
		TrainTo:        trainTo,
		TestFrom:       testFrom,
		TestTo:         testTo,
	}, nil
}

func setResearchProjectStatus(ctx context.Context, pool *pgxpool.Pool, projectID, status string) error {
	_, err := pool.Exec(ctx, `UPDATE research_projects SET status = $2 WHERE id = $1::uuid`, projectID, status)
	return err
}

func resolveWalkForwardWindows(project *researchProjectRecord, req backtestRunRequest) (time.Time, time.Time, time.Time, time.Time) {
	fallbackFrom := parseDateOnly(req.From, time.Now().UTC().AddDate(0, 0, -30))
	fallbackTo := parseDateOnly(req.To, time.Now().UTC())
	if !fallbackTo.After(fallbackFrom) {
		fallbackTo = fallbackFrom.AddDate(0, 0, 5)
	}
	trainFrom := fallbackFrom
	trainTo := fallbackTo
	testFrom := fallbackFrom
	testTo := fallbackTo
	if project.TrainFrom != nil {
		trainFrom = project.TrainFrom.UTC()
	}
	if project.TrainTo != nil {
		trainTo = project.TrainTo.UTC()
	}
	if project.TestFrom != nil {
		testFrom = project.TestFrom.UTC()
	}
	if project.TestTo != nil {
		testTo = project.TestTo.UTC()
	}
	// Fallback split if explicit windows are missing.
	if project.TrainFrom == nil || project.TrainTo == nil || project.TestFrom == nil || project.TestTo == nil {
		total := fallbackTo.Sub(fallbackFrom)
		if total <= 0 {
			total = 48 * time.Hour
		}
		split := fallbackFrom.Add(time.Duration(float64(total) * 0.7))
		trainFrom = fallbackFrom
		trainTo = split
		testFrom = split.Add(24 * time.Hour)
		testTo = fallbackTo
		if !testTo.After(testFrom) {
			testTo = testFrom.AddDate(0, 0, 2)
		}
	}
	if !trainTo.After(trainFrom) {
		trainTo = trainFrom.AddDate(0, 0, 2)
	}
	if !testTo.After(testFrom) {
		testTo = testFrom.AddDate(0, 0, 2)
	}
	return trainFrom, trainTo, testFrom, testTo
}

func applySweepCombo(base backtestRunRequest, combo map[string]any, seed int64) backtestRunRequest {
	out := base
	out.Seed = seed
	for key, value := range combo {
		switch key {
		case "strategyId":
			if s := toString(value); s != "" {
				out.StrategyID = s
			}
		case "strategyConfigId":
			if s := toString(value); s != "" {
				out.StrategyConfigID = s
			}
		case "datasetId":
			if s := toString(value); s != "" {
				out.DatasetID = s
			}
		case "seed":
			if v, ok := toInt64(value); ok {
				out.Seed = v
			}
		case "initialCapital":
			if v, ok := toFloat64(value); ok {
				out.InitialCapital = v
			}
		case "riskPerTrade":
			if v, ok := toFloat64(value); ok {
				out.RiskPerTrade = v
			}
		case "symbols", "symbolsOverride":
			if ss := toStringSlice(value); len(ss) > 0 {
				out.SymbolsOverride = ss
			}
		}
	}
	return out
}

func normalizeBacktestMetrics(raw any) map[string]float64 {
	out := map[string]float64{
		"totalTrades":  0,
		"winRate":      0,
		"maxDrawdown":  0,
		"totalReturn":  0,
		"finalCapital": 0,
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return out
	}
	out["totalTrades"], _ = toFloat64(m["total_trades"])
	out["winRate"], _ = toFloat64(m["win_rate"])
	out["maxDrawdown"], _ = toFloat64(m["max_drawdown"])
	out["totalReturn"], _ = toFloat64(m["total_return"])
	out["finalCapital"], _ = toFloat64(m["final_capital"])
	periodDays := 1.0
	if out["totalTrades"] > 0 {
		periodDays = out["totalTrades"] / 3.0
		if periodDays < 1 {
			periodDays = 1
		}
	}
	out["avgDailyPnL"] = out["totalReturn"] / periodDays
	if out["totalReturn"] < 0 {
		out["tailLoss"] = -out["totalReturn"]
	}
	return out
}

func computeResearchRankScore(metrics map[string]float64) float64 {
	maxDD := metrics["maxDrawdown"]
	avgDaily := metrics["avgDailyPnL"]
	tailLoss := metrics["tailLoss"]
	return (avgDaily * 100) - (maxDD * 100) - (tailLoss * 50)
}

func deterministicSweepSeed(projectID string, idx int, combo map[string]any) int64 {
	encoded, _ := json.Marshal(combo)
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%s", projectID, idx, string(encoded))))
	return int64(sum[0])<<56 | int64(sum[1])<<48 | int64(sum[2])<<40 | int64(sum[3])<<32 |
		int64(sum[4])<<24 | int64(sum[5])<<16 | int64(sum[6])<<8 | int64(sum[7])
}

func expandParameterGrid(grid map[string]any, maxCombos int) []map[string]any {
	if len(grid) == 0 {
		return nil
	}
	keys := make([]string, 0, len(grid))
	for k := range grid {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	valuesByKey := make([][]any, 0, len(keys))
	for _, key := range keys {
		raw := grid[key]
		switch typed := raw.(type) {
		case []any:
			if len(typed) == 0 {
				valuesByKey = append(valuesByKey, []any{nil})
			} else {
				valuesByKey = append(valuesByKey, typed)
			}
		default:
			valuesByKey = append(valuesByKey, []any{typed})
		}
	}
	out := make([]map[string]any, 0, 16)
	var walk func(i int, current map[string]any)
	walk = func(i int, current map[string]any) {
		if maxCombos > 0 && len(out) >= maxCombos {
			return
		}
		if i >= len(keys) {
			cloned := make(map[string]any, len(current))
			for k, v := range current {
				cloned[k] = v
			}
			out = append(out, cloned)
			return
		}
		key := keys[i]
		for _, value := range valuesByKey[i] {
			current[key] = value
			walk(i+1, current)
		}
		delete(current, key)
	}
	walk(0, map[string]any{})
	return out
}

func runBacktestAndPersist(ctx context.Context, pool *pgxpool.Pool, orchestratorURL string, req backtestRunRequest, source string) (map[string]any, error) {
	orchestratorURL = strings.TrimRight(strings.TrimSpace(orchestratorURL), "/")
	if orchestratorURL == "" {
		orchestratorURL = strings.TrimRight(strings.TrimSpace(envStr("ORCHESTRATOR_URL", "http://localhost:8091")), "/")
	}
	if req.DatasetID == "" {
		req.DatasetID = envStr("BACKTEST_DATASET_ID", "")
	}
	if req.StrategyID == "" {
		req.StrategyID = req.StrategyConfigID
	}
	if req.StrategyID == "" {
		req.StrategyID = "rsi_momentum_v1"
	}
	if req.InstanceID != "" {
		if req.Parameters == nil || req.SessionTimezone == "" || req.FlattenByCloseTime == "" {
			if params, tz, flatten, err := loadInstanceBacktestConfig(ctx, pool, req.InstanceID); err == nil {
				if req.Parameters == nil {
					req.Parameters = params
				}
				if req.SessionTimezone == "" {
					req.SessionTimezone = tz
				}
				if req.FlattenByCloseTime == "" {
					req.FlattenByCloseTime = flatten
				}
			}
		}
	}
	symbols := req.SymbolsOverride
	if len(symbols) == 0 {
		symbols = []string{envStr("BACKTEST_DEFAULT_SYMBOL", "SPY")}
	}
	fromDate := parseDateOnly(req.From, time.Now().UTC().AddDate(0, 0, -30))
	toDate := parseDateOnly(req.To, time.Now().UTC())
	if !toDate.After(fromDate) {
		toDate = fromDate.AddDate(0, 0, 5)
	}
	payload := map[string]any{
		"strategy":        req.StrategyID,
		"symbols":         symbols,
		"start_date":      fromDate.Format("2006-01-02"),
		"end_date":        toDate.Format("2006-01-02"),
		"dataset_id":      req.DatasetID,
		"seed":            req.Seed,
		"initial_capital": req.InitialCapital,
		"risk_per_trade":  req.RiskPerTrade,
	}
	if len(req.Parameters) > 0 {
		payload["parameters"] = req.Parameters
	}
	if req.SessionTimezone != "" {
		payload["session_timezone"] = req.SessionTimezone
	}
	if req.FlattenByCloseTime != "" {
		payload["flatten_by_close_time"] = req.FlattenByCloseTime
	}
	body, _ := json.Marshal(payload)
	respRaw, err := proxyPost(ctx, orchestratorURL+"/backtest", body)
	if err != nil {
		return nil, err
	}
	var upstream map[string]any
	if err := json.Unmarshal(respRaw, &upstream); err != nil {
		return nil, fmt.Errorf("invalid backtest response")
	}
	runID := toString(upstream["run_id"])
	if runID == "" {
		runID = uuid.NewString()
	}
	statsJSON, _ := json.Marshal(map[string]any{
		"totalTrades":   upstream["total_trades"],
		"winningTrades": upstream["winning_trades"],
		"losingTrades":  upstream["losing_trades"],
		"winRate":       upstream["win_rate"],
		"totalReturn":   upstream["total_return"],
		"sharpe":        upstream["sharpe_ratio"],
		"maxDrawdown":   upstream["max_drawdown"],
		"finalCapital":  upstream["final_capital"],
	})
	cfgJSON, _ := json.Marshal(req)
	_, _ = pool.Exec(ctx, `
		INSERT INTO backtest_runs (
			external_run_id, instance_id, strategy_type_id, strategy_config_id, symbols, run_from, run_to,
			seed, dataset_id, status, stats, config_snapshot, started_at, completed_at, flow_id,
			data_source_type, source_provider, dataset_hash, is_synthetic, provenance_verified_at
		) VALUES (
			$1, NULLIF($2,'')::uuid, $3, $4, $5, $6, $7,
			$8, $9, 'completed', $10::jsonb, $11::jsonb, NOW(), NOW(), $12,
			'real', 'research.backtest', NULLIF($13,''), FALSE, NOW()
		)
		ON CONFLICT (external_run_id)
		DO UPDATE SET
			stats = EXCLUDED.stats,
			completed_at = NOW(),
			data_source_type = EXCLUDED.data_source_type,
			source_provider = EXCLUDED.source_provider,
			dataset_hash = EXCLUDED.dataset_hash,
			is_synthetic = EXCLUDED.is_synthetic,
			provenance_verified_at = EXCLUDED.provenance_verified_at
	`, runID, req.InstanceID, req.StrategyID, req.StrategyConfigID, symbols, fromDate.UTC(), toDate.UTC(),
		req.Seed, req.DatasetID, string(statsJSON), string(cfgJSON), observability.FlowIDFromContext(ctx), toString(upstream["dataset_hash"]))

	runRowID := uuid.NewString()
	_ = pool.QueryRow(ctx, `
		INSERT INTO runs (
			run_type, status, flow_id, source, instance_id, summary, started_at, completed_at, backtest_run_id,
			data_source_type, source_provider, dataset_id, dataset_hash, is_synthetic, provenance_verified_at
		)
		VALUES ('backtest', 'completed', $1, $2, NULLIF($3,'')::uuid, $4::jsonb, NOW(), NOW(),
		        (SELECT id FROM backtest_runs WHERE external_run_id=$5), 'real', 'research.backtest', NULLIF($6,''), NULLIF($7,''), FALSE, NOW())
		RETURNING id::text
	`, observability.FlowIDFromContext(ctx), source, req.InstanceID, string(statsJSON), runID, req.DatasetID, toString(upstream["dataset_hash"])).Scan(&runRowID)
	if err := persistDatasetSnapshotAndLinks(ctx, pool, req.DatasetID, toString(upstream["dataset_hash"]), upstream, runRowID, runID); err != nil {
		observability.LogEvent(ctx, "warn", "dataset.snapshot_link_failed", map[string]any{
			"dataset_id": req.DatasetID,
			"run_id":     runID,
			"error":      err.Error(),
		})
	}

	return map[string]any{
		"runId":       runID,
		"status":      "completed",
		"summary":     upstream,
		"parentRunId": runRowID,
	}, nil
}

type trustGateCommand struct {
	Name string
	Args []string
}

func runTrustGateJob(ctx context.Context, pool *pgxpool.Pool, gate, testType string) (string, map[string]any) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	commands := trustGateCommands(testType)
	results := make([]map[string]any, 0, len(commands))
	overall := "passed"
	for _, cmdSpec := range commands {
		start := time.Now().UTC()
		cmd := exec.CommandContext(timeoutCtx, cmdSpec.Name, cmdSpec.Args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		duration := time.Since(start).Milliseconds()
		exitCode := 0
		status := "passed"
		if err != nil {
			status = "failed"
			overall = "failed"
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				exitCode = ee.ExitCode()
			} else {
				exitCode = -1
			}
		}
		output := truncateForArtifact(out.String(), 24_000)
		results = append(results, map[string]any{
			"command":    cmdSpec.Name + " " + strings.Join(cmdSpec.Args, " "),
			"status":     status,
			"exitCode":   exitCode,
			"durationMs": duration,
			"output":     output,
		})
	}
	if testType == "data_recon" {
		recon := datasetReconciliationSummary(ctx, pool)
		reconStatus := "passed"
		exitCode := 0
		if toString(recon["status"]) == "failed" {
			reconStatus = "failed"
			overall = "failed"
			exitCode = 1
		}
		results = append(results, map[string]any{
			"command":    "dataset-hash-reconciliation",
			"status":     reconStatus,
			"exitCode":   exitCode,
			"durationMs": 0,
			"output":     toJSONString(recon),
			"summary":    recon,
		})
	}
	summary := map[string]any{
		"gate":      gate,
		"testType":  testType,
		"status":    overall,
		"commands":  results,
		"startedAt": time.Now().UTC().Format(time.RFC3339),
	}
	artifactPath := writeTestingArtifactReport(pool, gate, testType, summary)
	return artifactPath, summary
}

func trustGateCommands(testType string) []trustGateCommand {
	switch testType {
	case "data_recon":
		return []trustGateCommand{
			{Name: "go", Args: []string{"test", "./tests/replay/..."}},
			{Name: "go", Args: []string{"test", "-tags=golden", "./tests/golden/..."}},
		}
	case "pnl_recon":
		return []trustGateCommand{
			{Name: "go", Args: []string{"test", "./internal/modules/execution/..."}},
			{Name: "go", Args: []string{"test", "./libs/risk/..."}},
		}
	case "failure_suite":
		return []trustGateCommand{
			{Name: "go", Args: []string{"test", "./internal/modules/execution/..."}},
			{Name: "go", Args: []string{"test", "./internal/modules/orchestration/..."}},
		}
	case "flatten_proof":
		return []trustGateCommand{
			{Name: "go", Args: []string{"test", "./internal/modules/execution/...", "-run", "Flatten|Close|EOD"}},
		}
	default:
		return []trustGateCommand{
			{Name: "go", Args: []string{"test", "./cmd/trader/..."}},
		}
	}
}

func writeTestingArtifactReport(pool *pgxpool.Pool, gate, testType string, summary map[string]any) string {
	datePath := time.Now().UTC().Format("2006-01-02")
	artifactDir := testingArtifactDir(testType)
	dir := filepath.Join("reports", artifactDir, datePath)
	_ = os.MkdirAll(dir, 0o755)
	file := filepath.Join(dir, testingPrimaryArtifactFile(testType))
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("# %s\n\n", testType))
	buf.WriteString(fmt.Sprintf("- gate: %s\n", gate))
	buf.WriteString(fmt.Sprintf("- status: %s\n", toString(summary["status"])))
	buf.WriteString(fmt.Sprintf("- ts: %s\n\n", time.Now().UTC().Format(time.RFC3339)))
	commands, _ := summary["commands"].([]map[string]any)
	for i, cmd := range commands {
		buf.WriteString(fmt.Sprintf("## Command %d\n\n", i+1))
		buf.WriteString(fmt.Sprintf("- command: `%s`\n", toString(cmd["command"])))
		buf.WriteString(fmt.Sprintf("- status: %s\n", toString(cmd["status"])))
		buf.WriteString(fmt.Sprintf("- exitCode: %s\n", toString(cmd["exitCode"])))
		buf.WriteString(fmt.Sprintf("- durationMs: %s\n\n", toString(cmd["durationMs"])))
		buf.WriteString("```text\n")
		buf.WriteString(toString(cmd["output"]))
		buf.WriteString("\n```\n\n")
	}
	_ = os.WriteFile(file, []byte(buf.String()), 0o644)
	jsonFile := filepath.Join(dir, fmt.Sprintf("%s.json", testType))
	if raw, err := json.MarshalIndent(summary, "", "  "); err == nil {
		_ = os.WriteFile(jsonFile, raw, 0o644)
	}
	switch testType {
	case "data_recon":
		writeDataReconCSV(pool, dir)
		_ = os.WriteFile(filepath.Join(dir, "summary.md"), []byte(buf.String()), 0o644)
	case "pnl_recon":
		writePnLReconFiles(pool, dir, summary)
	case "failure_suite":
		_ = os.WriteFile(filepath.Join(dir, "report.md"), []byte(buf.String()), 0o644)
	case "flatten_proof":
		writeFlattenProof(pool, dir, summary)
	}
	return file
}

func testingArtifactDir(testType string) string {
	switch testType {
	case "data_recon":
		return "data_recon"
	case "pnl_recon":
		return "pnl_recon"
	case "failure_suite":
		return "failure_tests"
	case "flatten_proof":
		return "flatten"
	default:
		return "testing"
	}
}

func testingPrimaryArtifactFile(testType string) string {
	switch testType {
	case "data_recon":
		return "summary.md"
	case "pnl_recon":
		return "pnl_recon.md"
	case "failure_suite":
		return "report.md"
	case "flatten_proof":
		return "proof.md"
	default:
		return fmt.Sprintf("%s.md", testType)
	}
}

func writeDataReconCSV(pool *pgxpool.Pool, dir string) {
	file := filepath.Join(dir, "recon.csv")
	if pool == nil {
		_ = os.WriteFile(file, []byte("symbol,day,bars,error\n,,,database pool unavailable\n"), 0o644)
		return
	}
	rows, err := pool.Query(context.Background(), `
		SELECT symbol,
		       DATE_TRUNC('day', timestamp) AS day,
		       COUNT(*) AS bars
		FROM candles
		WHERE timestamp >= NOW() - INTERVAL '30 days'
		GROUP BY symbol, DATE_TRUNC('day', timestamp)
		ORDER BY day DESC, symbol
		LIMIT 2000
	`)
	if err != nil {
		_ = os.WriteFile(file, []byte("symbol,day,bars,error\n,,,"+sanitizeCSV(err.Error())+"\n"), 0o644)
		return
	}
	defer rows.Close()
	var out strings.Builder
	out.WriteString("symbol,day,bars\n")
	for rows.Next() {
		var symbol string
		var day time.Time
		var bars int
		if err := rows.Scan(&symbol, &day, &bars); err != nil {
			continue
		}
		out.WriteString(fmt.Sprintf("%s,%s,%d\n", sanitizeCSV(symbol), day.Format("2006-01-02"), bars))
	}
	_ = os.WriteFile(file, []byte(out.String()), 0o644)

	datasetFile := filepath.Join(dir, "dataset_recon.csv")
	if pool == nil {
		_ = os.WriteFile(datasetFile, []byte("run_id,dataset_id,run_hash,snapshot_hash,status,error\n,,,,,database pool unavailable\n"), 0o644)
		return
	}
	dsRows, err := pool.Query(context.Background(), `
		SELECT
			COALESCE(b.external_run_id, b.id::text) AS run_id,
			COALESCE(b.dataset_id,'') AS dataset_id,
			COALESCE(b.dataset_hash,'') AS run_hash,
			COALESCE(s.dataset_hash,'') AS snapshot_hash
		FROM backtest_runs b
		LEFT JOIN dataset_snapshots s ON s.dataset_id = b.dataset_id
		WHERE b.created_at >= NOW() - INTERVAL '30 days'
		ORDER BY b.created_at DESC
		LIMIT 2000
	`)
	if err != nil {
		_ = os.WriteFile(datasetFile, []byte("run_id,dataset_id,run_hash,snapshot_hash,status,error\n,,,,,"+sanitizeCSV(err.Error())+"\n"), 0o644)
		return
	}
	defer dsRows.Close()
	var dsOut strings.Builder
	dsOut.WriteString("run_id,dataset_id,run_hash,snapshot_hash,status\n")
	for dsRows.Next() {
		var runID, datasetID, runHash, snapHash string
		if err := dsRows.Scan(&runID, &datasetID, &runHash, &snapHash); err != nil {
			continue
		}
		status := "ok"
		switch {
		case strings.TrimSpace(datasetID) == "":
			status = "missing_dataset_id"
		case strings.TrimSpace(snapHash) == "":
			status = "missing_snapshot"
		case strings.TrimSpace(runHash) == "":
			status = "missing_run_hash"
		case runHash != snapHash:
			status = "hash_mismatch"
		}
		dsOut.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
			sanitizeCSV(runID), sanitizeCSV(datasetID), sanitizeCSV(runHash), sanitizeCSV(snapHash), sanitizeCSV(status)))
	}
	_ = os.WriteFile(datasetFile, []byte(dsOut.String()), 0o644)
}

func datasetReconciliationSummary(ctx context.Context, pool *pgxpool.Pool) map[string]any {
	result := map[string]any{
		"status":             "passed",
		"checkedRuns":        0,
		"missingDatasetID":   0,
		"missingSnapshot":    0,
		"missingRunHash":     0,
		"hashMismatch":       0,
		"windowDays":         30,
		"checkedAt":          time.Now().UTC().Format(time.RFC3339),
		"blockingViolations": 0,
	}
	if pool == nil {
		result["status"] = "failed"
		result["error"] = "database pool unavailable"
		result["blockingViolations"] = 1
		return result
	}
	rows, err := pool.Query(ctx, `
		SELECT
			COALESCE(b.dataset_id,'') AS dataset_id,
			COALESCE(b.dataset_hash,'') AS run_hash,
			COALESCE(s.dataset_hash,'') AS snapshot_hash
		FROM backtest_runs b
		LEFT JOIN dataset_snapshots s ON s.dataset_id = b.dataset_id
		WHERE b.created_at >= NOW() - INTERVAL '30 days'
	`)
	if err != nil {
		result["status"] = "failed"
		result["error"] = err.Error()
		result["blockingViolations"] = 1
		return result
	}
	defer rows.Close()

	checked := 0
	missingDatasetID := 0
	missingSnapshot := 0
	missingRunHash := 0
	hashMismatch := 0
	for rows.Next() {
		checked++
		var datasetID, runHash, snapHash string
		if err := rows.Scan(&datasetID, &runHash, &snapHash); err != nil {
			continue
		}
		switch {
		case strings.TrimSpace(datasetID) == "":
			missingDatasetID++
		case strings.TrimSpace(snapHash) == "":
			missingSnapshot++
		case strings.TrimSpace(runHash) == "":
			missingRunHash++
		case runHash != snapHash:
			hashMismatch++
		}
	}
	blocking := missingDatasetID + missingSnapshot + missingRunHash + hashMismatch
	result["checkedRuns"] = checked
	result["missingDatasetID"] = missingDatasetID
	result["missingSnapshot"] = missingSnapshot
	result["missingRunHash"] = missingRunHash
	result["hashMismatch"] = hashMismatch
	result["blockingViolations"] = blocking
	if blocking > 0 {
		result["status"] = "failed"
	}
	return result
}

func writePnLReconFiles(pool *pgxpool.Pool, dir string, summary map[string]any) {
	csvFile := filepath.Join(dir, "fills.csv")
	if pool == nil {
		_ = os.WriteFile(csvFile, []byte("trade_id,symbol,side,qty,price,status,error\n,,,,,,database pool unavailable\n"), 0o644)
		var md strings.Builder
		md.WriteString("# pnl_recon\n\n")
		md.WriteString(fmt.Sprintf("- generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
		md.WriteString("- rows: 0\n")
		md.WriteString("- status: failed\n")
		md.WriteString("- reason: database pool unavailable\n")
		_ = os.WriteFile(filepath.Join(dir, "pnl_recon.md"), []byte(md.String()), 0o644)
		summary["status"] = "failed"
		return
	}
	rows, err := pool.Query(context.Background(), `
		SELECT COALESCE(f.trade_id, t.id) AS trade_id,
		       COALESCE(f.symbol, t.symbol) AS symbol,
		       COALESCE(f.side, t.side) AS side,
		       COALESCE(f.filled_qty, t.quantity) AS qty,
		       COALESCE(f.avg_fill_price, t.entry_price) AS price,
		       COALESCE(f.status, t.status, 'unknown') AS status
		FROM trades t
		LEFT JOIN fills f ON f.trade_id = t.id
		ORDER BY t.created_at DESC
		LIMIT 2000
	`)
	if err != nil {
		_ = os.WriteFile(csvFile, []byte("trade_id,symbol,side,qty,price,status,error\n,,,,,,"+sanitizeCSV(err.Error())+"\n"), 0o644)
		return
	}
	defer rows.Close()
	var out strings.Builder
	out.WriteString("trade_id,symbol,side,qty,price,status\n")
	count := 0
	for rows.Next() {
		var tradeID, symbol, side, status string
		var qty, price float64
		if err := rows.Scan(&tradeID, &symbol, &side, &qty, &price, &status); err != nil {
			continue
		}
		out.WriteString(fmt.Sprintf("%s,%s,%s,%.4f,%.6f,%s\n",
			sanitizeCSV(tradeID), sanitizeCSV(symbol), sanitizeCSV(side), qty, price, sanitizeCSV(status)))
		count++
	}
	_ = os.WriteFile(csvFile, []byte(out.String()), 0o644)

	var md strings.Builder
	md.WriteString("# pnl_recon\n\n")
	md.WriteString(fmt.Sprintf("- generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	md.WriteString(fmt.Sprintf("- rows: %d\n", count))
	md.WriteString(fmt.Sprintf("- status: %s\n", toString(summary["status"])))
	_ = os.WriteFile(filepath.Join(dir, "pnl_recon.md"), []byte(md.String()), 0o644)
}

func writeFlattenProof(pool *pgxpool.Pool, dir string, summary map[string]any) {
	if pool == nil {
		summary["status"] = "failed"
		content := fmt.Sprintf("# flatten proof\n\n- generated: %s\n- status: failed\n- reason: database pool unavailable\n",
			time.Now().UTC().Format(time.RFC3339))
		_ = os.WriteFile(filepath.Join(dir, "proof.md"), []byte(content), 0o644)
		return
	}
	var breaches int
	_ = pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM trades t
		JOIN strategy_instances si ON si.id = t.instance_id
		WHERE to_char((t.created_at AT TIME ZONE si.session_timezone), 'HH24:MI') > si.flatten_by_close_time
	`).Scan(&breaches)
	status := "passed"
	if breaches > 0 {
		status = "failed"
		summary["status"] = "failed"
	}
	content := fmt.Sprintf("# flatten proof\n\n- generated: %s\n- status: %s\n- trades_after_flatten: %d\n",
		time.Now().UTC().Format(time.RFC3339), status, breaches)
	_ = os.WriteFile(filepath.Join(dir, "proof.md"), []byte(content), 0o644)
}

func sanitizeCSV(raw string) string {
	raw = strings.ReplaceAll(raw, "\n", " ")
	raw = strings.ReplaceAll(raw, "\r", " ")
	raw = strings.ReplaceAll(raw, ",", " ")
	return strings.TrimSpace(raw)
}

func parseOptionalDate(v any) *time.Time {
	raw := strings.TrimSpace(toString(v))
	if raw == "" {
		return nil
	}
	t := parseDateOnly(raw, time.Now().UTC())
	return &t
}

func nullableTimeString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}

func toSliceAny(v any) []any {
	switch typed := v.(type) {
	case []any:
		return typed
	default:
		return nil
	}
}

func intFromAny(v any) int {
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case json.Number:
		n, err := typed.Int64()
		if err == nil {
			return int(n)
		}
		return 0
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return n
		}
		return 0
	default:
		return 0
	}
}

func extractRunnerLeg(v any) (string, map[string]any) {
	row, ok := v.(map[string]any)
	if !ok {
		return "", nil
	}
	runID := toString(row["runId"])
	summary, _ := row["summary"].(map[string]any)
	return runID, summary
}

func toString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func toFloat64(v any) (float64, bool) {
	switch typed := v.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case json.Number:
		f, err := typed.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func toInt64(v any) (int64, bool) {
	switch typed := v.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func toStringSlice(v any) []string {
	switch typed := v.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, s := range typed {
			if ss := strings.ToUpper(strings.TrimSpace(s)); ss != "" {
				out = append(out, ss)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, raw := range typed {
			if ss := strings.ToUpper(strings.TrimSpace(toString(raw))); ss != "" {
				out = append(out, ss)
			}
		}
		return out
	case string:
		parts := strings.Split(typed, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if ss := strings.ToUpper(strings.TrimSpace(p)); ss != "" {
				out = append(out, ss)
			}
		}
		return out
	default:
		return nil
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func truncateForArtifact(raw string, maxChars int) string {
	if len(raw) <= maxChars || maxChars <= 0 {
		return raw
	}
	return raw[:maxChars] + "\n... (truncated) ..."
}

func parseDateOnly(raw string, fallback time.Time) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback.UTC()
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t.UTC()
	}
	return fallback.UTC()
}

func hashConfig(raw json.RawMessage) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func validateStrategyInstance(reg *strategytypes.Registry, strategyTypeID string, config json.RawMessage) error {
	if reg == nil {
		return nil
	}
	strategyTypeID = strings.TrimSpace(strategyTypeID)
	if strategyTypeID == "" || strings.EqualFold(strategyTypeID, "legacy") {
		return nil
	}
	st, ok := reg.Get(strategyTypeID)
	if !ok {
		return fmt.Errorf("unknown strategyTypeId: %s", strategyTypeID)
	}
	params, err := parseStrategyParams(config)
	if err != nil {
		return err
	}
	return st.Validate(params)
}

func parseStrategyParams(config json.RawMessage) (map[string]any, error) {
	if len(config) == 0 {
		return map[string]any{}, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(config, &raw); err != nil {
		return nil, fmt.Errorf("invalid configJson: %w", err)
	}
	if params, ok := raw["parameters"].(map[string]any); ok {
		return params, nil
	}
	return raw, nil
}

func loadInstanceBacktestConfig(ctx context.Context, pool *pgxpool.Pool, instanceID string) (map[string]any, string, string, error) {
	var cfgRaw string
	var tz string
	var flatten string
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(config::text, '{}'), COALESCE(session_timezone, ''), COALESCE(flatten_by_close_time, '')
		FROM strategy_instances
		WHERE id = $1::uuid
	`, instanceID).Scan(&cfgRaw, &tz, &flatten)
	if err != nil {
		return nil, "", "", err
	}
	params, err := parseStrategyParams(json.RawMessage(cfgRaw))
	if err != nil {
		return nil, tz, flatten, err
	}
	if tz == "" {
		tz = "America/New_York"
	}
	if flatten == "" {
		flatten = "15:55"
	}
	return params, tz, flatten, nil
}

func strOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func projectBaseInstanceID(ctx context.Context, pool *pgxpool.Pool, projectID string) string {
	var instanceID string
	_ = pool.QueryRow(ctx, `SELECT COALESCE(base_instance_id::text,'') FROM research_projects WHERE id = $1::uuid`, projectID).Scan(&instanceID)
	return instanceID
}

func instanceStrategyID(ctx context.Context, pool *pgxpool.Pool, instanceID string) string {
	var strategyID string
	_ = pool.QueryRow(ctx, `
		SELECT COALESCE(strategy_id, strategy_type_id, 'rsi_momentum_v1')
		FROM strategy_instances
		WHERE id = $1::uuid
	`, instanceID).Scan(&strategyID)
	if strategyID == "" {
		return "rsi_momentum_v1"
	}
	return strategyID
}

func instanceStrategyTypeID(ctx context.Context, pool *pgxpool.Pool, instanceID string) string {
	var strategyTypeID string
	_ = pool.QueryRow(ctx, `
		SELECT COALESCE(strategy_type_id, '')
		FROM strategy_instances
		WHERE id = $1::uuid
	`, instanceID).Scan(&strategyTypeID)
	return strategyTypeID
}

func runFlowID(ctx context.Context, pool *pgxpool.Pool, runID string) string {
	var flowID string
	_ = pool.QueryRow(ctx, `SELECT COALESCE(flow_id,'') FROM runs WHERE id = $1::uuid`, runID).Scan(&flowID)
	return flowID
}

func toJSONString(v any) string {
	if v == nil {
		return "{}"
	}
	b, _ := json.Marshal(v)
	if len(b) == 0 {
		return "{}"
	}
	return string(b)
}

func nullableFloat(v sql.NullFloat64) any {
	if !v.Valid {
		return nil
	}
	return v.Float64
}
