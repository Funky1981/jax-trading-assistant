package main

import "testing"

func TestGetenvInt_Defaults(t *testing.T) {
	if got := getenvInt("JAX_MEMORY_TEST_PORT", 8090); got != 8090 {
		t.Fatalf("expected default 8090, got %d", got)
	}
}

func TestGetenvInt_Invalid(t *testing.T) {
	t.Setenv("JAX_MEMORY_TEST_PORT", "not-a-number")
	if got := getenvInt("JAX_MEMORY_TEST_PORT", 8090); got != 8090 {
		t.Fatalf("expected default 8090, got %d", got)
	}
}

func TestGetenvInt_Valid(t *testing.T) {
	t.Setenv("JAX_MEMORY_TEST_PORT", "9001")
	if got := getenvInt("JAX_MEMORY_TEST_PORT", 8090); got != 9001 {
		t.Fatalf("expected 9001, got %d", got)
	}
}
