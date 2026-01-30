package strategyregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Registry provides read access to approved strategy documents.
type Registry struct {
	pool *pgxpool.Pool
}

// New creates a new Registry with the given connection pool.
func New(pool *pgxpool.Pool) *Registry {
	return &Registry{pool: pool}
}

// NewFromDSN creates a new Registry by connecting to the given DSN.
func NewFromDSN(ctx context.Context, dsn string) (*Registry, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("strategyregistry: connect: %w", err)
	}
	return &Registry{pool: pool}, nil
}

// Close closes the underlying connection pool.
func (r *Registry) Close() {
	if r.pool != nil {
		r.pool.Close()
	}
}

// HealthCheck verifies connectivity to the database.
func (r *Registry) HealthCheck(ctx context.Context) error {
	var n int
	err := r.pool.QueryRow(ctx, "SELECT 1").Scan(&n)
	if err != nil {
		return fmt.Errorf("strategyregistry: health check failed: %w", err)
	}
	return nil
}

// base SQL for all queries - columns in order
const baseSelect = `
SELECT doc_id, doc_type, rel_path, title, version, status, 
       created_utc, updated_utc, tags, sha256, markdown
FROM strategy_documents
`

// GetApprovedStrategies returns all approved strategy documents.
func (r *Registry) GetApprovedStrategies(ctx context.Context) ([]Document, error) {
	return r.queryByType(ctx, DocTypeStrategy)
}

// GetAntiPatterns returns all approved anti-pattern documents.
func (r *Registry) GetAntiPatterns(ctx context.Context) ([]Document, error) {
	return r.queryByType(ctx, DocTypeAntiPattern)
}

// GetPatterns returns all approved pattern documents.
func (r *Registry) GetPatterns(ctx context.Context) ([]Document, error) {
	return r.queryByType(ctx, DocTypePattern)
}

// GetMetaDocs returns all approved meta documents.
func (r *Registry) GetMetaDocs(ctx context.Context) ([]Document, error) {
	return r.queryByType(ctx, DocTypeMeta)
}

// GetRiskDocs returns all approved risk documents.
func (r *Registry) GetRiskDocs(ctx context.Context) ([]Document, error) {
	return r.queryByType(ctx, DocTypeRisk)
}

// GetEvaluationDocs returns all approved evaluation documents.
func (r *Registry) GetEvaluationDocs(ctx context.Context) ([]Document, error) {
	return r.queryByType(ctx, DocTypeEvaluation)
}

// GetByRelPath returns a single approved document by its relative path.
func (r *Registry) GetByRelPath(ctx context.Context, relPath string) (Document, error) {
	query := baseSelect + `WHERE rel_path = $1 AND status = 'approved'`

	row := r.pool.QueryRow(ctx, query, relPath)
	doc, err := scanDocument(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, fmt.Errorf("strategyregistry: document not found: %s", relPath)
	}
	if err != nil {
		return Document{}, fmt.Errorf("strategyregistry: query by path: %w", err)
	}
	return doc, nil
}

// GetByID returns a single approved document by its UUID.
func (r *Registry) GetByID(ctx context.Context, id uuid.UUID) (Document, error) {
	query := baseSelect + `WHERE doc_id = $1 AND status = 'approved'`

	row := r.pool.QueryRow(ctx, query, id)
	doc, err := scanDocument(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, fmt.Errorf("strategyregistry: document not found: %s", id)
	}
	if err != nil {
		return Document{}, fmt.Errorf("strategyregistry: query by id: %w", err)
	}
	return doc, nil
}

// GetAll returns all approved documents.
func (r *Registry) GetAll(ctx context.Context) ([]Document, error) {
	query := baseSelect + `WHERE status = 'approved' ORDER BY doc_type, rel_path`
	return r.queryDocs(ctx, query)
}

// CountsByType returns counts of approved documents grouped by doc_type.
func (r *Registry) CountsByType(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT doc_type, COUNT(*) 
		FROM strategy_documents 
		WHERE status = 'approved' 
		GROUP BY doc_type
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("strategyregistry: count query: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var docType string
		var count int
		if err := rows.Scan(&docType, &count); err != nil {
			return nil, fmt.Errorf("strategyregistry: scan count: %w", err)
		}
		counts[docType] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("strategyregistry: rows error: %w", err)
	}
	return counts, nil
}

// queryByType is the internal helper for type-specific queries.
// SAFETY GATE: Always includes status='approved'.
func (r *Registry) queryByType(ctx context.Context, docType string) ([]Document, error) {
	query := baseSelect + `WHERE doc_type = $1 AND status = 'approved' ORDER BY rel_path`
	return r.queryDocsWithArg(ctx, query, docType)
}

func (r *Registry) queryDocs(ctx context.Context, query string) ([]Document, error) {
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("strategyregistry: query: %w", err)
	}
	defer rows.Close()
	return scanDocuments(rows)
}

func (r *Registry) queryDocsWithArg(ctx context.Context, query string, arg any) ([]Document, error) {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, fmt.Errorf("strategyregistry: query: %w", err)
	}
	defer rows.Close()
	return scanDocuments(rows)
}

// scanDocuments scans multiple rows into a slice of Document.
func scanDocuments(rows pgx.Rows) ([]Document, error) {
	var docs []Document
	for rows.Next() {
		doc, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("strategyregistry: rows error: %w", err)
	}
	return docs, nil
}

// scanDocument scans a single row into a Document.
func scanDocument(row pgx.Row) (Document, error) {
	var doc Document
	var tagsJSON []byte

	err := row.Scan(
		&doc.DocID,
		&doc.DocType,
		&doc.RelPath,
		&doc.Title,
		&doc.Version,
		&doc.Status,
		&doc.CreatedUTC,
		&doc.UpdatedUTC,
		&tagsJSON,
		&doc.Sha256,
		&doc.Markdown,
	)
	if err != nil {
		return Document{}, err
	}

	// Decode JSONB tags array
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &doc.Tags); err != nil {
			doc.Tags = []string{} // fallback to empty
		}
	} else {
		doc.Tags = []string{}
	}

	return doc, nil
}

// scanRow is a helper to scan from pgx.Rows (implements pgx.Row interface).
func scanRow(rows pgx.Rows) (Document, error) {
	var doc Document
	var tagsJSON []byte

	err := rows.Scan(
		&doc.DocID,
		&doc.DocType,
		&doc.RelPath,
		&doc.Title,
		&doc.Version,
		&doc.Status,
		&doc.CreatedUTC,
		&doc.UpdatedUTC,
		&tagsJSON,
		&doc.Sha256,
		&doc.Markdown,
	)
	if err != nil {
		return Document{}, fmt.Errorf("strategyregistry: scan: %w", err)
	}

	// Decode JSONB tags array
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &doc.Tags); err != nil {
			doc.Tags = []string{} // fallback to empty
		}
	} else {
		doc.Tags = []string{}
	}

	return doc, nil
}
