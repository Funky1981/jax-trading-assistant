package artifacts

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewArtifact(t *testing.T) {
	params := map[string]any{
		"rsi_period":     14,
		"buy_threshold":  30,
		"sell_threshold": 70,
	}

	riskProfile := RiskProfile{
		MaxPositionPct:    0.20,
		MaxDailyLoss:      1000.0,
		AllowedOrderTypes: []string{"LMT"},
	}

	artifact, err := NewArtifact("rsi_momentum", "1.0.0", params, riskProfile, "test-user")
	if err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	// Verify artifact fields
	if artifact.ID == uuid.Nil {
		t.Error("artifact ID should not be nil")
	}
	if artifact.Strategy.Name != "rsi_momentum" {
		t.Errorf("expected strategy name 'rsi_momentum', got '%s'", artifact.Strategy.Name)
	}
	if artifact.Strategy.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", artifact.Strategy.Version)
	}
	if artifact.Hash == "" {
		t.Error("artifact hash should not be empty")
	}
	if artifact.CreatedBy != "test-user" {
		t.Errorf("expected created_by 'test-user', got '%s'", artifact.CreatedBy)
	}
}

func TestCanonicalPayloadDeterministic(t *testing.T) {
	params := map[string]any{
		"z_param": "last",
		"a_param": "first",
		"m_param": "middle",
	}

	riskProfile := RiskProfile{
		MaxPositionPct:    0.20,
		MaxDailyLoss:      1000.0,
		AllowedOrderTypes: []string{"LMT", "MKT"},
	}

	// Create artifact with fixed timestamp
	now := time.Now().UTC()
	artifact1 := &Artifact{
		ID:            uuid.New(),
		ArtifactID:    "test-artifact-1",
		SchemaVersion: "1.0.0",
		Strategy: StrategyInfo{
			Name:    "test_strategy",
			Version: "1.0.0",
			Params:  params,
		},
		RiskProfile: riskProfile,
		CreatedBy:   "test-user",
		CreatedAt:   now,
	}
	hash1, _ := artifact1.ComputeHash()
	artifact1.Hash = hash1

	// Create identical artifact (same timestamp, different ID)
	artifact2 := &Artifact{
		ID:            uuid.New(),
		ArtifactID:    "test-artifact-1",
		SchemaVersion: "1.0.0",
		Strategy: StrategyInfo{
			Name:    "test_strategy",
			Version: "1.0.0",
			Params:  params,
		},
		RiskProfile: riskProfile,
		CreatedBy:   "test-user",
		CreatedAt:   now,
	}
	hash2, _ := artifact2.ComputeHash()
	artifact2.Hash = hash2

	// Identical content (including timestamp) should produce identical hashes
	if artifact1.Hash != artifact2.Hash {
		t.Error("artifacts with identical content should have identical hashes")
	}

	// Create artifact with different timestamp
	artifact3 := &Artifact{
		ID:            uuid.New(),
		ArtifactID:    "test-artifact-1",
		SchemaVersion: "1.0.0",
		Strategy: StrategyInfo{
			Name:    "test_strategy",
			Version: "1.0.0",
			Params:  params,
		},
		RiskProfile: riskProfile,
		CreatedBy:   "test-user",
		CreatedAt:   now.Add(1 * time.Second),
	}
	hash3, _ := artifact3.ComputeHash()
	artifact3.Hash = hash3

	// Different timestamps should produce different hashes
	if artifact1.Hash == artifact3.Hash {
		t.Error("artifacts with different timestamps should have different hashes")
	}

	// Same artifact should produce same canonical payload repeatedly
	payload1, err := artifact1.CanonicalPayload()
	if err != nil {
		t.Fatalf("failed to get canonical payload 1: %v", err)
	}

	payload2, err := artifact1.CanonicalPayload()
	if err != nil {
		t.Fatalf("failed to get canonical payload 2: %v", err)
	}

	if string(payload1) != string(payload2) {
		t.Error("same artifact should produce identical canonical payloads")
	}
}

func TestComputeHash(t *testing.T) {
	params := map[string]any{
		"param1": 100,
		"param2": "value",
	}

	riskProfile := RiskProfile{
		MaxPositionPct:    0.20,
		MaxDailyLoss:      1000.0,
		AllowedOrderTypes: []string{"LMT"},
	}

	artifact, err := NewArtifact("test_strategy", "1.0.0", params, riskProfile, "test-user")
	if err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	// Hash should be 64 characters (SHA-256 in hex)
	if len(artifact.Hash) != 64 {
		t.Errorf("expected hash length 64, got %d", len(artifact.Hash))
	}

	// Compute hash again should match
	hash2, err := artifact.ComputeHash()
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}

	if artifact.Hash != hash2 {
		t.Errorf("hash mismatch: %s != %s", artifact.Hash, hash2)
	}
}

func TestVerifyHash(t *testing.T) {
	params := map[string]any{"param": "value"}
	riskProfile := RiskProfile{
		MaxPositionPct:    0.20,
		MaxDailyLoss:      1000.0,
		AllowedOrderTypes: []string{"LMT"},
	}

	artifact, err := NewArtifact("test_strategy", "1.0.0", params, riskProfile, "test-user")
	if err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	// Verification should pass for correct hash
	if err := artifact.VerifyHash(); err != nil {
		t.Errorf("hash verification failed: %v", err)
	}

	// Tamper with hash
	artifact.Hash = "0000000000000000000000000000000000000000000000000000000000000000"
	if err := artifact.VerifyHash(); err == nil {
		t.Error("hash verification should fail for tampered hash")
	}
}

