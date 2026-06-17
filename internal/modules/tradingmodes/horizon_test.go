package tradingmodes

import "testing"

func TestIntradayHorizonRejectsOvernightRisk(t *testing.T) {
	policy := IntradayHorizonPolicy()
	policy.OvernightRiskAllowed = true

	if err := policy.Validate(); err == nil {
		t.Fatal("expected intraday horizon to reject overnight risk")
	}
}

func TestSwingHorizonRequiresDailyReview(t *testing.T) {
	policy := SwingHorizonPolicy(3, 10)
	policy.RequiresDailyReview = false

	if err := policy.Validate(); err == nil {
		t.Fatal("expected swing horizon to require daily review")
	}
}

func TestSwingHorizonRejectsMaxHoldAboveTenDays(t *testing.T) {
	policy := SwingHorizonPolicy(3, 11)

	if err := policy.Validate(); err == nil {
		t.Fatal("expected swing horizon to reject max hold days above 10")
	}
}

func TestSwingHorizonDefaultsToNoWeekendHold(t *testing.T) {
	policy := SwingHorizonPolicy(3, 10)

	if policy.WeekendHoldAllowed {
		t.Fatal("expected swing horizon to default to no weekend hold")
	}
	if err := policy.Validate(); err != nil {
		t.Fatalf("default swing horizon should validate: %v", err)
	}
}
