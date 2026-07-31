package eventdecisions

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSubjectAssociationIsConservativeAndDeterministic(t *testing.T) {
	now := testNow()
	first := testEvent()
	first.Headline = "Federal Reserve announces interest rate decision"
	first.Summary = "The Fed held its policy meeting and published the interest rate decision."
	first.EventType = "macro_rates"
	first.PublicationAt = now
	second := first
	second.InboxID = uuid.New()
	second.SourceEventID = "genuine-event-2"
	second.Headline = "Fed publishes interest rate decision after policy meeting"
	identityA := deriveSubjectIdentity(first)
	identityB := deriveSubjectIdentity(second)
	if identityA.Key != identityB.Key || identityA.EventScoped {
		t.Fatalf("stable topic identity did not associate: %#v %#v", identityA, identityB)
	}

	differentDay := second
	differentDay.InboxID = uuid.New()
	differentDay.PublicationAt = now.Add(24 * time.Hour)
	if deriveSubjectIdentity(differentDay).Key == identityA.Key {
		t.Fatal("separate observation date was merged into the same subject")
	}

	broadA := testEvent()
	broadA.EventType = "semiconductor_ai"
	broadA.Headline = "AI technology moves the market"
	broadA.Summary = "Stocks react to broad technology news."
	broadA.SourceURLs = []string{"https://example.com/broad-a"}
	broadB := broadA
	broadB.InboxID = uuid.New()
	broadB.SourceURLs = []string{"https://example.com/broad-b"}
	if deriveSubjectIdentity(broadA).Key == deriveSubjectIdentity(broadB).Key {
		t.Fatal("broad keyword collision was merged")
	}

	unknownA := testEvent()
	unknownA.EventType = "unknown"
	unknownB := unknownA
	unknownB.InboxID = uuid.New()
	if deriveSubjectIdentity(unknownA).Key == deriveSubjectIdentity(unknownB).Key {
		t.Fatal("unknown subjects were merged")
	}
}

func TestSourceGroupingDoesNotInflateRepeatedReports(t *testing.T) {
	event := testEvent()
	event.Headline = "Federal Reserve publishes a detailed interest rate decision for markets"
	event.Summary = "The complete policy statement describes the Federal Reserve interest rate decision and its economic projections."
	event.ArticleURL = "https://news.example/a"
	groupA, primaryA := evidenceSourceGroup(event)
	repeated := event
	repeated.InboxID = uuid.New()
	repeated.ArticleURL = "https://mirror.example/copied"
	groupB, primaryB := evidenceSourceGroup(repeated)
	if groupA != groupB || primaryA || primaryB {
		t.Fatalf("exact repeated report was not grouped as syndication: %q/%v %q/%v", groupA, primaryA, groupB, primaryB)
	}

	primaryOne := event
	primaryOne.ArticleURL = "https://www.federalreserve.gov/newsevents/pressreleases/a.htm"
	primaryOne.SourceNativeID = "fed-release-1"
	primaryTwo := event
	primaryTwo.ArticleURL = "https://www.sec.gov/newsroom/press-releases/b"
	primaryTwo.SourceNativeID = "sec-release-2"
	groupOne, officialOne := evidenceSourceGroup(primaryOne)
	groupTwo, officialTwo := evidenceSourceGroup(primaryTwo)
	if !officialOne || !officialTwo || groupOne == groupTwo {
		t.Fatalf("independent primary evidence was not distinguishable: %q/%v %q/%v", groupOne, officialOne, groupTwo, officialTwo)
	}
}

func TestSubjectEvaluationTransitionsAndCandidateThreshold(t *testing.T) {
	rules := subjectTestRules()
	now := testNow()
	watchUnknown := SubjectObservation{
		EventID: uuid.New(), Decision: DecisionWatch, UnknownAssets: true,
		PublicationAt: now.Add(-time.Hour), ReceiptAt: now, SourceGroupKey: "unknown-1", Independence: "unknown", ContradictionState: "corroborates",
	}
	result := evaluateSubject([]SubjectObservation{watchUnknown}, rules, now)
	if result.Decision != DecisionWatch || !contains(result.MissingEvidence, "truthful_asset_mapping") {
		t.Fatalf("NO_TRADE to WATCH readiness was not explained: %+v", result)
	}

	repeated := watchUnknown
	repeated.EventID = uuid.New()
	repeated.SourceGroupKey = watchUnknown.SourceGroupKey
	repeated.Independence = "not_independent"
	result = evaluateSubject([]SubjectObservation{watchUnknown, repeated}, rules, now)
	if result.Decision != DecisionWatch || result.IndependentSourceCount != 0 {
		t.Fatalf("repeated report inflated readiness: %+v", result)
	}

	candidateID := uuid.New()
	ready := watchUnknown
	ready.EventID = uuid.New()
	ready.Decision = DecisionCandidate
	ready.CandidateID = &candidateID
	ready.UnknownAssets = false
	ready.AffectedAssets = []string{"QQQ"}
	ready.SourceGroupKey = "primary-fed"
	ready.Independence = "primary"
	corroborating := ready
	corroborating.EventID = uuid.New()
	corroborating.Decision = DecisionWatch
	corroborating.CandidateID = nil
	corroborating.SourceGroupKey = "primary-sec"
	result = evaluateSubject([]SubjectObservation{ready, corroborating}, rules, now)
	if result.Decision != DecisionCandidate || result.CandidateID == nil || result.IndependentSourceCount != 2 {
		t.Fatalf("complete independent candidate evidence did not become CANDIDATE: %+v", result)
	}

	contradiction := corroborating
	contradiction.EventID = uuid.New()
	contradiction.SourceGroupKey = "primary-doj"
	contradiction.ContradictionState = "contradicts"
	result = evaluateSubject([]SubjectObservation{ready, corroborating, contradiction}, rules, now)
	if result.Decision != DecisionNoTrade || result.CandidateID != nil {
		t.Fatalf("explicit contradiction did not drive safe downward transition: %+v", result)
	}

	stale := watchUnknown
	stale.PublicationAt = now.Add(-25 * time.Hour)
	result = evaluateSubject([]SubjectObservation{stale}, rules, now)
	if result.Decision != DecisionNoTrade || !contains(result.MissingEvidence, "fresh_market_context") {
		t.Fatalf("stale WATCH did not return to NO_TRADE: %+v", result)
	}

	noTrade := watchUnknown
	noTrade.Decision = DecisionNoTrade
	result = evaluateSubject([]SubjectObservation{noTrade}, rules, now)
	if result.Decision != DecisionNoTrade {
		t.Fatalf("NO_TRADE did not remain NO_TRADE: %+v", result)
	}
}

func subjectTestRules() Ruleset {
	return Ruleset{
		Version: "genuine-event-decision-v1", ProcessorIdentity: "test",
		WatchConfidenceMinimum: 0.5, CandidateEvidenceMinimum: 0.6,
		SubjectRulesetVersion: "genuine-watch-evidence-v1", SubjectCandidateIndependentMin: 2, SubjectFreshnessHours: 24,
		AllowedCandidateInstrumentType: "etf", MaximumLeverage: 1, MaterialSeverities: []string{"medium", "high", "critical"},
	}
}
