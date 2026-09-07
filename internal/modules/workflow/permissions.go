package workflow

// PermissionAllowed is the deterministic least-privilege boundary for
// protected Phase-10 actions. Unknown role/action/state combinations deny.
func PermissionAllowed(role ActorRole, action Action, from, to State) bool {
	switch {
	case role == ActorSystem && action == ActionWorkflowCreated && from == State("") && (to == StateRiskAccepted || to == StateRiskAmended):
		return true
	case role == ActorSystem && action == ActionRequestConfirmation && (from == StateRiskAccepted || from == StateRiskAmended) && to == StateAwaitingHumanConfirmation:
		return true
	case role == ActorHuman && action == ActionHumanApprove && from == StateAwaitingHumanConfirmation && to == StateHumanApproved:
		return true
	case role == ActorHuman && action == ActionHumanReject && from == StateAwaitingHumanConfirmation && to == StateHumanRejected:
		return true
	case role == ActorSystem && action == ActionCreatePaperIntent && from == StateHumanApproved && to == StatePaperIntentCreated:
		return true
	case role == ActorHuman && action == ActionCancel && from != StatePaperIntentCreated && to == StateCancelled:
		return true
	case role == ActorOperator && action == ActionBlock && from != StatePaperIntentCreated && to == StateBlocked:
		return true
	case role == ActorSystem && action == ActionFail && from != StatePaperIntentCreated && to == StateFailed:
		return true
	case role == ActorRecovery && action == ActionRequireReconciliation && from != StatePaperIntentCreated && to == StateReconciliationRequired:
		return true
	default:
		return false
	}
}
