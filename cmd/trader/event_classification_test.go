package main

import "testing"

func TestClassifyEvent_EarningsBeatIsBullish(t *testing.T) {
	got := classifyEvent(eventClassificationInput{
		Kind:     "earnings",
		Title:    "AAPL beats earnings and raises guidance",
		Summary:  "Strong growth and raised outlook",
		Severity: "high",
		Symbols:  []string{"AAPL"},
	})
	if got.Class != "corporate_earnings" {
		t.Fatalf("class = %q, want corporate_earnings", got.Class)
	}
	if got.Sentiment != "bullish" {
		t.Fatalf("sentiment = %q, want bullish", got.Sentiment)
	}
	if got.Impact != "high" {
		t.Fatalf("impact = %q, want high", got.Impact)
	}
}

func TestClassifyEvent_DowngradeLawsuitIsBearish(t *testing.T) {
	got := classifyEvent(eventClassificationInput{
		Kind:     "news",
		Title:    "TSLA downgraded after lawsuit filing",
		Summary:  "Analyst downgrade follows investigation and legal risk",
		Severity: "medium",
		Symbols:  []string{"TSLA"},
	})
	if got.Sentiment != "bearish" {
		t.Fatalf("sentiment = %q, want bearish", got.Sentiment)
	}
	if got.Impact == "low" {
		t.Fatalf("impact = %q, want medium/high", got.Impact)
	}
}

func TestClassifyEvent_MacroDefaultsToMacroClass(t *testing.T) {
	got := classifyEvent(eventClassificationInput{
		Kind:     "macro",
		Title:    "US CPI release",
		Summary:  "Inflation data due",
		Severity: "medium",
	})
	if got.Class != "macro_event" {
		t.Fatalf("class = %q, want macro_event", got.Class)
	}
	if got.Horizon != "swing" {
		t.Fatalf("horizon = %q, want swing", got.Horizon)
	}
}
