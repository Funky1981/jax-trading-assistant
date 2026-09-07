package researchrecommendation

import "testing"

func TestConfidenceAssessmentSeparatesUncalibratedScoreFromEvidence(t *testing.T) {
	score := 0.82
	assessment, err := NewConfidenceAssessment(&score, true, 2, 1, 2, []string{"contradiction remains", "outcome calibration belongs to Phase 07"})
	if err != nil {
		t.Fatal(err)
	}
	if assessment.CalibrationStatus != CalibrationNotYetCalibrated || assessment.Scope != "RESEARCH_INTERPRETATION_ONLY" || assessment.ModelScore == nil {
		t.Fatalf("confidence semantics lost: %+v", assessment)
	}
}

func TestConfidenceAssessmentRejectsProbabilityLikeInvalidValues(t *testing.T) {
	for _, value := range []float64{-0.1, 1.1} {
		if _, err := NewConfidenceAssessment(&value, true, 1, 0, 1, []string{"uncalibrated"}); err == nil {
			t.Fatalf("invalid model score accepted: %v", value)
		}
	}
	if _, err := NewConfidenceAssessment(nil, true, 1, 0, 0, nil); err == nil {
		t.Fatal("confidence without explicit uncertainty accepted")
	}
}
