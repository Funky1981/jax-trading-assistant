package swingresearch

import "strings"

const (
	StatusBlocked = "blocked"
	StatusWatch   = "watch"
	StatusThesis  = "thesis"
)

type EvidenceSource struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Input struct {
	Symbol                   string
	Headline                 string
	EventSource              EvidenceSource
	DailyCandles             int
	Confounders              []string
	MappedETFs               []string
	HistoricalReactionWindow string
}

type Output struct {
	Status                   string   `json:"status"`
	ThesisSummary            string   `json:"thesisSummary,omitempty"`
	MappedETFs               []string `json:"mappedEtfs,omitempty"`
	EvidenceIDs              []string `json:"evidenceIds,omitempty"`
	SourceURLs               []string `json:"sourceUrls,omitempty"`
	HistoricalReactionWindow string   `json:"historicalReactionWindow,omitempty"`
	Invalidators             []string `json:"invalidators,omitempty"`
	DailyReviewSchedule      string   `json:"dailyReviewSchedule,omitempty"`
	RiskNotes                []string `json:"riskNotes,omitempty"`
	BlockerReasons           []string `json:"blockerReasons,omitempty"`
	OrderInstruction         string   `json:"orderInstruction,omitempty"`
}

type Engine struct{}

func NewEngine() Engine {
	return Engine{}
}

func (Engine) Evaluate(in Input) Output {
	out := Output{
		MappedETFs:               nonEmpty(in.MappedETFs),
		HistoricalReactionWindow: strings.TrimSpace(in.HistoricalReactionWindow),
		Invalidators: []string{
			"event_thesis_invalidated",
			"confounder_detected",
			"daily_close_breaks_risk_level",
		},
		DailyReviewSchedule: "daily_after_close",
		RiskNotes: []string{
			"paper_only",
			"human_approval_required",
			"max_hold_days_10",
		},
	}
	if strings.TrimSpace(in.EventSource.ID) == "" || strings.TrimSpace(in.EventSource.URL) == "" {
		out.Status = StatusBlocked
		out.BlockerReasons = append(out.BlockerReasons, "missing_event_source")
	}
	if in.DailyCandles < 20 {
		out.Status = StatusBlocked
		out.BlockerReasons = append(out.BlockerReasons, "missing_daily_candles")
	}
	if len(out.BlockerReasons) > 0 {
		return out
	}

	out.EvidenceIDs = []string{in.EventSource.ID}
	out.SourceURLs = []string{in.EventSource.URL}
	if len(in.Confounders) > 0 {
		out.Status = StatusWatch
		out.BlockerReasons = append(out.BlockerReasons, "confounder_present")
		out.RiskNotes = append(out.RiskNotes, in.Confounders...)
		return out
	}

	out.Status = StatusThesis
	symbol := strings.TrimSpace(in.Symbol)
	if symbol == "" {
		symbol = "ETF"
	}
	headline := strings.TrimSpace(in.Headline)
	if headline == "" {
		headline = "validated event evidence"
	}
	if len(out.MappedETFs) == 0 {
		out.MappedETFs = []string{symbol}
	}
	if out.HistoricalReactionWindow == "" {
		out.HistoricalReactionWindow = "2-5 trading days"
	}
	out.ThesisSummary = symbol + " swing thesis from " + headline
	return out
}

func nonEmpty(items []string) []string {
	out := []string{}
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
