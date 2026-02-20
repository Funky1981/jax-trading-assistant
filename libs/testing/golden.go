// Package testing provides shared test helpers: clocks, fixtures, in-memory
// stores, golden-snapshot comparison, and determinism harnesses.
package testing

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// updateGolden is set via -update flag to regenerate golden files.
var updateGolden = flag.Bool("update", false, "update golden fixture files")

// Golden compares got (any JSON-marshallable value) against the golden file
// stored at testdata/golden/<name>.json relative to the calling test file.
//
// If -update is passed, the golden file is written and the test passes
// unconditionally. This makes it easy to refresh baselines:
//
//	go test ./... -update
//
// Example:
//
//	result := myFunction()
//	testing.Golden(t, "my_function_result", result)
func Golden(t testing.TB, name string, got any) {
	t.Helper()
	path := goldenPath(t, name)
	if *updateGolden {
		writeGolden(t, path, got)
		return
	}
	assertGolden(t, path, got)
}

// GoldenBytes compares raw bytes against the golden file.
// The bytes are treated as a JSON-like blob; if they are valid JSON they are
// pretty-printed before writing so diffs are readable.
func GoldenBytes(t testing.TB, name string, got []byte) {
	t.Helper()
	path := goldenPath(t, name)
	if *updateGolden {
		writeBytesGolden(t, path, got)
		return
	}
	assertBytesGolden(t, path, got)
}

// AssertDeterministic calls fn twice and asserts that the JSON representation
// of each result is identical. This is a lightweight check that fn has no
// non-deterministic side effects (random ordering, time-dependent output, etc.).
func AssertDeterministic(t testing.TB, fn func() any) {
	t.Helper()
	a := fn()
	b := fn()

	aJSON, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("AssertDeterministic: marshal first result: %v", err)
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("AssertDeterministic: marshal second result: %v", err)
	}

	if string(aJSON) != string(bJSON) {
		t.Errorf("AssertDeterministic: results differ\nfirst:  %s\nsecond: %s", aJSON, bJSON)
	}
}

// AssertDeepEqual is a thin wrapper around reflect.DeepEqual that formats
// a readable diff message on failure.
func AssertDeepEqual(t testing.TB, want, got any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		wantJSON, _ := json.MarshalIndent(want, "", "  ")
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		t.Errorf("values differ\nwant: %s\n got: %s", wantJSON, gotJSON)
	}
}

// MustMarshal marshals v to JSON or fatals the test. Useful for building
// expected JSON blobs inline without error handling boilerplate.
func MustMarshal(t testing.TB, v any) []byte {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("MustMarshal: %v", err)
	}
	return b
}

// ─── internal helpers ───────────────────────────────────────────────────────

// goldenPath resolves the path to testdata/golden/<name>.json, anchored to
// the directory of the *calling test file* (not the working directory).
func goldenPath(t testing.TB, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(2) // 0=goldenPath, 1=Golden, 2=test
	if !ok {
		t.Fatalf("goldenPath: unable to resolve caller")
	}
	dir := filepath.Join(filepath.Dir(file), "testdata", "golden")
	return filepath.Join(dir, fmt.Sprintf("%s.json", name))
}

// writeGolden marshals v and writes it to path, creating directories as needed.
func writeGolden(t testing.TB, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("golden update: marshal: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("golden update: mkdir: %v", err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatalf("golden update: write %s: %v", path, err)
	}
	t.Logf("golden: updated %s", path)
}

// assertGolden reads the golden file and compares it against got (as JSON).
func assertGolden(t testing.TB, path string, got any) {
	t.Helper()
	wantBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Errorf("golden: file not found: %s — run with -update to create it", path)
			return
		}
		t.Fatalf("golden: read %s: %v", path, err)
	}

	gotBytes, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("golden: marshal got: %v", err)
	}

	// Normalise: unmarshal both sides and re-marshal so formatting is identical.
	var wantNorm, gotNorm any
	if err := json.Unmarshal(wantBytes, &wantNorm); err != nil {
		t.Fatalf("golden: unmarshal want: %v", err)
	}
	if err := json.Unmarshal(gotBytes, &gotNorm); err != nil {
		t.Fatalf("golden: unmarshal got: %v", err)
	}

	if !reflect.DeepEqual(wantNorm, gotNorm) {
		wantPretty, _ := json.MarshalIndent(wantNorm, "", "  ")
		gotPretty, _ := json.MarshalIndent(gotNorm, "", "  ")
		t.Errorf("golden mismatch for %s\nwant:\n%s\n got:\n%s", path, wantPretty, gotPretty)
	}
}

// writeBytesGolden writes raw bytes to the golden file.
func writeBytesGolden(t testing.TB, path string, b []byte) {
	t.Helper()
	// Pretty-print if valid JSON.
	var norm any
	if err := json.Unmarshal(b, &norm); err == nil {
		pretty, _ := json.MarshalIndent(norm, "", "  ")
		b = append(pretty, '\n')
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("golden update: mkdir: %v", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("golden update: write %s: %v", path, err)
	}
	t.Logf("golden: updated %s", path)
}

// assertBytesGolden compares raw bytes against the golden file.
func assertBytesGolden(t testing.TB, path string, got []byte) {
	t.Helper()
	wantBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Errorf("golden: file not found: %s — run with -update to create it", path)
			return
		}
		t.Fatalf("golden: read %s: %v", path, err)
	}

	var wantNorm, gotNorm any
	wantErr := json.Unmarshal(wantBytes, &wantNorm)
	gotErr := json.Unmarshal(got, &gotNorm)

	if wantErr == nil && gotErr == nil {
		if !reflect.DeepEqual(wantNorm, gotNorm) {
			wantPretty, _ := json.MarshalIndent(wantNorm, "", "  ")
			gotPretty, _ := json.MarshalIndent(gotNorm, "", "  ")
			t.Errorf("golden mismatch for %s\nwant:\n%s\n got:\n%s", path, wantPretty, gotPretty)
		}
		return
	}
	// Not JSON: byte-exact comparison.
	if string(wantBytes) != string(got) {
		t.Errorf("golden mismatch for %s\nwant:\n%s\n got:\n%s", path, wantBytes, got)
	}
}
