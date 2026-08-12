package aishadow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const (
	holdoutPolicyPath   = "../../../config/event-asset-resolution-v1.json"
	baselineDatasetPath = "../../../config/ai-shadow-issuer-diagnostic-manifest-v1.json"
	nearCopyThreshold   = 0.72
)

type frozenDatasetFile struct {
	Path                string `json:"path"`
	FileSHA256          string `json:"file_sha256"`
	SemanticFingerprint string `json:"semantic_fingerprint"`
}

type frozenPolicyIdentity struct {
	Identity   string `json:"identity"`
	Path       string `json:"path"`
	FileSHA256 string `json:"file_sha256"`
}

type holdoutFreezeRecord struct {
	Version                              string               `json:"version"`
	DatasetVersion                       string               `json:"dataset_version"`
	CreatedAt                            time.Time            `json:"created_at"`
	IndependenceStatement                string               `json:"independence_statement,omitempty"`
	ConstructionStatement                string               `json:"construction_statement,omitempty"`
	Manifest                             frozenDatasetFile    `json:"manifest"`
	InputFingerprintLock                 frozenDatasetFile    `json:"input_fingerprint_lock"`
	CaseCount                            int                  `json:"case_count"`
	MappingStatusDistribution            map[string]int       `json:"mapping_status_distribution"`
	ExpectedResolutionStatusDistribution map[string]int       `json:"expected_resolution_status_distribution"`
	CategoryDistribution                 map[string]int       `json:"category_distribution"`
	Policy                               frozenPolicyIdentity `json:"policy"`
	PromptVersion                        string               `json:"prompt_version"`
	OutputContract                       string               `json:"output_contract"`
	LabelVersion                         string               `json:"label_version"`
	MutationRule                         string               `json:"mutation_rule"`
}

type holdoutDatasetSpec struct {
	Name                 string
	ManifestPath         string
	LockPath             string
	FreezePath           string
	ManifestVersion      string
	LockVersion          string
	FreezeVersion        string
	IDPrefix             string
	CaseCount            int
	MappingDistribution  map[string]int
	CategoryDistribution map[string]int
}

var holdoutDatasetSpecs = []holdoutDatasetSpec{
	{
		Name:                "generalization",
		ManifestPath:        "../../../config/ai-shadow-issuer-generalization-holdout-v1.json",
		LockPath:            "../../../config/ai-shadow-issuer-generalization-holdout-input-fingerprints-v1.json",
		FreezePath:          "../../../config/ai-shadow-issuer-generalization-holdout-freeze-v1.json",
		ManifestVersion:     "ai-shadow-issuer-generalization-holdout-v1",
		LockVersion:         "ai-shadow-issuer-generalization-holdout-input-fingerprints-v1",
		FreezeVersion:       "ai-shadow-issuer-generalization-holdout-freeze-v1",
		IDPrefix:            "issuer-generalization-v1-",
		CaseCount:           48,
		MappingDistribution: map[string]int{"DIRECT": 25, "PROXY": 6, "UNRESOLVED": 17},
		CategoryDistribution: map[string]int{
			"ambiguous_company_reference": 6, "clear_single_issuer": 6,
			"company_mentioned_not_causal": 6, "legitimate_proxy_exposure": 6,
			"less_famous_issuer": 6, "multi_issuer_event": 6,
			"supported_issuer_alias": 6, "unsupported_or_unknown_issuer": 6,
		},
	},
	{
		Name:                "boundary",
		ManifestPath:        "../../../config/ai-shadow-issuer-boundary-challenge-v1.json",
		LockPath:            "../../../config/ai-shadow-issuer-boundary-challenge-input-fingerprints-v1.json",
		FreezePath:          "../../../config/ai-shadow-issuer-boundary-challenge-freeze-v1.json",
		ManifestVersion:     "ai-shadow-issuer-boundary-challenge-v1",
		LockVersion:         "ai-shadow-issuer-boundary-challenge-input-fingerprints-v1",
		FreezeVersion:       "ai-shadow-issuer-boundary-challenge-freeze-v1",
		IDPrefix:            "issuer-boundary-v1-",
		CaseCount:           24,
		MappingDistribution: map[string]int{"DIRECT": 8, "PROXY": 8, "UNRESOLVED": 8},
		CategoryDistribution: map[string]int{
			"ambiguous_company_reference": 2, "incidental_identifier_language": 2,
			"legitimate_proxy_exposure": 4, "multi_issuer_principal_boundary": 2,
			"named_company_causality_boundary": 2, "proxy_term_boundary": 2,
			"strong_relevance_no_direct_issuer": 2, "supported_issuer_alias": 2,
			"tempting_incorrect_proxy": 2, "unsupported_unknown_issuer": 2,
			"weak_legitimate_causal_effect": 2,
		},
	},
}

