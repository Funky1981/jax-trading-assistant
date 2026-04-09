package harness

import "testing"

func TestValidatorRejectsForbiddenActionLanguage(t *testing.T) {
	result := NewValidator().ValidateAnswer(DefaultPolicy(ModeResearch), EvidenceBundle{}, "I executed the trade for you.")
	if result.OK {
		t.Fatal("expected forbidden action language to be rejected")
	}
}

func TestValidatorRejectsWeakEvidenceWithoutUncertainty(t *testing.T) {
	result := NewValidator().ValidateAnswer(DefaultPolicy(ModeResearch), EvidenceBundle{}, "This setup is valid.")
	if result.OK {
		t.Fatal("expected weak evidence without uncertainty language to be rejected")
	}
}

func TestValidatorRejectsWeakEvidenceCertaintyLanguage(t *testing.T) {
	result := NewValidator().ValidateAnswer(DefaultPolicy(ModeResearch), EvidenceBundle{}, "This will definitely work.")
	if result.OK {
		t.Fatal("expected certainty language on weak evidence to be rejected")
	}
}

func TestValidatorRejectsUnsupportedPriceTargets(t *testing.T) {
	result := NewValidator().ValidateAnswer(DefaultPolicy(ModeResearch), EvidenceBundle{}, "My price target is $125.")
	if result.OK {
		t.Fatal("expected unsupported price target to be rejected")
	}
}
