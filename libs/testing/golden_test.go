package testing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ─── Golden ─────────────────────────────────────────────────────────────────

func TestGolden_CreateAndMatch(t *testing.T) {
	type result struct {
		Strategy string  `json:"strategy"`
		Sharpe   float64 `json:"sharpe"`
		Trades   int     `json:"trades"`
	}

	dir := t.TempDir()
	goldenFile := filepath.Join(dir, "testdata", "golden", "backtest_result.json")

	// Build want JSON manually and write it as golden.
	want := result{Strategy: "rsi_v1", Sharpe: 1.23, Trades: 42}
	b, _ := json.MarshalIndent(want, "", "  ")
	_ = os.MkdirAll(filepath.Dir(goldenFile), 0o755)
	_ = os.WriteFile(goldenFile, append(b, '\n'), 0o644)

	// assertGolden should pass when content matches.
	assertGolden(t, goldenFile, want)
}

func TestGolden_Mismatch(t *testing.T) {
	type result struct {
		Value int `json:"value"`
	}

	dir := t.TempDir()
	goldenFile := filepath.Join(dir, "testdata", "golden", "value.json")
	want := result{Value: 10}
	b, _ := json.MarshalIndent(want, "", "  ")
	_ = os.MkdirAll(filepath.Dir(goldenFile), 0o755)
	_ = os.WriteFile(goldenFile, append(b, '\n'), 0o644)

	// Mismatched value should call t.Errorf (not Fatal) — capture with sub-test.
	got := result{Value: 99}
	rec := &recordingTB{TB: t}
	assertGolden(rec, goldenFile, got)
	if !rec.failed {
		t.Error("expected mismatch to fail but it did not")
	}
}

func TestGolden_MissingFile(t *testing.T) {
	dir := t.TempDir()
	missingPath := filepath.Join(dir, "testdata", "golden", "missing.json")
	rec := &recordingTB{TB: t}
	assertGolden(rec, missingPath, map[string]int{"x": 1})
	if !rec.failed {
		t.Error("expected missing golden file to fail but it did not")
	}
}

// ─── WriteGolden ─────────────────────────────────────────────────────────────

func TestWriteGolden(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testdata", "golden", "output.json")
	writeGolden(t, path, map[string]string{"hello": "world"})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if m["hello"] != "world" {
		t.Errorf("expected 'world', got '%s'", m["hello"])
	}
}

// ─── AssertDeterministic ─────────────────────────────────────────────────────

func TestAssertDeterministic_Stable(t *testing.T) {
	// A deterministic function.
	call := 0
	AssertDeterministic(t, func() any {
		call++
		return map[string]int{"result": 42, "call": 1} // always same content
	})
	_ = call
}

func TestAssertDeterministic_Unstable(t *testing.T) {
	n := 0
	rec := &recordingTB{TB: t}
	AssertDeterministic(rec, func() any {
		n++
		return map[string]int{"n": n} // changes each call
	})
	if !rec.failed {
		t.Error("expected non-deterministic function to fail")
	}
}

// ─── AssertDeepEqual ─────────────────────────────────────────────────────────

func TestAssertDeepEqual_Equal(t *testing.T) {
	AssertDeepEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
}

func TestAssertDeepEqual_NotEqual(t *testing.T) {
	rec := &recordingTB{TB: t}
	AssertDeepEqual(rec, []int{1, 2, 3}, []int{1, 2, 4})
	if !rec.failed {
		t.Error("expected deep-equal failure but test passed")
	}
}

// ─── MustMarshal ─────────────────────────────────────────────────────────────

func TestMustMarshal(t *testing.T) {
	b := MustMarshal(t, map[string]int{"a": 1})
	if len(b) == 0 {
		t.Error("expected non-empty JSON output")
	}
	var m map[string]int
	if err := json.Unmarshal(b, &m); err != nil {
		t.Errorf("expected valid JSON: %v", err)
	}
	if m["a"] != 1 {
		t.Errorf("expected a=1, got %d", m["a"])
	}
}

// ─── GoldenBytes ─────────────────────────────────────────────────────────────

func TestGoldenBytes_Match(t *testing.T) {
	dir := t.TempDir()
	goldenFile := filepath.Join(dir, "testdata", "golden", "bytes.json")
	content := []byte(`{"x":1}`)
	_ = os.MkdirAll(filepath.Dir(goldenFile), 0o755)
	_ = os.WriteFile(goldenFile, content, 0o644)

	assertBytesGolden(t, goldenFile, content)
}

func TestGoldenBytes_Mismatch(t *testing.T) {
	dir := t.TempDir()
	goldenFile := filepath.Join(dir, "testdata", "golden", "bytes2.json")
	_ = os.MkdirAll(filepath.Dir(goldenFile), 0o755)
	_ = os.WriteFile(goldenFile, []byte(`{"x":1}`), 0o644)

	rec := &recordingTB{TB: t}
	assertBytesGolden(rec, goldenFile, []byte(`{"x":2}`))
	if !rec.failed {
		t.Error("expected bytes mismatch to fail")
	}
}

// ─── recordingTB ─────────────────────────────────────────────────────────────

// recordingTB wraps testing.TB and records whether Errorf or Fatalf were called.
type recordingTB struct {
	testing.TB
	failed bool
}

func (r *recordingTB) Errorf(format string, args ...any) {
	r.failed = true
	// don't forward to parent — this is intentional failure
}

func (r *recordingTB) Fatalf(format string, args ...any) {
	r.failed = true
	panic("fatalf") // stop execution like real Fatalf
}

func (r *recordingTB) Helper() {}
func (r *recordingTB) Logf(format string, args ...any) {
	r.TB.Logf(format, args...)
}
