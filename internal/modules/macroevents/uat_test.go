package macroevents

import (
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

type macroUATFixture struct {
	event              EventInput
	symbol             string
	candles            []marketdata.Candle
	preEventMove       float64
	newsSaturation     float64
	volatilityElevated bool
	confounders        []Confounder
	plan               CandidatePlan
	side               CandidateSide
}

func TestMacroReactionUATDeterministicFixtures(t *testing.T) {
	eventTime := time.Date(2026, 6, 10, 13, 30, 0, 0, time.UTC)
	longEntry, longStop, longTarget := 102.0, 100.8, 104.2
	shortEntry, shortStop, shortTarget := 98.0, 99.2, 95.8

	tests := []struct {
		name              string
		fixture           macroUATFixture
		wantDirection     ReactionDirection
		wantConfirms      bool
		wantEvidence      EvidenceVerdict
		wantCandidate     MacroCandidateStatus
		wantCandidateSide CandidateSide
		wantReason        string
	}{
		{
			name: "fixture hot CPI confirms bearish QQQ",
			fixture: macroUATFixture{
				event:   macroUATEvent(eventTime, EventTypeUSCPIHeadline, DirectionInflationHot, 5.1, 3.0, "Hot CPI maps to bearish QQQ"),
				symbol:  "QQQ",
				candles: macroUATCandles(eventTime, "QQQ", 100.0, 99.4, 98.8),
				plan:    macroUATPlan(&shortEntry, &shortStop, &shortTarget),
				side:    CandidateSideShortBias,
			},
			wantDirection:     ReactionDirectionDown,
			wantConfirms:      true,
			wantEvidence:      EvidenceVerdictCandidateAllowed,
			wantCandidate:     MacroCandidateStatusAwaitingHumanApproval,
			wantCandidateSide: CandidateSideShortBias,
		},
		{
			name: "fixture cool CPI confirms bullish QQQ",
			fixture: macroUATFixture{
				event:   macroUATEvent(eventTime, EventTypeUSCPIHeadline, DirectionInflationCool, 2.0, 4.0, "Cool CPI maps to bullish QQQ"),
				symbol:  "QQQ",
				candles: macroUATCandles(eventTime, "QQQ", 100.0, 100.6, 101.0),
				plan:    macroUATPlan(&longEntry, &longStop, &longTarget),
				side:    CandidateSideLong,
			},
			wantDirection:     ReactionDirectionUp,
			wantConfirms:      true,
			wantEvidence:      EvidenceVerdictCandidateAllowed,
			wantCandidate:     MacroCandidateStatusAwaitingHumanApproval,
			wantCandidateSide: CandidateSideLong,
		},
		{
			name: "fixture whipsaw Fed event rejects",
			fixture: macroUATFixture{
				event:  macroUATEvent(eventTime, EventTypeFOMCStatement, DirectionHawkishRates, 5.5, 5.25, "Fed statement whipsaws QQQ"),
				symbol: "QQQ",
				candles: []marketdata.Candle{
					macroUATCandle(eventTime.Add(-time.Minute), "QQQ", 100.0, 100.1, 99.9, 100.0),
					macroUATCandle(eventTime.Add(5*time.Minute), "QQQ", 100.0, 103.0, 99.8, 102.0),
					macroUATCandle(eventTime.Add(15*time.Minute), "QQQ", 102.0, 102.1, 97.0, 99.5),
				},
				plan: macroUATPlan(&shortEntry, &shortStop, &shortTarget),
				side: CandidateSideShortBias,
			},
			wantDirection:     ReactionDirectionWhipsaw,
			wantConfirms:      false,
			wantEvidence:      EvidenceVerdictCandidateBlocked,
			wantCandidate:     MacroCandidateStatusBlocked,
			wantCandidateSide: CandidateSideNoTrade,
			wantReason:        "chart reaction does not confirm scenario",
		},
		{
			name: "fixture already-priced-in event rejects",
			fixture: macroUATFixture{
				event:        macroUATEvent(eventTime, EventTypeUSCPIHeadline, DirectionInflationCool, 3.01, 3.0, "Small CPI surprise after large pre move"),
				symbol:       "QQQ",
				candles:      macroUATCandles(eventTime, "QQQ", 100.0, 100.6, 101.0),
				preEventMove: 0.02,
				plan:         macroUATPlan(&longEntry, &longStop, &longTarget),
				side:         CandidateSideLong,
			},
			wantDirection:     ReactionDirectionUp,
			wantConfirms:      true,
			wantEvidence:      EvidenceVerdictCandidateBlocked,
			wantCandidate:     MacroCandidateStatusBlocked,
			wantCandidateSide: CandidateSideNoTrade,
			wantReason:        "priced-in verdict blocks candidate",
		},
		{
			name: "fixture missing candles rejects",
			fixture: macroUATFixture{
				event:  macroUATEvent(eventTime, EventTypeUSNonfarmPayrolls, DirectionGrowthStrong, 350000, 200000, "Strong jobs but missing candles"),
				symbol: "IWM",
				candles: []marketdata.Candle{
					macroUATCandle(eventTime.Add(-time.Minute), "IWM", 200.0, 200.1, 199.9, 200.0),
				},
				plan: macroUATPlan(&shortEntry, &shortStop, &shortTarget),
				side: CandidateSideShortBias,
			},
			wantDirection:     ReactionDirectionUnknown,
			wantConfirms:      false,
			wantEvidence:      EvidenceVerdictInsufficientEvidence,
			wantCandidate:     MacroCandidateStatusBlocked,
			wantCandidateSide: CandidateSideNoTrade,
			wantReason:        "chart reaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := runMacroUATFixture(tt.fixture)
			if out.reaction.Direction != tt.wantDirection {
				t.Fatalf("reaction direction = %q, want %q; reason=%s", out.reaction.Direction, tt.wantDirection, out.reaction.Reason)
			}
			if out.reaction.ConfirmsEvent != tt.wantConfirms {
				t.Fatalf("confirms = %v, want %v; reason=%s", out.reaction.ConfirmsEvent, tt.wantConfirms, out.reaction.Reason)
			}
			if out.evidence.Verdict != tt.wantEvidence {
				t.Fatalf("evidence verdict = %q, want %q; missing=%v walkaway=%v", out.evidence.Verdict, tt.wantEvidence, out.evidence.MissingEvidence, out.evidence.WalkawayReasons)
			}
			if out.candidate.Status != tt.wantCandidate {
				t.Fatalf("candidate status = %q, want %q; rejection=%s", out.candidate.Status, tt.wantCandidate, out.candidate.RejectionReason)
			}
			if out.candidate.Side != tt.wantCandidateSide {
				t.Fatalf("candidate side = %q, want %q", out.candidate.Side, tt.wantCandidateSide)
			}
			if tt.wantReason != "" && !macroUATContains(out, tt.wantReason) {
				t.Fatalf("fixture output did not contain reason %q; evidence=%v candidate=%s", tt.wantReason, out.evidence.WalkawayReasons, out.candidate.RejectionReason)
			}
		})
	}
}

