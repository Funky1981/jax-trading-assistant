package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	DiagnosticManifestVersion             = "ai-shadow-issuer-diagnostic-manifest-v1"
	DiagnosticLabelVersion                = "issuer-recognition-adjudication-v1"
	DiagnosticFingerprintLockVersion      = "ai-shadow-issuer-diagnostic-input-fingerprints-v1"
	ExpectedDiagnosticManifestFingerprint = "3419f1a4b5228ddee6d554ccda68d0bfd44075fa9e4e5657f4e0d550e6006fa5"
	ExpectedDiagnosticManifestFileSHA256  = "544333c68cbca408fc8e39eb9e8d42d670e57a3a65cd94ddaca02dd57f7a3f6b"
	diagnosticEventCount                  = 48
	diagnosticRepetitionCount             = 3
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

type DiagnosticFingerprintLock struct {
	Version             string                       `json:"version"`
	ManifestFingerprint string                       `json:"manifest_fingerprint"`
	PromptVersion       string                       `json:"prompt_version"`
	OutputContract      string                       `json:"output_contract"`
	PolicyVersion       string                       `json:"policy_version"`
	Fingerprint         string                       `json:"fingerprint"`
	Events              []DiagnosticEventFingerprint `json:"events"`
}

type DiagnosticEventFingerprint struct {
	ID               string `json:"id"`
	InputFingerprint string `json:"input_fingerprint"`
}

// LoadFrozenDiagnosticManifest verifies both the exact committed file bytes
// and the manifest's canonical content fingerprint before decoding any cases.
func LoadFrozenDiagnosticManifest(path string, proxyExposures []string) (DiagnosticManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return DiagnosticManifest{}, fmt.Errorf("read frozen issuer diagnostic manifest: %w", err)
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != ExpectedDiagnosticManifestFileSHA256 {
		return DiagnosticManifest{}, fmt.Errorf("frozen issuer diagnostic manifest file hash changed: got %s want %s", got, ExpectedDiagnosticManifestFileSHA256)
	}
	manifest, err := LoadDiagnosticManifest(path, proxyExposures)
	if err != nil {
		return DiagnosticManifest{}, err
	}
	if manifest.Fingerprint != ExpectedDiagnosticManifestFingerprint {
		return DiagnosticManifest{}, fmt.Errorf("frozen issuer diagnostic manifest fingerprint changed: got %s want %s", manifest.Fingerprint, ExpectedDiagnosticManifestFingerprint)
	}
	return manifest, nil
}

func LoadDiagnosticFingerprintLock(path string) (DiagnosticFingerprintLock, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return DiagnosticFingerprintLock{}, fmt.Errorf("read issuer diagnostic fingerprint lock: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var lock DiagnosticFingerprintLock
	if err := decoder.Decode(&lock); err != nil {
		return DiagnosticFingerprintLock{}, fmt.Errorf("decode issuer diagnostic fingerprint lock: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return DiagnosticFingerprintLock{}, fmt.Errorf("decode issuer diagnostic fingerprint lock: %w", err)
	}
	if lock.Version != DiagnosticFingerprintLockVersion || lock.ManifestFingerprint != ExpectedDiagnosticManifestFingerprint ||
		lock.PromptVersion != PromptVersion || lock.OutputContract != SchemaVersion ||
		strings.TrimSpace(lock.PolicyVersion) == "" || len(lock.Events) != diagnosticEventCount {
		return DiagnosticFingerprintLock{}, fmt.Errorf("issuer diagnostic fingerprint lock has incompatible metadata or event count")
	}
	want, err := diagnosticFingerprintLockFingerprint(lock)
	if err != nil || lock.Fingerprint != want {
		return DiagnosticFingerprintLock{}, fmt.Errorf("corrupted issuer diagnostic fingerprint lock: want %s", want)
	}
	seen := map[string]bool{}
	for _, event := range lock.Events {
		if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.InputFingerprint) == "" || seen[event.ID] {
			return DiagnosticFingerprintLock{}, fmt.Errorf("issuer diagnostic fingerprint lock contains an empty or duplicate event")
		}
		seen[event.ID] = true
	}
	return lock, nil
}

func EventInputFingerprint(input EventInput) (string, error) {
	return fingerprint(input)
}

func diagnosticFingerprintLockFingerprint(lock DiagnosticFingerprintLock) (string, error) {
	return fingerprint(struct {
		Version             string                       `json:"version"`
		ManifestFingerprint string                       `json:"manifest_fingerprint"`
		PromptVersion       string                       `json:"prompt_version"`
		OutputContract      string                       `json:"output_contract"`
		PolicyVersion       string                       `json:"policy_version"`
		Events              []DiagnosticEventFingerprint `json:"events"`
	}{lock.Version, lock.ManifestFingerprint, lock.PromptVersion, lock.OutputContract, lock.PolicyVersion, lock.Events})
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
