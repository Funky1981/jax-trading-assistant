package macroevents

import "testing"

func TestLookupEventPlaybookSelectsBankStressPlaybook(t *testing.T) {
	playbook, ok := LookupEventPlaybook(EventInput{
		EventType: EventTypeUSCPIHeadline,
		Headline:  "Regional bank stress rattles credit markets",
		Direction: DirectionRiskOff,
	})
	if !ok {
		t.Fatal("expected playbook match")
	}
	if playbook.Key != "bank_stress" {
		t.Fatalf("playbook = %q, want bank_stress", playbook.Key)
	}
	if playbook.ScenarioKey != ScenarioBankStress {
		t.Fatalf("scenario key = %q, want %q", playbook.ScenarioKey, ScenarioBankStress)
	}
	if len(playbook.PrimarySymbols) == 0 || playbook.PrimarySymbols[0] != "XLF" {
		t.Fatalf("primary symbols = %#v, want XLF first", playbook.PrimarySymbols)
	}
}

func TestLookupEventPlaybookSelectsOilShockPlaybook(t *testing.T) {
	playbook, ok := LookupEventPlaybook(EventInput{
		EventType: EventTypeUSCPIHeadline,
		Headline:  "Oil shock sends crude higher",
		Direction: DirectionRiskOff,
	})
	if !ok {
		t.Fatal("expected playbook match")
	}
	if playbook.Key != "oil_shock" {
		t.Fatalf("playbook = %q, want oil_shock", playbook.Key)
	}
	if playbook.ScenarioKey != ScenarioOilShock {
		t.Fatalf("scenario key = %q, want %q", playbook.ScenarioKey, ScenarioOilShock)
	}
}

func TestLookupEventPlaybookSelectsSemiconductorPlaybook(t *testing.T) {
	playbook, ok := LookupEventPlaybook(EventInput{
		EventType: EventTypeUSCPIHeadline,
		Headline:  "Nvidia AI demand boosts semiconductor sentiment",
		Direction: DirectionRiskOn,
	})
	if !ok {
		t.Fatal("expected playbook match")
	}
	if playbook.Key != "mega_cap_ai_semiconductor" {
		t.Fatalf("playbook = %q, want mega_cap_ai_semiconductor", playbook.Key)
	}
	if playbook.ScenarioKey != ScenarioSemiconductorAI {
		t.Fatalf("scenario key = %q, want %q", playbook.ScenarioKey, ScenarioSemiconductorAI)
	}
}
