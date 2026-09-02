package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRawPayloadStorageMigrationDeclaresDurableSplit(t *testing.T) {
	up, err := os.ReadFile(filepath.Join("000055_raw_payload_storage.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile(filepath.Join("000055_raw_payload_storage.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	upText := string(up)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS raw_payload_contents",
		"payload BYTEA NOT NULL",
		"CREATE TABLE IF NOT EXISTS raw_payload_acquisitions",
		"REFERENCES raw_payload_contents(content_digest)",
		"reference JSONB NOT NULL",
		"BEFORE UPDATE OR DELETE",
	} {
		if !strings.Contains(upText, required) {
			t.Errorf("migration missing %q", required)
		}
	}
	for _, required := range []string{
		"DROP TABLE IF EXISTS raw_payload_acquisitions",
		"DROP TABLE IF EXISTS raw_payload_contents",
	} {
		if !strings.Contains(string(down), required) {
			t.Errorf("rollback missing %q", required)
		}
	}
}
