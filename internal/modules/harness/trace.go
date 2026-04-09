package harness

import (
	"encoding/json"
	"time"
)

type Trace struct {
	TraceID            string              `json:"traceId"`
	SessionID          string              `json:"sessionId"`
	Question           string              `json:"question"`
	ToolNames          []string            `json:"toolNames"`
	ToolRuns           []ToolRun           `json:"toolRuns,omitempty"`
	ValidatorNotes     []string            `json:"validatorNotes"`
	ValidationAttempts []ValidationAttempt `json:"validationAttempts,omitempty"`
	FinalAnswer        string              `json:"finalAnswer,omitempty"`
	CreatedAt          time.Time           `json:"createdAt"`
}

type TraceSink interface {
	WriteTrace(t Trace) error
}

type ToolCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type ToolRun struct {
	Call   ToolCall        `json:"call"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type ValidationAttempt struct {
	Attempt  int      `json:"attempt"`
	Answer   string   `json:"answer"`
	Accepted bool     `json:"accepted"`
	Reasons  []string `json:"reasons,omitempty"`
}
