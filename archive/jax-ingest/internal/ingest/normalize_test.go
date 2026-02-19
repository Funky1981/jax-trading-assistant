package ingest

import (
	"reflect"
	"testing"
	"time"
)

func TestNormalizeDexterObservation(t *testing.T) {
	ts := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		name         string
		input        DexterObservation
		kind         string
		eventType    string
		signalType   string
		summary      string
		tags         []string
		impact     float64
		confidence float64
		volumeMult float64
		gapPercent float64
		headline   string
	}{
		{
			name: "earnings event",
			input: DexterObservation{
				Type:           "earnings_detected",
				Symbol:         "aapl",
				ImpactEstimate: 0.82,
				Confidence:     0.71,
				Tags:           []string{"EARNINGS", "Q4"},
				TS:             ts,
			},
			kind:       KindMarketEvent,
			eventType:  ObservationEarnings,
			summary:    "Dexter detected earnings for AAPL.",
			tags:       []string{"earnings", "q4"},
			impact:     0.82,
			confidence: 0.71,
		},
		{
			name: "news headline",
			input: DexterObservation{
				Type:           "news_headline",
				Symbol:         "msft",
				Headline:       "Microsoft announces new AI chips",
				ImpactEstimate: 0.64,
				Confidence:     0.66,
				Tags:           []string{"Breaking"},
				TS:             ts,
			},
			kind:       KindMarketEvent,
			eventType:  ObservationNewsHeadline,
			summary:    "Dexter news for MSFT: Microsoft announces new AI chips.",
			tags:       []string{"news_headline", "breaking"},
			impact:     0.64,
			confidence: 0.66,
			headline:   "Microsoft announces new AI chips",
		},
		{
			name: "unusual volume signal",
			input: DexterObservation{
				Type:           "unusual_volume",
				Symbol:         "TSLA",
				ImpactEstimate: 0.78,
				Confidence:     0.7,
				VolumeMultiple: 3.25,
				Tags:           []string{"volume"},
				Bookmarked:     true,
				TS:             ts,
			},
			kind:         KindSignal,
			signalType:   ObservationUnusualVolume,
			summary:      "Dexter detected unusual volume for TSLA (3.25x avg).",
			tags:         []string{"unusual_volume", "bookmarked", "volume"},
			impact:     0.78,
			confidence: 0.7,
			volumeMult: 3.25,
		},
		{
			name: "price gap signal",
			input: DexterObservation{
				Type:           "price_gap",
				Symbol:         "NVDA",
				ImpactEstimate: 0.55,
				Confidence:     0.61,
				GapPercent:     -7.5,
				TS:             ts,
			},
			kind:         KindSignal,
			signalType:   ObservationPriceGap,
			summary:      "Dexter detected price gap for NVDA (-7.50%).",
			tags:         []string{"price_gap"},
			impact:     0.55,
			confidence: 0.61,
			gapPercent: -7.5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeDexterObservation(tc.input)
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			if got.Kind != tc.kind {
				t.Fatalf("expected kind %q, got %q", tc.kind, got.Kind)
			}

			if tc.kind == KindMarketEvent {
				if got.Event == nil {
					t.Fatalf("expected event, got nil")
				}
				if got.Event.EventType != tc.eventType {
					t.Fatalf("expected event type %q, got %q", tc.eventType, got.Event.EventType)
				}
				if got.Event.Summary != tc.summary {
					t.Fatalf("expected summary %q, got %q", tc.summary, got.Event.Summary)
				}
				if !reflect.DeepEqual(got.Event.Tags, tc.tags) {
					t.Fatalf("expected tags %#v, got %#v", tc.tags, got.Event.Tags)
				}
				if got.Event.ImpactEstimate != tc.impact {
					t.Fatalf("expected impact %.2f, got %.2f", tc.impact, got.Event.ImpactEstimate)
				}
				if got.Event.Confidence != tc.confidence {
					t.Fatalf("expected confidence %.2f, got %.2f", tc.confidence, got.Event.Confidence)
				}
				if tc.headline != "" && got.Event.Headline != tc.headline {
					t.Fatalf("expected headline %q, got %q", tc.headline, got.Event.Headline)
				}
				if got.Signal != nil {
					t.Fatalf("expected signal to be nil")
				}
				return
			}

			if got.Signal == nil {
				t.Fatalf("expected signal, got nil")
			}
			if got.Signal.SignalType != tc.signalType {
				t.Fatalf("expected signal type %q, got %q", tc.signalType, got.Signal.SignalType)
			}
			if got.Signal.Summary != tc.summary {
				t.Fatalf("expected summary %q, got %q", tc.summary, got.Signal.Summary)
			}
			if !reflect.DeepEqual(got.Signal.Tags, tc.tags) {
				t.Fatalf("expected tags %#v, got %#v", tc.tags, got.Signal.Tags)
			}
			if got.Signal.ImpactEstimate != tc.impact {
				t.Fatalf("expected impact %.2f, got %.2f", tc.impact, got.Signal.ImpactEstimate)
			}
			if got.Signal.Confidence != tc.confidence {
				t.Fatalf("expected confidence %.2f, got %.2f", tc.confidence, got.Signal.Confidence)
			}
			if got.Signal.VolumeMultiple != tc.volumeMult {
				t.Fatalf("expected volume multiple %.2f, got %.2f", tc.volumeMult, got.Signal.VolumeMultiple)
			}
			if got.Signal.GapPercent != tc.gapPercent {
				t.Fatalf("expected gap percent %.2f, got %.2f", tc.gapPercent, got.Signal.GapPercent)
			}
			if got.Event != nil {
				t.Fatalf("expected event to be nil")
			}
		})
	}
}
