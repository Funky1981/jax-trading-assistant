package provider

import (
	"testing"
	"time"
)

func newSyntheticHealthTracker(t *testing.T, policy HealthPolicy) (*HealthTracker, ProviderDefinition) {
	t.Helper()
	definition := validDefinition(CapabilityCorporateFiling)
	tracker, err := NewHealthTracker(policy, syntheticOperationalPolicy().Component, definition, CapabilityCorporateFiling)
	if err != nil {
		t.Fatal(err)
	}
	return tracker, definition
}

func TestHealthTransitionsAreCapabilityScopedVersionedAndRecoverable(t *testing.T) {
	policy := syntheticOperationalPolicy().Health
	policy.DegradedAfterFailures = 1
	policy.UnavailableAfterFailures = 3
	policy.RecoverySuccesses = 2
	tracker, _ := newSyntheticHealthTracker(t, policy)
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	assessment, transition, err := tracker.ObserveFailure(now, FailureTransportTransient)
	if err != nil || assessment.Status != RuntimeDegraded || transition == nil || transition.From != RuntimeUnknown || transition.To != RuntimeDegraded {
		t.Fatalf("first failure = %+v, transition=%+v, err=%v", assessment, transition, err)
	}
	if assessment.Policy.ID != policy.Identity.ID || assessment.Component.ID != syntheticOperationalPolicy().Component.ID {
		t.Fatalf("health identity missing: %+v", assessment)
	}
	_, _, _ = tracker.ObserveFailure(now.Add(time.Second), FailureProviderServer)
	assessment, transition, err = tracker.ObserveFailure(now.Add(2*time.Second), FailureTemporaryUnavailable)
	if err != nil || assessment.Status != RuntimeUnavailable || transition == nil || transition.To != RuntimeUnavailable {
		t.Fatalf("unavailable transition = %+v, transition=%+v, err=%v", assessment, transition, err)
	}
	assessment, transition, err = tracker.ObserveSuccess(now.Add(3 * time.Second))
	if err != nil || assessment.Status != RuntimeUnavailable || transition != nil || assessment.ReasonCode != HealthReasonRecovering {
		t.Fatalf("first recovery success = %+v, transition=%+v, err=%v", assessment, transition, err)
	}
	assessment, transition, err = tracker.ObserveSuccess(now.Add(4 * time.Second))
	if err != nil || assessment.Status != RuntimeHealthy || transition == nil || transition.To != RuntimeHealthy || assessment.ReasonCode != "" {
		t.Fatalf("recovered assessment = %+v, transition=%+v, err=%v", assessment, transition, err)
	}
}

func TestHealthInputsDistinguishRateAuthDataQualityAndCallerControl(t *testing.T) {
	policy := syntheticOperationalPolicy().Health
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	tracker, _ := newSyntheticHealthTracker(t, policy)
	assessment, _, err := tracker.ObserveFailure(now, FailureRateLimited)
	if err != nil || assessment.Status != RuntimeDegraded || assessment.ReasonCode != HealthReasonRateLimitExhausted {
		t.Fatalf("rate-limit assessment = %+v, %v", assessment, err)
	}

	tracker, _ = newSyntheticHealthTracker(t, policy)
	assessment, _, err = tracker.ObserveFailure(now, FailureAuthentication)
	if err != nil || assessment.Status != RuntimeUnavailable || assessment.ReasonCode != HealthReasonAuthentication {
		t.Fatalf("authentication assessment = %+v, %v", assessment, err)
	}

	for _, class := range []FailureClass{FailureMalformedRequest, FailureProviderPayloadParse, FailureCanonicalValidation, FailureProvenanceMismatch, FailureCallerCancelled, FailureOverallDeadline, FailurePermanentRejection, FailureAttemptDeadline} {
		tracker, _ = newSyntheticHealthTracker(t, policy)
		assessment, transition, err := tracker.ObserveFailure(now, class)
		if err != nil || assessment.Status != RuntimeUnknown || transition != nil || assessment.ConsecutiveFailures != 0 {
			t.Fatalf("neutral class %q changed health: %+v transition=%+v err=%v", class, assessment, transition, err)
		}
	}

	policy.CountAttemptDeadline = true
	tracker, _ = newSyntheticHealthTracker(t, policy)
	assessment, _, err = tracker.ObserveFailure(now, FailureAttemptDeadline)
	if err != nil || assessment.Status != RuntimeDegraded {
		t.Fatalf("policy-counted attempt deadline = %+v, %v", assessment, err)
	}
}

