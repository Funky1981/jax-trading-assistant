// Package strategyregistry provides read access to approved strategy documents
// stored in the jax_knowledge PostgreSQL database.
package strategyregistry

import (
	"time"

	"github.com/google/uuid"
)

// Document represents a strategy document from the strategy_documents table.
type Document struct {
	DocID      uuid.UUID `json:"doc_id"`
	DocType    string    `json:"doc_type"`
	RelPath    string    `json:"rel_path"`
	Title      string    `json:"title"`
	Version    string    `json:"version"`
	Status     string    `json:"status"`
	CreatedUTC time.Time `json:"created_utc"`
	UpdatedUTC time.Time `json:"updated_utc"`
	Tags       []string  `json:"tags"`
	Sha256     string    `json:"sha256"`
	Markdown   string    `json:"markdown"`
}

// DocType constants matching the ingestion pipeline.
const (
	DocTypeStrategy    = "strategy"
	DocTypePattern     = "pattern"
	DocTypeAntiPattern = "anti-pattern"
	DocTypeMeta        = "meta"
	DocTypeRisk        = "risk"
	DocTypeEvaluation  = "evaluation"
	DocTypeUnknown     = "unknown"
)

// Status constants - only "approved" documents are returned by queries.
const (
	StatusApproved  = "approved"
	StatusCandidate = "candidate"
	StatusRetired   = "retired"
	StatusDraft     = "draft"
)
