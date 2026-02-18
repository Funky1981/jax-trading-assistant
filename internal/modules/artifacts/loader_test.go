package artifacts

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/domain/artifacts"
	"jax-trading-assistant/libs/strategies"

	"github.com/google/uuid"
)

// makeTestArtifact creates a valid artifact for testing
func makeTestArtifact(name, version string) *artifacts.Artifact {
	params := map[string]any{
		"period":    14,
		"threshold": 30,
	}
	riskProfile := artifacts.RiskProfile{
		MaxPositionPct:    0.20,
		MaxDailyLoss:      1000.0,
		AllowedOrderTypes: []string{"LMT"},
	}
	a, err := artifacts.NewArtifact(name, version, params, riskProfile, "test-user")
	if err != nil {
		panic("failed to create test artifact: " + err.Error())
	}
	return a
}

func TestRegisterStrategyFromArtifact_RSI(t *testing.T) {
	registry := strategies.NewRegistry()
	loader := &Loader{registry: registry}

	artifact := makeTestArtifact("rsi_momentum", "1.0.0")

	if err := loader.registerStrategyFromArtifact(artifact); err != nil {
		t.Fatalf("failed to register rsi_momentum: %v", err)
	}

	// Verify strategy is registered (actual ID has _v1 suffix)
	s, err := registry.Get("rsi_momentum_v1")
	if err != nil {
		t.Fatalf("rsi_momentum_v1 not found in registry: %v", err)
	}
	if s.ID() != "rsi_momentum_v1" {
		t.Errorf("expected ID 'rsi_momentum_v1', got '%s'", s.ID())
	}

	// Verify metadata has artifact tracking
	meta, err := registry.GetMetadata("rsi_momentum_v1")
	if err != nil {
		t.Fatalf("metadata not found: %v", err)
	}
	if meta.Extra == nil {
		t.Fatal("metadata.Extra should not be nil")
	}
	if meta.Extra["artifact_id"] != artifact.ArtifactID {
		t.Errorf("expected artifact_id '%s', got '%v'", artifact.ArtifactID, meta.Extra["artifact_id"])
	}
	if meta.Extra["artifact_hash"] != artifact.Hash {
		t.Errorf("expected artifact_hash '%s', got '%v'", artifact.Hash, meta.Extra["artifact_hash"])
	}
}

func TestRegisterStrategyFromArtifact_MACD(t *testing.T) {
	registry := strategies.NewRegistry()
	loader := &Loader{registry: registry}

	artifact := makeTestArtifact("macd_crossover", "1.0.0")

	if err := loader.registerStrategyFromArtifact(artifact); err != nil {
		t.Fatalf("failed to register macd_crossover: %v", err)
	}

	s, err := registry.Get("macd_crossover_v1")
	if err != nil {
		t.Fatalf("macd_crossover_v1 not found in registry: %v", err)
	}
	if s.ID() != "macd_crossover_v1" {
		t.Errorf("expected ID 'macd_crossover_v1', got '%s'", s.ID())
	}
}

func TestRegisterStrategyFromArtifact_MACrossover(t *testing.T) {
	registry := strategies.NewRegistry()
	loader := &Loader{registry: registry}

	artifact := makeTestArtifact("ma_crossover", "1.0.0")

	if err := loader.registerStrategyFromArtifact(artifact); err != nil {
		t.Fatalf("failed to register ma_crossover: %v", err)
	}

	s, err := registry.Get("ma_crossover_v1")
	if err != nil {
		t.Fatalf("ma_crossover_v1 not found in registry: %v", err)
	}
	if s.ID() != "ma_crossover_v1" {
		t.Errorf("expected ID 'ma_crossover_v1', got '%s'", s.ID())
	}
}

func TestRegisterStrategyFromArtifact_UnknownStrategy(t *testing.T) {
	registry := strategies.NewRegistry()
	loader := &Loader{registry: registry}

	artifact := makeTestArtifact("unknown_strategy", "1.0.0")
	// Override strategy name (NewArtifact accepts any name)
	artifact.Strategy.Name = "nonexistent_strategy"

	err := loader.registerStrategyFromArtifact(artifact)
	if err == nil {
		t.Fatal("expected error for unknown strategy, got nil")
	}
}

func TestRegisterStrategyFromArtifact_DuplicateRegistration(t *testing.T) {
	registry := strategies.NewRegistry()
	loader := &Loader{registry: registry}

	artifact := makeTestArtifact("rsi_momentum", "1.0.0")

	// First registration should succeed
	if err := loader.registerStrategyFromArtifact(artifact); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Second registration should fail (registry rejects duplicates)
	err := loader.registerStrategyFromArtifact(artifact)
	if err == nil {
		t.Fatal("expected error on duplicate registration, got nil")
	}
}

