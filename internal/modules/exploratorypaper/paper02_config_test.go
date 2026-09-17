package exploratorypaper

import (
	"testing"
	"time"
)

func paper02CalendarFixture() VersionedSessionCalendar {
	return VersionedSessionCalendar{
		Version: "us-equities-test-v1", Market: "US_EQUITIES_REGULAR", Timezone: "America/New_York",
		OpenTime: "09:30", CloseTime: "16:00", CoverageStart: "2026-09-17", CoverageEnd: "2027-01-31",
		WeekendDays: []int{0, 6}, Holidays: []string{"2026-11-26"},
		SpecialWindows: map[string]SessionWindow{"2026-11-27": {OpenTime: "09:30", CloseTime: "13:00"}},
		Provenance:     "deterministic test manifest", ApplicableMode: "PAPER",
	}
}

func paper02UniverseFixture() EligibleUniverseManifest {
	return EligibleUniverseManifest{
		Version: "paper-02-test-v1", EffectiveFrom: "2026-09-17", Market: "US_EQUITIES_REGULAR", ApplicableMode: "PAPER", Provenance: "test",
		Instruments: []EligibleInstrument{{InstrumentID: "NASDAQ:AAPL", Symbol: "AAPL", Exchange: "NASDAQ", Market: "US_EQUITIES_REGULAR"}},
	}
}

func paper02ReadinessFixture() Paper02ReadinessInput {
	calendar := paper02CalendarFixture()
	universe := paper02UniverseFixture()
	risk := &RiskPolicyIdentity{Version: "v-risk", Hash: "sha256:risk", MaxRiskPerTrade: .01, MaxPositionPercentage: .20, MaxLeverage: 1, MaxConcurrentPositions: 10, MaxPositionValue: 50000, MaxAggregateRisk: .15, MaxConcentration: .30, MaxCorrelatedExposure: .40, RiskKillBehavior: "fail-closed"}
	entry := &EntryPolicyIdentity{Version: "paper-02-entry-v1", Hash: "sha256:entry", Mode: "PAPER", CandidateRequirement: "CANDIDATE", EvidencePolicy: "candidate-evidence-scoring-v1", RiskAcceptance: "deterministic", HumanApproval: "explicit", PaperAcknowledgement: "EXPLORATORY_PAPER_ONLY_NO_LIVE_NO_FORMAL", ThesisContract: ContractVersion, SimulatedVenue: "jax.paper.venue/v1", ExecutionAuthority: "NONE", MaximumLeverage: 1, BoundFields: []string{"candidate", "thesis", "risk", "approval"}}
	start := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	return Paper02ReadinessInput{Calendar: &calendar, EventSource: ProspectiveEventSourceReadiness{Ready: true, SourceIdentity: "test-source-v1"}, Universe: &universe, RiskPolicy: risk, EntryPolicy: entry, RuntimeMode: "PAPER", ExecutionAuthority: "NONE", MaximumLeverage: 1, PilotStartTimestamp: &start, MaximumDurationDays: 90}
}

