package harness

import "strings"

func containsInsensitive(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

func evidenceLevelFromString(raw string) EvidenceLevel {
	switch EvidenceLevel(raw) {
	case EvidenceHardInternal:
		return EvidenceHardInternal
	case EvidenceDerivedInternal:
		return EvidenceDerivedInternal
	default:
		return EvidenceWeakInference
	}
}
