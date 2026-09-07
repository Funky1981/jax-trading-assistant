package harness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ResearchMemoryContractV1 = "jax.research_memory/v1"
const SemanticResultCacheEnabled = false

const (
	MemorySourceEvidence        = "SOURCE_EVIDENCE"
	MemoryHistoricalConclusion  = "HISTORICAL_RESEARCH_CONCLUSION"
	MemoryCurrentConclusion     = "CURRENT_VALID_CONCLUSION"
	MemorySupersededConclusion  = "SUPERSEDED_CONCLUSION"
	MemoryInvalidatedConclusion = "INVALIDATED_CONCLUSION"
	MemoryUnknown               = "UNKNOWN"
	MemoryStatusValid           = "VALID"
	MemoryStatusSuperseded      = "SUPERSEDED"
	MemoryStatusInvalidated     = "INVALIDATED"
	MemoryStatusUnknown         = "UNKNOWN"
)

type ResearchValidityKey struct {
	ContractVersion           string   `json:"contract_version"`
	TaskType                  string   `json:"task_type"`
	ObjectiveFingerprint      string   `json:"objective_fingerprint"`
	EvidenceFingerprint       string   `json:"evidence_fingerprint"`
	SourceVintages            []string `json:"source_vintages"`
	ToolVersions              []string `json:"tool_versions"`
	PromptSystemVersion       string   `json:"prompt_system_version"`
	OutputContract            string   `json:"output_contract"`
	Provider                  string   `json:"provider"`
	Model                     string   `json:"model"`
	ResearchContractVersion   string   `json:"research_contract_version"`
	PolicyVersion             string   `json:"policy_version"`
	QuantVersions             []string `json:"quant_versions"`
	RuntimeOptionsFingerprint string   `json:"runtime_options_fingerprint"`
}

func (key ResearchValidityKey) Validate() error {
	if key.ContractVersion != ResearchMemoryContractV1 || strings.TrimSpace(key.TaskType) == "" || strings.TrimSpace(key.PromptSystemVersion) == "" || strings.TrimSpace(key.OutputContract) == "" || strings.TrimSpace(key.Provider) == "" || strings.TrimSpace(key.Model) == "" || strings.TrimSpace(key.ResearchContractVersion) == "" || strings.TrimSpace(key.PolicyVersion) == "" || !validSHA256Usage(key.ObjectiveFingerprint) || !validSHA256Usage(key.EvidenceFingerprint) || !validSHA256Usage(key.RuntimeOptionsFingerprint) || !schemaStringList(key.SourceVintages) || !schemaStringList(key.ToolVersions) || !schemaStringList(key.QuantVersions) {
		return fmt.Errorf("research validity key is incomplete or non-deterministic")
	}
	return nil
}