func TestVersionedCalendarSessionBoundariesAndCoverage(t *testing.T) {
	manifest := paper02CalendarFixture()
	calendar, err := manifest.SessionCalendar()
	if err != nil {
		t.Fatal(err)
	}
	location, _ := time.LoadLocation("America/New_York")
	cases := []struct {
		name    string
		at      time.Time
		want    MarketSession
		unknown bool
	}{
		{"open", time.Date(2026, 9, 17, 9, 30, 0, 0, location), SessionOpen, false},
		{"before open", time.Date(2026, 9, 17, 9, 29, 0, 0, location), SessionClosed, false},
		{"after close", time.Date(2026, 9, 17, 16, 1, 0, 0, location), SessionClosed, false},
		{"weekend", time.Date(2026, 9, 19, 12, 0, 0, 0, location), SessionClosed, false},
		{"October bank holiday is open", time.Date(2026, 10, 12, 12, 0, 0, 0, location), SessionOpen, false},
		{"Veterans bank holiday is open", time.Date(2026, 11, 11, 12, 0, 0, 0, location), SessionOpen, false},
		{"Thanksgiving", time.Date(2026, 11, 26, 12, 0, 0, 0, location), SessionClosed, false},
		{"unsupported", time.Date(2027, 2, 1, 12, 0, 0, 0, location), SessionUnknown, true},
		{"DST open", time.Date(2026, 11, 2, 9, 30, 0, 0, location), SessionOpen, false},
		{"special close before 13:00", time.Date(2026, 11, 27, 12, 59, 0, 0, location), SessionOpen, false},
		{"special close after 13:00", time.Date(2026, 11, 27, 13, 1, 0, 0, location), SessionClosed, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := calendar.SessionState(tc.at.UTC())
			if tc.unknown {
				if err == nil || got != SessionUnknown {
					t.Fatalf("expected unknown session, got %s, err=%v", got, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
	hash := manifest.ContentHash()
	if hash != manifest.ContentHash() {
		t.Fatal("calendar content hash is not stable")
	}
	if manifest.Version == "" {
		t.Fatal("calendar version is required")
	}
}

func TestVersionedCalendarRejectsInvalidOrUnsupportedManifest(t *testing.T) {
	manifest := paper02CalendarFixture()
	manifest.ApplicableMode = "LIVE"
	if err := manifest.Validate(); err == nil {
		t.Fatal("expected non-PAPER calendar to fail")
	}
	manifest = paper02CalendarFixture()
	manifest.CoverageEnd = "not-a-date"
	if err := manifest.Validate(); err == nil {
		t.Fatal("expected invalid coverage to fail")
	}
}

func TestEligibleUniverseIdentityAndCompatibility(t *testing.T) {
	manifest := paper02UniverseFixture()
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	hash := manifest.ContentHash()
	if hash != manifest.ContentHash() {
		t.Fatal("universe hash is not stable")
	}
	changed := manifest
	changed.Instruments = append(changed.Instruments, EligibleInstrument{InstrumentID: "NYSE:IBM", Symbol: "IBM", Exchange: "NYSE", Market: manifest.Market})
	if manifest.ContentHash() == changed.ContentHash() {
		t.Fatal("constituent change did not change hash")
	}
	duplicate := manifest
	duplicate.Instruments = append(duplicate.Instruments, manifest.Instruments[0])
	if err := duplicate.Validate(); err == nil {
		t.Fatal("duplicate constituent must fail")
	}
	unsupported := manifest
	unsupported.Instruments[0].Exchange = "OTC"
	if err := unsupported.Validate(); err == nil {
		t.Fatal("unsupported instrument must fail")
	}
}

func TestPaper02ReadinessBlocksEachPrerequisiteAndRequiresPaperSafety(t *testing.T) {
	base := paper02ReadinessFixture()
	if got := AssessPaper02Readiness(base); !got.OverallReady {
		t.Fatalf("valid readiness blocked: %#v", got.BlockingReasons)
	}
	tests := []struct {
		name   string
		mutate func(*Paper02ReadinessInput)
	}{
		{"missing calendar", func(i *Paper02ReadinessInput) { i.Calendar = nil }},
		{"invalid calendar", func(i *Paper02ReadinessInput) {
			c := paper02CalendarFixture()
			c.ApplicableMode = "LIVE"
			i.Calendar = &c
		}},
		{"missing event source", func(i *Paper02ReadinessInput) {
			i.EventSource = ProspectiveEventSourceReadiness{SourceIdentity: "source"}
		}},
		{"missing universe", func(i *Paper02ReadinessInput) { i.Universe = nil }},
		{"missing risk policy", func(i *Paper02ReadinessInput) { i.RiskPolicy = nil }},
		{"missing entry policy", func(i *Paper02ReadinessInput) { i.EntryPolicy = nil }},
		{"wrong mode", func(i *Paper02ReadinessInput) { i.RuntimeMode = "LIVE" }},
		{"wrong authority", func(i *Paper02ReadinessInput) { i.ExecutionAuthority = "BROKER" }},
		{"broker enabled", func(i *Paper02ReadinessInput) { i.BrokerExecutionAllowed = true }},
		{"leverage too high", func(i *Paper02ReadinessInput) { i.MaximumLeverage = 2 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := base
			tc.mutate(&input)
			if result := AssessPaper02Readiness(input); result.OverallReady {
				t.Fatal("expected readiness to be blocked")
			}
		})
	}
}

func TestPaper02ReadinessIsPureAndDoesNotCreateRuntimeState(t *testing.T) {
	result := AssessPaper02Readiness(paper02ReadinessFixture())
	if !result.OverallReady || len(result.BlockingReasons) != 0 {
		t.Fatalf("unexpected readiness: %#v", result)
	}
	if result.CalendarVersion == "" || result.EligibleUniverseHash == "" || result.RiskPolicyHash == "" || result.EntryPolicyHash == "" {
		t.Fatal("readiness did not expose frozen identities")
	}
}

func TestPaper02ReadinessEnforcesMaximumPilotCalendarCoverage(t *testing.T) {
	input := paper02ReadinessFixture()
	input.PilotStartTimestamp = func() *time.Time { value := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC); return &value }()
	blocked := AssessPaper02Readiness(input)
	if blocked.CalendarCoverageReady || blocked.OverallReady {
		t.Fatalf("pilot deadline beyond calendar coverage was accepted: %+v", blocked)
	}
	input.PilotStartTimestamp = func() *time.Time { value := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC); return &value }()
	ready := AssessPaper02Readiness(input)
	if !ready.CalendarCoverageReady || !ready.CalendarReady || ready.CalendarDeadline != "2026-12-16" {
		t.Fatalf("sufficient calendar coverage was rejected: %+v", ready)
	}
}
