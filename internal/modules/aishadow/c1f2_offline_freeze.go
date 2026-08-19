package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const (
	C1F2WorkPackageIdentity     = "WP-00.03C1F2"
	C1F2OfflineEvidenceIdentity = "ai-shadow-c1f2-offline-freeze-v1"
	C1E3ScoreSHA256             = "918f1b6dc05b14c04b5238022afd44066062ba9bb085d0915a68d38ad3985e8e"
)

type C1F2ComponentFreeze struct {
	Identity string `json:"identity"`
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
}
type C1F2V3HashFreeze struct {
	Dataset             string `json:"dataset"`
	ManifestSHA256      string `json:"manifest_sha256"`
	SemanticFingerprint string `json:"semantic_fingerprint"`
	InputLockSHA256     string `json:"input_lock_sha256"`
	InputFingerprint    string `json:"input_fingerprint"`
	FreezeSHA256        string `json:"freeze_sha256"`
}
type C1F2ProfileFreeze struct {
	Identity          string `json:"identity"`
	Fingerprint       string `json:"fingerprint"`
	DefaultDenyReason string `json:"default_deny_reason"`
	Executable        bool   `json:"executable"`
}
type C1F2OfflineFreeze struct {
	Identity            string                `json:"identity"`
	WorkPackage         string                `json:"work_package"`
	GeneratedAt         time.Time             `json:"generated_at"`
	ProviderContact     bool                  `json:"provider_contact"`
	Inference           bool                  `json:"inference"`
	HostedAuthorization bool                  `json:"hosted_authorization"`
	CredentialLoaded    bool                  `json:"credential_loaded"`
	DatabaseMutation    bool                  `json:"database_mutation"`
	TradingMutation     bool                  `json:"trading_mutation"`
	V3ContentsInspected bool                  `json:"v3_contents_inspected"`
	Components          []C1F2ComponentFreeze `json:"components"`
	Profiles            []C1F2ProfileFreeze   `json:"profiles"`
	V3                  []C1F2V3HashFreeze    `json:"v3_hash_only_preservation"`
	C1E3ScorePath       string                `json:"c1e3_score_path"`
	C1E3ScoreSHA256     string                `json:"c1e3_score_sha256"`
	DevelopmentReplay   []C1FDualScore        `json:"development_replay"`
}

