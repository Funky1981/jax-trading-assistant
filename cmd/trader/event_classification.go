package main

import (
	"strings"
)

type eventClassificationInput struct {
	Kind       string
	Title      string
	Summary    string
	Severity   string
	Symbols    []string
	Attributes map[string]any
}

type eventClassification struct {
	Class       string   `json:"class"`
	Impact      string   `json:"impact"`
	Sentiment   string   `json:"sentiment"`
	Horizon     string   `json:"horizon"`
	Tags        []string `json:"tags"`
	Explanation string   `json:"explanation"`
}

func classifyEvent(in eventClassificationInput) eventClassification {
	text := strings.ToLower(strings.TrimSpace(in.Title + " " + in.Summary))
	kind := strings.ToLower(strings.TrimSpace(in.Kind))
	severity := strings.ToLower(strings.TrimSpace(in.Severity))

	class := "general"
	switch kind {
	case "earnings":
		class = "corporate_earnings"
	case "macro":
		class = "macro_event"
	case "news":
		class = "company_news"
	}

	tags := make([]string, 0, 8)
	addTag := func(tag string) {
		for _, existing := range tags {
			if existing == tag {
				return
			}
		}
		tags = append(tags, tag)
	}
	addTag(kindOrDefault(kind))

	if hasAny(text, "upgrade", "outperform", "overweight") {
		addTag("analyst_upgrade")
	}
	if hasAny(text, "downgrade", "underperform", "underweight") {
		addTag("analyst_downgrade")
	}
	if hasAny(text, "guidance", "forecast", "outlook") {
		addTag("guidance")
	}
	if hasAny(text, "merger", "acquisition", "takeover", "buyout") {
		addTag("mna")
	}
	if hasAny(text, "lawsuit", "investigation", "sec", "probe", "penalty") {
		addTag("regulatory_risk")
	}
	if hasAny(text, "dividend", "buyback", "repurchase") {
		addTag("capital_return")
	}
	if hasAny(text, "fda", "approval", "approval granted") {
		addTag("approval")
	}

	impact := "low"
	switch {
	case severity == "high":
		impact = "high"
	case severity == "medium":
		impact = "medium"
	case kind == "earnings" || kind == "macro":
		impact = "medium"
	}
	if hasAny(text, "breaking", "guidance", "merger", "acquisition", "fda", "sec") {
		impact = "high"
	}

	bullishScore := countAny(text, "beat", "beats", "strong", "growth", "raised", "raise", "upgrade", "approval", "buyback", "profit")
	bearishScore := countAny(text, "miss", "missed", "weak", "cut", "lowered", "downgrade", "lawsuit", "investigation", "loss", "warn")
	sentiment := "neutral"
	switch {
	case bullishScore > bearishScore:
		sentiment = "bullish"
	case bearishScore > bullishScore:
		sentiment = "bearish"
	}

	horizon := "intraday"
	if kind == "macro" || hasAny(text, "guidance", "merger", "acquisition", "investigation") {
		horizon = "swing"
	}
	if kind == "earnings" && impact == "high" {
		horizon = "intraday_to_swing"
	}

	reasons := make([]string, 0, 4)
	reasons = append(reasons, "kind="+kindOrDefault(kind))
	reasons = append(reasons, "impact="+impact)
	reasons = append(reasons, "sentiment="+sentiment)
	if len(tags) > 1 {
		reasons = append(reasons, "tags="+strings.Join(tags[1:], ","))
	}

	return eventClassification{
		Class:       class,
		Impact:      impact,
		Sentiment:   sentiment,
		Horizon:     horizon,
		Tags:        tags,
		Explanation: strings.Join(reasons, "; "),
	}
}

func hasAny(text string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func countAny(text string, words ...string) int {
	count := 0
	for _, word := range words {
		if strings.Contains(text, word) {
			count++
		}
	}
	return count
}

func kindOrDefault(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return "unknown"
	}
	return kind
}
