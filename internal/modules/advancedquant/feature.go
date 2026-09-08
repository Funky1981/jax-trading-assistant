package advancedquant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	FeatureContractV1 = "jax.phase12.feature/v1"
	FactorContractV1  = "jax.phase12.factor/v1"
)

type FeatureStatus string

const (
	FeatureKnown       FeatureStatus = "KNOWN"
	FeatureUnknown     FeatureStatus = "UNKNOWN"
	FeatureNotEligible FeatureStatus = "NOT_ELIGIBLE"
)

// Feature is an immutable value with an explicit event-time knowledge claim.
// A feature cannot be eligible unless its source was knowable by AvailabilityAt.
type Feature struct {
	ContractVersion  string            `json:"contract_version"`
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	EventID          string            `json:"event_id"`
	DatasetID        string            `json:"dataset_id"`
	SourceIdentities []string          `json:"source_identities"`
	AvailabilityAt   time.Time         `json:"availability_at"`
	CalculatedAt     time.Time         `json:"calculated_at"`
	AlgorithmVersion string            `json:"algorithm_version"`
	Parameters       map[string]string `json:"parameters,omitempty"`
	Status           FeatureStatus     `json:"status"`
	Value            *float64          `json:"value,omitempty"`
	UnknownReason    string            `json:"unknown_reason,omitempty"`
}

func NewFeature(feature Feature) (Feature, error) {
	feature.ContractVersion = FeatureContractV1
	feature.ID = featureID(feature)
	if err := feature.Validate(); err != nil {
		return Feature{}, err
	}
	return feature, nil
}

func (f Feature) Validate() error {
	if f.ContractVersion != FeatureContractV1 || !validHashID(f.ID, "feat_") || strings.TrimSpace(f.Name) == "" || strings.TrimSpace(f.Version) == "" || strings.TrimSpace(f.EventID) == "" || strings.TrimSpace(f.DatasetID) == "" || len(f.SourceIdentities) == 0 || strings.TrimSpace(f.AlgorithmVersion) == "" {
		return fmt.Errorf("feature identity, provenance and algorithm fields are required")
	}
	if f.AvailabilityAt.IsZero() || f.AvailabilityAt.Location() != time.UTC || f.CalculatedAt.IsZero() || f.CalculatedAt.Location() != time.UTC || f.CalculatedAt.Before(f.AvailabilityAt) {
		return fmt.Errorf("feature availability and calculation timestamps must be UTC and ordered")
	}
	seen := make(map[string]struct{}, len(f.SourceIdentities))
	for _, source := range f.SourceIdentities {
		if strings.TrimSpace(source) == "" {
			return fmt.Errorf("feature source identity must not be empty")
		}
		if _, ok := seen[source]; ok {
			return fmt.Errorf("feature source identities must be unique")
		}
		seen[source] = struct{}{}
	}
	for key, value := range f.Parameters {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return fmt.Errorf("feature parameters must have non-empty names and values")
		}
	}
	switch f.Status {
	case FeatureKnown:
		if f.Value == nil || !finite(*f.Value) || strings.TrimSpace(f.UnknownReason) != "" {
			return fmt.Errorf("known feature requires one finite value and no unknown reason")
		}
	case FeatureUnknown, FeatureNotEligible:
		if f.Value != nil || strings.TrimSpace(f.UnknownReason) == "" {
			return fmt.Errorf("unknown or ineligible feature requires a reason and no value")
		}
	default:
		return fmt.Errorf("unsupported feature status %q", f.Status)
	}
	if f.ID != featureID(f) {
		return fmt.Errorf("feature ID does not match immutable contents")
	}
	return nil
}

// FeatureVector is the only supported way to present event features to a
// later experiment. It rejects mixed event/dataset provenance and duplicates.
type FeatureVector struct {
	ContractVersion string    `json:"contract_version"`
	ID              string    `json:"id"`
	EventID         string    `json:"event_id"`
	DatasetID       string    `json:"dataset_id"`
	AsOf            time.Time `json:"as_of"`
	Features        []Feature `json:"features"`
}

func NewFeatureVector(vector FeatureVector) (FeatureVector, error) {
	vector.ContractVersion = FeatureContractV1
	vector.ID = vectorID(vector)
	if err := vector.Validate(); err != nil {
		return FeatureVector{}, err
	}
	return vector, nil
}

