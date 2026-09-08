package advancedquant

import "testing"

func TestDefaultSuitabilityAssessmentRejectsNewRuntimeAdoption(t *testing.T) {
	assessments := DefaultSuitabilityAssessments()
	if len(assessments) != 3 {
		t.Fatalf("assessment count = %d, want 3", len(assessments))
	}
	for _, assessment := range assessments {
		if err := assessment.Validate(); err != nil {
			t.Fatalf("%s invalid: %v", assessment.Candidate, err)
		}
	}
	if assessments[0].Decision != DecisionRecommended || assessments[0].Candidate != CandidateExistingJax {
		t.Fatalf("existing Jax architecture was not recommended: %+v", assessments[0])
	}
	for _, assessment := range assessments[1:] {
		if assessment.Decision != DecisionNotJustified || assessment.ProductionDependency || assessment.ExternalServiceNeeded {
			t.Fatalf("external candidate was adopted by suitability spike: %+v", assessment)
		}
	}
}

func TestSuitabilityRejectsTamperingAndUnknownCandidate(t *testing.T) {
	assessment := DefaultSuitabilityAssessments()[0]
	assessment.DecisionRationale = "changed"
	if err := assessment.Validate(); err == nil {
		t.Fatal("tampered assessment was accepted")
	}
	assessment = DefaultSuitabilityAssessments()[0]
	assessment.Candidate = Candidate("ARBITRARY_RUNTIME")
	assessment.AssessmentID = assessmentID(assessment)
	if err := assessment.Validate(); err == nil {
		t.Fatal("unknown candidate was accepted")
	}
}
