package worldmonitorintelligence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	IntelligenceContractVersion = "jax.world_monitor_intelligence/v1"
	ObservationNormalizationV1  = "jax.world_monitor_observation_normalization/v1"
	ClusterAlgorithmV1          = "jax.world_monitor_event_cluster/v1"
)

type FreshnessState string

const (
	FreshnessFresh   FreshnessState = "FRESH"
	FreshnessStale   FreshnessState = "STALE"
	FreshnessUnknown FreshnessState = "UNKNOWN"
)

// Observation is the provider-facing input. RawPayload is the exact retained
// provider representation; callers must not substitute canonical JSON here.
type Observation struct {
	ID          string
	SourceID    string
	SourceName  string
	EventType   string
	Title       string
	Summary     string
	SourceURL   string
	PublishedAt *time.Time
	ObservedAt  *time.Time
	CollectedAt time.Time
	ProviderRev string
	RawPayload  []byte
	Freshness   FreshnessState
}

type CanonicalObservation struct {
	ID                   string
	SourceID             string
	SourceName           string
	EventType            string
	Title                string
	Summary              string
	SourceURL            string
	PublishedAt          *time.Time
	ObservedAt           *time.Time
	CollectedAt          time.Time
	RawEvidenceID        string
	RawContentSHA256     string
	NormalizationVersion string
	Freshness            FreshnessState
}

type EventCluster struct {
	ContractVersion string
	Algorithm       string
	ID              string
	EventType       string
	Members         []CanonicalObservation
	Unknowns        []string
}

func (observation Observation) Validate() error {
	if strings.TrimSpace(observation.ID) == "" || strings.TrimSpace(observation.SourceID) == "" {
		return fmt.Errorf("observation ID and source ID are required")
	}
	if strings.TrimSpace(observation.Title) == "" {
		return fmt.Errorf("observation %q has no title", observation.ID)
	}
	if observation.CollectedAt.IsZero() || observation.CollectedAt.Location() != time.UTC {
		return fmt.Errorf("observation %q requires a UTC collection timestamp", observation.ID)
	}
	if len(observation.RawPayload) == 0 {
		return fmt.Errorf("observation %q has no exact raw provider payload", observation.ID)
	}
	if observation.Freshness != FreshnessFresh && observation.Freshness != FreshnessStale && observation.Freshness != FreshnessUnknown {
		return fmt.Errorf("observation %q has unsupported freshness state %q", observation.ID, observation.Freshness)
	}
	return nil
}

func (observation Observation) canonical() CanonicalObservation {
	digest := sha256.Sum256(observation.RawPayload)
	contentDigest := hex.EncodeToString(digest[:])
	identity := sha256.Sum256([]byte(observation.SourceID + "\x00" + observation.ID + "\x00" + contentDigest))
	return CanonicalObservation{
		ID: observation.ID, SourceID: observation.SourceID, SourceName: observation.SourceName,
		EventType: observation.EventType, Title: strings.TrimSpace(observation.Title), Summary: strings.TrimSpace(observation.Summary),
		SourceURL: strings.TrimSpace(observation.SourceURL), PublishedAt: cloneTime(observation.PublishedAt), ObservedAt: cloneTime(observation.ObservedAt),
		CollectedAt: observation.CollectedAt.UTC(), RawEvidenceID: "rpa_" + hex.EncodeToString(identity[:]), RawContentSHA256: contentDigest,
		NormalizationVersion: ObservationNormalizationV1, Freshness: observation.Freshness,
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := value.UTC()
	return &copyValue
}
