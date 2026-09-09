package advancedquant

import "testing"

func TestPopulationSummarySeparatesTypedExclusionsAndFlagsSubsetBias(t *testing.T) {
	summary, err := BuildPopulationSummary([]PopulationRecord{
		{EventID: "evt-1", IssuerID: "issuer-1", Partition: PartitionDevelopment, ClassifierState: "POSITIVE", Classified: true, Directional: true, EvidenceConditioned: true, ReturnEligible: true, AnalysisIncluded: true},
		{EventID: "evt-2", IssuerID: "issuer-2", Partition: PartitionDevelopment, ClassifierState: "NEUTRAL", Classified: true, ExclusionReasons: []ExclusionReason{ExclusionNonDirectionalNeutral}},
		{EventID: "evt-3", IssuerID: "issuer-3", Partition: PartitionValidation, ClassifierState: "INSUFFICIENT_EVIDENCE", Classified: true, ExclusionReasons: []ExclusionReason{ExclusionClassifierAbstention}},
		{EventID: "evt-4", IssuerID: "issuer-4", Partition: PartitionValidation, ClassifierState: "NEGATIVE", Classified: true, Directional: true, ReturnEligible: false, ExclusionReasons: []ExclusionReason{ExclusionMarketUnavailable}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Counts.SourceEvents != 4 || summary.Counts.DirectionalEvents != 2 || summary.Counts.EvidenceConditioned != 1 || summary.Counts.ReturnEligibleEvents != 1 {
		t.Fatalf("unexpected counts: %+v", summary.Counts)
	}
	if summary.ExclusionCounts[ExclusionNonDirectionalNeutral] != 1 || summary.ExclusionCounts[ExclusionMarketUnavailable] != 1 {
		t.Fatalf("typed exclusions missing: %+v", summary.ExclusionCounts)
	}
	if !summary.SelectionBiasReviewRequired {
		t.Fatal("subset selection did not require bias review")
	}
	if err := summary.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPopulationRejectsGenericOrInconsistentExclusion(t *testing.T) {
	if _, err := BuildPopulationSummary([]PopulationRecord{{EventID: "evt", IssuerID: "issuer", Partition: PartitionValidation, ClassifierState: "NEUTRAL", Classified: true, ExclusionReasons: []ExclusionReason{ExclusionOther, ExclusionOther}}}); err == nil {
		t.Fatal("duplicate exclusion was accepted")
	}
	if _, err := BuildPopulationSummary([]PopulationRecord{{EventID: "evt", IssuerID: "issuer", Partition: PartitionValidation, ClassifierState: "POSITIVE", Directional: true, EvidenceConditioned: true}}); err == nil {
		t.Fatal("evidence-conditioned record without classification was accepted")
	}
}