func TestMacroReactionUATCandidateCannotBecomeOrderWithoutSeparateApproval(t *testing.T) {
	eventTime := time.Date(2026, 6, 10, 13, 30, 0, 0, time.UTC)
	entry, stop, target := 102.0, 100.8, 104.2
	out := runMacroUATFixture(macroUATFixture{
		event:   macroUATEvent(eventTime, EventTypeUSCPIHeadline, DirectionInflationCool, 2.0, 4.0, "Cool CPI creates paper candidate only"),
		symbol:  "QQQ",
		candles: macroUATCandles(eventTime, "QQQ", 100.0, 100.6, 101.0),
		plan:    macroUATPlan(&entry, &stop, &target),
		side:    CandidateSideLong,
	})

	if out.candidate.Status != MacroCandidateStatusAwaitingHumanApproval {
		t.Fatalf("candidate status = %q, want awaiting human approval", out.candidate.Status)
	}
	if out.candidate.ID != "" {
		t.Fatalf("deterministic UAT should not persist or promote candidate, got id %q", out.candidate.ID)
	}
	if strings.Contains(strings.ToLower(out.candidate.CreatedReason), "order") {
		t.Fatalf("candidate reason should not imply order creation: %q", out.candidate.CreatedReason)
	}
}

type macroUATOutput struct {
	scenario  ScenarioEvaluation
	reaction  ReactionSnapshot
	pricedIn  PricedInScore
	evidence  EvidenceBundle
	candidate MacroCandidate
}

