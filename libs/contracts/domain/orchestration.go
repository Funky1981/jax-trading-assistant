package domain

import "time"

// OrchestrationRun represents an AI orchestration execution
type OrchestrationRun struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Query       string                 `json:"query"`
	Status      string                 `json:"status"` // "pending", "running", "completed", "failed"
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Response    string                 `json:"response"`
	Metadata    map[string]interface{} `json:"metadata"`
	Providers   []ProviderUsage        `json:"providers"`
}

// ProviderUsage tracks which AI providers were used
type ProviderUsage struct {
	Name     string  `json:"name"`
	Duration float64 `json:"duration_ms"`
	Success  bool    `json:"success"`
	ErrorMsg string  `json:"error_msg,omitempty"`
}
