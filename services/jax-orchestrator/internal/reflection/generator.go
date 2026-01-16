package reflection

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"
)

func GenerateBeliefs(now time.Time, window Window, decisions, outcomes []contracts.MemoryItem) []contracts.MemoryItem {
	now = now.UTC()
	window = normalizeWindow(window, now)

	beliefs := make([]contracts.MemoryItem, 0, 2)
	beliefs = append(beliefs, buildOutcomeBelief(now, window, decisions, outcomes))

	if pattern := buildPatternBelief(now, window, decisions, outcomes); pattern != nil {
		beliefs = append(beliefs, *pattern)
	}

	return beliefs
}

func normalizeWindow(window Window, fallback time.Time) Window {
	from := window.From
	to := window.To
	if to.IsZero() {
		to = fallback
	}
	if from.IsZero() || from.After(to) {
		from = to.AddDate(0, 0, -7)
	}
	return Window{From: from.UTC(), To: to.UTC()}
}

func buildOutcomeBelief(now time.Time, window Window, decisions, outcomes []contracts.MemoryItem) contracts.MemoryItem {
	stats := outcomeStats(outcomes)
	classified := stats.wins + stats.losses
	total := stats.total

	summary := ""
	switch {
	case total == 0:
		summary = fmt.Sprintf("No trade outcomes found between %s and %s; performance is unknown.",
			window.From.Format(time.RFC3339), window.To.Format(time.RFC3339))
	case classified == 0:
		summary = fmt.Sprintf("Trade outcomes between %s and %s lack win/loss labels; performance is unclear.",
			window.From.Format(time.RFC3339), window.To.Format(time.RFC3339))
	case stats.wins > stats.losses:
		summary = fmt.Sprintf("Recent outcomes show more wins (%d) than losses (%d).", stats.wins, stats.losses)
	case stats.losses > stats.wins:
		summary = fmt.Sprintf("Recent outcomes show more losses (%d) than wins (%d).", stats.losses, stats.wins)
	default:
		summary = fmt.Sprintf("Recent outcomes are mixed (%d wins, %d losses).", stats.wins, stats.losses)
	}

	evidenceItems := outcomes
	if len(evidenceItems) == 0 {
		evidenceItems = decisions
	}

	data := map[string]any{
		"belief_type":  "outcome_balance",
		"time_window":  windowData(window),
		"evidence_ids": buildEvidenceIDs(evidenceItems),
		"confidence":   roundConfidence(outcomeConfidence(classified, total, stats.wins != stats.losses)),
		"counts": map[string]any{
			"decisions": len(decisions),
			"outcomes":  len(outcomes),
		},
		"metrics": map[string]any{
			"wins":    stats.wins,
			"losses":  stats.losses,
			"unknown": stats.unknown,
			"total":   stats.total,
		},
	}

	return buildBeliefItem(now, summary, []string{"reflection", "outcomes"}, data)
}

func buildPatternBelief(now time.Time, window Window, decisions, outcomes []contracts.MemoryItem) *contracts.MemoryItem {
	tagCounts := make(map[string]int)
	tagEvidence := make(map[string][]contracts.MemoryItem)

	for _, item := range append(decisions, outcomes...) {
		for _, tag := range normalizeTags(item.Tags) {
			if shouldSkipTag(tag) {
				continue
			}
			tagCounts[tag]++
			tagEvidence[tag] = append(tagEvidence[tag], item)
		}
	}

	bestTag, count := topTag(tagCounts)
	if count < 2 {
		return nil
	}

	summary := fmt.Sprintf("Pattern: tag %q appears in %d recent decisions/outcomes.", bestTag, count)
	data := map[string]any{
		"belief_type":  "tag_pattern",
		"time_window":  windowData(window),
		"evidence_ids": buildEvidenceIDs(tagEvidence[bestTag]),
		"confidence":   roundConfidence(patternConfidence(count)),
		"metrics": map[string]any{
			"tag":   bestTag,
			"count": count,
		},
	}

	item := buildBeliefItem(now, summary, []string{"reflection", "pattern", bestTag}, data)
	return &item
}

func buildBeliefItem(now time.Time, summary string, tags []string, data map[string]any) contracts.MemoryItem {
	return contracts.MemoryItem{
		TS:      now.UTC(),
		Type:    "belief",
		Summary: strings.TrimSpace(summary),
		Tags:    normalizeTags(tags),
		Data:    data,
		Source:  &contracts.MemorySource{System: SourceSystem},
	}
}

func windowData(window Window) map[string]string {
	return map[string]string{
		"from": window.From.UTC().Format(time.RFC3339),
		"to":   window.To.UTC().Format(time.RFC3339),
	}
}

