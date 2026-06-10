package macroevents

import "testing"

func TestGenerateCandidateFromAllowedBundle(t *testing.T) {
	entry := 430.25
	stop := 435.10
	target := 421.00
	candidate := GenerateCandidate(CandidateInput{
		Bundle: allowedEvidenceBundle(),
		Side:   CandidateSideShortBias,
		Plan: CandidatePlan{
			EntryType:       EntryTypePullbackRetest,
			EntryPrice:      &entry,
			StopPrice:       &stop,
			TargetPrice:     &target,
			RiskPercent:     0.5,
			TimeLimit:       "end_of_session",
			RewardRiskRatio: 1.9,
		},
	})

	if candidate.Status != MacroCandidateStatusAwaitingHumanApproval {
		t.Fatalf("status = %q, want awaiting_human_approval", candidate.Status)
	}
	if candidate.RejectionReason != "" {
		t.Fatalf("rejection reason = %q, want empty", candidate.RejectionReason)
	}
}

func TestGenerateCandidateBlockedBundleCreatesNoTradeCandidate(t *testing.T) {
	bundle := allowedEvidenceBundle()
	bundle.Verdict = EvidenceVerdictCandidateBlocked

	candidate := GenerateCandidate(CandidateInput{Bundle: bundle})

	if candidate.Status != MacroCandidateStatusBlocked {
		t.Fatalf("status = %q, want blocked", candidate.Status)
	}
	if candidate.Side != CandidateSideNoTrade {
		t.Fatalf("side = %q, want no_trade", candidate.Side)
	}
}

func TestGenerateCandidateWatchOnlyBundleCreatesWatchRecord(t *testing.T) {
	bundle := allowedEvidenceBundle()
	bundle.Verdict = EvidenceVerdictWatchOnly

	candidate := GenerateCandidate(CandidateInput{Bundle: bundle})

	if candidate.Status != MacroCandidateStatusWatchOnly {
		t.Fatalf("status = %q, want watch_only", candidate.Status)
	}
	if candidate.Side != CandidateSideWatchOnly {
		t.Fatalf("side = %q, want watch_only", candidate.Side)
	}
}

func TestGenerateCandidateMissingStopBlocksCandidate(t *testing.T) {
	entry := 430.25
	target := 421.00
	candidate := GenerateCandidate(CandidateInput{
		Bundle: allowedEvidenceBundle(),
		Side:   CandidateSideShortBias,
		Plan: CandidatePlan{
			EntryType:       EntryTypePullbackRetest,
			EntryPrice:      &entry,
			TargetPrice:     &target,
			RiskPercent:     0.5,
			TimeLimit:       "end_of_session",
			RewardRiskRatio: 1.9,
		},
	})

	if candidate.Status != MacroCandidateStatusBlocked {
		t.Fatalf("status = %q, want blocked", candidate.Status)
	}
	if candidate.RejectionReason != "stop reference price is required" {
		t.Fatalf("rejection = %q", candidate.RejectionReason)
	}
}

func TestGenerateCandidateRiskAboveLimitRejected(t *testing.T) {
	entry := 430.25
	stop := 435.10
	target := 421.00
	candidate := GenerateCandidate(CandidateInput{
		Bundle: allowedEvidenceBundle(),
		Side:   CandidateSideShortBias,
		Plan: CandidatePlan{
			EntryType:       EntryTypePullbackRetest,
			EntryPrice:      &entry,
			StopPrice:       &stop,
			TargetPrice:     &target,
			RiskPercent:     0.75,
			TimeLimit:       "end_of_session",
			RewardRiskRatio: 1.9,
		},
	})

	if candidate.Status != MacroCandidateStatusBlocked {
		t.Fatalf("status = %q, want blocked", candidate.Status)
	}
	if candidate.RejectionReason != "risk percent exceeds 0.5 limit" {
		t.Fatalf("rejection = %q", candidate.RejectionReason)
	}
}

func allowedEvidenceBundle() EvidenceBundle {
	return EvidenceBundle{
		ID:           "bundle-1",
		MacroEventID: "macro-1",
		Symbol:       "QQQ",
		Verdict:      EvidenceVerdictCandidateAllowed,
		Summary:      "Macro evidence supports a paper-only candidate subject to human approval.",
	}
}
