package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

type sentimentSourceDocument struct {
	ID           string
	Symbol       string
	Title        string
	Body         string
	SourceFamily string
	PublishedAt  time.Time
	Metadata     map[string]any
}

type sentimentScore struct {
	Score        float64
	Label        string
	Confidence   float64
	Drivers      []string
	Limitations  []string
	ProviderMode string
	Degraded     bool
}

type sentimentAggregate struct {
	Symbol         string
	Window         string
	ProviderMode   string
	Score          float64
	Label          string
	Confidence     float64
	SourceCount    int
	SourceGroups   map[string]int
	PriceAgreement string
	TopDrivers     []string
	Limitations    []string
	State          string
	ComputedAt     time.Time
	StaleAfter     time.Time
}

type sentimentAggregateOptions struct {
	Symbol             string
	Window             string
	MinimumSourceCount int
	Now                time.Time
}

type sentimentProviderConfig struct {
	Mode string
}

type sentimentProvider interface {
	Score(ctx context.Context, document sentimentSourceDocument) (sentimentScore, error)
}

type localSentimentProvider struct{}

type disabledSentimentProvider struct{}

type hybridSentimentProvider struct {
	mode      string
	primary   sentimentProvider
	fallback  sentimentProvider
	degraded  bool
	lastError string
}

func newSentimentProvider(config sentimentProviderConfig) sentimentProvider {
	switch strings.ToLower(strings.TrimSpace(config.Mode)) {
	case "", "local":
		return newLocalSentimentProvider()
	case "disabled":
		return disabledSentimentProvider{}
	case "external":
		return newHybridSentimentProvider(config, failingExternalSentimentProvider{}, nil)
	case "hybrid":
		return newHybridSentimentProvider(config, failingExternalSentimentProvider{}, newLocalSentimentProvider())
	default:
		return newLocalSentimentProvider()
	}
}

func newLocalSentimentProvider() sentimentProvider {
	return localSentimentProvider{}
}

func newHybridSentimentProvider(config sentimentProviderConfig, primary sentimentProvider, fallback sentimentProvider) sentimentProvider {
	mode := strings.ToLower(strings.TrimSpace(config.Mode))
	if mode == "" {
		mode = "hybrid"
	}
	return hybridSentimentProvider{mode: mode, primary: primary, fallback: fallback}
}

func (disabledSentimentProvider) Score(ctx context.Context, document sentimentSourceDocument) (sentimentScore, error) {
	return sentimentScore{Label: "unavailable", ProviderMode: "disabled", Limitations: []string{"Sentiment scoring is disabled."}}, errSentimentDisabled
}

func (localSentimentProvider) Score(ctx context.Context, document sentimentSourceDocument) (sentimentScore, error) {
	text := strings.ToLower(document.Title + " " + document.Body)
	positive := countSentimentTerms(text, "beat", "beats", "strong", "growth", "raised", "raise", "upgrade", "upgraded", "buyback", "profit", "inflow", "momentum")
	negative := countSentimentTerms(text, "miss", "weak", "downgrade", "downgraded", "lawsuit", "risk", "recession", "outflow", "pullback", "warning")
	total := positive + negative
	score := 0.0
	confidence := 0.35
	if total > 0 {
		score = float64(positive-negative) / float64(total)
		confidence = math.Min(0.95, 0.45+float64(total)*0.08)
	}
	label := sentimentLabel(score)
	drivers := sentimentDrivers(text)
	limitations := []string{"Sentiment is evidence only and does not override policy or risk controls."}
	if strings.TrimSpace(document.Body) == "" {
		limitations = append(limitations, "Source body is sparse; score relies on title text.")
	}
	return sentimentScore{
		Score:        roundSentiment(score),
		Label:        label,
		Confidence:   roundSentiment(confidence),
		Drivers:      drivers,
		Limitations:  limitations,
		ProviderMode: "local",
	}, nil
}

type failingExternalSentimentProvider struct{}

