package observability

type Metrics struct {
	WarningCount int `json:"warning_count"`
	ErrorCount   int `json:"error_count"`
}

func NewMetrics(trace Trace) Metrics {
	return Metrics{
		WarningCount: len(trace.Warnings),
		ErrorCount:   len(trace.Errors),
	}
}