func (v FeatureVector) Validate() error {
	if v.ContractVersion != FeatureContractV1 || !validHashID(v.ID, "fvec_") || strings.TrimSpace(v.EventID) == "" || strings.TrimSpace(v.DatasetID) == "" || v.AsOf.IsZero() || v.AsOf.Location() != time.UTC || len(v.Features) == 0 {
		return fmt.Errorf("feature vector identity, event, dataset, as-of and features are required")
	}
	seen := make(map[string]struct{}, len(v.Features))
	for _, feature := range v.Features {
		if err := feature.Validate(); err != nil {
			return err
		}
		if feature.EventID != v.EventID || feature.DatasetID != v.DatasetID || feature.AvailabilityAt.After(v.AsOf) {
			return fmt.Errorf("feature is not knowable for this vector as-of time")
		}
		if _, ok := seen[feature.Name+"@"+feature.Version]; ok {
			return fmt.Errorf("feature vector contains a duplicate feature version")
		}
		seen[feature.Name+"@"+feature.Version] = struct{}{}
	}
	if v.ID != vectorID(v) {
		return fmt.Errorf("feature vector ID does not match immutable contents")
	}
	return nil
}

type FactorDefinition struct {
	ContractVersion string            `json:"contract_version"`
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Rationale       string            `json:"rationale"`
	InputFeatureIDs []string          `json:"input_feature_ids"`
	Algorithm       string            `json:"algorithm"`
	Parameters      map[string]string `json:"parameters,omitempty"`
}

func NewFactorDefinition(factor FactorDefinition) (FactorDefinition, error) {
	factor.ContractVersion = FactorContractV1
	factor.ID = factorID(factor)
	if err := factor.Validate(); err != nil {
		return FactorDefinition{}, err
	}
	return factor, nil
}

func (f FactorDefinition) Validate() error {
	if f.ContractVersion != FactorContractV1 || !validHashID(f.ID, "factor_") || strings.TrimSpace(f.Name) == "" || strings.TrimSpace(f.Version) == "" || strings.TrimSpace(f.Rationale) == "" || strings.TrimSpace(f.Algorithm) == "" || len(f.InputFeatureIDs) == 0 {
		return fmt.Errorf("factor definition identity, rationale, algorithm and inputs are required")
	}
	seen := map[string]struct{}{}
	for _, id := range f.InputFeatureIDs {
		if !validHashID(id, "feat_") {
			return fmt.Errorf("factor input %q is not a feature identity", id)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("factor inputs must be unique")
		}
		seen[id] = struct{}{}
	}
	if f.ID != factorID(f) {
		return fmt.Errorf("factor ID does not match immutable contents")
	}
	return nil
}

// FeatureStore is an in-memory immutable content-addressed store. It is
// intentionally not a feature-store service; a later persistence adapter may
// serialize these validated artifacts without changing their identities.
type FeatureStore struct {
	mu    sync.RWMutex
	items map[string]Feature
}

func NewFeatureStore() *FeatureStore { return &FeatureStore{items: map[string]Feature{}} }

func (s *FeatureStore) Put(feature Feature) error {
	if err := feature.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.items[feature.ID]; ok && !reflect.DeepEqual(existing, feature) {
		return fmt.Errorf("feature identity collision")
	}
	s.items[feature.ID] = cloneFeature(feature)
	return nil
}

func (s *FeatureStore) Get(id string) (Feature, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	feature, ok := s.items[id]
	if !ok {
		return Feature{}, fmt.Errorf("feature %q not found", id)
	}
	return cloneFeature(feature), nil
}

func (s *FeatureStore) List() []Feature {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Feature, 0, len(s.items))
	for _, feature := range s.items {
		out = append(out, cloneFeature(feature))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func validHashID(id, prefix string) bool {
	if !strings.HasPrefix(id, prefix) || len(id) != len(prefix)+64 {
		return false
	}
	for _, r := range id[len(prefix):] {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return true
}

func featureID(f Feature) string {
	f.ID = ""
	b, _ := json.Marshal(f)
	digest := sha256.Sum256(b)
	return "feat_" + hex.EncodeToString(digest[:])
}

func vectorID(v FeatureVector) string {
	v.ID = ""
	b, _ := json.Marshal(v)
	digest := sha256.Sum256(b)
	return "fvec_" + hex.EncodeToString(digest[:])
}

func factorID(f FactorDefinition) string {
	f.ID = ""
	b, _ := json.Marshal(f)
	digest := sha256.Sum256(b)
	return "factor_" + hex.EncodeToString(digest[:])
}

func cloneFeature(feature Feature) Feature {
	clone := feature
	clone.SourceIdentities = append([]string(nil), feature.SourceIdentities...)
	if feature.Parameters != nil {
		clone.Parameters = make(map[string]string, len(feature.Parameters))
		for key, value := range feature.Parameters {
			clone.Parameters[key] = value
		}
	}
	if feature.Value != nil {
		value := *feature.Value
		clone.Value = &value
	}
	return clone
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
