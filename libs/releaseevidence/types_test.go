package releaseevidence

import (
	"testing"
	"time"
)

func TestDateOnlyScheduleCannotFabricatePublicationTime(t *testing.T) {
	date := Date("2026-09-10")
	received := time.Date(2026, 9, 11, 15, 4, 5, 0, time.UTC)
	if err := date.Validate(); err != nil {
		t.Fatal(err)
	}
	if received.Equal(time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("acquisition timestamp unexpectedly equals fabricated date midnight")
	}
	if got := TimingDateOnlySchedule; got != "DATE_ONLY_SCHEDULE" {
		t.Fatalf("timing authority = %q", got)
	}
}

func TestActualReleaseInstantRequiresActualAuthority(t *testing.T) {
	actual := time.Date(2026, 9, 10, 12, 30, 0, 0, time.UTC)
	value := EconomicRelease{TimingAuthority: TimingScheduledInstant, ActualReleaseInstant: &actual}
	if err := value.Validate(); err == nil {
		t.Fatal("scheduled timing accepted an actual-release timestamp")
	}
}
