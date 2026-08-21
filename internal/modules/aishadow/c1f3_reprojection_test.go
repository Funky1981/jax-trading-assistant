package aishadow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestC1F3ReprojectionRejectsArtifactIndexMismatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "artifact-index.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := verifyC1F3ArtifactIndex(dir, strings.Repeat("0", 64)); err == nil || !strings.Contains(err.Error(), "artifact-index SHA-256 mismatch") {
		t.Fatalf("artifact-index mismatch did not fail closed: %v", err)
	}
}

func TestC1F3ReprojectionRejectsRawResponseHashMismatch(t *testing.T) {
	profile, err := LoadC1F3EvaluationProfile(C1F3ProfileGeneralization)
	if err != nil {
		t.Fatal(err)
	}
	input := v5TestInput()
	inputFingerprint, err := EventInputFingerprint(input)
	if err != nil {
		t.Fatal(err)
	}
	raw := marshalV5(t, v5BaseResult())
	audit := DiagnosticAttemptAudit{
		RunID: C1F3GeneralizationSourceRunID, Repetition: 1, CaseID: "offline-case", AttemptNumber: 1,
		InputFingerprint: inputFingerprint, Provider: OpenAIDiagnosticProvider, ConfiguredModel: OpenAIDiagnosticLunaModel,
		ModelReportedIdentifier: OpenAIDiagnosticLunaModel, PromptVersion: V6PromptVersion, OutputContract: V5SchemaVersion,
		RawResponseHash: strings.Repeat("f", 64), RawResponseBody: raw, ValidationStatus: "accepted",
		RequestID: "req-offline", ResponseID: "resp-offline",
	}
	relative := "repetition-01/offline-case-attempt-01.json"
	path := filepath.Join(t.TempDir(), filepath.FromSlash(relative))
	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(audit)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err = reprojectC1F3Attempts(filepath.Dir(filepath.Dir(path)), C1F3ReprojectionSourceSpec{RunID: C1F3GeneralizationSourceRunID}, profile,
		map[string]EventInput{"offline-case": input}, map[string]string{"offline-case": inputFingerprint}, map[string]string{relative: strings.Repeat("a", 64)}, testAssetResolver(t))
	if err == nil || !strings.Contains(err.Error(), "raw-response hash mismatch") {
		t.Fatalf("raw-response mismatch did not fail closed: %v", err)
	}
}

func TestC1F3FrozenComponentsAndScorerRemainUnchanged(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	components, err := validateC1F3ReprojectionComponents(root)
	if err != nil {
		t.Fatal(err)
	}
	if components.Prompt.SHA256 != frozenV6PromptSHA256 || components.OutputContract.SHA256 != frozenV5SchemaSHA256 ||
		components.Validator.SHA256 != frozenC1FValidatorSHA256 || components.Policy.SHA256 != frozenC1EPolicySHA256 ||
		components.Resolver.SHA256 != expectedAssetRulesetFileSHA256 || components.Comparator.SHA256 != frozenSemanticIdentitySHA256 ||
		components.Scorer.SHA256 != frozenC1FScoringSourceSHA256 {
		t.Fatalf("frozen C1F component binding changed: %+v", components)
	}
	for _, profileID := range []string{C1F3ProfileGeneralization, C1F3ProfileBoundary} {
		profile, loadErr := LoadC1F3EvaluationProfile(profileID)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		files := []struct{ path, want string }{
			{profile.Dataset.ManifestPath, profile.Dataset.ManifestSHA256},
			{profile.Dataset.InputLockPath, profile.Dataset.InputLockSHA256},
			{profile.Dataset.FreezePath, profile.Dataset.FreezeSHA256},
			{profile.TypedSidecarPath, profile.TypedSidecarSHA256},
			{profile.ScoringRubricPath, profile.ScoringRubricSHA256},
			{profile.AdjudicationRubricPath, profile.AdjudicationRubricSHA256},
		}
		for _, file := range files {
			got, hashErr := hashOpaqueFile(filepath.Join(root, filepath.FromSlash(file.path)))
			if hashErr != nil || got != file.want {
				t.Fatalf("frozen C1F3 file changed: profile=%s path=%s got=%s want=%s err=%v", profileID, file.path, got, file.want, hashErr)
			}
		}
	}

	resolver := testAssetResolver(t)
	labels := []TypedExpectedCase{{
		CaseID: "offline-direct", ExpectedMappingStatus: "DIRECT", ExpectedDirectIssuer: "Apple",
		ExpectedIssuerAttributions:       []IssuerAttribution{{Issuer: "Apple", CausalRole: CausalRolePrincipal}},
		ExpectedPrincipalProxyCandidates: []string{},
	}}
	mapping := AssetMapping{MappingStatus: "DIRECT", DirectIssuer: "Apple", ProxyExposure: NoProxyExposure}
	attribution := TypedCausalAttribution{IssuerAttributions: labels[0].ExpectedIssuerAttributions, PrincipalProxyCandidates: []string{}}
	score := ScoreC1FDataset("offline-scorer-regression", labels, []DiagnosticAttemptAudit{{
		CaseID: "offline-direct", ValidationStatus: "accepted", EffectiveSemanticMapping: &mapping, TypedAttribution: &attribution,
	}}, NewIssuerSemanticIdentity(resolver.Rules))
	if score.Version != C1FScoringVersion || score.Semantic.WholeMapping.Numerator != 1 || score.Semantic.DirectPrecision.Numerator != 1 ||
		score.Semantic.DirectRecall.Numerator != 1 || score.Semantic.WholeAttribution.Numerator != 1 || score.Semantic.AttributionCompleteness.Numerator != 1 {
		t.Fatalf("frozen scorer regression changed: %+v", score.Semantic)
	}
}