func GenerateC1F2OfflineFreeze(repoRoot, outputRoot string) (string, string, error) {
	if value := os.Getenv(OpenAIDiagnosticInferenceAuthEnv); value != "" && value != "false" {
		return "", "", fmt.Errorf("C1F2 requires %s false or unset", OpenAIDiagnosticInferenceAuthEnv)
	}
	rulesPath := filepath.Join(repoRoot, "config", "event-asset-resolution-v1.json")
	rules, err := assetresolution.LoadRuleset(rulesPath)
	if err != nil {
		return "", "", err
	}
	exposures, err := (assetresolution.Resolver{Rules: rules}).ProxyExposures()
	if err != nil {
		return "", "", err
	}
	schemaHash, err := fingerprint(V5OutputSchema(exposures))
	if err != nil {
		return "", "", err
	}
	components := []C1F2ComponentFreeze{{V6PromptVersion, "internal/modules/aishadow/prompt_v6.go", V6PromptSHA256()}, {V5SchemaVersion, "internal/modules/aishadow/prompt_v5.go", schemaHash}}
	for _, spec := range []struct{ id, path string }{{C1FValidatorVersion, "internal/modules/aishadow/validation_c1f.go"}, {CausalAttributionPolicyVersion, "internal/modules/aishadow/causal_attribution.go"}, {IssuerSemanticIdentityVersion, "internal/modules/aishadow/semantic_identity.go"}, {"event-asset-resolution-v1", "config/event-asset-resolution-v1.json"}, {C1FScoringVersion, "internal/modules/aishadow/scoring_c1f.go"}} {
		hash, e := hashOpaqueFile(filepath.Join(repoRoot, filepath.FromSlash(spec.path)))
		if e != nil {
			return "", "", e
		}
		components = append(components, C1F2ComponentFreeze{spec.id, spec.path, hash})
	}
	if schemaHash != frozenV5SchemaSHA256 {
		return "", "", fmt.Errorf("unchanged v5 schema hash mismatch: got %s", schemaHash)
	}
	if components[2].SHA256 == "" || components[3].SHA256 != frozenC1EPolicySHA256 || components[5].SHA256 != expectedAssetRulesetFileSHA256 {
		return "", "", fmt.Errorf("C1F2 preserved-component hash check failed")
	}
	profiles := []C1F2ProfileFreeze{}
	v3 := []C1F2V3HashFreeze{}
	for _, id := range []string{C1F3ProfileGeneralization, C1F3ProfileBoundary} {
		p, e := LoadC1F3EvaluationProfile(id)
		if e != nil {
			return "", "", e
		}
		hash, e := p.Fingerprint()
		if e != nil {
			return "", "", e
		}
		if e = ValidateC1F3ExecutionReadiness(p); e == nil {
			return "", "", fmt.Errorf("C1F3 profile %s unexpectedly executable", id)
		}
		profiles = append(profiles, C1F2ProfileFreeze{p.Identity, hash, e.Error(), p.Executable})
		for _, file := range []struct{ path, want string }{{p.Dataset.ManifestPath, p.Dataset.ManifestSHA256}, {p.Dataset.InputLockPath, p.Dataset.InputLockSHA256}, {p.Dataset.FreezePath, p.Dataset.FreezeSHA256}} {
			got, e := hashOpaqueFile(filepath.Join(repoRoot, filepath.FromSlash(file.path)))
			if e != nil || got != file.want {
				return "", "", fmt.Errorf("v3 hash-only preservation mismatch for %s", file.path)
			}
		}
		v3 = append(v3, C1F2V3HashFreeze{p.Dataset.Identity, p.Dataset.ManifestSHA256, p.Dataset.SemanticFingerprint, p.Dataset.InputLockSHA256, p.Dataset.InputFingerprint, p.Dataset.FreezeSHA256})
	}
	scoreRel := ".runtime/diagnostics/ai-shadow-c1e3-scoring-v1/WP-00.03C1E3/60f4a31f-e736ac51/score.json"
	scoreHash, err := hashOpaqueFile(filepath.Join(repoRoot, filepath.FromSlash(scoreRel)))
	if err != nil || scoreHash != C1E3ScoreSHA256 {
		return "", "", fmt.Errorf("immutable C1E3 score hash mismatch")
	}
	replays := []C1FDualScore{}
	identity := NewIssuerSemanticIdentity(rules)
	for _, spec := range []struct {
		name, labels, attempts string
		strict, normalized     int
	}{{"Generalization v2", "config/ai-shadow-causal-attribution-labels-generalization-v2-v1.json", ".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1e3-generalization-v2/WP-00.03C1E3-GENERALIZATION/60f4a31f-7363-4801-8724-36a76add70aa/repetition-01/*-attempt-*.json", 19, 44}, {"Boundary v2", "config/ai-shadow-causal-attribution-labels-boundary-v2-v1.json", ".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1e3-boundary-v2/WP-00.03C1E3-BOUNDARY/e736ac51-485a-44a2-b3e1-8e0812ae3793/repetition-01/*-attempt-*.json", 12, 25}} {
		labels, audits, e := loadC1FDevelopmentReplay(repoRoot, spec.labels, spec.attempts)
		if e != nil {
			return "", "", e
		}
		score := ScoreC1FDataset(spec.name, labels, audits, identity)
		if score.Strict.WholeMapping.Numerator != spec.strict || score.Semantic.WholeMapping.Numerator != spec.normalized {
			return "", "", fmt.Errorf("C1E3 replay mismatch for %s", spec.name)
		}
		replays = append(replays, score)
	}
	freeze := C1F2OfflineFreeze{Identity: C1F2OfflineEvidenceIdentity, WorkPackage: C1F2WorkPackageIdentity, GeneratedAt: time.Now().UTC(), Components: components, Profiles: profiles, V3: v3, C1E3ScorePath: scoreRel, C1E3ScoreSHA256: scoreHash, DevelopmentReplay: replays}
	raw, err := json.MarshalIndent(freeze, "", "  ")
	if err != nil {
		return "", "", err
	}
	raw = append(raw, '\n')
	dir := filepath.Join(outputRoot, C1F2OfflineEvidenceIdentity, C1F2WorkPackageIdentity)
	if err = os.MkdirAll(dir, 0o750); err != nil {
		return "", "", err
	}
	path := filepath.Join(dir, "offline-freeze.json")
	if _, e := os.Stat(path); e == nil {
		return "", "", fmt.Errorf("C1F2 offline freeze already exists: %s", path)
	}
	if err = os.WriteFile(path, raw, 0o600); err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(raw)
	return path, hex.EncodeToString(sum[:]), nil
}

func loadC1FDevelopmentReplay(root, labelPath, attemptPattern string) ([]TypedExpectedCase, []DiagnosticAttemptAudit, error) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(labelPath)))
	if err != nil {
		return nil, nil, err
	}
	var sidecar TypedLabelSidecar
	if err = json.Unmarshal(raw, &sidecar); err != nil {
		return nil, nil, err
	}
	paths, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(attemptPattern)))
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(paths)
	audits := []DiagnosticAttemptAudit{}
	for _, path := range paths {
		raw, err = os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		var audit DiagnosticAttemptAudit
		if err = json.Unmarshal(raw, &audit); err != nil {
			return nil, nil, err
		}
		audits = append(audits, audit)
	}
	return sidecar.Cases, audits, nil
}
