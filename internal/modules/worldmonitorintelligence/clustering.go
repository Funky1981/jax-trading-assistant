package worldmonitorintelligence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
)

var clusterStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "after": {}, "at": {}, "by": {}, "for": {}, "from": {},
	"in": {}, "of": {}, "on": {}, "the": {}, "to": {}, "with": {}, "says": {}, "said": {},
}

// DeduplicateObservations removes exact replays of the same source event and
// rejects an identity collision whose raw bytes disagree.
func DeduplicateObservations(input []Observation) ([]Observation, error) {
	byIdentity := make(map[string]Observation, len(input))
	for _, observation := range input {
		if err := observation.Validate(); err != nil {
			return nil, err
		}
		key := observation.SourceID + "\x00" + observation.ID
		if previous, exists := byIdentity[key]; exists {
			if string(previous.RawPayload) != string(observation.RawPayload) {
				return nil, fmt.Errorf("source event identity collision for %q", observation.ID)
			}
			continue
		}
		byIdentity[key] = observation
	}
	identities := make([]string, 0, len(byIdentity))
	for identity := range byIdentity {
		identities = append(identities, identity)
	}
	sort.Strings(identities)
	result := make([]Observation, 0, len(identities))
	for _, identity := range identities {
		result = append(result, byIdentity[identity])
	}
	return result, nil
}

// BuildClusters creates deterministic event clusters. Two observations may
// corroborate one event when their event type matches, their significant title
// tokens overlap, and they were collected within 24 hours. This is evidence
// grouping only; it creates no trade candidate or execution intent.
func BuildClusters(input []Observation) ([]EventCluster, error) {
	observations, err := DeduplicateObservations(input)
	if err != nil {
		return nil, err
	}
	if len(observations) == 0 {
		return []EventCluster{}, nil
	}
	parent := make([]int, len(observations))
	for index := range parent {
		parent[index] = index
	}
	var find func(int) int
	find = func(index int) int {
		if parent[index] != index {
			parent[index] = find(parent[index])
		}
		return parent[index]
	}
	union := func(left, right int) {
		leftRoot, rightRoot := find(left), find(right)
		if leftRoot != rightRoot {
			if leftRoot < rightRoot {
				parent[rightRoot] = leftRoot
			} else {
				parent[leftRoot] = rightRoot
			}
		}
	}
	for left := 0; left < len(observations); left++ {
		for right := left + 1; right < len(observations); right++ {
			if relatedObservations(observations[left], observations[right]) {
				union(left, right)
			}
		}
	}
	groups := make(map[int][]Observation)
	for index, observation := range observations {
		root := find(index)
		groups[root] = append(groups[root], observation)
	}
	clusters := make([]EventCluster, 0, len(groups))
	for _, members := range groups {
		sort.Slice(members, func(left, right int) bool { return members[left].ID < members[right].ID })
		canonicalMembers := make([]CanonicalObservation, 0, len(members))
		memberIDs := make([]string, 0, len(members))
		for _, member := range members {
			canonicalMembers = append(canonicalMembers, member.canonical())
			memberIDs = append(memberIDs, member.SourceID+"/"+member.ID)
		}
		signature := sha256.Sum256([]byte(strings.Join(memberIDs, "\x00")))
		unknowns := []string{}
		if len(members) == 1 {
			unknowns = append(unknowns, "single-source event is not corroborated")
		}
		clusters = append(clusters, EventCluster{ContractVersion: IntelligenceContractVersion, Algorithm: ClusterAlgorithmV1, ID: "ecl_" + hex.EncodeToString(signature[:]), EventType: members[0].EventType, Members: canonicalMembers, Unknowns: unknowns})
	}
	sort.Slice(clusters, func(left, right int) bool { return clusters[left].ID < clusters[right].ID })
	return clusters, nil
}

func relatedObservations(left, right Observation) bool {
	if left.EventType != right.EventType {
		return false
	}
	leftTokens, rightTokens := significantTokens(left.Title+" "+left.Summary), significantTokens(right.Title+" "+right.Summary)
	shared := 0
	for token := range leftTokens {
		if _, exists := rightTokens[token]; exists {
			shared++
		}
	}
	if shared < 2 {
		return false
	}
	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return false
	}
	shorter := len(leftTokens)
	if len(rightTokens) < shorter {
		shorter = len(rightTokens)
	}
	return float64(shared)/float64(shorter) >= 0.4 && within24Hours(left.CollectedAt, right.CollectedAt)
}

func significantTokens(text string) map[string]struct{} {
	var builder strings.Builder
	for _, character := range strings.ToLower(text) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			builder.WriteRune(character)
		} else {
			builder.WriteByte(' ')
		}
	}
	result := map[string]struct{}{}
	for _, token := range strings.Fields(builder.String()) {
		if len(token) < 3 {
			continue
		}
		if _, stop := clusterStopWords[token]; !stop {
			result[token] = struct{}{}
		}
	}
	return result
}

func within24Hours(left, right time.Time) bool {
	delta := left.Sub(right)
	if delta < 0 {
		delta = -delta
	}
	return delta <= 24*60*60*1e9
}