type outcomeCount struct {
	wins    int
	losses  int
	unknown int
	total   int
}

func outcomeStats(outcomes []contracts.MemoryItem) outcomeCount {
	stats := outcomeCount{}
	for _, item := range outcomes {
		stats.total++
		switch classifyOutcome(item) {
		case "win":
			stats.wins++
		case "loss":
			stats.losses++
		default:
			stats.unknown++
		}
	}
	return stats
}

func classifyOutcome(item contracts.MemoryItem) string {
	if item.Data != nil {
		if v, ok := item.Data["success"].(bool); ok {
			if v {
				return "win"
			}
			return "loss"
		}
		if v, ok := item.Data["result"].(string); ok {
			return classifyOutcomeLabel(v)
		}
		if v, ok := item.Data["status"].(string); ok {
			return classifyOutcomeLabel(v)
		}
		if v, ok := numericValue(item.Data["pnl"]); ok {
			if v > 0 {
				return "win"
			}
			if v < 0 {
				return "loss"
			}
		}
		if v, ok := numericValue(item.Data["return_pct"]); ok {
			if v > 0 {
				return "win"
			}
			if v < 0 {
				return "loss"
			}
		}
	}

	for _, tag := range normalizeTags(item.Tags) {
		switch tag {
		case "win", "won", "profit", "gain", "success":
			return "win"
		case "loss", "lost", "fail", "failed", "stop", "stopped":
			return "loss"
		}
	}

	summary := strings.ToLower(item.Summary)
	if strings.Contains(summary, "profit") || strings.Contains(summary, "gain") || strings.Contains(summary, "winner") {
		return "win"
	}
	if strings.Contains(summary, "loss") || strings.Contains(summary, "loser") || strings.Contains(summary, "stopped out") {
		return "loss"
	}

	return "unknown"
}

func classifyOutcomeLabel(label string) string {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "win", "won", "profit", "profitable", "success", "successful", "gain":
		return "win"
	case "loss", "lost", "failed", "fail", "unprofitable", "stop", "stopped":
		return "loss"
	default:
		return "unknown"
	}
}

func numericValue(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	case uint32:
		return float64(n), true
	case jsonNumber:
		val, err := n.Float64()
		if err == nil {
			return val, true
		}
	}
	return 0, false
}

type jsonNumber interface {
	Float64() (float64, error)
}

func outcomeConfidence(classified, total int, decisive bool) float64 {
	if total == 0 {
		return 0.2
	}
	confidence := 0.3 + 0.1*float64(classified)
	if decisive {
		confidence += 0.05
	}
	if classified < total {
		confidence -= 0.1
	}
	if classified < 2 && confidence > 0.4 {
		confidence = 0.4
	}
	return clamp(confidence, 0.1, 0.9)
}

func patternConfidence(count int) float64 {
	confidence := 0.3 + 0.1*float64(count)
	if count < 3 && confidence > 0.5 {
		confidence = 0.5
	}
	return clamp(confidence, 0.1, 0.85)
}

func roundConfidence(value float64) float64 {
	return math.Round(value*100) / 100
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func normalizeTags(tags []string) []string {
	return contracts.NormalizeMemoryTags(tags)
}

func shouldSkipTag(tag string) bool {
	switch tag {
	case "decision", "decisions", "outcome", "outcomes", "trade", "trades", "signal", "signals":
		return true
	case "win", "wins", "loss", "losses", "bookmarked":
		return true
	default:
		return false
	}
}

func topTag(tagCounts map[string]int) (string, int) {
	if len(tagCounts) == 0 {
		return "", 0
	}
	type tagCount struct {
		tag   string
		count int
	}
	list := make([]tagCount, 0, len(tagCounts))
	for tag, count := range tagCounts {
		list = append(list, tagCount{tag: tag, count: count})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].count == list[j].count {
			return list[i].tag < list[j].tag
		}
		return list[i].count > list[j].count
	})
	return list[0].tag, list[0].count
}

func buildEvidenceIDs(items []contracts.MemoryItem) []string {
	if len(items) == 0 {
		return []string{"no-evidence"}
	}
	ids := make([]string, 0, len(items))
	for i, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = fallbackEvidenceID(item, i)
		}
		ids = append(ids, id)
	}
	return ids
}

func fallbackEvidenceID(item contracts.MemoryItem, idx int) string {
	ts := "unknown"
	if !item.TS.IsZero() {
		ts = item.TS.UTC().Format(time.RFC3339)
	}
	kind := strings.TrimSpace(item.Type)
	if kind == "" {
		kind = "memory"
	}
	return fmt.Sprintf("%s@%s#%d", kind, ts, idx+1)
}
