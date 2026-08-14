package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
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

type DiagnosticFrozenFileIdentity struct {
	Path                string `json:"path"`
	FileSHA256          string `json:"file_sha256"`
	SemanticFingerprint string `json:"semantic_fingerprint"`
}

type DiagnosticFrozenPolicyIdentity struct {
	Identity   string `json:"identity"`
	Path       string `json:"path"`
	FileSHA256 string `json:"file_sha256"`
}

type DiagnosticFreezeRecord struct {
	Version                              string                         `json:"version"`
	DatasetVersion                       string                         `json:"dataset_version"`
	CreatedAt                            time.Time                      `json:"created_at"`
	IndependenceStatement                string                         `json:"independence_statement,omitempty"`
	ConstructionStatement                string                         `json:"construction_statement,omitempty"`
	Manifest                             DiagnosticFrozenFileIdentity   `json:"manifest"`
	InputFingerprintLock                 DiagnosticFrozenFileIdentity   `json:"input_fingerprint_lock"`
	CaseCount                            int                            `json:"case_count"`
	MappingStatusDistribution            map[string]int                 `json:"mapping_status_distribution"`
	ExpectedResolutionStatusDistribution map[string]int                 `json:"expected_resolution_status_distribution"`
	CategoryDistribution                 map[string]int                 `json:"category_distribution"`
	Policy                               DiagnosticFrozenPolicyIdentity `json:"policy"`
	PromptVersion                        string                         `json:"prompt_version"`
	OutputContract                       string                         `json:"output_contract"`
	LabelVersion                         string                         `json:"label_version"`
	MutationRule                         string                         `json:"mutation_rule"`
}

// LoadFrozenDiagnosticManifest verifies both the exact committed file bytes
// and the manifest's canonical content fingerprint before decoding any cases.
func LoadFrozenDiagnosticManifest(path string, proxyExposures []string) (DiagnosticManifest, error) {
	profile, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileOriginal)
	return LoadFrozenDiagnosticManifestForProfile(profile, path, proxyExposures)
}

func LoadFrozenDiagnosticManifestForProfile(profile DiagnosticEvaluationProfile, path string, proxyExposures []string) (DiagnosticManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return DiagnosticManifest{}, fmt.Errorf("read frozen issuer diagnostic manifest: %w", err)
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != profile.ManifestFileSHA256 {
		return DiagnosticManifest{}, fmt.Errorf("frozen issuer diagnostic manifest file hash changed for profile %s: got %s want %s", profile.Identity, got, profile.ManifestFileSHA256)
	}
	manifest, err := loadDiagnosticManifestForProfile(profile, raw, proxyExposures)
	if err != nil {
		return DiagnosticManifest{}, err
	}
	if manifest.Fingerprint != profile.ManifestFingerprint {
		return DiagnosticManifest{}, fmt.Errorf("frozen issuer diagnostic manifest fingerprint changed for profile %s: got %s want %s", profile.Identity, manifest.Fingerprint, profile.ManifestFingerprint)
	}
	return manifest, nil
}

func LoadDiagnosticFingerprintLock(path string) (DiagnosticFingerprintLock, error) {
	profile, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileOriginal)
	return LoadDiagnosticFingerprintLockForProfile(profile, path)
}

