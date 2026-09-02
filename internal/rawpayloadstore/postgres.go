// Package rawpayloadstore contains provider-neutral durable raw-evidence
// storage adapters. The contract port remains owned by libs/contracts.
package rawpayloadstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	postgresStoreNamespace = "jax.raw_payload_store"
	postgresStoreVersion   = "postgres/v1"
)

var _ providercontract.RawPayloadStore = (*PostgresRawPayloadStore)(nil)

// PostgresRawPayloadStore stores deduplicated exact bytes separately from
// immutable acquisition records. It exposes only the logical RawPayloadStore
// contract; physical PostgreSQL details never enter RawPayloadLocation.
type PostgresRawPayloadStore struct {
	pool *pgxpool.Pool
}

// NewPostgresRawPayloadStore constructs a durable store over an existing pool.
// Runtime composition is responsible for connecting, migrating, and probing
// the pool before creating provider acquisition paths.
func NewPostgresRawPayloadStore(pool *pgxpool.Pool) (*PostgresRawPayloadStore, error) {
	if pool == nil {
		return nil, errors.New("postgres raw payload store requires a database pool")
	}
	return &PostgresRawPayloadStore{pool: pool}, nil
}

func (store *PostgresRawPayloadStore) Put(ctx context.Context, ref providercontract.RawPayloadRef, payload []byte) (providercontract.RawPayloadLocation, error) {
	if err := contextError(ctx, ref.ID, providercontract.RawPayloadErrorPersistenceFailed); err != nil {
		return providercontract.RawPayloadLocation{}, err
	}
	if store == nil || store.pool == nil {
		return providercontract.RawPayloadLocation{}, storeError(providercontract.RawPayloadErrorPersistenceFailed, ref.ID, "postgres raw payload store is unavailable", nil)
	}
	if err := ref.Validate(); err != nil {
		return providercontract.RawPayloadLocation{}, storeError(providercontract.RawPayloadErrorInvalidReference, ref.ID, "reference is invalid", err)
	}
	if err := verifyPayload(ref, payload); err != nil {
		return providercontract.RawPayloadLocation{}, err
	}
	refJSON, err := providercontract.EncodeRawPayloadRefJSON(ref)
	if err != nil {
		return providercontract.RawPayloadLocation{}, storeError(providercontract.RawPayloadErrorInvalidReference, ref.ID, "reference could not be encoded", err)
	}

	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return providercontract.RawPayloadLocation{}, storeError(providercontract.RawPayloadErrorPersistenceFailed, ref.ID, "database transaction could not start", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO raw_payload_contents
			(digest_algorithm, content_digest, representation, payload, size_bytes)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (content_digest) DO NOTHING
	`, ref.Content.Digest.Algorithm, ref.Content.Digest.Value, ref.Content.Representation, payload, ref.SizeBytes); err != nil {
		return providercontract.RawPayloadLocation{}, storeError(providercontract.RawPayloadErrorPersistenceFailed, ref.ID, "content could not be persisted", err)
	}

	var storedAlgorithm, storedDigest, storedRepresentation string
	var storedPayload []byte
	var storedSize int64
	if err := tx.QueryRow(ctx, `
		SELECT digest_algorithm, content_digest, representation, payload, size_bytes
		FROM raw_payload_contents
		WHERE content_digest = $1
		FOR UPDATE
	`, ref.Content.Digest.Value).Scan(&storedAlgorithm, &storedDigest, &storedRepresentation, &storedPayload, &storedSize); err != nil {
		return providercontract.RawPayloadLocation{}, storeError(providercontract.RawPayloadErrorPersistenceFailed, ref.ID, "content could not be read back", err)
	}
	if err := verifyStoredContent(ref, storedAlgorithm, storedDigest, storedRepresentation, storedPayload, storedSize); err != nil {
		return providercontract.RawPayloadLocation{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO raw_payload_acquisitions (
			payload_id, contract_version, content_digest, content_algorithm,
			content_representation, provider_id, provider_namespace,
			provider_external_id, capability_id, raw_boundary, raw_format,
			raw_schema_namespace, raw_schema_value, raw_media_type,
			capture_byte_form, capture_content_coding, capture_content_coding_state,
			capture_character_encoding, capture_related_to, source_id, source_kind,
			revision_namespace, revision_value, received_at, size_bytes,
			retention_class, retention_policy_namespace, retention_policy_value,
			retention_redistribution, reference
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27,
			$28, $29, $30
		)
		ON CONFLICT (payload_id) DO NOTHING
	`, acquisitionArgs(ref, refJSON)...); err != nil {
		return providercontract.RawPayloadLocation{}, storeError(providercontract.RawPayloadErrorPersistenceFailed, ref.ID, "acquisition could not be persisted", err)
	}

	var persistedJSON []byte
	var persistedDigest string
	if err := tx.QueryRow(ctx, `
		SELECT reference, content_digest
		FROM raw_payload_acquisitions
		WHERE payload_id = $1
		FOR UPDATE
	`, string(ref.ID)).Scan(&persistedJSON, &persistedDigest); err != nil {
		return providercontract.RawPayloadLocation{}, storeError(providercontract.RawPayloadErrorPersistenceFailed, ref.ID, "acquisition could not be read back", err)
	}
	var persistedRef providercontract.RawPayloadRef
	if err := providercontract.DecodeRawPayloadRefJSON(persistedJSON, &persistedRef); err != nil {
		return providercontract.RawPayloadLocation{}, storeError(providercontract.RawPayloadErrorPersistenceFailed, ref.ID, "stored acquisition metadata is invalid", err)
	}
	if persistedDigest != ref.Content.Digest.Value || !rawPayloadRefsEqual(persistedRef, ref) {
		return providercontract.RawPayloadLocation{}, storeError(providercontract.RawPayloadErrorIdentityConflict, ref.ID, "existing acquisition identity has different immutable meaning", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return providercontract.RawPayloadLocation{}, storeError(providercontract.RawPayloadErrorPersistenceFailed, ref.ID, "database transaction could not commit", err)
	}
	return locationFor(ref), nil
}

func (store *PostgresRawPayloadStore) Get(ctx context.Context, ref providercontract.RawPayloadRef) ([]byte, error) {
	if err := contextError(ctx, ref.ID, providercontract.RawPayloadErrorRetrievalMissing); err != nil {
		return nil, err
	}
	if store == nil || store.pool == nil {
		return nil, storeError(providercontract.RawPayloadErrorRetrievalMissing, ref.ID, "postgres raw payload store is unavailable", nil)
	}
	if err := ref.Validate(); err != nil {
		return nil, storeError(providercontract.RawPayloadErrorInvalidReference, ref.ID, "reference is invalid", err)
	}

	var persistedJSON []byte
	var storedAlgorithm, storedDigest, storedRepresentation string
	var storedPayload []byte
	var storedSize int64
	err := store.pool.QueryRow(ctx, `
		SELECT a.reference, c.digest_algorithm, c.content_digest, c.representation, c.payload, c.size_bytes
		FROM raw_payload_acquisitions a
		JOIN raw_payload_contents c ON c.content_digest = a.content_digest
		WHERE a.payload_id = $1
	`, string(ref.ID)).Scan(&persistedJSON, &storedAlgorithm, &storedDigest, &storedRepresentation, &storedPayload, &storedSize)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storeError(providercontract.RawPayloadErrorRetrievalMissing, ref.ID, "acquisition or content is not stored", err)
		}
		return nil, storeError(providercontract.RawPayloadErrorRetrievalMissing, ref.ID, "stored payload could not be retrieved", err)
	}

	var persistedRef providercontract.RawPayloadRef
	if err := providercontract.DecodeRawPayloadRefJSON(persistedJSON, &persistedRef); err != nil {
		return nil, storeError(providercontract.RawPayloadErrorRetrievalMissing, ref.ID, "stored acquisition metadata is invalid", err)
	}
	if !rawPayloadRefsEqual(persistedRef, ref) {
		return nil, storeError(providercontract.RawPayloadErrorIdentityConflict, ref.ID, "stored acquisition reference does not match requested reference", nil)
	}
	if err := verifyStoredContent(ref, storedAlgorithm, storedDigest, storedRepresentation, storedPayload, storedSize); err != nil {
		return nil, err
	}
	return append([]byte(nil), storedPayload...), nil
}