func TestHealthAssessmentHorizonAndContradictoryTransition(t *testing.T) {
	policy := syntheticOperationalPolicy().Health
	policy.DegradedAfterFailures = 2
	policy.UnavailableAfterFailures = 3
	policy.AssessmentHorizon = 10 * time.Second
	tracker, _ := newSyntheticHealthTracker(t, policy)
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	if _, _, err := tracker.ObserveSuccess(now); err != nil {
		t.Fatal(err)
	}
	assessment, err := tracker.Snapshot(now.Add(11 * time.Second))
	if err != nil || assessment.Status != RuntimeUnknown || assessment.ReasonCode != HealthReasonHorizonElapsed {
		t.Fatalf("horizon assessment = %+v, %v", assessment, err)
	}
	assessment, transition, err := tracker.ObserveFailure(now.Add(12*time.Second), FailureTransportTransient)
	if err != nil || assessment.Status != RuntimeUnknown || assessment.ReasonCode != HealthReasonInsufficientEvidence || assessment.ConsecutiveFailures != 1 || transition != nil {
		t.Fatalf("post-horizon failure reused expired evidence: assessment=%+v transition=%+v err=%v", assessment, transition, err)
	}
	if _, _, err := tracker.ObserveFailure(now.Add(-time.Second), FailureTransportTransient); err == nil {
		t.Fatal("ObserveFailure() accepted out-of-order contradictory evidence")
	}
}

func TestProviderHealthAndDatumFreshnessRemainIndependent(t *testing.T) {
	tracker, definition := newSyntheticHealthTracker(t, syntheticOperationalPolicy().Health)
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	health, _, err := tracker.ObserveSuccess(now)
	if err != nil || health.Status != RuntimeHealthy {
		t.Fatalf("healthy assessment = %+v, %v", health, err)
	}

	particularDatumState := TemporalStale
	if health.Status != RuntimeHealthy || particularDatumState != TemporalStale {
		t.Fatal("provider health altered particular datum freshness")
	}
	state, err := health.AsRuntimeState(definition, FreshnessFresh, QualityAcceptable)
	if err != nil || state.Status != RuntimeHealthy {
		t.Fatalf("runtime projection = %+v, %v", state, err)
	}
	if _, err := health.AsRuntimeState(definition, FreshnessStale, QualityAcceptable); err == nil {
		t.Fatal("aggregate runtime projection bypassed accepted WP-02.01 consistency rules")
	}
	if particularDatumState != TemporalStale {
		t.Fatal("runtime projection mutated datum freshness")
	}
}

func TestStaticCapabilityStateCannotContradictRegistration(t *testing.T) {
	definition := validDefinition(CapabilityCorporateFiling)
	definition.Capabilities[0].Support = SupportDisabled
	tracker, err := NewHealthTracker(syntheticOperationalPolicy().Health, syntheticOperationalPolicy().Component, definition, CapabilityCorporateFiling)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	assessment, err := tracker.Snapshot(now)
	if err != nil || assessment.Status != RuntimeDisabled || assessment.ReasonCode != HealthReasonStaticDisabled {
		t.Fatalf("disabled snapshot = %+v, %v", assessment, err)
	}
	if _, _, err := tracker.ObserveSuccess(now); err == nil {
		t.Fatal("disabled capability accepted healthy transition")
	}
}
