package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ProvidersPath == "" {
		t.Fatalf("expected providers path")
	}
	if cfg.Threshold == 0 {
		t.Fatalf("expected non-zero threshold")
	}
}

func TestParseConfigOverrides(t *testing.T) {
	cfg, err := Parse([]string{
		"-providers", "config/custom.json",
		"-input", "payload.json",
		"-threshold", "0.9",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.ProvidersPath != "config/custom.json" {
		t.Fatalf("expected providers override, got %q", cfg.ProvidersPath)
	}
	if cfg.InputPath != "payload.json" {
		t.Fatalf("expected input override, got %q", cfg.InputPath)
	}
	if cfg.Threshold != 0.9 {
		t.Fatalf("expected threshold 0.9, got %v", cfg.Threshold)
	}
}
