package harness

import "encoding/json"

type EvidenceItem struct {
	SourceTool    string          `json:"sourceTool"`
	EvidenceLevel EvidenceLevel   `json:"evidenceLevel"`
	Freshness     string          `json:"freshness"`
	Summary       string          `json:"summary"`
	Raw           json.RawMessage `json:"raw"`
}

type EvidenceBundle struct {
	Items []EvidenceItem `json:"items"`
}

func (b *EvidenceBundle) Add(item EvidenceItem) {
	b.Items = append(b.Items, item)
}

func (b *EvidenceBundle) IsWeak() bool {
	if len(b.Items) == 0 {
		return true
	}
	for _, it := range b.Items {
		if it.EvidenceLevel == EvidenceHardInternal {
			return false
		}
	}
	return true
}
