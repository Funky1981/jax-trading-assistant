package workflow

import "testing"

func TestPermissionMatrixFailsClosedForUnknownAndAgentActions(t *testing.T) {
	allowed := []struct {
		role     ActorRole
		action   Action
		from, to State
	}{
		{ActorSystem, ActionWorkflowCreated, State(""), StateRiskAccepted},
		{ActorSystem, ActionRequestConfirmation, StateRiskAmended, StateAwaitingHumanConfirmation},
		{ActorHuman, ActionHumanApprove, StateAwaitingHumanConfirmation, StateHumanApproved},
		{ActorHuman, ActionHumanReject, StateAwaitingHumanConfirmation, StateHumanRejected},
		{ActorSystem, ActionCreatePaperIntent, StateHumanApproved, StatePaperIntentCreated},
	}
	for _, test := range allowed {
		if !PermissionAllowed(test.role, test.action, test.from, test.to) {
			t.Errorf("expected allowed permission: %#v", test)
		}
	}
	denied := []struct {
		role     ActorRole
		action   Action
		from, to State
	}{
		{ActorResearcher, ActionHumanApprove, StateAwaitingHumanConfirmation, StateHumanApproved},
		{ActorResearcher, ActionCreatePaperIntent, StateHumanApproved, StatePaperIntentCreated},
		{ActorSystem, ActionHumanApprove, StateAwaitingHumanConfirmation, StateHumanApproved},
		{ActorHuman, ActionCreatePaperIntent, StateHumanApproved, StatePaperIntentCreated},
		{ActorOperator, ActionHumanApprove, StateAwaitingHumanConfirmation, StateHumanApproved},
		{ActorSystem, Action("ARBITRARY_STATE_JUMP"), StateRiskAccepted, StateHumanApproved},
		{ActorRole("FORGED"), ActionHumanApprove, StateAwaitingHumanConfirmation, StateHumanApproved},
	}
	for _, test := range denied {
		if PermissionAllowed(test.role, test.action, test.from, test.to) {
			t.Errorf("expected denied permission: %#v", test)
		}
	}
}
