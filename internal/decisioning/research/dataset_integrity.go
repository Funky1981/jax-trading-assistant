package research

type DatasetIntegrityCheck struct {
	DatasetID          string    `json:"dataset_id"`
	DatasetHash        string    `json:"dataset_hash"`
	DateRange          DateRange `json:"date_range"`
	InstrumentUniverse []string  `json:"instrument_universe"`
	KnownLimitations   []string  `json:"known_limitations,omitempty"`
}

func ValidateDatasetIntegrity(check DatasetIntegrityCheck) []string {
	var errors []string
	if check.DatasetID == "" {
		errors = append(errors, "dataset_id is required")
	}
	if check.DatasetHash == "" {
		errors = append(errors, "dataset_hash is required")
	}
	if !check.DateRange.IsDefined() {
		errors = append(errors, "date_range is required")
	}
	if len(check.InstrumentUniverse) == 0 {
		errors = append(errors, "instrument_universe is required")
	}
	return errors
}
