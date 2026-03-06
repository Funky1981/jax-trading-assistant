package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/audit"
	"jax-trading-assistant/libs/auth"
	"jax-trading-assistant/libs/observability"

	"github.com/google/uuid"
)

type tradingPilotStatusResponse struct {
	PilotMode                        bool      `json:"pilotMode"`
	AuthRequired                     bool      `json:"authRequired"`
	OperatorRole                     string    `json:"operatorRole"`
	AllowedRoles                     []string  `json:"allowedRoles"`
	OperatorAccess                   bool      `json:"operatorAccess"`
	BrokerConnected                  bool      `json:"brokerConnected"`
	MarketDataMode                   string    `json:"marketDataMode"`
	PaperTrading                     bool      `json:"paperTrading"`
	ReadOnly                         bool      `json:"readOnly"`
	CanTrade                         bool      `json:"canTrade"`
	QuoteAuthority                   bool      `json:"quoteAuthority"`
	IntradayAuthority                bool      `json:"intradayAuthority"`
	ExecutionFromChartBlocked        bool      `json:"executionFromChartBlocked"`
	RequiresManualBrokerConfirmation bool      `json:"requiresManualBrokerConfirmation"`
	ReviewAgainstBroker              bool      `json:"reviewAgainstBroker"`
	RollbackToReadOnly               bool      `json:"rollbackToReadOnly"`
	Reasons                          []string  `json:"reasons"`
	Checklist                        []string  `json:"checklist"`
	CheckedAt                        time.Time `json:"checkedAt"`
}

func tradingPilotStatusHandler(authEnabled bool, mt *marketTools) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jsonOK(w, buildTradingPilotStatus(r.Context(), authEnabled, mt))
	}
}

func buildTradingPilotStatus(ctx context.Context, authEnabled bool, mt *marketTools) tradingPilotStatusResponse {
	allowedRoles := tradingPilotAllowedRoles()
	role := currentFrontendRole(ctx, authEnabled)
	operatorAccess := roleAllowed(role, allowedRoles)
	quoteAuthority := envBoolDefault("TRADING_PILOT_ALLOW_QUOTE_AUTHORITY", false)
	intradayAuthority := envBoolDefault("TRADING_PILOT_ALLOW_INTRADAY_AUTHORITY", false)

	status := tradingPilotStatusResponse{
		PilotMode:                        envBoolDefault("TRADING_PILOT_MODE", true),
		AuthRequired:                     authEnabled,
		OperatorRole:                     role,
		AllowedRoles:                     allowedRoles,
		OperatorAccess:                   operatorAccess,
		MarketDataMode:                   "unknown",
		PaperTrading:                     true,
		QuoteAuthority:                   quoteAuthority,
		IntradayAuthority:                intradayAuthority,
		ExecutionFromChartBlocked:        !intradayAuthority,
		RequiresManualBrokerConfirmation: true,
		ReviewAgainstBroker:              true,
		RollbackToReadOnly:               true,
		Checklist: []string{
			"Verify IB/TWS is connected and paper trading remains enabled.",
			"Verify the market-data badge before using prices on this screen.",
			"Confirm symbol, quantity, and exits in IB/TWS before every broker mutation.",
			"Review audit and broker events after each pilot session.",
		},
		CheckedAt: time.Now().UTC(),
	}

	if mt != nil {
		reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if bridgeHealth, err := mt.getIBBridgeHealth(reqCtx); err == nil && bridgeHealth != nil {
			status.BrokerConnected = bridgeHealth.Connected
			if strings.TrimSpace(bridgeHealth.MarketDataMode) != "" {
				status.MarketDataMode = strings.TrimSpace(bridgeHealth.MarketDataMode)
			}
			status.PaperTrading = bridgeHealth.PaperTrading
		}
	}

	if !status.PilotMode {
		status.Reasons = append(status.Reasons, "Pilot mode is disabled.")
	}
	if !status.AuthRequired {
		status.Reasons = append(status.Reasons, "Authentication is disabled; trading is forced into read-only pilot mode.")
	}
	if !status.OperatorAccess {
		status.Reasons = append(status.Reasons, fmt.Sprintf("Role %q is not in the allowed pilot operator list.", role))
	}
	if !status.BrokerConnected {
		status.Reasons = append(status.Reasons, "Broker bridge is not connected; trading actions are disabled.")
	}
	if !status.PaperTrading {
		status.Reasons = append(status.Reasons, "Paper trading is not enabled; pilot mode blocks live order submission.")
	}
	if !status.QuoteAuthority {
		status.Reasons = append(status.Reasons, "Quotes on this screen are non-authoritative during the pilot; confirm in IB/TWS.")
	}
	if !status.IntradayAuthority {
		status.Reasons = append(status.Reasons, "Intraday candles are non-authoritative during the pilot; use IB/TWS as the execution source of truth.")
	}

	status.CanTrade = status.PilotMode && status.AuthRequired && status.OperatorAccess && status.BrokerConnected && status.PaperTrading
	status.ReadOnly = !status.CanTrade
	return status
}