func TestLoadArtifact_HashVerification(t *testing.T) {
	registry := strategies.NewRegistry()
	loader := &Loader{registry: registry}

	artifact := makeTestArtifact("rsi_momentum", "1.0.0")

	// loadArtifact verifies hash first — should succeed for valid artifact
	// We can't call loadArtifact directly since it takes context, but we can
	// test hash verification + registration separately
	if err := artifact.VerifyHash(); err != nil {
		t.Fatalf("valid artifact hash verification failed: %v", err)
	}
	if err := loader.registerStrategyFromArtifact(artifact); err != nil {
		t.Fatalf("registration after hash verify failed: %v", err)
	}
}

func TestLoadArtifact_TamperedHashRejected(t *testing.T) {
	artifact := makeTestArtifact("rsi_momentum", "1.0.0")

	// Tamper with the hash
	artifact.Hash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	err := artifact.VerifyHash()
	if err == nil {
		t.Fatal("expected hash verification error for tampered artifact, got nil")
	}
}

func TestAllThreeStrategiesRegister(t *testing.T) {
	registry := strategies.NewRegistry()
	loader := &Loader{registry: registry}

	artifactNames := []string{"rsi_momentum", "macd_crossover", "ma_crossover"}
	registryIDs := []string{"rsi_momentum_v1", "macd_crossover_v1", "ma_crossover_v1"}
	for _, name := range artifactNames {
		artifact := makeTestArtifact(name, "1.0.0")
		if err := loader.registerStrategyFromArtifact(artifact); err != nil {
			t.Fatalf("failed to register %s: %v", name, err)
		}
	}

	// Verify all three are registered
	listed := registry.List()
	if len(listed) != 3 {
		t.Errorf("expected 3 strategies, got %d: %v", len(listed), listed)
	}

	for _, id := range registryIDs {
		if _, err := registry.Get(id); err != nil {
			t.Errorf("strategy %s not found after registration: %v", id, err)
		}
	}
}

func TestArtifactMetadataTracking(t *testing.T) {
	registry := strategies.NewRegistry()
	loader := &Loader{registry: registry}

	type testCase struct {
		artifactName string
		registryID   string
	}
	cases := []testCase{
		{"rsi_momentum", "rsi_momentum_v1"},
		{"macd_crossover", "macd_crossover_v1"},
		{"ma_crossover", "ma_crossover_v1"},
	}
	artifactsByID := make(map[string]*artifacts.Artifact)

	for _, tc := range cases {
		a := makeTestArtifact(tc.artifactName, "2.0.0")
		artifactsByID[tc.registryID] = a
		if err := loader.registerStrategyFromArtifact(a); err != nil {
			t.Fatalf("failed to register %s: %v", tc.artifactName, err)
		}
	}

	// Verify each strategy's metadata contains the correct artifact tracking info
	for _, tc := range cases {
		meta, err := registry.GetMetadata(tc.registryID)
		if err != nil {
			t.Fatalf("metadata for %s not found: %v", tc.registryID, err)
		}

		expected := artifactsByID[tc.registryID]

		gotID, ok := meta.Extra["artifact_id"].(string)
		if !ok {
			t.Errorf("%s: artifact_id not a string", tc.registryID)
			continue
		}
		if gotID != expected.ArtifactID {
			t.Errorf("%s: artifact_id = %s, want %s", tc.registryID, gotID, expected.ArtifactID)
		}

		gotHash, ok := meta.Extra["artifact_hash"].(string)
		if !ok {
			t.Errorf("%s: artifact_hash not a string", tc.registryID)
			continue
		}
		if gotHash != expected.Hash {
			t.Errorf("%s: artifact_hash = %s, want %s", tc.registryID, gotHash, expected.Hash)
		}
	}
}

// Ensure Artifact fields are consistent after creation
func TestArtifactIntegrity(t *testing.T) {
	artifact := makeTestArtifact("rsi_momentum", "1.0.0")

	// ArtifactID should be a readable string identifier
	if artifact.ArtifactID == "" {
		t.Error("ArtifactID should not be empty")
	}

	// ID should be a valid UUID
	if artifact.ID == uuid.Nil {
		t.Error("ID should not be nil UUID")
	}

	// Hash must be set
	if artifact.Hash == "" {
		t.Error("Hash should not be empty")
	}

	// CreatedAt must be recent
	if time.Since(artifact.CreatedAt) > 5*time.Second {
		t.Error("CreatedAt should be recent")
	}

	// Schema version should be set
	if artifact.SchemaVersion == "" {
		t.Error("SchemaVersion should not be empty")
	}
}