func (key ResearchValidityKey) Digest() (string, error) {
	if err := key.Validate(); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(key)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

type ResearchMemoryEntry struct {
	ContractVersion string              `json:"contract_version"`
	ID              string              `json:"id"`
	ValidityKey     ResearchValidityKey `json:"validity_key"`
	ValidityDigest  string              `json:"validity_digest"`
	Kind            string              `json:"kind"`
	Status          string              `json:"status"`
	Content         string              `json:"content"`
	EvidenceIDs     []string            `json:"evidence_ids"`
	ProvenanceIDs   []string            `json:"provenance_ids"`
	CreatedAt       time.Time           `json:"created_at"`
	ValidUntil      time.Time           `json:"valid_until"`
}

func (entry ResearchMemoryEntry) Validate() error {
	if entry.ContractVersion != ResearchMemoryContractV1 || !validIdentityLike("memory_", entry.ID) || strings.TrimSpace(entry.Content) == "" || len(entry.Content) > 128*1024 || !schemaStringList(entry.EvidenceIDs) || len(entry.EvidenceIDs) == 0 || !schemaStringList(entry.ProvenanceIDs) || len(entry.ProvenanceIDs) == 0 || entry.CreatedAt.IsZero() || entry.CreatedAt.Location() != time.UTC || entry.ValidUntil.IsZero() || entry.ValidUntil.Location() != time.UTC || !entry.ValidUntil.After(entry.CreatedAt) {
		return fmt.Errorf("research memory entry is incomplete or unbounded")
	}
	if err := entry.ValidityKey.Validate(); err != nil {
		return err
	}
	digest, err := entry.ValidityKey.Digest()
	if err != nil || entry.ValidityDigest != digest || entry.ID != deriveMemoryID(entry) {
		return fmt.Errorf("research memory provenance identity is invalid")
	}
	for _, evidenceID := range entry.EvidenceIDs {
		if !validArgumentIdentifier(evidenceID) {
			return fmt.Errorf("research memory has invalid evidence ID")
		}
	}
	switch entry.Kind {
	case MemorySourceEvidence, MemoryHistoricalConclusion, MemoryCurrentConclusion, MemorySupersededConclusion, MemoryInvalidatedConclusion, MemoryUnknown:
	default:
		return fmt.Errorf("unsupported research memory kind %q", entry.Kind)
	}
	switch entry.Status {
	case MemoryStatusValid, MemoryStatusSuperseded, MemoryStatusInvalidated, MemoryStatusUnknown:
	default:
		return fmt.Errorf("unsupported research memory status %q", entry.Status)
	}
	return nil
}

func NewResearchMemoryEntry(key ResearchValidityKey, kind, status, content string, evidenceIDs, provenanceIDs []string, createdAt, validUntil time.Time) (ResearchMemoryEntry, error) {
	digest, err := key.Digest()
	if err != nil {
		return ResearchMemoryEntry{}, err
	}
	entry := ResearchMemoryEntry{ContractVersion: ResearchMemoryContractV1, ValidityKey: key, ValidityDigest: digest, Kind: kind, Status: status, Content: content, EvidenceIDs: append([]string(nil), evidenceIDs...), ProvenanceIDs: append([]string(nil), provenanceIDs...), CreatedAt: createdAt, ValidUntil: validUntil}
	entry.ID = deriveMemoryID(entry)
	if err := entry.Validate(); err != nil {
		return ResearchMemoryEntry{}, err
	}
	return entry, nil
}

type ResearchMemoryResult struct {
	Entry  ResearchMemoryEntry `json:"entry"`
	Label  string              `json:"label"`
	Reason string              `json:"reason"`
}

type ResearchMemoryStore interface {
	Put(context.Context, ResearchMemoryEntry) error
	GetExact(context.Context, ResearchValidityKey, time.Time) (ResearchMemoryResult, error)
	Invalidate(context.Context, string) error
}

type ExactResearchMemory struct {
	mu      sync.Mutex
	entries map[string]ResearchMemoryEntry
}

func NewExactResearchMemory() *ExactResearchMemory {
	return &ExactResearchMemory{entries: map[string]ResearchMemoryEntry{}}
}

func (store *ExactResearchMemory) Put(_ context.Context, entry ResearchMemoryEntry) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if existing, ok := store.entries[entry.ValidityDigest]; ok && existing.ID != entry.ID {
		return fmt.Errorf("exact memory validity key is immutable")
	}
	store.entries[entry.ValidityDigest] = entry
	return nil
}

