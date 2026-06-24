package triage

type Priority string

const (
	PriorityLow      Priority = "LOW"
	PriorityMedium   Priority = "MEDIUM"
	PriorityHigh     Priority = "HIGH"
	PriorityCritical Priority = "CRITICAL"
)

func DefaultPriority(source SourceType) Priority {
	switch source {
	case SourceRiskVetoTooStrict, SourcePaperSetupFailed, SourceDataQualityReview:
		return PriorityHigh
	case SourceMissedOpportunity:
		return PriorityHigh
	case SourceRiskVetoHelped:
		return PriorityLow
	case SourcePaperSetupWorked:
		return PriorityMedium
	default:
		return PriorityMedium
	}
}

func priorityRank(priority Priority) int {
	switch priority {
	case PriorityCritical:
		return 4
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 0
	}
}
