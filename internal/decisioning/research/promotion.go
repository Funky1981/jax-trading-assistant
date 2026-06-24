package research

type PromotionState string

const (
	PromotionIdea                PromotionState = "IDEA"
	PromotionHypothesis          PromotionState = "HYPOTHESIS"
	PromotionBacktestedWeak      PromotionState = "BACKTESTED_WEAK"
	PromotionBacktestedPromising PromotionState = "BACKTESTED_PROMISING"
	PromotionPaperReady          PromotionState = "PAPER_READY"
	PromotionPaperRejected       PromotionState = "PAPER_REJECTED"
	PromotionPaperProven         PromotionState = "PAPER_PROVEN"
)

const (
	disallowedLiveReady PromotionState = "LIVE_READY"
)

var promotionRank = map[PromotionState]int{
	PromotionIdea:                0,
	PromotionHypothesis:          1,
	PromotionBacktestedWeak:      2,
	PromotionBacktestedPromising: 3,
	PromotionPaperReady:          4,
	PromotionPaperRejected:       5,
	PromotionPaperProven:         6,
}

func IsAllowedPromotionState(state PromotionState) bool {
	_, ok := promotionRank[state]
	return ok
}

func capPromotion(requested PromotionState, maxAllowed PromotionState) PromotionState {
	if requested == "" {
		return maxAllowed
	}
	if !IsAllowedPromotionState(requested) {
		return maxAllowed
	}
	if promotionRank[requested] > promotionRank[maxAllowed] {
		return maxAllowed
	}
	return requested
}
