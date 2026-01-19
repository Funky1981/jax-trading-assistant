package testing

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func LoadFixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("fixtures: unable to resolve caller path")
	}
	base := filepath.Join(filepath.Dir(file), "fixtures")
	path := filepath.Join(base, name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fixtures: read %s: %v", path, err)
	}
	return raw
}
