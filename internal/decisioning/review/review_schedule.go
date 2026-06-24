package review

import "time"

const (
	ReviewWindow1Day   = "1_day"
	ReviewWindow1Week  = "1_week"
	ReviewWindow1Month = "1_month"
)

type ReviewStatus string

const (
	ReviewStatusScheduled ReviewStatus = "SCHEDULED"
	ReviewStatusDue       ReviewStatus = "DUE"
	ReviewStatusCompleted ReviewStatus = "COMPLETED"
	ReviewStatusCancelled ReviewStatus = "CANCELLED"
)

type ReviewSchedule struct {
	ScheduleID    string       `json:"schedule_id"`
	DecisionID    string       `json:"decision_id"`
	ReviewWindows []string     `json:"review_windows"`
	NextReviewAt  time.Time    `json:"next_review_at"`
	ReviewStatus  ReviewStatus `json:"review_status"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

func DefaultReviewWindows() []string {
	return []string{ReviewWindow1Day, ReviewWindow1Week, ReviewWindow1Month}
}

func NewReviewSchedule(decisionID string, createdAt time.Time) ReviewSchedule {
	return ReviewSchedule{
		ScheduleID:    "sched_" + decisionID,
		DecisionID:    decisionID,
		ReviewWindows: DefaultReviewWindows(),
		NextReviewAt:  createdAt.Add(24 * time.Hour),
		ReviewStatus:  ReviewStatusScheduled,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
}

func ReviewWindowAllowed(window string) bool {
	for _, allowed := range DefaultReviewWindows() {
		if window == allowed {
			return true
		}
	}
	return false
}