func TestApprovalStateMachine(t *testing.T) {
	artifactID := uuid.New()
	approval := NewApproval(artifactID, "test-user")

	// Should start in DRAFT state
	if approval.State != StateDraft {
		t.Errorf("expected initial state DRAFT, got %s", approval.State)
	}

	// Valid transitions from DRAFT
	if !approval.CanTransitionTo(StateValidated) {
		t.Error("DRAFT should allow transition to VALIDATED")
	}
	if !approval.CanTransitionTo(StateRevoked) {
		t.Error("DRAFT should allow transition to REVOKED")
	}
	if approval.CanTransitionTo(StateApproved) {
		t.Error("DRAFT should NOT allow direct transition to APPROVED")
	}

	// Move to VALIDATED
	approval.State = StateValidated

	if !approval.CanTransitionTo(StateReviewed) {
		t.Error("VALIDATED should allow transition to REVIEWED")
	}
	if !approval.CanTransitionTo(StateDraft) {
		t.Error("VALIDATED should allow transition back to DRAFT")
	}

	// Move to APPROVED
	approval.State = StateApproved

	if !approval.CanTransitionTo(StateActive) {
		t.Error("APPROVED should allow transition to ACTIVE")
	}
	if approval.CanTransitionTo(StateDraft) {
		t.Error("APPROVED should NOT allow transition back to DRAFT")
	}
}

func TestIsApproved(t *testing.T) {
	tests := []struct {
		state    ApprovalState
		expected bool
	}{
		{StateDraft, false},
		{StateValidated, false},
		{StateReviewed, false},
		{StateApproved, true},
		{StateActive, true},
		{StateDeprecated, false},
		{StateRevoked, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			approval := &Approval{State: tt.state}
			if approval.IsApproved() != tt.expected {
				t.Errorf("IsApproved() for %s: expected %v, got %v",
					tt.state, tt.expected, approval.IsApproved())
			}
		})
	}
}

func TestIsUsable(t *testing.T) {
	tests := []struct {
		state    ApprovalState
		expected bool
	}{
		{StateDraft, false},
		{StateValidated, false},
		{StateReviewed, false},
		{StateApproved, true},
		{StateActive, true},
		{StateDeprecated, false},
		{StateRevoked, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			approval := &Approval{State: tt.state}
			if approval.IsUsable() != tt.expected {
				t.Errorf("IsUsable() for %s: expected %v, got %v",
					tt.state, tt.expected, approval.IsUsable())
			}
		})
	}
}

func TestHashConsistency(t *testing.T) {
	// Create two artifacts with identical content but different IDs
	params := map[string]any{
		"param1": 100,
		"param2": "value",
	}

	riskProfile := RiskProfile{
		MaxPositionPct:    0.20,
		MaxDailyLoss:      1000.0,
		AllowedOrderTypes: []string{"LMT"},
	}

	now := time.Now().UTC()

	artifact1 := &Artifact{
		ID:            uuid.New(),
		ArtifactID:    "test-artifact-1",
		SchemaVersion: "1.0.0",
		Strategy: StrategyInfo{
			Name:    "test_strategy",
			Version: "1.0.0",
			Params:  params,
		},
		RiskProfile: riskProfile,
		CreatedBy:   "test-user",
		CreatedAt:   now,
	}

	artifact2 := &Artifact{
		ID:            uuid.New(),
		ArtifactID:    "test-artifact-1",
		SchemaVersion: "1.0.0",
		Strategy: StrategyInfo{
			Name:    "test_strategy",
			Version: "1.0.0",
			Params:  params,
		},
		RiskProfile: riskProfile,
		CreatedBy:   "test-user",
		CreatedAt:   now,
	}

	hash1, err := artifact1.ComputeHash()
	if err != nil {
		t.Fatalf("failed to compute hash1: %v", err)
	}

	hash2, err := artifact2.ComputeHash()
	if err != nil {
		t.Fatalf("failed to compute hash2: %v", err)
	}

	// Same content should produce same hash (ID is not part of canonical payload)
	if hash1 != hash2 {
		t.Errorf("identical artifacts should have same hash: %s != %s", hash1, hash2)
	}
}

func TestNewApproval(t *testing.T) {
	artifactID := uuid.New()
	approval := NewApproval(artifactID, "test-user")

	if approval.ID == uuid.Nil {
		t.Error("approval ID should not be nil")
	}
	if approval.ArtifactID != artifactID {
		t.Errorf("expected artifact_id %s, got %s", artifactID, approval.ArtifactID)
	}
	if approval.State != StateDraft {
		t.Errorf("expected initial state DRAFT, got %s", approval.State)
	}
	if approval.StateChangedBy != "test-user" {
		t.Errorf("expected state_changed_by 'test-user', got '%s'", approval.StateChangedBy)
	}
	if approval.CreatedAt.IsZero() {
		t.Error("created_at should not be zero")
	}
}