func runMacroUATFixture(f macroUATFixture) macroUATOutput {
	scenario := EvaluateScenario(f.event)
	reaction := EvaluateReaction(ReactionInput{
		MacroEventID: f.event.MacroEventID,
		Symbol:       f.symbol,
		Timeframe:    TimeframePostEvent15M,
		EventType:    f.event.EventType,
		Direction:    f.event.Direction,
		EventTimeUTC: f.event.EventTimeUTC,
		Candles:      f.candles,
	})
	pricedIn := ScorePricedIn(PricedInInput{
		MacroEventID:          f.event.MacroEventID,
		Symbol:                f.symbol,
		Event:                 f.event,
		PreEventMovePercent:   f.preEventMove,
		NewsSaturationScore:   f.newsSaturation,
		VolatilityElevated:    f.volatilityElevated,
		AnalystConsensusTight: false,
		Reaction:              reaction,
	})
	evidence := BuildEvidenceBundle(EvidenceInput{
		MacroEvent:           f.event,
		Scenario:             scenario,
		Reaction:             reaction,
		PricedIn:             pricedIn,
		Confounders:          f.confounders,
		HistoricalComparison: "deterministic fixture historical comparison",
		RiskGuardrail:        "paper-only risk guardrail passed",
		EntryStopTarget:      "fixture entry/stop/target proposal",
		Symbol:               f.symbol,
	})
	candidate := GenerateCandidate(CandidateInput{
		Bundle: evidence,
		Side:   f.side,
		Plan:   f.plan,
	})
	return macroUATOutput{scenario: scenario, reaction: reaction, pricedIn: pricedIn, evidence: evidence, candidate: candidate}
}

func macroUATEvent(eventTime time.Time, eventType EventType, direction Direction, actual float64, expected float64, headline string) EventInput {
	return EventInput{
		MacroEventID:  "00000000-0000-0000-0000-000000000001",
		Source:        "uat",
		SourceEventID: "uat-" + strings.ToLower(strings.ReplaceAll(headline, " ", "-")),
		EventType:     eventType,
		Region:        "US",
		EventTimeUTC:  eventTime,
		Headline:      headline,
		ActualValue:   &actual,
		ExpectedValue: &expected,
		Unit:          "fixture",
		Direction:     direction,
		Confidence:    0.9,
		RawPayload:    map[string]any{"fixture": true},
		AffectedETFs:  []string{"QQQ", "SPY", "TLT", "IWM", "XLK", "SMH", "SOXX", "XLF", "GLD"},
	}
}

func macroUATPlan(entry, stop, target *float64) CandidatePlan {
	return CandidatePlan{
		EntryType:       EntryTypePullbackRetest,
		EntryPrice:      entry,
		StopPrice:       stop,
		TargetPrice:     target,
		RiskPercent:     0.01,
		TimeLimit:       "same_session",
		RewardRiskRatio: 2.0,
	}
}

func macroUATCandles(eventTime time.Time, symbol string, pre float64, mid float64, post float64) []marketdata.Candle {
	return []marketdata.Candle{
		macroUATCandle(eventTime.Add(-time.Minute), symbol, pre, pre+0.1, pre-0.1, pre),
		macroUATCandle(eventTime.Add(5*time.Minute), symbol, pre, mid+0.1, mid-0.1, mid),
		macroUATCandle(eventTime.Add(15*time.Minute), symbol, mid, post+0.1, post-0.1, post),
	}
}

func macroUATCandle(ts time.Time, symbol string, open float64, high float64, low float64, close float64) marketdata.Candle {
	return marketdata.Candle{
		Symbol:    symbol,
		Timestamp: ts,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    1000,
	}
}

func macroUATContains(out macroUATOutput, want string) bool {
	needle := strings.ToLower(want)
	if strings.Contains(strings.ToLower(out.reaction.Reason), needle) ||
		strings.Contains(strings.ToLower(out.candidate.RejectionReason), needle) {
		return true
	}
	for _, reason := range out.evidence.WalkawayReasons {
		if strings.Contains(strings.ToLower(reason), needle) {
			return true
		}
	}
	for _, missing := range out.evidence.MissingEvidence {
		if strings.Contains(strings.ToLower(missing), needle) {
			return true
		}
	}
	return false
}
