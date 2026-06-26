package app

import (
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
)

type requestMetadata struct {
	requestID        string
	createdAt        time.Time
	attemptedActions []string
}

func (s ReviewOperationsService) baseResult(operation string, metadata requestMetadata, readOnly bool) OperationResult {
	requestID := metadata.requestID
	if requestID == "" {
		requestID = s.config.DefaultRequestID
	}
	createdAt := metadata.createdAt
	if createdAt.IsZero() {
		createdAt = s.config.DefaultCreatedAt
	}
	result := OperationResult{
		RequestID:             requestID,
		Operation:             operation,
		Succeeded:             true,
		ForbiddenActions:      append([]string{}, s.config.ForbiddenActions...),
		RequiresHumanApproval: true,
		AutoApplyAllowed:      false,
		ReadOnly:              readOnly,
		CreatedAt:             createdAt,
	}
	result.ValidationErrors = append(result.ValidationErrors, validateAttemptedActions(metadata.attemptedActions)...)
	result.Succeeded = len(result.ValidationErrors) == 0
	return result
}

func (r OperationResult) withValidation(errors []string, warnings []string) OperationResult {
	r.ValidationErrors = append(r.ValidationErrors, errors...)
	r.ValidationWarnings = append(r.ValidationWarnings, warnings...)
	r.Succeeded = len(r.ValidationErrors) == 0
	r.AutoApplyAllowed = false
	r.RequiresHumanApproval = true
	r.ForbiddenActions = mergeForbiddenActions(r.ForbiddenActions)
	return r
}

func validateAttemptedActions(actions []string) []string {
	var errors []string
	for _, action := range actions {
		normalized := strings.ToLower(strings.TrimSpace(action))
		switch {
		case normalized == "":
			continue
		case isForbiddenAction(normalized):
			errors = append(errors, fmt.Sprintf("forbidden action %q is blocked", action))
		case containsLiveOrExecutionRequest(normalized):
			errors = append(errors, fmt.Sprintf("live, broker, order, or execution request %q is blocked", action))
		}
	}
	return errors
}

func isForbiddenAction(action string) bool {
	for _, forbidden := range feedback.ForbiddenActions() {
		if strings.EqualFold(action, forbidden) {
			return true
		}
	}
	return false
}

func containsLiveOrExecutionRequest(value string) bool {
	normalized := strings.ToLower(value)
	blocked := []string{
		"live_ready",
		"live approval",
		"live_order",
		"create_live_order",
		"execute_trade",
		"execution",
		"broker",
		"auto_approve",
		"auto approval",
		"paper execution",
	}
	for _, marker := range blocked {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func mergeForbiddenActions(values []string) []string {
	seen := map[string]bool{}
	var merged []string
	for _, value := range append(append([]string{}, values...), feedback.ForbiddenActions()...) {
		normalized := strings.TrimSpace(value)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		merged = append(merged, normalized)
	}
	return merged
}
