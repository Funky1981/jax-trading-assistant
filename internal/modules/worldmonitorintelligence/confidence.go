package worldmonitorintelligence

import (
	"fmt"
	"sort"
)

const ConfidenceAlgorithmV1 = "jax.world_monitor_confidence/v1"

type CorroborationState string

const (
	CorroborationUncorroborated CorroborationState = "UNCORROBORATED"
	CorroborationCorroborated   CorroborationState = "CORROBORATED"
)

type ConfidenceAssessment struct {
	Algorithm              string
	Score                  float64
	Band                   string
	Corroboration          CorroborationState
	SourceCount            int
	IndependentSourceCount int
	Reasons                []string
	Unknowns               []string
}

// AssessConfidence is a deterministic evidence-quality assessment. It
// rewards independent source corroboration and fresh timestamps, but never
// converts confidence into an approval or trade candidate.
func AssessConfidence(cluster EventCluster) (ConfidenceAssessment, error) {
	if len(cluster.Members) == 0 {
		return ConfidenceAssessment{}, fmt.Errorf("cannot assess an empty event cluster")
	}
	sources := map[string]struct{}{}
	for _, member := range cluster.Members {
		if member.SourceID == "" {
			return ConfidenceAssessment{}, fmt.Errorf("cluster member %q has no source identity", member.ID)
		}
		sources[member.SourceID] = struct{}{}
	}
	sourceCount := len(sources)
	score := 0.35
	reasons := []string{"deterministic baseline for a normalized event cluster"}
	if sourceCount >= 3 {
		score += 0.40
		reasons = append(reasons, "three or more independent source identities")
	} else if sourceCount == 2 {
		score += 0.30
		reasons = append(reasons, "two independent source identities corroborate the cluster")
	} else {
		score += 0.10
		reasons = append(reasons, "only one source identity is available")
	}
	freshCount := 0
	unknowns := append([]string{}, cluster.Unknowns...)
	for _, member := range cluster.Members {
		switch member.Freshness {
		case FreshnessFresh:
			freshCount++
		case FreshnessStale:
			unknowns = append(unknowns, fmt.Sprintf("source event %q is stale", member.ID))
		case FreshnessUnknown:
			unknowns = append(unknowns, fmt.Sprintf("freshness is unknown for source event %q", member.ID))
		default:
			unknowns = append(unknowns, fmt.Sprintf("unsupported freshness state for source event %q", member.ID))
		}
		if member.PublishedAt == nil && member.ObservedAt == nil {
			unknowns = append(unknowns, fmt.Sprintf("publication and observation time are absent for source event %q", member.ID))
		}
	}
	if freshCount == len(cluster.Members) {
		score += 0.15
		reasons = append(reasons, "all member observations are marked fresh")
	} else {
		unknowns = append(unknowns, "not all member observations have fresh quality")
	}
	if sourceCount < 2 {
		unknowns = append(unknowns, "independent source corroboration is unavailable")
	}
	if score > 0.95 {
		score = 0.95
	}
	band := "LOW"
	if score >= 0.75 {
		band = "HIGH"
	} else if score >= 0.55 {
		band = "MEDIUM"
	}
	if sourceCount < 2 {
		band = "LOW"
	}
	corroboration := CorroborationUncorroborated
	if sourceCount >= 2 {
		corroboration = CorroborationCorroborated
	}
	sort.Strings(reasons)
	sort.Strings(unknowns)
	return ConfidenceAssessment{Algorithm: ConfidenceAlgorithmV1, Score: score, Band: band, Corroboration: corroboration, SourceCount: sourceCount, IndependentSourceCount: sourceCount, Reasons: uniqueStrings(reasons), Unknowns: uniqueStrings(unknowns)}, nil
}