func (store *ExactResearchMemory) GetExact(_ context.Context, key ResearchValidityKey, now time.Time) (ResearchMemoryResult, error) {
	digest, err := key.Digest()
	if err != nil {
		return ResearchMemoryResult{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, ok := store.entries[digest]
	if !ok {
		return ResearchMemoryResult{}, fmt.Errorf("exact research memory miss")
	}
	if err := entry.Validate(); err != nil {
		return ResearchMemoryResult{}, fmt.Errorf("research memory integrity failure: %w", err)
	}
	if entry.Status != MemoryStatusValid || (entry.Kind != MemorySourceEvidence && entry.Kind != MemoryCurrentConclusion) {
		return ResearchMemoryResult{}, fmt.Errorf("research memory entry is not reusable: %s/%s", entry.Kind, entry.Status)
	}
	if now.IsZero() || now.Location() != time.UTC || !now.Before(entry.ValidUntil) {
		return ResearchMemoryResult{}, fmt.Errorf("research memory entry is stale")
	}
	return ResearchMemoryResult{Entry: entry, Label: "EXACT_REUSE", Reason: "complete validity key matched"}, nil
}

func (store *ExactResearchMemory) Invalidate(_ context.Context, digest string) error {
	if !validSHA256Usage(digest) {
		return fmt.Errorf("invalid memory validity digest")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, ok := store.entries[digest]
	if !ok {
		return fmt.Errorf("exact research memory miss")
	}
	entry.Status = MemoryStatusInvalidated
	store.entries[digest] = entry
	return nil
}

type PostgresResearchMemory struct{ pool *pgxpool.Pool }

func NewPostgresResearchMemory(pool *pgxpool.Pool) *PostgresResearchMemory {
	if pool == nil {
		return nil
	}
	return &PostgresResearchMemory{pool: pool}
}

func (store *PostgresResearchMemory) Put(ctx context.Context, entry ResearchMemoryEntry) error {
	if store == nil || store.pool == nil {
		return fmt.Errorf("research memory store unavailable")
	}
	if err := entry.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = store.pool.Exec(ctx, `INSERT INTO research_memory_entries (memory_id, validity_digest, kind, status, payload, created_at, valid_until) VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7)`, entry.ID, entry.ValidityDigest, entry.Kind, entry.Status, payload, entry.CreatedAt, entry.ValidUntil)
	if err != nil {
		return fmt.Errorf("save research memory: %w", err)
	}
	return nil
}

func (store *PostgresResearchMemory) GetExact(ctx context.Context, key ResearchValidityKey, now time.Time) (ResearchMemoryResult, error) {
	if store == nil || store.pool == nil {
		return ResearchMemoryResult{}, fmt.Errorf("research memory store unavailable")
	}
	digest, err := key.Digest()
	if err != nil {
		return ResearchMemoryResult{}, err
	}
	var payload []byte
	if err := store.pool.QueryRow(ctx, `SELECT payload FROM research_memory_entries WHERE validity_digest = $1`, digest).Scan(&payload); err != nil {
		return ResearchMemoryResult{}, err
	}
	var entry ResearchMemoryEntry
	if err := json.Unmarshal(payload, &entry); err != nil {
		return ResearchMemoryResult{}, fmt.Errorf("decode research memory: %w", err)
	}
	result := ResearchMemoryResult{Entry: entry, Label: "EXACT_REUSE", Reason: "complete validity key matched"}
	if err := entry.Validate(); err != nil || entry.Status != MemoryStatusValid || (entry.Kind != MemorySourceEvidence && entry.Kind != MemoryCurrentConclusion) || now.IsZero() || now.Location() != time.UTC || !now.Before(entry.ValidUntil) {
		return ResearchMemoryResult{}, fmt.Errorf("research memory entry is stale, invalid or non-reusable")
	}
	return result, nil
}

func (store *PostgresResearchMemory) Invalidate(ctx context.Context, digest string) error {
	if store == nil || store.pool == nil || !validSHA256Usage(digest) {
		return fmt.Errorf("research memory store or digest is invalid")
	}
	command, err := store.pool.Exec(ctx, `UPDATE research_memory_entries SET status = 'INVALIDATED', payload = jsonb_set(payload, '{status}', '"INVALIDATED"'::jsonb) WHERE validity_digest = $1`, digest)
	if err != nil {
		return fmt.Errorf("invalidate research memory: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("invalidate research memory: entry not found")
	}
	return nil
}

var _ ResearchMemoryStore = (*ExactResearchMemory)(nil)
var _ ResearchMemoryStore = (*PostgresResearchMemory)(nil)

func deriveMemoryID(entry ResearchMemoryEntry) string {
	entry.ID = ""
	entry.Status = ""
	seed, _ := json.Marshal(entry)
	digest := sha256.Sum256(seed)
	return "memory_" + hex.EncodeToString(digest[:])
}
