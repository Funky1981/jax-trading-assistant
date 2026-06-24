package paper

type ApprovalStatus string

const (
	ApprovalPendingReview    ApprovalStatus = "PENDING_REVIEW"
	ApprovalApprovedForPaper ApprovalStatus = "APPROVED_FOR_PAPER"
	ApprovalRejectedByUser   ApprovalStatus = "REJECTED_BY_USER"
	ApprovalDeferred         ApprovalStatus = "DEFERRED"
	ApprovalExpired          ApprovalStatus = "EXPIRED"
)

type LifecycleState string

const (
	LifecycleCreated          LifecycleState = "CREATED"
	LifecyclePendingReview    LifecycleState = "PENDING_REVIEW"
	LifecycleApprovedForPaper LifecycleState = "APPROVED_FOR_PAPER"
	LifecycleRejectedByUser   LifecycleState = "REJECTED_BY_USER"
	LifecycleDeferred         LifecycleState = "DEFERRED"
	LifecycleExpired          LifecycleState = "EXPIRED"
	LifecycleCancelled        LifecycleState = "CANCELLED"
)

func isTerminalStatus(status ApprovalStatus) bool {
	return status == ApprovalApprovedForPaper ||
		status == ApprovalRejectedByUser ||
		status == ApprovalExpired
}