func (failingExternalSentimentProvider) Score(ctx context.Context, document sentimentSourceDocument) (sentimentScore, error) {
	return sentimentScore{}, errors.New("external sentiment provider unavailable")
}

func (p hybridSentimentProvider) Score(ctx context.Context, document sentimentSourceDocument) (sentimentScore, error) {
	if p.primary != nil {
		score, err := p.primary.Score(ctx, document)
		if err == nil {
			score.ProviderMode = p.mode
			return score, nil
		}
		if p.fallback == nil {
			return sentimentScore{Label: "unavailable", ProviderMode: p.mode, Degraded: true, Limitations: []string{err.Error()}}, err
		}
		score, fallbackErr := p.fallback.Score(ctx, document)
		if fallbackErr != nil {
			return sentimentScore{Label: "unavailable", ProviderMode: p.mode, Degraded: true, Limitations: []string{err.Error(), fallbackErr.Error()}}, fallbackErr
		}
		score.ProviderMode = p.mode
		score.Degraded = true
		score.Limitations = append(score.Limitations, "External provider failed; local fallback was used.")
		return score, nil
	}
	return sentimentScore{}, errors.New("sentiment provider has no primary scorer")
}

var errSentimentDisabled = errors.New("sentiment scoring disabled")

func scoreAndAggregateSentiment(ctx context.Context, provider sentimentProvider, documents []sentimentSourceDocument, opts sentimentAggregateOptions) (sentimentAggregate, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	aggregate := sentimentAggregate{
		Symbol:         strings.ToUpper(strings.TrimSpace(opts.Symbol)),
		Window:         defaultString(opts.Window, "24h"),
		ProviderMode:   "local",
		SourceGroups:   map[string]int{},
		PriceAgreement: "unknown",
		State:          "available",
		ComputedAt:     now,
		StaleAfter:     now.Add(24 * time.Hour),
	}
	if provider == nil {
		provider = newLocalSentimentProvider()
	}
	if len(documents) == 0 {
		aggregate.State = "missing"
		aggregate.Label = "unavailable"
		aggregate.Limitations = []string{"No sentiment sources were available for this symbol and window."}
		return aggregate, nil
	}

	var weightedScore float64
	var weightedConfidence float64
	var degraded bool
	driverCounts := map[string]int{}
	for _, document := range documents {
		score, err := provider.Score(ctx, document)
		if errors.Is(err, errSentimentDisabled) {
			aggregate.State = "disabled"
			aggregate.Label = "unavailable"
			aggregate.ProviderMode = "disabled"
			aggregate.Limitations = score.Limitations
			return aggregate, nil
		}
		if err != nil && score.ProviderMode == "" {
			return aggregate, fmt.Errorf("score sentiment: %w", err)
		}
		aggregate.ProviderMode = defaultString(score.ProviderMode, aggregate.ProviderMode)
		aggregate.SourceCount++
		aggregate.SourceGroups[defaultString(document.SourceFamily, "unknown")]++
		weightedScore += score.Score * math.Max(score.Confidence, 0.1)
		weightedConfidence += math.Max(score.Confidence, 0.1)
		degraded = degraded || score.Degraded
		aggregate.Limitations = appendUnique(aggregate.Limitations, score.Limitations...)
		for _, driver := range score.Drivers {
			driverCounts[driver]++
		}
	}
	if weightedConfidence > 0 {
		aggregate.Score = roundSentiment(weightedScore / weightedConfidence)
		aggregate.Confidence = roundSentiment(math.Min(0.95, weightedConfidence/float64(aggregate.SourceCount)))
	}
	aggregate.Label = sentimentLabel(aggregate.Score)
	aggregate.TopDrivers = topSentimentDrivers(driverCounts)
	if aggregate.SourceCount < opts.MinimumSourceCount {
		aggregate.State = "sparse"
		aggregate.Limitations = appendUnique(aggregate.Limitations, "Sentiment source count is below the configured minimum.")
	} else if aggregate.Confidence < 0.5 {
		aggregate.State = "low_confidence"
	} else if degraded {
		aggregate.State = "degraded"
	}
	return aggregate, nil
}

