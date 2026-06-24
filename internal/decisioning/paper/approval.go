package paper

import "time"

type ApprovalRequest struct {
	Ticket                PaperTicket `json:"ticket"`
	ApprovedBy            string      `json:"approved_by"`
	ApprovedAt            time.Time   `json:"approved_at"`
	Now                   time.Time   `json:"now"`
	ExplicitHumanApproval bool        `json:"explicit_human_approval"`
	AutomaticApproval     bool        `json:"automatic_approval"`
}

type ApprovalResult struct {
	Ticket              PaperTicket      `json:"ticket"`
	Validation          ValidationResult `json:"validation"`
	HumanApprovalStatus ApprovalStatus   `json:"human_approval_status"`
	LifecycleState      LifecycleState   `json:"lifecycle_state"`
	CanApproveForPaper  bool             `json:"can_approve_for_paper"`
}

func ApproveForPaper(request ApprovalRequest) ApprovalResult {
	validation := ValidateApprovalRequest(request)
	ticket := request.Ticket
	if !validation.CanApproveForPaper {
		return ApprovalResult{
			Ticket:              ticket,
			Validation:          validation,
			HumanApprovalStatus: ticket.HumanApprovalStatus,
			LifecycleState:      ticket.LifecycleState,
			CanApproveForPaper:  false,
		}
	}

	ticket.HumanApprovalStatus = ApprovalApprovedForPaper
	ticket.LifecycleState = LifecycleApprovedForPaper
	return ApprovalResult{
		Ticket:              ticket,
		Validation:          validation,
		HumanApprovalStatus: ApprovalApprovedForPaper,
		LifecycleState:      LifecycleApprovedForPaper,
		CanApproveForPaper:  true,
	}
}
