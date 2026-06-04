package llmcontext

import (
	"fmt"
	"strings"
)

type ApprovalTemplateData struct {
	Symbol          string
	StrategyName    string
	PaperAction     string
	Confidence      string
	ModelReason     string
	PricedInVerdict string
	PricedInReason  string
	Entry           string
	StopLoss        string
	TakeProfit      string
	RiskAmount      string
	ExpiresAt       string
}

func RenderApprovalSummary(data ApprovalTemplateData) string {
	return fmt.Sprintf(`ETF: %s
Strategy: %s
Action: %s
Confidence: %s

Why:
%s

Priced-in:
%s - %s

Risk:
Entry %s, stop %s, target %s, risk %s.

Expires:
%s

Decision:
Approve / Reject / Snooze`,
		data.Symbol,
		data.StrategyName,
		data.PaperAction,
		data.Confidence,
		limitWords(data.ModelReason, 80),
		data.PricedInVerdict,
		limitWords(data.PricedInReason, 40),
		data.Entry,
		data.StopLoss,
		data.TakeProfit,
		data.RiskAmount,
		data.ExpiresAt,
	)
}

func CountWords(text string) int {
	return len(strings.Fields(text))
}

func limitWords(text string, limit int) string {
	words := strings.Fields(text)
	if len(words) <= limit {
		return strings.TrimSpace(text)
	}
	return strings.Join(words[:limit], " ")
}
