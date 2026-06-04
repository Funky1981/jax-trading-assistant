package llmcontext

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

type EventInput struct {
	ProviderEventID string
	Headline        string
	EventType       string
	PrimaryRegion   string
	AffectedTheme   string
	AffectedETFs    []string
	Source          string
	Timestamp       time.Time
}

type EventCluster struct {
	CanonicalEventID string
	IsDuplicate      bool
	ClusterSize      int
	Sources          []string
	Summary          string
	AffectedETFs     []string
	DedupeReason     string
}

type EventClusterer struct {
	clusters map[string]*EventCluster
	aiCalled map[string]bool
}

func (c *EventClusterer) Add(in EventInput) EventCluster {
	if c.clusters == nil {
		c.clusters = map[string]*EventCluster{}
	}
	key := canonicalKey(in)
	cluster, exists := c.clusters[key]
	if !exists {
		cluster = &EventCluster{
			CanonicalEventID: key,
			Summary:          in.Headline,
			AffectedETFs:     uniqueStrings(in.AffectedETFs),
			DedupeReason:     "canonical event key",
		}
		c.clusters[key] = cluster
	}
	cluster.ClusterSize++
	cluster.Sources = uniqueStrings(append(cluster.Sources, in.Source))
	cluster.AffectedETFs = uniqueStrings(append(cluster.AffectedETFs, in.AffectedETFs...))
	out := *cluster
	out.IsDuplicate = exists
	return out
}

func (c *EventClusterer) MarkAICall(canonicalEventID string) bool {
	if c.aiCalled == nil {
		c.aiCalled = map[string]bool{}
	}
	if c.aiCalled[canonicalEventID] {
		return false
	}
	c.aiCalled[canonicalEventID] = true
	return true
}

func canonicalKey(in EventInput) string {
	bucket := in.Timestamp.UTC().Truncate(15 * time.Minute).Format(time.RFC3339)
	return strings.Join([]string{
		normalizeToken(in.EventType),
		normalizeHeadline(in.Headline),
		bucket,
		strings.ToUpper(strings.TrimSpace(in.PrimaryRegion)),
		normalizeToken(in.AffectedTheme),
	}, "|")
}

var nonWord = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeHeadline(headline string) string {
	text := nonWord.ReplaceAllString(strings.ToLower(headline), " ")
	words := strings.Fields(text)
	drop := map[string]bool{"us": true, "in": true, "may": true, "comes": true, "came": true, "yields": true, "rise": true}
	out := make([]string, 0, len(words))
	for _, word := range words {
		if !drop[word] {
			out = append(out, word)
		}
	}
	sort.Strings(out)
	if len(out) > 5 {
		out = out[:5]
	}
	return strings.Join(out, "_")
}

func normalizeToken(value string) string {
	return strings.Trim(nonWord.ReplaceAllString(strings.ToLower(value), "_"), "_")
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