func LoadDiagnosticFingerprintLockForProfile(profile DiagnosticEvaluationProfile, path string) (DiagnosticFingerprintLock, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return DiagnosticFingerprintLock{}, fmt.Errorf("read issuer diagnostic fingerprint lock: %w", err)
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != profile.FingerprintLockFileSHA256 {
		return DiagnosticFingerprintLock{}, fmt.Errorf("frozen issuer diagnostic input lock file hash changed for profile %s: got %s want %s", profile.Identity, got, profile.FingerprintLockFileSHA256)
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
	if lock.Version != profile.FingerprintLockVersion || lock.ManifestFingerprint != profile.ManifestFingerprint ||
		lock.PromptVersion != PromptVersion || lock.OutputContract != SchemaVersion ||
		strings.TrimSpace(lock.PolicyVersion) == "" || len(lock.Events) != profile.CaseCount {
		return DiagnosticFingerprintLock{}, fmt.Errorf("issuer diagnostic fingerprint lock has incompatible metadata or event count")
	}
	want, err := diagnosticFingerprintLockFingerprint(lock)
	if err != nil || lock.Fingerprint != want || lock.Fingerprint != profile.FingerprintLockFingerprint {
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
	profile, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileOriginal)
	raw, err := os.ReadFile(path)
	if err != nil {
		return DiagnosticManifest{}, fmt.Errorf("read issuer diagnostic manifest: %w", err)
	}
	return loadDiagnosticManifestForProfile(profile, raw, proxyExposures)
}

func loadDiagnosticManifestForProfile(profile DiagnosticEvaluationProfile, raw []byte, proxyExposures []string) (DiagnosticManifest, error) {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var manifest DiagnosticManifest
	if err := decoder.Decode(&manifest); err != nil {
		return DiagnosticManifest{}, fmt.Errorf("decode issuer diagnostic manifest: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return DiagnosticManifest{}, fmt.Errorf("decode issuer diagnostic manifest: %w", err)
	}
	if err := ValidateDiagnosticManifestForProfile(profile, manifest, proxyExposures); err != nil {
		return DiagnosticManifest{}, err
	}
	return manifest, nil
}

func ValidateDiagnosticManifest(manifest DiagnosticManifest, proxyExposures []string) error {
	profile, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileOriginal)
	return ValidateDiagnosticManifestForProfile(profile, manifest, proxyExposures)
}

func ValidateDiagnosticManifestForProfile(profile DiagnosticEvaluationProfile, manifest DiagnosticManifest, proxyExposures []string) error {
	if manifest.Version != profile.ManifestVersion || manifest.OutputContract != SchemaVersion || manifest.LabelVersion != DiagnosticLabelVersion {
		return fmt.Errorf("issuer diagnostic manifest has incompatible version metadata")
	}
	if strings.TrimSpace(manifest.PolicyVersion) == "" || strings.TrimSpace(manifest.AdjudicationNote) == "" {
		return fmt.Errorf("issuer diagnostic manifest requires policy and adjudication provenance")
	}
	if len(manifest.Events) != profile.CaseCount {
		return fmt.Errorf("issuer diagnostic profile %s requires exactly %d events", profile.Identity, profile.CaseCount)
	}
	wantFingerprint, err := diagnosticManifestFingerprint(manifest)
	if err != nil || manifest.Fingerprint != wantFingerprint {
		return fmt.Errorf("corrupted issuer diagnostic manifest fingerprint: want %s", wantFingerprint)
	}
	allowedExposures := map[string]bool{NoProxyExposure: true}
	for _, exposure := range proxyExposures {
		allowedExposures[exposure] = true
	}
	categoryCounts := map[string]int{}
	seen := map[string]bool{}
	for _, event := range manifest.Events {
		if strings.TrimSpace(event.ID) == "" || seen[event.ID] {
			return fmt.Errorf("issuer diagnostic manifest contains an empty or duplicate event id")
		}
		seen[event.ID] = true
		if _, allowed := profile.CategoryCounts[event.Category]; !allowed {
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
	for category, expected := range profile.CategoryCounts {
		if categoryCounts[category] != expected {
			return fmt.Errorf("issuer diagnostic category %s requires %d events", category, expected)
		}
	}
	return nil
}

func LoadDiagnosticFreezeRecord(profile DiagnosticEvaluationProfile, path string) (DiagnosticFreezeRecord, error) {
	if !profile.isHoldout() || profile.FreezeVersion == "" || profile.FreezeFileSHA256 == "" {
		return DiagnosticFreezeRecord{}, fmt.Errorf("diagnostic profile %s has no registered freeze record", profile.Identity)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return DiagnosticFreezeRecord{}, fmt.Errorf("read issuer diagnostic freeze record: %w", err)
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != profile.FreezeFileSHA256 {
		return DiagnosticFreezeRecord{}, fmt.Errorf("frozen issuer diagnostic freeze file hash changed for profile %s: got %s want %s", profile.Identity, got, profile.FreezeFileSHA256)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var freeze DiagnosticFreezeRecord
	if err := decoder.Decode(&freeze); err != nil {
		return DiagnosticFreezeRecord{}, fmt.Errorf("decode issuer diagnostic freeze record: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return DiagnosticFreezeRecord{}, fmt.Errorf("decode issuer diagnostic freeze record: %w", err)
	}
	if freeze.Version != profile.FreezeVersion || freeze.DatasetVersion != profile.ManifestVersion || freeze.CaseCount != profile.CaseCount ||
		freeze.Manifest.FileSHA256 != profile.ManifestFileSHA256 || freeze.Manifest.SemanticFingerprint != profile.ManifestFingerprint ||
		freeze.InputFingerprintLock.FileSHA256 != profile.FingerprintLockFileSHA256 || freeze.InputFingerprintLock.SemanticFingerprint != profile.FingerprintLockFingerprint ||
		freeze.Policy.Identity != "event-asset-resolution-v1" || freeze.Policy.FileSHA256 != expectedAssetRulesetFileSHA256 ||
		freeze.PromptVersion != PromptVersion || freeze.OutputContract != SchemaVersion || freeze.LabelVersion != DiagnosticLabelVersion || freeze.CreatedAt.IsZero() {
		return DiagnosticFreezeRecord{}, fmt.Errorf("issuer diagnostic freeze record does not match registered profile %s", profile.Identity)
	}
	return freeze, nil
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

func diagnosticFileSHA256(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read frozen diagnostic file: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

const (
	assetStatusResolved   = "resolved"
	assetStatusAmbiguous  = "ambiguous"
	assetStatusUnresolved = "unresolved"
)
