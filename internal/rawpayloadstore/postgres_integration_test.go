package rawpayloadstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/database"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresRawPayloadStoreDurabilityAndIdentity(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("JAX_RAW_PAYLOAD_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("set JAX_RAW_PAYLOAD_TEST_DATABASE_URL to run PostgreSQL raw-payload integration tests")
	}

	ctx := context.Background()
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	if err := database.RunMigrations(sqlDB, "file://../../db/postgres/migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	storeA, err := NewPostgresRawPayloadStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("{\"line\":1}\r\n\x00\xff")
	ref := integrationRef("rpa_postgres_"+fmt.Sprint(time.Now().UnixNano()), payload)
	registry, err := integrationRegistry()
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := providercontract.PersistRawPayload(ctx, registry, storeA, providercontract.RawPayloadPersistenceRequest{
		ID: ref.ID, Provider: ref.Provider, Capability: ref.CapabilityID, Raw: ref.Raw,
		Capture: ref.Capture, Source: ref.Source, Revision: ref.Revision, ReceivedAt: ref.ReceivedAt,
		Retention: ref.Retention, Complete: true,
	}, payload)
	if err != nil {
		t.Fatalf("PersistRawPayload: %v", err)
	}
	if err := providercontract.VerifyRawPayload(ctx, storeA, descriptor.Ref); err != nil {
		t.Fatalf("VerifyRawPayload: %v", err)
	}
	retrieved, err := providercontract.RetrieveRawPayload(ctx, storeA, descriptor.Ref)
	if err != nil || string(retrieved) != string(payload) {
		t.Fatalf("RetrieveRawPayload: bytes=%x err=%v", retrieved, err)
	}

	location, err := storeA.Put(ctx, ref, payload)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if location.Key != "sha256/"+ref.Content.Digest.Value || location.Store.Value != postgresStoreVersion {
		t.Fatalf("unexpected logical location: %+v", location)
	}

	stored, err := storeA.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(stored) != string(payload) {
		t.Fatalf("exact bytes changed: got %x want %x", stored, payload)
	}
	stored[0] = 'X'
	storedAgain, err := storeA.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get after defensive-copy mutation: %v", err)
	}
	if string(storedAgain) != string(payload) {
		t.Fatal("Get did not return an independent byte slice")
	}

	storeB, err := NewPostgresRawPayloadStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	if recovered, err := storeB.Get(ctx, ref); err != nil || string(recovered) != string(payload) {
		t.Fatalf("restart durability Get: bytes=%x err=%v", recovered, err)
	}

	if _, err := storeA.Put(ctx, ref, payload); err != nil {
		t.Fatalf("identical acquisition was not idempotent: %v", err)
	}
	conflict := ref
	conflict.Provider.Namespace = "different.provider"
	if _, err := storeA.Put(ctx, conflict, payload); err == nil || !isRawPayloadCode(err, providercontract.RawPayloadErrorIdentityConflict) {
		t.Fatalf("provider metadata conflict error = %v", err)
	}

	secondRef := integrationRef("rpa_postgres_"+fmt.Sprint(time.Now().UnixNano()+1), payload)
	if _, err := storeA.Put(ctx, secondRef, payload); err != nil {
		t.Fatalf("second acquisition with same bytes: %v", err)
	}
	var contentCount, acquisitionCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM raw_payload_contents WHERE content_digest = $1`, ref.Content.Digest.Value).Scan(&contentCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM raw_payload_acquisitions WHERE content_digest = $1`, ref.Content.Digest.Value).Scan(&acquisitionCount); err != nil {
		t.Fatal(err)
	}
	if contentCount != 1 || acquisitionCount != 2 {
		t.Fatalf("dedup counts = content %d, acquisitions %d; want 1, 2", contentCount, acquisitionCount)
	}

	concurrentBase := time.Now().UnixNano()
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func(i int) {
			concurrentRef := integrationRef(fmt.Sprintf("rpa_postgres_concurrent_%d_%d", concurrentBase, i), payload)
			_, putErr := storeB.Put(ctx, concurrentRef, payload)
			errs <- putErr
		}(i)
	}
	for i := 0; i < 8; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent distinct acquisition %d: %v", i, err)
		}
	}

	identicalRef := integrationRef(fmt.Sprintf("rpa_postgres_identical_%d", concurrentBase), payload)
	for i := 0; i < 8; i++ {
		go func() {
			_, putErr := storeB.Put(ctx, identicalRef, payload)
			errs <- putErr
		}()
	}
	for i := 0; i < 8; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent identical acquisition %d: %v", i, err)
		}
	}

	conflictBase := integrationRef(fmt.Sprintf("rpa_postgres_conflict_%d", concurrentBase), payload)
	conflictA := conflictBase
	conflictB := conflictBase
	conflictB.Provider.Namespace = "conflicting.provider"
	conflictResults := make(chan error, 2)
	go func() { _, putErr := storeB.Put(ctx, conflictA, payload); conflictResults <- putErr }()
	go func() { _, putErr := storeB.Put(ctx, conflictB, payload); conflictResults <- putErr }()
	var successes, conflicts int
	for i := 0; i < 2; i++ {
		err := <-conflictResults
		if err == nil {
			successes++
		} else if isRawPayloadCode(err, providercontract.RawPayloadErrorIdentityConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected concurrent identity conflict error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent conflict outcomes = successes %d, conflicts %d; want 1, 1", successes, conflicts)
	}

	failedRef := integrationRef(fmt.Sprintf("rpa_postgres_failed_%d", concurrentBase), payload)
	if _, err := storeB.Put(ctx, failedRef, []byte("wrong")); err == nil {
		t.Fatal("invalid bytes unexpectedly published")
	}
	var failedCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM raw_payload_acquisitions WHERE payload_id = $1`, string(failedRef.ID)).Scan(&failedCount); err != nil {
		t.Fatal(err)
	}
	if failedCount != 0 {
		t.Fatalf("failed acquisition published %d rows", failedCount)
	}

	missing := ref
	missing.ID = providercontract.RawPayloadID("rpa_postgres_missing_" + fmt.Sprint(time.Now().UnixNano()))
	if _, err := storeA.Get(ctx, missing); err == nil || !isRawPayloadCode(err, providercontract.RawPayloadErrorRetrievalMissing) {
		t.Fatalf("missing Get error = %v", err)
	}
}

func integrationRef(id string, payload []byte) providercontract.RawPayloadRef {
	return providercontract.RawPayloadRef{
		ContractVersion: providercontract.RawPayloadRefContractV1,
		ID:              providercontract.RawPayloadID(id),
		Content:         canonical.RawContentIdentity(payload),
		Provider: canonical.ProviderIdentity{
			ID: "pvd_integration", Namespace: "integration.provider",
		},
		CapabilityID: providercontract.CapabilityMarketBars,
		Raw: providercontract.RawRepresentation{
			Boundary: providercontract.RawBoundaryProvider,
			Format:   providercontract.RawFormatJSONDocument,
			Schema: canonical.VersionIdentity{
				Namespace: "integration.raw", Value: "v1",
			},
			MediaType: "application/json",
		},
		Capture: providercontract.RawPayloadCapture{
			ByteForm:           providercontract.RawPayloadByteFormEntityBody,
			ContentCodingState: providercontract.ContentCodingIdentity,
		},
		ReceivedAt: time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC),
		SizeBytes:  int64(len(payload)),
		Retention: providercontract.RawPayloadRetentionPolicy{
			Class: providercontract.RawPayloadRetentionReplayAudit,
			Policy: canonical.VersionIdentity{
				Namespace: "jax.retention", Value: "v1",
			},
			Redistribution: providercontract.RawPayloadRedistributionNotAuthorized,
		},
	}
}

func integrationRegistry() (*providercontract.Registry, error) {
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		return nil, err
	}
	return registry, registry.Register(providercontract.ProviderDefinition{
		ContractVersion: providercontract.ProviderDefinitionV1,
		Identity:        canonical.ProviderIdentity{ID: "pvd_integration", Namespace: "integration.provider"},
		DisplayName:     "integration provider",
		AdapterVersion:  canonical.VersionIdentity{Namespace: "integration.adapter", Value: "v1"},
		Capabilities: []providercontract.Capability{{
			ContractVersion: providercontract.CapabilityContractV1,
			ID:              providercontract.CapabilityMarketBars,
			Category:        providercontract.DataCategoryMarketData,
			Support:         providercontract.SupportSupported,
			Raw: providercontract.RawRepresentation{
				Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument,
				Schema: canonical.VersionIdentity{Namespace: "integration.raw", Value: "v1"}, MediaType: "application/json",
			},
			Authentication: providercontract.AuthenticationRequirement{Class: providercontract.AuthenticationNone},
			Operational: providercontract.OperationalSemantics{
				DeliveryModes:      []providercontract.DeliveryMode{providercontract.DeliverySnapshot},
				FreshnessModes:     []providercontract.FreshnessMode{providercontract.FreshnessOnDemand},
				QualityRequirement: providercontract.QualityCanonicalValidationRequired,
			},
			CanonicalOutputs: []canonical.ContractSchemaRef{{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2}},
		}},
	})
}

func isRawPayloadCode(err error, want providercontract.RawPayloadErrorCode) bool {
	var rawErr *providercontract.RawPayloadError
	return errors.As(err, &rawErr) && rawErr.Code == want
}