func TestFrozenIssuerHoldoutDatasets(t *testing.T) {
	rules, err := assetresolution.LoadRuleset(holdoutPolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	resolver := assetresolution.Resolver{Rules: rules}
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		t.Fatal(err)
	}
	allowedExposures := map[string]bool{NoProxyExposure: true}
	for _, exposure := range exposures {
		allowedExposures[exposure] = true
	}
	policySHA, err := fileSHA256(holdoutPolicyPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, spec := range holdoutDatasetSpecs {
		spec := spec
		t.Run(spec.Name, func(t *testing.T) {
			manifest := loadStrictJSON[DiagnosticManifest](t, spec.ManifestPath)
			lock := loadStrictJSON[DiagnosticFingerprintLock](t, spec.LockPath)
			freeze := loadStrictJSON[holdoutFreezeRecord](t, spec.FreezePath)

			if manifest.Version != spec.ManifestVersion || lock.Version != spec.LockVersion || freeze.Version != spec.FreezeVersion {
				t.Fatalf("unexpected version identity: manifest=%q lock=%q freeze=%q", manifest.Version, lock.Version, freeze.Version)
			}
			if freeze.DatasetVersion != manifest.Version || freeze.CreatedAt.IsZero() {
				t.Fatalf("incomplete freeze identity: %+v", freeze)
			}
			if spec.Name == "generalization" && strings.TrimSpace(freeze.IndependenceStatement) == "" {
				t.Fatal("generalization freeze lacks independence statement")
			}
			if spec.Name == "boundary" && strings.TrimSpace(freeze.ConstructionStatement) == "" {
				t.Fatal("boundary freeze lacks construction statement")
			}
			if !strings.Contains(freeze.MutationRule, "new version") {
				t.Fatal("freeze record lacks version-on-observation mutation rule")
			}

			manifestSHA, err := fileSHA256(spec.ManifestPath)
			if err != nil {
				t.Fatal(err)
			}
			lockSHA, err := fileSHA256(spec.LockPath)
			if err != nil {
				t.Fatal(err)
			}
			if manifestSHA != freeze.Manifest.FileSHA256 || lockSHA != freeze.InputFingerprintLock.FileSHA256 {
				t.Fatalf("frozen file hash mismatch: manifest=%s lock=%s", manifestSHA, lockSHA)
			}
			manifestFingerprint, err := diagnosticManifestFingerprint(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if manifest.Fingerprint != manifestFingerprint || freeze.Manifest.SemanticFingerprint != manifestFingerprint {
				t.Fatalf("manifest semantic fingerprint mismatch: embedded=%s computed=%s frozen=%s", manifest.Fingerprint, manifestFingerprint, freeze.Manifest.SemanticFingerprint)
			}
			lockFingerprint, err := diagnosticFingerprintLockFingerprint(lock)
			if err != nil {
				t.Fatal(err)
			}
			if lock.Fingerprint != lockFingerprint || freeze.InputFingerprintLock.SemanticFingerprint != lockFingerprint {
				t.Fatalf("input lock semantic fingerprint mismatch: embedded=%s computed=%s frozen=%s", lock.Fingerprint, lockFingerprint, freeze.InputFingerprintLock.SemanticFingerprint)
			}

			if manifest.OutputContract != SchemaVersion || manifest.PolicyVersion != rules.Version || manifest.LabelVersion != DiagnosticLabelVersion {
				t.Fatalf("manifest contract metadata is incompatible: %+v", manifest)
			}
			if lock.ManifestFingerprint != manifest.Fingerprint || lock.PromptVersion != PromptVersion || lock.OutputContract != SchemaVersion || lock.PolicyVersion != rules.Version {
				t.Fatalf("fingerprint lock metadata is incompatible: %+v", lock)
			}
			if freeze.Policy.Identity != rules.Version || freeze.Policy.FileSHA256 != policySHA || freeze.PromptVersion != PromptVersion || freeze.OutputContract != SchemaVersion || freeze.LabelVersion != DiagnosticLabelVersion {
				t.Fatalf("freeze policy or contract provenance mismatch: %+v", freeze)
			}

			if len(manifest.Events) != spec.CaseCount || len(lock.Events) != spec.CaseCount || freeze.CaseCount != spec.CaseCount {
				t.Fatalf("case count mismatch: manifest=%d lock=%d freeze=%d want=%d", len(manifest.Events), len(lock.Events), freeze.CaseCount, spec.CaseCount)
			}
			mappingCounts := map[string]int{}
			resolutionCounts := map[string]int{}
			categoryCounts := map[string]int{}
			seenIDs := map[string]bool{}
			for index, event := range manifest.Events {
				if !strings.HasPrefix(event.ID, spec.IDPrefix) || seenIDs[event.ID] {
					t.Fatalf("empty, duplicate, or non-versioned case id %q", event.ID)
				}
				seenIDs[event.ID] = true
				if strings.TrimSpace(event.Category) == "" || strings.TrimSpace(event.Input.Title) == "" || strings.TrimSpace(event.Input.Summary) == "" || strings.TrimSpace(event.Input.Source) == "" || strings.TrimSpace(event.Input.EventCategory) == "" || event.Input.Entities == nil || len(event.Input.ReceiptEvidence) == 0 {
					t.Fatalf("case %s lacks a required canonical input field", event.ID)
				}
				if event.Input.PublicationTimestamp.IsZero() || event.Input.ReceiptTimestamp.IsZero() || event.Input.PublicationTimestamp.After(event.Input.ReceiptTimestamp) {
					t.Fatalf("case %s has invalid receipt-time anchors", event.ID)
				}
				if err := validateDiagnosticLabel(event.Label, allowedExposures); err != nil {
					t.Fatalf("case %s: %v", event.ID, err)
				}
				assertExpectedResolution(t, resolver, event)
				if lock.Events[index].ID != event.ID {
					t.Fatalf("input lock order mismatch at %d: got=%s want=%s", index, lock.Events[index].ID, event.ID)
				}
				inputFingerprint, err := EventInputFingerprint(event.Input)
				if err != nil {
					t.Fatal(err)
				}
				if lock.Events[index].InputFingerprint != inputFingerprint {
					t.Fatalf("case %s input fingerprint mismatch: got=%s want=%s", event.ID, lock.Events[index].InputFingerprint, inputFingerprint)
				}
				mappingCounts[event.Label.MappingStatus]++
				resolutionCounts[event.Label.ExpectedResolutionStatus]++
				categoryCounts[event.Category]++
			}
			if !reflect.DeepEqual(mappingCounts, spec.MappingDistribution) || !reflect.DeepEqual(mappingCounts, freeze.MappingStatusDistribution) {
				t.Fatalf("mapping distribution mismatch: got=%v spec=%v freeze=%v", mappingCounts, spec.MappingDistribution, freeze.MappingStatusDistribution)
			}
			if !reflect.DeepEqual(categoryCounts, spec.CategoryDistribution) || !reflect.DeepEqual(categoryCounts, freeze.CategoryDistribution) {
				t.Fatalf("category distribution mismatch: got=%v spec=%v freeze=%v", categoryCounts, spec.CategoryDistribution, freeze.CategoryDistribution)
			}
			if !reflect.DeepEqual(resolutionCounts, freeze.ExpectedResolutionStatusDistribution) {
				t.Fatalf("expected resolution distribution mismatch: got=%v freeze=%v", resolutionCounts, freeze.ExpectedResolutionStatusDistribution)
			}
		})
	}
}

func TestIssuerHoldoutDatasetsHaveNoDuplicateOrNearCopyInputs(t *testing.T) {
	type corpusCase struct {
		Dataset string
		Event   DiagnosticEvent
	}
	corpus := []corpusCase{}
	baseline := loadStrictJSON[DiagnosticManifest](t, baselineDatasetPath)
	for _, event := range baseline.Events {
		corpus = append(corpus, corpusCase{Dataset: "baseline", Event: event})
	}
	for _, spec := range holdoutDatasetSpecs {
		manifest := loadStrictJSON[DiagnosticManifest](t, spec.ManifestPath)
		for _, event := range manifest.Events {
			corpus = append(corpus, corpusCase{Dataset: spec.Name, Event: event})
		}
	}

	seenIDs := map[string]string{}
	seenInputs := map[string]string{}
	seenWording := map[string]string{}
	maximumSimilarity := 0.0
	maximumPair := ""
	for index, current := range corpus {
		if previous, ok := seenIDs[current.Event.ID]; ok {
			t.Fatalf("duplicate case id %s across %s and %s", current.Event.ID, previous, current.Dataset)
		}
		seenIDs[current.Event.ID] = current.Dataset
		fingerprint, err := EventInputFingerprint(current.Event.Input)
		if err != nil {
			t.Fatal(err)
		}
		if previous, ok := seenInputs[fingerprint]; ok {
			t.Fatalf("duplicate canonical input %s and %s", previous, current.Event.ID)
		}
		seenInputs[fingerprint] = current.Event.ID
		wording := normalizedWording(current.Event)
		if previous, ok := seenWording[wording]; ok {
			t.Fatalf("duplicate normalized event wording %s and %s", previous, current.Event.ID)
		}
		seenWording[wording] = current.Event.ID

		if current.Dataset == "baseline" {
			continue
		}
		for priorIndex := 0; priorIndex < index; priorIndex++ {
			prior := corpus[priorIndex]
			if prior.Dataset != "baseline" && prior.Dataset == current.Dataset {
				continue
			}
			similarity := tokenJaccard(normalizedWording(prior.Event), wording)
			if similarity > maximumSimilarity {
				maximumSimilarity = similarity
				maximumPair = prior.Event.ID + "/" + current.Event.ID
			}
			if similarity >= nearCopyThreshold {
				t.Fatalf("near-copy event wording %.3f >= %.2f for %s and %s", similarity, nearCopyThreshold, prior.Event.ID, current.Event.ID)
			}
		}
	}
	t.Logf("maximum cross-dataset token similarity %.3f for %s (threshold %.2f)", maximumSimilarity, maximumPair, nearCopyThreshold)
}

func assertExpectedResolution(t *testing.T, resolver assetresolution.Resolver, event DiagnosticEvent) {
	t.Helper()
	switch event.Label.MappingStatus {
	case "DIRECT":
		result := resolver.ResolveIssuer(assetresolution.IssuerInput{
			IssuerName: event.Label.DirectIssuer, PublicationAt: event.Input.PublicationTimestamp, ReceiptAt: event.Input.ReceiptTimestamp,
		})
		if result.Status != event.Label.ExpectedResolutionStatus {
			t.Fatalf("case %s expected resolution %s, resolver returned %s: %s", event.ID, event.Label.ExpectedResolutionStatus, result.Status, result.Reason)
		}
	case "PROXY":
		result, ok := resolver.ResolveProxyExposure(event.Label.ProxyExposure)
		if !ok || result.Status != event.Label.ExpectedResolutionStatus {
			t.Fatalf("case %s proxy resolution mismatch: ok=%v status=%s expected=%s", event.ID, ok, result.Status, event.Label.ExpectedResolutionStatus)
		}
	case "UNRESOLVED":
		if event.Label.ExpectedResolutionStatus != "unresolved" {
			t.Fatalf("case %s unresolved label has expected resolution %s", event.ID, event.Label.ExpectedResolutionStatus)
		}
	default:
		t.Fatalf("case %s has unsupported mapping status %q", event.ID, event.Label.MappingStatus)
	}
}

func loadStrictJSON[T any](t *testing.T, path string) T {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value T
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	if err := ensureEOF(decoder); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return value
}

func fileSHA256(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

var wordingTokenPattern = regexp.MustCompile(`[a-z0-9]+`)

func normalizedWording(event DiagnosticEvent) string {
	return strings.Join(wordingTokenPattern.FindAllString(strings.ToLower(event.Input.Title+" "+event.Input.Summary), -1), " ")
}

func tokenJaccard(left, right string) float64 {
	leftSet := map[string]bool{}
	rightSet := map[string]bool{}
	for _, token := range strings.Fields(left) {
		leftSet[token] = true
	}
	for _, token := range strings.Fields(right) {
		rightSet[token] = true
	}
	union := map[string]bool{}
	for token := range leftSet {
		union[token] = true
	}
	for token := range rightSet {
		union[token] = true
	}
	intersection := 0
	for token := range leftSet {
		if rightSet[token] {
			intersection++
		}
	}
	if len(union) == 0 {
		return 0
	}
	return float64(intersection) / float64(len(union))
}

func sortedMap(values map[string]int) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return strings.Join(parts, ",")
}
