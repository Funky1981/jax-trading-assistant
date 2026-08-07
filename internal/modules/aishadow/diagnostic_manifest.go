package aishadow

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	DiagnosticManifestVersion = "ai-shadow-issuer-diagnostic-manifest-v1"
	DiagnosticLabelVersion    = "issuer-recognition-adjudication-v1"
	diagnosticEventCount      = 48
)

var diagnosticCategories = []string{
	"clear_single_issuer_positive",
	"clear_exposure_only_negative",
	"ambiguous_company_reference",
	"multi_issuer_event",
	"less_famous_issuer",
	"common_word_issuer_name",
	"company_mentioned_not_causal",
	"unsupported_unknown_issuer",
}

type DiagnosticManifest struct {
	Version          string            `json:"version"`
	OutputContract   string            `json:"output_contract"`
	PolicyVersion    string            `json:"policy_version"`
	LabelVersion     string            `json:"label_version"`
	AdjudicationNote string            `json:"adjudication_note"`
	Fingerprint      string            `json:"fingerprint"`
	Events           []DiagnosticEvent `json:"events"`
}

type DiagnosticEvent struct {
	ID       string          `json:"id"`
	Category string          `json:"category"`
	Input    EventInput      `json:"input"`
	Label    DiagnosticLabel `json:"adjudicated_label"`
}

type DiagnosticLabel struct {
	MappingStatus            string `json:"mapping_status"`
	DirectIssuer             string `json:"direct_issuer"`
	ProxyExposure            string `json:"proxy_exposure"`
	ExpectedResolutionStatus string `json:"expected_resolution_status"`
	Rationale                string `json:"rationale"`
}

func LoadDiagnosticManifest(path string, proxyExposures []string) (DiagnosticManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return DiagnosticManifest{}, fmt.Errorf("read issuer diagnostic manifest: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var manifest DiagnosticManifest
	if err := decoder.Decode(&manifest); err != nil {
		return DiagnosticManifest{}, fmt.Errorf("decode issuer diagnostic manifest: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return DiagnosticManifest{}, fmt.Errorf("decode issuer diagnostic manifest: %w", err)
	}
	if err := ValidateDiagnosticManifest(manifest, proxyExposures); err != nil {
		return DiagnosticManifest{}, err
	}
	return manifest, nil
}

func ValidateDiagnosticManifest(manifest DiagnosticManifest, proxyExposures []string) error {
	if manifest.Version != DiagnosticManifestVersion || manifest.OutputContract != SchemaVersion || manifest.LabelVersion != DiagnosticLabelVersion {
		return fmt.Errorf("issuer diagnostic manifest has incompatible version metadata")
	}
	if strings.TrimSpace(manifest.PolicyVersion) == "" || strings.TrimSpace(manifest.AdjudicationNote) == "" {
		return fmt.Errorf("issuer diagnostic manifest requires policy and adjudication provenance")
	}
	if len(manifest.Events) != diagnosticEventCount {
		return fmt.Errorf("issuer diagnostic manifest requires exactly %d events", diagnosticEventCount)
	}
	wantFingerprint, err := diagnosticManifestFingerprint(manifest)
	if err != nil || manifest.Fingerprint != wantFingerprint {
		return fmt.Errorf("corrupted issuer diagnostic manifest fingerprint: want %s", wantFingerprint)
	}
	allowedExposures := map[string]bool{NoProxyExposure: true}
	for _, exposure := range proxyExposures {
		allowedExposures[exposure] = true
	}
	allowedCategories := map[string]bool{}
	categoryCounts := map[string]int{}
	for _, category := range diagnosticCategories {
		allowedCategories[category] = true
	}
	seen := map[string]bool{}
	for _, event := range manifest.Events {
		if strings.TrimSpace(event.ID) == "" || seen[event.ID] {
			return fmt.Errorf("issuer diagnostic manifest contains an empty or duplicate event id")
		}
		seen[event.ID] = true
		if !allowedCategories[event.Category] {
			return fmt.Errorf("issuer diagnostic event %s has unsupported category %q", event.ID, event.Category)
		}
		categoryCounts[event.Category]++
		if strings.TrimSpace(event.Input.Title) == "" || event.Input.PublicationTimestamp.IsZero() || event.Input.ReceiptTimestamp.IsZero() || event.Input.PublicationTimestamp.After(event.Input.ReceiptTimestamp) {
			return fmt.Errorf("issuer diagnostic event %s has invalid receipt-time input", event.ID)
		}
		if err := validateDiagnosticLabel(event.Label, allowedExposures); err != nil {
			return fmt.Errorf("issuer diagnostic event %s: %w", event.ID, err)
		}
	}
	for _, category := range diagnosticCategories {
		if categoryCounts[category] != diagnosticEventCount/len(diagnosticCategories) {
			return fmt.Errorf("issuer diagnostic category %s requires %d events", category, diagnosticEventCount/len(diagnosticCategories))
		}
	}
	return nil
}

func validateDiagnosticLabel(label DiagnosticLabel, allowedExposures map[string]bool) error {
	if len(strings.TrimSpace(label.Rationale)) < 20 {
		return fmt.Errorf("adjudicated label requires a reviewable rationale")
	}
	if !contains([]string{assetStatusResolved, assetStatusAmbiguous, assetStatusUnresolved}, label.ExpectedResolutionStatus) {
		return fmt.Errorf("adjudicated label has invalid expected_resolution_status")
	}
	switch label.MappingStatus {
	case "DIRECT":
		if strings.TrimSpace(label.DirectIssuer) == "" || label.ProxyExposure != NoProxyExposure {
			return fmt.Errorf("DIRECT adjudicated label requires one issuer and proxy_exposure NONE")
		}
	case "PROXY":
		if label.DirectIssuer != "" || label.ProxyExposure == NoProxyExposure || !allowedExposures[label.ProxyExposure] || label.ExpectedResolutionStatus != assetStatusResolved {
			return fmt.Errorf("PROXY adjudicated label requires one allowlisted exposure and no issuer")
		}
	case "UNRESOLVED":
		if label.DirectIssuer != "" || label.ProxyExposure != NoProxyExposure || label.ExpectedResolutionStatus != assetStatusUnresolved {
			return fmt.Errorf("UNRESOLVED adjudicated label requires empty issuer, NONE exposure, and unresolved policy status")
		}
	default:
		return fmt.Errorf("adjudicated label has invalid mapping_status")
	}
	return nil
}

func diagnosticManifestFingerprint(manifest DiagnosticManifest) (string, error) {
	return fingerprint(struct {
		Version          string            `json:"version"`
		OutputContract   string            `json:"output_contract"`
		PolicyVersion    string            `json:"policy_version"`
		LabelVersion     string            `json:"label_version"`
		AdjudicationNote string            `json:"adjudication_note"`
		Events           []DiagnosticEvent `json:"events"`
	}{manifest.Version, manifest.OutputContract, manifest.PolicyVersion, manifest.LabelVersion, manifest.AdjudicationNote, manifest.Events})
}

const (
	assetStatusResolved   = "resolved"
	assetStatusAmbiguous  = "ambiguous"
	assetStatusUnresolved = "unresolved"
)
