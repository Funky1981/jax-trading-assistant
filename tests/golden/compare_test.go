package golden

import "testing"

func TestCompareSnapshots_IgnoresVolatileFields(t *testing.T) {
	expected := &Snapshot{
		Service:  "jax-trader",
		Endpoint: "/api/v1/artifacts",
		Response: map[string]interface{}{
			"id":         "5d4f9cf1-2466-45ec-b2cc-eafde8ea8f9e",
			"created_at": "2026-03-05T10:00:00Z",
			"status":     "ok",
		},
	}
	actual := &Snapshot{
		Service:  "jax-trader",
		Endpoint: "/api/v1/artifacts",
		Response: map[string]interface{}{
			"id":         "f8a76ec5-249b-47f5-a2f5-c35f7a5ec3d4",
			"created_at": "2026-03-05T10:05:00Z",
			"status":     "ok",
		},
	}

	result := CompareSnapshots(expected, actual)
	if !result.Match {
		t.Fatalf("expected snapshots to match, differences: %v", result.Differences)
	}
}

func TestCompareSnapshots_DetectsNonVolatileDiff(t *testing.T) {
	expected := &Snapshot{
		Service:  "jax-trader",
		Endpoint: "/api/v1/artifacts",
		Response: map[string]interface{}{
			"status": "ok",
		},
	}
	actual := &Snapshot{
		Service:  "jax-trader",
		Endpoint: "/api/v1/artifacts",
		Response: map[string]interface{}{
			"status": "failed",
		},
	}

	result := CompareSnapshots(expected, actual)
	if result.Match {
		t.Fatal("expected snapshots to mismatch for non-volatile field")
	}
}
