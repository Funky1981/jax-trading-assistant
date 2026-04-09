package harness

import "time"

type Trace struct {
    TraceID        string    `json:"traceId"`
    SessionID      string    `json:"sessionId"`
    Question       string    `json:"question"`
    ToolNames      []string  `json:"toolNames"`
    ValidatorNotes []string  `json:"validatorNotes"`
    CreatedAt      time.Time `json:"createdAt"`
}

type TraceSink interface {
    WriteTrace(t Trace) error
}
