package main

import "testing"

func TestSourcePreflightDoesNotExposeCredentials(t *testing.T) {
	t.Setenv("ALPACA_API_KEY", "")
	t.Setenv("ALPACA_API_SECRET", "")
	t.Setenv("SEC_USER_AGENT", "")
	t.Setenv("SEC_CONTACT", "")
	for _, status := range []sourceStatus{marketStatus(), secStatus()} {
		if status.Status != "BLOCKED" {
			t.Fatalf("status = %+v", status)
		}
		if status.Detail == "" {
			t.Fatal("missing bounded blocker detail")
		}
	}
}
