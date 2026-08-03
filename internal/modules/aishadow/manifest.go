package aishadow

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"jax-trading-assistant/internal/modules/evidencequality"

	"github.com/google/uuid"
)

func EligibleEvents(events []evidencequality.EvaluatedEvent) ([]BenchmarkEvent, error) {
	result := []BenchmarkEvent{}
	for _, event := range events {
		if event.Decision != evidencequality.DecisionWatch && event.Decision != evidencequality.DecisionNoTrade {
			continue
		}
		oneHour, hasOneHour := outcome(event.Outcomes, "receipt", "1h")
		oneDay, hasOneDay := outcome(event.Outcomes, "receipt", "1d")
		if !hasOneHour || !hasOneDay {
			continue
		}
		source := strings.TrimSpace(event.Event.SourceName)
		if source == "" {
			source = strings.TrimSpace(event.Event.Source)
		}
		input := EventInput{
			Title: event.Event.Headline, Summary: event.Event.Summary, Source: source,
			PublicationTimestamp: event.PublicationAt, ReceiptTimestamp: event.ReceiptAt,
			EventCategory: event.EventType, Entities: []string{}, ReceiptEvidence: []string{},
		}
		inputFingerprint, err := fingerprint(input)
		if err != nil {
			return nil, fmt.Errorf("fingerprint event %s: %w", event.Event.InboxID, err)
		}
		result = append(result, BenchmarkEvent{
			ID: event.Event.InboxID, Input: input, InputFingerprint: inputFingerprint,
			Decision: event.Decision, Mapping: event.Mapping,
			Outcome1H: oneHour.AbsoluteRawReturn, Outcome1D: oneDay.AbsoluteRawReturn,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Input.ReceiptTimestamp.Equal(result[j].Input.ReceiptTimestamp) {
			return result[i].ID < result[j].ID
		}
		return result[i].Input.ReceiptTimestamp.Before(result[j].Input.ReceiptTimestamp)
	})
	return result, nil
}

func NewManifest(events []BenchmarkEvent, limit int) (Manifest, error) {
	if limit < 1 || limit > 60 || len(events) < limit {
		return Manifest{}, fmt.Errorf("manifest requires %d eligible events but only %d are available", limit, len(events))
	}
	manifest := Manifest{Version: ManifestVersion, Events: make([]ManifestEvent, limit)}
	for index, event := range events[:limit] {
		manifest.Events[index] = ManifestEvent{EventID: event.ID, InputFingerprint: event.InputFingerprint}
	}
	fingerprint, err := fingerprint(struct {
		Version string          `json:"version"`
		Events  []ManifestEvent `json:"events"`
	}{manifest.Version, manifest.Events})
	if err != nil {
		return Manifest{}, err
	}
	manifest.Fingerprint = fingerprint
	return manifest, nil
}

func WriteManifest(path string, manifest Manifest) error {
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func LoadManifest(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read benchmark manifest: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode benchmark manifest: %w", err)
	}
	if manifest.Version != ManifestVersion || len(manifest.Events) == 0 || len(manifest.Events) > 60 {
		return Manifest{}, fmt.Errorf("invalid benchmark manifest version or event count")
	}
	want, err := fingerprint(struct {
		Version string          `json:"version"`
		Events  []ManifestEvent `json:"events"`
	}{manifest.Version, manifest.Events})
	if err != nil || manifest.Fingerprint != want {
		return Manifest{}, fmt.Errorf("corrupted benchmark manifest fingerprint")
	}
	seen := map[string]bool{}
	for _, event := range manifest.Events {
		if _, err := uuid.Parse(event.EventID); err != nil || strings.TrimSpace(event.InputFingerprint) == "" || seen[event.EventID] {
			return Manifest{}, fmt.Errorf("invalid or duplicate event in benchmark manifest")
		}
		seen[event.EventID] = true
	}
	return manifest, nil
}

func ResolveManifest(manifest Manifest, eligible []BenchmarkEvent, limit int) ([]BenchmarkEvent, error) {
	if limit < 1 || limit > len(manifest.Events) {
		return nil, fmt.Errorf("requested %d events but manifest contains %d", limit, len(manifest.Events))
	}
	byID := map[string]BenchmarkEvent{}
	for _, event := range eligible {
		byID[event.ID] = event
	}
	result := make([]BenchmarkEvent, 0, limit)
	for _, entry := range manifest.Events[:limit] {
		event, ok := byID[entry.EventID]
		if !ok {
			return nil, fmt.Errorf("manifest event %s is no longer qualified", entry.EventID)
		}
		if event.InputFingerprint != entry.InputFingerprint {
			return nil, fmt.Errorf("manifest event %s input fingerprint changed", entry.EventID)
		}
		result = append(result, event)
	}
	return result, nil
}

func outcome(outcomes []evidencequality.Outcome, anchor, horizon string) (evidencequality.Outcome, bool) {
	for _, candidate := range outcomes {
		if candidate.Anchor == anchor && candidate.Horizon == horizon {
			return candidate, true
		}
	}
	return evidencequality.Outcome{}, false
}