type sentimentAlertRule struct {
	ID                string
	Enabled           bool
	TriggerType       string
	MinimumMove       float64
	MinimumConfidence float64
	CooldownSeconds   int
	Channels          []string
}

type sentimentNotificationEvent struct {
	IdentityKey          string
	Kind                 string
	Title                string
	Summary              string
	SentimentTriggerType string
	EntityType           string
	EntityID             string
	Route                string
	Channels             []string
	CreatedAt            time.Time
}

func evaluateSentimentAlert(rule sentimentAlertRule, previous sentimentAggregate, current sentimentAggregate, now time.Time, recent map[string]time.Time) (sentimentNotificationEvent, bool) {
	if !rule.Enabled || current.State == "disabled" || current.State == "missing" {
		return sentimentNotificationEvent{}, false
	}
	if rule.MinimumConfidence > 0 && current.Confidence > 0 && current.Confidence < rule.MinimumConfidence {
		return sentimentNotificationEvent{}, false
	}
	moved := math.Abs(current.Score - previous.Score)
	flipped := previous.Label != "" && current.Label != "" && previous.Label != current.Label
	if rule.TriggerType == "sentiment_flip" && !flipped {
		return sentimentNotificationEvent{}, false
	}
	if rule.MinimumMove > 0 && moved < rule.MinimumMove {
		return sentimentNotificationEvent{}, false
	}
	identity := fmt.Sprintf("%s:%s:%s:%s", rule.ID, current.Symbol, rule.TriggerType, current.Label)
	if last, ok := recent[identity]; ok && rule.CooldownSeconds > 0 && now.Sub(last) < time.Duration(rule.CooldownSeconds)*time.Second {
		return sentimentNotificationEvent{}, false
	}
	return sentimentNotificationEvent{
		IdentityKey:          identity,
		Kind:                 rule.TriggerType,
		Title:                fmt.Sprintf("Sentiment changed for %s", current.Symbol),
		Summary:              fmt.Sprintf("%s sentiment moved to %s with %.0f%% confidence.", current.Symbol, current.Label, current.Confidence*100),
		SentimentTriggerType: rule.TriggerType,
		EntityType:           "opportunity",
		EntityID:             current.Symbol,
		Route:                "/ai-trading?symbol=" + current.Symbol,
		Channels:             nonEmptyStrings(rule.Channels, []string{"in_app"}),
		CreatedAt:            now,
	}, true
}

func countSentimentTerms(text string, terms ...string) int {
	count := 0
	for _, term := range terms {
		count += strings.Count(text, term)
	}
	return count
}

func sentimentLabel(score float64) string {
	switch {
	case score >= 0.25:
		return "positive"
	case score <= -0.25:
		return "negative"
	default:
		return "mixed"
	}
}

func sentimentDrivers(text string) []string {
	candidates := []string{"upgrade", "strong", "growth", "profit", "momentum", "weak", "risk", "pullback", "downgrade", "outflow"}
	drivers := make([]string, 0, 3)
	for _, candidate := range candidates {
		if strings.Contains(text, candidate) {
			drivers = append(drivers, candidate)
		}
		if len(drivers) >= 3 {
			break
		}
	}
	if len(drivers) == 0 {
		return []string{"neutral source tone"}
	}
	return drivers
}

func topSentimentDrivers(counts map[string]int) []string {
	out := make([]string, 0, 3)
	for len(out) < 3 {
		best := ""
		bestCount := 0
		for driver, count := range counts {
			if count > bestCount && !containsString(out, driver) {
				best = driver
				bestCount = count
			}
		}
		if best == "" {
			break
		}
		out = append(out, best)
	}
	return out
}

func appendUnique(values []string, additions ...string) []string {
	for _, value := range additions {
		value = strings.TrimSpace(value)
		if value == "" || containsString(values, value) {
			continue
		}
		values = append(values, value)
	}
	return values
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func nonEmptyStrings(values, fallback []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func roundSentiment(value float64) float64 {
	return math.Round(value*1000) / 1000
}
