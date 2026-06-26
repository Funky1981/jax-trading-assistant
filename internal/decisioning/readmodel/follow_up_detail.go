package readmodel

import "jax-trading-assistant/internal/decisioning/operations"

type FollowUpActionDetail = operations.FollowUpAction

func BuildFollowUpActionDetail(action operations.FollowUpAction) (FollowUpActionDetail, error) {
	copied := action
	copied.ForbiddenActions = append([]string{}, action.ForbiddenActions...)
	copied.RequiresHumanApproval = true
	copied.AutoApplyAllowed = false
	if err := operations.ValidateFollowUpAction(copied); err != nil {
		return operations.FollowUpAction{}, err
	}
	return copied, nil
}
