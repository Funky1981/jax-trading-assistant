package risk

const (
	defaultMaxCurrentExposurePct    = 0.50
	defaultMaxSectorExposurePct     = 0.25
	defaultMaxCorrelatedExposurePct = 0.30
)

type Exposure struct {
	Name            string   `json:"name,omitempty"`
	PercentOfEquity float64  `json:"percent_of_equity"`
	Notes           []string `json:"notes,omitempty"`
}

func (exposure Exposure) above(threshold float64) bool {
	return exposure.PercentOfEquity > threshold
}