func acquisitionArgs(ref providercontract.RawPayloadRef, refJSON []byte) []any {
	var externalID, relatedTo any
	if ref.Provider.ExternalID != nil {
		externalID = mustJSON(ref.Provider.ExternalID)
	}
	if ref.Capture.RelatedTo != nil {
		relatedTo = mustJSON(ref.Capture.RelatedTo)
	}
	var sourceID, sourceKind any
	if ref.Source != nil {
		sourceID, sourceKind = ref.Source.ID, ref.Source.Kind
	}
	var revisionNamespace, revisionValue any
	if ref.Revision != nil {
		revisionNamespace, revisionValue = ref.Revision.Namespace, ref.Revision.Value
	}
	return []any{
		string(ref.ID), string(ref.ContractVersion), ref.Content.Digest.Value, string(ref.Content.Digest.Algorithm),
		string(ref.Content.Representation), ref.Provider.ID, ref.Provider.Namespace, externalID, string(ref.CapabilityID),
		string(ref.Raw.Boundary), string(ref.Raw.Format), ref.Raw.Schema.Namespace, ref.Raw.Schema.Value, nullableString(ref.Raw.MediaType),
		string(ref.Capture.ByteForm), nullableString(ref.Capture.ContentCoding), string(ref.Capture.ContentCodingState), nullableString(ref.Capture.CharacterEncoding), relatedTo,
		sourceID, sourceKind, revisionNamespace, revisionValue, ref.ReceivedAt, ref.SizeBytes, string(ref.Retention.Class),
		ref.Retention.Policy.Namespace, ref.Retention.Policy.Value, string(ref.Retention.Redistribution), refJSON,
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func mustJSON(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}

func verifyPayload(ref providercontract.RawPayloadRef, payload []byte) error {
	if int64(len(payload)) != ref.SizeBytes {
		return storeError(providercontract.RawPayloadErrorSizeMismatch, ref.ID, "supplied byte count does not match the reference", nil)
	}
	if err := ref.Content.Digest.VerifyBytes(payload); err != nil {
		return storeError(providercontract.RawPayloadErrorDigestMismatch, ref.ID, "supplied bytes failed content verification", err)
	}
	return nil
}

func verifyStoredContent(ref providercontract.RawPayloadRef, algorithm, digest, representation string, payload []byte, size int64) error {
	if algorithm != string(ref.Content.Digest.Algorithm) || digest != ref.Content.Digest.Value || representation != string(ref.Content.Representation) {
		return storeError(providercontract.RawPayloadErrorDigestMismatch, ref.ID, "stored content identity is inconsistent", nil)
	}
	if size != ref.SizeBytes || int64(len(payload)) != size {
		return storeError(providercontract.RawPayloadErrorSizeMismatch, ref.ID, "stored byte count does not match the reference", nil)
	}
	if err := ref.Content.Digest.VerifyBytes(payload); err != nil {
		return storeError(providercontract.RawPayloadErrorDigestMismatch, ref.ID, "stored bytes failed content verification", err)
	}
	return nil
}

func locationFor(ref providercontract.RawPayloadRef) providercontract.RawPayloadLocation {
	return providercontract.RawPayloadLocation{
		Store: canonical.VersionIdentity{Namespace: postgresStoreNamespace, Value: postgresStoreVersion},
		Key:   "sha256/" + ref.Content.Digest.Value,
	}
}

func contextError(ctx context.Context, id providercontract.RawPayloadID, code providercontract.RawPayloadErrorCode) error {
	if err := ctx.Err(); err != nil {
		return storeError(code, id, "context ended before storage operation", err)
	}
	return nil
}

func storeError(code providercontract.RawPayloadErrorCode, id providercontract.RawPayloadID, detail string, cause error) error {
	return &providercontract.RawPayloadError{Code: code, PayloadID: id, Detail: detail, Cause: cause}
}

func rawPayloadRefsEqual(left, right providercontract.RawPayloadRef) bool {
	leftJSON, leftErr := providercontract.EncodeRawPayloadRefJSON(left)
	rightJSON, rightErr := providercontract.EncodeRawPayloadRefJSON(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}
