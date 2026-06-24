package swing

type SetupFamily string

const (
	EventDrivenPullbackContinuation  SetupFamily = "EVENT_DRIVEN_PULLBACK_CONTINUATION"
	PostEarningsDriftContinuation    SetupFamily = "POST_EARNINGS_DRIFT_CONTINUATION"
	CommodityLinkedEquityDislocation SetupFamily = "COMMODITY_LINKED_EQUITY_DISLOCATION"
	SectorRelativeRepricing          SetupFamily = "SECTOR_RELATIVE_REPRICING"
	IndexHeavyweightDistortionWatch  SetupFamily = "INDEX_HEAVYWEIGHT_DISTORTION_WATCH"
	UnknownSwingSetup                SetupFamily = "UNKNOWN_SWING_SETUP"
)

func normaliseSetupFamily(family SetupFamily) SetupFamily {
	switch family {
	case EventDrivenPullbackContinuation,
		PostEarningsDriftContinuation,
		CommodityLinkedEquityDislocation,
		SectorRelativeRepricing,
		IndexHeavyweightDistortionWatch:
		return family
	default:
		return UnknownSwingSetup
	}
}
