package utcp

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, name string, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestLoadProvidersConfig_Valid(t *testing.T) {
	path := writeTempFile(t, "providers.json", `{
  "providers": [
    { "id": "risk", "transport": "local" },
    { "id": "dexter", "transport": "http", "endpoint": "http://localhost:3000/tools" }
  ]
}`)

	cfg, err := LoadProvidersConfig(path)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got := len(cfg.Providers); got != 2 {
		t.Fatalf("expected 2 providers, got %d", got)
	}
}

func TestLoadProvidersConfig_DuplicateProviderID(t *testing.T) {
	path := writeTempFile(t, "providers.json", `{
  "providers": [
    { "id": "risk", "transport": "local" },
    { "id": "risk", "transport": "local" }
  ]
}`)

	_, err := LoadProvidersConfig(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadProvidersConfig_HTTPMissingEndpoint(t *testing.T) {
	path := writeTempFile(t, "providers.json", `{
  "providers": [
    { "id": "dexter", "transport": "http" }
  ]
}`)

	_, err := LoadProvidersConfig(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadProvidersConfig_LocalWithEndpointRejected(t *testing.T) {
	path := writeTempFile(t, "providers.json", `{
  "providers": [
    { "id": "risk", "transport": "local", "endpoint": "http://should-not-exist" }
  ]
}`)

	_, err := LoadProvidersConfig(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadProvidersConfig_UnknownTransportRejected(t *testing.T) {
	path := writeTempFile(t, "providers.json", `{
  "providers": [
    { "id": "x", "transport": "ftp" }
  ]
}`)

	_, err := LoadProvidersConfig(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadProvidersConfig_UnknownFieldRejected(t *testing.T) {
	path := writeTempFile(t, "providers.json", `{
  "providers": [
    { "id": "risk", "transport": "local", "unexpected": true }
  ]
}`)

	_, err := LoadProvidersConfig(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
