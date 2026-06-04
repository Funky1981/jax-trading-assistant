package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLocalSentimentProviderScoresAndAggregatesDocuments(t *testing.T) {
	documents := []sentimentSourceDocument{
		{
			ID:           "news-1",
			Symbol:       "SPY",
			Title:        "SPY momentum upgraded after strong flows",
			Body:         "Analysts raised targets after strong growth and profit signals.",
			SourceFamily: "trusted_news",
			PublishedAt:  time.Date(2026, 6, 4, 9, 0, 0, 0, time.UTC),
		},
		{
			ID:           "news-2",
			Symbol:       "SPY",
			Title:        "SPY faces weak breadth",
			Body:         "Weak internals raise risk of a pullback.",
			SourceFamily: "market_news",
			PublishedAt:  time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC),
		},
	}

	provider := newSentimentProvider(sentimentProviderConfig{Mode: "local"})
	aggregate, err := scoreAndAggregateSentiment(context.Background(), provider, documents, sentimentAggregateOptions{
		Symbol:             "SPY",
		Window:             "24h",
		MinimumSourceCount: 2,
		Now:                time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("score aggregate: %v", err)
	}

	if aggregate.State != "available" {
		t.Fatalf("state = %q, want available", aggregate.State)
	}
	if aggregate.SourceCount != 2 {
		t.Fatalf("source count = %d, want 2", aggregate.SourceCount)
	}
	if len(aggregate.TopDrivers) == 0 {
		t.Fatal("expected top drivers")
	}
	if aggregate.ProviderMode != "local" {
		t.Fatalf("provider mode = %q, want local", aggregate.ProviderMode)
	}
}

func TestSentimentProviderDisabledAndExternalFailureBecomeExplicitStates(t *testing.T) {
	document := sentimentSourceDocument{ID: "news-1", Symbol: "QQQ", Title: "QQQ upgrade", Body: "upgrade", SourceFamily: "trusted_news"}

	disabled := newSentimentProvider(sentimentProviderConfig{Mode: "disabled"})
	disabledAggregate, err := scoreAndAggregateSentiment(context.Background(), disabled, []sentimentSourceDocument{document}, sentimentAggregateOptions{
		Symbol:             "QQQ",
		Window:             "24h",
		MinimumSourceCount: 1,
		Now:                time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("disabled aggregate: %v", err)
	}
	if disabledAggregate.State != "disabled" {
		t.Fatalf("disabled state = %q", disabledAggregate.State)
	}

	failing := newHybridSentimentProvider(
		sentimentProviderConfig{Mode: "hybrid"},
		failingSentimentProvider{err: errors.New("external timeout")},
		newLocalSentimentProvider(),
	)
	aggregate, err := scoreAndAggregateSentiment(context.Background(), failing, []sentimentSourceDocument{document}, sentimentAggregateOptions{
		Symbol:             "QQQ",
		Window:             "24h",
		MinimumSourceCount: 1,
		Now:                time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("hybrid aggregate: %v", err)
	}
	if aggregate.State != "degraded" {
		t.Fatalf("hybrid state = %q, want degraded fallback", aggregate.State)
	}
	if aggregate.ProviderMode != "hybrid" {
		t.Fatalf("provider mode = %q, want hybrid", aggregate.ProviderMode)
	}
}

func TestExternalHTTPSentimentProviderNormalizesProviderResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request["symbol"] != "SPY" {
			t.Fatalf("symbol = %v, want SPY", request["symbol"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"score":       0.61,
			"label":       "positive",
			"confidence":  0.82,
			"drivers":     []string{"trusted source upgrade"},
			"limitations": []string{"External model coverage varies by source."},
		})
	}))
	defer server.Close()

	provider := newSentimentProvider(sentimentProviderConfig{Mode: "external", Endpoint: server.URL, Timeout: time.Second})
	score, err := provider.Score(context.Background(), sentimentSourceDocument{
		ID:           "news-external",
		Symbol:       "SPY",
		Title:        "SPY upgraded",
		Body:         "Trusted provider sees stronger flows.",
		SourceFamily: "trusted_news",
	})
	if err != nil {
		t.Fatalf("score: %v", err)
	}
	if score.ProviderMode != "external" || score.Score != 0.61 || score.Confidence != 0.82 {
		t.Fatalf("unexpected score: %+v", score)
	}
	if len(score.Drivers) != 1 || score.Drivers[0] != "trusted source upgrade" {
		t.Fatalf("drivers = %+v", score.Drivers)
	}
}

func TestSentimentAlertRuleSuppressesDuplicatesWithinCooldown(t *testing.T) {
	rule := sentimentAlertRule{
		ID:              "rule-1",
		Enabled:         true,
		TriggerType:     "sentiment_flip",
		MinimumMove:     0.25,
		CooldownSeconds: 3600,
		Channels:        []string{"in_app"},
	}
	previous := sentimentAggregate{Symbol: "IWM", Score: 0.4, Label: "positive"}
	current := sentimentAggregate{Symbol: "IWM", Score: -0.2, Label: "negative", SourceCount: 3, State: "available"}

	event, ok := evaluateSentimentAlert(rule, previous, current, time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC), nil)
	if !ok {
		t.Fatal("expected alert event")
	}
	if event.IdentityKey == "" || event.Route != "/ai-trading?symbol=IWM" {
		t.Fatalf("unexpected event: %+v", event)
	}

	_, ok = evaluateSentimentAlert(rule, previous, current, time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC), map[string]time.Time{
		event.IdentityKey: event.CreatedAt,
	})
	if ok {
		t.Fatal("expected duplicate alert to be suppressed inside cooldown")
	}
}

type failingSentimentProvider struct {
	err error
}

func (p failingSentimentProvider) Score(ctx context.Context, document sentimentSourceDocument) (sentimentScore, error) {
	return sentimentScore{}, p.err
}
