package advancedquant

import (
	"testing"
	"time"
)

func TestOOSAdmissionRequiresOrderedPrerequisites(t *testing.T) {
	p := testProtocol(t)
	config, err := NewExperimentConfig(ExperimentConfig{ProtocolID: p.ID, HypothesisID: p.HypothesisID, Target: "return", Baseline: "direction", Algorithm: "bounded", Parameters: map[string]string{"minimum_evidence_quality": ".8"}})
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = NewOOSAdmission(OOSAdmission{HypothesisID: config.HypothesisID, DatasetID: "dataset", ClassifierID: "classifier", ProtocolID: config.ProtocolID, DevelopmentResultID: "dev", ValidationResultID: "val", FalsificationSuiteID: "fals", CandidateID: config.ID, ProgressionRuleID: "rule", ProgressionDecisionID: "decision", ProgressionDecision: ProgressionProceedToOOS, CandidateFreezeID: config.ID, OOSPartitionID: "oos", DevelopmentCompleted: base, ValidationCompleted: base.Add(-time.Hour), FalsificationCompleted: base.Add(2 * time.Hour), DecisionAt: base.Add(3 * time.Hour), CandidateFrozenAt: base.Add(4 * time.Hour)})
	if err == nil {
		t.Fatal("out-of-order prerequisite was accepted")
	}
}

func TestFrozenOOSRunBlocksEveryMissingAdmissionStage(t *testing.T) {
	p := testProtocol(t)
	config, err := NewExperimentConfig(ExperimentConfig{ProtocolID: p.ID, HypothesisID: p.HypothesisID, Target: "return", Baseline: "direction", Algorithm: "bounded", Parameters: map[string]string{"minimum_evidence_quality": ".8"}})
	if err != nil {
		t.Fatal(err)
	}
	run := &FrozenOOSRun{}
	if err := run.Freeze(config); err != nil {
		t.Fatal(err)
	}
	if _, err := run.ScoreOOS(testObservations(t, p), p); err == nil {
		t.Fatal("OOS opened without admission")
	}
	if run.AdmissionState() != OOSSealed {
		t.Fatalf("missing admission changed state: %s", run.AdmissionState())
	}
}

func TestOOSAdmissionRejectsEachMissingPrerequisite(t *testing.T) {
	p := testProtocol(t)
	config, err := NewExperimentConfig(ExperimentConfig{ProtocolID: p.ID, HypothesisID: p.HypothesisID, Target: "return", Baseline: "direction", Algorithm: "bounded"})
	if err != nil {
		t.Fatal(err)
	}
	valid := testAdmission(t, p, config)
	cases := []struct {
		name   string
		mutate func(*OOSAdmission)
	}{
		{"development", func(a *OOSAdmission) { a.DevelopmentResultID = "" }},
		{"validation", func(a *OOSAdmission) { a.ValidationResultID = "" }},
		{"falsification", func(a *OOSAdmission) { a.FalsificationSuiteID = "" }},
		{"progression decision", func(a *OOSAdmission) { a.ProgressionDecision = "REJECT" }},
		{"candidate freeze", func(a *OOSAdmission) { a.CandidateFreezeID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			admission := valid
			tc.mutate(&admission)
			admission.ID = oosAdmissionID(admission)
			if err := admission.Validate(); err == nil {
				t.Fatalf("missing %s prerequisite was accepted", tc.name)
			}
		})
	}
}

func TestFrozenOOSRunAdmissionLifecycleIsOneWay(t *testing.T) {
	p := testProtocol(t)
	config, err := NewExperimentConfig(ExperimentConfig{ProtocolID: p.ID, HypothesisID: p.HypothesisID, Target: "return", Baseline: "direction", Algorithm: "bounded", Parameters: map[string]string{"minimum_evidence_quality": ".8"}})
	if err != nil {
		t.Fatal(err)
	}
	run := &FrozenOOSRun{}
	admission := testAdmission(t, p, config)
	if err := run.FreezeWithAdmission(config, admission); err != nil {
		t.Fatal(err)
	}
	if run.AdmissionState() != OOSAdmissible {
		t.Fatal("admission was not admissible before access")
	}
	if _, err := run.ScoreOOS(testObservations(t, p), p); err != nil {
		t.Fatal(err)
	}
	if run.AdmissionState() != OOSOpenedFormal {
		t.Fatalf("unexpected opened state: %s", run.AdmissionState())
	}
	if err := run.MarkContaminated("historical admission was not valid"); err != nil {
		t.Fatal(err)
	}
	if run.AdmissionState() != OOSContaminated {
		t.Fatal("OOS did not become contaminated")
	}
	if err := run.MarkExploratoryOnly("historical admission was not valid"); err != nil {
		t.Fatal(err)
	}
	if run.AdmissionState() != OOSExploratoryOnly {
		t.Fatal("OOS did not become exploratory-only")
	}
	if _, err := run.ScoreOOS(testObservations(t, p), p); err == nil {
		t.Fatal("exploratory OOS was scored again")
	}
	if err := run.MarkExploratoryOnly("reset"); err == nil {
		t.Fatal("exploratory state was reset")
	}
}

func TestFrozenOOSRunRejectsMismatchedAdmissionBinding(t *testing.T) {
	p := testProtocol(t)
	config, err := NewExperimentConfig(ExperimentConfig{ProtocolID: p.ID, HypothesisID: p.HypothesisID, Target: "return", Baseline: "direction", Algorithm: "bounded", Parameters: map[string]string{"minimum_evidence_quality": ".8"}})
	if err != nil {
		t.Fatal(err)
	}
	admission := testAdmission(t, p, config)
	admission.CandidateID = "different"
	admission.ID = oosAdmissionID(admission)
	run := &FrozenOOSRun{}
	if err := run.FreezeWithAdmission(config, admission); err == nil {
		t.Fatal("mismatched admission was accepted")
	}
}