func envBoolDefault(key string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func tradingPilotAllowedRoles() []string {
	raw := strings.TrimSpace(os.Getenv("TRADING_PILOT_ALLOWED_ROLES"))
	if raw == "" {
		raw = "admin"
	}
	parts := strings.Split(raw, ",")
	roles := make([]string, 0, len(parts))
	for _, part := range parts {
		role := strings.ToLower(strings.TrimSpace(part))
		if role != "" {
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return []string{"admin"}
	}
	return roles
}

func currentFrontendRole(ctx context.Context, authEnabled bool) string {
	if role, ok := auth.RoleFromContext(ctx); ok && strings.TrimSpace(role) != "" {
		return strings.ToLower(strings.TrimSpace(role))
	}
	if authEnabled {
		return "unauthenticated"
	}
	return "anonymous"
}

func currentFrontendUsername(ctx context.Context, authEnabled bool) string {
	if username, ok := auth.UsernameFromContext(ctx); ok && strings.TrimSpace(username) != "" {
		return username
	}
	if authEnabled {
		return "unauthenticated"
	}
	return "anonymous"
}

func roleAllowed(role string, allowedRoles []string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	for _, allowedRole := range allowedRoles {
		if role == strings.ToLower(strings.TrimSpace(allowedRole)) {
			return true
		}
	}
	return false
}

func brokerOrdersHandler(authEnabled bool, mt *marketTools, auditSvc *audit.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			data, err := proxyBrokerRequest(r.Context(), mt, http.MethodGet, "/orders", nil)
			if err != nil {
				if writeProxyError(w, err) {
					return
				}
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			writeJSONProxyResponse(w, data)
		case http.MethodPost:
			body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if err != nil {
				http.Error(w, "read error", http.StatusBadRequest)
				return
			}
			if !allowPilotBrokerWrite(w, r, authEnabled, mt, auditSvc, "orders.submit", "", body) {
				return
			}
			data, err := proxyBrokerRequest(r.Context(), mt, http.MethodPost, "/orders", body)
			if err != nil {
				logPilotBrokerAction(r.Context(), auditSvc, "orders.submit", "error", "Broker order submit failed", map[string]any{
					"body":  string(body),
					"error": err.Error(),
				})
				if writeProxyError(w, err) {
					return
				}
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			logPilotBrokerAction(r.Context(), auditSvc, "orders.submit", "success", "Pilot broker order submitted", map[string]any{
				"body": string(body),
			})
			writeJSONProxyResponse(w, data)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func brokerOrderDetailHandler(authEnabled bool, mt *marketTools, auditSvc *audit.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := strings.TrimPrefix(r.URL.Path, "/api/v1/broker/orders/")
		orderID = strings.TrimSpace(orderID)
		if orderID == "" {
			http.Error(w, "order id required", http.StatusBadRequest)
			return
		}

		if orderID == "bracket" {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if err != nil {
				http.Error(w, "read error", http.StatusBadRequest)
				return
			}
			if !allowPilotBrokerWrite(w, r, authEnabled, mt, auditSvc, "orders.bracket_submit", "", body) {
				return
			}
			data, err := proxyBrokerRequest(r.Context(), mt, http.MethodPost, "/orders/bracket", body)
			if err != nil {
				logPilotBrokerAction(r.Context(), auditSvc, "orders.bracket_submit", "error", "Broker bracket order submit failed", map[string]any{
					"body":  string(body),
					"error": err.Error(),
				})
				if writeProxyError(w, err) {
					return
				}
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			logPilotBrokerAction(r.Context(), auditSvc, "orders.bracket_submit", "success", "Pilot bracket order submitted", map[string]any{
				"body": string(body),
			})
			writeJSONProxyResponse(w, data)
			return
		}

		switch r.Method {
		case http.MethodDelete:
			if !allowPilotBrokerWrite(w, r, authEnabled, mt, auditSvc, "orders.cancel", orderID, nil) {
				return
			}
			data, err := proxyBrokerRequest(r.Context(), mt, http.MethodDelete, "/orders/"+orderID, nil)
			if err != nil {
				logPilotBrokerAction(r.Context(), auditSvc, "orders.cancel", "error", "Broker order cancel failed", map[string]any{
					"order_id": orderID,
					"error":    err.Error(),
				})
				if writeProxyError(w, err) {
					return
				}
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			logPilotBrokerAction(r.Context(), auditSvc, "orders.cancel", "success", "Pilot broker order cancel submitted", map[string]any{
				"order_id": orderID,
			})
			writeJSONProxyResponse(w, data)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func brokerPositionsHandler(mt *marketTools) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err := proxyBrokerRequest(r.Context(), mt, http.MethodGet, "/positions", nil)
		if err != nil {
			if writeProxyError(w, err) {
				return
			}
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSONProxyResponse(w, data)
	}
}

func brokerPositionDetailHandler(authEnabled bool, mt *marketTools, auditSvc *audit.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/broker/positions/")
		path = strings.Trim(path, "/")
		parts := strings.Split(path, "/")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			http.Error(w, "position action path invalid", http.StatusBadRequest)
			return
		}
		symbol := strings.ToUpper(strings.TrimSpace(parts[0]))
		action := strings.ToLower(strings.TrimSpace(parts[1]))
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		auditAction := "positions." + action
		if !allowPilotBrokerWrite(w, r, authEnabled, mt, auditSvc, auditAction, symbol, body) {
			return
		}

		upstreamPath := fmt.Sprintf("/positions/%s/%s", symbol, action)
		data, err := proxyBrokerRequest(r.Context(), mt, http.MethodPost, upstreamPath, body)
		if err != nil {
			logPilotBrokerAction(r.Context(), auditSvc, auditAction, "error", "Pilot position action failed", map[string]any{
				"symbol": symbol,
				"body":   string(body),
				"error":  err.Error(),
			})
			if writeProxyError(w, err) {
				return
			}
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		logPilotBrokerAction(r.Context(), auditSvc, auditAction, "success", "Pilot position action submitted", map[string]any{
			"symbol": symbol,
			"body":   string(body),
		})
		writeJSONProxyResponse(w, data)
	}
}

func brokerAccountHandler(mt *marketTools) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err := proxyBrokerRequest(r.Context(), mt, http.MethodGet, "/account", nil)
		if err != nil {
			if writeProxyError(w, err) {
				return
			}
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSONProxyResponse(w, data)
	}
}

func allowPilotBrokerWrite(w http.ResponseWriter, r *http.Request, authEnabled bool, mt *marketTools, auditSvc *audit.Service, action string, target string, body []byte) bool {
	status := buildTradingPilotStatus(r.Context(), authEnabled, mt)
	if status.CanTrade {
		return true
	}

	httpStatus := http.StatusForbidden
	if !status.BrokerConnected || !status.PaperTrading {
		httpStatus = http.StatusServiceUnavailable
	}

	logPilotBrokerAction(r.Context(), auditSvc, action, "blocked", "Pilot broker action blocked", map[string]any{
		"target":  target,
		"body":    string(body),
		"reasons": status.Reasons,
	})
	jsonError(w, httpStatus, map[string]any{
		"error":    "trading pilot is in read-only mode",
		"reasons":  status.Reasons,
		"readOnly": status.ReadOnly,
		"canTrade": status.CanTrade,
	})
	return false
}

func proxyBrokerRequest(ctx context.Context, mt *marketTools, method string, path string, body []byte) ([]byte, error) {
	if mt == nil || strings.TrimSpace(mt.ibBridgeURL) == "" {
		return nil, fmt.Errorf("ib bridge unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(mt.ibBridgeURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if flowID := observability.FlowIDFromContext(ctx); flowID != "" {
		req.Header.Set("X-Flow-ID", flowID)
	}

	client := http.DefaultClient
	if mt.httpClient != nil {
		client = mt.httpClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, proxyError{Status: resp.StatusCode, Body: strings.TrimSpace(string(data))}
	}
	return data, nil
}

func logPilotBrokerAction(ctx context.Context, auditSvc *audit.Service, action string, outcome string, message string, metadata map[string]any) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["role"] = currentFrontendRole(ctx, true)
	metadata["username"] = currentFrontendUsername(ctx, true)

	correlationID := observability.FlowIDFromContext(ctx)
	if correlationID == "" {
		correlationID = uuid.NewString()
	}
	observability.LogEvent(ctx, "info", "trading.pilot_action", map[string]any{
		"action":         action,
		"outcome":        outcome,
		"message":        message,
		"correlation_id": correlationID,
		"metadata":       metadata,
	})
	if auditSvc == nil {
		return
	}
	if err := auditSvc.LogAuditEvent(ctx, correlationID, "trading_pilot", action, outcome, message, metadata); err != nil {
		observability.LogEvent(ctx, "warn", "trading.pilot_audit_failed", map[string]any{
			"action": action,
			"error":  err.Error(),
		})
	}
}

func writeJSONProxyResponse(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func jsonError(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
