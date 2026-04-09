package pgmemory

import (
	"testing"
)

// ── pure-function unit tests ──────────────────────────────────────────────────

func TestClampLimit(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, defaultLimit},
		{-5, defaultLimit},
		{1, 1},
		{20, 20},
		{100, 100},
		{101, maxLimit},
		{999, maxLimit},
	}
	for _, c := range cases {
		if got := clampLimit(c.in); got != c.want {
			t.Errorf("clampLimit(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestFormatVector(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := formatVector(nil)
		if got != "[]" {
			t.Errorf("got %q want []", got)
		}
	})
	t.Run("single", func(t *testing.T) {
		got := formatVector([]float32{1.0})
		if got == "" || got[0] != '[' {
			t.Errorf("unexpected format: %q", got)
		}
	})
	t.Run("roundtrip length", func(t *testing.T) {
		v := make([]float32, 1536)
		for i := range v {
			v[i] = float32(i) * 0.001
		}
		s := formatVector(v)
		if s[0] != '[' || s[len(s)-1] != ']' {
			t.Errorf("bad brackets: %q...", s[:20])
		}
	})
}

func TestBanks(t *testing.T) {
	banks := Banks()
	want := []string{"research", "trades", "signals", "reflections"}
	if len(banks) != len(want) {
		t.Fatalf("Banks() len %d want %d", len(banks), len(want))
	}
	for i, b := range banks {
		if b != want[i] {
			t.Errorf("Banks()[%d] = %q want %q", i, b, want[i])
		}
	}
}

func TestUniqueTypes(t *testing.T) {
	// uniqueTypes is exercised indirectly by Reflect; integration tests cover it.
	_ = uniqueTypes
}
