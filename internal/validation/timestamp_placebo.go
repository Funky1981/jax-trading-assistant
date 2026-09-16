package validation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// TimestampEligibility is prepared once per date. The selector consumes
// these flags and never invokes strategy or structural logic per replicate.
type TimestampEligibility struct {
	RecoveryDateEligible   bool
	ActualSignalExcluded   bool
	ActiveIntervalExcluded bool
	StructuralEligible     bool
}

// TimestampEligibilityEvaluator is a future-validation adapter. Its work is
// performed only by PrepareTimestampPools, once per in-range date.
type TimestampEligibilityEvaluator func(instrument string, date string, year int) TimestampEligibility

// TimestampSelection is the stable scientific output shape used by future
// timestamp-placebo reports.
type TimestampSelection struct {
	Replicate  int    `json:"replicate"`
	Instrument string `json:"instrument"`
	Year       int    `json:"year"`
	Date       string `json:"date"`
}

// TimestampPreparationStats provides deterministic evidence that eligibility
// work was precomputed rather than repeated for each replicate.
type TimestampPreparationStats struct {
	DatesConsidered        int `json:"dates_considered"`
	EligibilityEvaluations int `json:"eligibility_evaluations"`
	EligibleCandidates     int `json:"eligible_candidates"`
}

type timestampStratum struct {
	instrument string
	year       int
	candidates []string
}

// PreparedTimestampPools is immutable after construction and has stable
// instrument/year ordering independent of map iteration order.
type PreparedTimestampPools struct {
	strata         []timestampStratum
	FutureMatching bool
	Stats          TimestampPreparationStats
}

// PrepareTimestampPools filters and groups dates once per instrument/year.
// The future selector intentionally supports only future_matching=false,
// matching the frozen constrained-selection semantics.
func PrepareTimestampPools(dates map[string][]string, start, end string, futureMatching bool, evaluator TimestampEligibilityEvaluator) (PreparedTimestampPools, error) {
	if futureMatching {
		return PreparedTimestampPools{}, fmt.Errorf("future_matching=true is unsupported by the deterministic selector")
	}
	if start == "" || end == "" || start > end {
		return PreparedTimestampPools{}, fmt.Errorf("invalid timestamp placebo boundary %q..%q", start, end)
	}
	instruments := make([]string, 0, len(dates))
	years := map[int]struct{}{}
	stats := TimestampPreparationStats{}
	eligibleByInstrumentYear := map[string]map[int][]string{}
	for instrument := range dates {
		instruments = append(instruments, instrument)
	}
	sort.Strings(instruments)
	for _, instrument := range instruments {
		orderedDates := append([]string(nil), dates[instrument]...)
		sort.Strings(orderedDates)
		for _, date := range orderedDates {
			if date < start || date > end || len(date) < 4 {
				continue
			}
			year := parseYear(date)
			years[year] = struct{}{}
			stats.DatesConsidered++
			stats.EligibilityEvaluations++
			eligibility := TimestampEligibility{
				RecoveryDateEligible: true,
				StructuralEligible:   true,
			}
			if evaluator != nil {
				eligibility = evaluator(instrument, date, year)
			}
			if !eligibility.RecoveryDateEligible || eligibility.ActualSignalExcluded || eligibility.ActiveIntervalExcluded || !eligibility.StructuralEligible {
				continue
			}
			if eligibleByInstrumentYear[instrument] == nil {
				eligibleByInstrumentYear[instrument] = map[int][]string{}
			}
			eligibleByInstrumentYear[instrument][year] = append(eligibleByInstrumentYear[instrument][year], date)
			stats.EligibleCandidates++
		}
	}
	orderedYears := make([]int, 0, len(years))
	for year := range years {
		orderedYears = append(orderedYears, year)
	}
	sort.Ints(orderedYears)
	strata := make([]timestampStratum, 0)
	for _, instrument := range instruments {
		for _, year := range orderedYears {
			candidates := eligibleByInstrumentYear[instrument][year]
			if len(candidates) == 0 {
				continue
			}
			// Candidate order is canonicalized for stable preparation. Selection
			// itself scans this slice and does not sort to find its minimum.
			sort.Strings(candidates)
			strata = append(strata, timestampStratum{instrument: instrument, year: year, candidates: candidates})
		}
	}
	return PreparedTimestampPools{strata: strata, FutureMatching: false, Stats: stats}, nil
}

// Select chooses one minimum SHA-256 score per replicate/instrument/year
// stratum with a linear scan. Ties are resolved by date for total stability.
func (p PreparedTimestampPools) Select(manifestHash string, replicates int) []TimestampSelection {
	if replicates <= 0 {
		return []TimestampSelection{}
	}
	result := make([]TimestampSelection, 0, replicates*len(p.strata))
	for replicate := 0; replicate < replicates; replicate++ {
		for _, stratum := range p.strata {
			bestDate, bestScore := "", ""
			for _, candidate := range stratum.candidates {
				score := timestampPlaceboScore(manifestHash, replicate, stratum.instrument, candidate)
				if bestDate == "" || score < bestScore || (score == bestScore && candidate < bestDate) {
					bestDate, bestScore = candidate, score
				}
			}
			if bestDate != "" {
				result = append(result, TimestampSelection{Replicate: replicate, Instrument: stratum.instrument, Year: stratum.year, Date: bestDate})
			}
		}
	}
	return result
}

// SelectionDigest matches the historical JSON-array digest shape while
// remaining separate from any formal result artifact.
func SelectionDigest(selections []TimestampSelection) string {
	b, _ := json.Marshal(selections)
	digest := sha256.Sum256(b)
	return hex.EncodeToString(digest[:])
}

// SeedSHA256 returns the stable seed identity for timestamp-placebo scoring.
func SeedSHA256(manifestHash string) string {
	digest := sha256.Sum256([]byte(manifestHash + "|timestamp-placebo-v1|"))
	return hex.EncodeToString(digest[:])
}

func timestampPlaceboScore(manifestHash string, replicate int, instrument, candidateDate string) string {
	seed := SeedSHA256(manifestHash)
	return sha256Hex(fmt.Sprintf("%s|%d|%s|%s", seed, replicate, instrument, candidateDate))
}

func sha256Hex(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func parseYear(date string) int {
	year := 0
	for _, digit := range date[:4] {
		year = year*10 + int(digit-'0')
	}
	return year
}
